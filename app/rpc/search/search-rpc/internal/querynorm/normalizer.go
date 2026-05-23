package querynorm

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode"
)

const shortHashBytes = 8

type Query struct {
	Raw        string
	SearchText string
	Canonical  string
	Hash       string
	LogValue   string
}

func (q Query) Empty() bool {
	return q.SearchText == ""
}

type Normalizer interface {
	Normalize(raw string) Query
}

type DefaultNormalizer struct{}

func NewDefaultNormalizer() DefaultNormalizer {
	return DefaultNormalizer{}
}

func (DefaultNormalizer) Normalize(raw string) Query {
	searchText := strings.TrimSpace(raw)
	canonical := strings.ToLower(strings.Join(strings.Fields(searchText), " "))
	if canonical == "" {
		canonical = searchText
	}

	return Query{
		Raw:        raw,
		SearchText: searchText,
		Canonical:  canonical,
		Hash:       shortHash(canonical),
		LogValue:   maskForLog(canonical),
	}
}

func shortHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:shortHashBytes])
}

func maskForLog(value string) string {
	var digits int
	for _, r := range value {
		if unicode.IsDigit(r) {
			digits++
		}
	}
	if digits >= 7 {
		return "[numeric-query]"
	}
	return value
}
