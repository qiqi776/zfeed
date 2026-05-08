package repositories

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"

	"zfeed/app/rpc/interaction/internal/model"
)

type FavoriteEventRepository interface {
	WithTx(tx *gorm.DB) FavoriteEventRepository
	Create(event *model.ZfeedFavoriteEvent) error
}

type favoriteEventRepositoryImpl struct {
	ctx context.Context
	db  *gorm.DB
	tx  *gorm.DB
	logx.Logger
}

func NewFavoriteEventRepository(ctx context.Context, db *gorm.DB) FavoriteEventRepository {
	return &favoriteEventRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *favoriteEventRepositoryImpl) WithTx(tx *gorm.DB) FavoriteEventRepository {
	return &favoriteEventRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

func (r *favoriteEventRepositoryImpl) getDB() *gorm.DB {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

func (r *favoriteEventRepositoryImpl) Create(event *model.ZfeedFavoriteEvent) error {
	if event == nil {
		return nil
	}

	return r.getDB().WithContext(r.ctx).Create(event).Error
}
