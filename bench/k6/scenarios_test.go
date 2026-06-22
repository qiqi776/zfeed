package k6

import (
	"os"
	"strings"
	"testing"
)

func TestSmokeScenarioCoversFavorite(t *testing.T) {
	workload := readText(t, "lib/workload.js")
	smokeBody := functionBody(t, workload, "runSmoke")

	for _, want := range []string{
		"favoriteContent(state);",
		"queryFavoriteInfo(state);",
	} {
		if !strings.Contains(smokeBody, want) {
			t.Fatalf("runSmoke missing %q:\n%s", want, smokeBody)
		}
	}

	config := readText(t, "config.js")
	if !strings.Contains(config, "\"http_req_duration{name:interaction_favorite}\"") {
		t.Fatalf("config.js missing interaction_favorite threshold")
	}
}

func readText(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(body)
}

func functionBody(t *testing.T, source, name string) string {
	t.Helper()
	start := strings.Index(source, "function "+name+"(")
	if start < 0 {
		t.Fatalf("function %s not found", name)
	}

	open := strings.Index(source[start:], "{")
	if open < 0 {
		t.Fatalf("function %s has no body", name)
	}
	open += start

	depth := 0
	for i := open; i < len(source); i++ {
		switch source[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return source[open+1 : i]
			}
		}
	}

	t.Fatalf("function %s body is not closed", name)
	return ""
}
