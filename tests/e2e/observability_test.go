//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"testing"
	"time"
)

type registerReq struct {
	Mobile   string `json:"mobile"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Bio      string `json:"bio"`
	Gender   int32  `json:"gender"`
	Email    string `json:"email"`
	Birthday int64  `json:"birthday"`
}

type registerRes struct {
	UserID    int64  `json:"user_id"`
	Token     string `json:"token"`
	ExpiredAt int64  `json:"expired_at"`
}

type publishArticleReq struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Cover       string `json:"cover"`
	Content     string `json:"content"`
	Visibility  int32  `json:"visibility"`
}

type publishArticleRes struct {
	ContentID int64 `json:"content_id"`
}

type followUserReq struct {
	TargetUserID int64 `json:"target_user_id,string"`
}

func TestObservabilityE2E(t *testing.T) {
	env := loadE2EEnv(t)
	client := newHTTPClient()

	for _, metricsURL := range []string{
		env.FrontMetricsURL,
		env.ContentMetricsURL,
		env.InteractionMetricsURL,
		env.CountMetricsURL,
		env.UserMetricsURL,
	} {
		requireMetricsEndpoint(t, client, metricsURL)
	}

	for _, job := range []string{
		"zfeed-front",
		"zfeed-content",
		"zfeed-interaction",
		"zfeed-count",
		"zfeed-user",
	} {
		waitPromJobUp(t, client, env, job)
	}

	httpBefore := promQueryScalar(t, client, env, `sum(http_server_requests_code_total)`)
	userRPCBefore := promQueryScalar(t, client, env, `sum(rpc_server_requests_code_total{job="zfeed-user"})`)
	contentRPCBefore := promQueryScalar(t, client, env, `sum(rpc_server_requests_code_total{job="zfeed-content"})`)
	interactionRPCBefore := promQueryScalar(t, client, env, `sum(rpc_server_requests_code_total{job="zfeed-interaction"})`)
	countRPCBefore := promQueryScalar(t, client, env, `sum(rpc_server_requests_code_total{job="zfeed-count"})`)
	dbBefore := promQueryScalar(t, client, env, `sum(zfeed_db_statement_total)`)

	seed := time.Now().UnixNano()
	authorMobile := fmt.Sprintf("+861%010d", seed%10000000000)
	viewerMobile := fmt.Sprintf("+861%010d", (seed+1)%10000000000)

	authorStatus, authorBody := doJSONRequest(t, client, http.MethodPost, env.FrontAPIBaseURL+"/v1/users", registerReq{
		Mobile:   authorMobile,
		Password: "123456Aa!",
		Nickname: fmt.Sprintf("obs-author-%d", seed),
		Avatar:   "https://example.com/avatar.png",
		Bio:      "observability check",
		Gender:   1,
		Email:    fmt.Sprintf("obs-author-%d@example.com", seed),
		Birthday: 946684800,
	}, "")
	author := decodeJSONResponse[registerRes](t, authorStatus, authorBody)

	viewerStatus, viewerBody := doJSONRequest(t, client, http.MethodPost, env.FrontAPIBaseURL+"/v1/users", registerReq{
		Mobile:   viewerMobile,
		Password: "123456Aa!",
		Nickname: fmt.Sprintf("obs-viewer-%d", seed),
		Avatar:   "https://example.com/avatar.png",
		Bio:      "observability check",
		Gender:   1,
		Email:    fmt.Sprintf("obs-viewer-%d@example.com", seed),
		Birthday: 946684800,
	}, "")
	viewer := decodeJSONResponse[registerRes](t, viewerStatus, viewerBody)

	getMeStatus, getMeBody := doJSONRequest(t, client, http.MethodGet, env.FrontAPIBaseURL+"/v1/users/me", nil, author.Token)
	if getMeStatus < 200 || getMeStatus >= 300 {
		t.Fatalf("get me failed: status=%d body=%s", getMeStatus, string(getMeBody))
	}

	profileURL := fmt.Sprintf("%s/v1/user/profile/%d", env.FrontAPIBaseURL, author.UserID)
	profileStatus, profileBody := doJSONRequest(t, client, http.MethodGet, profileURL, nil, viewer.Token)
	if profileStatus < 200 || profileStatus >= 300 {
		t.Fatalf("query profile failed: status=%d body=%s", profileStatus, string(profileBody))
	}

	publishStatus, publishBody := doJSONRequest(t, client, http.MethodPost, env.FrontAPIBaseURL+"/v1/content/article/publish", publishArticleReq{
		Title:       fmt.Sprintf("obs-article-%d", seed),
		Description: "observability article",
		Cover:       "https://example.com/cover.png",
		Content:     "hello observability",
		Visibility:  10,
	}, author.Token)
	publish := decodeJSONResponse[publishArticleRes](t, publishStatus, publishBody)
	if publish.ContentID <= 0 {
		t.Fatalf("publish content_id = %d, want > 0", publish.ContentID)
	}

	followStatus, followBody := doJSONRequest(t, client, http.MethodPost, env.FrontAPIBaseURL+"/v1/interaction/followings", followUserReq{
		TargetUserID: author.UserID,
	}, viewer.Token)
	if followStatus < 200 || followStatus >= 300 {
		t.Fatalf("follow user failed: status=%d body=%s", followStatus, string(followBody))
	}

	var httpAfter float64
	var userRPCAfter float64
	var contentRPCAfter float64
	var interactionRPCAfter float64
	var countRPCAfter float64
	var dbAfter float64

	requireEventually(t, 30*time.Second, time.Second, func() error {
		httpAfter = promQueryScalar(t, client, env, `sum(http_server_requests_code_total)`)
		userRPCAfter = promQueryScalar(t, client, env, `sum(rpc_server_requests_code_total{job="zfeed-user"})`)
		contentRPCAfter = promQueryScalar(t, client, env, `sum(rpc_server_requests_code_total{job="zfeed-content"})`)
		interactionRPCAfter = promQueryScalar(t, client, env, `sum(rpc_server_requests_code_total{job="zfeed-interaction"})`)
		countRPCAfter = promQueryScalar(t, client, env, `sum(rpc_server_requests_code_total{job="zfeed-count"})`)
		dbAfter = promQueryScalar(t, client, env, `sum(zfeed_db_statement_total)`)

		switch {
		case httpAfter <= httpBefore:
			return fmt.Errorf("http_server_requests_code_total did not increase: %f -> %f", httpBefore, httpAfter)
		case userRPCAfter <= userRPCBefore:
			return fmt.Errorf("user rpc requests did not increase: %f -> %f", userRPCBefore, userRPCAfter)
		case contentRPCAfter <= contentRPCBefore:
			return fmt.Errorf("content rpc requests did not increase: %f -> %f", contentRPCBefore, contentRPCAfter)
		case interactionRPCAfter <= interactionRPCBefore:
			return fmt.Errorf("interaction rpc requests did not increase: %f -> %f", interactionRPCBefore, interactionRPCAfter)
		case countRPCAfter <= countRPCBefore:
			return fmt.Errorf("count rpc requests did not increase: %f -> %f", countRPCBefore, countRPCAfter)
		case dbAfter <= dbBefore:
			return fmt.Errorf("zfeed_db_statement_total did not increase: %f -> %f", dbBefore, dbAfter)
		default:
			return nil
		}
	})

	requireEventually(t, 20*time.Second, time.Second, func() error {
		if err := requireRegexInFiles(`/v1/users|/v1/content/article/publish|/v1/interaction/followings`, filepath.Join(env.LogRoot, "front-api", "access.log")); err != nil {
			return err
		}
		if err := requireRegexInFiles(`/user.UserService/Register|/user.UserService/GetMe|/user.UserService/GetUserProfile`, filepath.Join(env.LogRoot, "user-rpc", "access.log")); err != nil {
			return err
		}
		if err := requireRegexInFiles(`/content.ContentService/PublishArticle|/content.ContentService/BackfillFollowInbox`, filepath.Join(env.LogRoot, "content-rpc", "access.log")); err != nil {
			return err
		}
		if err := requireRegexInFiles(`/interaction.FollowService/FollowUser|/interaction.FollowService/GetFollowSummary`, filepath.Join(env.LogRoot, "interaction-rpc", "access.log")); err != nil {
			return err
		}
		for _, logPath := range []string{
			filepath.Join(env.LogRoot, "front-api", "access.log"),
			filepath.Join(env.LogRoot, "user-rpc", "access.log"),
			filepath.Join(env.LogRoot, "content-rpc", "access.log"),
			filepath.Join(env.LogRoot, "interaction-rpc", "access.log"),
			filepath.Join(env.LogRoot, "count-rpc", "access.log"),
		} {
			if err := requireRegexInFiles(`"layer":"db"`, logPath); err != nil {
				return err
			}
		}
		return nil
	})

	waitJaegerPatterns(t, client, env.JaegerBaseURL+"/api/services",
		`"front-api"`,
		`"user-rpc"`,
		`"content-rpc"`,
		`"interaction-rpc"`,
		`"count-rpc"`,
	)

	getMeTraceURL := jaegerTraceURL(env.JaegerBaseURL, "front-api", "/v1/users/me")
	waitJaegerPatterns(t, client, getMeTraceURL,
		`"/v1/users/me"`,
		`"user.UserService/GetMe"`,
		`"count.CounterService/GetUserProfileCounts"`,
		`"serviceName":"front-api"`,
		`"serviceName":"user-rpc"`,
		`"serviceName":"count-rpc"`,
	)

	followTraceURL := jaegerTraceURL(env.JaegerBaseURL, "front-api", "/v1/interaction/followings")
	waitJaegerPatterns(t, client, followTraceURL,
		`"/v1/interaction/followings"`,
		`"interaction.FollowService/FollowUser"`,
		`"content.ContentService/BackfillFollowInbox"`,
		`"serviceName":"front-api"`,
		`"serviceName":"interaction-rpc"`,
		`"serviceName":"user-rpc"`,
		`"serviceName":"content-rpc"`,
	)

	if env.EnableLogPipeline {
		requireEventually(t, 20*time.Second, time.Second, func() error {
			collectedFiles, err := logPipelineFiles(env.LogRoot)
			if err != nil {
				return err
			}
			if len(collectedFiles) == 0 {
				return fmt.Errorf("ENABLE_LOG_PIPELINE=1 but logs/collected has no ndjson output")
			}
			if err := requireRegexInFiles(`"event_kind":"http"`, filepath.Join(env.LogRoot, "collected", "front-api-*.ndjson")); err != nil {
				return err
			}
			if err := requireRegexInFiles(`"event_kind":"rpc"`, filepath.Join(env.LogRoot, "collected", "user-rpc-*.ndjson")); err != nil {
				return err
			}
			if err := requireRegexInFiles(`"event_kind":"db"`,
				filepath.Join(env.LogRoot, "collected", "front-api-*.ndjson"),
				filepath.Join(env.LogRoot, "collected", "user-rpc-*.ndjson"),
				filepath.Join(env.LogRoot, "collected", "content-rpc-*.ndjson"),
				filepath.Join(env.LogRoot, "collected", "interaction-rpc-*.ndjson"),
				filepath.Join(env.LogRoot, "collected", "count-rpc-*.ndjson"),
			); err != nil {
				return err
			}
			return nil
		})
	}
}

func jaegerTraceURL(baseURL, service, operation string) string {
	values := url.Values{}
	values.Set("service", service)
	values.Set("operation", operation)
	values.Set("limit", "5")
	values.Set("lookback", "1h")
	return baseURL + "/api/traces?" + values.Encode()
}

