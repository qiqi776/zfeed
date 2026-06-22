package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"zfeed/pkg/utils"
)

func TestBuildSeedSQLUsesFixtureRowsAndLoginPassword(t *testing.T) {
	dataDir := t.TempDir()
	writeFile(t, filepath.Join(dataDir, "users.csv"), `user_id,mobile,password,nickname,avatar,bio,email,birthday
10001,+8610000010001,123456Aa!,bench_user_10001,https://example.com/a.png,bench user 10001,bench_user_10001@example.com,946684800
10002,+8610000010002,123456Aa!,bench_user_10002,https://example.com/b.png,bench user 10002,bench_user_10002@example.com,946684800
`)
	writeFile(t, filepath.Join(dataDir, "content_ids.csv"), `content_id,author_id,scene,title
900001,10001,ARTICLE,bench_article_900001
900002,10002,VIDEO,bench_video_900002
`)
	writeFile(t, filepath.Join(dataDir, "follow_edges.csv"), `follower_id,followee_id
10002,10001
10002,10001
`)
	writeFile(t, filepath.Join(dataDir, "search_terms.csv"), `query,kind
bench,common
`)

	sql, err := buildSeedSQL(dataDir)
	if err != nil {
		t.Fatalf("buildSeedSQL returned error: %v", err)
	}

	for _, want := range []string{
		"START TRANSACTION;",
		"INSERT INTO `zfeed_user`",
		"10001, '10000010001'",
		"'bench_user_10001'",
		"INSERT INTO `zfeed_content`",
		"900001, 10001, 10, 30, 10",
		"900002, 10002, 20, 30, 10",
		"INSERT INTO `zfeed_article`",
		"900001, 'bench_article_900001'",
		"INSERT INTO `zfeed_video`",
		"900002, 'bench_video_900002'",
		"INSERT INTO `zfeed_follow`",
		"10002, 10001, 10",
		"`version` = VALUES(`version`)",
		"COMMIT;",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("seed SQL missing %q:\n%s", want, sql)
		}
	}

	if !utils.CheckPassword(benchPasswordHash, "123456Aa!"+benchPasswordSalt) {
		t.Fatalf("bench password hash does not match default fixture password and salt")
	}

	if got := strings.Count(sql, "(10002, 10001, 10, 1, 0, 10002, 10002)"); got != 1 {
		t.Fatalf("seed SQL should deduplicate repeated follow edges, got %d copies:\n%s", got, sql)
	}
}

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
