package rebuild

import (
	"context"
	"testing"

	"zfeed/app/rpc/search/internal/common/indexdoc"
)

func TestRunRebuildsContentWithBulkBatches(t *testing.T) {
	repo := &fakeRepo{
		contentDocs: []indexdoc.ContentDocument{
			{ContentID: 1, Title: "one"},
			{ContentID: 2, Title: "two"},
			{ContentID: 3, Title: "three"},
		},
	}
	idx := &fakeIndexer{}

	result, err := Run(context.Background(), repo, idx, Options{
		Entity:    EntityContent,
		BatchSize: 2,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Indexed != 3 || result.LastContentID != 3 {
		t.Fatalf("result = %+v, want indexed=3 last_content_id=3", result)
	}
	if len(idx.contentBatches) != 2 || len(idx.contentBatches[0]) != 2 || len(idx.contentBatches[1]) != 1 {
		t.Fatalf("content batches = %+v", idx.contentBatches)
	}
}

func TestRunDryRunSkipsBulkWrites(t *testing.T) {
	repo := &fakeRepo{
		userDocs: []indexdoc.UserDocument{
			{UserID: 1, Nickname: "one"},
		},
	}
	idx := &fakeIndexer{}

	result, err := Run(context.Background(), repo, idx, Options{
		Entity: EntityUser,
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Indexed != 1 || len(idx.userBatches) != 0 {
		t.Fatalf("result=%+v user batches=%+v", result, idx.userBatches)
	}
}

type fakeRepo struct {
	contentDocs []indexdoc.ContentDocument
	userDocs    []indexdoc.UserDocument
}

func (r *fakeRepo) ListContentDocumentsAfter(_ context.Context, cursorID int64, endID int64, limit int) ([]indexdoc.ContentDocument, error) {
	out := make([]indexdoc.ContentDocument, 0, limit)
	for _, doc := range r.contentDocs {
		if doc.ContentID <= cursorID {
			continue
		}
		if endID > 0 && doc.ContentID > endID {
			continue
		}
		out = append(out, doc)
		if len(out) == limit {
			break
		}
	}
	return out, nil
}

func (r *fakeRepo) ListUserDocumentsAfter(_ context.Context, cursorID int64, endID int64, limit int) ([]indexdoc.UserDocument, error) {
	out := make([]indexdoc.UserDocument, 0, limit)
	for _, doc := range r.userDocs {
		if doc.UserID <= cursorID {
			continue
		}
		if endID > 0 && doc.UserID > endID {
			continue
		}
		out = append(out, doc)
		if len(out) == limit {
			break
		}
	}
	return out, nil
}

type fakeIndexer struct {
	contentBatches [][]indexdoc.ContentDocument
	userBatches    [][]indexdoc.UserDocument
}

func (i *fakeIndexer) BulkIndexContent(_ context.Context, docs []indexdoc.ContentDocument) error {
	i.contentBatches = append(i.contentBatches, docs)
	return nil
}

func (i *fakeIndexer) BulkIndexUser(_ context.Context, docs []indexdoc.UserDocument) error {
	i.userBatches = append(i.userBatches, docs)
	return nil
}
