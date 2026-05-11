package repositories

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	contentStatusPublished  = 30
	contentVisibilityPublic = 10
	followActiveStatus      = 10
)

type SearchRepository interface {
	SearchUsers(query string, cursor int64, limit int) ([]SearchUserRow, error)
	BatchFollowing(viewerID int64, userIDs []int64) (map[int64]bool, error)
	SearchContents(query string, cursor int64, limit int) ([]SearchContentRow, error)
}

type SearchUserRow struct {
	UserID   int64  `gorm:"column:user_id"`
	Nickname string `gorm:"column:nickname"`
	Avatar   string `gorm:"column:avatar"`
	Bio      string `gorm:"column:bio"`
}

type SearchContentRow struct {
	ContentID    int64      `gorm:"column:content_id"`
	ContentType  int32      `gorm:"column:content_type"`
	AuthorID     int64      `gorm:"column:author_id"`
	AuthorName   string     `gorm:"column:author_name"`
	AuthorAvatar string     `gorm:"column:author_avatar"`
	Title        string     `gorm:"column:title"`
	CoverURL     string     `gorm:"column:cover_url"`
	PublishedAt  *time.Time `gorm:"column:published_at"`
}

type followStateRow struct {
	FollowUserID int64 `gorm:"column:follow_user_id"`
}

type searchRepositoryImpl struct {
	ctx context.Context
	db  *gorm.DB
}

func NewSearchRepository(ctx context.Context, db *gorm.DB) SearchRepository {
	return &searchRepositoryImpl{
		ctx: ctx,
		db:  db,
	}
}

