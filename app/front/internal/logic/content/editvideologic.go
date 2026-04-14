package content

import (
	"context"
	"strings"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type EditVideoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewEditVideoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EditVideoLogic {
	return &EditVideoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EditVideoLogic) EditVideo(req *types.EditVideoReq) (*types.EditVideoRes, error) {
	if req == nil || req.ContentId <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}

	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewUnauthorized("用户未登录"))
	}

	updates := map[string]any{}
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return nil, errorx.NewBadRequest("标题不能为空")
		}
		updates["title"] = title
	}
	if req.Description != nil {
		updates["description"] = strings.TrimSpace(*req.Description)
	}
	if req.VideoUrl != nil {
		videoURL := strings.TrimSpace(*req.VideoUrl)
		if videoURL == "" {
			return nil, errorx.NewBadRequest("视频地址不能为空")
		}
		updates["origin_url"] = videoURL
	}
	if req.CoverUrl != nil {
		coverURL := strings.TrimSpace(*req.CoverUrl)
		if coverURL == "" {
			return nil, errorx.NewBadRequest("封面不能为空")
		}
		updates["cover_url"] = coverURL
	}
	if req.Duration != nil {
		if *req.Duration <= 0 {
			return nil, errorx.NewBadRequest("时长参数错误")
		}
		updates["duration"] = *req.Duration
	}
	if len(updates) == 0 {
		return nil, errorx.NewBadRequest("没有可更新的字段")
	}

	guard := NewEditArticleLogic(l.ctx, l.svcCtx)
	if err := guard.ensureEditableContent(req.ContentId, userID, 20); err != nil {
		return nil, err
	}

	result := l.svcCtx.MysqlDb.WithContext(l.ctx).
		Table("zfeed_video").
		Where("content_id = ? AND is_deleted = 0", req.ContentId).
		Updates(updates)
	if result.Error != nil {
		return nil, errorx.Wrap(l.ctx, result.Error, errorx.NewMsg("更新视频失败"))
	}
	if result.RowsAffected == 0 {
		return nil, errorx.NewNotFound("内容不存在")
	}

	return &types.EditVideoRes{ContentId: req.ContentId}, nil
}
