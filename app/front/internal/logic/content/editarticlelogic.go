package content

import (
	"context"
	"errors"
	"strings"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
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
	if req.Cover != nil {
		cover := strings.TrimSpace(*req.Cover)
		if cover == "" {
			return nil, errorx.NewBadRequest("封面不能为空")
		}
		updates["cover"] = cover
	}
	if req.Content != nil {
		content := strings.TrimSpace(*req.Content)
		if content == "" {
			return nil, errorx.NewBadRequest("正文不能为空")
		}
		updates["content"] = content
	}
	if len(updates) == 0 {
		return nil, errorx.NewBadRequest("没有可更新的字段")
	}

	if err := l.ensureEditableContent(req.ContentId, userID, 10); err != nil {
		return nil, err
	}

	result := l.svcCtx.MysqlDb.WithContext(l.ctx).
		Table("zfeed_article").
		Where("content_id = ? AND is_deleted = 0", req.ContentId).
		Updates(updates)
	if result.Error != nil {
		return nil, errorx.Wrap(l.ctx, result.Error, errorx.NewMsg("更新文章失败"))
	}
	if result.RowsAffected == 0 {
		return nil, errorx.NewNotFound("内容不存在")
	}

	return &types.EditArticleRes{ContentId: req.ContentId}, nil
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
