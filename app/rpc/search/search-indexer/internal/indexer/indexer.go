package indexer

import (
	"context"
	"fmt"
	"strings"

	"zfeed/app/rpc/search/internal/common/indexdoc"
	"zfeed/app/rpc/search/search-indexer/internal/indexconfig"
)

type Indexer interface {
	IndexContent(ctx context.Context, doc indexdoc.ContentDocument) error
	DeleteContent(ctx context.Context, contentID int64) error
	IndexUser(ctx context.Context, doc indexdoc.UserDocument) error
	DeleteUser(ctx context.Context, userID int64) error
	BulkIndexContent(ctx context.Context, docs []indexdoc.ContentDocument) error
	BulkDeleteContent(ctx context.Context, contentIDs []int64) error
	BulkIndexUser(ctx context.Context, docs []indexdoc.UserDocument) error
	BulkDeleteUser(ctx context.Context, userIDs []int64) error
	Count(ctx context.Context, index string) (int64, error)
	SwitchAlias(ctx context.Context, alias string, targetIndex string) error
	GetContentDocuments(ctx context.Context, index string, ids []int64) (map[int64]indexdoc.ContentDocument, error)
	GetUserDocuments(ctx context.Context, index string, ids []int64) (map[int64]indexdoc.UserDocument, error)
	SearchContentIDs(ctx context.Context, index string, query string, mode string, limit int) ([]int64, error)
	SearchUserIDs(ctx context.Context, index string, query string, limit int) ([]int64, error)
}

func New(conf indexconfig.IndexEngineConf) (Indexer, error) {
	switch normalizeEngineType(conf.Type) {
	case "", "noop":
		return Noop{}, nil
	case "elastic", "elasticsearch", "opensearch":
		if conf.Endpoint == "" {
			return nil, fmt.Errorf("search index engine endpoint is required")
		}
		return NewHTTPIndexer(conf), nil
	default:
		return nil, fmt.Errorf("unsupported search index engine type: %s", conf.Type)
	}
}

func normalizeEngineType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

type Noop struct{}

func (Noop) IndexContent(context.Context, indexdoc.ContentDocument) error       { return nil }
func (Noop) DeleteContent(context.Context, int64) error                         { return nil }
func (Noop) IndexUser(context.Context, indexdoc.UserDocument) error             { return nil }
func (Noop) DeleteUser(context.Context, int64) error                            { return nil }
func (Noop) BulkIndexContent(context.Context, []indexdoc.ContentDocument) error { return nil }
func (Noop) BulkDeleteContent(context.Context, []int64) error                   { return nil }
func (Noop) BulkIndexUser(context.Context, []indexdoc.UserDocument) error       { return nil }
func (Noop) BulkDeleteUser(context.Context, []int64) error                      { return nil }
func (Noop) Count(context.Context, string) (int64, error)                       { return 0, nil }
func (Noop) SwitchAlias(context.Context, string, string) error                  { return nil }
func (Noop) GetContentDocuments(context.Context, string, []int64) (map[int64]indexdoc.ContentDocument, error) {
	return map[int64]indexdoc.ContentDocument{}, nil
}
func (Noop) GetUserDocuments(context.Context, string, []int64) (map[int64]indexdoc.UserDocument, error) {
	return map[int64]indexdoc.UserDocument{}, nil
}
func (Noop) SearchContentIDs(context.Context, string, string, string, int) ([]int64, error) {
	return []int64{}, nil
}
func (Noop) SearchUserIDs(context.Context, string, string, int) ([]int64, error) {
	return []int64{}, nil
}
