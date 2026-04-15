package contentservicelogic

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"zfeed/app/rpc/content/content"
	"zfeed/app/rpc/content/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

const userPublishRedisPrefix = "feed:user:publish"

type DeleteContentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteContentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteContentLogic {
	return &DeleteContentLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteContentLogic) DeleteContent(in *content.DeleteContentReq) (*content.DeleteContentRes, error) {
	if in == nil || in.GetContentId() <= 0 || in.GetUserId() <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}

	if l.svcCtx == nil || l.svcCtx.MysqlDb == nil {
		return nil, errorx.NewMsg("删除内容失败")
	}

	var contentType int32
	err := l.svcCtx.MysqlDb.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		var row struct {
			ID          int64 `gorm:"column:id"`
			UserID      int64 `gorm:"column:user_id"`
			ContentType int32 `gorm:"column:content_type"`
			IsDeleted   int32 `gorm:"column:is_deleted"`
		}

		queryErr := tx.Table("zfeed_content").
			Select("id", "user_id", "content_type", "is_deleted").
			Where("id = ?", in.GetContentId()).
			Take(&row).Error
		if queryErr != nil {
			if errors.Is(queryErr, gorm.ErrRecordNotFound) {
				return errorx.NewNotFound("内容不存在")
			}
			return queryErr
		}
		if row.IsDeleted != 0 {
			return errorx.NewNotFound("内容不存在")
		}
		if row.UserID != in.GetUserId() {
			return errorx.NewForbidden("只能删除自己的内容")
		}

		contentType = row.ContentType
		updateRes := tx.Table("zfeed_content").
			Where("id = ? AND is_deleted = 0", in.GetContentId()).
			Updates(map[string]any{
				"is_deleted": 1,
				"updated_by": in.GetUserId(),
			})
		if updateRes.Error != nil {
			return updateRes.Error
		}
		if updateRes.RowsAffected == 0 {
			return errorx.NewNotFound("内容不存在")
		}

		switch row.ContentType {
		case 10:
			if err := tx.Table("zfeed_article").
				Where("content_id = ? AND is_deleted = 0", in.GetContentId()).
				Update("is_deleted", 1).Error; err != nil {
				return err
			}
		case 20:
			if err := tx.Table("zfeed_video").
				Where("content_id = ? AND is_deleted = 0", in.GetContentId()).
				Update("is_deleted", 1).Error; err != nil {
				return err
			}
		default:
			return errorx.NewBadRequest("内容类型错误")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if l.svcCtx.Redis != nil {
		contentID := strconv.FormatInt(in.GetContentId(), 10)
		publishKey := fmt.Sprintf("%s:%d", userPublishRedisPrefix, in.GetUserId())
		if _, redisErr := l.svcCtx.Redis.ZremCtx(l.ctx, publishKey, contentID); redisErr != nil {
			l.Errorf("remove content from publish cache failed, key=%s, content_id=%d, content_type=%d, err=%v", publishKey, in.GetContentId(), contentType, redisErr)
		}
	}

	return &content.DeleteContentRes{}, nil
}
