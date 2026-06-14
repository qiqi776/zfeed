package feedlogic

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	contentpb "zfeed/app/rpc/content/content"
	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	luautils "zfeed/app/rpc/content/internal/common/utils/lua"
	contentconfig "zfeed/app/rpc/content/internal/config"
	"zfeed/app/rpc/content/internal/recommend"
	"zfeed/app/rpc/content/internal/recommend/track"
	"zfeed/app/rpc/content/internal/svc"
	"zfeed/pkg/errorx"
)

type CacheResult int

const (
	cacheHit CacheResult = iota
	cacheMiss
	cacheError
)

type hotFeedResult struct {
	ids                []int64
	nextCursor         int64
	hasMore            bool
	resolvedSnapshotID string
}

type personalizedSnapshotResult struct {
	resp *contentpb.RecommendFeedRes
	meta recommend.SnapshotMeta
}

type recommendRuntime struct {
	cfg        contentconfig.RecommendConfig
	variantID  string
	configHash string
}

type RecommendFeedLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	itemBuilder *FeedItemBuilder
}

func NewRecommendFeedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RecommendFeedLogic {
	return &RecommendFeedLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		itemBuilder: NewFeedItemBuilder(ctx, svcCtx),
	}
}

func (l *RecommendFeedLogic) RecommendFeed(in *contentpb.RecommendFeedReq) (*contentpb.RecommendFeedRes, error) {
	if in == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}

	pageSize := int(in.GetPageSize())
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}

	totalStarted := time.Now()
	defer func() {
		recordRecommendStageDurationMetric(
			recommendStageTotal,
			recommendVariantControl,
			time.Since(totalStarted),
		)
	}()

	if snapshotID := strings.TrimSpace(in.GetSnapshotId()); recommend.IsPersonalizedSnapshot(snapshotID) {
		snapshotLookupStarted := time.Now()
		snapshotResult, ok, err := l.recommendFromPersonalizedSnapshot(in, pageSize, snapshotID)
		variantID := recommendVariantControl
		if ok {
			variantID = strings.TrimSpace(snapshotResult.meta.VariantID)
			if variantID == "" {
				variantID = recommendVariantControl
			}
		}
		recordRecommendStageDurationMetric(
			recommendStageSnapshotLookup,
			variantID,
			time.Since(snapshotLookupStarted),
		)
		if err == nil && ok {
			recordRecommendSnapshotMetric(recommendSnapshotKindPersonalized, recommendSnapshotResultHit)
			result := recommendResultSuccess
			if snapshotResult.resp == nil || len(snapshotResult.resp.GetItems()) == 0 {
				result = recommendResultEmpty
			}
			recordRecommendRequestMetric(recommendModeSnapshot, variantID, result)
			l.emitExposureTrackEvents(in.GetUserId(), snapshotResult.resp, variantID)
			return snapshotResult.resp, nil
		}
		if err != nil {
			recordRecommendSnapshotMetric(recommendSnapshotKindPersonalized, recommendSnapshotResultError)
			recordRecommendFallbackMetric(recommendFallbackReasonSnapshotError)
			l.Errorf("query personalized recommend snapshot failed, snapshot_id=%s, err=%v", snapshotID, err)
		} else {
			recordRecommendSnapshotMetric(recommendSnapshotKindPersonalized, recommendSnapshotResultMiss)
			recordRecommendFallbackMetric(recommendFallbackReasonSnapshotMiss)
		}
	}

	runtime, cfgErr := l.loadRecommendRuntime(in.GetUserId())
	if cfgErr != nil {
		l.Errorf("load recommend runtime config failed, user_id=%d, err=%v", in.GetUserId(), cfgErr)
	}
	if cfgErr == nil {
		scoped, cancel := l.withRecommendTimeout(runtime.cfg)
		defer cancel()

		if !runtime.cfg.Enabled {
			recordRecommendFallbackMetric(recommendFallbackReasonDisabled)
		} else if scoped.shouldUseRecommendEnhancement(in, runtime.cfg) {
			resp, err := scoped.recommendWithNewContent(in, pageSize, runtime)
			if err == nil && len(resp.GetItems()) > 0 {
				recordRecommendRequestMetric(
					recommendModePersonalized,
					runtime.variantID,
					recommendResultSuccess,
				)
				l.emitExposureTrackEvents(in.GetUserId(), resp, runtime.variantID)
				return resp, nil
			}
			if err != nil {
				recordRecommendFallbackMetric(recommendFallbackReasonEnhancementError)
				l.Errorf("recommend enhancement failed, user_id=%d, err=%v", in.GetUserId(), err)
			}
		}
	}

	resp, err := l.recommendHotFallback(in, pageSize)
	if err != nil {
		recordRecommendRequestMetric(recommendModeHot, recommendVariantControl, recommendResultError)
		return nil, err
	}
	if len(resp.GetItems()) == 0 {
		recordRecommendRequestMetric(recommendModeHot, recommendVariantControl, recommendResultEmpty)
		l.emitExposureTrackEvents(in.GetUserId(), resp, recommendVariantControl)
		return resp, nil
	}
	recordRecommendRequestMetric(recommendModeHot, recommendVariantControl, recommendResultSuccess)
	l.emitExposureTrackEvents(in.GetUserId(), resp, recommendVariantControl)
	return resp, nil
}

