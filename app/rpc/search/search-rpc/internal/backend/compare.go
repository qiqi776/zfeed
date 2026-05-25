package backend

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/core/logx"

	"zfeed/app/rpc/search/search-rpc/internal/repositories"
)

const defaultCompareTimeout = 800 * time.Millisecond
const (
	compareEntityUsers    = "users"
	compareEntityContents = "contents"
	compareDefaultMode    = "default"
	compareHashBytes      = 8
)

type CompareBackend struct {
	primary       SearchBackend
	shadow        SearchBackend
	shadowTimeout time.Duration
}

func NewCompareBackend(primary SearchBackend, shadow SearchBackend) *CompareBackend {
	return &CompareBackend{primary: primary, shadow: shadow, shadowTimeout: defaultCompareTimeout}
}

func (b *CompareBackend) Name() string {
	if b == nil || b.primary == nil {
		return NameMySQL
	}
	return b.primary.Name()
}

func (b *CompareBackend) SearchUsers(ctx context.Context, query string, cursor int64, limit int) (SearchUsersResult, error) {
	result, err := b.primary.SearchUsers(ctx, query, cursor, limit)
	if b.shadow != nil {
		b.compareUsersAsync(ctx, query, cursor, limit, result.Rows)
	}
	return result, err
}

func (b *CompareBackend) SearchContents(ctx context.Context, query string, mode string, cursor int64, limit int) (SearchContentsResult, error) {
	result, err := b.primary.SearchContents(ctx, query, mode, cursor, limit)
	if b.shadow != nil {
		b.compareContentsAsync(ctx, query, mode, cursor, limit, result.Rows)
	}
	return result, err
}

func (b *CompareBackend) compareUsersAsync(ctx context.Context, query string, cursor int64, limit int, primaryRows []repositories.SearchUserRow) {
	go func() {
		shadowCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), b.shadowTimeout)
		defer cancel()

		shadow, shadowErr := b.shadow.SearchUsers(shadowCtx, query, cursor, limit)
		if shadowErr != nil {
			logc.Errorf(shadowCtx, "search compare users shadow failed, err=%v", shadowErr)
			return
		}

		primaryIDs := topUserIDs(primaryRows, limit)
		shadowIDs := topUserIDs(shadow.Rows, limit)
		overlap := overlapRatio(primaryIDs, shadowIDs)
		observeSearchCompare(compareEntityUsers, b.primary.Name(), b.shadow.Name(), compareDefaultMode, "success", overlap)
		logc.Infow(shadowCtx, "search compare users",
			logx.Field("entity", compareEntityUsers),
			logx.Field("mode", compareDefaultMode),
			logx.Field("query_hash", compareQueryHash(query)),
			logx.Field("primary_backend", b.primary.Name()),
			logx.Field("shadow_backend", b.shadow.Name()),
			logx.Field("primary_count", len(primaryRows)),
			logx.Field("shadow_count", len(shadow.Rows)),
			logx.Field("primary_top_ids", primaryIDs),
			logx.Field("shadow_top_ids", shadowIDs),
			logx.Field("overlap_ratio", overlap),
		)
	}()
}

func (b *CompareBackend) compareContentsAsync(ctx context.Context, query string, mode string, cursor int64, limit int, primaryRows []repositories.SearchContentRow) {
	go func() {
		shadowCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), b.shadowTimeout)
		defer cancel()

		shadow, shadowErr := b.shadow.SearchContents(shadowCtx, query, mode, cursor, limit)
		if shadowErr != nil {
			logc.Errorf(shadowCtx, "search compare contents shadow failed, err=%v", shadowErr)
			return
		}

		primaryIDs := topContentIDs(primaryRows, limit)
		shadowIDs := topContentIDs(shadow.Rows, limit)
		overlap := overlapRatio(primaryIDs, shadowIDs)
		observeSearchCompare(compareEntityContents, b.primary.Name(), b.shadow.Name(), mode, "success", overlap)
		logc.Infow(shadowCtx, "search compare contents",
			logx.Field("entity", compareEntityContents),
			logx.Field("mode", mode),
			logx.Field("query_hash", compareQueryHash(query)),
			logx.Field("primary_backend", b.primary.Name()),
			logx.Field("shadow_backend", b.shadow.Name()),
			logx.Field("primary_count", len(primaryRows)),
			logx.Field("shadow_count", len(shadow.Rows)),
			logx.Field("primary_top_ids", primaryIDs),
			logx.Field("shadow_top_ids", shadowIDs),
			logx.Field("overlap_ratio", overlap),
		)
	}()
}

func topUserIDs(rows []repositories.SearchUserRow, limit int) []int64 {
	if limit <= 0 || limit > len(rows) {
		limit = len(rows)
	}
	ids := make([]int64, 0, limit)
	for _, row := range rows[:limit] {
		if row.UserID > 0 {
			ids = append(ids, row.UserID)
		}
	}
	return ids
}

func topContentIDs(rows []repositories.SearchContentRow, limit int) []int64 {
	if limit <= 0 || limit > len(rows) {
		limit = len(rows)
	}
	ids := make([]int64, 0, limit)
	for _, row := range rows[:limit] {
		if row.ContentID > 0 {
			ids = append(ids, row.ContentID)
		}
	}
	return ids
}

func overlapRatio(left []int64, right []int64) float64 {
	if len(left) == 0 && len(right) == 0 {
		return 1
	}
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	set := make(map[int64]struct{}, len(left))
	for _, id := range left {
		set[id] = struct{}{}
	}
	matches := 0
	for _, id := range right {
		if _, ok := set[id]; ok {
			matches++
		}
	}
	denominator := len(left)
	if len(right) > denominator {
		denominator = len(right)
	}
	return float64(matches) / float64(denominator)
}

func compareQueryHash(query string) string {
	canonical := strings.ToLower(strings.Join(strings.Fields(query), " "))
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:compareHashBytes])
}
