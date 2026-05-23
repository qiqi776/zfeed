package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	redislock "github.com/zeromicro/go-zero/core/stores/redis"

	"zfeed/app/rpc/search/internal/backend"
	"zfeed/app/rpc/search/internal/querynorm"
	"zfeed/app/rpc/search/internal/repositories"
	"zfeed/app/rpc/search/internal/svc"
)

const (
	searchCacheLayerQuerySnapshot = "query_snapshot"
	searchCacheLayerDocSummary    = "doc_summary"

	searchQueryCacheVersion = 1
	searchDocCacheVersion   = 1

	defaultSearchQueryCacheTTLSeconds = 60
	minSearchQueryCacheTTLSeconds     = 30
	maxSearchQueryCacheTTLSeconds     = 90
	searchQueryCacheTTLJitterSeconds  = 10

	defaultSearchDocCacheTTLSeconds = 600
	minSearchDocCacheTTLSeconds     = 300
	maxSearchDocCacheTTLSeconds     = 600
	searchDocCacheTTLJitterSeconds  = 60

	defaultSearchQueryCacheMaxPages = 3
	maxSearchQueryCacheMaxPages     = 5
	searchQueryRebuildLockSeconds   = 10
)

type searchCachedQuery struct {
	Version     int                      `json:"version"`
	Entity      string                   `json:"entity"`
	Mode        string                   `json:"mode"`
	QueryHash   string                   `json:"query_hash"`
	Page        int                      `json:"page"`
	Limit       int                      `json:"limit"`
	CreatedAt   int64                    `json:"created_at"`
	Meta        repositories.SearchMeta  `json:"meta"`
	UserRows    []searchCachedUserDoc    `json:"user_rows,omitempty"`
	ContentRows []searchCachedContentDoc `json:"content_rows,omitempty"`
}

type searchCachedUserDoc struct {
	Version      int     `json:"version,omitempty"`
	UserID       int64   `json:"user_id"`
	Nickname     string  `json:"nickname"`
	Avatar       string  `json:"avatar"`
	Bio          string  `json:"bio"`
	RankPosition int     `json:"rank_position,omitempty"`
	RankScore    float64 `json:"rank_score,omitempty"`
}

type searchCachedContentDoc struct {
	Version      int     `json:"version,omitempty"`
	ContentID    int64   `json:"content_id"`
	ContentType  int32   `json:"content_type"`
	AuthorID     int64   `json:"author_id"`
	AuthorName   string  `json:"author_name"`
	AuthorAvatar string  `json:"author_avatar"`
	Title        string  `json:"title"`
	CoverURL     string  `json:"cover_url"`
	PublishedAt  int64   `json:"published_at"`
	TextScore    float64 `json:"text_score,omitempty"`
	HotScore     float64 `json:"hot_score,omitempty"`
	RankPosition int     `json:"rank_position,omitempty"`
	RankScore    float64 `json:"rank_score,omitempty"`
}

func (l *SearchUsersLogic) queryUsersWithCache(
	normalized querynorm.Query,
	mode string,
	page int,
	limit int,
	loader func() (backend.SearchUsersResult, error),
) (backend.SearchUsersResult, string, error) {
	if !searchQueryCacheEnabled(l.svcCtx) || !cacheableSearchPage(l.svcCtx, page) {
		result, err := loader()
		return result, cacheStatus(l.svcCtx), err
	}

	cacheKey := searchQueryCacheKey(searchEntityUsers, mode, normalized.Hash, page)
	if result, ok, err := l.getCachedUserQuery(cacheKey, normalized, mode, page); ok || err != nil {
		if err != nil {
			l.Errorf("query users cache failed, key=%s, err=%v", cacheKey, err)
			result, loadErr := loader()
			return result, searchCacheFallback, loadErr
		}
		return result, searchCacheHit, nil
	}

	return l.rebuildUserQueryCacheWithLock(cacheKey, normalized, mode, page, limit, loader)
}

