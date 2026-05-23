package indexer

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"zfeed/app/rpc/search/internal/common/indexdoc"
	"zfeed/app/rpc/search/search-indexer/internal/indexconfig"
)

func TestHTTPIndexerIndexesContentDocument(t *testing.T) {
	var gotPath string
	var gotDoc indexdoc.ContentDocument
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Method != http.MethodPut {
			t.Fatalf("method = %s, want PUT", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotDoc); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	idx := NewHTTPIndexer(indexconfig.IndexEngineConf{
		Endpoint:     server.URL,
		ContentIndex: "contents",
		TimeoutMs:    1000,
	})
	err := idx.IndexContent(context.Background(), indexdoc.ContentDocument{
		ContentID:   1001,
		ContentType: 10,
		Title:       "Growth",
		HotScore:    12.5,
	})
	if err != nil {
		t.Fatalf("IndexContent returned error: %v", err)
	}
	if gotPath != "/contents/_doc/1001" {
		t.Fatalf("path = %q, want /contents/_doc/1001", gotPath)
	}
	if gotDoc.ContentID != 1001 || gotDoc.Title != "Growth" || gotDoc.HotScore != 12.5 {
		t.Fatalf("unexpected doc: %+v", gotDoc)
	}
}

func TestHTTPIndexerDeletesMissingDocumentAsSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	idx := NewHTTPIndexer(indexconfig.IndexEngineConf{
		Endpoint:  server.URL,
		UserIndex: "users",
		TimeoutMs: 1000,
	})
	if err := idx.DeleteUser(context.Background(), 2001); err != nil {
		t.Fatalf("DeleteUser returned error: %v", err)
	}
}

func TestHTTPIndexerBulkIndexesContentDocuments(t *testing.T) {
	var gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_bulk" {
			t.Fatalf("path = %q, want /_bulk", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		gotBody = string(data)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors":false,"items":[{"index":{"_index":"contents","_id":"1001","status":201}}]}`))
	}))
	defer server.Close()

	idx := NewHTTPIndexer(indexconfig.IndexEngineConf{
		Endpoint:     server.URL,
		ContentIndex: "contents",
		TimeoutMs:    1000,
	})
	err := idx.BulkIndexContent(context.Background(), []indexdoc.ContentDocument{
		{ContentID: 1001, Title: "Growth"},
	})
	if err != nil {
		t.Fatalf("BulkIndexContent returned error: %v", err)
	}
	for _, want := range []string{`"index":{"_id":"1001","_index":"contents"}`, `"content_id":1001`, `"title":"Growth"`} {
		if !strings.Contains(gotBody, want) {
			t.Fatalf("bulk body %q missing %q", gotBody, want)
		}
	}
}

func TestHTTPIndexerBulkReturnsItemError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors":true,"items":[{"index":{"_index":"contents","_id":"1001","status":500,"error":{"reason":"boom"}}}]}`))
	}))
	defer server.Close()

	idx := NewHTTPIndexer(indexconfig.IndexEngineConf{
		Endpoint:     server.URL,
		ContentIndex: "contents",
		TimeoutMs:    1000,
	})
	err := idx.BulkIndexContent(context.Background(), []indexdoc.ContentDocument{
		{ContentID: 1001, Title: "Growth"},
	})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected bulk item error, got %v", err)
	}
}
