package querynorm

import "testing"

func TestDefaultNormalizerNormalize(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		wantSearch  string
		wantCanon   string
		wantLog     string
		wantNonHash bool
	}{
		{
			name:        "trims search text and folds canonical whitespace",
			raw:         "  Growth   Notes  ",
			wantSearch:  "Growth   Notes",
			wantCanon:   "growth notes",
			wantLog:     "growth notes",
			wantNonHash: true,
		},
		{
			name:        "masks numeric query for logs",
			raw:         " 13800138000 ",
			wantSearch:  "13800138000",
			wantCanon:   "13800138000",
			wantLog:     "[numeric-query]",
			wantNonHash: true,
		},
		{
			name:       "empty query",
			raw:        "   ",
			wantSearch: "",
			wantCanon:  "",
			wantLog:    "",
		},
	}

	normalizer := NewDefaultNormalizer()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizer.Normalize(tt.raw)
			if got.SearchText != tt.wantSearch {
				t.Fatalf("SearchText = %q, want %q", got.SearchText, tt.wantSearch)
			}
			if got.Canonical != tt.wantCanon {
				t.Fatalf("Canonical = %q, want %q", got.Canonical, tt.wantCanon)
			}
			if got.LogValue != tt.wantLog {
				t.Fatalf("LogValue = %q, want %q", got.LogValue, tt.wantLog)
			}
			if tt.wantNonHash && got.Hash == "" {
				t.Fatal("Hash is empty")
			}
		})
	}
}