func (l *SearchContentsLogic) queryContentsWithCache(
	normalized querynorm.Query,
	mode string,
	page int,
	limit int,
	loader func() (backend.SearchContentsResult, error),
) (backend.SearchContentsResult, string, error) {
	if !searchQueryCacheEnabled(l.svcCtx) || !cacheableSearchPage(l.svcCtx, page) {
		result, err := loader()
		return result, cacheStatus(l.svcCtx), err
	}

	cacheKey := searchQueryCacheKey(searchEntityContents, mode, normalized.Hash, page)
	if result, ok, err := l.getCachedContentQuery(cacheKey, normalized, mode, page); ok || err != nil {
		if err != nil {
			l.Errorf("query contents cache failed, key=%s, err=%v", cacheKey, err)
			result, loadErr := loader()
			return result, searchCacheFallback, loadErr
		}
		return result, searchCacheHit, nil
	}

	return l.rebuildContentQueryCacheWithLock(cacheKey, normalized, mode, page, limit, loader)
}

func (l *SearchUsersLogic) cachedUsersAfterCursor(
	normalized querynorm.Query,
	mode string,
	cursor int64,
	pageSize int,
) (backend.SearchUsersResult, bool) {
	if !searchQueryCacheEnabled(l.svcCtx) || cursor <= 0 {
		return backend.SearchUsersResult{}, false
	}

	cacheKey := searchQueryCacheKey(searchEntityUsers, mode, normalized.Hash, 0)
	result, ok, err := l.getCachedUserQuery(cacheKey, normalized, mode, 0)
	if err != nil {
		l.Errorf("query users cursor cache failed, key=%s, cursor=%d, err=%v", cacheKey, cursor, err)
		return backend.SearchUsersResult{}, false
	}
	if !ok {
		return backend.SearchUsersResult{}, false
	}

	offset := userRowsOffsetAfterCursor(result.Rows, cursor)
	if !cacheableCursorOffset(l.svcCtx, pageSize, offset) {
		return backend.SearchUsersResult{}, false
	}

	result.Rows = trimRowsForPage(result.Rows[offset:], pageSize)
	return result, true
}

func (l *SearchContentsLogic) cachedContentsAfterCursor(
	normalized querynorm.Query,
	mode string,
	cursor int64,
	pageSize int,
) (backend.SearchContentsResult, bool) {
	if !searchQueryCacheEnabled(l.svcCtx) || cursor <= 0 {
		return backend.SearchContentsResult{}, false
	}

	cacheKey := searchQueryCacheKey(searchEntityContents, mode, normalized.Hash, 0)
	result, ok, err := l.getCachedContentQuery(cacheKey, normalized, mode, 0)
	if err != nil {
		l.Errorf("query contents cursor cache failed, key=%s, cursor=%d, err=%v", cacheKey, cursor, err)
		return backend.SearchContentsResult{}, false
	}
	if !ok {
		return backend.SearchContentsResult{}, false
	}

	offset := contentRowsOffsetAfterCursor(result.Rows, cursor)
	if !cacheableCursorOffset(l.svcCtx, pageSize, offset) {
		return backend.SearchContentsResult{}, false
	}

	result.Rows = trimRowsForPage(result.Rows[offset:], pageSize)
	return result, true
}

func (l *SearchUsersLogic) rebuildUserQueryCacheWithLock(
	cacheKey string,
	normalized querynorm.Query,
	mode string,
	page int,
	limit int,
	loader func() (backend.SearchUsersResult, error),
) (backend.SearchUsersResult, string, error) {
	lock := redislock.NewRedisLock(l.svcCtx.Redis, searchQueryRebuildLockKey(cacheKey))
	lock.SetExpire(searchQueryRebuildLockSeconds)

	lockAcquired, err := lock.AcquireCtx(l.ctx)
	if err != nil {
		l.Errorf("acquire users search cache lock failed, key=%s, err=%v", cacheKey, err)
		result, loadErr := loader()
		return result, searchCacheFallback, loadErr
	}
	if !lockAcquired {
		if result, ok := l.waitUserQueryCache(cacheKey, normalized, mode, page); ok {
			return result, searchCacheHit, nil
		}
		result, loadErr := loader()
		return result, searchCacheFallback, loadErr
	}

	defer func() {
		if releaseOK, releaseErr := lock.ReleaseCtx(l.ctx); !releaseOK || releaseErr != nil {
			l.Errorf("release users search cache lock failed, key=%s, err=%v", cacheKey, releaseErr)
		}
	}()

	if result, ok, err := l.getCachedUserQuery(cacheKey, normalized, mode, page); ok || err != nil {
		if err != nil {
			l.Errorf("recheck users search cache failed, key=%s, err=%v", cacheKey, err)
		} else {
			return result, searchCacheHit, nil
		}
	}

	result, loadErr := loader()
	if loadErr != nil {
		return result, searchCacheFallback, loadErr
	}
	l.setCachedUserQuery(cacheKey, normalized, mode, page, limit, result)
	return result, searchCacheMiss, nil
}

