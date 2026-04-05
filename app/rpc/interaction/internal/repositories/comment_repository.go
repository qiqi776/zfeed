package repositories

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"

	"zfeed/app/rpc/interaction/internal/do"
)

const (
	CommentStatusNormal  int32 = 10
	CommentStatusDeleted int32 = 20
)

type CommentRepository interface {
	WithTx(tx *gorm.DB) CommentRepository
	Create(commentDO *do.CommentDO) (int64, error)
	GetByID(commentID int64) (*do.CommentDO, error)
	GetByIDIncludeDeleted(commentID int64) (*do.CommentDO, error)
	ListRootComments(contentID, cursor int64, pageSize uint32) ([]*do.CommentDO, error)
	ListRootCommentsIncludeDeleted(contentID, cursor int64, pageSize uint32) ([]*do.CommentDO, error)
	ListReplies(rootID, cursor int64, pageSize uint32) ([]*do.CommentDO, error)
	ListRepliesIncludeDeleted(rootID, cursor int64, pageSize uint32) ([]*do.CommentDO, error)
	BatchGetByIDs(commentIDs []int64) (map[int64]*do.CommentDO, error)
	BatchGetByIDsIncludeDeleted(commentIDs []int64) (map[int64]*do.CommentDO, error)
	IncReplyCount(commentID int64) error
	DecReplyCount(commentID int64) error
	MarkDeleted(commentID int64, updatedBy int64) error
	DeleteByID(commentID int64) error
	HasChildren(commentID int64) (bool, error)
}

type commentRepositoryImpl struct {
	ctx context.Context
	db  *gorm.DB
	tx  *gorm.DB
	logx.Logger
}

