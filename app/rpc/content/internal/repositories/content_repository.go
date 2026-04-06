package repositories

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"

	"zfeed/app/rpc/content/internal/do"
	"zfeed/app/rpc/content/internal/model"
)

type ContentRepository interface {
	WithTx(tx *gorm.DB) ContentRepository
	CreateContent(contentDO *do.ContentDO) (int64, error)
	ListLatestPublishedIDsByAuthor(authorID int64, limit int) ([]int64, error)
}

type contentRepositoryImpl struct {
	ctx context.Context
	db  *gorm.DB
	tx  *gorm.DB
	logx.Logger
}

func NewContentRepository(ctx context.Context, db *gorm.DB) ContentRepository {
	return &contentRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *contentRepositoryImpl) WithTx(tx *gorm.DB) ContentRepository {
	return &contentRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

func (r *contentRepositoryImpl) getDB() *gorm.DB {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

func (r *contentRepositoryImpl) CreateContent(contentDO *do.ContentDO) (int64, error) {
	row := &model.ZfeedContent{
		ID:            contentDO.ID,
		UserID:        contentDO.UserID,
		ContentType:   contentDO.ContentType,
		Status:        contentDO.Status,
		Visibility:    contentDO.Visibility,
		LikeCount:     contentDO.LikeCount,
		FavoriteCount: contentDO.FavoriteCount,
		CommentCount:  contentDO.CommentCount,
		PublishedAt:   contentDO.PublishedAt,
		IsDeleted:     contentDO.IsDeleted,
		CreatedBy:     contentDO.CreatedBy,
		UpdatedBy:     contentDO.UpdatedBy,
	}

	if err := r.getDB().WithContext(r.ctx).Create(row).Error; err != nil {
		return 0, err
	}

	return row.ID, nil
}

func (r *contentRepositoryImpl) ListLatestPublishedIDsByAuthor(authorID int64, limit int) ([]int64, error) {
	if authorID <= 0 {
		return []int64{}, nil
	}
	if limit <= 0 {
		limit = 20
	}

	rows := make([]*model.ZfeedContent, 0, limit)
	err := r.getDB().WithContext(r.ctx).
		Model(&model.ZfeedContent{}).
		Select("id").
		Where("user_id = ? AND status = ? AND visibility = ? AND is_deleted = 0", authorID, 30, 10).
		Order("id DESC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		if row == nil || row.ID <= 0 {
			continue
		}
		ids = append(ids, row.ID)
	}
	return ids, nil
}
