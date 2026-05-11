package repositories

import (
	"context"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"

	"zfeed/app/rpc/interaction/internal/do"
)

const (
	LikeStatusLike   int32 = 10
	LikeStatusCancel int32 = 20
)

type LikeTarget struct {
	Scene     int32
	ContentID int64
}

type LikeRepository interface {
	Upsert(likeDO *do.LikeDO) error
	CountByTarget(scene int32, contentID int64) (int64, error)
	CountByTargets(targets []LikeTarget) (map[string]int64, error)
	IsLiked(userID int64, scene int32, contentID int64) (bool, error)
	BatchIsLiked(userID int64, targets []LikeTarget) (map[string]bool, error)
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
  scene,
  content_id,
  content_user_id,
  status,
  last_event_ts,
  is_deleted,
  created_by,
  updated_by
) VALUES (?, ?, ?, ?, ?, ?, 0, ?, ?)
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
		likeDO.Scene,
		likeDO.ContentID,
		likeDO.ContentUserID,
		likeDO.Status,
		likeDO.LastEventTs,
		likeDO.CreatedBy,
		likeDO.UpdatedBy,
	).Error
}

func likeTargetKey(scene int32, contentID int64) string {
	return strconv.FormatInt(int64(scene), 10) + ":" + strconv.FormatInt(contentID, 10)
}

func (r *likeRepositoryImpl) CountByTarget(scene int32, contentID int64) (int64, error) {
	if contentID <= 0 {
		return 0, nil
	}

	var count int64
	err := r.db.WithContext(r.ctx).
		Table("zfeed_like").
		Where("scene = ? AND content_id = ? AND status = ? AND is_deleted = 0", scene, contentID, LikeStatusLike).
		Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *likeRepositoryImpl) CountByTargets(targets []LikeTarget) (map[string]int64, error) {
	result := make(map[string]int64, len(targets))
	targets = uniqueLikeTargets(targets)
	if len(targets) == 0 {
		return result, nil
	}

	type row struct {
		Scene     int32 `gorm:"column:scene"`
		ContentID int64 `gorm:"column:content_id"`
		Count     int64 `gorm:"column:count"`
	}

	rows := make([]row, 0)
	query := r.db.WithContext(r.ctx).
		Table("zfeed_like").
		Select("scene, content_id, COUNT(*) AS count").
		Where("status = ? AND is_deleted = 0", LikeStatusLike)

	query = applyLikeTargetFilter(query, targets)
	err := query.Group("scene, content_id").Find(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, item := range rows {
		result[likeTargetKey(item.Scene, item.ContentID)] = item.Count
	}
	return result, nil
}

func (r *likeRepositoryImpl) IsLiked(userID int64, scene int32, contentID int64) (bool, error) {
	if userID <= 0 || contentID <= 0 {
		return false, nil
	}

	var count int64
	err := r.db.WithContext(r.ctx).
		Table("zfeed_like").
		Where("user_id = ? AND scene = ? AND content_id = ? AND status = ? AND is_deleted = 0", userID, scene, contentID, LikeStatusLike).
		Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *likeRepositoryImpl) BatchIsLiked(userID int64, targets []LikeTarget) (map[string]bool, error) {
	result := make(map[string]bool, len(targets))
	targets = uniqueLikeTargets(targets)
	if userID <= 0 || len(targets) == 0 {
		return result, nil
	}

	type row struct {
		Scene     int32 `gorm:"column:scene"`
		ContentID int64 `gorm:"column:content_id"`
	}

	rows := make([]row, 0)
	query := r.db.WithContext(r.ctx).
		Table("zfeed_like").
		Select("scene, content_id").
		Where("user_id = ? AND status = ? AND is_deleted = 0", userID, LikeStatusLike)

	query = applyLikeTargetFilter(query, targets)
	err := query.Find(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, item := range rows {
		result[likeTargetKey(item.Scene, item.ContentID)] = true
	}
	return result, nil
}

func applyLikeTargetFilter(query *gorm.DB, targets []LikeTarget) *gorm.DB {
	if len(targets) == 0 {
		return query.Where("1 = 0")
	}

	clauses := make([]string, 0, len(targets))
	args := make([]any, 0, len(targets)*2)
	for _, target := range uniqueLikeTargets(targets) {
		clauses = append(clauses, "(scene = ? AND content_id = ?)")
		args = append(args, target.Scene, target.ContentID)
	}

	return query.Where(strings.Join(clauses, " OR "), args...)
}

func uniqueLikeTargets(targets []LikeTarget) []LikeTarget {
	seen := make(map[string]struct{}, len(targets))
	result := make([]LikeTarget, 0, len(targets))
	for _, target := range targets {
		if target.ContentID <= 0 || target.Scene <= 0 {
			continue
		}
		key := likeTargetKey(target.Scene, target.ContentID)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, target)
	}
	return result
}
