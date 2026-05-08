package likelogic

import (
	"context"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"

	"zfeed/app/rpc/interaction/interaction"
	rediskey "zfeed/app/rpc/interaction/internal/common/consts/redis"
	luautils "zfeed/app/rpc/interaction/internal/common/utils/lua"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/pkg/errorx"
)

type UnlikeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo repositories.ContentRepository
	commentRepo repositories.CommentRepository
	likeRepo    repositories.LikeRepository
}

func NewUnlikeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UnlikeLogic {
	return &UnlikeLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		commentRepo: repositories.NewCommentRepository(ctx, svcCtx.MysqlDb),
		likeRepo:    repositories.NewLikeRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *UnlikeLogic) Unlike(in *interaction.UnlikeReq) (*interaction.UnlikeRes, error) {
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

	changed, err := l.processUnlike(in.GetUserId(), in.GetScene(), in.GetContentId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("取消点赞失败"))
	}
	if changed {
		scene := in.GetScene().String()
		threading.GoSafe(func() {
			l.publishCancelLikeEvent(in.GetUserId(), in.GetContentId(), contentUserID, scene)
		})
	}

	return &interaction.UnlikeRes{}, nil
}

func (l *UnlikeLogic) processUnlike(userID int64, scene interaction.Scene, contentID int64) (changed bool, err error) {
	userIDStr := strconv.FormatInt(userID, 10)
	userLikeKey := rediskey.BuildLikeUserKey(userIDStr)
	field := likeTargetKey(scene, contentID)

	resultVal, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.CancelLikeUserHashScript,
		[]string{userLikeKey},
		field,
		strconv.FormatInt(rediskey.LikeExpireSeconds, 10),
	)
	if err != nil {
		return false, err
	}

	arr, ok := resultVal.([]interface{})
	if !ok || len(arr) < 2 {
		return false, errorx.NewMsg("解析取消点赞脚本返回值失败")
	}

	changedVal, _ := arr[0].(int64)
	if changedVal == 1 {
		return true, nil
	}

	existedVal, _ := arr[1].(int64)
	if existedVal == 1 {
		return false, nil
	}

	isLiked, err := l.likeRepo.IsLiked(userID, int32(scene), contentID)
	if err != nil {
		return false, err
	}

	return isLiked, nil
}

func (l *UnlikeLogic) publishCancelLikeEvent(userID, contentID, contentUserID int64, scene string) {
	l.svcCtx.LikeProducer.SendCancelLikeEvent(l.ctx, userID, contentID, contentUserID, scene)
}
