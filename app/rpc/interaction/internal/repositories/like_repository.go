package repositories

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"

	"zfeed/app/rpc/interaction/internal/do"
)

const (
	LikeStatusLike   int32 = 10
	LikeStatusCancel int32 = 20
)

type LikeRepository interface {
	Upsert(likeDO *do.LikeDO) error
}

type likeRepositoryImpl struct {
	ctx context.Context
	db  *gorm.DB
	logx.Logger
}

func NewLikeRepository(ctx context.Context, db *gorm.DB) LikeRepository {
	return &likeRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *likeRepositoryImpl) Upsert(likeDO *do.LikeDO) error {
	query := `
INSERT INTO zfeed_like (
  user_id,
  content_id,
  content_user_id,
  status,
  last_event_ts,
  is_deleted,
  created_by,
  updated_by
) VALUES (?, ?, ?, ?, ?, 0, ?, ?)
ON DUPLICATE KEY UPDATE
  status = IF(VALUES(last_event_ts) >= last_event_ts, VALUES(status), status),
  content_user_id = IF(VALUES(last_event_ts) >= last_event_ts AND VALUES(content_user_id) <> 0, VALUES(content_user_id), content_user_id),
  updated_by = IF(VALUES(last_event_ts) >= last_event_ts, VALUES(updated_by), updated_by),
  is_deleted = 0,
  last_event_ts = GREATEST(last_event_ts, VALUES(last_event_ts)),
  updated_at = IF(VALUES(last_event_ts) >= last_event_ts, CURRENT_TIMESTAMP, updated_at)
`

	return r.db.WithContext(r.ctx).Exec(
		query,
		likeDO.UserID,
		likeDO.ContentID,
		likeDO.ContentUserID,
		likeDO.Status,
		likeDO.LastEventTs,
		likeDO.CreatedBy,
		likeDO.UpdatedBy,
	).Error
}
