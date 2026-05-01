//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

type e2eEnv struct {
	RootDir             string
	FrontAPIBaseURL     string
	PrometheusBaseURL   string
	JaegerBaseURL       string
	FrontMetricsURL     string
	ContentMetricsURL   string
	InteractionMetricsURL string
	CountMetricsURL     string
	UserMetricsURL      string
	RedisAddr           string
	MySQLPort           string
	MySQLDatabase       string
	MySQLUser           string
	MySQLPassword       string
	MySQLRootPassword   string
	KafkaBrokers        []string
	LogRoot             string
	EnableLogPipeline   bool
}

type redisKeyBackup struct {
	Key    string
	Dump   string
	TTL    time.Duration
	Exists bool
}

type promQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		Result []struct {
			Value []json.RawMessage `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

func loadE2EEnv(t *testing.T) e2eEnv {
	t.Helper()

	rootDir := moduleRoot(t)
	composeEnvPath := filepath.Join(rootDir, "deploy", ".env")

	composeEnv, err := godotenv.Read(composeEnvPath)
	if err != nil {
		t.Fatalf("read compose env %s: %v", composeEnvPath, err)
	}

	merged := make(map[string]string, len(composeEnv))
	for key, value := range composeEnv {
		merged[key] = value
	}

	localEnvPath := filepath.Join(rootDir, ".env.local")
	if localEnv, err := godotenv.Read(localEnvPath); err == nil {
		for key, value := range localEnv {
			merged[key] = value
		}
	} else if !os.IsNotExist(err) {
		t.Fatalf("read local env %s: %v", localEnvPath, err)
	}

	for _, entry := range os.Environ() {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) == 2 {
			merged[parts[0]] = parts[1]
		}
	}

	frontAPIPort := envString(merged, "FRONT_API_PORT", "5000")
	prometheusHostPort := envString(merged, "PROMETHEUS_HOST_PORT", "19090")
	jaegerHostPort := envString(merged, "JAEGER_HOST_PORT", "16686")
	promPort := envString(merged, "PROM_PORT", "9290")
	contentPromPort := envString(merged, "CONTENT_PROM_PORT", "9291")
	interactionPromPort := envString(merged, "INTERACTION_PROM_PORT", "9293")
	countPromPort := envString(merged, "COUNT_PROM_PORT", "9292")
	userPromPort := envString(merged, "USER_PROM_PORT", "9294")
	redisPort := envString(merged, "REDIS_HOST_PORT", envString(merged, "REDIS_PORT", "16379"))
	mysqlPort := envString(merged, "MYSQL_PORT", envString(merged, "MYSQL_APP_PORT", "33306"))
	logPath := envString(merged, "LOG_PATH", "logs")
	if filepath.IsAbs(logPath) {
		logPath = "logs"
	}

	kafkaBrokers := splitCSV(envString(merged, "KAFKA_BROKERS", ""))
	if len(kafkaBrokers) == 0 || strings.Contains(strings.Join(kafkaBrokers, ","), "kafka:") {
		kafkaBrokers = []string{"127.0.0.1:" + envString(merged, "KAFKA_HOST_PORT", "19092")}
	}

	return e2eEnv{
		RootDir:               rootDir,
		FrontAPIBaseURL:       "http://127.0.0.1:" + frontAPIPort,
		PrometheusBaseURL:     "http://127.0.0.1:" + prometheusHostPort,
		JaegerBaseURL:         "http://127.0.0.1:" + jaegerHostPort,
		FrontMetricsURL:       "http://127.0.0.1:" + promPort + "/metrics",
		ContentMetricsURL:     "http://127.0.0.1:" + contentPromPort + "/metrics",
		InteractionMetricsURL: "http://127.0.0.1:" + interactionPromPort + "/metrics",
		CountMetricsURL:       "http://127.0.0.1:" + countPromPort + "/metrics",
		UserMetricsURL:        "http://127.0.0.1:" + userPromPort + "/metrics",
		RedisAddr:             "127.0.0.1:" + redisPort,
		MySQLPort:             mysqlPort,
		MySQLDatabase:         envString(merged, "MYSQL_DATABASE", "zfeed"),
		MySQLUser:             envString(merged, "MYSQL_USER", "zfeed"),
		MySQLPassword:         envString(merged, "MYSQL_PASSWORD", "123456"),
		MySQLRootPassword:     envString(merged, "MYSQL_ROOT_PASSWORD", "root"),
		KafkaBrokers:          kafkaBrokers,
		LogRoot:               filepath.Join(rootDir, logPath),
		EnableLogPipeline:     envBool(merged, "ENABLE_LOG_PIPELINE"),
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Dir(filepath.Dir(filename))
}

func envString(env map[string]string, key, fallback string) string {
	value := strings.TrimSpace(env[key])
	if value == "" {
		return fallback
	}
	return value
}

func envBool(env map[string]string, key string) bool {
	value := strings.TrimSpace(strings.ToLower(env[key]))
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}

func doJSONRequest(t *testing.T, client *http.Client, method, targetURL string, payload any, token string) (int, []byte) {
	t.Helper()

	var body io.Reader
	if payload != nil {
		requestBody, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal request %s %s: %v", method, targetURL, err)
		}
		body = bytes.NewReader(requestBody)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, targetURL, body)
	if err != nil {
		t.Fatalf("new request %s %s: %v", method, targetURL, err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request %s %s failed: %v", method, targetURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response %s %s: %v", method, targetURL, err)
	}
	return resp.StatusCode, respBody
}

func decodeJSONResponse[T any](t *testing.T, status int, body []byte) T {
	t.Helper()

	if status < 200 || status >= 300 {
		t.Fatalf("unexpected status %d: %s", status, string(body))
	}

	var out T
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode response body %q: %v", string(body), err)
	}
	return out
}

func requireEventually(t *testing.T, timeout, interval time.Duration, fn func() error) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		if err := fn(); err == nil {
			return
		} else {
			lastErr = err
		}
		time.Sleep(interval)
	}

	if lastErr != nil {
		t.Fatal(lastErr)
	}
	t.Fatalf("condition not met within %s", timeout)
}

func openMySQL(t *testing.T, env e2eEnv, asRoot bool) *sql.DB {
	t.Helper()

	user := env.MySQLUser
	password := env.MySQLPassword
	if asRoot {
		user = "root"
		password = env.MySQLRootPassword
	}

	dsn := fmt.Sprintf("%s:%s@tcp(127.0.0.1:%s)/%s?charset=utf8mb4&parseTime=true&multiStatements=true",
		user,
		password,
		env.MySQLPort,
		env.MySQLDatabase,
	)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping mysql: %v", err)
	}
	return db
}

func openRedis(t *testing.T, env e2eEnv) *redis.Client {
	t.Helper()

	client := redis.NewClient(&redis.Options{Addr: env.RedisAddr})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("ping redis: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func redisGetString(ctx context.Context, client *redis.Client, key string) (string, error) {
	value, err := client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return value, err
}

func backupRedisKeys(ctx context.Context, client *redis.Client, keys ...string) ([]redisKeyBackup, error) {
	backups := make([]redisKeyBackup, 0, len(keys))
	for _, key := range keys {
		exists, err := client.Exists(ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("redis exists %s: %w", key, err)
		}
		if exists == 0 {
			backups = append(backups, redisKeyBackup{Key: key})
			continue
		}

		dump, err := client.Dump(ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("redis dump %s: %w", key, err)
		}
		ttl, err := client.PTTL(ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("redis pttl %s: %w", key, err)
		}
		backups = append(backups, redisKeyBackup{
			Key:    key,
			Dump:   dump,
			TTL:    ttl,
			Exists: true,
		})
	}
	return backups, nil
}

func restoreRedisKeys(ctx context.Context, client *redis.Client, backups []redisKeyBackup) error {
	for _, backup := range backups {
		if !backup.Exists {
			if err := client.Del(ctx, backup.Key).Err(); err != nil {
				return fmt.Errorf("redis del %s: %w", backup.Key, err)
			}
			continue
		}

		restoreTTL := backup.TTL
		if restoreTTL < 0 {
			restoreTTL = 0
		}
		if err := client.RestoreReplace(ctx, backup.Key, restoreTTL, backup.Dump).Err(); err != nil {
			return fmt.Errorf("redis restore %s: %w", backup.Key, err)
		}
	}
	return nil
}

func promQueryScalar(t *testing.T, client *http.Client, env e2eEnv, query string) float64 {
	t.Helper()

	values := url.Values{}
	values.Set("query", query)
	status, body := doJSONRequest(t, client, http.MethodGet, env.PrometheusBaseURL+"/api/v1/query?"+values.Encode(), nil, "")
	if status < 200 || status >= 300 {
		t.Fatalf("prometheus query %q failed: status=%d body=%s", query, status, string(body))
	}

	var response promQueryResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("decode prometheus response: %v", err)
	}
	if len(response.Data.Result) == 0 || len(response.Data.Result[0].Value) < 2 {
		return 0
	}

	var raw string
	if err := json.Unmarshal(response.Data.Result[0].Value[1], &raw); err == nil {
		value, parseErr := strconv.ParseFloat(raw, 64)
		if parseErr != nil {
			t.Fatalf("parse prometheus scalar %q: %v", raw, parseErr)
		}
		return value
	}

	var value float64
	if err := json.Unmarshal(response.Data.Result[0].Value[1], &value); err != nil {
		t.Fatalf("decode prometheus scalar: %v", err)
	}
	return value
}

func waitPromJobUp(t *testing.T, client *http.Client, env e2eEnv, job string) {
	t.Helper()

	requireEventually(t, 60*time.Second, 2*time.Second, func() error {
		value := promQueryScalar(t, client, env, fmt.Sprintf(`sum(up{job="%s"})`, job))
		if value < 1 {
			return fmt.Errorf("prometheus target not ready: %s", job)
		}
		return nil
	})
}

func requireMetricsEndpoint(t *testing.T, client *http.Client, targetURL string) {
	t.Helper()

	status, body := doJSONRequest(t, client, http.MethodGet, targetURL, nil, "")
	if status < 200 || status >= 300 {
		t.Fatalf("metrics endpoint failed: status=%d url=%s body=%s", status, targetURL, string(body))
	}
	if len(bytes.TrimSpace(body)) == 0 {
		t.Fatalf("metrics endpoint returned empty body: %s", targetURL)
	}
}

func waitJaegerPatterns(t *testing.T, client *http.Client, targetURL string, patterns ...string) {
	t.Helper()

	requireEventually(t, 60*time.Second, 2*time.Second, func() error {
		status, body := doJSONRequest(t, client, http.MethodGet, targetURL, nil, "")
		if status < 200 || status >= 300 {
			return fmt.Errorf("jaeger request failed: status=%d body=%s", status, string(body))
		}
		for _, pattern := range patterns {
			if !strings.Contains(string(body), pattern) {
				return fmt.Errorf("jaeger response missing pattern %q from %s", pattern, targetURL)
			}
		}
		return nil
	})
}

func requireRegexInFiles(pattern string, filePatterns ...string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("compile regex %q: %w", pattern, err)
	}

	var files []string
	for _, filePattern := range filePatterns {
		if strings.ContainsAny(filePattern, "*?[") {
			matches, globErr := filepath.Glob(filePattern)
			if globErr != nil {
				return fmt.Errorf("glob %s: %w", filePattern, globErr)
			}
			files = append(files, matches...)
			continue
		}
		files = append(files, filePattern)
	}

	if len(files) == 0 {
		return fmt.Errorf("no files matched for pattern %q", strings.Join(filePatterns, ", "))
	}

	for _, filePath := range files {
		body, readErr := os.ReadFile(filePath)
		if readErr != nil {
			continue
		}
		if re.Match(body) {
			return nil
		}
	}
	return fmt.Errorf("pattern %q not found in %s", pattern, strings.Join(files, ", "))
}

func logPipelineFiles(logRoot string) ([]string, error) {
	return filepath.Glob(filepath.Join(logRoot, "collected", "*.ndjson"))
}