func (l *RecommendFeedLogic) loadRecommendRuntime(userID int64) (recommendRuntime, error) {
	if l == nil || l.svcCtx == nil {
		cfg := recommend.NormalizeConfig(contentconfig.RecommendConfig{})
		return buildRecommendRuntime(cfg, userID), nil
	}
	cfg, err := recommend.LoadRuntimeConfig(l.ctx, l.svcCtx.Redis, l.svcCtx.Config.Recommend)
	if err != nil {
		return recommendRuntime{}, err
	}
	return buildRecommendRuntime(cfg, userID), nil
}

func buildRecommendRuntime(cfg contentconfig.RecommendConfig, userID int64) recommendRuntime {
	cfg = recommend.NormalizeConfig(cfg)
	variant := recommend.ResolveExperimentVariant(
		userID,
		recommend.ExperimentConfigFromContent(cfg.Experiment),
	)
	variantID := strings.TrimSpace(variant.ID)
	if variantID == "" {
		variantID = recommendVariantControl
	}
	cfg = recommend.ApplyExperimentVariantOverrides(cfg, variant)

	return recommendRuntime{
		cfg:        cfg,
		variantID:  variantID,
		configHash: recommend.ConfigHash(cfg),
	}
}

func (l *RecommendFeedLogic) withRecommendTimeout(
	cfg contentconfig.RecommendConfig,
) (*RecommendFeedLogic, context.CancelFunc) {
	if l == nil {
		return l, func() {}
	}

	timeout := recommendTimeoutBudget(cfg)
	if timeout <= 0 {
		return l, func() {}
	}

	baseCtx := l.ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	ctx, cancel := context.WithTimeout(baseCtx, timeout)

	scoped := *l
	scoped.ctx = ctx
	scoped.Logger = logx.WithContext(ctx)
	scoped.itemBuilder = NewFeedItemBuilder(ctx, l.svcCtx)
	return &scoped, cancel
}

func recommendTimeoutBudget(cfg contentconfig.RecommendConfig) time.Duration {
	cfg = recommend.NormalizeConfig(cfg)
	return time.Duration(cfg.TimeoutMs) * time.Millisecond
}