func (r *searchRepositoryImpl) SearchUsers(query string, cursor int64, limit int) ([]SearchUserRow, error) {
	rows := make([]SearchUserRow, 0, limit)
	if r.db == nil {
		return rows, nil
	}

	trimmed := strings.TrimSpace(query)
	if isMySQL(r.db) {
		ftRows, err := r.searchUsersFullText(trimmed, cursor, limit)
		if err == nil && len(ftRows) > 0 {
			return ftRows, nil
		}
	}

	pattern := "%" + trimmed + "%"
	dbQuery := r.db.WithContext(r.ctx).
		Table("zfeed_user").
		Select("id AS user_id", "nickname", "avatar", "bio").
		Where("is_deleted = 0").
		Where("(nickname LIKE ? OR bio LIKE ? OR mobile LIKE ?)", pattern, pattern, pattern)

	if cursor > 0 {
		dbQuery = dbQuery.Where("id < ?", cursor)
	}

	err := dbQuery.Order("id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *searchRepositoryImpl) searchUsersFullText(query string, cursor int64, limit int) ([]SearchUserRow, error) {
	rows := make([]SearchUserRow, 0, limit)
	fullTextQuery := buildBooleanFullTextQuery(query)
	if fullTextQuery == "" {
		return rows, nil
	}

	dbQuery := r.db.WithContext(r.ctx).
		Table("zfeed_user").
		Select("id AS user_id", "nickname", "avatar", "bio").
		Where("is_deleted = 0").
		Where("MATCH(nickname, bio, mobile) AGAINST(? IN BOOLEAN MODE)", fullTextQuery)

	if cursor > 0 {
		dbQuery = dbQuery.Where("id < ?", cursor)
	}

	err := dbQuery.Order("id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *searchRepositoryImpl) BatchFollowing(viewerID int64, userIDs []int64) (map[int64]bool, error) {
	result := make(map[int64]bool)
	if viewerID <= 0 || len(userIDs) == 0 || r.db == nil {
		return result, nil
	}

	rows := make([]followStateRow, 0, len(userIDs))
	err := r.db.WithContext(r.ctx).
		Table("zfeed_follow").
		Select("follow_user_id").
		Where("user_id = ? AND follow_user_id IN ? AND status = ? AND is_deleted = 0", viewerID, userIDs, followActiveStatus).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		result[row.FollowUserID] = true
	}
	return result, nil
}

func (r *searchRepositoryImpl) SearchContents(query string, cursor int64, limit int) ([]SearchContentRow, error) {
	rows := make([]SearchContentRow, 0, limit)
	if r.db == nil {
		return rows, nil
	}

	trimmed := strings.TrimSpace(query)
	if isMySQL(r.db) {
		ftRows, err := r.searchContentsFullText(trimmed, cursor, limit)
		if err == nil && len(ftRows) > 0 {
			return ftRows, nil
		}
	}

	pattern := "%" + trimmed + "%"
	dbQuery := r.db.WithContext(r.ctx).
		Table("zfeed_content AS c").
		Select(`
			c.id AS content_id,
			c.content_type AS content_type,
			c.user_id AS author_id,
			COALESCE(u.nickname, '') AS author_name,
			COALESCE(u.avatar, '') AS author_avatar,
			COALESCE(a.title, v.title, '') AS title,
			COALESCE(a.cover, v.cover_url, '') AS cover_url,
			c.published_at AS published_at
		`).
		Joins("LEFT JOIN zfeed_article AS a ON a.content_id = c.id AND a.is_deleted = 0").
		Joins("LEFT JOIN zfeed_video AS v ON v.content_id = c.id AND v.is_deleted = 0").
		Joins("LEFT JOIN zfeed_user AS u ON u.id = c.user_id AND u.is_deleted = 0").
		Where("c.status = ? AND c.visibility = ? AND c.is_deleted = 0", contentStatusPublished, contentVisibilityPublic).
		Where("(a.title LIKE ? OR v.title LIKE ? OR a.description LIKE ? OR v.description LIKE ?)", pattern, pattern, pattern, pattern)

	if cursor > 0 {
		dbQuery = dbQuery.Where("c.id < ?", cursor)
	}

	err := dbQuery.Order("c.id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *searchRepositoryImpl) searchContentsFullText(query string, cursor int64, limit int) ([]SearchContentRow, error) {
	rows := make([]SearchContentRow, 0, limit)
	fullTextQuery := buildBooleanFullTextQuery(query)
	if fullTextQuery == "" {
		return rows, nil
	}

	dbQuery := r.db.WithContext(r.ctx).
		Table("zfeed_content AS c").
		Select(`
			c.id AS content_id,
			c.content_type AS content_type,
			c.user_id AS author_id,
			COALESCE(u.nickname, '') AS author_name,
			COALESCE(u.avatar, '') AS author_avatar,
			COALESCE(a.title, v.title, '') AS title,
			COALESCE(a.cover, v.cover_url, '') AS cover_url,
			c.published_at AS published_at
		`).
		Joins("LEFT JOIN zfeed_article AS a ON a.content_id = c.id AND a.is_deleted = 0").
		Joins("LEFT JOIN zfeed_video AS v ON v.content_id = c.id AND v.is_deleted = 0").
		Joins("LEFT JOIN zfeed_user AS u ON u.id = c.user_id AND u.is_deleted = 0").
		Where("c.status = ? AND c.visibility = ? AND c.is_deleted = 0", contentStatusPublished, contentVisibilityPublic).
		Where(
			"(MATCH(a.title, a.description) AGAINST(? IN BOOLEAN MODE) OR MATCH(v.title, v.description) AGAINST(? IN BOOLEAN MODE))",
			fullTextQuery,
			fullTextQuery,
		)

	if cursor > 0 {
		dbQuery = dbQuery.Where("c.id < ?", cursor)
	}

	err := dbQuery.Order("c.id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func isMySQL(db *gorm.DB) bool {
	return db != nil && strings.EqualFold(db.Dialector.Name(), "mysql")
}

func buildBooleanFullTextQuery(query string) string {
	terms := strings.Fields(strings.TrimSpace(query))
	if len(terms) == 0 {
		return ""
	}

	parts := make([]string, 0, len(terms))
	for _, term := range terms {
		term = sanitizeBooleanFullTextTerm(term)
		if term == "" {
			continue
		}
		parts = append(parts, "+"+term+"*")
	}
	return strings.Join(parts, " ")
}

func sanitizeBooleanFullTextTerm(term string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '+', '-', '>', '<', '(', ')', '~', '*', ':', '"', '&', '|', '@':
			return -1
		default:
			return r
		}
	}, term)
}
