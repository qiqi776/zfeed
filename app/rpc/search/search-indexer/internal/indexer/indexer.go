package indexer

import (
	"context"
	"fmt"
	"strings"

	"zfeed/app/rpc/search/search-indexer/internal/indexconfig"
	"zfeed/app/rpc/search/internal/common/indexdoc"
)

type Indexer interface {
	IndexContent(ctx context.Context, doc indexdoc.ContentDocument) error
	DeleteContent(ctx context.Context, contentID int64) error
	IndexUser(ctx context.Context, doc indexdoc.UserDocument) error
	DeleteUser(ctx context.Context, userID int64) error
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

func (Noop) IndexContent(context.Context, indexdoc.ContentDocument) error { return nil }
func (Noop) DeleteContent(context.Context, int64) error                   { return nil }
func (Noop) IndexUser(context.Context, indexdoc.UserDocument) error       { return nil }
func (Noop) DeleteUser(context.Context, int64) error                      { return nil }