func (l *RecommendFeedLogic) recommendHotFallback(in *contentpb.RecommendFeedReq, pageSize int) (*contentpb.RecommendFeedRes, error) {
	preferredKey, preferredSnapshotID := l.resolveSnapshotKey(in.SnapshotId)
	result, err := l.queryHotIDsByCursor(preferredKey, preferredSnapshotID, strings.TrimSpace(in.GetCursor()), pageSize)
	if err != nil {
		recordRecommendFallbackMetric(recommendFallbackReasonHotError)
		return nil, err
	}
	resp, err := l.buildFeedResponse(in, result)
	if err != nil {
		recordRecommendFallbackMetric(recommendFallbackReasonBuildError)
		return nil, err
	}
	if resp == nil || len(resp.GetItems()) == 0 {
		recordRecommendFallbackMetric(recommendFallbackReasonColdStart)
	}
	return resp, nil
}

func (l *RecommendFeedLogic) recommendFromPersonalizedSnapshot(
	in *contentpb.RecommendFeedReq,
	pageSize int,
	snapshotID string,
) (personalizedSnapshotResult, bool, error) {
	snapshotKey := redisconsts.BuildRecommendUserSnapshotKey(snapshotID)
	exists, err := l.svcCtx.Redis.ExistsCtx(l.ctx, snapshotKey)
	if err != nil {
		return personalizedSnapshotResult{}, false, err
	}
	if !exists {
		return personalizedSnapshotResult{}, false, nil
	}

	meta, _, err := recommend.LoadPersonalizedSnapshotMeta(l.ctx, l.svcCtx.Redis, snapshotID)
	if err != nil {
		return personalizedSnapshotResult{}, true, err
	}
	result, err := l.queryHotIDsByCursor(snapshotKey, snapshotID, strings.TrimSpace(in.GetCursor()), pageSize)
	if err != nil {
		return personalizedSnapshotResult{}, true, err
	}
	resp, err := l.buildFeedResponse(in, result)
	if err != nil {
		return personalizedSnapshotResult{}, true, err
	}
	return personalizedSnapshotResult{
		resp: resp,
		meta: meta,
	}, true, nil
}

func (l *RecommendFeedLogic) buildFeedResponse(
	in *contentpb.RecommendFeedReq,
	result *hotFeedResult,
) (*contentpb.RecommendFeedRes, error) {
	if result == nil {
		result = &hotFeedResult{}
	}
	if len(result.ids) == 0 {
		return &contentpb.RecommendFeedRes{
			Items:      []*contentpb.ContentItem{},
			NextCursor: 0,
			HasMore:    false,
			SnapshotId: result.resolvedSnapshotID,
		}, nil
	}

	contents, err := l.itemBuilder.LoadContentsByIDs(result.ids)
	if err != nil {
		return nil, err
	}
	if len(contents) == 0 {
		return &contentpb.RecommendFeedRes{
			Items:      []*contentpb.ContentItem{},
			NextCursor: 0,
			HasMore:    false,
			SnapshotId: result.resolvedSnapshotID,
		}, nil
	}

	items, err := l.itemBuilder.BuildContentItems(contents, in.UserId)
	if err != nil {
		return nil, err
	}

	return &contentpb.RecommendFeedRes{
		Items:      items,
		NextCursor: result.nextCursor,
		HasMore:    result.hasMore,
		SnapshotId: result.resolvedSnapshotID,
	}, nil
}

func (l *RecommendFeedLogic) shouldUseRecommendEnhancement(
	in *contentpb.RecommendFeedReq,
	cfg contentconfig.RecommendConfig,
) bool {
	if l == nil || l.svcCtx == nil || l.svcCtx.Redis == nil || l.svcCtx.MysqlDb == nil || in == nil {
		return false
	}

	hasNewRecall := cfg.NewContent.Enabled
	hasInterestRecall := cfg.Interest.Enabled && in.GetUserId() > 0
	if !cfg.Enabled || (!hasNewRecall && !hasInterestRecall) {
		return false
	}

	cursor := strings.TrimSpace(in.GetCursor())
	if cursor != "" && cursor != "0" {
		return false
	}
	if in.SnapshotId != nil && strings.TrimSpace(in.GetSnapshotId()) != "" {
		return false
	}
	return true
}

