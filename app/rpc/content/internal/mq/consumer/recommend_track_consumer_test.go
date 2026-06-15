package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"

	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	contentconfig "zfeed/app/rpc/content/internal/config"
	"zfeed/app/rpc/content/internal/recommend"
	"zfeed/app/rpc/content/internal/recommend/track"
	"zfeed/app/rpc/content/internal/svc"
)

type fakeDailyAggregator struct {
	events []track.Event
	err    error
}

func (a *fakeDailyAggregator) Aggregate(_ context.Context, event track.Event) error {
	a.events = append(a.events, event)
	return a.err
}

type fakeProfileUpdater struct {
	events []track.Event
	err    error
}

func (u *fakeProfileUpdater) Apply(_ context.Context, event track.Event) error {
	u.events = append(u.events, event)
	return u.err
}

type recommendTrackConsumeMetricRecord struct {
	eventType string
	variant   string
	source    string
	result    string
}

type recommendTrackConsumeLagRecord struct {
	eventType string
	source    string
	seconds   float64
}

type recommendUserActionConsumeLagRecord struct {
	eventType string
	seconds   float64
}

func TestRecommendUserActionConsumeMetricLabelsExcludeHighCardinalityIDs(t *testing.T) {
	metricLabels := map[string][]string{
		"zfeed_recommend_user_action_consume_total":       recommendUserActionConsumeMetricLabels,
		"zfeed_recommend_user_action_consume_lag_seconds": recommendUserActionConsumeLagMetricLabels,
		"zfeed_recommend_track_consume_total":             recommendTrackConsumeMetricLabels,
		"zfeed_recommend_track_consume_lag_seconds":       recommendTrackConsumeLagMetricLabels,
	}

	for metricName, labels := range metricLabels {
		t.Run(metricName, func(t *testing.T) {
			for _, label := range labels {
				switch label {
				case "user_id", "content_id", "target_id", "event_id", "request_id", "snapshot_id":
					t.Fatalf("metric %q must not include high-cardinality label %q", metricName, label)
				}
			}
		})
	}
}