func (l *SearchContentsLogic) rebuildContentQueryCacheWithLock(
	cacheKey string,
	normalized querynorm.Query,
	mode string,
	page int,
	limit int,
	loader func() (backend.SearchContentsResult, error),
) (backend.SearchContentsResult, string, error) {
	lock := redislock.NewRedisLock(l.svcCtx.Redis, searchQueryRebuildLockKey(cacheKey))
	lock.SetExpire(searchQueryRebuildLockSeconds)

	lockAcquired, err := lock.AcquireCtx(l.ctx)
	if err != nil {
		l.Errorf("acquire contents search cache lock failed, key=%s, err=%v", cacheKey, err)
		result, loadErr := loader()
		return result, searchCacheFallback, loadErr
	}
	if !lockAcquired {
		if result, ok := l.waitContentQueryCache(cacheKey, normalized, mode, page); ok {
			return result, searchCacheHit, nil
		}
		result, loadErr := loader()
		return result, searchCacheFallback, loadErr
	}

	defer func() {
		if releaseOK, releaseErr := lock.ReleaseCtx(l.ctx); !releaseOK || releaseErr != nil {
			l.Errorf("release contents search cache lock failed, key=%s, err=%v", cacheKey, releaseErr)
		}
	}()

	if result, ok, err := l.getCachedContentQuery(cacheKey, normalized, mode, page); ok || err != nil {
		if err != nil {
			l.Errorf("recheck contents search cache failed, key=%s, err=%v", cacheKey, err)
		} else {
			return result, searchCacheHit, nil
		}
	}

	result, loadErr := loader()
	if loadErr != nil {
		return result, searchCacheFallback, loadErr
	}
	l.setCachedContentQuery(cacheKey, normalized, mode, page, limit, result)
	return result, searchCacheMiss, nil
}

func (l *SearchUsersLogic) waitUserQueryCache(cacheKey string, normalized querynorm.Query, mode string, page int) (backend.SearchUsersResult, bool) {
	for i := 0; i < 5; i++ {
		select {
		case <-l.ctx.Done():
			return backend.SearchUsersResult{}, false
		default:
		}
		time.Sleep(time.Duration(30+rand.Intn(50)) * time.Millisecond)

		result, ok, err := l.getCachedUserQuery(cacheKey, normalized, mode, page)
		if err != nil {
			l.Errorf("wait users search cache failed, key=%s, err=%v", cacheKey, err)
			return backend.SearchUsersResult{}, false
		}
		if ok {
			return result, true
		}
	}
	return backend.SearchUsersResult{}, false
}

func (l *SearchContentsLogic) waitContentQueryCache(cacheKey string, normalized querynorm.Query, mode string, page int) (backend.SearchContentsResult, bool) {
	for i := 0; i < 5; i++ {
		select {
		case <-l.ctx.Done():
			return backend.SearchContentsResult{}, false
		default:
		}
		time.Sleep(time.Duration(30+rand.Intn(50)) * time.Millisecond)

		result, ok, err := l.getCachedContentQuery(cacheKey, normalized, mode, page)
		if err != nil {
			l.Errorf("wait contents search cache failed, key=%s, err=%v", cacheKey, err)
			return backend.SearchContentsResult{}, false
		}
		if ok {
			return result, true
		}
	}
	return backend.SearchContentsResult{}, false
}

