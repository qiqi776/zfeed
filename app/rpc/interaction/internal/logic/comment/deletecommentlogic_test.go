package commentlogic

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"testing"
	"time"

	"gorm.io/gorm"

	"zfeed/app/rpc/interaction/interaction"
	rediskey "zfeed/app/rpc/interaction/internal/common/consts/redis"
	"zfeed/app/rpc/interaction/internal/do"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/pkg/errorx"
)

func TestDeleteCommentRoot(t *testing.T) {
	ctx := context.Background()
	db := newCommentUserActionTestDB(t)
	store, redisClient := newCommentCacheRedis(t)

	const (
		contentID int64 = 9801
		rootID    int64 = 980101
		replyID   int64 = 980102
		rootUser  int64 = 981
		replyUser int64 = 982
	)
	seedDeleteCommentRows(t, db,
		commentTestComment{
			ID:         rootID,
			ContentID:  contentID,
			UserID:     rootUser,
			Comment:    "root",
			Status:     repositories.CommentStatusNormal,
			ReplyCount: 1,
		},
		commentTestComment{
			ID:        replyID,
			ContentID: contentID,
			UserID:    replyUser,
			ParentID:  rootID,
			RootID:    rootID,
			Comment:   "reply",
			Status:    repositories.CommentStatusNormal,
		},
	)

	rootItemKey := rediskey.BuildCommentItemKey(strconv.FormatInt(rootID, 10))
	listKey := rediskey.BuildCommentListKey(interaction.Scene_ARTICLE.String(), strconv.FormatInt(contentID, 10))
	store.Set(rootItemKey, "cached-root")
	store.ZAdd(listKey, float64(rootID), strconv.FormatInt(rootID, 10))

	_, err := NewDeleteCommentLogic(ctx, &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
	}).DeleteComment(&interaction.DeleteCommentReq{
		UserId:    rootUser,
		CommentId: rootID,
		ContentId: contentID,
		Scene:     interaction.Scene_ARTICLE,
	})
	if err != nil {
		t.Fatalf("DeleteComment returned error: %v", err)
	}

	root := requireDeleteCommentRow(t, db, rootID)
	if root.IsDeleted != 1 || root.Status != repositories.CommentStatusDeleted || root.UpdatedBy != rootUser {
		t.Fatalf("root after delete = %+v, want tombstone updated by owner", root)
	}
	if reply := requireDeleteCommentRow(t, db, replyID); reply.IsDeleted != 0 {
		t.Fatalf("reply after root tombstone = %+v, want still visible", reply)
	}
	if store.Exists(rootItemKey) || store.Exists(listKey) {
		t.Fatalf("expected root item and list caches to be invalidated")
	}
}

