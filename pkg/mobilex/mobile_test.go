package mobilex

import (
	"reflect"
	"testing"
)

func TestNormalize(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"13800000000":    "13800000000",
		"8613800000000":  "13800000000",
		"+8613800000000": "13800000000",
		"+15551234567":   "+15551234567",
	}

	for input, want := range cases {
		if got := Normalize(input); got != want {
			t.Fatalf("Normalize(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestIsValid(t *testing.T) {
	t.Parallel()

	valid := []string{"13800000000", "8613800000000", "+8613800000000", "+15551234567"}
	for _, input := range valid {
		if !IsValid(input) {
			t.Fatalf("expected %q to be valid", input)
		}
	}

	invalid := []string{"", "abc", "+86123", "0038613800000000"}
	for _, input := range invalid {
		if IsValid(input) {
			t.Fatalf("expected %q to be invalid", input)
		}
	}
}

func TestLookupCandidates(t *testing.T) {
	t.Parallel()

	got := LookupCandidates("+8613800000000")
	want := []string{"+8613800000000", "13800000000", "8613800000000"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LookupCandidates returned %v, want %v", got, want)
	}
}