func (l *SearchUsersLogic) getCachedUserQuery(cacheKey string, normalized querynorm.Query, mode string, page int) (backend.SearchUsersResult, bool, error) {
	payload, ok, err := getCachedQuery(l.ctx, l.svcCtx, cacheKey, searchEntityUsers, mode, normalized.Hash, page)
	if err != nil || !ok {
		return backend.SearchUsersResult{}, false, err
	}
	rows := l.userRowsFromCachedQuery(payload, mode)
	return backend.SearchUsersResult{Rows: rows, Meta: payload.Meta}, true, nil
}

func (l *SearchContentsLogic) getCachedContentQuery(cacheKey string, normalized querynorm.Query, mode string, page int) (backend.SearchContentsResult, bool, error) {
	payload, ok, err := getCachedQuery(l.ctx, l.svcCtx, cacheKey, searchEntityContents, mode, normalized.Hash, page)
	if err != nil || !ok {
		return backend.SearchContentsResult{}, false, err
	}
	rows := l.contentRowsFromCachedQuery(payload, mode)
	return backend.SearchContentsResult{Rows: rows, Meta: payload.Meta}, true, nil
}

func getCachedQuery(ctx context.Context, svcCtx *svc.ServiceContext, cacheKey string, entity string, mode string, queryHash string, page int) (searchCachedQuery, bool, error) {
	raw, err := svcCtx.Redis.GetCtx(ctx, cacheKey)
	if err != nil {
		observeSearchCache(searchCacheLayerQuerySnapshot, entity, mode, searchCacheError)
		return searchCachedQuery{}, false, err
	}
	if raw == "" {
		observeSearchCache(searchCacheLayerQuerySnapshot, entity, mode, searchCacheMiss)
		return searchCachedQuery{}, false, nil
	}

	var payload searchCachedQuery
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		observeSearchCache(searchCacheLayerQuerySnapshot, entity, mode, searchCacheError)
		return searchCachedQuery{}, false, err
	}
	if payload.Version != searchQueryCacheVersion ||
		payload.Entity != entity ||
		payload.Mode != mode ||
		payload.QueryHash != queryHash ||
		payload.Page != page {
		observeSearchCache(searchCacheLayerQuerySnapshot, entity, mode, searchCacheError)
		return searchCachedQuery{}, false, fmt.Errorf("invalid search query cache payload")
	}

	observeSearchCache(searchCacheLayerQuerySnapshot, entity, mode, searchCacheHit)
	return payload, true, nil
}

func (l *SearchUsersLogic) setCachedUserQuery(cacheKey string, normalized querynorm.Query, mode string, page int, limit int, result backend.SearchUsersResult) {
	payload := searchCachedQuery{
		Version:   searchQueryCacheVersion,
		Entity:    searchEntityUsers,
		Mode:      mode,
		QueryHash: normalized.Hash,
		Page:      page,
		Limit:     limit,
		CreatedAt: time.Now().Unix(),
		Meta:      result.Meta,
		UserRows:  cachedUserDocsFromRows(result.Rows),
	}
	if err := setCachedQuery(l.ctx, l.svcCtx, cacheKey, payload); err != nil {
		l.Errorf("set users search query cache failed, key=%s, err=%v", cacheKey, err)
	}
	l.setUserDocSummaries(result.Rows, mode)
}

func (l *SearchContentsLogic) setCachedContentQuery(cacheKey string, normalized querynorm.Query, mode string, page int, limit int, result backend.SearchContentsResult) {
	payload := searchCachedQuery{
		Version:     searchQueryCacheVersion,
		Entity:      searchEntityContents,
		Mode:        mode,
		QueryHash:   normalized.Hash,
		Page:        page,
		Limit:       limit,
		CreatedAt:   time.Now().Unix(),
		Meta:        result.Meta,
		ContentRows: cachedContentDocsFromRows(result.Rows),
	}
	if err := setCachedQuery(l.ctx, l.svcCtx, cacheKey, payload); err != nil {
		l.Errorf("set contents search query cache failed, key=%s, err=%v", cacheKey, err)
	}
	l.setContentDocSummaries(result.Rows, mode)
}

func setCachedQuery(ctx context.Context, svcCtx *svc.ServiceContext, cacheKey string, payload searchCachedQuery) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return svcCtx.Redis.SetexCtx(ctx, cacheKey, string(data), searchQueryCacheTTLSeconds(svcCtx))
}

