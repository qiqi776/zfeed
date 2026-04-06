package favoriteservicelogic

import (
	"context"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type QueryFavoriteListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	favoriteRepo repositories.FavoriteRepository
}

func NewQueryFavoriteListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryFavoriteListLogic {
	return &QueryFavoriteListLogic{
		ctx:          ctx,
		svcCtx:       svcCtx,
		Logger:       logx.WithContext(ctx),
		favoriteRepo: repositories.NewFavoriteRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *QueryFavoriteListLogic) QueryFavoriteList(in *interaction.QueryFavoriteListReq) (*interaction.QueryFavoriteListRes, error) {
	if in == nil || in.GetUserId() <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	pageSize := int(in.GetPageSize())
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	rows, err := l.favoriteRepo.ListByUserCursor(in.GetUserId(), in.GetCursor(), pageSize+1)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询收藏列表失败"))
	}

	hasMore := len(rows) > pageSize
	if hasMore {
		rows = rows[:pageSize]
	}

	items := make([]*interaction.FavoriteItem, 0, len(rows))
	for _, row := range rows {
		if row == nil || row.ContentID <= 0 {
			continue
		}
		items = append(items, &interaction.FavoriteItem{
			FavoriteId:    row.ID,
			ContentId:     row.ContentID,
			ContentUserId: row.ContentUserID,
		})
	}

	var nextCursor int64
	if hasMore && len(items) > 0 {
		nextCursor = items[len(items)-1].FavoriteId
	}

	return &interaction.QueryFavoriteListRes{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}
