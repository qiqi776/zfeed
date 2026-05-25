package consumer

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultFailureLogPath = "/var/log/zfeed/search-indexer-failures.jsonl"
	failureEventVersion   = 1
	defaultReplayLimit    = 100
)

type FailureRecorder interface {
	Record(ctx context.Context, event FailedEvent) error
}

type JSONLFailureRecorder struct {
	path string
}

type FailedEvent struct {
	Version    int            `json:"version"`
	RecordedAt int64          `json:"recorded_at"`
	Table      string         `json:"table"`
	EventType  string         `json:"event_type"`
	EventTS    int64          `json:"event_ts"`
	RowIndex   int            `json:"row_index"`
	ContentID  int64          `json:"content_id,omitempty"`
	UserID     int64          `json:"user_id,omitempty"`
	Row        map[string]any `json:"row"`
	Error      string         `json:"error"`
	ElapsedMS  int64          `json:"elapsed_ms"`
}

type ReplayResult struct {
	Scanned  int
	Replayed int
	Failed   int
	Elapsed  time.Duration
}

func NewJSONLFailureRecorder(path string) *JSONLFailureRecorder {
	path = strings.TrimSpace(path)
	if path == "" {
		path = DefaultFailureLogPath
	}
	return &JSONLFailureRecorder{path: path}
}

func (r *JSONLFailureRecorder) Record(_ context.Context, event FailedEvent) error {
	if r == nil || strings.TrimSpace(r.path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return err
	}

	var line bytes.Buffer
	if err := json.NewEncoder(&line).Encode(event); err != nil {
		return err
	}

	f, err := os.OpenFile(r.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(line.Bytes())
	return err
}

func ReplayFailedEvents(ctx context.Context, path string, limit int, consumer *CanalSearchConsumer) (ReplayResult, error) {
	start := time.Now()
	if limit <= 0 {
		limit = defaultReplayLimit
	}
	if consumer == nil {
		return ReplayResult{Elapsed: time.Since(start)}, fmt.Errorf("search index replay consumer is nil")
	}

	f, err := os.Open(path)
	if err != nil {
		return ReplayResult{Elapsed: time.Since(start)}, err
	}
	defer f.Close()

	result, err := replayFailedEventsFromReader(ctx, f, limit, consumer)
	result.Elapsed = time.Since(start)
	return result, err
}

func replayFailedEventsFromReader(ctx context.Context, r io.Reader, limit int, consumer *CanalSearchConsumer) (ReplayResult, error) {
	var result ReplayResult
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if limit > 0 && result.Scanned >= limit {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		result.Scanned++

		var event FailedEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			result.Failed++
			return result, fmt.Errorf("decode failed event line %d: %w", result.Scanned, err)
		}
		if event.Version != failureEventVersion || event.Table == "" || event.EventType == "" || event.Row == nil {
			result.Failed++
			return result, fmt.Errorf("invalid failed event line %d", result.Scanned)
		}

		msg := canalMessage{
			ID:    fmt.Sprintf("replay-%d", result.Scanned),
			Table: event.Table,
			Type:  event.EventType,
			Ts:    event.EventTS,
			Data:  []map[string]any{event.Row},
		}
		if err := consumer.dispatchRow(ctx, msg, event.RowIndex, event.Row); err != nil {
			result.Failed++
			return result, fmt.Errorf("replay failed event line %d: %w", result.Scanned, err)
		}
		result.Replayed++
	}
	if err := scanner.Err(); err != nil {
		return result, err
	}
	return result, nil
}

func buildFailedEvent(msg canalMessage, idx int, row map[string]any, start time.Time, cause error) FailedEvent {
	return FailedEvent{
		Version:    failureEventVersion,
		RecordedAt: time.Now().UnixMilli(),
		Table:      msg.Table,
		EventType:  msg.Type,
		EventTS:    msg.Ts,
		RowIndex:   idx,
		ContentID:  failedContentID(msg.Table, row),
		UserID:     failedUserID(msg.Table, row),
		Row:        row,
		Error:      errorString(cause),
		ElapsedMS:  time.Since(start).Milliseconds(),
	}
}

func failedContentID(table string, row map[string]any) int64 {
	switch table {
	case tableContent:
		return int64Value(row["id"])
	case tableArticle, tableVideo:
		return int64Value(row["content_id"])
	default:
		return 0
	}
}

func failedUserID(table string, row map[string]any) int64 {
	if table == tableUser {
		return int64Value(row["id"])
	}
	return 0
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