func (l *SearchUsersLogic) userRowsFromCachedQuery(payload searchCachedQuery, mode string) []repositories.SearchUserRow {
	rows := make([]repositories.SearchUserRow, 0, len(payload.UserRows))
	for _, cached := range payload.UserRows {
		row, ok := l.getUserDocSummary(cached.UserID, mode)
		if !ok {
			row = cached.toUserRow()
			l.setUserDocSummary(row, mode)
		}
		rows = append(rows, row)
	}
	return rows
}

func (l *SearchContentsLogic) contentRowsFromCachedQuery(payload searchCachedQuery, mode string) []repositories.SearchContentRow {
	rows := make([]repositories.SearchContentRow, 0, len(payload.ContentRows))
	for _, cached := range payload.ContentRows {
		row, ok := l.getContentDocSummary(cached.ContentID, mode)
		if !ok {
			row = cached.toContentRow()
			l.setContentDocSummary(row, mode)
		}
		rows = append(rows, row)
	}
	return rows
}

func (l *SearchUsersLogic) setUserDocSummaries(rows []repositories.SearchUserRow, mode string) {
	for _, row := range rows {
		l.setUserDocSummary(row, mode)
	}
}

func (l *SearchUsersLogic) setUserDocSummary(row repositories.SearchUserRow, mode string) {
	if !searchQueryCacheEnabled(l.svcCtx) || row.UserID <= 0 {
		return
	}
	data, err := json.Marshal(cachedUserDocFromRow(row, 0))
	if err != nil {
		l.Errorf("marshal user doc summary failed, user_id=%d, err=%v", row.UserID, err)
		return
	}
	if err := l.svcCtx.Redis.SetexCtx(l.ctx, searchUserDocCacheKey(row.UserID), string(data), searchDocCacheTTLSeconds(l.svcCtx)); err != nil {
		l.Errorf("set user doc summary cache failed, user_id=%d, err=%v", row.UserID, err)
		observeSearchCache(searchCacheLayerDocSummary, searchEntityUsers, mode, searchCacheError)
	}
}

func (l *SearchContentsLogic) setContentDocSummaries(rows []repositories.SearchContentRow, mode string) {
	for _, row := range rows {
		l.setContentDocSummary(row, mode)
	}
}

func (l *SearchContentsLogic) setContentDocSummary(row repositories.SearchContentRow, mode string) {
	if !searchQueryCacheEnabled(l.svcCtx) || row.ContentID <= 0 {
		return
	}
	data, err := json.Marshal(cachedContentDocFromRow(row, 0))
	if err != nil {
		l.Errorf("marshal content doc summary failed, content_id=%d, err=%v", row.ContentID, err)
		return
	}
	if err := l.svcCtx.Redis.SetexCtx(l.ctx, searchContentDocCacheKey(row.ContentID), string(data), searchDocCacheTTLSeconds(l.svcCtx)); err != nil {
		l.Errorf("set content doc summary cache failed, content_id=%d, err=%v", row.ContentID, err)
		observeSearchCache(searchCacheLayerDocSummary, searchEntityContents, mode, searchCacheError)
	}
}

func (l *SearchUsersLogic) getUserDocSummary(userID int64, mode string) (repositories.SearchUserRow, bool) {
	raw, err := l.svcCtx.Redis.GetCtx(l.ctx, searchUserDocCacheKey(userID))
	if err != nil {
		l.Errorf("get user doc summary cache failed, user_id=%d, err=%v", userID, err)
		observeSearchCache(searchCacheLayerDocSummary, searchEntityUsers, mode, searchCacheError)
		return repositories.SearchUserRow{}, false
	}
	if raw == "" {
		observeSearchCache(searchCacheLayerDocSummary, searchEntityUsers, mode, searchCacheMiss)
		return repositories.SearchUserRow{}, false
	}

	var cached searchCachedUserDoc
	if err := json.Unmarshal([]byte(raw), &cached); err != nil {
		l.Errorf("unmarshal user doc summary cache failed, user_id=%d, err=%v", userID, err)
		observeSearchCache(searchCacheLayerDocSummary, searchEntityUsers, mode, searchCacheError)
		return repositories.SearchUserRow{}, false
	}
	if cached.Version != searchDocCacheVersion || cached.UserID != userID {
		observeSearchCache(searchCacheLayerDocSummary, searchEntityUsers, mode, searchCacheError)
		return repositories.SearchUserRow{}, false
	}

	observeSearchCache(searchCacheLayerDocSummary, searchEntityUsers, mode, searchCacheHit)
	return cached.toUserRow(), true
}

