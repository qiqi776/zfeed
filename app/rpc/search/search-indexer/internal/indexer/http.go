package indexer

import (
	"bufio"
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

	"zfeed/app/rpc/search/internal/common/indexdoc"
	"zfeed/app/rpc/search/search-indexer/internal/indexconfig"
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

func (h *HTTPIndexer) IndexContent(ctx context.Context, doc indexdoc.ContentDocument) (err error) {
	start := time.Now()
	defer func() { observeWrite(indexEntityContent, indexOperationIndex, start, err) }()
	return h.put(ctx, h.contentIndex, strconv.FormatInt(doc.ContentID, 10), doc)
}

func (h *HTTPIndexer) DeleteContent(ctx context.Context, contentID int64) (err error) {
	start := time.Now()
	defer func() { observeWrite(indexEntityContent, indexOperationDelete, start, err) }()
	return h.delete(ctx, h.contentIndex, strconv.FormatInt(contentID, 10))
}

func (h *HTTPIndexer) IndexUser(ctx context.Context, doc indexdoc.UserDocument) (err error) {
	start := time.Now()
	defer func() { observeWrite(indexEntityUser, indexOperationIndex, start, err) }()
	return h.put(ctx, h.userIndex, strconv.FormatInt(doc.UserID, 10), doc)
}

func (h *HTTPIndexer) DeleteUser(ctx context.Context, userID int64) (err error) {
	start := time.Now()
	defer func() { observeWrite(indexEntityUser, indexOperationDelete, start, err) }()
	return h.delete(ctx, h.userIndex, strconv.FormatInt(userID, 10))
}

func (h *HTTPIndexer) BulkIndexContent(ctx context.Context, docs []indexdoc.ContentDocument) (err error) {
	start := time.Now()
	defer func() { observeWrite(indexEntityContent, indexOperationBulkIndex, start, err) }()
	if len(docs) == 0 {
		return nil
	}
	operations := make([]bulkOperation, 0, len(docs))
	for _, doc := range docs {
		operations = append(operations, bulkOperation{
			Action: "index",
			Index:  h.contentIndex,
			ID:     strconv.FormatInt(doc.ContentID, 10),
			Doc:    doc,
		})
	}
	return h.bulk(ctx, operations)
}

func (h *HTTPIndexer) BulkDeleteContent(ctx context.Context, contentIDs []int64) (err error) {
	start := time.Now()
	defer func() { observeWrite(indexEntityContent, indexOperationBulkDelete, start, err) }()
	if len(contentIDs) == 0 {
		return nil
	}
	operations := make([]bulkOperation, 0, len(contentIDs))
	for _, id := range contentIDs {
		if id <= 0 {
			continue
		}
		operations = append(operations, bulkOperation{
			Action: "delete",
			Index:  h.contentIndex,
			ID:     strconv.FormatInt(id, 10),
		})
	}
	return h.bulk(ctx, operations)
}

func (h *HTTPIndexer) BulkIndexUser(ctx context.Context, docs []indexdoc.UserDocument) (err error) {
	start := time.Now()
	defer func() { observeWrite(indexEntityUser, indexOperationBulkIndex, start, err) }()
	if len(docs) == 0 {
		return nil
	}
	operations := make([]bulkOperation, 0, len(docs))
	for _, doc := range docs {
		operations = append(operations, bulkOperation{
			Action: "index",
			Index:  h.userIndex,
			ID:     strconv.FormatInt(doc.UserID, 10),
			Doc:    doc,
		})
	}
	return h.bulk(ctx, operations)
}

func (h *HTTPIndexer) BulkDeleteUser(ctx context.Context, userIDs []int64) (err error) {
	start := time.Now()
	defer func() { observeWrite(indexEntityUser, indexOperationBulkDelete, start, err) }()
	if len(userIDs) == 0 {
		return nil
	}
	operations := make([]bulkOperation, 0, len(userIDs))
	for _, id := range userIDs {
		if id <= 0 {
			continue
		}
		operations = append(operations, bulkOperation{
			Action: "delete",
			Index:  h.userIndex,
			ID:     strconv.FormatInt(id, 10),
		})
	}
	return h.bulk(ctx, operations)
}

func (h *HTTPIndexer) Count(ctx context.Context, index string) (int64, error) {
	var out struct {
		Count int64 `json:"count"`
	}
	if err := h.postJSON(ctx, index, "_count", map[string]any{"query": map[string]any{"match_all": map[string]any{}}}, &out); err != nil {
		return 0, err
	}
	return out.Count, nil
}

func (h *HTTPIndexer) SwitchAlias(ctx context.Context, alias string, targetIndex string) error {
	body := map[string]any{
		"actions": []map[string]any{
			{"remove": map[string]any{"index": "*", "alias": alias, "must_exist": false}},
			{"add": map[string]any{"index": targetIndex, "alias": alias}},
		},
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := h.newRootRequest(ctx, http.MethodPost, "_aliases", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return h.do(req, http.StatusOK)
}

func (h *HTTPIndexer) GetContentDocuments(ctx context.Context, index string, ids []int64) (map[int64]indexdoc.ContentDocument, error) {
	out := make(map[int64]indexdoc.ContentDocument, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var resp mgetContentResponse
	if err := h.mget(ctx, index, int64IDs(ids), &resp); err != nil {
		return nil, err
	}
	for _, doc := range resp.Docs {
		if doc.Found {
			out[doc.Source.ContentID] = doc.Source
		}
	}
	return out, nil
}

func (h *HTTPIndexer) GetUserDocuments(ctx context.Context, index string, ids []int64) (map[int64]indexdoc.UserDocument, error) {
	out := make(map[int64]indexdoc.UserDocument, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var resp mgetUserResponse
	if err := h.mget(ctx, index, int64IDs(ids), &resp); err != nil {
		return nil, err
	}
	for _, doc := range resp.Docs {
		if doc.Found {
			out[doc.Source.UserID] = doc.Source
		}
	}
	return out, nil
}

func (h *HTTPIndexer) SearchContentIDs(ctx context.Context, index string, query string, mode string, limit int) ([]int64, error) {
	if limit <= 0 {
		limit = 20
	}
	body := map[string]any{
		"size": limit,
		"_source": []string{
			"content_id",
		},
		"query": map[string]any{
			"bool": map[string]any{
				"must": map[string]any{"multi_match": map[string]any{
					"query":  query,
					"fields": []string{"title^3", "description", "author_name"},
				}},
				"filter": []map[string]any{
					{"term": map[string]any{"status": 30}},
					{"term": map[string]any{"visibility": 10}},
				},
			},
		},
		"sort": contentSort(mode),
	}
	var resp searchContentIDResponse
	if err := h.postJSON(ctx, index, "_search", body, &resp); err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		if hit.Source.ContentID > 0 {
			ids = append(ids, hit.Source.ContentID)
		}
	}
	return ids, nil
}

func (h *HTTPIndexer) SearchUserIDs(ctx context.Context, index string, query string, limit int) ([]int64, error) {
	if limit <= 0 {
		limit = 20
	}
	body := map[string]any{
		"size": limit,
		"_source": []string{
			"user_id",
		},
		"query": map[string]any{
			"bool": map[string]any{
				"must": map[string]any{"multi_match": map[string]any{
					"query":  query,
					"fields": []string{"nickname^3", "bio", "mobile_search_field"},
				}},
				"filter": []map[string]any{{"term": map[string]any{"status": 10}}},
			},
		},
		"sort": []map[string]any{{"_score": "desc"}, {"user_id": "desc"}},
	}
	var resp searchUserIDResponse
	if err := h.postJSON(ctx, index, "_search", body, &resp); err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		if hit.Source.UserID > 0 {
			ids = append(ids, hit.Source.UserID)
		}
	}
	return ids, nil
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

func (h *HTTPIndexer) bulk(ctx context.Context, operations []bulkOperation) error {
	if len(operations) == 0 {
		return nil
	}

	var body bytes.Buffer
	writer := bufio.NewWriter(&body)
	encoder := json.NewEncoder(writer)
	for _, op := range operations {
		if op.ID == "" || op.Index == "" {
			continue
		}
		meta := map[string]any{op.Action: map[string]any{"_index": op.Index, "_id": op.ID}}
		if err := encoder.Encode(meta); err != nil {
			return err
		}
		if op.Action != "delete" {
			if err := encoder.Encode(op.Doc); err != nil {
				return err
			}
		}
	}
	if err := writer.Flush(); err != nil {
		return err
	}
	if body.Len() == 0 {
		return nil
	}

	var resp bulkResponse
	if err := h.postRootJSON(ctx, "_bulk", bytes.NewReader(body.Bytes()), &resp); err != nil {
		return err
	}
	if resp.Errors {
		for _, item := range resp.Items {
			for action, result := range item {
				if result.Status >= 300 && result.Status != http.StatusNotFound {
					return fmt.Errorf("search index bulk item failed: action=%s index=%s id=%s status=%d reason=%s", action, result.Index, result.ID, result.Status, result.Error.Reason)
				}
			}
		}
		return fmt.Errorf("search index bulk request failed")
	}
	return nil
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

func (h *HTTPIndexer) newIndexRequest(ctx context.Context, method string, index string, path string, body io.Reader) (*http.Request, error) {
	if h.endpoint == "" {
		return nil, fmt.Errorf("search index engine endpoint is empty")
	}
	u := h.endpoint + "/" + url.PathEscape(index) + "/" + strings.TrimLeft(path, "/")
	return h.newRawRequest(ctx, method, u, body)
}

func (h *HTTPIndexer) newRootRequest(ctx context.Context, method string, path string, body io.Reader) (*http.Request, error) {
	if h.endpoint == "" {
		return nil, fmt.Errorf("search index engine endpoint is empty")
	}
	u := h.endpoint + "/" + strings.TrimLeft(path, "/")
	return h.newRawRequest(ctx, method, u, body)
}

func (h *HTTPIndexer) newRawRequest(ctx context.Context, method string, rawURL string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return nil, err
	}
	if h.username != "" {
		req.SetBasicAuth(h.username, h.password)
	}
	return req, nil
}

func (h *HTTPIndexer) postJSON(ctx context.Context, index string, path string, body any, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := h.newIndexRequest(ctx, http.MethodPost, index, path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return h.doJSON(req, out)
}

func (h *HTTPIndexer) postRootJSON(ctx context.Context, path string, body io.Reader, out any) error {
	req, err := h.newRootRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-ndjson")
	return h.doJSON(req, out)
}

func (h *HTTPIndexer) mget(ctx context.Context, index string, ids []string, out any) error {
	body := map[string]any{"ids": ids}
	return h.postJSON(ctx, index, "_mget", body, out)
}

func (h *HTTPIndexer) doJSON(req *http.Request, out any) error {
	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("search index request failed: method=%s url=%s status=%d body=%s", req.Method, req.URL.String(), resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
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

type bulkOperation struct {
	Action string
	Index  string
	ID     string
	Doc    any
}

type bulkResponse struct {
	Errors bool                       `json:"errors"`
	Items  []map[string]bulkItemState `json:"items"`
}

type bulkItemState struct {
	Index  string `json:"_index"`
	ID     string `json:"_id"`
	Status int    `json:"status"`
	Error  struct {
		Reason string `json:"reason"`
	} `json:"error"`
}

type mgetContentResponse struct {
	Docs []struct {
		Found  bool                     `json:"found"`
		Source indexdoc.ContentDocument `json:"_source"`
	} `json:"docs"`
}

type mgetUserResponse struct {
	Docs []struct {
		Found  bool                  `json:"found"`
		Source indexdoc.UserDocument `json:"_source"`
	} `json:"docs"`
}

type searchContentIDResponse struct {
	Hits struct {
		Hits []struct {
			Source struct {
				ContentID int64 `json:"content_id"`
			} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

type searchUserIDResponse struct {
	Hits struct {
		Hits []struct {
			Source struct {
				UserID int64 `json:"user_id"`
			} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

func int64IDs(ids []int64) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id > 0 {
			out = append(out, strconv.FormatInt(id, 10))
		}
	}
	return out
}

func contentSort(mode string) []map[string]any {
	switch mode {
	case "latest":
		return []map[string]any{{"published_at": "desc"}, {"content_id": "desc"}}
	case "hybrid":
		return []map[string]any{{"_score": "desc"}, {"hot_score": "desc"}, {"content_id": "desc"}}
	default:
		return []map[string]any{{"_score": "desc"}, {"content_id": "desc"}}
	}
}
