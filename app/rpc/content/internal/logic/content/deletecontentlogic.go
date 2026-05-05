package contentlogic

import (
	"context"
	"errors"
	"strconv"

	"zfeed/app/rpc/content/content"
	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	"zfeed/app/rpc/content/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

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

	l.cleanupIndexes(in.GetUserId(), in.GetContentId(), contentType)

	return &content.DeleteContentRes{}, nil
}

func (l *DeleteContentLogic) cleanupIndexes(authorID, contentID int64, contentType int32) {
	if l.svcCtx == nil || l.svcCtx.Redis == nil || contentID <= 0 {
		return
	}

	member := strconv.FormatInt(contentID, 10)
	l.zrem(redisconsts.BuildUserPublishFeedKey(authorID), member, "publish cache", contentID, contentType)
	l.cleanFollowInboxes(authorID, member, contentID, contentType)
	l.cleanFavoriteFeeds(member, contentID, contentType)
	l.cleanHotIndexes(contentID, member, contentType)
}

func (l *DeleteContentLogic) cleanFollowInboxes(authorID int64, member string, contentID int64, contentType int32) {
	if l.svcCtx.MysqlDb == nil || authorID <= 0 {
		return
	}

	rows := make([]followerRow, 0)
	err := l.svcCtx.MysqlDb.WithContext(l.ctx).
		Table("zfeed_follow").
		Select("user_id").
		Where("follow_user_id = ? AND status = ? AND is_deleted = 0", authorID, followStatusActive).
		Find(&rows).Error
	if err != nil {
		l.Errorf("query followers for delete cleanup failed, author_id=%d, content_id=%d, err=%v", authorID, contentID, err)
		return
	}

	for _, row := range rows {
		if row.UserID <= 0 {
			continue
		}
		l.zrem(redisconsts.BuildFollowInboxKey(row.UserID), member, "follow inbox", contentID, contentType)
	}
}

type favoriteUserRow struct {
	UserID int64 `gorm:"column:user_id"`
}

func (l *DeleteContentLogic) cleanFavoriteFeeds(member string, contentID int64, contentType int32) {
	if l.svcCtx.MysqlDb == nil {
		return
	}

	rows := make([]favoriteUserRow, 0)
	err := l.svcCtx.MysqlDb.WithContext(l.ctx).
		Table("zfeed_favorite").
		Select("user_id").
		Where("content_id = ?", contentID).
		Find(&rows).Error
	if err != nil {
		l.Errorf("query favorite users for delete cleanup failed, content_id=%d, content_type=%d, err=%v", contentID, contentType, err)
		return
	}

	for _, row := range rows {
		if row.UserID <= 0 {
			continue
		}
		l.zrem(redisconsts.BuildUserFavoriteFeedKey(row.UserID), member, "favorite feed", contentID, contentType)
	}
}

func (l *DeleteContentLogic) cleanHotIndexes(contentID int64, member string, contentType int32) {
	l.zrem(redisconsts.HotFeedKey, member, "hot feed", contentID, contentType)

	latestSnapshotID, err := l.svcCtx.Redis.GetCtx(l.ctx, redisconsts.HotFeedLatestKey)
	if err != nil {
		l.Errorf("query latest hot snapshot failed, content_id=%d, content_type=%d, err=%v", contentID, contentType, err)
	} else if latestSnapshotID != "" {
		l.zrem(redisconsts.BuildHotFeedSnapshotKey(latestSnapshotID), member, "hot snapshot", contentID, contentType)
	}

	incKey := redisconsts.BuildHotFeedIncKey(int(contentID % int64(redisconsts.HotFeedIncShards)))
	if _, err := l.svcCtx.Redis.HdelCtx(l.ctx, incKey, member); err != nil {
		l.Errorf("remove content from hot increment bucket failed, key=%s, content_id=%d, content_type=%d, err=%v", incKey, contentID, contentType, err)
	}
}

func (l *DeleteContentLogic) zrem(key, member, desc string, contentID int64, contentType int32) {
	if _, err := l.svcCtx.Redis.ZremCtx(l.ctx, key, member); err != nil {
		l.Errorf("remove content from %s failed, key=%s, content_id=%d, content_type=%d, err=%v", desc, key, contentID, contentType, err)
	}
}
