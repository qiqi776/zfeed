// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package content

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"
)

const userPublishRedisPrefix = "feed:user:publish"

type DeleteContentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteContentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteContentLogic {
	return &DeleteContentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteContentLogic) DeleteContent(req *types.DeleteContentReq) (resp *types.DeleteContentRes, err error) {
	if req == nil || req.ContentId <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}

	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewUnauthorized("用户未登录"))
	}

	if l.svcCtx == nil || l.svcCtx.MysqlDb == nil {
		return nil, errorx.NewMsg("删除内容失败")
	}

	var contentType int32
	err = l.svcCtx.MysqlDb.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		var row struct {
			ID          int64 `gorm:"column:id"`
			UserID      int64 `gorm:"column:user_id"`
			ContentType int32 `gorm:"column:content_type"`
			IsDeleted   int32 `gorm:"column:is_deleted"`
		}

		queryErr := tx.Table("zfeed_content").
			Select("id", "user_id", "content_type", "is_deleted").
			Where("id = ?", req.ContentId).
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
		if row.UserID != userID {
			return errorx.NewForbidden("只能删除自己的内容")
		}

		contentType = row.ContentType
		updateRes := tx.Table("zfeed_content").
			Where("id = ? AND is_deleted = 0", req.ContentId).
			Updates(map[string]any{
				"is_deleted": 1,
				"updated_by": userID,
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
				Where("content_id = ? AND is_deleted = 0", req.ContentId).
				Update("is_deleted", 1).Error; err != nil {
				return err
			}
		case 20:
			if err := tx.Table("zfeed_video").
				Where("content_id = ? AND is_deleted = 0", req.ContentId).
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
		contentID := strconv.FormatInt(req.ContentId, 10)
		publishKey := fmt.Sprintf("%s:%d", userPublishRedisPrefix, userID)
		if _, redisErr := l.svcCtx.Redis.ZremCtx(l.ctx, publishKey, contentID); redisErr != nil {
			l.Errorf("remove content from publish cache failed, key=%s, content_id=%d, content_type=%d, err=%v", publishKey, req.ContentId, contentType, redisErr)
		}
	}

	return &types.DeleteContentRes{}, nil
}