func (l *SearchContentsLogic) getContentDocSummary(contentID int64, mode string) (repositories.SearchContentRow, bool) {
	raw, err := l.svcCtx.Redis.GetCtx(l.ctx, searchContentDocCacheKey(contentID))
	if err != nil {
		l.Errorf("get content doc summary cache failed, content_id=%d, err=%v", contentID, err)
		observeSearchCache(searchCacheLayerDocSummary, searchEntityContents, mode, searchCacheError)
		return repositories.SearchContentRow{}, false
	}
	if raw == "" {
		observeSearchCache(searchCacheLayerDocSummary, searchEntityContents, mode, searchCacheMiss)
		return repositories.SearchContentRow{}, false
	}

	var cached searchCachedContentDoc
	if err := json.Unmarshal([]byte(raw), &cached); err != nil {
		l.Errorf("unmarshal content doc summary cache failed, content_id=%d, err=%v", contentID, err)
		observeSearchCache(searchCacheLayerDocSummary, searchEntityContents, mode, searchCacheError)
		return repositories.SearchContentRow{}, false
	}
	if cached.Version != searchDocCacheVersion || cached.ContentID != contentID {
		observeSearchCache(searchCacheLayerDocSummary, searchEntityContents, mode, searchCacheError)
		return repositories.SearchContentRow{}, false
	}

	observeSearchCache(searchCacheLayerDocSummary, searchEntityContents, mode, searchCacheHit)
	return cached.toContentRow(), true
}

func cachedUserDocsFromRows(rows []repositories.SearchUserRow) []searchCachedUserDoc {
	docs := make([]searchCachedUserDoc, 0, len(rows))
	for index, row := range rows {
		docs = append(docs, cachedUserDocFromRow(row, index+1))
	}
	return docs
}

func cachedContentDocsFromRows(rows []repositories.SearchContentRow) []searchCachedContentDoc {
	docs := make([]searchCachedContentDoc, 0, len(rows))
	for index, row := range rows {
		docs = append(docs, cachedContentDocFromRow(row, index+1))
	}
	return docs
}

func cachedUserDocFromRow(row repositories.SearchUserRow, rankPosition int) searchCachedUserDoc {
	return searchCachedUserDoc{
		Version:      searchDocCacheVersion,
		UserID:       row.UserID,
		Nickname:     row.Nickname,
		Avatar:       row.Avatar,
		Bio:          row.Bio,
		RankPosition: rankPosition,
	}
}

func cachedContentDocFromRow(row repositories.SearchContentRow, rankPosition int) searchCachedContentDoc {
	return searchCachedContentDoc{
		Version:      searchDocCacheVersion,
		ContentID:    row.ContentID,
		ContentType:  row.ContentType,
		AuthorID:     row.AuthorID,
		AuthorName:   row.AuthorName,
		AuthorAvatar: row.AuthorAvatar,
		Title:        row.Title,
		CoverURL:     row.CoverURL,
		PublishedAt:  unixOrZero(row.PublishedAt),
		TextScore:    row.TextScore,
		HotScore:     row.HotScore,
		RankPosition: rankPosition,
		RankScore:    row.RankScore,
	}
}

func (d searchCachedUserDoc) toUserRow() repositories.SearchUserRow {
	return repositories.SearchUserRow{
		UserID:   d.UserID,
		Nickname: d.Nickname,
		Avatar:   d.Avatar,
		Bio:      d.Bio,
	}
}

func (d searchCachedContentDoc) toContentRow() repositories.SearchContentRow {
	return repositories.SearchContentRow{
		ContentID:    d.ContentID,
		ContentType:  d.ContentType,
		AuthorID:     d.AuthorID,
		AuthorName:   d.AuthorName,
		AuthorAvatar: d.AuthorAvatar,
		Title:        d.Title,
		CoverURL:     d.CoverURL,
		PublishedAt:  unixTimePtr(d.PublishedAt),
		TextScore:    d.TextScore,
		HotScore:     d.HotScore,
		RankScore:    d.RankScore,
	}
}

