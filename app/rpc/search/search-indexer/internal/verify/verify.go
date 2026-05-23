package verify

import (
	"context"
	"fmt"
	"math"
	"time"

	"zfeed/app/rpc/search/internal/common/indexdoc"
)

const (
	EntityAll     = "all"
	EntityContent = "content"
	EntityUser    = "user"
)

type Repository interface {
	CountContentDocuments(ctx context.Context) (int64, error)
	CountUserDocuments(ctx context.Context) (int64, error)
	SampleContentDocuments(ctx context.Context, limit int) ([]indexdoc.ContentDocument, error)
	SampleUserDocuments(ctx context.Context, limit int) ([]indexdoc.UserDocument, error)
	SearchContentIDs(ctx context.Context, query string, mode string, limit int) ([]int64, error)
	SearchUserIDs(ctx context.Context, query string, limit int) ([]int64, error)
}

type Indexer interface {
	Count(ctx context.Context, index string) (int64, error)
	GetContentDocuments(ctx context.Context, index string, ids []int64) (map[int64]indexdoc.ContentDocument, error)
	GetUserDocuments(ctx context.Context, index string, ids []int64) (map[int64]indexdoc.UserDocument, error)
	SearchContentIDs(ctx context.Context, index string, query string, mode string, limit int) ([]int64, error)
	SearchUserIDs(ctx context.Context, index string, query string, limit int) ([]int64, error)
	SwitchAlias(ctx context.Context, alias string, targetIndex string) error
}

type Options struct {
	Entity       string
	ContentIndex string
	UserIndex    string
	SampleSize   int
	TopQueries   []string
	TopN         int
	MinOverlap   float64
}

type Result struct {
	Entity             string
	ContentCountOK     bool
	UserCountOK        bool
	ContentSampleOK    bool
	UserSampleOK       bool
	TopNOK             bool
	MySQLContentCount  int64
	IndexContentCount  int64
	MySQLUserCount     int64
	IndexUserCount     int64
	ContentSampleCount int
	UserSampleCount    int
	MinOverlapObserved float64
	Elapsed            time.Duration
}

func (r Result) OK() bool {
	return r.ContentCountOK && r.UserCountOK && r.ContentSampleOK && r.UserSampleOK && r.TopNOK
}

func Run(ctx context.Context, repo Repository, idx Indexer, opts Options) (Result, error) {
	start := time.Now()
	entity := normalizeEntity(opts.Entity)
	sampleSize := opts.SampleSize
	if sampleSize <= 0 {
		sampleSize = 20
	}
	topN := opts.TopN
	if topN <= 0 {
		topN = 20
	}
	minOverlap := opts.MinOverlap
	if minOverlap <= 0 {
		minOverlap = 0.7
	}

	result := Result{
		Entity:             entity,
		ContentCountOK:     true,
		UserCountOK:        true,
		ContentSampleOK:    true,
		UserSampleOK:       true,
		TopNOK:             true,
		MinOverlapObserved: 1,
	}
	if entity == EntityAll || entity == EntityContent {
		if err := verifyContent(ctx, repo, idx, opts.ContentIndex, sampleSize, &result); err != nil {
			return result, err
		}
	}
	if entity == EntityAll || entity == EntityUser {
		if err := verifyUser(ctx, repo, idx, opts.UserIndex, sampleSize, &result); err != nil {
			return result, err
		}
	}
	if len(opts.TopQueries) > 0 {
		if err := verifyTopN(ctx, repo, idx, opts, topN, minOverlap, &result); err != nil {
			return result, err
		}
	}
	result.Elapsed = time.Since(start)
	return result, nil
}

func SwitchAliasAfterVerify(ctx context.Context, repo Repository, idx Indexer, opts Options, contentAlias string, userAlias string) (Result, error) {
	result, err := Run(ctx, repo, idx, opts)
	if err != nil {
		return result, err
	}
	if !result.OK() {
		return result, fmt.Errorf("search index verification failed")
	}
	entity := normalizeEntity(opts.Entity)
	if (entity == EntityAll || entity == EntityContent) && contentAlias != "" {
		if err := idx.SwitchAlias(ctx, contentAlias, opts.ContentIndex); err != nil {
			return result, err
		}
	}
	if (entity == EntityAll || entity == EntityUser) && userAlias != "" {
		if err := idx.SwitchAlias(ctx, userAlias, opts.UserIndex); err != nil {
			return result, err
		}
	}
	return result, nil
}

