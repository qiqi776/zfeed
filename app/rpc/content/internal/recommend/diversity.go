package recommend

import contentconfig "zfeed/app/rpc/content/internal/config"

func DiversityRerank(candidates []Candidate, cfg contentconfig.RecommendDiversityConfig) []Candidate {
	if len(candidates) <= 1 || !cfg.Enabled {
		return candidates
	}
	cfg = NormalizeConfig(contentconfig.RecommendConfig{Diversity: cfg}).Diversity

	result := make([]Candidate, 0, len(candidates))
	used := make([]bool, len(candidates))

	for len(result) < len(candidates) {
		pick := -1
		for i, candidate := range candidates {
			if used[i] {
				continue
			}
			if violatesWindow(result, candidate, cfg) {
				continue
			}
			pick = i
			break
		}
		if pick < 0 {
			pick = firstUnused(used)
		}
		if pick < 0 {
			break
		}

		used[pick] = true
		result = append(result, candidates[pick])
	}

	return result
}

func violatesWindow(current []Candidate, next Candidate, cfg contentconfig.RecommendDiversityConfig) bool {
	if cfg.MaxSameAuthor > 0 && next.AuthorID > 0 {
		if countRecentAuthors(current, next.AuthorID, cfg.AuthorWindow) >= cfg.MaxSameAuthor {
			return true
		}
	}
	if cfg.MaxSameType > 0 && next.ContentType > 0 {
		if countRecentTypes(current, next.ContentType, cfg.TypeWindow) >= cfg.MaxSameType {
			return true
		}
	}
	return false
}

func countRecentAuthors(candidates []Candidate, authorID int64, window int) int {
	if window <= 0 {
		return 0
	}
	start := len(candidates) - window
	if start < 0 {
		start = 0
	}

	count := 0
	for _, candidate := range candidates[start:] {
		if candidate.AuthorID == authorID {
			count++
		}
	}
	return count
}

func countRecentTypes(candidates []Candidate, contentType int32, window int) int {
	if window <= 0 {
		return 0
	}
	start := len(candidates) - window
	if start < 0 {
		start = 0
	}

	count := 0
	for _, candidate := range candidates[start:] {
		if candidate.ContentType == contentType {
			count++
		}
	}
	return count
}

func firstUnused(used []bool) int {
	for i, ok := range used {
		if !ok {
			return i
		}
	}
	return -1
}
