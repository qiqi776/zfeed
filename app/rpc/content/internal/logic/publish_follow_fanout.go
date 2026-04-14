package logic

import (
	"context"
	"strconv"

	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	luautils "zfeed/app/rpc/content/internal/common/utils/lua"
	"zfeed/app/rpc/content/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

const followStatusActive = 10

type followerRow struct {
	UserID int64 `gorm:"column:user_id"`
}

func tryFanoutFollowInbox(ctx context.Context, svcCtx *svc.ServiceContext, authorID, contentID int64) {
	if svcCtx == nil || svcCtx.MysqlDb == nil || svcCtx.Redis == nil || authorID <= 0 || contentID <= 0 {
		return
	}

	rows := make([]followerRow, 0)
	err := svcCtx.MysqlDb.WithContext(ctx).
		Table("zfeed_follow").
		Select("user_id").
		Where("follow_user_id = ? AND status = ? AND is_deleted = 0", authorID, followStatusActive).
		Find(&rows).Error
	if err != nil {
		logx.WithContext(ctx).Errorf("query followers for publish fanout failed, author_id=%d, content_id=%d, err=%v", authorID, contentID, err)
		return
	}

	score := strconv.FormatInt(contentID, 10)
	args := []any{
		strconv.Itoa(redisconsts.RedisFollowInboxKeepLatestN),
		score,
		score,
	}

	for _, row := range rows {
		if row.UserID <= 0 {
			continue
		}
		inboxKey := redisconsts.BuildFollowInboxKey(row.UserID)
		if _, evalErr := svcCtx.Redis.EvalCtx(ctx, luautils.UpdateFollowInboxZSetScript, []string{inboxKey}, args...); evalErr != nil {
			logx.WithContext(ctx).Errorf("update follow inbox after publish failed, follower_id=%d, author_id=%d, content_id=%d, err=%v", row.UserID, authorID, contentID, evalErr)
		}
	}
}
