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