func TestRecommendTrackConsumeMetricNormalizersClampUnknownValues(t *testing.T) {
	tests := []struct {
		name string
		fn   func(string) string
		in   string
		want string
	}{
		{
			name: "event type allowed",
			fn:   normalizeRecommendTrackConsumeEventTypeLabel,
			in:   " Click ",
			want: track.EventTypeClick,
		},
		{
			name: "event type unknown",
			fn:   normalizeRecommendTrackConsumeEventTypeLabel,
			in:   "user_1001",
			want: recommendTrackConsumeUnknownLabel,
		},
		{
			name: "variant allowed",
			fn:   normalizeRecommendTrackConsumeVariantLabel,
			in:   " B ",
			want: "b",
		},
		{
			name: "variant default",
			fn:   normalizeRecommendTrackConsumeVariantLabel,
			in:   "",
			want: recommendTrackConsumeVariantControl,
		},
		{
			name: "variant unknown",
			fn:   normalizeRecommendTrackConsumeVariantLabel,
			in:   "user_1001",
			want: recommendTrackConsumeUnknownLabel,
		},
		{
			name: "source allowed",
			fn:   normalizeRecommendTrackConsumeSourceLabel,
			in:   " New-Content ",
			want: recommendTrackConsumeSourceNewContent,
		},
		{
			name: "source unknown",
			fn:   normalizeRecommendTrackConsumeSourceLabel,
			in:   "content_2001",
			want: recommendTrackConsumeUnknownLabel,
		},
		{
			name: "result allowed",
			fn:   normalizeRecommendTrackConsumeResultLabel,
			in:   " Aggregate-Error ",
			want: recommendTrackConsumeResultAggregateError,
		},
		{
			name: "result unknown",
			fn:   normalizeRecommendTrackConsumeResultLabel,
			in:   "content_2001",
			want: recommendTrackConsumeUnknownLabel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fn(tt.in); got != tt.want {
				t.Fatalf("normalizer(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestRecommendTrackConsumerRecordsTrackConsumeMetrics(t *testing.T) {
	tests := []struct {
		name       string
		aggregator *fakeDailyAggregator
		updater    *fakeProfileUpdater
		raw        string
		want       recommendTrackConsumeMetricRecord
		wantErr    bool
	}{
		{
			name:       "success",
			aggregator: &fakeDailyAggregator{},
			updater:    &fakeProfileUpdater{},
			raw: `{"event_id":"rec_click_1001_2001_1781480000","event_type":"click","user_id":1001,` +
				`"content_id":2001,"variant_id":"b","source":"recommend","occurred_at":1781480000}`,
			want: recommendTrackConsumeMetricRecord{
				eventType: track.EventTypeClick,
				variant:   "b",
				source:    recommendTrackConsumeSourceRecommend,
				result:    recommendTrackConsumeResultSuccess,
			},
		},
		{
			name:       "profile error",
			aggregator: &fakeDailyAggregator{},
			updater:    &fakeProfileUpdater{err: errors.New("redis down")},
			raw: `{"event_id":"rec_like_1001_2001_1781480001","event_type":"like","user_id":1001,` +
				`"content_id":2001,"variant_id":"a","source":"interaction","occurred_at":1781480001}`,
			want: recommendTrackConsumeMetricRecord{
				eventType: track.EventTypeLike,
				variant:   "a",
				source:    recommendTrackConsumeSourceInteraction,
				result:    recommendTrackConsumeResultProfileError,
			},
			wantErr: true,
		},
		{
			name:       "aggregate error",
			aggregator: &fakeDailyAggregator{err: errors.New("mysql down")},
			updater:    &fakeProfileUpdater{},
			raw: `{"event_id":"rec_dwell_1001_2001_1781480002","event_type":"dwell","user_id":1001,` +
				`"content_id":2001,"variant_id":"control","source":"new_content","occurred_at":1781480002}`,
			want: recommendTrackConsumeMetricRecord{
				eventType: track.EventTypeDwell,
				variant:   recommendTrackConsumeVariantControl,
				source:    recommendTrackConsumeSourceNewContent,
				result:    recommendTrackConsumeResultAggregateError,
			},
			wantErr: true,
		},
		{
			name:       "parse error",
			aggregator: &fakeDailyAggregator{},
			updater:    &fakeProfileUpdater{},
			raw:        "{",
			want: recommendTrackConsumeMetricRecord{
				eventType: recommendTrackConsumeUnknownLabel,
				variant:   recommendTrackConsumeVariantControl,
				source:    recommendTrackConsumeUnknownLabel,
				result:    recommendTrackConsumeResultParseError,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldRecord := recordRecommendTrackConsumeMetric
			defer func() {
				recordRecommendTrackConsumeMetric = oldRecord
			}()

			records := []recommendTrackConsumeMetricRecord{}
			recordRecommendTrackConsumeMetric = func(eventType, variant, source, result string) {
				records = append(records, recommendTrackConsumeMetricRecord{
					eventType: eventType,
					variant:   variant,
					source:    source,
					result:    result,
				})
			}

			consumer := newRecommendTrackConsumerWithProfileForTest(context.Background(), tt.aggregator, tt.updater)
			err := consumer.Consume(context.Background(), "", tt.raw)
			if tt.wantErr && err == nil {
				t.Fatal("Consume returned nil error, want error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Consume returned error: %v", err)
			}

			if len(records) != 1 {
				t.Fatalf("metric records = %+v, want one record", records)
			}
			if records[0] != tt.want {
				t.Fatalf("metric record = %+v, want %+v", records[0], tt.want)
			}
		})
	}
}

func TestRecommendTrackConsumerRecordsConsumeLagMetrics(t *testing.T) {
	now := time.Now()
	trackOccurredAt := now.Add(-6 * time.Second).Unix()
	userActionOccurredAt := now.Add(-4 * time.Second).Unix()

	tests := []struct {
		name              string
		raw               string
		wantTrackLag      bool
		wantUserActionLag bool
		wantTrackType     string
		wantTrackSource   string
		wantUserType      string
	}{
		{
			name: "recommend track event",
			raw: fmt.Sprintf(
				`{"event_id":"rec_click_1001_2001_%d","event_type":"click","user_id":1001,`+
					`"content_id":2001,"variant_id":"b","source":"new_content","occurred_at":%d}`,
				trackOccurredAt,
				trackOccurredAt,
			),
			wantTrackLag:    true,
			wantTrackType:   track.EventTypeClick,
			wantTrackSource: recommendTrackConsumeSourceNewContent,
		},
		{
			name: "user action event records both track and user-action lag",
			raw: fmt.Sprintf(
				`{"event_id":"ua_1001_2001_%d","action":"favorite","user_id":1001,`+
					`"target_type":"content","target_id":2001,"source":"interaction","occurred_at":%d}`,
				userActionOccurredAt,
				userActionOccurredAt,
			),
			wantTrackLag:      true,
			wantUserActionLag: true,
			wantTrackType:     track.EventTypeFavorite,
			wantTrackSource:   recommendTrackConsumeSourceInteraction,
			wantUserType:      track.EventTypeFavorite,
		},
		{
			name: "missing occurred_at skips lag metric",
			raw: `{"event_id":"rec_click_1001_2001_0","event_type":"click","user_id":1001,` +
				`"content_id":2001,"variant_id":"b","source":"recommend"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldTrackLag := observeRecommendTrackConsumeLagMetric
			oldUserActionLag := observeRecommendUserActionConsumeLagMetric
			defer func() {
				observeRecommendTrackConsumeLagMetric = oldTrackLag
				observeRecommendUserActionConsumeLagMetric = oldUserActionLag
			}()

			trackLags := []recommendTrackConsumeLagRecord{}
			userActionLags := []recommendUserActionConsumeLagRecord{}
			observeRecommendTrackConsumeLagMetric = func(eventType, source string, seconds float64) {
				trackLags = append(trackLags, recommendTrackConsumeLagRecord{
					eventType: eventType,
					source:    source,
					seconds:   seconds,
				})
			}
			observeRecommendUserActionConsumeLagMetric = func(eventType string, seconds float64) {
				userActionLags = append(userActionLags, recommendUserActionConsumeLagRecord{
					eventType: eventType,
					seconds:   seconds,
				})
			}

			consumer := newRecommendTrackConsumerWithProfileForTest(
				context.Background(),
				&fakeDailyAggregator{},
				&fakeProfileUpdater{},
			)
			if err := consumer.Consume(context.Background(), "", tt.raw); err != nil {
				t.Fatalf("Consume returned error: %v", err)
			}

			if !tt.wantTrackLag {
				if len(trackLags) != 0 {
					t.Fatalf("track lag records = %+v, want none", trackLags)
				}
			} else {
				if len(trackLags) != 1 {
					t.Fatalf("track lag records = %+v, want one record", trackLags)
				}
				got := trackLags[0]
				if got.eventType != tt.wantTrackType || got.source != tt.wantTrackSource {
					t.Fatalf("track lag record = %+v, want %s/%s", got, tt.wantTrackType, tt.wantTrackSource)
				}
				assertLagSecondsBetween(t, got.seconds, 3, 8)
			}

			if !tt.wantUserActionLag {
				if len(userActionLags) != 0 {
					t.Fatalf("user-action lag records = %+v, want none", userActionLags)
				}
				return
			}
			if len(userActionLags) != 1 {
				t.Fatalf("user-action lag records = %+v, want one record", userActionLags)
			}
			got := userActionLags[0]
			if got.eventType != tt.wantUserType {
				t.Fatalf("user-action lag record = %+v, want event type %s", got, tt.wantUserType)
			}
			assertLagSecondsBetween(t, got.seconds, 3, 8)
		})
	}
}

func assertLagSecondsBetween(t *testing.T, got, min, max float64) {
	t.Helper()

	if got < min || got > max {
		t.Fatalf("lag seconds = %.3f, want between %.3f and %.3f", got, min, max)
	}
}

func TestRecommendUserActionConsumeMetricNormalizersClampUnknownValues(t *testing.T) {
	tests := []struct {
		name string
		fn   func(string) string
		in   string
		want string
	}{
		{
			name: "event type allowed",
			fn:   normalizeRecommendUserActionConsumeEventTypeLabel,
			in:   " Favorite ",
			want: track.EventTypeFavorite,
		},
		{
			name: "event type unknown",
			fn:   normalizeRecommendUserActionConsumeEventTypeLabel,
			in:   "user_1001",
			want: recommendUserActionConsumeEventUnknown,
		},
		{
			name: "result allowed",
			fn:   normalizeRecommendUserActionConsumeResultLabel,
			in:   " Profile-Error ",
			want: recommendUserActionConsumeResultProfileError,
		},
		{
			name: "result unknown",
			fn:   normalizeRecommendUserActionConsumeResultLabel,
			in:   "content_2001",
			want: recommendUserActionConsumeEventUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fn(tt.in); got != tt.want {
				t.Fatalf("normalizer(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestRecommendTrackConsumerAggregatesTrackEvent(t *testing.T) {
	aggregator := &fakeDailyAggregator{}
	consumer := newRecommendTrackConsumerForTest(context.Background(), aggregator)

	event := track.Event{
		EventID:    "rec_click_1001_2001_1",
		EventType:  track.EventTypeClick,
		UserID:     1001,
		ContentID:  2001,
		SnapshotID: "rec:0001:b:hash:1",
		VariantID:  "b",
		Source:     "recommend",
		OccurredAt: 1781480000,
	}
	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	if err := consumer.Consume(context.Background(), "", string(raw)); err != nil {
		t.Fatalf("Consume returned error: %v", err)
	}

	if len(aggregator.events) != 1 {
		t.Fatalf("aggregated events = %+v, want one event", aggregator.events)
	}
	if aggregator.events[0] != event {
		t.Fatalf("aggregated event = %+v, want %+v", aggregator.events[0], event)
	}
}

func TestRecommendTrackConsumerNormalizesInteractionLikeEvent(t *testing.T) {
	aggregator := &fakeDailyAggregator{}
	updater := &fakeProfileUpdater{}
	consumer := newRecommendTrackConsumerWithProfileForTest(context.Background(), aggregator, updater)
	raw := `{"event_id":"like_1001_2001_1781480000000000000","event_type":"like","user_id":1001,"content_id":2001,"content_user_id":3001,"scene":"ARTICLE","timestamp":1781480000000000000}`

	if err := consumer.Consume(context.Background(), "", raw); err != nil {
		t.Fatalf("Consume returned error: %v", err)
	}

	want := track.Event{
		EventID:    "like_1001_2001_1781480000000000000",
		EventType:  track.EventTypeLike,
		UserID:     1001,
		ContentID:  2001,
		Source:     "interaction",
		OccurredAt: 1781480000,
	}
	if len(updater.events) != 1 || updater.events[0] != want {
		t.Fatalf("profile events = %+v, want [%+v]", updater.events, want)
	}
	if len(aggregator.events) != 1 || aggregator.events[0] != want {
		t.Fatalf("aggregated events = %+v, want [%+v]", aggregator.events, want)
	}
}

func TestRecommendTrackConsumerMapsCancelLikeToUnlike(t *testing.T) {
	aggregator := &fakeDailyAggregator{}
	updater := &fakeProfileUpdater{}
	consumer := newRecommendTrackConsumerWithProfileForTest(context.Background(), aggregator, updater)
	raw := `{"event_id":"cancel_like_1001_2001_1781480000000000000","event_type":"cancel_like","user_id":1001,"content_id":2001,"content_user_id":3001,"scene":"ARTICLE","timestamp":1781480000000000000}`

	if err := consumer.Consume(context.Background(), "", raw); err != nil {
		t.Fatalf("Consume returned error: %v", err)
	}

	want := track.Event{
		EventID:    "cancel_like_1001_2001_1781480000000000000",
		EventType:  track.EventTypeUnlike,
		UserID:     1001,
		ContentID:  2001,
		Source:     "interaction",
		OccurredAt: 1781480000,
	}
	if len(updater.events) != 1 || updater.events[0] != want {
		t.Fatalf("profile events = %+v, want [%+v]", updater.events, want)
	}
	if len(aggregator.events) != 1 || aggregator.events[0] != want {
		t.Fatalf("aggregated events = %+v, want [%+v]", aggregator.events, want)
	}
}

func TestRecommendTrackConsumerNormalizesFavoriteEventRow(t *testing.T) {
	aggregator := &fakeDailyAggregator{}
	updater := &fakeProfileUpdater{}
	consumer := newRecommendTrackConsumerWithProfileForTest(context.Background(), aggregator, updater)
	raw := `{"id":1,"event_id":"favorite_1001_2001_1781480000000000000","event_type":"favorite","scene":1,"user_id":1001,"content_id":2001,"content_user_id":3001}`

	if err := consumer.Consume(context.Background(), "", raw); err != nil {
		t.Fatalf("Consume returned error: %v", err)
	}

	want := track.Event{
		EventID:    "favorite_1001_2001_1781480000000000000",
		EventType:  track.EventTypeFavorite,
		UserID:     1001,
		ContentID:  2001,
		Source:     "interaction",
		OccurredAt: 1781480000,
	}
	if len(updater.events) != 1 || updater.events[0] != want {
		t.Fatalf("profile events = %+v, want [%+v]", updater.events, want)
	}
	if len(aggregator.events) != 1 || aggregator.events[0] != want {
		t.Fatalf("aggregated events = %+v, want [%+v]", aggregator.events, want)
	}
}

func TestRecommendTrackConsumerMapsRemoveFavoriteToUnfavorite(t *testing.T) {
	aggregator := &fakeDailyAggregator{}
	updater := &fakeProfileUpdater{}
	consumer := newRecommendTrackConsumerWithProfileForTest(context.Background(), aggregator, updater)
	raw := `{"id":2,"event_id":"remove_favorite_1001_2001_1781480000000000000",` +
		`"event_type":"remove_favorite","scene":1,"user_id":1001,"content_id":2001,"content_user_id":3001}`

	if err := consumer.Consume(context.Background(), "", raw); err != nil {
		t.Fatalf("Consume returned error: %v", err)
	}

	want := track.Event{
		EventID:    "remove_favorite_1001_2001_1781480000000000000",
		EventType:  track.EventTypeUnfavorite,
		UserID:     1001,
		ContentID:  2001,
		Source:     "interaction",
		OccurredAt: 1781480000,
	}
	if len(updater.events) != 1 || updater.events[0] != want {
		t.Fatalf("profile events = %+v, want [%+v]", updater.events, want)
	}
	if len(aggregator.events) != 1 || aggregator.events[0] != want {
		t.Fatalf("aggregated events = %+v, want [%+v]", aggregator.events, want)
	}
}

func TestRecommendTrackConsumerNormalizesUserActionEvent(t *testing.T) {
	aggregator := &fakeDailyAggregator{}
	updater := &fakeProfileUpdater{}
	consumer := newRecommendTrackConsumerWithProfileForTest(context.Background(), aggregator, updater)
	raw := `{"event_id":"ua_1001_2001_1781480000","action":"favorite","user_id":1001,` +
		`"target_type":"content","target_id":2001,"source":"interaction","occurred_at":1781480000}`

	if err := consumer.Consume(context.Background(), "", raw); err != nil {
		t.Fatalf("Consume returned error: %v", err)
	}

	want := track.Event{
		EventID:    "ua_1001_2001_1781480000",
		EventType:  track.EventTypeFavorite,
		UserID:     1001,
		ContentID:  2001,
		Source:     "interaction",
		OccurredAt: 1781480000,
	}
	if len(updater.events) != 1 || updater.events[0] != want {
		t.Fatalf("profile events = %+v, want [%+v]", updater.events, want)
	}
	if len(aggregator.events) != 1 || aggregator.events[0] != want {
		t.Fatalf("aggregated events = %+v, want [%+v]", aggregator.events, want)
	}
}

func TestRecommendTrackConsumerRecordsUserActionConsumeMetrics(t *testing.T) {
	tests := []struct {
		name       string
		aggregator *fakeDailyAggregator
		updater    *fakeProfileUpdater
		raw        string
		wantType   string
		wantResult string
		wantErr    bool
	}{
		{
			name:       "success",
			aggregator: &fakeDailyAggregator{},
			updater:    &fakeProfileUpdater{},
			raw: `{"event_id":"ua_1001_2001_1781480000","action":"favorite","user_id":1001,` +
				`"target_type":"content","target_id":2001,"source":"interaction","occurred_at":1781480000}`,
			wantType:   track.EventTypeFavorite,
			wantResult: recommendUserActionConsumeResultSuccess,
		},
		{
			name:       "profile error",
			aggregator: &fakeDailyAggregator{},
			updater:    &fakeProfileUpdater{err: errors.New("redis down")},
			raw: `{"event_id":"ua_1001_2001_1781480001","action":"like","user_id":1001,` +
				`"target_type":"content","target_id":2001,"source":"interaction","occurred_at":1781480001}`,
			wantType:   track.EventTypeLike,
			wantResult: recommendUserActionConsumeResultProfileError,
			wantErr:    true,
		},
		{
			name:       "aggregate error",
			aggregator: &fakeDailyAggregator{err: errors.New("mysql down")},
			updater:    &fakeProfileUpdater{},
			raw: `{"event_id":"ua_1001_2001_1781480002","action":"comment","user_id":1001,` +
				`"target_type":"content","target_id":2001,"source":"interaction","occurred_at":1781480002}`,
			wantType:   track.EventTypeComment,
			wantResult: recommendUserActionConsumeResultAggregateError,
			wantErr:    true,
		},
		{
			name:       "parse error",
			aggregator: &fakeDailyAggregator{},
			updater:    &fakeProfileUpdater{},
			raw: `{"event_id":"ua_1001_2001_1781480003","action":"share","user_id":1001,` +
				`"target_type":"content","target_id":2001,"source":"interaction","occurred_at":1781480003}`,
			wantType:   recommendUserActionConsumeEventUnknown,
			wantResult: recommendUserActionConsumeResultParseError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldRecord := recordRecommendUserActionConsumeMetric
			defer func() {
				recordRecommendUserActionConsumeMetric = oldRecord
			}()

			records := []struct {
				eventType string
				result    string
			}{}
			recordRecommendUserActionConsumeMetric = func(eventType, result string) {
				records = append(records, struct {
					eventType string
					result    string
				}{eventType: eventType, result: result})
			}

			consumer := newRecommendTrackConsumerWithProfileForTest(context.Background(), tt.aggregator, tt.updater)
			err := consumer.Consume(context.Background(), "", tt.raw)
			if tt.wantErr && err == nil {
				t.Fatal("Consume returned nil error, want error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Consume returned error: %v", err)
			}

			if len(records) != 1 {
				t.Fatalf("metric records = %+v, want one record", records)
			}
			if records[0].eventType != tt.wantType || records[0].result != tt.wantResult {
				t.Fatalf("metric record = %+v, want %s/%s", records[0], tt.wantType, tt.wantResult)
			}
		})
	}
}

func TestRecommendTrackConsumerNormalizesCommentRow(t *testing.T) {
	aggregator := &fakeDailyAggregator{}
	updater := &fakeProfileUpdater{}
	consumer := newRecommendTrackConsumerWithProfileForTest(context.Background(), aggregator, updater)
	raw := `{"id":9001,"content_id":2001,"content_user_id":3001,"user_id":1001,` +
		`"comment":"nice post","status":10,"is_deleted":0,"created_at":"1970-01-01T00:00:42Z"}`

	if err := consumer.Consume(context.Background(), "", raw); err != nil {
		t.Fatalf("Consume returned error: %v", err)
	}

	want := track.Event{
		EventID:    "comment_1001_2001_9001",
		EventType:  track.EventTypeComment,
		UserID:     1001,
		ContentID:  2001,
		Source:     "interaction",
		OccurredAt: 42,
	}
	if len(updater.events) != 1 || updater.events[0] != want {
		t.Fatalf("profile events = %+v, want [%+v]", updater.events, want)
	}
	if len(aggregator.events) != 1 || aggregator.events[0] != want {
		t.Fatalf("aggregated events = %+v, want [%+v]", aggregator.events, want)
	}
}

func TestRecommendTrackConsumerAppliesProfileEvent(t *testing.T) {
	store := miniredis.RunT(t)
	redisClient := gzredis.MustNewRedis(gzredis.RedisConf{
		Host: store.Addr(),
		Type: "node",
	})
	cfg := contentconfig.RecommendConfig{}
	contentID := int64(2001)
	if err := recommend.WriteContentTags(
		context.Background(),
		redisClient,
		cfg,
		contentID,
		map[string]float64{"go": 1},
		1,
	); err != nil {
		t.Fatalf("WriteContentTags returned error: %v", err)
	}

	consumer := NewRecommendTrackConsumer(context.Background(), &svc.ServiceContext{
		Config: contentconfig.Config{
			Recommend: cfg,
		},
		Redis: redisClient,
	})

	event := track.Event{
		EventID:    "rec_like_1001_2001_1",
		EventType:  track.EventTypeLike,
		UserID:     1001,
		ContentID:  contentID,
		Source:     "interaction",
		OccurredAt: 1781480000,
	}
	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	for range 2 {
		if err := consumer.Consume(context.Background(), "", string(raw)); err != nil {
			t.Fatalf("Consume returned error: %v", err)
		}
	}

	gotRaw := store.HGet(redisconsts.BuildRecommendUserProfileKey(1001), "go")
	got, err := strconv.ParseFloat(gotRaw, 64)
	if err != nil {
		t.Fatalf("parse profile tag: %v", err)
	}
	if got != 1 {
		t.Fatalf("profile go weight = %v, want 1 after idempotent replay", got)
	}
}

func TestRecommendTrackConsumerRejectsInvalidJSON(t *testing.T) {
	consumer := newRecommendTrackConsumerForTest(context.Background(), &fakeDailyAggregator{})

	if err := consumer.Consume(context.Background(), "", "{"); err == nil {
		t.Fatal("Consume returned nil error, want JSON error")
	}
}

func TestRecommendTrackConsumerReturnsAggregatorError(t *testing.T) {
	wantErr := errors.New("db down")
	consumer := newRecommendTrackConsumerForTest(context.Background(), &fakeDailyAggregator{err: wantErr})

	raw, err := json.Marshal(track.Event{
		EventID:    "rec_click_1001_2001_1",
		EventType:  track.EventTypeClick,
		ContentID:  2001,
		OccurredAt: 1781480000,
	})
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	if err := consumer.Consume(context.Background(), "", string(raw)); !errors.Is(err, wantErr) {
		t.Fatalf("Consume error = %v, want %v", err, wantErr)
	}
}