func TestDeleteCommentReply(t *testing.T) {
	ctx := context.Background()
	db := newCommentUserActionTestDB(t)
	store, redisClient := newCommentCacheRedis(t)

	const (
		contentID int64 = 9802
		rootID    int64 = 980201
		replyID   int64 = 980202
		rootUser  int64 = 983
		replyUser int64 = 984
	)
	seedDeleteCommentRows(t, db,
		commentTestComment{
			ID:         rootID,
			ContentID:  contentID,
			UserID:     rootUser,
			Comment:    "deleted root",
			Status:     repositories.CommentStatusDeleted,
			ReplyCount: 1,
			IsDeleted:  1,
			UpdatedBy:  rootUser,
		},
		commentTestComment{
			ID:        replyID,
			ContentID: contentID,
			UserID:    replyUser,
			ParentID:  rootID,
			RootID:    rootID,
			Comment:   "last reply",
			Status:    repositories.CommentStatusNormal,
		},
	)

	replyItemKey := rediskey.BuildCommentItemKey(strconv.FormatInt(replyID, 10))
	rootItemKey := rediskey.BuildCommentItemKey(strconv.FormatInt(rootID, 10))
	listKey := rediskey.BuildCommentListKey(interaction.Scene_ARTICLE.String(), strconv.FormatInt(contentID, 10))
	replyKey := rediskey.BuildCommentReplyKey(strconv.FormatInt(rootID, 10))
	store.Set(replyItemKey, "cached-reply")
	store.Set(rootItemKey, "cached-root")
	store.ZAdd(listKey, float64(rootID), strconv.FormatInt(rootID, 10))
	store.ZAdd(replyKey, float64(replyID), strconv.FormatInt(replyID, 10))

	_, err := NewDeleteCommentLogic(ctx, &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
	}).DeleteComment(&interaction.DeleteCommentReq{
		UserId:    replyUser,
		CommentId: replyID,
		ContentId: contentID,
		Scene:     interaction.Scene_ARTICLE,
	})
	if err != nil {
		t.Fatalf("DeleteComment returned error: %v", err)
	}

	if row := findDeleteCommentRow(t, db, replyID); row != nil {
		t.Fatalf("reply row still exists after delete: %+v", row)
	}
	if row := findDeleteCommentRow(t, db, rootID); row != nil {
		t.Fatalf("empty deleted root ancestor still exists after last child delete: %+v", row)
	}
	for _, key := range []string{replyItemKey, rootItemKey, listKey, replyKey} {
		if store.Exists(key) {
			t.Fatalf("cache key %s still exists after reply delete", key)
		}
	}
}

func TestDeleteCommentRejects(t *testing.T) {
	ctx := context.Background()
	db := newCommentUserActionTestDB(t)
	const (
		contentID int64 = 9803
		commentID int64 = 980301
		ownerID   int64 = 985
		otherID   int64 = 986
	)
	seedDeleteCommentRows(t, db, commentTestComment{
		ID:        commentID,
		ContentID: contentID,
		UserID:    ownerID,
		Comment:   "owned comment",
		Status:    repositories.CommentStatusNormal,
	})

	logic := NewDeleteCommentLogic(ctx, &svc.ServiceContext{MysqlDb: db})
	tests := []struct {
		name string
		req  *interaction.DeleteCommentReq
	}{
		{name: "nil request"},
		{name: "missing user id", req: &interaction.DeleteCommentReq{CommentId: commentID, ContentId: contentID, Scene: interaction.Scene_ARTICLE}},
		{name: "missing comment id", req: &interaction.DeleteCommentReq{UserId: ownerID, ContentId: contentID, Scene: interaction.Scene_ARTICLE}},
		{name: "missing content id", req: &interaction.DeleteCommentReq{UserId: ownerID, CommentId: commentID, Scene: interaction.Scene_ARTICLE}},
		{name: "unknown scene", req: &interaction.DeleteCommentReq{UserId: ownerID, CommentId: commentID, ContentId: contentID, Scene: interaction.Scene_SCENE_UNKNOWN}},
		{name: "content mismatch", req: &interaction.DeleteCommentReq{UserId: ownerID, CommentId: commentID, ContentId: contentID + 1, Scene: interaction.Scene_ARTICLE}},
		{name: "non owner", req: &interaction.DeleteCommentReq{UserId: otherID, CommentId: commentID, ContentId: contentID, Scene: interaction.Scene_ARTICLE}},
		{name: "missing comment", req: &interaction.DeleteCommentReq{UserId: ownerID, CommentId: commentID + 1, ContentId: contentID, Scene: interaction.Scene_ARTICLE}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := logic.DeleteComment(tt.req); err == nil {
				t.Fatal("expected error")
			}
		})
	}

	row := requireDeleteCommentRow(t, db, commentID)
	if row.IsDeleted != 0 || row.Status != repositories.CommentStatusNormal {
		t.Fatalf("comment mutated by rejected delete requests: %+v", row)
	}
}

