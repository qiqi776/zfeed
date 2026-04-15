package contentservicelogic

import (
	"context"
	"strings"

	"zfeed/app/rpc/content/content"
	"zfeed/app/rpc/content/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type EditVideoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewEditVideoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EditVideoLogic {
	return &EditVideoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *EditVideoLogic) EditVideo(in *content.EditVideoReq) (*content.EditVideoRes, error) {
	if in == nil || in.GetUserId() <= 0 || in.GetContentId() <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}

	updates := map[string]any{}
	if in.Title != nil {
		title := strings.TrimSpace(in.GetTitle())
		if title == "" {
			return nil, errorx.NewBadRequest("标题不能为空")
		}
		updates["title"] = title
	}
	if in.Description != nil {
		updates["description"] = strings.TrimSpace(in.GetDescription())
	}
	if in.OriginUrl != nil {
		originURL := strings.TrimSpace(in.GetOriginUrl())
		if originURL == "" {
			return nil, errorx.NewBadRequest("视频地址不能为空")
		}
		updates["origin_url"] = originURL
	}
	if in.CoverUrl != nil {
		coverURL := strings.TrimSpace(in.GetCoverUrl())
		if coverURL == "" {
			return nil, errorx.NewBadRequest("封面不能为空")
		}
		updates["cover_url"] = coverURL
	}
	if in.Duration != nil {
		if in.GetDuration() <= 0 {
			return nil, errorx.NewBadRequest("时长参数错误")
		}
		updates["duration"] = in.GetDuration()
	}
	if len(updates) == 0 {
		return nil, errorx.NewBadRequest("没有可更新的字段")
	}

	guard := NewEditArticleLogic(l.ctx, l.svcCtx)
	if err := guard.ensureEditableContent(in.GetContentId(), in.GetUserId(), int32(content.ContentType_CONTENT_TYPE_VIDEO)); err != nil {
		return nil, err
	}

	result := l.svcCtx.MysqlDb.WithContext(l.ctx).
		Table("zfeed_video").
		Where("content_id = ? AND is_deleted = 0", in.GetContentId()).
		Updates(updates)
	if result.Error != nil {
		return nil, errorx.Wrap(l.ctx, result.Error, errorx.NewMsg("更新视频失败"))
	}
	if result.RowsAffected == 0 {
		return nil, errorx.NewNotFound("内容不存在")
	}

	return &content.EditVideoRes{ContentId: in.GetContentId()}, nil
}
