package strategy

import (
	"encoding/json"
	"strconv"
	"strings"
)

func getInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case nil:
		return 0, false
	case int:
		return int64(n), true
	case int32:
		return int64(n), true
	case int64:
		return n, true
	case uint:
		return int64(n), true
	case uint32:
		return int64(n), true
	case uint64:
		return int64(n), true
	case float64:
		return int64(n), true
	case json.Number:
		val, err := n.Int64()
		if err != nil {
			return 0, false
		}
		return val, true
	case string:
		s := strings.TrimSpace(n)
		if s == "" {
			return 0, false
		}
		val, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, false
		}
		return val, true
	default:
		return 0, false
	}
}

func mergedValue(current map[string]any, previous map[string]any, key string) any {
	if previous != nil {
		if value, ok := previous[key]; ok {
			return value
		}
	}
	if current == nil {
		return nil
	}
	return current[key]
}

func boolDelta(before, after bool) int64 {
	if !before && after {
		return 1
	}
	if before && !after {
		return -1
	}
	return 0
}

func isLikeActive(row map[string]any) bool {
	status, ok := getInt64(row["status"])
	return ok && status == 10
}

func isFavoriteActive(row map[string]any) bool {
	status, ok := getInt64(row["status"])
	return ok && status == 10
}

func isCommentActive(row map[string]any) bool {
	status, ok := getInt64(row["status"])
	if !ok || status != 10 {
		return false
	}
	isDeleted, ok := getInt64(row["is_deleted"])
	if !ok {
		isDeleted = 0
	}
	return isDeleted == 0
}

func isFollowActive(row map[string]any) bool {
	status, ok := getInt64(row["status"])
	if !ok || status != 10 {
		return false
	}
	isDeleted, ok := getInt64(row["is_deleted"])
	if !ok {
		isDeleted = 0
	}
	return isDeleted == 0
}