func (l *RecommendFeedLogic) recommendWithNewContent(
	in *contentpb.RecommendFeedReq,
	pageSize int,
	runtime recommendRuntime,
) (*contentpb.RecommendFeedRes, error) {
	cfg := runtime.cfg
	var hotSnapshotID string
	recallStarted := time.Now()
	cacheKey := recommend.BuildCandidateCacheKey(in.GetUserId(), runtime.variantID, runtime.configHash)
	merged, cached, err := recommend.LoadCandidateCache(
		l.ctx,
		l.svcCtx.Redis,
		cacheKey,
		cfg.CandidateLimit,
	)
	if err != nil {
		recordRecommendErrorMetric(recommendErrorStageCandidateCache, runtime.variantID)
		return nil, err
	}
	if !cached {
		var inputs []recommend.MergeInput
		inputs, hotSnapshotID, err = l.recallRecommendCandidates(in, pageSize, cfg, runtime.variantID)
		if err != nil {
			recordRecommendErrorMetric(recommendErrorStageRecall, runtime.variantID)
			return nil, err
		}
		merged = recommend.Merge(inputs, cfg.CandidateLimit)
		if err := recommend.SaveCandidateCache(l.ctx, l.svcCtx.Redis, cfg, cacheKey, merged); err != nil {
			recordRecommendErrorMetric(recommendErrorStageCandidateCache, runtime.variantID)
			return nil, err
		}
	}
	recordRecommendStageDurationMetric(
		recommendStageRecall,
		runtime.variantID,
		time.Since(recallStarted),
	)

	coarseRankStarted := time.Now()
	merged = recommend.CoarseRank(merged, cfg.Rank)
	recordRecommendStageDurationMetric(
		recommendStageCoarseRank,
		runtime.variantID,
		time.Since(coarseRankStarted),
	)
	if len(merged) == 0 {
		return &contentpb.RecommendFeedRes{
			Items:      []*contentpb.ContentItem{},
			NextCursor: 0,
			HasMore:    false,
			SnapshotId: hotSnapshotID,
		}, nil
	}

	featureLoadStarted := time.Now()
	features, err := l.loadCandidateFeatures(recommend.IDs(merged))
	recordRecommendStageDurationMetric(
		recommendStageFeatureLoad,
		runtime.variantID,
		time.Since(featureLoadStarted),
	)
	if err != nil {
		recordRecommendErrorMetric(recommendErrorStageFeatureLoad, runtime.variantID)
		return nil, err
	}
	ranked := recommend.ApplyFeatures(merged, features)
	if err := l.applySeenCounts(in.GetUserId(), ranked); err != nil {
		recordRecommendErrorMetric(recommendErrorStageSeenLoad, runtime.variantID)
		return nil, err
	}
	fineRankStarted := time.Now()
	ranked = recommend.FineRank(ranked, cfg.Rank)
	recordRecommendStageDurationMetric(
		recommendStageFineRank,
		runtime.variantID,
		time.Since(fineRankStarted),
	)
	rerankStarted := time.Now()
	var rerankAdjustments map[string]int
	ranked, rerankAdjustments = recommend.DiversityRerankWithAdjustments(ranked, cfg.Diversity)
	recordRerankAdjustments(runtime.variantID, rerankAdjustments)
	recordRecommendStageDurationMetric(
		recommendStageRerank,
		runtime.variantID,
		time.Since(rerankStarted),
	)
	if len(ranked) == 0 {
		return &contentpb.RecommendFeedRes{
			Items:      []*contentpb.ContentItem{},
			NextCursor: 0,
			HasMore:    false,
			SnapshotId: hotSnapshotID,
		}, nil
	}

	snapshotSaveStarted := time.Now()
	snapshotID, err := recommend.SavePersonalizedSnapshotWithMeta(
		l.ctx,
		l.svcCtx.Redis,
		cfg,
		in.GetUserId(),
		ranked,
		recommend.SnapshotMeta{
			VariantID:  runtime.variantID,
			ConfigHash: runtime.configHash,
		},
		time.Now(),
	)
	recordRecommendStageDurationMetric(
		recommendStageSnapshotSave,
		runtime.variantID,
		time.Since(snapshotSaveStarted),
	)
	if err != nil {
		recordRecommendErrorMetric(recommendErrorStageSnapshotSave, runtime.variantID)
		return nil, err
	}
	if snapshotID == "" {
		recordRecommendSnapshotMetric(
			recommendSnapshotKindPersonalized,
			recommendSnapshotResultSkipped,
		)
		return &contentpb.RecommendFeedRes{
			Items:      []*contentpb.ContentItem{},
			NextCursor: 0,
			HasMore:    false,
			SnapshotId: hotSnapshotID,
		}, nil
	}
	recordRecommendSnapshotMetric(
		recommendSnapshotKindPersonalized,
		recommendSnapshotResultSaved,
	)

	result, err := l.queryHotIDsByCursor(
		redisconsts.BuildRecommendUserSnapshotKey(snapshotID),
		snapshotID,
		strings.TrimSpace(in.GetCursor()),
		pageSize,
	)
	if err != nil {
		recordRecommendErrorMetric(recommendErrorStageSnapshotRead, runtime.variantID)
		return nil, err
	}
	resp, err := l.buildFeedResponse(in, result)
	if err != nil {
		recordRecommendErrorMetric(recommendErrorStageBuildItems, runtime.variantID)
		return nil, err
	}
	if err := l.recordSeenResponse(in.GetUserId(), cfg, resp); err != nil {
		recordRecommendErrorMetric(recommendErrorStageSeenWrite, runtime.variantID)
		return nil, err
	}
	return resp, nil
}

