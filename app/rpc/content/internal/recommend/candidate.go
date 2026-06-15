package recommend

import (
	"sort"
	"strings"
)

type Source string

const (
	SourceHot        Source = "hot"
	SourceNewContent Source = "new_content"
	SourceInterest   Source = "interest"
)

type Candidate struct {
	ContentID      int64
	AuthorID       int64
	ContentType    int32
	PublishedAt    int64
	Score          float64
	HotScore       float64
	InterestScore  float64
	FreshnessScore float64
	QualityScore   float64
	FinalScore     float64
	SeenCount      int
	SourceScores   map[Source]float64
	SourceRanks    map[Source]int
}

type MergeInput struct {
	Source Source
	Weight float64
	IDs    []int64
}

func PrimarySource(candidate Candidate) Source {
	var primary Source
	var bestScore float64
	for source, score := range candidate.SourceScores {
		source = normalizeSource(string(source))
		if source == "" || score <= 0 {
			continue
		}
		if primary == "" || score > bestScore || (score == bestScore && sourcePriority(source) > sourcePriority(primary)) {
			primary = source
			bestScore = score
		}
	}
	return primary
}

func Merge(inputs []MergeInput, limit int) []Candidate {
	if limit <= 0 {
		limit = defaultCandidateLimit
	}

	byID := make(map[int64]*Candidate)
	for _, input := range inputs {
		if len(input.IDs) == 0 {
			continue
		}
		weight := input.Weight
		if weight <= 0 {
			weight = 1
		}
		for rank, id := range input.IDs {
			if id <= 0 {
				continue
			}

			candidate := byID[id]
			if candidate == nil {
				candidate = &Candidate{
					ContentID:    id,
					SourceScores: make(map[Source]float64),
					SourceRanks:  make(map[Source]int),
				}
				byID[id] = candidate
			}

			sourceScore := rankScore(rank)
			if sourceScore > candidate.SourceScores[input.Source] {
				candidate.SourceScores[input.Source] = sourceScore
			}
			applySourceScore(candidate, input.Source, sourceScore)
			candidate.SourceRanks[input.Source] = minPositive(candidate.SourceRanks[input.Source], rank+1)
			candidate.Score += weight * sourceScore
		}
	}

	result := make([]Candidate, 0, len(byID))
	for _, candidate := range byID {
		result = append(result, *candidate)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Score == result[j].Score {
			return result[i].ContentID > result[j].ContentID
		}
		return result[i].Score > result[j].Score
	})
	if len(result) > limit {
		return result[:limit]
	}
	return result
}

func applySourceScore(candidate *Candidate, source Source, score float64) {
	if candidate == nil {
		return
	}

	switch source {
	case SourceHot:
		if score > candidate.HotScore {
			candidate.HotScore = score
		}
	case SourceNewContent:
		if score > candidate.FreshnessScore {
			candidate.FreshnessScore = score
		}
	case SourceInterest:
		if score > candidate.InterestScore {
			candidate.InterestScore = score
		}
	}
}

func ApplyFeatures(candidates []Candidate, features map[int64]Candidate) []Candidate {
	result := make([]Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		feature, ok := features[candidate.ContentID]
		if !ok {
			continue
		}
		candidate.AuthorID = feature.AuthorID
		candidate.ContentType = feature.ContentType
		candidate.PublishedAt = feature.PublishedAt
		result = append(result, candidate)
	}
	return result
}

func IDs(candidates []Candidate) []int64 {
	ids := make([]int64, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.ContentID <= 0 {
			continue
		}
		ids = append(ids, candidate.ContentID)
	}
	return ids
}

func rankScore(rank int) float64 {
	if rank < 0 {
		rank = 0
	}
	return 1 / float64(rank+1)
}

func minPositive(current, next int) int {
	if current <= 0 || next < current {
		return next
	}
	return current
}

func normalizeSource(value string) Source {
	switch Source(strings.TrimSpace(strings.ToLower(value))) {
	case SourceNewContent:
		return SourceNewContent
	case SourceInterest:
		return SourceInterest
	case SourceHot:
		return SourceHot
	default:
		return ""
	}
}

func sourcePriority(source Source) int {
	switch source {
	case SourceNewContent:
		return 3
	case SourceInterest:
		return 2
	case SourceHot:
		return 1
	default:
		return 0
	}
}
