package repositories

import (
	"context"
	"testing"

	"gorm.io/gorm"

	"zfeed/app/rpc/interaction/internal/do"
	"zfeed/app/rpc/interaction/internal/testutil/mysqltest"
)

const (
	commentRepositoryTestMinID int64 = 910000
	commentRepositoryTestMaxID int64 = 910999
)

func TestCommentRepositoryCreateListAndBatchGet(t *testing.T) {
	db, cleanup := openCommentRepositoryTestDB(t)
	defer cleanup()

	repo := NewCommentRepository(context.Background(), db)
	contentID := commentRepositoryTestMinID + 101
	rootUserID := commentRepositoryTestMinID + 1
	replyUserID := commentRepositoryTestMinID + 2
	nestedUserID := commentRepositoryTestMinID + 3

	rootID, err := repo.Create(&do.CommentDO{
		ContentID: contentID,
		UserID:    rootUserID,
		Comment:   "root comment",
		Status:    CommentStatusNormal,
		CreatedBy: rootUserID,
		UpdatedBy: rootUserID,
	})
	if err != nil {
		t.Fatalf("create root comment: %v", err)
	}

	replyID, err := repo.Create(&do.CommentDO{
		ContentID:     contentID,
		UserID:        replyUserID,
		ReplyToUserID: rootUserID,
		ParentID:      rootID,
		RootID:        rootID,
		Comment:       "reply comment",
		Status:        CommentStatusNormal,
		CreatedBy:     replyUserID,
		UpdatedBy:     replyUserID,
	})
	if err != nil {
		t.Fatalf("create reply comment: %v", err)
	}

	nestedID, err := repo.Create(&do.CommentDO{
		ContentID:     contentID,
		UserID:        nestedUserID,
		ReplyToUserID: replyUserID,
		ParentID:      replyID,
		RootID:        rootID,
		Comment:       "nested reply",
		Status:        CommentStatusNormal,
		CreatedBy:     nestedUserID,
		UpdatedBy:     nestedUserID,
	})
	if err != nil {
		t.Fatalf("create nested reply: %v", err)
	}

	if err := repo.IncReplyCount(rootID); err != nil {
		t.Fatalf("inc reply count first time: %v", err)
	}
	if err := repo.IncReplyCount(rootID); err != nil {
		t.Fatalf("inc reply count second time: %v", err)
	}

	rootComment, err := repo.GetByID(rootID)
	if err != nil {
		t.Fatalf("get root comment: %v", err)
	}
	if rootComment == nil {
		t.Fatal("root comment is nil")
	}
	if rootComment.ReplyCount != 2 {
		t.Fatalf("root reply_count = %d, want 2", rootComment.ReplyCount)
	}
	if rootComment.Version != 1 {
		t.Fatalf("root version = %d, want 1", rootComment.Version)
	}

	roots, err := repo.ListRootComments(contentID, 0, 10)
	if err != nil {
		t.Fatalf("list root comments: %v", err)
	}
	if len(roots) != 1 {
		t.Fatalf("root comment count = %d, want 1", len(roots))
	}
	if roots[0].ID != rootID {
		t.Fatalf("root comment id = %d, want %d", roots[0].ID, rootID)
	}

	replies, err := repo.ListReplies(rootID, 0, 10)
	if err != nil {
		t.Fatalf("list replies: %v", err)
	}
	if len(replies) != 2 {
		t.Fatalf("reply comment count = %d, want 2", len(replies))
	}
	if replies[0].ID != nestedID || replies[1].ID != replyID {
		t.Fatalf("reply order = [%d %d], want [%d %d]", replies[0].ID, replies[1].ID, nestedID, replyID)
	}
	if replies[0].RootID != rootID || replies[1].RootID != rootID {
		t.Fatalf("reply root ids = [%d %d], want both %d", replies[0].RootID, replies[1].RootID, rootID)
	}

	commentMap, err := repo.BatchGetByIDs([]int64{nestedID, rootID, commentRepositoryTestMaxID + 1})
	if err != nil {
		t.Fatalf("batch get comments: %v", err)
	}
	if len(commentMap) != 2 {
		t.Fatalf("batch get map size = %d, want 2", len(commentMap))
	}
	if commentMap[rootID] == nil || commentMap[nestedID] == nil {
		t.Fatal("batch get comments missed created rows")
	}
}

