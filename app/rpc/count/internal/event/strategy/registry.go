package strategy

import (
	"context"
	"strings"

	"zfeed/app/rpc/count/count"
	"zfeed/app/rpc/count/internal/changeevent"
)

type Update struct {
	BizType    count.BizType
	TargetType count.TargetType
	TargetID   int64
	OwnerID    int64
	Delta      int64
}

type TableStrategy interface {
	TableName() string
	ExtractUpdates(ctx context.Context, evt changeevent.ChangeEvent) []Update
}

type Registry struct {
	strategies map[string]TableStrategy
}

func NewDefaultRegistry() *Registry {
	entries := []TableStrategy{
		newLikeStrategy(),
		newFavoriteStrategy(),
		newCommentStrategy(),
		newFollowStrategy(),
	}
	result := &Registry{strategies: make(map[string]TableStrategy, len(entries))}
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		result.strategies[normalizeTableName(entry.TableName())] = entry
	}
	return result
}

func (r *Registry) Get(table string) (TableStrategy, bool) {
	entry, ok := r.strategies[normalizeTableName(table)]
	return entry, ok
}

func normalizeTableName(table string) string {
	return strings.ToLower(strings.TrimSpace(table))
}
