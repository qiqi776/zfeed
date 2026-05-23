package consumer

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/core/logx"

	"zfeed/app/rpc/search/internal/common/indexdoc"
	"zfeed/app/rpc/search/search-indexer/internal/indexer"
	"zfeed/app/rpc/search/search-indexer/internal/indexsvc"
)

const (
	tableContent = "zfeed_content"
	tableArticle = "zfeed_article"
	tableVideo   = "zfeed_video"
	tableUser    = "zfeed_user"
)

type canalMessage struct {
	ID    any              `json:"id"`
	Table string           `json:"table"`
	Type  string           `json:"type"`
	Ts    int64            `json:"ts"`
	Data  []map[string]any `json:"data"`
	Old   []map[string]any `json:"old"`
}

type CanalSearchConsumer struct {
	repo    documentRepository
	indexer indexer.Indexer
	logx.Logger
}

type documentRepository interface {
	BuildContentDocument(ctx context.Context, contentID int64) (*indexdoc.ContentDocument, bool, error)
	BuildUserDocument(ctx context.Context, userID int64) (*indexdoc.UserDocument, bool, error)
	ListContentIDsByAuthor(ctx context.Context, authorID int64, limit int) ([]int64, error)
}

func NewCanalSearchConsumer(ctx context.Context, svcCtx *indexsvc.ServiceContext) *CanalSearchConsumer {
	return &CanalSearchConsumer{
		repo:    svcCtx.Repository,
		indexer: svcCtx.Indexer,
		Logger:  logx.WithContext(ctx),
	}
}

func newCanalSearchConsumerForTest(ctx context.Context, repo documentRepository, idx indexer.Indexer) *CanalSearchConsumer {
	return &CanalSearchConsumer{
		repo:    repo,
		indexer: idx,
		Logger:  logx.WithContext(ctx),
	}
}

func (c *CanalSearchConsumer) Consume(ctx context.Context, _, val string) error {
	var msg canalMessage
	if err := json.Unmarshal([]byte(val), &msg); err != nil {
		logc.Errorf(ctx, "parse search canal message failed, err=%v", err)
		return err
	}

	start := time.Now()
	indexed := 0
	for idx, row := range msg.Data {
		if row == nil {
			continue
		}
		if err := c.dispatchRow(ctx, msg, idx, row); err != nil {
			return err
		}
		indexed++
	}
	c.Infow("search index canal message consumed",
		logx.Field("table", msg.Table),
		logx.Field("type", msg.Type),
		logx.Field("rows", indexed),
		logx.Field("elapsed_ms", time.Since(start).Milliseconds()),
		logx.Field("event_ts", canalTimestampToTime(msg.Ts).UnixMilli()),
	)
	return nil
}

func (c *CanalSearchConsumer) dispatchRow(ctx context.Context, msg canalMessage, idx int, row map[string]any) error {
	switch msg.Table {
	case tableContent:
		return c.reindexContentByID(ctx, int64Value(row["id"]), isDelete(msg.Type))
	case tableArticle, tableVideo:
		return c.reindexContentByID(ctx, int64Value(row["content_id"]), isDelete(msg.Type))
	case tableUser:
		userID := int64Value(row["id"])
		if err := c.reindexUserByID(ctx, userID, isDelete(msg.Type)); err != nil {
			return err
		}
		return c.reindexAuthorContents(ctx, userID)
	default:
		c.Debugw("ignore unsupported search index table",
			logx.Field("table", msg.Table),
			logx.Field("event_id", buildRowEventID(msg, idx, row)),
		)
		return nil
	}
}

func (c *CanalSearchConsumer) reindexContentByID(ctx context.Context, contentID int64, forceDelete bool) error {
	if contentID <= 0 {
		return nil
	}
	if forceDelete {
		return c.indexer.DeleteContent(ctx, contentID)
	}

	doc, ok, err := c.repo.BuildContentDocument(ctx, contentID)
	if err != nil {
		return err
	}
	if !ok {
		return c.indexer.DeleteContent(ctx, contentID)
	}
	return c.indexer.IndexContent(ctx, *doc)
}

func (c *CanalSearchConsumer) reindexUserByID(ctx context.Context, userID int64, forceDelete bool) error {
	if userID <= 0 {
		return nil
	}
	if forceDelete {
		return c.indexer.DeleteUser(ctx, userID)
	}

	doc, ok, err := c.repo.BuildUserDocument(ctx, userID)
	if err != nil {
		return err
	}
	if !ok {
		return c.indexer.DeleteUser(ctx, userID)
	}
	return c.indexer.IndexUser(ctx, *doc)
}

func (c *CanalSearchConsumer) reindexAuthorContents(ctx context.Context, authorID int64) error {
	contentIDs, err := c.repo.ListContentIDsByAuthor(ctx, authorID, 200)
	if err != nil {
		return err
	}
	for _, contentID := range contentIDs {
		if err := c.reindexContentByID(ctx, contentID, false); err != nil {
			return err
		}
	}
	return nil
}

func isDelete(operation string) bool {
	return strings.EqualFold(strings.TrimSpace(operation), "DELETE")
}

func int64Value(value any) int64 {
	switch v := value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case json.Number:
		n, _ := v.Int64()
		return n
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return n
	default:
		n, _ := strconv.ParseInt(strings.TrimSpace(fmt.Sprint(v)), 10, 64)
		return n
	}
}

func buildRowEventID(msg canalMessage, idx int, row map[string]any) string {
	baseID := strings.TrimSpace(fmt.Sprint(msg.ID))
	if baseID == "" || baseID == "<nil>" {
		hash := sha1.Sum([]byte(fmt.Sprintf("%s|%s|%d|%v", msg.Table, msg.Type, idx, row)))
		baseID = hex.EncodeToString(hash[:])
	}
	rowID := strings.TrimSpace(fmt.Sprint(row["id"]))
	if rowID == "" || rowID == "<nil>" {
		rowID = fmt.Sprintf("%d", idx)
	}
	raw := fmt.Sprintf("%s|%s|%s|%s", baseID, msg.Table, msg.Type, rowID)
	hash := sha1.Sum([]byte(raw))
	return hex.EncodeToString(hash[:])
}

func canalTimestampToTime(ts int64) time.Time {
	switch {
	case ts > 1_000_000_000_000:
		return time.UnixMilli(ts)
	case ts > 0:
		return time.Unix(ts, 0)
	default:
		return time.Now()
	}
}
