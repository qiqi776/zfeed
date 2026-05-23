package indexer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"zfeed/app/rpc/search/search-indexer/internal/indexconfig"
	"zfeed/app/rpc/search/internal/common/indexdoc"
)

const defaultTimeoutMs = 3000

type HTTPIndexer struct {
	client       *http.Client
	endpoint     string
	contentIndex string
	userIndex    string
	username     string
	password     string
}

func NewHTTPIndexer(conf indexconfig.IndexEngineConf) *HTTPIndexer {
	timeoutMs := conf.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = defaultTimeoutMs
	}
	return &HTTPIndexer{
		client:       &http.Client{Timeout: time.Duration(timeoutMs) * time.Millisecond},
		endpoint:     strings.TrimRight(conf.Endpoint, "/"),
		contentIndex: defaultString(conf.ContentIndex, "zfeed_content"),
		userIndex:    defaultString(conf.UserIndex, "zfeed_user"),
		username:     conf.Username,
		password:     conf.Password,
	}
}

func (h *HTTPIndexer) IndexContent(ctx context.Context, doc indexdoc.ContentDocument) error {
	return h.put(ctx, h.contentIndex, strconv.FormatInt(doc.ContentID, 10), doc)
}

func (h *HTTPIndexer) DeleteContent(ctx context.Context, contentID int64) error {
	return h.delete(ctx, h.contentIndex, strconv.FormatInt(contentID, 10))
}

func (h *HTTPIndexer) IndexUser(ctx context.Context, doc indexdoc.UserDocument) error {
	return h.put(ctx, h.userIndex, strconv.FormatInt(doc.UserID, 10), doc)
}

func (h *HTTPIndexer) DeleteUser(ctx context.Context, userID int64) error {
	return h.delete(ctx, h.userIndex, strconv.FormatInt(userID, 10))
}

func (h *HTTPIndexer) put(ctx context.Context, index string, id string, doc any) error {
	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	req, err := h.newRequest(ctx, http.MethodPut, index, id, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return h.do(req, http.StatusOK, http.StatusCreated)
}

func (h *HTTPIndexer) delete(ctx context.Context, index string, id string) error {
	req, err := h.newRequest(ctx, http.MethodDelete, index, id, nil)
	if err != nil {
		return err
	}
	return h.do(req, http.StatusOK, http.StatusNotFound)
}

func (h *HTTPIndexer) newRequest(ctx context.Context, method string, index string, id string, body io.Reader) (*http.Request, error) {
	if h.endpoint == "" {
		return nil, fmt.Errorf("search index engine endpoint is empty")
	}
	u := h.endpoint + "/" + url.PathEscape(index) + "/_doc/" + url.PathEscape(id)
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, err
	}
	if h.username != "" {
		req.SetBasicAuth(h.username, h.password)
	}
	return req, nil
}

func (h *HTTPIndexer) do(req *http.Request, expected ...int) error {
	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	for _, code := range expected {
		if resp.StatusCode == code {
			return nil
		}
	}
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("search index request failed: method=%s url=%s status=%d body=%s", req.Method, req.URL.String(), resp.StatusCode, strings.TrimSpace(string(data)))
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
