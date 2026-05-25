package consumer

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"zfeed/app/rpc/search/internal/common/indexdoc"
)

func TestJSONLFailureRecorderRecordsFailureEvent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "failures.jsonl")
	recorder := NewJSONLFailureRecorder(path)
	event := FailedEvent{
		Version:   failureEventVersion,
		Table:     tableArticle,
		EventType: "UPDATE",
		EventTS:   1_700_000_000_000,
		RowIndex:  0,
		ContentID: 4001,
		Row:       map[string]any{"content_id": float64(4001)},
		Error:     "boom",
		ElapsedMS: 12,
	}

	if err := recorder.Record(context.Background(), event); err != nil {
		t.Fatalf("Record returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failure log: %v", err)
	}
	var got FailedEvent
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("decode failure event: %v", err)
	}
	if got.Version != failureEventVersion || got.Table != tableArticle || got.ContentID != 4001 || got.Error != "boom" {
		t.Fatalf("unexpected failure event: %+v", got)
	}
}

func TestReplayFailedEventsReindexesContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "failures.jsonl")
	recorder := NewJSONLFailureRecorder(path)
	if err := recorder.Record(context.Background(), FailedEvent{
		Version:   failureEventVersion,
		Table:     tableArticle,
		EventType: "UPDATE",
		EventTS:   1_700_000_000_000,
		RowIndex:  0,
		ContentID: 4001,
		Row:       map[string]any{"content_id": float64(4001)},
		Error:     "boom",
	}); err != nil {
		t.Fatalf("seed failure log: %v", err)
	}

	repo := &fakeRepo{
		contentDocs: map[int64]*indexdoc.ContentDocument{
			4001: {ContentID: 4001, Title: "Growth"},
		},
	}
	idx := &fakeIndexer{}
	consumer := newCanalSearchConsumerForTest(context.Background(), repo, idx)

	result, err := ReplayFailedEvents(context.Background(), path, 10, consumer)
	if err != nil {
		t.Fatalf("ReplayFailedEvents returned error: %v", err)
	}
	if result.Scanned != 1 || result.Replayed != 1 || result.Failed != 0 {
		t.Fatalf("result = %+v, want scanned=1 replayed=1 failed=0", result)
	}
	if len(idx.indexedContents) != 1 || idx.indexedContents[0] != 4001 {
		t.Fatalf("indexed contents = %v, want [4001]", idx.indexedContents)
	}
}
