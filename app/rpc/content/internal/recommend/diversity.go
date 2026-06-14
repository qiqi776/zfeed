package recommend

import contentconfig "zfeed/app/rpc/content/internal/config"

const (
	DiversityRuleAuthorWindow = "author_window"
	DiversityRuleTypeWindow   = "type_window"
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
