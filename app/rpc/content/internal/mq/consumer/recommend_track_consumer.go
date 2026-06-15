package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/core/logx"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"

	contentconfig "zfeed/app/rpc/content/internal/config"
	"zfeed/app/rpc/content/internal/recommend"
	"zfeed/app/rpc/content/internal/recommend/track"
	"zfeed/app/rpc/content/internal/svc"
)

const (
	recommendTrackSourceInteraction = "interaction"

	interactionEventTypeCancelLike     = "cancel_like"
	interactionEventTypeRemoveFavorite = "remove_favorite"

	interactionCommentStatusNormal int32 = 10
)

type dailyAggregator interface {
	Aggregate(ctx context.Context, event track.Event) error
}

type profileUpdater interface {
	Apply(ctx context.Context, event track.Event) error
}

type RecommendTrackConsumer struct {
	aggregator     dailyAggregator
	profileUpdater profileUpdater
	logx.Logger
}

func NewRecommendTrackConsumer(ctx context.Context, svcCtx *svc.ServiceContext) *RecommendTrackConsumer {
	var aggregator dailyAggregator
	var updater profileUpdater
	if svcCtx != nil {
		aggregator = svcCtx.RecommendDailyAggregator
		if svcCtx.Redis != nil {
			updater = redisProfileUpdater{
				rds: svcCtx.Redis,
				cfg: svcCtx.Config.Recommend,
			}
		}
	}
	return newRecommendTrackConsumerWithProfileForTest(ctx, aggregator, updater)
}

func newRecommendTrackConsumerForTest(ctx context.Context, aggregator dailyAggregator) *RecommendTrackConsumer {
	return newRecommendTrackConsumerWithProfileForTest(ctx, aggregator, nil)
}

func newRecommendTrackConsumerWithProfileForTest(
	ctx context.Context,
	aggregator dailyAggregator,
	updater profileUpdater,
) *RecommendTrackConsumer {
	return &RecommendTrackConsumer{
		aggregator:     aggregator,
		profileUpdater: updater,
		Logger:         logx.WithContext(ctx),
	}
}

func (c *RecommendTrackConsumer) Consume(ctx context.Context, _, val string) error {
	event, isUserAction, err := parseRecommendTrackEventWithMeta(val)
	if err != nil {
		recordRecommendTrackConsumeMetric(
			recommendTrackConsumeUnknownLabel,
			recommendTrackConsumeVariantControl,
			recommendTrackConsumeUnknownLabel,
			recommendTrackConsumeResultParseError,
		)
		if isUserAction {
			recordRecommendUserActionConsumeMetric(
				recommendUserActionConsumeEventUnknown,
				recommendUserActionConsumeResultParseError,
			)
		}
		logc.Errorf(ctx, "parse recommend track event failed, err=%v", err)
		return err
	}

	if c != nil && c.profileUpdater != nil {
		if err := c.profileUpdater.Apply(ctx, event); err != nil {
			recordRecommendTrackConsumeMetric(
				event.EventType,
				event.VariantID,
				event.Source,
				recommendTrackConsumeResultProfileError,
			)
			if isUserAction {
				recordRecommendUserActionConsumeMetric(
					event.EventType,
					recommendUserActionConsumeResultProfileError,
				)
			}
			logc.Errorf(ctx, "apply recommend profile event failed, event_id=%s, err=%v", event.EventID, err)
			return err
		}
	}

	if c == nil || c.aggregator == nil {
		recordRecommendTrackConsumeMetric(
			event.EventType,
			event.VariantID,
			event.Source,
			recommendTrackConsumeResultSuccess,
		)
		if isUserAction {
			recordRecommendUserActionConsumeMetric(event.EventType, recommendUserActionConsumeResultSuccess)
		}
		return nil
	}
	if err := c.aggregator.Aggregate(ctx, event); err != nil {
		recordRecommendTrackConsumeMetric(
			event.EventType,
			event.VariantID,
			event.Source,
			recommendTrackConsumeResultAggregateError,
		)
		if isUserAction {
			recordRecommendUserActionConsumeMetric(
				event.EventType,
				recommendUserActionConsumeResultAggregateError,
			)
		}
		logc.Errorf(ctx, "aggregate recommend track event failed, event_id=%s, err=%v", event.EventID, err)
		return err
	}
	recordRecommendTrackConsumeMetric(
		event.EventType,
		event.VariantID,
		event.Source,
		recommendTrackConsumeResultSuccess,
	)
	if isUserAction {
		recordRecommendUserActionConsumeMetric(event.EventType, recommendUserActionConsumeResultSuccess)
	}
	return nil
}

type recommendTrackEnvelope struct {
	track.Event
	Timestamp int64  `json:"timestamp"`
	ID        int64  `json:"id"`
	Action    string `json:"action"`
	TargetID  int64  `json:"target_id"`
	Comment   string `json:"comment"`
	Status    int32  `json:"status"`
	IsDeleted int32  `json:"is_deleted"`
	CreatedAt string `json:"created_at"`
}

func parseRecommendTrackEvent(val string) (track.Event, error) {
	event, _, err := parseRecommendTrackEventWithMeta(val)
	return event, err
}

