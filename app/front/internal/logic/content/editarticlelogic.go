package content

import (
	"context"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	contentservice "zfeed/app/rpc/content/contentservice"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type EditArticleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewEditArticleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EditArticleLogic {
	return &EditArticleLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EditArticleLogic) EditArticle(req *types.EditArticleReq) (*types.EditArticleRes, error) {
	if req == nil || req.ContentId <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}

	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewUnauthorized("用户未登录"))
	}

	rpcResp, err := l.svcCtx.ContentRpc.EditArticle(l.ctx, &contentservice.EditArticleReq{
		UserId:      userID,
		ContentId:   req.ContentId,
		Title:       req.Title,
		Description: req.Description,
		Cover:       req.Cover,
		Content:     req.Content,
	})
	if err != nil {
		return nil, err
	}
	if rpcResp == nil {
		return nil, errorx.NewMsg("更新文章失败")
	}

	return &types.EditArticleRes{ContentId: rpcResp.GetContentId()}, nil
}