func (l *RecommendFeedLogic) recallRecommendCandidates(
	in *contentpb.RecommendFeedReq,
	pageSize int,
	cfg contentconfig.RecommendConfig,
	variantID string,
) ([]recommend.MergeInput, string, error) {
	inputs := make([]recommend.MergeInput, 0, 3)
	var hotSnapshotID string

	if cfg.Hot.Enabled {
		hotLimit := cfg.Hot.Limit
		if hotLimit > cfg.CandidateLimit {
			hotLimit = cfg.CandidateLimit
		}
		if hotLimit < pageSize {
			hotLimit = pageSize
		}

		hotResult, err := l.queryHotIDsByCursor("", "", "", hotLimit)
		if err != nil {
			return nil, "", err
		}
		hotSnapshotID = hotResult.resolvedSnapshotID
		recordRecommendRecallItemsMetric(
			recommendRecallSourceHot,
			variantID,
			len(hotResult.ids),
		)
		inputs = append(inputs, recommend.MergeInput{
			Source: recommend.SourceHot,
			Weight: cfg.Hot.Weight,
			IDs:    hotResult.ids,
		})
	}
	if cfg.NewContent.Enabled {
		newIDs, err := recommend.RecallNewContent(l.ctx, l.svcCtx.Redis, cfg.NewContent.Limit)
		if err != nil {
			return nil, "", err
		}
		recordRecommendRecallItemsMetric(
			recommendRecallSourceNewContent,
			variantID,
			len(newIDs),
		)
		inputs = append(inputs, recommend.MergeInput{
			Source: recommend.SourceNewContent,
			Weight: cfg.NewContent.Weight,
			IDs:    newIDs,
		})
	}
	if cfg.Interest.Enabled && in.GetUserId() > 0 {
		interestIDs, err := recommend.RecallInterest(l.ctx, l.svcCtx.Redis, in.GetUserId(), cfg.Interest)
		if err != nil {
			recordRecommendProfileMetric(recommendProfileResultError)
			return nil, "", err
		}
		if len(interestIDs) == 0 {
			recordRecommendProfileMetric(recommendProfileResultMiss)
		} else {
			recordRecommendProfileMetric(recommendProfileResultHit)
		}
		recordRecommendRecallItemsMetric(
			recommendRecallSourceInterest,
			variantID,
			len(interestIDs),
		)
		inputs = append(inputs, recommend.MergeInput{
			Source: recommend.SourceInterest,
			Weight: cfg.Interest.Weight,
			IDs:    interestIDs,
		})
	} else if cfg.Interest.Enabled {
		recordRecommendProfileMetric(recommendProfileResultSkipped)
	} else {
		recordRecommendProfileMetric(recommendProfileResultDisabled)
	}
	return inputs, hotSnapshotID, nil
}

