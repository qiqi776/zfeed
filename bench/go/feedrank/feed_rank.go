package feedrank

import "sort"

type candidate struct {
	contentID      int64
	score          float64
	hotScore       float64
	interestScore  float64
	freshnessScore float64
	qualityScore   float64
	seenCount      int
	sourceScores   map[string]float64
	sourceRanks    map[string]int
}

type recallInput struct {
	source     string
	weight     float64
	candidates []candidate
}

type candidateFeature struct {
	freshnessScore float64
	qualityScore   float64
}

type rankConfig struct {
	limit              int
	hotWeight          float64
	interestWeight     float64
	freshnessWeight    float64
	qualityWeight      float64
	seenPenalty        float64
	repeatedSeenFilter int
}

func rankFeedCandidates(
	inputs []recallInput,
	features map[int64]candidateFeature,
	seenCounts map[int64]int,
	cfg rankConfig,
) []candidate {
	merged := mergeCandidates(inputs, cfg.limit)
	applyFeatures(merged, features)
	applySeenCounts(merged, seenCounts)

	ranked := make([]candidate, 0, len(merged))
	for _, item := range merged {
		if item.contentID <= 0 {
			continue
		}
		if cfg.repeatedSeenFilter > 0 && item.seenCount >= cfg.repeatedSeenFilter {
			continue
		}
		item.score = item.hotScore*cfg.hotWeight +
			item.interestScore*cfg.interestWeight +
			item.freshnessScore*cfg.freshnessWeight +
			item.qualityScore*cfg.qualityWeight -
			float64(item.seenCount)*cfg.seenPenalty
		ranked = append(ranked, item)
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score == ranked[j].score {
			return ranked[i].contentID > ranked[j].contentID
		}
		return ranked[i].score > ranked[j].score
	})
	if cfg.limit > 0 && len(ranked) > cfg.limit {
		return ranked[:cfg.limit]
	}
	return ranked
}

func mergeCandidates(inputs []recallInput, limit int) []candidate {
	byID := make(map[int64]*candidate)
	for _, input := range inputs {
		source := input.source
		weight := input.weight
		if weight <= 0 {
			weight = 1
		}
		for rank, item := range input.candidates {
			if item.contentID <= 0 {
				continue
			}
			current := byID[item.contentID]
			if current == nil {
				current = &candidate{
					contentID:    item.contentID,
					sourceScores: map[string]float64{},
					sourceRanks:  map[string]int{},
				}
				byID[item.contentID] = current
			}

			score := sourceScore(item, source, rank)
			if score > current.sourceScores[source] {
				current.sourceScores[source] = score
			}
			current.sourceRanks[source] = minPositive(current.sourceRanks[source], sourceRank(item, source, rank))
			current.score += weight * score
			applySourceScore(current, source, score)
		}
	}

	merged := make([]candidate, 0, len(byID))
	for _, item := range byID {
		merged = append(merged, *item)
	}
	sort.SliceStable(merged, func(i, j int) bool {
		if merged[i].score == merged[j].score {
			return merged[i].contentID > merged[j].contentID
		}
		return merged[i].score > merged[j].score
	})
	if limit > 0 && len(merged) > limit {
		return merged[:limit]
	}
	return merged
}

func applyFeatures(candidates []candidate, features map[int64]candidateFeature) {
	for i := range candidates {
		feature := features[candidates[i].contentID]
		candidates[i].freshnessScore = feature.freshnessScore
		candidates[i].qualityScore = feature.qualityScore
	}
}

func applySeenCounts(candidates []candidate, seenCounts map[int64]int) {
	for i := range candidates {
		candidates[i].seenCount = seenCounts[candidates[i].contentID]
	}
}

func sourceScore(item candidate, source string, rank int) float64 {
	if item.sourceScores != nil {
		if score := item.sourceScores[source]; score > 0 {
			return score
		}
	}
	return rankScore(rank)
}

func sourceRank(item candidate, source string, rank int) int {
	if item.sourceRanks != nil {
		if sourceRank := item.sourceRanks[source]; sourceRank > 0 {
			return sourceRank
		}
	}
	return rank + 1
}

func applySourceScore(item *candidate, source string, score float64) {
	switch source {
	case "hot":
		if score > item.hotScore {
			item.hotScore = score
		}
	case "interest":
		if score > item.interestScore {
			item.interestScore = score
		}
	}
}

func candidateIDs(candidates []candidate) []int64 {
	ids := make([]int64, 0, len(candidates))
	for _, item := range candidates {
		if item.contentID <= 0 {
			continue
		}
		ids = append(ids, item.contentID)
	}
	return ids
}

func buildFixture(size int, overlap int) ([]recallInput, map[int64]candidateFeature, map[int64]int) {
	if overlap < 0 {
		overlap = 0
	}
	if overlap > size {
		overlap = size
	}

	hot := make([]candidate, 0, size)
	interest := make([]candidate, 0, size)
	features := make(map[int64]candidateFeature, size*2-overlap)
	seenCounts := make(map[int64]int, size/4)

	for i := 0; i < size; i++ {
		hotID := int64(900000 + i)
		hot = append(hot, candidate{
			contentID: hotID,
			sourceScores: map[string]float64{
				"hot": rankScore(i),
			},
			sourceRanks: map[string]int{
				"hot": i + 1,
			},
		})
		features[hotID] = candidateFeature{
			freshnessScore: float64((i%17)+1) / 17,
			qualityScore:   float64((i%11)+1) / 11,
		}
		if i%9 == 0 {
			seenCounts[hotID] = 3
		} else if i%5 == 0 {
			seenCounts[hotID] = 1
		}
	}

	for i := 0; i < size; i++ {
		contentID := int64(950000 + i)
		if i < overlap {
			contentID = int64(900000 + i*2)
		}
		interest = append(interest, candidate{
			contentID: contentID,
			sourceScores: map[string]float64{
				"interest": rankScore(i),
			},
			sourceRanks: map[string]int{
				"interest": i + 1,
			},
		})
		features[contentID] = candidateFeature{
			freshnessScore: float64((i%13)+1) / 13,
			qualityScore:   float64((i%7)+1) / 7,
		}
		if i%12 == 0 {
			seenCounts[contentID] = 2
		}
	}

	return []recallInput{
		{source: "hot", weight: 0.55, candidates: hot},
		{source: "interest", weight: 0.45, candidates: interest},
	}, features, seenCounts
}

func minPositive(current int, next int) int {
	if current <= 0 {
		return next
	}
	if next <= 0 || current < next {
		return current
	}
	return next
}

func rankScore(rank int) float64 {
	if rank < 0 {
		rank = 0
	}
	return 1 / float64(rank+1)
}
