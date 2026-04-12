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

type RemoveFavoriteLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	favoriteRepo repositories.FavoriteRepository
}

func NewRemoveFavoriteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveFavoriteLogic {
	return &RemoveFavoriteLogic{
		ctx:          ctx,
		svcCtx:       svcCtx,
		Logger:       logx.WithContext(ctx),
		favoriteRepo: repositories.NewFavoriteRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *RemoveFavoriteLogic) RemoveFavorite(in *interaction.RemoveFavoriteReq) (*interaction.RemoveFavoriteRes, error) {
	if in == nil || in.GetUserId() <= 0 || in.GetContentId() <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}
	if in.GetScene() == interaction.Scene_SCENE_UNKNOWN {
		return nil, errorx.NewBadRequest("场景参数错误")
	}

	if _, err := l.favoriteRepo.DeleteByUserAndContent(in.GetUserId(), in.GetContentId()); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("取消收藏失败"))
	}

	scene := in.GetScene().String()
	contentIDStr := strconv.FormatInt(in.GetContentId(), 10)
	userIDStr := strconv.FormatInt(in.GetUserId(), 10)

	relKey := rediskey.BuildFavoriteRelKey(scene, userIDStr, contentIDStr)
	if _, delErr := l.svcCtx.Redis.DelCtx(l.ctx, relKey); delErr != nil {
		l.Errorf("delete favorite relation cache failed: %v", delErr)
	}

	favKey := rediskey.BuildUserFavoriteFeedKey(userIDStr)
	if _, zremErr := l.svcCtx.Redis.ZremCtx(l.ctx, favKey, contentIDStr); zremErr != nil {
		l.Errorf("remove favorite feed cache failed, user_id=%d, content_id=%d, err=%v", in.GetUserId(), in.GetContentId(), zremErr)
	}

	return &interaction.RemoveFavoriteRes{}, nil
}