func recordRerankAdjustments(variantID string, adjustments map[string]int) {
	for rule, count := range adjustments {
		if count <= 0 {
			continue
		}
		recordRecommendRerankAdjustMetric(rule, variantID, count)
	}
}

func (l *RecommendFeedLogic) loadCandidateFeatures(ids []int64) (map[int64]recommend.Candidate, error) {
	contents, err := l.itemBuilder.LoadContentsByIDs(ids)
	if err != nil {
		return nil, err
	}

	features := make(map[int64]recommend.Candidate, len(contents))
	for _, row := range contents {
		if row == nil || row.ID <= 0 {
			continue
		}
		feature := recommend.Candidate{
			ContentID:   row.ID,
			AuthorID:    row.UserID,
			ContentType: row.ContentType,
		}
		if row.PublishedAt != nil {
			feature.PublishedAt = row.PublishedAt.Unix()
		}
		features[row.ID] = feature
	}
	return features, nil
}

func (l *RecommendFeedLogic) applySeenCounts(userID int64, candidates []recommend.Candidate) error {
	if userID <= 0 || len(candidates) == 0 {
		return nil
	}
	seenCounts, err := recommend.LoadSeenCounts(l.ctx, l.svcCtx.Redis, userID, recommend.IDs(candidates))
	if err != nil {
		return err
	}
	if len(seenCounts) == 0 {
		return nil
	}
	for i := range candidates {
		if candidates[i].ContentID <= 0 {
			continue
		}
		candidates[i].SeenCount = seenCounts[candidates[i].ContentID]
	}
	return nil
}

func (l *RecommendFeedLogic) recordSeenResponse(
	userID int64,
	cfg contentconfig.RecommendConfig,
	resp *contentpb.RecommendFeedRes,
) error {
	if userID <= 0 || resp == nil || len(resp.GetItems()) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		if item == nil || item.GetContentId() <= 0 {
			continue
		}
		ids = append(ids, item.GetContentId())
	}
	return recommend.RecordSeenContents(l.ctx, l.svcCtx.Redis, cfg, userID, ids, time.Now())
}

func (l *RecommendFeedLogic) emitExposureTrackEvents(
	userID int64,
	resp *contentpb.RecommendFeedRes,
	variantID string,
) {
	if l == nil || l.svcCtx == nil || l.svcCtx.RecommendTrackProducer == nil || resp == nil {
		return
	}
	if len(resp.GetItems()) == 0 {
		return
	}

	variantID = strings.TrimSpace(variantID)
	if variantID == "" {
		variantID = recommendVariantControl
	}

	now := time.Now()
	for pos, item := range resp.GetItems() {
		if item == nil || item.GetContentId() <= 0 {
			continue
		}

		event := track.Event{
			EventID:    fmt.Sprintf("rec_exposure_%d_%d_%d", userID, item.GetContentId(), now.UnixNano()),
			EventType:  track.EventTypeExposure,
			UserID:     userID,
			ContentID:  item.GetContentId(),
			SnapshotID: resp.GetSnapshotId(),
			VariantID:  variantID,
			Source:     "recommend",
			Position:   pos + 1,
			OccurredAt: now.Unix(),
		}
		if err := l.svcCtx.RecommendTrackProducer.Emit(l.ctx, event); err != nil {
			recordRecommendTrackEmitMetric(track.EventTypeExposure, recommendResultError)
			l.Errorf("emit recommend exposure track event failed, event_id=%s, err=%v", event.EventID, err)
			continue
		}
		recordRecommendTrackEmitMetric(track.EventTypeExposure, recommendResultSuccess)
	}
}

