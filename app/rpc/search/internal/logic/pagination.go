package logic

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"zfeed/app/rpc/search/internal/svc"
	"zfeed/pkg/errorx"
)

const (
	searchModeLatest    = "latest"
	searchModeRelevance = "relevance"
	searchModeHybrid    = "hybrid"

	defaultSearchSnapshotTTLSeconds = 60
	defaultSearchSnapshotMaxItems   = 100
	minSearchSnapshotTTLSeconds     = 30
	maxSearchSnapshotTTLSeconds     = 90
	maxSearchSnapshotItems          = 500
)

type snapshotPageToken struct {
	SnapshotID string `json:"snapshot_id"`
	Offset     int    `json:"offset"`
}

func normalizeSearchMode(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", searchModeLatest:
		return searchModeLatest, nil
	case searchModeRelevance:
		return searchModeRelevance, nil
	case searchModeHybrid:
		return searchModeHybrid, nil
	default:
		return "", errorx.NewBadRequest("搜索模式不支持")
	}
}

func usesSnapshotPagination(mode string) bool {
	return mode == searchModeRelevance || mode == searchModeHybrid
}

func snapshotPaginationEnabled(svcCtx *svc.ServiceContext) bool {
	return svcCtx != nil && svcCtx.Config.SearchSnapshotEnabled
}

func encodePageToken(snapshotID string, offset int) (string, error) {
	if strings.TrimSpace(snapshotID) == "" || offset < 0 {
		return "", errorx.NewBadRequest("分页参数错误")
	}

	data, err := json.Marshal(snapshotPageToken{
		SnapshotID: snapshotID,
		Offset:     offset,
	})
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(data), nil
}

func decodePageToken(token string) (snapshotPageToken, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return snapshotPageToken{}, errorx.NewBadRequest("分页令牌不能为空")
	}

	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return snapshotPageToken{}, errorx.NewBadRequest("分页令牌无效")
	}

	var decoded snapshotPageToken
	if err := json.Unmarshal(data, &decoded); err != nil {
		return snapshotPageToken{}, errorx.NewBadRequest("分页令牌无效")
	}
	if strings.TrimSpace(decoded.SnapshotID) == "" || decoded.Offset < 0 {
		return snapshotPageToken{}, errorx.NewBadRequest("分页令牌无效")
	}

	return decoded, nil
}

func snapshotPageRequest(pageToken string, snapshotID string) (snapshotPageToken, bool, error) {
	pageToken = strings.TrimSpace(pageToken)
	snapshotID = strings.TrimSpace(snapshotID)
	if pageToken == "" {
		if snapshotID == "" {
			return snapshotPageToken{}, false, nil
		}
		return snapshotPageToken{SnapshotID: snapshotID}, true, nil
	}

	decoded, err := decodePageToken(pageToken)
	if err != nil {
		return snapshotPageToken{}, false, err
	}
	if snapshotID != "" && snapshotID != decoded.SnapshotID {
		return snapshotPageToken{}, false, errorx.NewBadRequest("分页令牌与快照不匹配")
	}

	return decoded, true, nil
}

func pageRows[T any](rows []T, offset int, pageSize int) ([]T, bool, int) {
	if offset < 0 {
		offset = 0
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if offset >= len(rows) {
		return []T{}, false, len(rows)
	}

	end := offset + pageSize
	if end > len(rows) {
		end = len(rows)
	}

	return rows[offset:end], end < len(rows), end
}

func snapshotTTLSeconds(svcCtx *svc.ServiceContext) int {
	ttl := defaultSearchSnapshotTTLSeconds
	if svcCtx != nil && svcCtx.Config.SearchSnapshotTTLSeconds > 0 {
		ttl = svcCtx.Config.SearchSnapshotTTLSeconds
	}
	if ttl < minSearchSnapshotTTLSeconds {
		return minSearchSnapshotTTLSeconds
	}
	if ttl > maxSearchSnapshotTTLSeconds {
		return maxSearchSnapshotTTLSeconds
	}
	return ttl
}

func snapshotMaxItems(svcCtx *svc.ServiceContext, pageSize int) int {
	limit := defaultSearchSnapshotMaxItems
	if svcCtx != nil && svcCtx.Config.SearchSnapshotMaxItems > 0 {
		limit = svcCtx.Config.SearchSnapshotMaxItems
	}
	if limit < pageSize {
		limit = pageSize
	}
	if limit > maxSearchSnapshotItems {
		return maxSearchSnapshotItems
	}
	return limit
}
