package backend

import (
	"context"

	"gorm.io/gorm"

	"zfeed/app/rpc/search/search-rpc/internal/repositories"
)

type MySQLBackend struct {
	db *gorm.DB
}

func NewMySQLBackend(db *gorm.DB) *MySQLBackend {
	return &MySQLBackend{db: db}
}

func (b *MySQLBackend) Name() string {
	return NameMySQL
}

func (b *MySQLBackend) SearchUsers(
	ctx context.Context,
	query string,
	cursor int64,
	limit int,
) (SearchUsersResult, error) {
	repo := repositories.NewSearchRepository(ctx, b.db)
	result, err := repo.SearchUsersWithMeta(query, cursor, limit)
	return SearchUsersResult{
		Rows: result.Rows,
		Meta: result.Meta,
	}, err
}

func (b *MySQLBackend) SearchContents(
	ctx context.Context,
	query string,
	mode string,
	cursor int64,
	limit int,
) (SearchContentsResult, error) {
	repo := repositories.NewSearchRepository(ctx, b.db)
	result, err := repo.SearchContentsWithMeta(query, mode, cursor, limit)
	return SearchContentsResult{
		Rows: result.Rows,
		Meta: result.Meta,
	}, err
}
