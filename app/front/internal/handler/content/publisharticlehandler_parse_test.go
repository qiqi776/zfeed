package content

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zeromicro/go-zero/rest/httpx"

	"zfeed/app/front/internal/types"
)

func TestPublishArticleParseAllowsMissingCover(t *testing.T) {
	req := httptest.NewRequest("POST", "/v1/content/article/publish",
		strings.NewReader(`{"title":"1","content":"1","visibility":10}`))
	req.Header.Set("Content-Type", "application/json")

	var payload types.PublishArticleReq
	if err := httpx.Parse(req, &payload); err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if payload.Title == nil || *payload.Title != "1" {
		t.Fatalf("unexpected title: %#v", payload.Title)
	}
	if payload.Content == nil || *payload.Content != "1" {
		t.Fatalf("unexpected content: %#v", payload.Content)
	}
	if payload.Visibility == nil || *payload.Visibility != 10 {
		t.Fatalf("unexpected visibility: %#v", payload.Visibility)
	}
	if payload.Cover != nil {
		t.Fatalf("cover should be nil when omitted, got %#v", payload.Cover)
	}
}
