// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package content

import (
	"context"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	contentpb "zfeed/app/rpc/content/content"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"
)

type PublishArticleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPublishArticleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PublishArticleLogic {
	return &PublishArticleLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PublishArticleLogic) PublishArticle(req *types.PublishArticleReq) (resp *types.PublishArticleRes, err error) {
	if req == nil || req.Title == nil || req.Content == nil || req.Visibility == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}

	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewUnauthorized("用户未登录"))
	}

	cover := ""
	if req.Cover != nil {
		cover = strings.TrimSpace(*req.Cover)
	}

	rpcResp, err := l.svcCtx.ContentRpc.PublishArticle(l.ctx, &contentpb.ArticlePublishReq{
		UserId:      userID,
		Title:       strings.TrimSpace(*req.Title),
		Description: req.Description,
		Cover:       cover,
		Content:     *req.Content,
		Visibility:  contentpb.Visibility(*req.Visibility),
	})
	if err != nil {
		return nil, err
	}

	return &types.PublishArticleRes{
		ContentId: rpcResp.GetContentId(),
	}, nil
}
