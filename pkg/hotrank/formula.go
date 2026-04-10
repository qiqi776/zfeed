package hotrank

import (
	"math"
	"time"
)

const DefaultHalfLifeHours = 24.0

type Weights struct {
	Like     float64
	Comment  float64
	Favorite float64
}

func DefaultWeights() Weights {
	return Weights{
		Like:     1,
		Comment:  3,
		Favorite: 4,
	}
}

type Formula struct {
	Weights       Weights
	HalfLifeHours float64
}

func DefaultFormula() Formula {
	return Formula{
		Weights:       DefaultWeights(),
		HalfLifeHours: DefaultHalfLifeHours,
	}
}

func (f Formula) Score(likeCount, commentCount, favoriteCount int64, publishedAt, now time.Time) float64 {
	weights := normalizedWeights(f.Weights)
	weighted := float64(nonNegative(likeCount))*weights.Like +
		float64(nonNegative(commentCount))*weights.Comment +
		float64(nonNegative(favoriteCount))*weights.Favorite
	if weighted <= 0 {
		return 0
	}

	ageHours := now.Sub(publishedAt).Hours()
	if ageHours < 0 {
		ageHours = 0
	}

	decay := 1.0
	if f.HalfLifeHours > 0 {
		decay = math.Exp(-math.Ln2 * ageHours / f.HalfLifeHours)
	}

	return Round3(math.Log1p(weighted) * decay)
}

func Round3(value float64) float64 {
	return math.Round(value*1000) / 1000
}

func normalizedWeights(weights Weights) Weights {
	base := DefaultWeights()
	if weights.Like > 0 {
		base.Like = weights.Like
	}
	if weights.Comment > 0 {
		base.Comment = weights.Comment
	}
	if weights.Favorite > 0 {
		base.Favorite = weights.Favorite
	}
	return base
}

func nonNegative(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}
