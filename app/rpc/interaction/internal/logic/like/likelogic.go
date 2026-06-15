package likelogic

import (
	"context"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"

	"zfeed/app/rpc/interaction/interaction"
	rediskey "zfeed/app/rpc/interaction/internal/common/consts/redis"
	luautils "zfeed/app/rpc/interaction/internal/common/utils/lua"
	"zfeed/app/rpc/interaction/internal/mq/event"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/pkg/errorx"
)

type LikeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo repositories.ContentRepository
	commentRepo repositories.CommentRepository
}

func NewLikeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LikeLogic {
	return &LikeLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		commentRepo: repositories.NewCommentRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *LikeLogic) Like(in *interaction.LikeReq) (*interaction.LikeRes, error) {
	if in == nil || in.GetUserId() <= 0 || in.GetContentId() <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}
	if in.GetScene() == interaction.Scene_SCENE_UNKNOWN {
		return nil, errorx.NewBadRequest("场景参数错误")
	}

	contentUserID, err := resolveLikeTargetOwner(l.contentRepo, l.commentRepo, in.GetScene(), in.GetContentId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询内容作者失败"))
	}
	if contentUserID <= 0 {
		return nil, errorx.NewNotFound("内容不存在")
	}

	changed, err := l.processLike(in.GetUserId(), in.GetScene(), in.GetContentId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("点赞处理失败"))
	}
	if changed {
		scene := in.GetScene().String()
		if err := l.publishLikeEvent(in.GetUserId(), in.GetContentId(), contentUserID, scene); err != nil {
			l.rollbackLikeCacheState(in.GetUserId(), in.GetScene(), in.GetContentId(), false)
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("持久化点赞事件失败"))
		}
		emitUserAction(
			l.ctx,
			l.Logger,
			l.svcCtx.UserActionProducer,
			event.UserActionLike,
			in.GetUserId(),
			in.GetContentId(),
			contentUserID,
			in.GetScene(),
		)
	}

	return &interaction.LikeRes{}, nil
}

func (l *LikeLogic) processLike(userID int64, scene interaction.Scene, contentID int64) (changed bool, err error) {
	userIDStr := strconv.FormatInt(userID, 10)
	userLikeKey := rediskey.BuildLikeUserKey(userIDStr)
	field := likeTargetKey(scene, contentID)

	resultVal, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.LikeUserHashScript,
		[]string{userLikeKey},
		field,
		strconv.FormatInt(rediskey.LikeExpireSeconds, 10),
	)
	if err != nil {
		return false, err
	}

	arr, ok := resultVal.([]interface{})
	if !ok || len(arr) < 2 {
		return false, errorx.NewMsg("解析点赞脚本返回值失败")
	}

	changedVal, _ := arr[0].(int64)
	return changedVal == 1, nil
}

func (l *LikeLogic) publishLikeEvent(userID, contentID, contentUserID int64, scene string) error {
	return l.svcCtx.LikeProducer.SendLikeEvent(l.ctx, userID, contentID, contentUserID, scene)
}

func (l *LikeLogic) rollbackLikeCacheState(userID int64, scene interaction.Scene, contentID int64, isLiked bool) {
	if l.svcCtx == nil || l.svcCtx.Redis == nil {
		return
	}

	if err := cacheLikeState(
		l.ctx,
		l.svcCtx.Redis,
		likeCacheKey(userID),
		likeTargetKey(scene, contentID),
		isLiked,
	); err != nil {
		l.Errorf(
			"rollback like cache failed, user_id=%d, scene=%s, content_id=%d, err=%v",
			userID,
			scene.String(),
			contentID,
			err,
		)
	}
}

func resolveLikeTargetOwner(
	contentRepo repositories.ContentRepository,
	commentRepo repositories.CommentRepository,
	scene interaction.Scene,
	contentID int64,
) (int64, error) {
	switch scene {
	case interaction.Scene_ARTICLE, interaction.Scene_VIDEO:
		return contentRepo.GetAuthorID(contentID)
	case interaction.Scene_COMMENT:
		commentDO, err := commentRepo.GetByID(contentID)
		if err != nil {
			return 0, err
		}
		if commentDO == nil {
			return 0, nil
		}
		return commentDO.UserID, nil
	default:
		return 0, nil
	}
}
