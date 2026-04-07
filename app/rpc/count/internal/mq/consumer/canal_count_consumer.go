package consumer

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/core/logx"

	"zfeed/app/rpc/count/internal/changeevent"
	countEvent "zfeed/app/rpc/count/internal/event"
	"zfeed/app/rpc/count/internal/svc"
)

type canalMessage struct {
	ID    any              `json:"id"`
	Table string           `json:"table"`
	Type  string           `json:"type"`
	Ts    int64            `json:"ts"`
	Data  []map[string]any `json:"data"`
	Old   []map[string]any `json:"old"`
}

type CanalCountConsumer struct {
	svcCtx     *svc.ServiceContext
	logx.Logger
	dispatcher *countEvent.Dispatcher
}

func NewCanalCountConsumer(ctx context.Context, svcCtx *svc.ServiceContext) *CanalCountConsumer {
	return &CanalCountConsumer{
		svcCtx:     svcCtx,
		Logger:     logx.WithContext(ctx),
		dispatcher: countEvent.NewDispatcher(ctx, svcCtx, "count.canal_consumer"),
	}
}

func (c *CanalCountConsumer) Consume(ctx context.Context, _, val string) error {
	var msg canalMessage
	if err := json.Unmarshal([]byte(val), &msg); err != nil {
		logc.Errorf(ctx, "parse canal message failed, err=%v", err)
		return err
	}
	for idx, row := range msg.Data {
		if row == nil {
			continue
		}
		evt := changeevent.ChangeEvent{
			EventID:   buildRowEventID(msg, idx, row),
			Source:    "canal",
			Table:     msg.Table,
			Operation: msg.Type,
			Timestamp: canalTimestampToTime(msg.Ts),
			Current:   row,
			Previous:  getOldRow(msg.Old, idx),
		}
		if _, err := c.dispatcher.Dispatch(ctx, evt); err != nil {
			return err
		}
	}
	return nil
}

func getOldRow(old []map[string]any, idx int) map[string]any {
	if idx < 0 || idx >= len(old) {
		return nil
	}
	return old[idx]
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
