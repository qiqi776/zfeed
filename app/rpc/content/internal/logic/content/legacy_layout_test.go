package contentlogic

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNoLegacyLayout(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	repoRoot := findRepoRoot(t, thisFile)
	legacyDirs := []string{
		filepath.Join(repoRoot, "app/rpc/content/internal/logic/contentservice"),
		filepath.Join(repoRoot, "app/rpc/content/internal/logic/feedservice"),
		filepath.Join(repoRoot, "app/rpc/content/internal/server/contentservice"),
		filepath.Join(repoRoot, "app/rpc/content/internal/server/feedservice"),
	}

	for _, dir := range legacyDirs {
		if _, err := os.Stat(dir); err == nil {
			t.Fatalf("legacy generated directory still exists: %s", dir)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", dir, err)
		}
	}

	contentRoot := filepath.Join(repoRoot, "app/rpc/content")
	err := filepath.WalkDir(contentRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Clean(path) == filepath.Clean(thisFile) {
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), "todo: add your logic here and delete this line") {
			t.Fatalf("generated todo stub remains in %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk content tree: %v", err)
	}
}

func findRepoRoot(t *testing.T, file string) string {
	t.Helper()

	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("repo root not found from %s", file)
		}
		dir = parent
	}
}
