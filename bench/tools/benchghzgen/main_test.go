package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateConfigsUsesFixtureData(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	configDir := filepath.Join(root, "configs")
	outputDir := filepath.Join(root, "out")
	mkdirAll(t, dataDir)
	mkdirAll(t, configDir)

	writeFile(t, filepath.Join(dataDir, "users.csv"), `user_id,mobile,password,nickname,avatar,bio,email,birthday
10001,+8619000000001,123456Aa!,bench_user_10001,avatar,bio,email,946684800
10002,+8619000000002,123456Aa!,bench_user_10002,avatar,bio,email,946684800
`)
	writeFile(t, filepath.Join(dataDir, "content_ids.csv"), `content_id,author_id,scene,title
900001,10001,ARTICLE,bench_article_900001
`)
	writeFile(t, filepath.Join(dataDir, "follow_edges.csv"), `follower_id,followee_id
10002,10001
`)
	writeFile(t, filepath.Join(dataDir, "search_terms.csv"), `query,kind
bench_article,content
`)
	writeFile(t, filepath.Join(configDir, "user-login.json"), `{
  "name": "user-login",
  "data": {"mobile": "+8619000009999", "password": "old"}
}`)
	writeFile(t, filepath.Join(configDir, "content-detail.json"), `{
  "name": "content-detail",
  "data": {"content_id": 1, "viewer_id": 2}
}`)
	writeFile(t, filepath.Join(configDir, "follow-list-followers.json"), `{
  "name": "follow-list-followers",
  "data": {"user_id": 1, "viewer_id": 2}
}`)
	writeFile(t, filepath.Join(configDir, "search-contents.json"), `{
  "name": "search-contents",
  "data": {"query": "old", "mode": "cursor", "viewer_id": 2}
}`)

	if err := generateConfigs(configDir, dataDir, outputDir); err != nil {
		t.Fatalf("generateConfigs returned error: %v", err)
	}

	login := readJSON(t, filepath.Join(outputDir, "user-login.json"))
	loginData := login["data"].(map[string]any)
	if loginData["mobile"] != "+8619000000001" || loginData["password"] != "123456Aa!" {
		t.Fatalf("login data = %+v, want first fixture user", loginData)
	}

	detail := readJSON(t, filepath.Join(outputDir, "content-detail.json"))
	detailData := detail["data"].(map[string]any)
	if detailData["content_id"] != float64(900001) || detailData["viewer_id"] != float64(10001) {
		t.Fatalf("detail data = %+v, want fixture content and viewer", detailData)
	}

	followers := readJSON(t, filepath.Join(outputDir, "follow-list-followers.json"))
	followersData := followers["data"].(map[string]any)
	if followersData["user_id"] != float64(10001) || followersData["viewer_id"] != float64(10002) {
		t.Fatalf("followers data = %+v, want follow edge followee and follower", followersData)
	}

	search := readJSON(t, filepath.Join(outputDir, "search-contents.json"))
	searchData := search["data"].(map[string]any)
	if searchData["query"] != "bench_article" || searchData["viewer_id"] != float64(10001) || searchData["mode"] != "latest" {
		t.Fatalf("search data = %+v, want fixture query, viewer, and supported latest mode", searchData)
	}
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var value map[string]any
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return value
}
