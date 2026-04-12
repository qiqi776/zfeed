package favoriteservicelogic

import (
	"context"
	"strconv"

	"zfeed/app/rpc/interaction/interaction"
	rediskey "zfeed/app/rpc/interaction/internal/common/consts/redis"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type QueryFavoriteInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	favoriteRepo repositories.FavoriteRepository
}

func NewQueryFavoriteInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryFavoriteInfoLogic {
	return &QueryFavoriteInfoLogic{
		ctx:          ctx,
		svcCtx:       svcCtx,
		Logger:       logx.WithContext(ctx),
		favoriteRepo: repositories.NewFavoriteRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *QueryFavoriteInfoLogic) QueryFavoriteInfo(in *interaction.QueryFavoriteInfoReq) (*interaction.QueryFavoriteInfoRes, error) {
	if in == nil || in.GetContentId() <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}
	if in.GetScene() == interaction.Scene_SCENE_UNKNOWN {
		return nil, errorx.NewBadRequest("场景参数错误")
	}

	favoriteCount, err := l.favoriteRepo.CountByContentID(in.GetContentId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询收藏信息失败"))
	}

	isFavorited := false
	if in.GetUserId() > 0 {
		isFavorited, err = l.queryIsFavorited(in.GetUserId(), in.GetContentId(), in.GetScene())
		if err != nil {
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询收藏信息失败"))
		}
	}

	return &interaction.QueryFavoriteInfoRes{
		FavoriteCount: favoriteCount,
		IsFavorited:   isFavorited,
		ContentId:     in.GetContentId(),
		Scene:         in.GetScene(),
	}, nil
}

func (l *QueryFavoriteInfoLogic) queryIsFavorited(userID, contentID int64, scene interaction.Scene) (bool, error) {
	relKey := rediskey.BuildFavoriteRelKey(scene.String(), strconv.FormatInt(userID, 10), strconv.FormatInt(contentID, 10))

	cacheVal, err := l.svcCtx.Redis.GetCtx(l.ctx, relKey)
	if err == nil {
		switch cacheVal {
		case "1":
			return true, nil
		case "0":
			return false, nil
		}
	} else {
		l.Errorf("query favorite relation cache failed, key=%s, err=%v", relKey, err)
	}

	isFavorited, err := l.favoriteRepo.IsFavorited(userID, contentID)
	if err != nil {
		return false, err
	}

	cacheValue := "0"
	expireSecs := rediskey.RedisFavoriteRelNegExpireSecs
	if isFavorited {
		cacheValue = "1"
		expireSecs = rediskey.RedisFavoriteRelExpireSecs
	}
	if setErr := l.svcCtx.Redis.SetexCtx(l.ctx, relKey, cacheValue, expireSecs); setErr != nil {
		l.Errorf("rebuild favorite relation cache failed, key=%s, err=%v", relKey, setErr)
	}

	return isFavorited, nil
}