func verifyContent(ctx context.Context, repo Repository, idx Indexer, index string, sampleSize int, result *Result) error {
	mysqlCount, err := repo.CountContentDocuments(ctx)
	if err != nil {
		return err
	}
	indexCount, err := idx.Count(ctx, index)
	if err != nil {
		return err
	}
	result.MySQLContentCount = mysqlCount
	result.IndexContentCount = indexCount
	result.ContentCountOK = mysqlCount == indexCount

	samples, err := repo.SampleContentDocuments(ctx, sampleSize)
	if err != nil {
		return err
	}
	result.ContentSampleCount = len(samples)
	ids := make([]int64, 0, len(samples))
	for _, doc := range samples {
		ids = append(ids, doc.ContentID)
	}
	docs, err := idx.GetContentDocuments(ctx, index, ids)
	if err != nil {
		return err
	}
	for _, sample := range samples {
		got, ok := docs[sample.ContentID]
		if !ok || got.Title != sample.Title || got.AuthorID != sample.AuthorID || got.Status != sample.Status || got.Visibility != sample.Visibility {
			result.ContentSampleOK = false
			return nil
		}
	}
	return nil
}

func verifyUser(ctx context.Context, repo Repository, idx Indexer, index string, sampleSize int, result *Result) error {
	mysqlCount, err := repo.CountUserDocuments(ctx)
	if err != nil {
		return err
	}
	indexCount, err := idx.Count(ctx, index)
	if err != nil {
		return err
	}
	result.MySQLUserCount = mysqlCount
	result.IndexUserCount = indexCount
	result.UserCountOK = mysqlCount == indexCount

	samples, err := repo.SampleUserDocuments(ctx, sampleSize)
	if err != nil {
		return err
	}
	result.UserSampleCount = len(samples)
	ids := make([]int64, 0, len(samples))
	for _, doc := range samples {
		ids = append(ids, doc.UserID)
	}
	docs, err := idx.GetUserDocuments(ctx, index, ids)
	if err != nil {
		return err
	}
	for _, sample := range samples {
		got, ok := docs[sample.UserID]
		if !ok || got.Nickname != sample.Nickname || got.Status != sample.Status {
			result.UserSampleOK = false
			return nil
		}
	}
	return nil
}

func verifyTopN(ctx context.Context, repo Repository, idx Indexer, opts Options, topN int, minOverlap float64, result *Result) error {
	minObserved := math.MaxFloat64
	for _, query := range opts.TopQueries {
		if opts.ContentIndex != "" {
			mysqlIDs, err := repo.SearchContentIDs(ctx, query, "relevance", topN)
			if err != nil {
				return err
			}
			indexIDs, err := idx.SearchContentIDs(ctx, opts.ContentIndex, query, "relevance", topN)
			if err != nil {
				return err
			}
			overlap := overlapRatio(mysqlIDs, indexIDs)
			minObserved = math.Min(minObserved, overlap)
		}
		if opts.UserIndex != "" {
			mysqlIDs, err := repo.SearchUserIDs(ctx, query, topN)
			if err != nil {
				return err
			}
			indexIDs, err := idx.SearchUserIDs(ctx, opts.UserIndex, query, topN)
			if err != nil {
				return err
			}
			overlap := overlapRatio(mysqlIDs, indexIDs)
			minObserved = math.Min(minObserved, overlap)
		}
	}
	if minObserved == math.MaxFloat64 {
		minObserved = 1
	}
	result.MinOverlapObserved = minObserved
	result.TopNOK = minObserved >= minOverlap
	return nil
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

func normalizeEntity(value string) string {
	switch value {
	case EntityContent, EntityUser, EntityAll:
		return value
	default:
		return EntityAll
	}
}
