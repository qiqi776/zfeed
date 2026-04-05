package commentservicelogic

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"

	"zfeed/app/rpc/interaction/interaction"
	rediskey "zfeed/app/rpc/interaction/internal/common/consts/redis"
	"zfeed/app/rpc/interaction/internal/do"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type CommentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	commentRepo repositories.CommentRepository
	contentRepo repositories.ContentRepository
}

func NewCommentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CommentLogic {
	return &CommentLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		commentRepo: repositories.NewCommentRepository(ctx, svcCtx.MysqlDb),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *CommentLogic) Comment(in *interaction.CommentReq) (*interaction.CommentRes, error) {
	if in == nil || in.GetUserId() <= 0 || in.GetContentId() <= 0 || in.GetContentUserId() <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	if in.GetScene() == interaction.Scene_SCENE_UNKNOWN {
		return nil, errorx.NewMsg("场景参数错误")
	}

	commentText := strings.TrimSpace(in.GetComment())
	if commentText == "" || utf8.RuneCountInString(commentText) > 255 {
		return nil, errorx.NewMsg("评论内容错误")
	}

	contentUserID, err := l.contentRepo.GetAuthorID(in.GetContentId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询内容作者失败"))
	}
	if contentUserID <= 0 {
		return nil, errorx.NewMsg("内容不存在")
	}
	if in.GetContentUserId() != contentUserID {
		return nil, errorx.NewMsg("内容作者错误")
	}

	var commentID int64
	var normalizedParentID int64
	var normalizedRootID int64
	err = l.svcCtx.MysqlDb.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		commentRepo := l.commentRepo.WithTx(tx)
		parentID, rootID, replyToUserID, err := l.resolveThread(commentRepo, in.GetContentId(), in.GetParentId(), in.GetRootId(), in.GetReplyToUserId())
		if err != nil {
			return err
		}
		normalizedParentID = parentID
		normalizedRootID = rootID

		commentID, err = commentRepo.Create(&do.CommentDO{
			ContentID:     in.GetContentId(),
			ContentUserID: contentUserID,
			UserID:        in.GetUserId(),
			ReplyToUserID: replyToUserID,
			ParentID:      parentID,
			RootID:        rootID,
			Comment:       commentText,
			Status:        repositories.CommentStatusNormal,
			Version:       1,
			ReplyCount:    0,
			IsDeleted:     0,
			CreatedBy:     in.GetUserId(),
			UpdatedBy:     in.GetUserId(),
		})
		if err != nil {
			return err
		}

		if rootID > 0 {
			return commentRepo.IncReplyCount(rootID)
		}
		return nil
	})
	if err != nil {
		var bizErr *errorx.BizError
		if errors.As(err, &bizErr) {
			return nil, bizErr
		}
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("发表评论失败"))
	}

	l.updateCommentCacheAfterCreate(commentID, in.GetContentId(), in.GetScene(), normalizedParentID, normalizedRootID)

	return &interaction.CommentRes{CommentId: commentID}, nil
}

func (l *CommentLogic) updateCommentCacheAfterCreate(commentID, contentID int64, scene interaction.Scene, parentID, rootID int64) {
	if l.svcCtx.Redis == nil || commentID <= 0 {
		return
	}

	if rootID == 0 {
		commentDO, err := l.commentRepo.GetByID(commentID)
		if err != nil || commentDO == nil {
			if err != nil {
				l.Errorf("查询新建评论失败: %v, comment_id=%d", err, commentID)
			}
			return
		}

		userMap, err := batchLoadCommentUsers(l.ctx, l.svcCtx.UserRpc, []*do.CommentDO{commentDO})
		if err != nil {
			l.Errorf("查询评论用户失败: %v, comment_id=%d", err, commentID)
			return
		}

		items := buildCommentItems([]*do.CommentDO{commentDO}, userMap)
		cacheCommentItemsAndIndexBestEffort(
			l.ctx,
			l.Logger,
			l.svcCtx.Redis,
			rediskey.BuildCommentListKey(scene.String(), strconv.FormatInt(contentID, 10)),
			items,
		)
		return
	}

	keys := []string{
		rediskey.BuildCommentReplyKey(strconv.FormatInt(rootID, 10)),
		rediskey.BuildCommentItemKey(strconv.FormatInt(rootID, 10)),
		rediskey.BuildCommentListKey(scene.String(), strconv.FormatInt(contentID, 10)),
	}
	if parentID > 0 {
		keys = append(keys, rediskey.BuildCommentItemKey(strconv.FormatInt(parentID, 10)))
	}
	invalidateCommentCacheKeysBestEffort(l.ctx, l.Logger, l.svcCtx.Redis, keys...)
}

func (l *CommentLogic) resolveThread(commentRepo repositories.CommentRepository, contentID, parentID, rootID, replyToUserID int64) (int64, int64, int64, error) {
	if parentID <= 0 && rootID <= 0 && replyToUserID <= 0 {
		return 0, 0, 0, nil
	}
	if parentID <= 0 {
		return 0, 0, 0, errorx.NewMsg("回复参数错误")
	}

	parentComment, err := commentRepo.GetByID(parentID)
	if err != nil {
		return 0, 0, 0, err
	}
	if parentComment == nil || parentComment.ContentID != contentID {
		return 0, 0, 0, errorx.NewMsg("回复评论不存在")
	}

	normalizedRootID := parentComment.RootID
	if normalizedRootID <= 0 {
		normalizedRootID = parentComment.ID
	}
	if rootID > 0 && rootID != normalizedRootID {
		return 0, 0, 0, errorx.NewMsg("根评论参数错误")
	}
	if replyToUserID > 0 && replyToUserID != parentComment.UserID {
		return 0, 0, 0, errorx.NewMsg("回复目标错误")
	}

	rootComment := parentComment
	if normalizedRootID != parentComment.ID {
		rootComment, err = commentRepo.GetByID(normalizedRootID)
		if err != nil {
			return 0, 0, 0, err
		}
		if rootComment == nil {
			return 0, 0, 0, errorx.NewMsg("根评论不存在")
		}
	}
	if rootComment.ContentID != contentID || rootComment.RootID != 0 {
		return 0, 0, 0, errorx.NewMsg("根评论参数错误")
	}

	return parentComment.ID, normalizedRootID, parentComment.UserID, nil
}
