package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logc"

	"zfeed/app/rpc/search/search-rpc/internal/repositories"
)

const defaultEngineTimeoutMs = 3000
const (
	engineContentStatusPublished  = 30
	engineContentVisibilityPublic = 10
	engineUserStatusActive        = 10
)

type EngineConfig struct {
	Endpoint       string
	ContentIndex   string
	UserIndex      string
	Username       string
	Password       string
	TimeoutMs      int
	CompareEnabled bool
}

type EngineBackend struct {
	client       *http.Client
	endpoint     string
	contentIndex string
	userIndex    string
	username     string
	password     string
	fallback     SearchBackend
}

func NewEngineBackend(conf EngineConfig, fallback SearchBackend) *EngineBackend {
	timeoutMs := conf.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = defaultEngineTimeoutMs
	}
	return &EngineBackend{
		client:       &http.Client{Timeout: time.Duration(timeoutMs) * time.Millisecond},
		endpoint:     strings.TrimRight(conf.Endpoint, "/"),
		contentIndex: defaultString(conf.ContentIndex, "zfeed_content"),
		userIndex:    defaultString(conf.UserIndex, "zfeed_user"),
		username:     conf.Username,
		password:     conf.Password,
		fallback:     fallback,
	}
}

func (b *EngineBackend) Name() string {
	return NameEngine
}

func (b *EngineBackend) SearchUsers(ctx context.Context, query string, cursor int64, limit int) (SearchUsersResult, error) {
	if b == nil || b.endpoint == "" {
		return b.fallbackUsers(ctx, query, cursor, limit)
	}

	filters := []map[string]any{{"term": map[string]any{"status": engineUserStatusActive}}}
	if cursor > 0 {
		filters = append(filters, map[string]any{"range": map[string]any{"user_id": map[string]any{"lt": cursor}}})
	}
	body := engineSearchRequest{
		Size: limit,
		Query: map[string]any{
			"bool": map[string]any{
				"must": map[string]any{
					"multi_match": map[string]any{
						"query":  query,
						"fields": []string{"nickname^3", "bio", "mobile_search_field"},
					},
				},
				"filter": filters,
			},
		},
		Sort: []map[string]any{
			{"_score": "desc"},
			{"user_id": "desc"},
		},
	}

	var resp engineUserSearchResponse
	if err := b.search(ctx, b.userIndex, body, &resp); err != nil {
		logc.Errorf(ctx, "search engine users failed, err=%v", err)
		return b.fallbackUsers(ctx, query, cursor, limit)
	}
	rows := make([]repositories.SearchUserRow, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		rows = append(rows, repositories.SearchUserRow{
			UserID:   hit.Source.UserID,
			Nickname: hit.Source.Nickname,
			Avatar:   "",
			Bio:      hit.Source.Bio,
		})
	}
	return SearchUsersResult{Rows: rows, Meta: repositories.SearchMeta{QueryPath: "engine"}}, nil
}

func (b *EngineBackend) SearchContents(ctx context.Context, query string, mode string, cursor int64, limit int) (SearchContentsResult, error) {
	if b == nil || b.endpoint == "" {
		return b.fallbackContents(ctx, query, mode, cursor, limit)
	}

	filters := []map[string]any{
		{"term": map[string]any{"status": engineContentStatusPublished}},
		{"term": map[string]any{"visibility": engineContentVisibilityPublic}},
	}
	if cursor > 0 {
		filters = append(filters, map[string]any{"range": map[string]any{"content_id": map[string]any{"lt": cursor}}})
	}
	body := engineSearchRequest{
		Size: limit,
		Query: map[string]any{
			"bool": map[string]any{
				"must": map[string]any{
					"multi_match": map[string]any{
						"query":  query,
						"fields": []string{"title^3", "description", "author_name"},
					},
				},
				"filter": filters,
			},
		},
		Sort: contentEngineSort(mode),
	}

	var resp engineContentSearchResponse
	if err := b.search(ctx, b.contentIndex, body, &resp); err != nil {
		logc.Errorf(ctx, "search engine contents failed, err=%v", err)
		return b.fallbackContents(ctx, query, mode, cursor, limit)
	}
	rows := make([]repositories.SearchContentRow, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		rows = append(rows, repositories.SearchContentRow{
			ContentID:    hit.Source.ContentID,
			ContentType:  hit.Source.ContentType,
			AuthorID:     hit.Source.AuthorID,
			AuthorName:   hit.Source.AuthorName,
			AuthorAvatar: hit.Source.AuthorAvatar,
			Title:        hit.Source.Title,
			CoverURL:     "",
			PublishedAt:  unixTimePtr(hit.Source.PublishedAt),
			TextScore:    hit.Score,
			HotScore:     hit.Source.HotScore,
			RankScore:    hit.Score + hit.Source.HotScore,
		})
	}
	return SearchContentsResult{Rows: rows, Meta: repositories.SearchMeta{QueryPath: "engine"}}, nil
}

func (b *EngineBackend) search(ctx context.Context, index string, body engineSearchRequest, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.endpoint+"/"+url.PathEscape(index)+"/_search", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if b.username != "" {
		req.SetBasicAuth(b.username, b.password)
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("engine search failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (b *EngineBackend) fallbackUsers(ctx context.Context, query string, cursor int64, limit int) (SearchUsersResult, error) {
	if b != nil && b.fallback != nil {
		return b.fallback.SearchUsers(ctx, query, cursor, limit)
	}
	return SearchUsersResult{}, nil
}

func (b *EngineBackend) fallbackContents(ctx context.Context, query string, mode string, cursor int64, limit int) (SearchContentsResult, error) {
	if b != nil && b.fallback != nil {
		return b.fallback.SearchContents(ctx, query, mode, cursor, limit)
	}
	return SearchContentsResult{}, nil
}

type engineSearchRequest struct {
	Query any              `json:"query"`
	Sort  []map[string]any `json:"sort,omitempty"`
	Size  int              `json:"size"`
}

type engineUserSearchResponse struct {
	Hits struct {
		Hits []struct {
			Score  float64 `json:"_score"`
			Source struct {
				UserID   int64  `json:"user_id"`
				Nickname string `json:"nickname"`
				Bio      string `json:"bio"`
			} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

type engineContentSearchResponse struct {
	Hits struct {
		Hits []struct {
			Score  float64 `json:"_score"`
			Source struct {
				ContentID    int64   `json:"content_id"`
				ContentType  int32   `json:"content_type"`
				Title        string  `json:"title"`
				AuthorID     int64   `json:"author_id"`
				AuthorName   string  `json:"author_name"`
				AuthorAvatar string  `json:"author_avatar"`
				PublishedAt  int64   `json:"published_at"`
				HotScore     float64 `json:"hot_score"`
			} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

func contentEngineSort(mode string) []map[string]any {
	switch mode {
	case "latest":
		return []map[string]any{{"published_at": "desc"}, {"content_id": "desc"}}
	case "hybrid":
		return []map[string]any{{"_score": "desc"}, {"hot_score": "desc"}, {"content_id": "desc"}}
	case "relevance":
		fallthrough
	default:
		return []map[string]any{{"_score": "desc"}, {"content_id": "desc"}}
	}
}

func unixTimePtr(value int64) *time.Time {
	if value <= 0 {
		return nil
	}
	t := time.Unix(value, 0)
	return &t
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
