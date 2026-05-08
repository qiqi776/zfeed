package likelogic

import (
	"context"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type QueryLikeInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	likeRepo repositories.LikeRepository
}

func NewQueryLikeInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryLikeInfoLogic {
	return &QueryLikeInfoLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		Logger:   logx.WithContext(ctx),
		likeRepo: repositories.NewLikeRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *QueryLikeInfoLogic) QueryLikeInfo(in *interaction.QueryLikeInfoReq) (*interaction.QueryLikeInfoRes, error) {
	if in == nil || in.GetContentId() <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}
	if in.GetScene() == interaction.Scene_SCENE_UNKNOWN {
		return nil, errorx.NewBadRequest("场景参数错误")
	}

	likeCount, err := l.likeRepo.CountByTarget(int32(in.GetScene()), in.GetContentId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询点赞信息失败"))
	}

	isLiked := false
	if in.GetUserId() > 0 {
		isLiked, err = l.queryIsLiked(in.GetUserId(), in.GetScene(), in.GetContentId())
		if err != nil {
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询点赞信息失败"))
		}
	}

	return &interaction.QueryLikeInfoRes{
		LikeCount: likeCount,
		IsLiked:   isLiked,
		ContentId: in.GetContentId(),
		Scene:     in.GetScene(),
	}, nil
}

func (l *QueryLikeInfoLogic) queryIsLiked(userID int64, scene interaction.Scene, contentID int64) (bool, error) {
	userLikeKey := likeCacheKey(userID)
	field := likeTargetKey(scene, contentID)

	cacheValues, err := l.svcCtx.Redis.HmgetCtx(l.ctx, userLikeKey, field)
	if err == nil {
		if len(cacheValues) > 0 {
			if isLiked, ok := parseLikeCacheValue(cacheValues[0]); ok {
				return isLiked, nil
			}
		}
	} else {
		l.Errorf("query like relation cache failed, key=%s, field=%s, err=%v", userLikeKey, field, err)
	}

	isLiked, err := l.likeRepo.IsLiked(userID, int32(scene), contentID)
	if err != nil {
		return false, err
	}
	if setErr := cacheLikeState(l.ctx, l.svcCtx.Redis, userLikeKey, field, isLiked); setErr != nil {
		l.Errorf("rebuild like relation cache failed, key=%s, field=%s, err=%v", userLikeKey, field, setErr)
	}

	return isLiked, nil
}
