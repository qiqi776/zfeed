package repositories

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"

	"zfeed/app/rpc/content/internal/do"
	"zfeed/app/rpc/content/internal/model"
)

type ArticleRepository interface {
	WithTx(tx *gorm.DB) ArticleRepository
	CreateArticle(articleDO *do.ArticleDO) error
	BatchGetBriefByContentIDs(contentIDs []int64) (map[int64]*model.ZfeedArticle, error)
}

type articleRepositoryImpl struct {
	ctx context.Context
	db  *gorm.DB
	tx  *gorm.DB
	logx.Logger
}

func NewArticleRepository(ctx context.Context, db *gorm.DB) ArticleRepository {
	return &articleRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *articleRepositoryImpl) WithTx(tx *gorm.DB) ArticleRepository {
	return &articleRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

func (r *articleRepositoryImpl) getDB() *gorm.DB {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

func (r *articleRepositoryImpl) CreateArticle(articleDO *do.ArticleDO) error {
	row := &model.ZfeedArticle{
		ID:          articleDO.ID,
		ContentID:   articleDO.ContentID,
		Title:       articleDO.Title,
		Description: articleDO.Description,
		Cover:       articleDO.Cover,
		Content:     articleDO.Content,
		IsDeleted:   articleDO.IsDeleted,
	}

	return r.getDB().WithContext(r.ctx).Create(row).Error
}

func (r *articleRepositoryImpl) BatchGetBriefByContentIDs(contentIDs []int64) (map[int64]*model.ZfeedArticle, error) {
	if len(contentIDs) == 0 {
		return map[int64]*model.ZfeedArticle{}, nil
	}

	rows := make([]*model.ZfeedArticle, 0, len(contentIDs))
	err := r.getDB().WithContext(r.ctx).
		Model(&model.ZfeedArticle{}).
		Select("content_id", "title", "cover").
		Where("content_id IN ?", contentIDs).
		Where("is_deleted = 0").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[int64]*model.ZfeedArticle, len(rows))
	for _, row := range rows {
		if row == nil || row.ContentID <= 0 {
			continue
		}
		result[row.ContentID] = row
	}
	return result, nil
}