func unixTimePtr(value int64) *time.Time {
	if value <= 0 {
		return nil
	}
	t := time.Unix(value, 0)
	return &t
}

func searchQueryCacheEnabled(svcCtx *svc.ServiceContext) bool {
	return svcCtx != nil && svcCtx.Redis != nil && svcCtx.Config.SearchCacheEnabled
}

func cacheableSearchPage(svcCtx *svc.ServiceContext, page int) bool {
	return page >= 0 && page < searchQueryCacheMaxPages(svcCtx)
}

func cacheableCursorOffset(svcCtx *svc.ServiceContext, pageSize int, offset int) bool {
	if offset < 0 {
		return false
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	return offset < pageSize*searchQueryCacheMaxPages(svcCtx)
}

func latestQueryCacheLimit(svcCtx *svc.ServiceContext) int {
	return maxSearchPageSize*searchQueryCacheMaxPages(svcCtx) + 1
}

func searchQueryCacheMaxPages(svcCtx *svc.ServiceContext) int {
	value := defaultSearchQueryCacheMaxPages
	if svcCtx != nil && svcCtx.Config.SearchQueryCacheMaxPages > 0 {
		value = svcCtx.Config.SearchQueryCacheMaxPages
	}
	if value > maxSearchQueryCacheMaxPages {
		return maxSearchQueryCacheMaxPages
	}
	return value
}

func searchQueryCacheTTLSeconds(svcCtx *svc.ServiceContext) int {
	value := defaultSearchQueryCacheTTLSeconds
	if svcCtx != nil && svcCtx.Config.SearchQueryCacheTTLSeconds > 0 {
		value = svcCtx.Config.SearchQueryCacheTTLSeconds
	}
	return ttlWithJitter(value, minSearchQueryCacheTTLSeconds, maxSearchQueryCacheTTLSeconds, searchQueryCacheTTLJitterSeconds)
}

func searchDocCacheTTLSeconds(svcCtx *svc.ServiceContext) int {
	value := defaultSearchDocCacheTTLSeconds
	if svcCtx != nil && svcCtx.Config.SearchDocCacheTTLSeconds > 0 {
		value = svcCtx.Config.SearchDocCacheTTLSeconds
	}
	return ttlWithJitter(value, minSearchDocCacheTTLSeconds, maxSearchDocCacheTTLSeconds, searchDocCacheTTLJitterSeconds)
}

func ttlWithJitter(base int, minValue int, maxValue int, jitter int) int {
	if base < minValue {
		base = minValue
	}
	if base > maxValue {
		base = maxValue
	}

	low := base - jitter
	if low < minValue {
		low = minValue
	}
	high := base + jitter
	if high > maxValue {
		high = maxValue
	}
	if high <= low {
		return base
	}
	return low + rand.Intn(high-low+1)
}

func searchQueryCacheKey(entity string, mode string, queryHash string, page int) string {
	return fmt.Sprintf("search:q:v1:%s:%s:%s:%d", entity, mode, queryHash, page)
}

func searchQueryRebuildLockKey(cacheKey string) string {
	return "search:lock:v1:" + cacheKey
}

func searchUserDocCacheKey(userID int64) string {
	return fmt.Sprintf("search:doc:v1:user:%d", userID)
}

func searchContentDocCacheKey(contentID int64) string {
	return fmt.Sprintf("search:doc:v1:content:%d", contentID)
}

func userRowsOffsetAfterCursor(rows []repositories.SearchUserRow, cursor int64) int {
	for index, row := range rows {
		if row.UserID == cursor {
			return index + 1
		}
	}
	return -1
}

func contentRowsOffsetAfterCursor(rows []repositories.SearchContentRow, cursor int64) int {
	for index, row := range rows {
		if row.ContentID == cursor {
			return index + 1
		}
	}
	return -1
}

func trimRowsForPage[T any](rows []T, pageSize int) []T {
	if pageSize <= 0 {
		pageSize = 10
	}
	limit := pageSize + 1
	if len(rows) > limit {
		return rows[:limit]
	}
	return rows
}