func (l *RecommendFeedLogic) resolveSnapshotKey(reqSnapshotID *string) (string, string) {
	if reqSnapshotID == nil {
		return "", ""
	}
	snapshotID := strings.TrimSpace(*reqSnapshotID)
	if snapshotID == "" {
		return "", ""
	}
	return redisconsts.BuildHotFeedSnapshotKey(snapshotID), snapshotID
}

func (l *RecommendFeedLogic) queryHotIDsByCursor(preferredKey, preferredSnapshotID, cursor string, pageSize int) (*hotFeedResult, error) {
	result, cacheResult := l.queryFromRedis(preferredKey, preferredSnapshotID, cursor, pageSize)
	if cacheResult == cacheHit {
		return result, nil
	}
	if cacheResult == cacheMiss {
		return &hotFeedResult{}, nil
	}
	return nil, mapHotFeedCacheError(cacheResult)
}

func mapHotFeedCacheError(cacheResult CacheResult) error {
	switch cacheResult {
	case cacheError:
		return errorx.NewMsg("查询热榜索引失败")
	default:
		return errorx.NewMsg("查询失败请稍后重试")
	}
}

func (l *RecommendFeedLogic) queryFromRedis(preferredKey, preferredSnapshotID, cursor string, pageSize int) (*hotFeedResult, CacheResult) {
	res, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.QueryHotFeedZSetScript,
		[]string{
			preferredKey,
			redisconsts.HotFeedLatestKey,
			redisconsts.HotFeedSnapshotPrefix,
			redisconsts.HotFeedKey,
		},
		cursor,
		strconv.FormatInt(int64(pageSize), 10),
		preferredSnapshotID,
	)
	if err != nil {
		l.Errorf("query hot feed from redis failed: %v", err)
		return nil, cacheError
	}

	parsed, exists, parseErr := parseHotFeedLuaResult(res)
	if parseErr != nil {
		l.Errorf("parse hot feed lua result failed: %v", parseErr)
		return nil, cacheError
	}
	if !exists {
		return nil, cacheMiss
	}
	return parsed, cacheHit
}

func parseHotFeedLuaResult(res any) (*hotFeedResult, bool, error) {
	arr, ok := res.([]interface{})
	if !ok || len(arr) < 4 {
		return nil, false, errorx.NewMsg("查询热榜索引失败")
	}

	existsVal, _ := luaReplyInt64(arr[0])
	exists := existsVal == 1
	hasMoreVal, _ := luaReplyInt64(arr[1])
	hasMore := hasMoreVal == 1

	nextCursor := int64(0)
	if hasMore {
		nextStr, _ := luaReplyString(arr[2])
		if nextStr != "" {
			value, err := strconv.ParseInt(nextStr, 10, 64)
			if err != nil {
				return nil, false, errorx.NewMsg("查询热榜索引失败")
			}
			nextCursor = value
		}
	}

	resolvedSnapshotID, _ := luaReplyString(arr[3])

	ids := make([]int64, 0, len(arr)-4)
	for i := 4; i < len(arr); i++ {
		idStr, _ := luaReplyString(arr[i])
		if idStr == "" {
			continue
		}
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		ids = append(ids, id)
	}

	return &hotFeedResult{
		ids:                ids,
		nextCursor:         nextCursor,
		hasMore:            hasMore,
		resolvedSnapshotID: resolvedSnapshotID,
	}, exists, nil
}
