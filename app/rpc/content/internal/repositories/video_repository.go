package repositories

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"

	"zfeed/app/rpc/content/internal/do"
	"zfeed/app/rpc/content/internal/model"
)

type VideoRepository interface {
	WithTx(tx *gorm.DB) VideoRepository
	CreateVideo(videoDO *do.VideoDO) error
	BatchGetBriefByContentIDs(contentIDs []int64) (map[int64]*model.ZfeedVideo, error)
}

type videoRepositoryImpl struct {
	ctx context.Context
	db  *gorm.DB
	tx  *gorm.DB
	logx.Logger
}

func NewVideoRepository(ctx context.Context, db *gorm.DB) VideoRepository {
	return &videoRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *videoRepositoryImpl) WithTx(tx *gorm.DB) VideoRepository {
	return &videoRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

func (r *videoRepositoryImpl) getDB() *gorm.DB {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

func (r *videoRepositoryImpl) CreateVideo(videoDO *do.VideoDO) error {
	row := &model.ZfeedVideo{
		ID:              videoDO.ID,
		ContentID:       videoDO.ContentID,
		Title:           videoDO.Title,
		Description:     videoDO.Description,
		OriginURL:       videoDO.OriginURL,
		CoverURL:        videoDO.CoverURL,
		Duration:        videoDO.Duration,
		TranscodeStatus: videoDO.TranscodeStatus,
		IsDeleted:       videoDO.IsDeleted,
	}

	return r.getDB().WithContext(r.ctx).Create(row).Error
}

func (r *videoRepositoryImpl) BatchGetBriefByContentIDs(contentIDs []int64) (map[int64]*model.ZfeedVideo, error) {
	if len(contentIDs) == 0 {
		return map[int64]*model.ZfeedVideo{}, nil
	}

	rows := make([]*model.ZfeedVideo, 0, len(contentIDs))
	err := r.getDB().WithContext(r.ctx).
		Model(&model.ZfeedVideo{}).
		Select("content_id", "title", "cover_url").
		Where("content_id IN ?", contentIDs).
		Where("is_deleted = 0").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[int64]*model.ZfeedVideo, len(rows))
	for _, row := range rows {
		if row == nil || row.ContentID <= 0 {
			continue
		}
		result[row.ContentID] = row
	}
	return result, nil
}
