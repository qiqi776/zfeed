package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type fixtureData struct {
	Users       []csvRow
	Contents    []csvRow
	FollowEdges []csvRow
	SearchTerms []csvRow
}

type csvRow map[string]string

func main() {
	configDir := flag.String("configs", "bench/ghz", "source ghz config directory")
	dataDir := flag.String("data", "bench/data/small", "benchmark fixture directory")
	outputDir := flag.String("output", "", "output directory for generated ghz configs")
	flag.Parse()

	if *outputDir == "" {
		fmt.Fprintln(os.Stderr, "用法：benchghzgen --configs bench/ghz --data bench/data/small --output <dir>")
		os.Exit(2)
	}

	if err := generateConfigs(*configDir, *dataDir, *outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "生成 ghz 配置失败：%v\n", err)
		os.Exit(1)
	}
	fmt.Printf("ghz 配置已生成：%s\n", *outputDir)
}

func generateConfigs(configDir string, dataDir string, outputDir string) error {
	fixture, err := loadFixtureData(dataDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}

	return filepath.WalkDir(configDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}

		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var config map[string]any
		if err := json.Unmarshal(body, &config); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}

		name := strings.TrimSuffix(filepath.Base(path), ".json")
		data, ok := config["data"].(map[string]any)
		if !ok {
			data = map[string]any{}
		}
		applyFixture(name, data, fixture)
		config["data"] = data

		generated, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return err
		}
		outputPath := filepath.Join(outputDir, filepath.Base(path))
		return os.WriteFile(outputPath, append(generated, '\n'), 0o644)
	})
}

func loadFixtureData(dataDir string) (fixtureData, error) {
	users, err := readCSVRows(filepath.Join(dataDir, "users.csv"))
	if err != nil {
		return fixtureData{}, err
	}
	contents, err := readCSVRows(filepath.Join(dataDir, "content_ids.csv"))
	if err != nil {
		return fixtureData{}, err
	}
	followEdges, err := readCSVRows(filepath.Join(dataDir, "follow_edges.csv"))
	if err != nil {
		return fixtureData{}, err
	}
	searchTerms, err := readCSVRows(filepath.Join(dataDir, "search_terms.csv"))
	if err != nil {
		return fixtureData{}, err
	}

	return fixtureData{
		Users:       users,
		Contents:    contents,
		FollowEdges: followEdges,
		SearchTerms: searchTerms,
	}, nil
}

func readCSVRows(path string) ([]csvRow, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	headers, err := reader.Read()
	if err != nil {
		return nil, err
	}
	for index := range headers {
		headers[index] = strings.TrimSpace(headers[index])
	}

	rows := []csvRow{}
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		row := csvRow{}
		for index, header := range headers {
			if index < len(record) {
				row[header] = strings.TrimSpace(record[index])
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func applyFixture(name string, data map[string]any, fixture fixtureData) {
	user := firstRow(fixture.Users)
	secondUser := nthRow(fixture.Users, 1)
	content := firstRow(fixture.Contents)
	edge := firstRow(fixture.FollowEdges)
	term := firstRow(fixture.SearchTerms)

	userID := rowInt(user, "user_id")
	viewerID := rowInt(secondUser, "user_id")
	if viewerID == 0 {
		viewerID = userID
	}
	contentID := rowInt(content, "content_id")
	authorID := rowInt(content, "author_id")
	if authorID == 0 {
		authorID = userID
	}
	followerID := rowInt(edge, "follower_id")
	followeeID := rowInt(edge, "followee_id")
	if followerID == 0 {
		followerID = viewerID
	}
	if followeeID == 0 {
		followeeID = authorID
	}

	switch {
	case name == "user-login":
		data["mobile"] = user["mobile"]
		data["password"] = user["password"]
	case name == "user-get-me":
		data["user_id"] = userID
	case strings.HasPrefix(name, "feed-"):
		data["user_id"] = userID
	case strings.HasPrefix(name, "content-detail"):
		data["content_id"] = contentID
		data["viewer_id"] = userID
	case strings.HasPrefix(name, "content-publish"):
		data["user_id"] = userID
		if title := content["title"]; title != "" {
			data["title"] = title + "_ghz"
		}
	case strings.HasPrefix(name, "like-"):
		data["user_id"] = viewerID
		data["content_id"] = contentID
		data["content_user_id"] = authorID
		data["scene"] = content["scene"]
	case strings.HasPrefix(name, "comment-"):
		data["content_id"] = contentID
		data["scene"] = content["scene"]
	case strings.HasPrefix(name, "count-"):
		data["keys"] = countKeys(contentID)
	case strings.HasPrefix(name, "follow-"):
		data["user_id"] = followeeID
		data["viewer_id"] = followerID
	case strings.HasPrefix(name, "search-"):
		if query := term["query"]; query != "" {
			data["query"] = query
		}
		data["viewer_id"] = userID
	}
}

func countKeys(contentID int64) []map[string]any {
	return []map[string]any{
		{"biz_type": "LIKE", "target_type": "CONTENT", "target_id": contentID},
		{"biz_type": "COMMENT", "target_type": "CONTENT", "target_id": contentID},
		{"biz_type": "FAVORITE", "target_type": "CONTENT", "target_id": contentID},
	}
}

func firstRow(rows []csvRow) csvRow {
	return nthRow(rows, 0)
}

func nthRow(rows []csvRow, index int) csvRow {
	if index < 0 || index >= len(rows) {
		return csvRow{}
	}
	return rows[index]
}

func rowInt(row csvRow, key string) int64 {
	value := strings.TrimSpace(row[key])
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}
