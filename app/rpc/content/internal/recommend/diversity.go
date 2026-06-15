package recommend

import (
	"time"

	contentconfig "zfeed/app/rpc/content/internal/config"
)

const (
	DiversityRuleAuthorWindow = "author_window"
	DiversityRuleTypeWindow   = "type_window"
	DiversityRuleNewContent   = "new_content_quota"
)

func DiversityRerank(candidates []Candidate, cfg contentconfig.RecommendDiversityConfig) []Candidate {
	result, _ := DiversityRerankWithAdjustments(candidates, cfg)
	return result
}

func DiversityRerankWithAdjustments(
	candidates []Candidate,
	cfg contentconfig.RecommendDiversityConfig,
) ([]Candidate, map[string]int) {
	if len(candidates) <= 1 || !cfg.Enabled {
		return candidates, map[string]int{}
	}
	cfg = NormalizeConfig(contentconfig.RecommendConfig{Diversity: cfg}).Diversity

	result := make([]Candidate, 0, len(candidates))
	used := make([]bool, len(candidates))
	adjustments := map[string]int{}

	for len(result) < len(candidates) {
		pick := -1
		for i, candidate := range candidates {
			if used[i] {
				continue
			}
			violations := diversityViolations(result, candidate, cfg)
			if len(violations) > 0 {
				for _, rule := range violations {
					adjustments[rule]++
				}
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

	result, adjustments[DiversityRuleNewContent] = ensureNewContentQuota(result, cfg, time.Now())
	if adjustments[DiversityRuleNewContent] == 0 {
		delete(adjustments, DiversityRuleNewContent)
	}
	return result, adjustments
}

func violatesWindow(current []Candidate, next Candidate, cfg contentconfig.RecommendDiversityConfig) bool {
	return len(diversityViolations(current, next, cfg)) > 0
}

func diversityViolations(
	current []Candidate,
	next Candidate,
	cfg contentconfig.RecommendDiversityConfig,
) []string {
	rules := []string{}
	if cfg.MaxSameAuthor > 0 && next.AuthorID > 0 {
		if countRecentAuthors(current, next.AuthorID, cfg.AuthorWindow) >= cfg.MaxSameAuthor {
			rules = append(rules, DiversityRuleAuthorWindow)
		}
	}
	if cfg.MaxSameType > 0 && next.ContentType > 0 {
		if countRecentTypes(current, next.ContentType, cfg.TypeWindow) >= cfg.MaxSameType {
			rules = append(rules, DiversityRuleTypeWindow)
		}
	}
	return rules
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

func ensureNewContentQuota(
	candidates []Candidate,
	cfg contentconfig.RecommendDiversityConfig,
	now time.Time,
) ([]Candidate, int) {
	if len(candidates) <= 1 || cfg.NewContentTopN <= 0 || cfg.NewContentMinCount <= 0 {
		return candidates, 0
	}

	topN := cfg.NewContentTopN
	if topN > len(candidates) {
		topN = len(candidates)
	}
	current := countTopNewContents(candidates[:topN], now)
	if current >= cfg.NewContentMinCount {
		return candidates, 0
	}

	result := append([]Candidate(nil), candidates...)
	adjustments := 0
	for pos := topN; pos < len(result) && current < cfg.NewContentMinCount; pos++ {
		if !isNewContent(result[pos], now) {
			continue
		}
		target := firstOldContentPosition(result[:topN], now)
		if target < 0 {
			break
		}
		candidate := result[pos]
		copy(result[target+1:pos+1], result[target:pos])
		result[target] = candidate
		current++
		adjustments++
	}
	return result, adjustments
}

func countTopNewContents(candidates []Candidate, now time.Time) int {
	count := 0
	for _, candidate := range candidates {
		if isNewContent(candidate, now) {
			count++
		}
	}
	return count
}

func firstOldContentPosition(candidates []Candidate, now time.Time) int {
	for i, candidate := range candidates {
		if !isNewContent(candidate, now) {
			return i
		}
	}
	return -1
}

func isNewContent(candidate Candidate, now time.Time) bool {
	if candidate.SourceScores[SourceNewContent] > 0 {
		return true
	}
	if candidate.PublishedAt <= 0 {
		return false
	}
	if now.IsZero() {
		now = time.Now()
	}
	age := now.Unix() - candidate.PublishedAt
	return age >= 0 && age < 24*3600
}
