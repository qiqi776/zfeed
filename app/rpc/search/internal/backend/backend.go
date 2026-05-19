package backend

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"zfeed/app/rpc/search/internal/repositories"
)

const (
	NameMySQL  = "mysql"
	NameEngine = "engine"
)

type SearchBackend interface {
	Name() string
	SearchUsers(ctx context.Context, query string, cursor int64, limit int) (SearchUsersResult, error)
	SearchContents(ctx context.Context, query string, cursor int64, limit int) (SearchContentsResult, error)
}

type SearchUsersResult struct {
	Rows []repositories.SearchUserRow
	Meta repositories.SearchMeta
}

type SearchContentsResult struct {
	Rows []repositories.SearchContentRow
	Meta repositories.SearchMeta
}

type Factory interface {
	Backend(ctx context.Context) SearchBackend
	ConfiguredBackend() string
	EffectiveBackend() string
}

type factory struct {
	mysql      *MySQLBackend
	configured string
	effective  string
}

func NewFactory(db *gorm.DB, configured string) Factory {
	configured = NormalizeName(configured)
	effective := configured
	if configured == NameEngine {
		effective = NameMySQL
	}

	return &factory{
		mysql:      NewMySQLBackend(db),
		configured: configured,
		effective:  effective,
	}
}

func (f *factory) Backend(context.Context) SearchBackend {
	return f.mysql
}

func (f *factory) ConfiguredBackend() string {
	return f.configured
}

func (f *factory) EffectiveBackend() string {
	return f.effective
}

func NormalizeName(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", NameMySQL:
		return NameMySQL
	case NameEngine:
		return NameEngine
	default:
		return NameMySQL
	}
}
