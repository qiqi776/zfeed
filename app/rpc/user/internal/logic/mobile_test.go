package logic

import (
	"testing"

	"zfeed/pkg/mobilex"
)

func TestMobileNormalizationSupportsLegacyAndE164Forms(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"13800000000":    "13800000000",
		"8613800000000":  "13800000000",
		"+8613800000000": "13800000000",
	}

	for input, want := range cases {
		if got := mobilex.Normalize(input); got != want {
			t.Fatalf("Normalize(%q) = %q, want %q", input, got, want)
		}
	}
}
