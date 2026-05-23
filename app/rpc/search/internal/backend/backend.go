package backend

import (
	"context"
	"hash/fnv"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"

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
	SearchContents(ctx context.Context, query string, mode string, cursor int64, limit int) (SearchContentsResult, error)
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
	engine     SearchBackend
	compare    SearchBackend
	configured string
	effective  string
	trafficPct int
}

func NewFactory(db *gorm.DB, configured string, trafficPct ...int) Factory {
	return NewFactoryWithEngineConfig(db, configured, EngineConfig{}, firstTrafficPercent(trafficPct))
}

func NewFactoryWithEngineConfig(db *gorm.DB, configured string, engineConfig EngineConfig, trafficPct int) Factory {
	configured = NormalizeName(configured)
	effective := configured
	if configured == NameEngine {
		effective = NameEngine
	}

	mysql := NewMySQLBackend(db)
	engine := NewEngineBackend(engineConfig, mysql)
	compare := SearchBackend(nil)
	if engineConfig.CompareEnabled {
		compare = NewCompareBackend(mysql, engine)
	}
	return &factory{
		mysql:      mysql,
		engine:     engine,
		compare:    compare,
		configured: configured,
		effective:  effective,
		trafficPct: normalizeTrafficPercent(trafficPct),
	}
}

func (f *factory) Backend(ctx context.Context) SearchBackend {
	if f.configured == NameEngine && f.shouldUseEngine(ctx) {
		return f.engine
	}
	if f.configured == NameEngine && f.compare != nil {
		return f.compare
	}
	return f.mysql
}

func (f *factory) ConfiguredBackend() string {
	return f.configured
}

func (f *factory) EffectiveBackend() string {
	if f.configured == NameEngine && f.trafficPct > 0 {
		return f.effective
	}
	return NameMySQL
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

func (f *factory) shouldUseEngine(ctx context.Context) bool {
	if f == nil || f.configured != NameEngine || f.trafficPct <= 0 {
		return false
	}
	if f.trafficPct >= 100 {
		return true
	}

	h := fnv.New32a()
	if ctx != nil {
		_, _ = h.Write([]byte(ctxHashSeed(ctx)))
	} else {
		_, _ = h.Write([]byte("search-engine"))
	}
	return int(h.Sum32()%100) < f.trafficPct
}

func ctxHashSeed(ctx context.Context) string {
	if spanCtx := trace.SpanContextFromContext(ctx); spanCtx.IsValid() {
		return spanCtx.TraceID().String()
	}
	return time.Now().Format(time.RFC3339Nano)
}

func firstTrafficPercent(values []int) int {
	if len(values) == 0 {
		return 0
	}
	return values[0]
}

func normalizeTrafficPercent(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}
