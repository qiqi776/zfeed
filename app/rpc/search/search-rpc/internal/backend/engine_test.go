package backend

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEngineBackendSearchContentsParsesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/contents/_search" {
			t.Fatalf("path = %q, want /contents/_search", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"hits": {
				"hits": [
					{
						"_score": 3.5,
						"_source": {
							"content_id": 4001,
							"content_type": 10,
							"title": "Growth",
							"author_id": 3001,
							"author_name": "writer",
							"author_avatar": "avatar",
							"published_at": 1700000000,
							"hot_score": 9.5
						}
					}
				]
			}
		}`))
	}))
	defer server.Close()

	backend := NewEngineBackend(EngineConfig{
		Endpoint:     server.URL,
		ContentIndex: "contents",
		TimeoutMs:    1000,
	}, nil)
	result, err := backend.SearchContents(context.Background(), "Growth", "hybrid", 0, 10)
	if err != nil {
		t.Fatalf("SearchContents returned error: %v", err)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(result.Rows))
	}
	row := result.Rows[0]
	if row.ContentID != 4001 || row.Title != "Growth" || row.TextScore != 3.5 || row.HotScore != 9.5 || row.RankScore != 13 {
		t.Fatalf("unexpected row: %+v", row)
	}
	if result.Meta.QueryPath != "engine" {
		t.Fatalf("query path = %q, want engine", result.Meta.QueryPath)
	}
}

func TestEngineBackendSearchContentsAddsVisibilityFilters(t *testing.T) {
	var body engineSearchRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hits":{"hits":[]}}`))
	}))
	defer server.Close()

	backend := NewEngineBackend(EngineConfig{
		Endpoint:     server.URL,
		ContentIndex: "contents",
		TimeoutMs:    1000,
	}, nil)
	if _, err := backend.SearchContents(context.Background(), "Growth", "latest", 123, 10); err != nil {
		t.Fatalf("SearchContents returned error: %v", err)
	}

	raw, err := json.Marshal(body.Query)
	if err != nil {
		t.Fatalf("marshal query: %v", err)
	}
	query := string(raw)
	for _, want := range []string{`"status":30`, `"visibility":10`, `"content_id":{"lt":123}`} {
		if !strings.Contains(query, want) {
			t.Fatalf("query %s missing %s", query, want)
		}
	}
}

func TestEngineBackendFallsBackWhenEndpointMissing(t *testing.T) {
	fallback := NewMySQLBackend(nil)
	backend := NewEngineBackend(EngineConfig{}, fallback)
	result, err := backend.SearchUsers(context.Background(), "Alice", 0, 10)
	if err != nil {
		t.Fatalf("SearchUsers returned error: %v", err)
	}
	if len(result.Rows) != 0 {
		t.Fatalf("len(rows) = %d, want 0", len(result.Rows))
	}
}
