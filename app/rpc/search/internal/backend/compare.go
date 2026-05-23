package backend

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/core/logx"
)

const defaultCompareTimeout = 800 * time.Millisecond

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
		primaryCount := len(result.Rows)
		go func() {
			shadowCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), b.shadowTimeout)
			defer cancel()
			if shadow, shadowErr := b.shadow.SearchUsers(shadowCtx, query, cursor, limit); shadowErr != nil {
				logc.Errorf(shadowCtx, "search compare users shadow failed, err=%v", shadowErr)
			} else {
				logc.Infow(shadowCtx, "search compare users",
					logx.Field("primary_backend", b.primary.Name()),
					logx.Field("shadow_backend", b.shadow.Name()),
					logx.Field("primary_count", primaryCount),
					logx.Field("shadow_count", len(shadow.Rows)),
				)
			}
		}()
	}
	return result, err
}

func (b *CompareBackend) SearchContents(ctx context.Context, query string, mode string, cursor int64, limit int) (SearchContentsResult, error) {
	result, err := b.primary.SearchContents(ctx, query, mode, cursor, limit)
	if b.shadow != nil {
		primaryCount := len(result.Rows)
		go func() {
			shadowCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), b.shadowTimeout)
			defer cancel()
			if shadow, shadowErr := b.shadow.SearchContents(shadowCtx, query, mode, cursor, limit); shadowErr != nil {
				logc.Errorf(shadowCtx, "search compare contents shadow failed, err=%v", shadowErr)
			} else {
				logc.Infow(shadowCtx, "search compare contents",
					logx.Field("primary_backend", b.primary.Name()),
					logx.Field("shadow_backend", b.shadow.Name()),
					logx.Field("primary_count", primaryCount),
					logx.Field("shadow_count", len(shadow.Rows)),
				)
			}
		}()
	}
	return result, err
}