func TestDeleteCommentIdempotent(t *testing.T) {
	ctx := context.Background()
	db := newCommentUserActionTestDB(t)

	const (
		contentID int64 = 9804
		commentID int64 = 980401
		ownerID   int64 = 987
	)
	seedDeleteCommentRows(t, db, commentTestComment{
		ID:        commentID,
		ContentID: contentID,
		UserID:    ownerID,
		Comment:   "deleted",
		Status:    repositories.CommentStatusDeleted,
		IsDeleted: 1,
		UpdatedBy: 12345,
	})

	_, err := NewDeleteCommentLogic(ctx, &svc.ServiceContext{
		MysqlDb: db,
	}).DeleteComment(&interaction.DeleteCommentReq{
		UserId:    ownerID,
		CommentId: commentID,
		ContentId: contentID,
		Scene:     interaction.Scene_ARTICLE,
	})
	if err != nil {
		t.Fatalf("DeleteComment returned error: %v", err)
	}

	row := requireDeleteCommentRow(t, db, commentID)
	if row.IsDeleted != 1 || row.Status != repositories.CommentStatusDeleted || row.UpdatedBy != 12345 {
		t.Fatalf("already deleted comment mutated by idempotent delete: %+v", row)
	}
}

func TestDeleteCommentErrors(t *testing.T) {
	ctx := context.Background()
	db := newCommentUserActionTestDB(t)
	const (
		contentID int64 = 9810
		commentID int64 = 981001
		replyID   int64 = 981002
		userID    int64 = 990
	)
	root := &do.CommentDO{
		ID:        commentID,
		ContentID: contentID,
		UserID:    userID,
		Comment:   "root",
		Status:    repositories.CommentStatusNormal,
	}
	reply := &do.CommentDO{
		ID:        replyID,
		ContentID: contentID,
		UserID:    userID,
		ParentID:  commentID,
		RootID:    commentID,
		Comment:   "reply",
		Status:    repositories.CommentStatusNormal,
	}

	tests := []struct {
		name string
		repo *fakeCommentRepository
		req  *interaction.DeleteCommentReq
	}{
		{
			name: "get include deleted error",
			repo: &fakeCommentRepository{includeDeletedErr: errors.New("get failed")},
			req:  deleteCommentReq(userID, commentID, contentID),
		},
		{
			name: "has children error",
			repo: &fakeCommentRepository{
				includeDeleted: map[int64]*do.CommentDO{commentID: root},
				hasChildrenErr: errors.New("has children failed"),
			},
			req: deleteCommentReq(userID, commentID, contentID),
		},
		{
			name: "mark deleted error",
			repo: &fakeCommentRepository{
				includeDeleted: map[int64]*do.CommentDO{commentID: root},
				children:       map[int64]bool{commentID: true},
				markErr:        errors.New("mark failed"),
			},
			req: deleteCommentReq(userID, commentID, contentID),
		},
		{
			name: "delete by id error",
			repo: &fakeCommentRepository{
				includeDeleted: map[int64]*do.CommentDO{commentID: root},
				children:       map[int64]bool{commentID: false},
				deleteErr:      errors.New("delete failed"),
			},
			req: deleteCommentReq(userID, commentID, contentID),
		},
		{
			name: "decrement reply count error",
			repo: &fakeCommentRepository{
				includeDeleted: map[int64]*do.CommentDO{replyID: reply},
				children:       map[int64]bool{replyID: false},
				decErr:         errors.New("decrement failed"),
			},
			req: deleteCommentReq(userID, replyID, contentID),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logic := NewDeleteCommentLogic(ctx, &svc.ServiceContext{MysqlDb: db})
			logic.commentRepo = tt.repo

			if _, err := logic.DeleteComment(tt.req); err == nil {
				t.Fatal("expected DeleteComment error")
			}
			if len(tt.repo.deletedIDs) > 1 {
				t.Fatalf("unexpected repeated delete operations: %v", tt.repo.deletedIDs)
			}
		})
	}
}

