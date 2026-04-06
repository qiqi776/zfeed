package followservicelogic

import (
	"context"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListFolloweesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	followRepo repositories.FollowRepository
}

func NewListFolloweesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListFolloweesLogic {
	return &ListFolloweesLogic{
		ctx:        ctx,
		svcCtx:     svcCtx,
		Logger:     logx.WithContext(ctx),
		followRepo: repositories.NewFollowRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *ListFolloweesLogic) ListFollowees(in *interaction.ListFolloweesReq) (*interaction.ListFolloweesRes, error) {
	if in == nil || in.GetUserId() <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	pageSize := int(in.GetPageSize())
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}

	ids, err := l.followRepo.ListFolloweesByCursor(in.GetUserId(), in.GetCursor(), pageSize+1)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关注列表失败"))
	}

	hasMore := len(ids) > pageSize
	if hasMore {
		ids = ids[:pageSize]
	}

	var nextCursor int64
	if hasMore && len(ids) > 0 {
		nextCursor = ids[len(ids)-1]
	}

	return &interaction.ListFolloweesRes{
		FollowUserIds: ids,
		NextCursor:    nextCursor,
		HasMore:       hasMore,
	}, nil
}
