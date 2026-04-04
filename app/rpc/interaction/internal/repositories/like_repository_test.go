package repositories

import (
	"context"
	"testing"

	"gorm.io/gorm"

	"zfeed/app/rpc/interaction/internal/do"
	"zfeed/app/rpc/interaction/internal/testutil/mysqltest"
)

const (
	likeRepositoryTestMinID int64 = 900000
	likeRepositoryTestMaxID int64 = 900999
)

type likeRow struct {
	UserID        int64 `gorm:"column:user_id"`
	ContentID     int64 `gorm:"column:content_id"`
	ContentUserID int64 `gorm:"column:content_user_id"`
	Status        int32 `gorm:"column:status"`
	LastEventTs   int64 `gorm:"column:last_event_ts"`
}

func TestLikeUpsertNewerWins(t *testing.T) {
	db, cleanup := openLikeRepositoryTestDB(t)
	defer cleanup()

	repo := NewLikeRepository(context.Background(), db)
	userID := likeRepositoryTestMinID + 1
	contentID := likeRepositoryTestMinID + 101
	contentUserID := likeRepositoryTestMinID + 201

	err := repo.Upsert(&do.LikeDO{
		UserID:        userID,
		ContentID:     contentID,
		ContentUserID: contentUserID,
		Status:        LikeStatusLike,
		LastEventTs:   100,
		CreatedBy:     userID,
		UpdatedBy:     userID,
	})
	if err != nil {
		t.Fatalf("first upsert returned error: %v", err)
	}

	err = repo.Upsert(&do.LikeDO{
		UserID:        userID,
		ContentID:     contentID,
		ContentUserID: contentUserID,
		Status:        LikeStatusCancel,
		LastEventTs:   200,
		CreatedBy:     userID,
		UpdatedBy:     userID,
	})
	if err != nil {
		t.Fatalf("second upsert returned error: %v", err)
	}

	row := mustGetLikeRow(t, db, userID, contentID)
	if row.Status != LikeStatusCancel {
		t.Fatalf("status = %d, want %d", row.Status, LikeStatusCancel)
	}
	if row.LastEventTs != 200 {
		t.Fatalf("last_event_ts = %d, want %d", row.LastEventTs, 200)
	}
}

func TestLikeUpsertOlderIgnored(t *testing.T) {
	db, cleanup := openLikeRepositoryTestDB(t)
	defer cleanup()

	repo := NewLikeRepository(context.Background(), db)
	userID := likeRepositoryTestMinID + 2
	contentID := likeRepositoryTestMinID + 102
	contentUserID := likeRepositoryTestMinID + 202

	err := repo.Upsert(&do.LikeDO{
		UserID:        userID,
		ContentID:     contentID,
		ContentUserID: contentUserID,
		Status:        LikeStatusCancel,
		LastEventTs:   200,
		CreatedBy:     userID,
		UpdatedBy:     userID,
	})
	if err != nil {
		t.Fatalf("first upsert returned error: %v", err)
	}

	err = repo.Upsert(&do.LikeDO{
		UserID:        userID,
		ContentID:     contentID,
		ContentUserID: contentUserID + 1,
		Status:        LikeStatusLike,
		LastEventTs:   100,
		CreatedBy:     userID,
		UpdatedBy:     userID,
	})
	if err != nil {
		t.Fatalf("second upsert returned error: %v", err)
	}

	row := mustGetLikeRow(t, db, userID, contentID)
	if row.Status != LikeStatusCancel {
		t.Fatalf("status = %d, want %d", row.Status, LikeStatusCancel)
	}
	if row.LastEventTs != 200 {
		t.Fatalf("last_event_ts = %d, want %d", row.LastEventTs, 200)
	}
	if row.ContentUserID != contentUserID {
		t.Fatalf("content_user_id = %d, want %d", row.ContentUserID, contentUserID)
	}
}

func TestLikeUpsertNewerAuthorWins(t *testing.T) {
	db, cleanup := openLikeRepositoryTestDB(t)
	defer cleanup()

	repo := NewLikeRepository(context.Background(), db)
	userID := likeRepositoryTestMinID + 3
	contentID := likeRepositoryTestMinID + 103
	initialContentUserID := likeRepositoryTestMinID + 203
	updatedContentUserID := likeRepositoryTestMinID + 204

	err := repo.Upsert(&do.LikeDO{
		UserID:        userID,
		ContentID:     contentID,
		ContentUserID: initialContentUserID,
		Status:        LikeStatusLike,
		LastEventTs:   100,
		CreatedBy:     userID,
		UpdatedBy:     userID,
	})
	if err != nil {
		t.Fatalf("first upsert returned error: %v", err)
	}

	err = repo.Upsert(&do.LikeDO{
		UserID:        userID,
		ContentID:     contentID,
		ContentUserID: updatedContentUserID,
		Status:        LikeStatusCancel,
		LastEventTs:   200,
		CreatedBy:     userID,
		UpdatedBy:     userID,
	})
	if err != nil {
		t.Fatalf("second upsert returned error: %v", err)
	}

	err = repo.Upsert(&do.LikeDO{
		UserID:        userID,
		ContentID:     contentID,
		ContentUserID: initialContentUserID,
		Status:        LikeStatusLike,
		LastEventTs:   150,
		CreatedBy:     userID,
		UpdatedBy:     userID,
	})
	if err != nil {
		t.Fatalf("third upsert returned error: %v", err)
	}

	row := mustGetLikeRow(t, db, userID, contentID)
	if row.Status != LikeStatusCancel {
		t.Fatalf("status = %d, want %d", row.Status, LikeStatusCancel)
	}
	if row.LastEventTs != 200 {
		t.Fatalf("last_event_ts = %d, want %d", row.LastEventTs, 200)
	}
	if row.ContentUserID != updatedContentUserID {
		t.Fatalf("content_user_id = %d, want %d", row.ContentUserID, updatedContentUserID)
	}
}

func openLikeRepositoryTestDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()

	db, err := mysqltest.Open()
	if err != nil {
		t.Skipf("skip MySQL-backed like repository test: %v", err)
	}

	if err := mysqltest.EnsureLikeTables(db); err != nil {
		_ = mysqltest.Close(db)
		t.Fatalf("ensure test tables: %v", err)
	}

	if err := mysqltest.CleanupLikeRowsByRange(db, likeRepositoryTestMinID, likeRepositoryTestMaxID); err != nil {
		_ = mysqltest.Close(db)
		t.Fatalf("cleanup test rows before run: %v", err)
	}

	return db, func() {
		if err := mysqltest.CleanupLikeRowsByRange(db, likeRepositoryTestMinID, likeRepositoryTestMaxID); err != nil {
			t.Fatalf("cleanup test rows after run: %v", err)
		}
		if err := mysqltest.Close(db); err != nil {
			t.Fatalf("close db: %v", err)
		}
	}
}

func mustGetLikeRow(t *testing.T, db *gorm.DB, userID, contentID int64) likeRow {
	t.Helper()

	var row likeRow
	if err := db.Table("zfeed_like").
		Select("user_id", "content_id", "content_user_id", "status", "last_event_ts").
		Where("user_id = ? AND content_id = ?", userID, contentID).
		Take(&row).Error; err != nil {
		t.Fatalf("query like row: %v", err)
	}

	return row
}