func TestCleanupDeletedAncestors(t *testing.T) {
	ctx := context.Background()
	logic := NewDeleteCommentLogic(ctx, &svc.ServiceContext{})
	deletedRoot := &do.CommentDO{ID: 1, Status: repositories.CommentStatusDeleted, IsDeleted: 1}
	deletedChild := &do.CommentDO{ID: 2, ParentID: 1, Status: repositories.CommentStatusDeleted, IsDeleted: 1}
	visibleChild := &do.CommentDO{ID: 3, ParentID: 1, Status: repositories.CommentStatusNormal}

	tests := []struct {
		name        string
		repo        *fakeCommentRepository
		parentID    int64
		wantDeleted []int64
		wantError   bool
	}{
		{
			name: "deletes empty deleted chain",
			repo: &fakeCommentRepository{
				includeDeleted: map[int64]*do.CommentDO{1: deletedRoot, 2: deletedChild},
				children:       map[int64]bool{1: false, 2: false},
			},
			parentID:    2,
			wantDeleted: []int64{2, 1},
		},
		{
			name:     "stops at zero parent",
			repo:     &fakeCommentRepository{},
			parentID: 0,
		},
		{
			name: "stops at visible ancestor",
			repo: &fakeCommentRepository{
				includeDeleted: map[int64]*do.CommentDO{3: visibleChild},
			},
			parentID: 3,
		},
		{
			name: "stops when deleted ancestor still has children",
			repo: &fakeCommentRepository{
				includeDeleted: map[int64]*do.CommentDO{2: deletedChild},
				children:       map[int64]bool{2: true},
			},
			parentID: 2,
		},
		{
			name: "returns get error",
			repo: &fakeCommentRepository{
				includeDeletedErr: errors.New("get include deleted failed"),
			},
			parentID:  2,
			wantError: true,
		},
		{
			name: "returns has children error",
			repo: &fakeCommentRepository{
				includeDeleted: map[int64]*do.CommentDO{2: deletedChild},
				hasChildrenErr: errors.New("has children failed"),
			},
			parentID:  2,
			wantError: true,
		},
		{
			name: "returns delete error",
			repo: &fakeCommentRepository{
				includeDeleted: map[int64]*do.CommentDO{2: deletedChild},
				deleteErr:      errors.New("delete failed"),
			},
			parentID:  2,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := logic.cleanupDeletedAncestors(tt.repo, tt.parentID)
			if tt.wantError {
				if err == nil {
					t.Fatal("expected cleanup error")
				}
				return
			}
			if err != nil {
				t.Fatalf("cleanupDeletedAncestors returned error: %v", err)
			}
			if !reflect.DeepEqual(tt.repo.deletedIDs, tt.wantDeleted) {
				t.Fatalf("deleted IDs = %v, want %v", tt.repo.deletedIDs, tt.wantDeleted)
			}
		})
	}
}

func TestDeleteCommentBizError(t *testing.T) {
	ctx := context.Background()
	db := newCommentUserActionTestDB(t)
	const (
		contentID int64 = 9821
		commentID int64 = 982101
		userID    int64 = 982201
	)
	root := &do.CommentDO{
		ID:        commentID,
		ContentID: contentID,
		UserID:    userID,
		Comment:   "root",
		Status:    repositories.CommentStatusNormal,
	}
	bizErr := errorx.NewMsg("mark deleted business error")
	repo := &fakeCommentRepository{
		includeDeleted: map[int64]*do.CommentDO{commentID: root},
		children:       map[int64]bool{commentID: true},
		markErr:        bizErr,
	}
	logic := NewDeleteCommentLogic(ctx, &svc.ServiceContext{MysqlDb: db})
	logic.commentRepo = repo

	resp, err := logic.DeleteComment(deleteCommentReq(userID, commentID, contentID))
	if !errors.Is(err, bizErr) {
		t.Fatalf("DeleteComment error = %v, want %v", err, bizErr)
	}
	if resp != nil {
		t.Fatalf("DeleteComment response = %+v, want nil", resp)
	}
}

