package indexer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"zfeed/app/rpc/search/search-indexer/internal/indexconfig"
	"zfeed/app/rpc/search/internal/common/indexdoc"
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
