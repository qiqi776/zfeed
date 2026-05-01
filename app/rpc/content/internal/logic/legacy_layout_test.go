package logic

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestContentLogicHasNoLegacyGeneratedLayout(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", ".."))
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