func parseRecommendTrackEventWithMeta(val string) (track.Event, bool, error) {
	var raw recommendTrackEnvelope
	if err := json.Unmarshal([]byte(val), &raw); err != nil {
		return track.Event{}, false, err
	}

	event := raw.Event
	isUserAction := strings.TrimSpace(raw.Action) != ""
	if userActionEvent, ok := normalizeUserActionEvent(raw); ok {
		return userActionEvent, true, nil
	}
	if isUserAction {
		return track.Event{}, true, fmt.Errorf("invalid user action event")
	}

	if event.EventType != "" && (event.Source != "" || event.OccurredAt > 0) {
		return event, false, nil
	}

	if commentEvent, ok := normalizeCommentRow(raw); ok {
		return commentEvent, false, nil
	}

	switch event.EventType {
	case track.EventTypeLike:
		event.Source = recommendTrackSourceInteraction
		event.OccurredAt = interactionEventUnixSeconds(event.EventID, raw.Timestamp)
	case interactionEventTypeCancelLike:
		event.EventType = track.EventTypeUnlike
		event.Source = recommendTrackSourceInteraction
		event.OccurredAt = interactionEventUnixSeconds(event.EventID, raw.Timestamp)
	case track.EventTypeFavorite:
		event.Source = recommendTrackSourceInteraction
		event.OccurredAt = interactionEventUnixSeconds(event.EventID, raw.Timestamp)
	case interactionEventTypeRemoveFavorite:
		event.EventType = track.EventTypeUnfavorite
		event.Source = recommendTrackSourceInteraction
		event.OccurredAt = interactionEventUnixSeconds(event.EventID, raw.Timestamp)
	}

	return event, false, nil
}

func normalizeUserActionEvent(raw recommendTrackEnvelope) (track.Event, bool) {
	action := normalizeUserActionType(raw.Action)
	event := raw.Event
	if action == "" || event.EventID == "" || event.UserID <= 0 {
		return track.Event{}, false
	}

	contentID := event.ContentID
	if contentID <= 0 {
		contentID = raw.TargetID
	}
	if contentID <= 0 {
		return track.Event{}, false
	}

	event.EventType = action
	event.ContentID = contentID
	if event.Source == "" {
		event.Source = recommendTrackSourceInteraction
	}
	if event.OccurredAt <= 0 {
		event.OccurredAt = interactionEventUnixSeconds(event.EventID, raw.Timestamp)
	}
	return event, true
}

func normalizeUserActionType(action string) string {
	switch strings.TrimSpace(action) {
	case track.EventTypeLike:
		return track.EventTypeLike
	case interactionEventTypeCancelLike:
		return track.EventTypeUnlike
	case track.EventTypeFavorite:
		return track.EventTypeFavorite
	case interactionEventTypeRemoveFavorite:
		return track.EventTypeUnfavorite
	case track.EventTypeComment:
		return track.EventTypeComment
	case track.EventTypeUnlike:
		return track.EventTypeUnlike
	case track.EventTypeUnfavorite:
		return track.EventTypeUnfavorite
	default:
		return ""
	}
}

func normalizeCommentRow(raw recommendTrackEnvelope) (track.Event, bool) {
	event := raw.Event
	if event.EventType != "" ||
		raw.ID <= 0 ||
		event.UserID <= 0 ||
		event.ContentID <= 0 ||
		raw.Status != interactionCommentStatusNormal ||
		raw.IsDeleted != 0 ||
		strings.TrimSpace(raw.Comment) == "" {
		return track.Event{}, false
	}

	event.EventID = "comment_" +
		strconv.FormatInt(event.UserID, 10) +
		"_" +
		strconv.FormatInt(event.ContentID, 10) +
		"_" +
		strconv.FormatInt(raw.ID, 10)
	event.EventType = track.EventTypeComment
	event.Source = recommendTrackSourceInteraction
	event.OccurredAt = unixSecondsFromCreatedAt(raw.CreatedAt)
	return event, true
}

func interactionEventUnixSeconds(eventID string, timestamp int64) int64 {
	if timestamp <= 0 {
		timestamp = unixNanoFromEventID(eventID)
	}
	if timestamp <= 0 {
		return 0
	}
	return unixNanoToSeconds(timestamp)
}

func unixNanoFromEventID(eventID string) int64 {
	idx := strings.LastIndex(eventID, "_")
	if idx < 0 || idx == len(eventID)-1 {
		return 0
	}
	timestamp, err := strconv.ParseInt(eventID[idx+1:], 10, 64)
	if err != nil {
		return 0
	}
	return timestamp
}

func unixNanoToSeconds(timestamp int64) int64 {
	return timestamp / 1_000_000_000
}

func unixSecondsFromCreatedAt(createdAt string) int64 {
	createdAt = strings.TrimSpace(createdAt)
	if createdAt == "" {
		return 0
	}

	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999",
		time.DateTime,
	} {
		parsed, err := time.Parse(layout, createdAt)
		if err == nil {
			return parsed.Unix()
		}
	}

	return 0
}

type redisProfileUpdater struct {
	rds *gzredis.Redis
	cfg contentconfig.RecommendConfig
}

func (u redisProfileUpdater) Apply(ctx context.Context, event track.Event) error {
	action, ok := recommend.ProfileActionForTrackEvent(event.EventType, event.DwellMs)
	if !ok {
		return nil
	}

	return recommend.ApplyProfileEvent(ctx, u.rds, u.cfg, recommend.ProfileEvent{
		EventID:   event.EventID,
		EventType: action,
		UserID:    event.UserID,
		ContentID: event.ContentID,
	})
}
