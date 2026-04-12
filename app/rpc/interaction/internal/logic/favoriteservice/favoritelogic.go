package favoriteservicelogic

import (
	"context"
	"strconv"

	"zfeed/app/rpc/interaction/interaction"
	rediskey "zfeed/app/rpc/interaction/internal/common/consts/redis"
	luautils "zfeed/app/rpc/interaction/internal/common/utils/lua"
	"zfeed/app/rpc/interaction/internal/do"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type FavoriteLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	favoriteRepo repositories.FavoriteRepository
}

func NewFavoriteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FavoriteLogic {
	return &FavoriteLogic{
		ctx:          ctx,
		svcCtx:       svcCtx,
		Logger:       logx.WithContext(ctx),
		favoriteRepo: repositories.NewFavoriteRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *FavoriteLogic) Favorite(in *interaction.FavoriteReq) (*interaction.FavoriteRes, error) {
	if in == nil || in.GetUserId() <= 0 || in.GetContentId() <= 0 || in.GetContentUserId() <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}
	if in.GetScene() == interaction.Scene_SCENE_UNKNOWN {
		return nil, errorx.NewBadRequest("场景参数错误")
	}

	err := l.favoriteRepo.Upsert(&do.FavoriteDO{
		UserID:        in.GetUserId(),
		ContentID:     in.GetContentId(),
		ContentUserID: in.GetContentUserId(),
		Status:        repositories.FavoriteStatusActive,
		CreatedBy:     in.GetUserId(),
		UpdatedBy:     in.GetUserId(),
	})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("收藏失败"))
	}

	scene := in.GetScene().String()
	contentIDStr := strconv.FormatInt(in.GetContentId(), 10)
	userIDStr := strconv.FormatInt(in.GetUserId(), 10)

	relKey := rediskey.BuildFavoriteRelKey(scene, userIDStr, contentIDStr)
	if _, delErr := l.svcCtx.Redis.DelCtx(l.ctx, relKey); delErr != nil {
		l.Errorf("delete favorite relation cache failed: %v", delErr)
	}

	row, err := l.favoriteRepo.GetByUserAndContent(in.GetUserId(), in.GetContentId())
	if err != nil {
		l.Errorf("query favorite row failed, user_id=%d, content_id=%d, err=%v", in.GetUserId(), in.GetContentId(), err)
		return &interaction.FavoriteRes{}, nil
	}
	if row == nil || row.ID <= 0 {
		return &interaction.FavoriteRes{}, nil
	}

	favKey := rediskey.BuildUserFavoriteFeedKey(userIDStr)
	if _, evalErr := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.AddUserFavoriteIfExistsScript,
		[]string{favKey},
		strconv.FormatInt(row.ID, 10),
		contentIDStr,
		strconv.FormatInt(5000, 10),
	); evalErr != nil {
		l.Errorf("update favorite feed cache failed, user_id=%d, content_id=%d, err=%v", in.GetUserId(), in.GetContentId(), evalErr)
	}

	return &interaction.FavoriteRes{}, nil
}
