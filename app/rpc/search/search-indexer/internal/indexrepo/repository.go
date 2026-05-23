package indexrepo

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"zfeed/app/rpc/search/internal/common/indexdoc"
)

const (
	tableContent = "zfeed_content"
	tableArticle = "zfeed_article"
	tableVideo   = "zfeed_video"
	tableUser    = "zfeed_user"

	contentStatusPublished  = 30
	contentVisibilityPublic = 10
	userStatusActive        = 10
)

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) BuildContentDocument(ctx context.Context, contentID int64) (*indexdoc.ContentDocument, bool, error) {
	if r == nil || r.db == nil || contentID <= 0 {
		return nil, false, nil
	}

	var row contentIndexRow
	err := r.db.WithContext(ctx).
		Table(tableContent+" AS c").
		Select(`
			c.id AS content_id,
			c.content_type AS content_type,
			COALESCE(a.title, v.title, '') AS title,
			COALESCE(a.description, v.description, '') AS description,
			c.user_id AS author_id,
			COALESCE(u.nickname, '') AS author_name,
			COALESCE(u.avatar, '') AS author_avatar,
			c.published_at AS published_at,
			c.visibility AS visibility,
			c.status AS status,
			COALESCE(c.hot_score, 0) AS hot_score,
			c.is_deleted AS content_deleted
		`).
		Joins("LEFT JOIN "+tableArticle+" AS a ON a.content_id = c.id AND a.is_deleted = 0").
		Joins("LEFT JOIN "+tableVideo+" AS v ON v.content_id = c.id AND v.is_deleted = 0").
		Joins("LEFT JOIN "+tableUser+" AS u ON u.id = c.user_id AND u.is_deleted = 0").
		Where("c.id = ?", contentID).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if row.ContentDeleted != 0 ||
		row.Status != contentStatusPublished ||
		row.Visibility != contentVisibilityPublic ||
		row.PublishedAt == nil {
		return nil, false, nil
	}
	if row.Title == "" && row.Description == "" {
		return nil, false, nil
	}

	return &indexdoc.ContentDocument{
		ContentID:    row.ContentID,
		ContentType:  row.ContentType,
		Title:        row.Title,
		Description:  row.Description,
		AuthorID:     row.AuthorID,
		AuthorName:   row.AuthorName,
		AuthorAvatar: row.AuthorAvatar,
		PublishedAt:  row.PublishedAt.Unix(),
		Visibility:   row.Visibility,
		Status:       row.Status,
		HotScore:     row.HotScore,
	}, true, nil
}

func (r *Repository) BuildUserDocument(ctx context.Context, userID int64) (*indexdoc.UserDocument, bool, error) {
	if r == nil || r.db == nil || userID <= 0 {
		return nil, false, nil
	}

	var row userIndexRow
	err := r.db.WithContext(ctx).
		Table(tableUser).
		Select("id AS user_id", "nickname", "bio", "mobile AS mobile_search_field", "status", "is_deleted").
		Where("id = ?", userID).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if row.IsDeleted != 0 || row.Status != userStatusActive {
		return nil, false, nil
	}

	return &indexdoc.UserDocument{
		UserID:            row.UserID,
		Nickname:          row.Nickname,
		Bio:               row.Bio,
		MobileSearchField: row.MobileSearchField,
		Status:            row.Status,
	}, true, nil
}

func (r *Repository) ListContentIDsByAuthor(ctx context.Context, authorID int64, limit int) ([]int64, error) {
	if r == nil || r.db == nil || authorID <= 0 {
		return []int64{}, nil
	}
	if limit <= 0 {
		limit = 200
	}

	rows := make([]struct {
		ID int64 `gorm:"column:id"`
	}, 0, limit)
	err := r.db.WithContext(ctx).
		Table(tableContent).
		Select("id").
		Where("user_id = ?", authorID).
		Where("is_deleted = 0").
		Order("id DESC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		if row.ID > 0 {
			ids = append(ids, row.ID)
		}
	}
	return ids, nil
}

type contentIndexRow struct {
	ContentID      int64      `gorm:"column:content_id"`
	ContentType    int32      `gorm:"column:content_type"`
	Title          string     `gorm:"column:title"`
	Description    string     `gorm:"column:description"`
	AuthorID       int64      `gorm:"column:author_id"`
	AuthorName     string     `gorm:"column:author_name"`
	AuthorAvatar   string     `gorm:"column:author_avatar"`
	PublishedAt    *time.Time `gorm:"column:published_at"`
	Visibility     int32      `gorm:"column:visibility"`
	Status         int32      `gorm:"column:status"`
	HotScore       float64    `gorm:"column:hot_score"`
	ContentDeleted int32      `gorm:"column:content_deleted"`
}

type userIndexRow struct {
	UserID            int64  `gorm:"column:user_id"`
	Nickname          string `gorm:"column:nickname"`
	Bio               string `gorm:"column:bio"`
	MobileSearchField string `gorm:"column:mobile_search_field"`
	Status            int32  `gorm:"column:status"`
	IsDeleted         int32  `gorm:"column:is_deleted"`
}