func TestCommentRepositoryMarkDeletedAndGetIncludeDeleted(t *testing.T) {
	db, cleanup := openCommentRepositoryTestDB(t)
	defer cleanup()

	repo := NewCommentRepository(context.Background(), db)
	userID := commentRepositoryTestMinID + 21
	commentID, err := repo.Create(&do.CommentDO{
		ContentID:     commentRepositoryTestMinID + 201,
		ContentUserID: commentRepositoryTestMinID + 301,
		UserID:        userID,
		Comment:       "tombstone target",
		Status:        CommentStatusNormal,
		CreatedBy:     userID,
		UpdatedBy:     userID,
	})
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}

	if err := repo.MarkDeleted(commentID, userID); err != nil {
		t.Fatalf("mark deleted: %v", err)
	}

	commentDO, err := repo.GetByID(commentID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if commentDO != nil {
		t.Fatal("expected GetByID to hide deleted comment")
	}

	deletedComment, err := repo.GetByIDIncludeDeleted(commentID)
	if err != nil {
		t.Fatalf("get include deleted: %v", err)
	}
	if deletedComment == nil {
		t.Fatal("expected GetByIDIncludeDeleted to return tombstone comment")
	}
	if deletedComment.IsDeleted != 1 || deletedComment.Status != CommentStatusDeleted {
		t.Fatalf("deleted state = (%d,%d), want (1,%d)", deletedComment.IsDeleted, deletedComment.Status, CommentStatusDeleted)
	}
}

func TestCommentRepositoryDeleteByID(t *testing.T) {
	db, cleanup := openCommentRepositoryTestDB(t)
	defer cleanup()

	repo := NewCommentRepository(context.Background(), db)
	userID := commentRepositoryTestMinID + 31
	commentID, err := repo.Create(&do.CommentDO{
		ContentID:     commentRepositoryTestMinID + 202,
		ContentUserID: commentRepositoryTestMinID + 302,
		UserID:        userID,
		Comment:       "delete me",
		Status:        CommentStatusNormal,
		CreatedBy:     userID,
		UpdatedBy:     userID,
	})
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}

	if err := repo.DeleteByID(commentID); err != nil {
		t.Fatalf("delete by id: %v", err)
	}

	deletedComment, err := repo.GetByIDIncludeDeleted(commentID)
	if err != nil {
		t.Fatalf("get include deleted after delete: %v", err)
	}
	if deletedComment != nil {
		t.Fatal("expected deleted comment to be physically removed")
	}
}

func TestCommentRepositoryHasChildren(t *testing.T) {
	db, cleanup := openCommentRepositoryTestDB(t)
	defer cleanup()

	repo := NewCommentRepository(context.Background(), db)
	contentID := commentRepositoryTestMinID + 203
	rootUserID := commentRepositoryTestMinID + 41
	replyUserID := commentRepositoryTestMinID + 42

	rootID, err := repo.Create(&do.CommentDO{
		ContentID:     contentID,
		ContentUserID: commentRepositoryTestMinID + 303,
		UserID:        rootUserID,
		Comment:       "root",
		Status:        CommentStatusNormal,
		CreatedBy:     rootUserID,
		UpdatedBy:     rootUserID,
	})
	if err != nil {
		t.Fatalf("create root comment: %v", err)
	}

	replyID, err := repo.Create(&do.CommentDO{
		ContentID:     contentID,
		ContentUserID: commentRepositoryTestMinID + 303,
		UserID:        replyUserID,
		ReplyToUserID: rootUserID,
		ParentID:      rootID,
		RootID:        rootID,
		Comment:       "reply",
		Status:        CommentStatusNormal,
		CreatedBy:     replyUserID,
		UpdatedBy:     replyUserID,
	})
	if err != nil {
		t.Fatalf("create reply comment: %v", err)
	}

	hasChildren, err := repo.HasChildren(rootID)
	if err != nil {
		t.Fatalf("has children for root: %v", err)
	}
	if !hasChildren {
		t.Fatal("expected root comment to have children")
	}

	hasChildren, err = repo.HasChildren(replyID)
	if err != nil {
		t.Fatalf("has children for reply: %v", err)
	}
	if hasChildren {
		t.Fatal("expected reply comment to have no children")
	}
}

func openCommentRepositoryTestDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()

	db, err := mysqltest.Open()
	if err != nil {
		t.Skipf("skip MySQL-backed comment repository test: %v", err)
	}

	if err := mysqltest.EnsureCommentTables(db); err != nil {
		_ = mysqltest.Close(db)
		t.Fatalf("ensure comment tables: %v", err)
	}

	if err := mysqltest.CleanupCommentRowsByRange(db, commentRepositoryTestMinID, commentRepositoryTestMaxID); err != nil {
		_ = mysqltest.Close(db)
		t.Fatalf("cleanup comment rows before run: %v", err)
	}
	if err := mysqltest.CleanupContentRowsByRange(db, commentRepositoryTestMinID, commentRepositoryTestMaxID); err != nil {
		_ = mysqltest.Close(db)
		t.Fatalf("cleanup content rows before run: %v", err)
	}

	return db, func() {
		if err := mysqltest.CleanupCommentRowsByRange(db, commentRepositoryTestMinID, commentRepositoryTestMaxID); err != nil {
			t.Fatalf("cleanup comment rows after run: %v", err)
		}
		if err := mysqltest.CleanupContentRowsByRange(db, commentRepositoryTestMinID, commentRepositoryTestMaxID); err != nil {
			t.Fatalf("cleanup content rows after run: %v", err)
		}
		if err := mysqltest.Close(db); err != nil {
			t.Fatalf("close db: %v", err)
		}
	}
}
