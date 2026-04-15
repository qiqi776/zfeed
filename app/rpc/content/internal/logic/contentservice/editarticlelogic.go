package contentservicelogic

import (
	"context"
	"errors"
	"strings"

	"zfeed/app/rpc/content/content"
	"zfeed/app/rpc/content/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type EditArticleLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewEditArticleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EditArticleLogic {
	return &EditArticleLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *EditArticleLogic) EditArticle(in *content.EditArticleReq) (*content.EditArticleRes, error) {
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
	if in.Cover != nil {
		cover := strings.TrimSpace(in.GetCover())
		if cover == "" {
			return nil, errorx.NewBadRequest("封面不能为空")
		}
		updates["cover"] = cover
	}
	if in.Content != nil {
		contentValue := strings.TrimSpace(in.GetContent())
		if contentValue == "" {
			return nil, errorx.NewBadRequest("正文不能为空")
		}
		updates["content"] = contentValue
	}
	if len(updates) == 0 {
		return nil, errorx.NewBadRequest("没有可更新的字段")
	}

	if err := l.ensureEditableContent(in.GetContentId(), in.GetUserId(), int32(content.ContentType_CONTENT_TYPE_ARTICLE)); err != nil {
		return nil, err
	}

	result := l.svcCtx.MysqlDb.WithContext(l.ctx).
		Table("zfeed_article").
		Where("content_id = ? AND is_deleted = 0", in.GetContentId()).
		Updates(updates)
	if result.Error != nil {
		return nil, errorx.Wrap(l.ctx, result.Error, errorx.NewMsg("更新文章失败"))
	}
	if result.RowsAffected == 0 {
		return nil, errorx.NewNotFound("内容不存在")
	}

	return &content.EditArticleRes{ContentId: in.GetContentId()}, nil
}

func (l *EditArticleLogic) ensureEditableContent(contentID, userID int64, wantType int32) error {
	var row struct {
		UserID      int64 `gorm:"column:user_id"`
		ContentType int32 `gorm:"column:content_type"`
		IsDeleted   int32 `gorm:"column:is_deleted"`
	}

	err := l.svcCtx.MysqlDb.WithContext(l.ctx).
		Table("zfeed_content").
		Select("user_id", "content_type", "is_deleted").
		Where("id = ?", contentID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errorx.NewNotFound("内容不存在")
		}
		return err
	}
	if row.IsDeleted != 0 {
		return errorx.NewNotFound("内容不存在")
	}
	if row.UserID != userID {
		return errorx.NewForbidden("只能编辑自己的内容")
	}
	if row.ContentType != wantType {
		return errorx.NewBadRequest("内容类型错误")
	}
	return nil
}
