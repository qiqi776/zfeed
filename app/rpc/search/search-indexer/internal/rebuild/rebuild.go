package rebuild

import (
	"context"
	"fmt"
	"io"
	"time"

	"zfeed/app/rpc/search/internal/common/indexdoc"
)

const (
	EntityAll      = "all"
	EntityContent  = "content"
	EntityUser     = "user"
	defaultBatchSz = 100
)

type Repository interface {
	ListContentDocumentsAfter(ctx context.Context, cursorID int64, endID int64, limit int) ([]indexdoc.ContentDocument, error)
	ListUserDocumentsAfter(ctx context.Context, cursorID int64, endID int64, limit int) ([]indexdoc.UserDocument, error)
}

type BulkIndexer interface {
	BulkIndexContent(ctx context.Context, docs []indexdoc.ContentDocument) error
	BulkIndexUser(ctx context.Context, docs []indexdoc.UserDocument) error
}

type Options struct {
	Entity    string
	BatchSize int
	StartID   int64
	EndID     int64
	DryRun    bool
	Out       io.Writer
}

type Result struct {
	Entity        string
	Indexed       int
	LastContentID int64
	LastUserID    int64
	Elapsed       time.Duration
}

func Run(ctx context.Context, repo Repository, idx BulkIndexer, opts Options) (Result, error) {
	start := time.Now()
	entity := normalizeEntity(opts.Entity)
	batchSize := opts.BatchSize
	if batchSize <= 0 {
		batchSize = defaultBatchSz
	}

	var result Result
	result.Entity = entity
	if entity == EntityAll || entity == EntityContent {
		contentResult, err := rebuildContents(ctx, repo, idx, opts.StartID, opts.EndID, batchSize, opts.DryRun, opts.Out)
		if err != nil {
			return result, err
		}
		result.Indexed += contentResult.Indexed
		result.LastContentID = contentResult.LastContentID
	}
	if entity == EntityAll || entity == EntityUser {
		userResult, err := rebuildUsers(ctx, repo, idx, opts.StartID, opts.EndID, batchSize, opts.DryRun, opts.Out)
		if err != nil {
			return result, err
		}
		result.Indexed += userResult.Indexed
		result.LastUserID = userResult.LastUserID
	}
	result.Elapsed = time.Since(start)
	return result, nil
}

func rebuildContents(ctx context.Context, repo Repository, idx BulkIndexer, startID int64, endID int64, batchSize int, dryRun bool, out io.Writer) (Result, error) {
	cursor := startID
	result := Result{Entity: EntityContent, LastContentID: cursor}
	for {
		docs, err := repo.ListContentDocumentsAfter(ctx, cursor, endID, batchSize)
		if err != nil {
			return result, err
		}
		if len(docs) == 0 {
			return result, nil
		}
		cursor = docs[len(docs)-1].ContentID
		result.LastContentID = cursor
		result.Indexed += len(docs)
		if !dryRun {
			if err := idx.BulkIndexContent(ctx, docs); err != nil {
				return result, err
			}
		}
		writeProgress(out, "content", result.Indexed, cursor, dryRun)
		if len(docs) < batchSize {
			return result, nil
		}
	}
}

func rebuildUsers(ctx context.Context, repo Repository, idx BulkIndexer, startID int64, endID int64, batchSize int, dryRun bool, out io.Writer) (Result, error) {
	cursor := startID
	result := Result{Entity: EntityUser, LastUserID: cursor}
	for {
		docs, err := repo.ListUserDocumentsAfter(ctx, cursor, endID, batchSize)
		if err != nil {
			return result, err
		}
		if len(docs) == 0 {
			return result, nil
		}
		cursor = docs[len(docs)-1].UserID
		result.LastUserID = cursor
		result.Indexed += len(docs)
		if !dryRun {
			if err := idx.BulkIndexUser(ctx, docs); err != nil {
				return result, err
			}
		}
		writeProgress(out, "user", result.Indexed, cursor, dryRun)
		if len(docs) < batchSize {
			return result, nil
		}
	}
}

func normalizeEntity(value string) string {
	switch value {
	case EntityContent, EntityUser, EntityAll:
		return value
	default:
		return EntityAll
	}
}

func writeProgress(out io.Writer, entity string, indexed int, cursor int64, dryRun bool) {
	if out == nil {
		return
	}
	_, _ = fmt.Fprintf(out, "search-indexer rebuild %s: indexed=%d cursor=%d dry_run=%t\n", entity, indexed, cursor, dryRun)
}
