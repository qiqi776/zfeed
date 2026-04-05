package commentservicelogic

import (
	"context"
	"errors"
	"strconv"

	"zfeed/app/rpc/interaction/interaction"
	rediskey "zfeed/app/rpc/interaction/internal/common/consts/redis"
	"zfeed/app/rpc/interaction/internal/do"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type DeleteCommentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	commentRepo repositories.CommentRepository
}

func NewDeleteCommentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteCommentLogic {
	return &DeleteCommentLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		commentRepo: repositories.NewCommentRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *DeleteCommentLogic) DeleteComment(in *interaction.DeleteCommentReq) (*interaction.DeleteCommentRes, error) {
	if in == nil || in.GetUserId() <= 0 || in.GetCommentId() <= 0 || in.GetContentId() <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	if in.GetScene() == interaction.Scene_SCENE_UNKNOWN {
		return nil, errorx.NewMsg("场景参数错误")
	}

	commentDO, err := l.commentRepo.GetByIDIncludeDeleted(in.GetCommentId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询评论失败"))
	}
	if commentDO == nil || commentDO.ContentID != in.GetContentId() {
		return nil, errorx.NewMsg("评论不存在")
	}
	if commentDO.UserID != in.GetUserId() {
		return nil, errorx.NewMsg("无权限删除评论")
	}
	if isDeletedComment(commentDO) {
		return &interaction.DeleteCommentRes{}, nil
	}

	err = l.svcCtx.MysqlDb.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		commentRepo := l.commentRepo.WithTx(tx)

		hasChildren, err := commentRepo.HasChildren(commentDO.ID)
		if err != nil {
			return err
		}

		if hasChildren {
			if err := commentRepo.MarkDeleted(commentDO.ID, in.GetUserId()); err != nil {
				return err
			}
		} else {
			if err := commentRepo.DeleteByID(commentDO.ID); err != nil {
				return err
			}
		}

		if commentDO.RootID > 0 {
			if err := commentRepo.DecReplyCount(commentDO.RootID); err != nil {
				return err
			}
			if !hasChildren {
				return l.cleanupDeletedAncestors(commentRepo, commentDO.ParentID)
			}
		}

		return nil
	})
	if err != nil {
		var bizErr *errorx.BizError
		if errors.As(err, &bizErr) {
			return nil, bizErr
		}
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("删除评论失败"))
	}

	l.invalidateCommentCachesAfterDelete(commentDO, in.GetScene())

	return &interaction.DeleteCommentRes{}, nil
}

func (l *DeleteCommentLogic) cleanupDeletedAncestors(commentRepo repositories.CommentRepository, parentID int64) error {
	current := parentID
	for current > 0 {
		parentComment, err := commentRepo.GetByIDIncludeDeleted(current)
		if err != nil {
			return err
		}
		if parentComment == nil || !isDeletedComment(parentComment) {
			return nil
		}

		hasChildren, err := commentRepo.HasChildren(parentComment.ID)
		if err != nil {
			return err
		}
		if hasChildren {
			return nil
		}

		if err := commentRepo.DeleteByID(parentComment.ID); err != nil {
			return err
		}
		current = parentComment.ParentID
	}
	return nil
}

func (l *DeleteCommentLogic) invalidateCommentCachesAfterDelete(commentDO *do.CommentDO, scene interaction.Scene) {
	if commentDO == nil || l.svcCtx.Redis == nil {
		return
	}

	keys := []string{
		rediskey.BuildCommentItemKey(strconv.FormatInt(commentDO.ID, 10)),
		rediskey.BuildCommentListKey(scene.String(), strconv.FormatInt(commentDO.ContentID, 10)),
	}

	if commentDO.RootID > 0 {
		keys = append(keys,
			rediskey.BuildCommentReplyKey(strconv.FormatInt(commentDO.RootID, 10)),
			rediskey.BuildCommentItemKey(strconv.FormatInt(commentDO.RootID, 10)),
		)
	}
	if commentDO.ParentID > 0 {
		keys = append(keys, rediskey.BuildCommentItemKey(strconv.FormatInt(commentDO.ParentID, 10)))
	}

	invalidateCommentCacheKeysBestEffort(l.ctx, l.Logger, l.svcCtx.Redis, keys...)
}
