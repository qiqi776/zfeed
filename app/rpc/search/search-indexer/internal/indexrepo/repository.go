package indexrepo

import (
	"context"
	"errors"
	"sort"
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

func (r *Repository) ListContentDocumentsAfter(ctx context.Context, cursorID int64, endID int64, limit int) ([]indexdoc.ContentDocument, error) {
	if r == nil || r.db == nil {
		return []indexdoc.ContentDocument{}, nil
	}
	if limit <= 0 {
		limit = 100
	}

	rows := make([]contentIndexRow, 0, limit)
	query := r.contentDocumentQuery(ctx).
		Where("c.id > ?", cursorID).
		Order("c.id ASC").
		Limit(limit)
	if endID > 0 {
		query = query.Where("c.id <= ?", endID)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	return contentDocumentsFromRows(rows), nil
}

func (r *Repository) ListUserDocumentsAfter(ctx context.Context, cursorID int64, endID int64, limit int) ([]indexdoc.UserDocument, error) {
	if r == nil || r.db == nil {
		return []indexdoc.UserDocument{}, nil
	}
	if limit <= 0 {
		limit = 100
	}

	rows := make([]userIndexRow, 0, limit)
	query := r.db.WithContext(ctx).
		Table(tableUser).
		Select("id AS user_id", "nickname", "bio", "mobile AS mobile_search_field", "status", "is_deleted").
		Where("id > ?", cursorID).
		Where("status = ? AND is_deleted = 0", userStatusActive).
		Order("id ASC").
		Limit(limit)
	if endID > 0 {
		query = query.Where("id <= ?", endID)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}

	docs := make([]indexdoc.UserDocument, 0, len(rows))
	for _, row := range rows {
		docs = append(docs, indexdoc.UserDocument{
			UserID:            row.UserID,
			Nickname:          row.Nickname,
			Bio:               row.Bio,
			MobileSearchField: row.MobileSearchField,
			Status:            row.Status,
		})
	}
	return docs, nil
}

func (r *Repository) CountContentDocuments(ctx context.Context) (int64, error) {
	if r == nil || r.db == nil {
		return 0, nil
	}
	var count int64
	err := r.contentDocumentQuery(ctx).Count(&count).Error
	return count, err
}

func (r *Repository) CountUserDocuments(ctx context.Context) (int64, error) {
	if r == nil || r.db == nil {
		return 0, nil
	}
	var count int64
	err := r.db.WithContext(ctx).
		Table(tableUser).
		Where("status = ? AND is_deleted = 0", userStatusActive).
		Count(&count).Error
	return count, err
}

func (r *Repository) SampleContentDocuments(ctx context.Context, limit int) ([]indexdoc.ContentDocument, error) {
	if limit <= 0 {
		limit = 20
	}
	rows := make([]contentIndexRow, 0, limit)
	err := r.contentDocumentQuery(ctx).
		Order("c.id ASC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return contentDocumentsFromRows(rows), nil
}

func (r *Repository) SampleUserDocuments(ctx context.Context, limit int) ([]indexdoc.UserDocument, error) {
	if limit <= 0 {
		limit = 20
	}
	return r.ListUserDocumentsAfter(ctx, 0, 0, limit)
}

func (r *Repository) SearchContentIDs(ctx context.Context, query string, mode string, limit int) ([]int64, error) {
	if r == nil || r.db == nil {
		return []int64{}, nil
	}
	if limit <= 0 {
		limit = 20
	}
	pattern := "%" + query + "%"
	rows := make([]struct {
		ContentID int64   `gorm:"column:content_id"`
		TextScore float64 `gorm:"column:text_score"`
		HotScore  float64 `gorm:"column:hot_score"`
	}, 0, limit)
	err := r.contentDocumentQuery(ctx).
		Select(`
			c.id AS content_id,
			COALESCE(c.hot_score, 0) AS hot_score,
			CASE
				WHEN COALESCE(a.title, v.title, '') = ? THEN 30
				WHEN COALESCE(a.title, v.title, '') LIKE ? THEN 20
				WHEN COALESCE(a.title, v.title, '') LIKE ? THEN 10
				WHEN COALESCE(a.description, v.description, '') LIKE ? THEN 5
				ELSE 1
			END AS text_score
		`, query, query+"%", pattern, pattern).
		Where("(a.title LIKE ? OR v.title LIKE ? OR a.description LIKE ? OR v.description LIKE ?)", pattern, pattern, pattern, pattern).
		Order(contentSearchOrder(mode)).
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	if mode == "hybrid" {
		sort.SliceStable(rows, func(i, j int) bool {
			left := rows[i].TextScore + rows[i].HotScore
			right := rows[j].TextScore + rows[j].HotScore
			if left != right {
				return left > right
			}
			return rows[i].ContentID > rows[j].ContentID
		})
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ContentID)
	}
	return ids, nil
}

func (r *Repository) SearchUserIDs(ctx context.Context, query string, limit int) ([]int64, error) {
	if r == nil || r.db == nil {
		return []int64{}, nil
	}
	if limit <= 0 {
		limit = 20
	}
	pattern := "%" + query + "%"
	rows := make([]struct {
		UserID int64 `gorm:"column:user_id"`
	}, 0, limit)
	err := r.db.WithContext(ctx).
		Table(tableUser).
		Select("id AS user_id").
		Where("status = ? AND is_deleted = 0", userStatusActive).
		Where("(nickname LIKE ? OR bio LIKE ? OR mobile LIKE ?)", pattern, pattern, pattern).
		Order("id DESC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.UserID)
	}
	return ids, nil
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

func (r *Repository) contentDocumentQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).
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
		Where("c.status = ? AND c.visibility = ? AND c.is_deleted = 0", contentStatusPublished, contentVisibilityPublic).
		Where("c.published_at IS NOT NULL").
		Where("(COALESCE(a.title, v.title, '') <> '' OR COALESCE(a.description, v.description, '') <> '')")
}

func contentDocumentsFromRows(rows []contentIndexRow) []indexdoc.ContentDocument {
	docs := make([]indexdoc.ContentDocument, 0, len(rows))
	for _, row := range rows {
		if row.PublishedAt == nil {
			continue
		}
		docs = append(docs, indexdoc.ContentDocument{
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
		})
	}
	return docs
}

func contentSearchOrder(mode string) string {
	switch mode {
	case "relevance", "hybrid":
		return "text_score DESC, c.id DESC"
	default:
		return "c.published_at DESC, c.id DESC"
	}
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