func TestCleanupDeletedAncestorsDeleteError(t *testing.T) {
	ctx := context.Background()
	logic := NewDeleteCommentLogic(ctx, &svc.ServiceContext{})
	deleteErr := errors.New("ancestor delete failed")
	repo := &fakeCommentRepository{
		includeDeleted: map[int64]*do.CommentDO{
			10: {ID: 10, Status: repositories.CommentStatusDeleted, IsDeleted: 1},
		},
		children:  map[int64]bool{10: false},
		deleteErr: deleteErr,
	}

	err := logic.cleanupDeletedAncestors(repo, 10)
	if !errors.Is(err, deleteErr) {
		t.Fatalf("cleanupDeletedAncestors error = %v, want %v", err, deleteErr)
	}
}

func TestDeleteCacheInvalidation(t *testing.T) {
	ctx := context.Background()
	NewDeleteCommentLogic(ctx, &svc.ServiceContext{}).invalidateCommentCachesAfterDelete(nil, interaction.Scene_ARTICLE)
	NewDeleteCommentLogic(ctx, &svc.ServiceContext{}).invalidateCommentCachesAfterDelete(&do.CommentDO{ID: 1}, interaction.Scene_ARTICLE)

	store, redisClient := newCommentCacheRedis(t)
	const (
		contentID int64 = 9820
		rootID    int64 = 982001
		parentID  int64 = 982002
		replyID   int64 = 982003
	)
	keys := []string{
		rediskey.BuildCommentItemKey(strconv.FormatInt(replyID, 10)),
		rediskey.BuildCommentListKey(interaction.Scene_ARTICLE.String(), strconv.FormatInt(contentID, 10)),
		rediskey.BuildCommentReplyKey(strconv.FormatInt(rootID, 10)),
		rediskey.BuildCommentItemKey(strconv.FormatInt(rootID, 10)),
		rediskey.BuildCommentItemKey(strconv.FormatInt(parentID, 10)),
	}
	for _, key := range keys {
		store.Set(key, "cached")
	}

	NewDeleteCommentLogic(ctx, &svc.ServiceContext{
		Redis: redisClient,
	}).invalidateCommentCachesAfterDelete(&do.CommentDO{
		ID:        replyID,
		ContentID: contentID,
		RootID:    rootID,
		ParentID:  parentID,
	}, interaction.Scene_ARTICLE)

	for _, key := range keys {
		if store.Exists(key) {
			t.Fatalf("cache key %s still exists after invalidation", key)
		}
	}
}

func deleteCommentReq(userID, commentID, contentID int64) *interaction.DeleteCommentReq {
	return &interaction.DeleteCommentReq{
		UserId:    userID,
		CommentId: commentID,
		ContentId: contentID,
		Scene:     interaction.Scene_ARTICLE,
	}
}

func seedDeleteCommentRows(t *testing.T, db *gorm.DB, comments ...commentTestComment) {
	t.Helper()

	now := time.Unix(1770000700, 0)
	for i := range comments {
		if comments[i].Status == 0 {
			comments[i].Status = repositories.CommentStatusNormal
		}
		if comments[i].CreatedAt.IsZero() {
			comments[i].CreatedAt = now
		}
		if comments[i].UpdatedAt.IsZero() {
			comments[i].UpdatedAt = now
		}
	}
	if err := db.Create(&comments).Error; err != nil {
		t.Fatalf("seed comments: %v", err)
	}
}

func requireDeleteCommentRow(t *testing.T, db *gorm.DB, commentID int64) *commentTestComment {
	t.Helper()

	row := findDeleteCommentRow(t, db, commentID)
	if row == nil {
		t.Fatalf("comment row %d not found", commentID)
	}
	return row
}

func findDeleteCommentRow(t *testing.T, db *gorm.DB, commentID int64) *commentTestComment {
	t.Helper()

	var row commentTestComment
	err := db.Where("id = ?", commentID).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		t.Fatalf("find comment row %d: %v", commentID, err)
	}
	return &row
}