type commentRow struct {
	ID            int64     `gorm:"column:id"`
	ContentID     int64     `gorm:"column:content_id"`
	ContentUserID int64     `gorm:"column:content_user_id"`
	UserID        int64     `gorm:"column:user_id"`
	ReplyToUserID int64     `gorm:"column:reply_to_user_id"`
	ParentID      int64     `gorm:"column:parent_id"`
	RootID        int64     `gorm:"column:root_id"`
	Comment       string    `gorm:"column:comment"`
	Status        int32     `gorm:"column:status"`
	Version       int32     `gorm:"column:version"`
	ReplyCount    int64     `gorm:"column:reply_count"`
	IsDeleted     int32     `gorm:"column:is_deleted"`
	CreatedBy     int64     `gorm:"column:created_by"`
	UpdatedBy     int64     `gorm:"column:updated_by"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

func NewCommentRepository(ctx context.Context, db *gorm.DB) CommentRepository {
	return &commentRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *commentRepositoryImpl) WithTx(tx *gorm.DB) CommentRepository {
	return &commentRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

func (r *commentRepositoryImpl) getDB() *gorm.DB {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

func (r *commentRepositoryImpl) Create(commentDO *do.CommentDO) (int64, error) {
	version := commentDO.Version
	if version <= 0 {
		version = 1
	}

	row := &commentRow{
		ID:            commentDO.ID,
		ContentID:     commentDO.ContentID,
		ContentUserID: commentDO.ContentUserID,
		UserID:        commentDO.UserID,
		ReplyToUserID: commentDO.ReplyToUserID,
		ParentID:      commentDO.ParentID,
		RootID:        commentDO.RootID,
		Comment:       commentDO.Comment,
		Status:        commentDO.Status,
		Version:       version,
		ReplyCount:    commentDO.ReplyCount,
		IsDeleted:     commentDO.IsDeleted,
		CreatedBy:     commentDO.CreatedBy,
		UpdatedBy:     commentDO.UpdatedBy,
	}

	if err := r.getDB().WithContext(r.ctx).Table("zfeed_comment").Omit("CreatedAt", "UpdatedAt").Create(row).Error; err != nil {
		return 0, err
	}

	return row.ID, nil
}

func (r *commentRepositoryImpl) GetByID(commentID int64) (*do.CommentDO, error) {
	return r.getByID(commentID, false)
}

func (r *commentRepositoryImpl) GetByIDIncludeDeleted(commentID int64) (*do.CommentDO, error) {
	return r.getByID(commentID, true)
}

func (r *commentRepositoryImpl) ListRootComments(contentID, cursor int64, pageSize uint32) ([]*do.CommentDO, error) {
	return r.listRootComments(contentID, cursor, pageSize, false)
}

func (r *commentRepositoryImpl) ListRootCommentsIncludeDeleted(contentID, cursor int64, pageSize uint32) ([]*do.CommentDO, error) {
	return r.listRootComments(contentID, cursor, pageSize, true)
}

func (r *commentRepositoryImpl) ListReplies(rootID, cursor int64, pageSize uint32) ([]*do.CommentDO, error) {
	return r.listReplies(rootID, cursor, pageSize, false)
}

func (r *commentRepositoryImpl) ListRepliesIncludeDeleted(rootID, cursor int64, pageSize uint32) ([]*do.CommentDO, error) {
	return r.listReplies(rootID, cursor, pageSize, true)
}

func (r *commentRepositoryImpl) BatchGetByIDs(commentIDs []int64) (map[int64]*do.CommentDO, error) {
	return r.batchGetByIDs(commentIDs, false)
}

func (r *commentRepositoryImpl) BatchGetByIDsIncludeDeleted(commentIDs []int64) (map[int64]*do.CommentDO, error) {
	return r.batchGetByIDs(commentIDs, true)
}

func (r *commentRepositoryImpl) MarkDeleted(commentID int64, updatedBy int64) error {
	if commentID <= 0 {
		return nil
	}

	return r.getDB().WithContext(r.ctx).
		Table("zfeed_comment").
		Where("id = ?", commentID).
		Updates(map[string]any{
			"is_deleted": 1,
			"status":     CommentStatusDeleted,
			"updated_by": updatedBy,
		}).Error
}

func (r *commentRepositoryImpl) DeleteByID(commentID int64) error {
	if commentID <= 0 {
		return nil
	}

	return r.getDB().WithContext(r.ctx).
		Table("zfeed_comment").
		Where("id = ?", commentID).
		Delete(nil).Error
}

func (r *commentRepositoryImpl) HasChildren(commentID int64) (bool, error) {
	if commentID <= 0 {
		return false, nil
	}

	var count int64
	if err := r.getDB().WithContext(r.ctx).
		Table("zfeed_comment").
		Where("parent_id = ?", commentID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *commentRepositoryImpl) getByID(commentID int64, includeDeleted bool) (*do.CommentDO, error) {
	if commentID <= 0 {
		return nil, nil
	}

	var row commentRow
	query := r.getDB().WithContext(r.ctx).
		Table("zfeed_comment").
		Where("id = ?", commentID)
	if !includeDeleted {
		query = query.Where("is_deleted = 0 AND status = ?", CommentStatusNormal)
	}

	err := query.Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return row.toDO(), nil
}

func (r *commentRepositoryImpl) listRootComments(contentID, cursor int64, pageSize uint32, includeDeleted bool) ([]*do.CommentDO, error) {
	var rows []commentRow
	query := r.getDB().WithContext(r.ctx).
		Table("zfeed_comment").
		Where("content_id = ? AND root_id = 0 AND id < ?", contentID, normalizeCommentCursor(cursor))
	if !includeDeleted {
		query = query.Where("is_deleted = 0 AND status = ?", CommentStatusNormal)
	}

	err := query.
		Order("id DESC").
		Limit(int(pageSize) + 1).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	return toCommentDOs(rows), nil
}

func (r *commentRepositoryImpl) listReplies(rootID, cursor int64, pageSize uint32, includeDeleted bool) ([]*do.CommentDO, error) {
	var rows []commentRow
	query := r.getDB().WithContext(r.ctx).
		Table("zfeed_comment").
		Where("root_id = ? AND id < ?", rootID, normalizeCommentCursor(cursor))
	if !includeDeleted {
		query = query.Where("is_deleted = 0 AND status = ?", CommentStatusNormal)
	}

	err := query.
		Order("id DESC").
		Limit(int(pageSize) + 1).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	return toCommentDOs(rows), nil
}

func (r *commentRepositoryImpl) batchGetByIDs(commentIDs []int64, includeDeleted bool) (map[int64]*do.CommentDO, error) {
	ids := uniquePositiveInt64s(commentIDs)
	if len(ids) == 0 {
		return map[int64]*do.CommentDO{}, nil
	}

	var rows []commentRow
	query := r.getDB().WithContext(r.ctx).
		Table("zfeed_comment").
		Where("id IN ?", ids)
	if !includeDeleted {
		query = query.Where("is_deleted = 0 AND status = ?", CommentStatusNormal)
	}

	err := query.Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[int64]*do.CommentDO, len(rows))
	for _, row := range rows {
		result[row.ID] = row.toDO()
	}
	return result, nil
}

func (r *commentRepositoryImpl) IncReplyCount(commentID int64) error {
	if commentID <= 0 {
		return nil
	}

	return r.getDB().WithContext(r.ctx).
		Table("zfeed_comment").
		Where("id = ? AND is_deleted = 0 AND status = ?", commentID, CommentStatusNormal).
		UpdateColumn("reply_count", gorm.Expr("reply_count + 1")).Error
}

func (r *commentRepositoryImpl) DecReplyCount(commentID int64) error {
	if commentID <= 0 {
		return nil
	}

	return r.getDB().WithContext(r.ctx).
		Table("zfeed_comment").
		Where("id = ? AND is_deleted = 0", commentID).
		UpdateColumn("reply_count", gorm.Expr("GREATEST(reply_count - 1, 0)")).Error
}

func normalizeCommentCursor(cursor int64) int64 {
	if cursor <= 0 {
		return math.MaxInt64
	}
	return cursor
}

func uniquePositiveInt64s(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func toCommentDOs(rows []commentRow) []*do.CommentDO {
	result := make([]*do.CommentDO, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.toDO())
	}
	return result
}

func (r commentRow) toDO() *do.CommentDO {
	return &do.CommentDO{
		ID:            r.ID,
		ContentID:     r.ContentID,
		ContentUserID: r.ContentUserID,
		UserID:        r.UserID,
		ReplyToUserID: r.ReplyToUserID,
		ParentID:      r.ParentID,
		RootID:        r.RootID,
		Comment:       r.Comment,
		Status:        r.Status,
		Version:       r.Version,
		ReplyCount:    r.ReplyCount,
		IsDeleted:     r.IsDeleted,
		CreatedBy:     r.CreatedBy,
		UpdatedBy:     r.UpdatedBy,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}
}
