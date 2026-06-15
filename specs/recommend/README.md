# Specification: zfeed 推荐流架构升级方案

## Metadata
- **Version**: 1.0.0
- **Status**: Draft
- **Author**: Codex
- **Created**: 2026-05-31
- **Last Updated**: 2026-06-15

## Overview

`zfeed` 当前推荐流由 `front-api /v1/feed/recommend` 调用 `content-rpc.FeedService.RecommendFeed`，再通过 Redis 热榜快照读取 `feed:hot:global:snap:{snapshot_id}` 或 `feed:hot:global`。热榜由 `count-rpc` 通过 Canal 计数事件写入 `feed:hot:global:inc:{shard}`，再由 `content-rpc` 的 `hot.fast.update`、`hot.cold.rebuild`、`hot.snapshot.refresh` 维护全局热度 ZSET 和快照。

本方案在保持 `RecommendFeed(cursor, page_size, snapshot_id)` 协议兼容的前提下，把推荐流升级为“热榜兜底 + 多路召回 + 可配置排序 + 多样性重排 + 用户画像 + 实验观测”的渐进式架构。第一阶段不建议直接引入独立推荐服务或向量数据库；对简历项目和面试讲解来说，先在 `content-rpc` 内新增推荐编排包更清晰，能复用现有 `FeedItemBuilder`、热榜 Lua 和 Redis/MySQL/Kafka 基础设施。等画像、实验、指标稳定后，再把编排层抽成 `recommend-rpc`。

## Requirements

### Functional Requirements
- FR-1: 保留现有热榜快照路径，任何异常、开关关闭、匿名用户无画像时均可回退到当前 `queryHotIDsByCursor`。
- FR-2: 推荐编排层支持热榜、新内容冷启动、用户兴趣三路召回，输出统一候选格式。
- FR-3: 支持粗排、精排、重排三段排序，排序参数可由 YAML 默认配置和 Redis 动态开关覆盖。
- FR-4: 支持用户画像 Redis Hash，能从点赞、收藏、评论等互动事件增量更新兴趣标签。
- FR-5: 个性化分页继续返回 `snapshot_id` 和 `next_cursor`，翻页结果在同一个 snapshot 内稳定。
- FR-6: 支持基于 `user_id` hash 的 A/B 实验，记录曝光、点击、互动、停留等事件。
- FR-7: 新增推荐核心指标、Kafka 行为日志和 Grafana 面板。

### Non-functional Requirements
- NFR-1: 推荐主链路 P99 目标不超过现有热榜链路的 2 倍；召回和排序需有超时，默认总预算 120ms。
- NFR-2: Redis key 必须有 TTL 或清理任务，避免候选快照和画像无限增长。
- NFR-3: Prometheus label 保持低基数，不允许把 `user_id`、`content_id` 作为指标 label。
- NFR-4: 所有新增能力必须能通过 feature flag 灰度和一键回滚。

## 1. 总体架构

### 1.1 分层架构图

```text
Client
  |
  v
front-api /v1/feed/recommend
  |
  v
content-rpc FeedService.RecommendFeed
  |
  +--> FeatureFlag + ExperimentResolver
  |
  +--> RecommendOrchestrator
        |
        +--> SnapshotResolver
        |     +--> personalized snapshot: feed:rec:user:snap:{snapshot_id}
        |     +--> fallback snapshot:      feed:hot:global:snap:{snapshot_id}
        |
        +--> Multi Recall
        |     +--> HotRecall          -> feed:hot:global:snap:{id} / feed:hot:global
        |     +--> NewContentRecall   -> feed:rec:new:global
        |     +--> InterestRecall     -> rec:user:profile:{user_id}
        |                               -> rec:tag:index:{tag}
        |
        +--> CandidateMerge + Dedup
        +--> FeatureLoad
        |     +--> MySQL zfeed_content
        |     +--> Redis rec:content:tags:{content_id}
        |     +--> count-rpc / user-rpc
        |
        +--> CoarseRank -> FineRank -> DiversityRerank
        |
        +--> SnapshotCache
        |     +--> feed:rec:candidate:{bucket}:{variant}:{version}
        |     +--> feed:rec:user:snap:{snapshot_id}
        |
        +--> FeedItemBuilder
              +--> content/article/video MySQL
              +--> user-rpc BatchGetUser
              +--> interaction-rpc BatchIsLiked
```

异步数据闭环：

```text
PublishArticle / PublishVideo
  -> zfeed_content / zfeed_article / zfeed_video
  -> feed:user:publish:{author_id}
  -> feed:rec:new:global
  -> rec:content:tags:{content_id}
  -> rec:tag:index:{tag}

interaction-rpc like/favorite/comment
  -> MySQL interaction tables
  -> Kafka / Canal
  -> count-rpc Dispatcher -> feed:hot:global:inc:{shard}
  -> rec-profile-worker  -> rec:user:profile:{user_id}

Recommend exposure/click/dwell
  -> Kafka zfeed-rec-track
  -> offline aggregation / Grafana
  -> profile update / hot score calibration
```

### 1.2 各层职责

| 层 | 职责 | 当前项目结合点 |
| --- | --- | --- |
| `front-api` | 参数校验、可选登录、透传 `user_id/cursor/snapshot_id` | 保持 `app/front/doc/feed/feed.api` 不变 |
| `content-rpc FeedService` | 推荐主入口、兜底控制、结果组装 | 扩展 `RecommendFeedLogic`，复用 `FeedItemBuilder` |
| `ExperimentResolver` | 解析开关和实验版本 | 新增 `RecommendConfig`，Redis 动态覆盖 |
| `RecallSource` | 多路候选生成 | 热榜复用 `queryHotIDsByCursor` 逻辑；新内容和兴趣新增 Redis 索引 |
| `Ranker` | 粗排、精排 | 使用本地 Go 公式，暂不引入模型服务 |
| `Reranker` | 多样性、去重、已曝光过滤 | 依赖内容作者、类型和 `rec:seen:{user_id}` |
| `SnapshotCache` | 个性化稳定分页 | 新增 `feed:rec:user:snap:{snapshot_id}` |
| 异步 Worker | 标签、画像、埋点、候选索引维护 | 初期放在 `content-rpc` cron/worker 或新 `recommend-worker`，后期可拆 `recommend-rpc` |

### 1.3 与现有热榜链路结合

当前热榜链路保留为基线：

- `feed:hot:global`: 全局热榜 ZSET。
- `feed:hot:global:latest`: 最新热榜快照 ID。
- `feed:hot:global:snap:{snapshot_id}`: 稳定分页快照。
- `feed:hot:global:inc:{shard}`: `count-rpc` 增量热度桶。
- `hot.fast.update` / `hot.cold.rebuild` / `hot.snapshot.refresh`: 继续负责热榜维护。

`RecommendFeedLogic` 建议改为两段：

```go
func (l *RecommendFeedLogic) RecommendFeed(in *contentpb.RecommendFeedReq) (*contentpb.RecommendFeedRes, error) {
    if !l.svcCtx.Config.Recommend.Enabled || in.GetUserId() <= 0 {
        return l.recommendHotFallback(in)
    }

    resp, err := l.recommendEngine.Recommend(l.ctx, in)
    if err == nil && len(resp.GetItems()) > 0 {
        return resp, nil
    }

    l.Errorf("personalized recommend fallback, user_id=%d, err=%v", in.GetUserId(), err)
    return l.recommendHotFallback(in)
}
```

这里 `recommendHotFallback` 就是当前 `queryHotIDsByCursor + FeedItemBuilder` 路径，确保上线风险可控。

## 2. 多路召回设计

### 2.1 统一候选格式

```go
type RecallSourceName string

const (
    RecallHot      RecallSourceName = "hot"
    RecallNew      RecallSourceName = "new_content"
    RecallInterest RecallSourceName = "interest"
)

type Candidate struct {
    ContentID    int64
    AuthorID     int64
    ContentType  int32
    PublishedAt  int64
    SourceScores map[RecallSourceName]float64
    SourceRanks  map[RecallSourceName]int
    HotScore     float64
    InterestScore float64
    FreshnessScore float64
    MergeScore   float64
    FinalScore   float64
    Tags         map[string]float64
}

type RecallRequest struct {
    UserID       int64
    Cursor       string
    PageSize     int
    Limit        int
    SnapshotID   string
    Experiment   ExperimentVariant
    NowUnix      int64
}

type RecallSource interface {
    Name() RecallSourceName
    Recall(ctx context.Context, req RecallRequest) ([]Candidate, error)
}
```

召回源内部只返回轻量候选，最终卡片仍由 `FeedItemBuilder` 批量补全，避免重复造详情聚合逻辑。

### 2.2 热榜召回

触发条件：

- 所有请求都可以触发，作为基础召回。
- 匿名用户、新用户无画像、个性化链路错误时，直接退化为热榜结果。

数据来源：

- `feed:hot:global:snap:{snapshot_id}` 优先。
- 无指定或快照失效时读 `feed:hot:global:latest` 对应快照。
- 再失败读 `feed:hot:global`。

输出规则：

- 默认召回 300 条。
- `HotScore` 取 Redis ZSET score。
- `SourceScores["hot"] = normalize(hot_score)`。

Redis 结构沿用：

```text
ZSET feed:hot:global
  member = content_id
  score  = hot_score

STRING feed:hot:global:latest = 20260531153000

ZSET feed:hot:global:snap:20260531153000
  member = content_id
  score  = hot_score
```

### 2.3 新内容冷启动召回

触发条件：

- `Recommend.NewContent.Enabled = true`。
- 内容发布时间在最近 24-72 小时内。
- 内容为 `status=30`、`visibility=10`、`is_deleted=0`。

数据来源：

- 发布成功后由 `PublishArticle` / `PublishVideo` 写入。
- 或由 Canal 监听 `zfeed_content` 的公开发布事件补偿写入。
- 作者质量可用 `count-rpc.GetUserProfileCounts(author_id).followed_count`，没有时默认 0。

Redis 结构：

```text
ZSET feed:rec:new:global
  member = content_id
  score  = cold_score

HASH feed:rec:new:meta:{content_id}
  author_id      = 1001
  content_type   = 10
  published_at   = 1780191000
  author_quality = 0.43
  init_quality   = 0.15
  tags           = {"go":0.8,"redis":0.6,"type:article":1.0}
TTL: 7d
```

推荐冷启动分数：

```text
freshness = exp(-age_hours / 36)
author_quality = min(1, log1p(author_followed_count) / log1p(100000))
init_quality = title_cover_score + media_ready_score
cold_score = 0.65 * freshness + 0.25 * author_quality + 0.10 * init_quality
```

写入伪代码：

```go
func (w *NewContentRecallWriter) OnPublished(ctx context.Context, c PublishedContent) error {
    if c.Status != 30 || c.Visibility != 10 || c.PublishedAt.IsZero() {
        return nil
    }

    quality := w.authorQuality(ctx, c.AuthorID)
    tags := w.tagger.Extract(c.Title, c.Description, c.ContentType)
    score := ColdStartScore(c.PublishedAt, quality, c.InitQuality, time.Now())

    pipe := w.redis.Pipeline()
    pipe.ZAdd(ctx, "feed:rec:new:global", redis.Z{Score: score, Member: c.ContentID})
    pipe.HSet(ctx, fmt.Sprintf("feed:rec:new:meta:%d", c.ContentID), map[string]any{
        "author_id": c.AuthorID, "content_type": c.ContentType,
        "published_at": c.PublishedAt.Unix(), "author_quality": quality,
        "tags": mustJSON(tags),
    })
    pipe.Expire(ctx, fmt.Sprintf("feed:rec:new:meta:%d", c.ContentID), 7*24*time.Hour)
    for tag, weight := range tags {
        pipe.HSet(ctx, fmt.Sprintf("rec:content:tags:%d", c.ContentID), tag, weight)
        pipe.ZAdd(ctx, fmt.Sprintf("rec:tag:index:%s", tag), redis.Z{Score: score * weight, Member: c.ContentID})
    }
    _, err := pipe.Exec(ctx)
    return err
}
```

清理策略：

- 新增 `rec.new.cleanup` XXL-Job，每小时删除 `published_at < now-7d` 的 member。
- 删除内容时沿用 `DeleteContentLogic` 的热榜清理思路，同时 `ZREM feed:rec:new:global`、删除 `feed:rec:new:meta:{id}`、从 tag index 移除。

### 2.4 用户兴趣召回

触发条件：

- `user_id > 0`。
- `rec:user:profile:{user_id}` 存在且 `tag_count >= 3`。
- 实验 variant 开启 `interest.enabled`。

画像来源：

- `interaction-rpc` 点赞、收藏、评论事件。
- 初期可复用 Canal 对 `zfeed_like/zfeed_favorite/zfeed_comment` 的变更；更推荐在 interaction 写路径补统一 outbox，发布 `zfeed-user-action`，避免画像逻辑依赖数据库表结构。

内容标签来源：

- 第一阶段可从 `zfeed_article.title/description`、`zfeed_video.title/description` 提取关键词，并补充 `type:article`、`type:video`。
- 第二阶段新增持久化表：

```sql
CREATE TABLE IF NOT EXISTS zfeed_content_tag (
  content_id BIGINT NOT NULL,
  tag VARCHAR(64) NOT NULL,
  weight DOUBLE NOT NULL DEFAULT 0,
  source VARCHAR(32) NOT NULL DEFAULT 'rule',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (content_id, tag),
  KEY idx_tag_weight (tag, weight)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

在线 Redis 索引：

```text
HASH rec:user:profile:{user_id}
  tags            = {"go":2.31,"redis":1.42,"type:article":0.9}
  tag_count       = 3
  updated_at      = 1780191000
  profile_version = 12
  last_event_id   = like_1001_9001_xxx
TTL: 30d, 每次更新续期

HASH rec:content:tags:{content_id}
  go           = 0.8
  redis        = 0.6
  type:article = 1.0
TTL: 30d

ZSET rec:tag:index:{tag}
  member = content_id
  score  = tag_weight * freshness * hot_score_norm
```

兴趣召回逻辑：

```go
func (r *InterestRecall) Recall(ctx context.Context, req RecallRequest) ([]Candidate, error) {
    profile, err := r.profileStore.Get(ctx, req.UserID)
    if err != nil || profile.IsCold() {
        return nil, nil
    }

    tags := profile.TopTags(8)
    merged := make(map[int64]*Candidate)
    for _, tag := range tags {
        ids, _ := r.redis.ZRevRangeWithScores(ctx, "rec:tag:index:"+tag.Name, 0, 100)
        for rank, item := range ids {
            id := parseID(item.Member)
            c := getOrCreate(merged, id)
            c.SourceScores[RecallInterest] += tag.Weight * normalize(item.Score)
            c.SourceRanks[RecallInterest] = minRank(c.SourceRanks[RecallInterest], rank+1)
        }
    }

    return topCandidates(merged, req.Limit), nil
}
```

### 2.5 Merge 逻辑

目标：多路召回去重、保留来源贡献、控制单通道垄断。

```go
type RecallWeights struct {
    Hot      float64 `json:"hot"`
    New      float64 `json:"new_content"`
    Interest float64 `json:"interest"`
}

func MergeCandidates(groups map[RecallSourceName][]Candidate, w RecallWeights, limit int) []Candidate {
    byID := make(map[int64]*Candidate)
    for source, list := range groups {
        for rank, item := range list {
            c := byID[item.ContentID]
            if c == nil {
                copy := item
                copy.SourceScores = map[RecallSourceName]float64{}
                copy.SourceRanks = map[RecallSourceName]int{}
                byID[item.ContentID] = &copy
                c = &copy
            }
            sourceScore := item.SourceScores[source]
            c.SourceScores[source] = max(c.SourceScores[source], sourceScore)
            c.SourceRanks[source] = minPositive(c.SourceRanks[source], rank+1)
            c.MergeScore += sourceWeight(source, w) * sourceScore
        }
    }
    return sortAndLimit(byID, limit)
}
```

默认权重建议：

```yaml
recall_weights:
  hot: 0.55
  new_content: 0.20
  interest: 0.25
```

## 3. 排序模型设计

### 3.1 特征结构

```go
type ContentFeature struct {
    ContentID     int64
    AuthorID      int64
    ContentType   int32
    PublishedAt   time.Time
    HotScore      float64
    LikeCount     int64
    FavoriteCount int64
    CommentCount  int64
    Tags          map[string]float64
}

type UserFeature struct {
    UserID int64
    Tags   map[string]float64
    Seen   map[int64]int64
}

type RankConfig struct {
    AlphaHot       float64 `json:"alpha_hot"`
    BetaInterest  float64 `json:"beta_interest"`
    GammaFresh    float64 `json:"gamma_fresh"`
    DeltaQuality  float64 `json:"delta_quality"`
    SeenPenalty   float64 `json:"seen_penalty"`
}
```

### 3.2 粗排

输入约 500 条候选，快速截断到 200 条。粗排只用 Redis/MySQL 已经有的轻量字段，避免调用过多 RPC。

```text
coarse_score =
  0.60 * merge_score
  + 0.25 * hot_score_norm
  + 0.15 * freshness_score
  - seen_penalty
```

```go
type CoarseRanker interface {
    Rank(ctx context.Context, candidates []Candidate, cfg RankConfig) []Candidate
}

func (r *SimpleCoarseRanker) Rank(ctx context.Context, cs []Candidate, cfg RankConfig) []Candidate {
    for i := range cs {
        cs[i].FreshnessScore = FreshnessScore(cs[i].PublishedAt, r.now())
        cs[i].HotScore = NormalizeHot(cs[i].HotScore)
        cs[i].FinalScore = 0.60*cs[i].MergeScore + 0.25*cs[i].HotScore + 0.15*cs[i].FreshnessScore
    }
    sort.Slice(cs, func(i, j int) bool { return cs[i].FinalScore > cs[j].FinalScore })
    return limitCandidates(cs, 200)
}
```

### 3.3 精排

精排支持可配置混合排序：

```text
final_score =
  alpha_hot      * hot_score_norm
  + beta_interest * interest_score
  + gamma_fresh   * freshness_score
  + delta_quality * quality_score
  - seen_penalty
```

建议初始值：

```yaml
rank:
  alpha_hot: 0.45
  beta_interest: 0.30
  gamma_fresh: 0.20
  delta_quality: 0.05
  seen_penalty: 0.30
```

兴趣匹配使用标签向量点积，先不引入深度模型：

```go
func InterestScore(userTags, contentTags map[string]float64) float64 {
    var dot, userNorm, contentNorm float64
    for tag, uw := range userTags {
        userNorm += uw * uw
        if cw, ok := contentTags[tag]; ok {
            dot += uw * cw
        }
    }
    for _, cw := range contentTags {
        contentNorm += cw * cw
    }
    if userNorm == 0 || contentNorm == 0 {
        return 0
    }
    return dot / math.Sqrt(userNorm*contentNorm)
}
```

精排接口：

```go
type FineRanker interface {
    Rank(ctx context.Context, req RankRequest) ([]Candidate, error)
}

type RankRequest struct {
    User       UserFeature
    Candidates []Candidate
    Config     RankConfig
    VariantID  string
}
```

### 3.4 重排与多样性

规则：

- 同作者打散：默认任意 5 条窗口内同作者最多 1 条。
- 内容类型打散：默认任意 6 条窗口内同类型最多 4 条，防止全是文章或全是视频。
- 新内容保底：前 20 条至少 2 条 `age < 24h` 的新内容，前提是候选池存在。
- 已曝光降权：`rec:seen:{user_id}` 最近 24 小时曝光过的内容不直接删除，而是降权；连续曝光 2 次以上才过滤，避免候选不足。

Redis 已曝光记录：

```text
ZSET rec:seen:{user_id}
  member = content_id
  score  = last_expose_unix
TTL: 7d
```

滑窗重排伪代码：

```go
type DiversityRule struct {
    AuthorWindow       int
    MaxSameAuthor      int
    TypeWindow         int
    MaxSameType        int
    NewContentTopN     int
    NewContentMinCount int
}

func DiversityRerank(items []Candidate, rule DiversityRule) []Candidate {
    out := make([]Candidate, 0, len(items))
    used := make([]bool, len(items))

    for len(out) < len(items) {
        pick := -1
        for i := range items {
            if used[i] {
                continue
            }
            if violatesWindow(out, items[i], rule) {
                continue
            }
            pick = i
            break
        }
        if pick < 0 {
            pick = firstUnused(used)
        }
        used[pick] = true
        out = append(out, items[pick])
    }
    return ensureNewContentQuota(out, rule)
}
```

## 4. 用户画像系统

### 4.1 Redis Hash 结构

```text
HASH rec:user:profile:{user_id}
  tags             = JSON map[string]float64
  positive_count   = 19
  negative_count   = 2
  last_active_at   = 1780191000
  updated_at       = 1780191000
  profile_version  = 12
  last_event_id    = favorite_1001_9001_xxx
TTL: 30d
```

标签权重建议：

```text
like       +1.0
favorite   +3.0
comment    +2.0
click      +0.5
dwell>10s  +0.8
unlike      -0.8
unfavorite  -1.5
```

时间衰减：

```text
new_weight = old_weight * exp(-hours_since_update / 168) + event_weight * content_tag_weight
```

### 4.2 从 interaction-rpc 事件增量更新

建议新增统一用户行为事件。已有 `zfeed-like` 可先扩展，收藏/评论可通过 Canal 补齐，最终统一到 `zfeed-user-action`。

```go
type UserActionEvent struct {
    EventID     string `json:"event_id"`
    EventType   string `json:"event_type"` // like, favorite, comment, click, dwell, unlike, unfavorite
    UserID      int64  `json:"user_id"`
    ContentID   int64  `json:"content_id"`
    Scene       string `json:"scene"` // ARTICLE, VIDEO
    DwellMs     int64  `json:"dwell_ms,omitempty"`
    OccurredAt  int64  `json:"occurred_at"`
}

type ProfileUpdater interface {
    Apply(ctx context.Context, evt UserActionEvent) error
}
```

画像更新伪代码：

```go
func (u *RedisProfileUpdater) Apply(ctx context.Context, evt UserActionEvent) error {
    if evt.UserID <= 0 || evt.ContentID <= 0 || evt.EventID == "" {
        return nil
    }
    if !u.dedup.MarkOnce(ctx, "rec:profile:event:"+evt.EventID, 24*time.Hour) {
        return nil
    }

    tags, err := u.contentTagStore.Get(ctx, evt.ContentID)
    if err != nil || len(tags) == 0 {
        return nil
    }

    profile, _ := u.profileStore.Get(ctx, evt.UserID)
    delta := eventWeight(evt)
    for tag, cw := range tags {
        profile.Tags[tag] = Decay(profile.Tags[tag], profile.UpdatedAt, time.Now()) + delta*cw
    }
    profile.TrimTopN(50)
    return u.profileStore.Save(ctx, profile)
}
```

### 4.3 标签匹配方法

初期用标签向量点积或余弦相似度即可，原因是：

- 当前内容模型没有显式类目和 embedding 字段。
- Go 本地计算可控，便于面试讲解。
- 不引入额外向量库，符合项目渐进式约束。

如果后续要增强，可把 `rec:content:tags` 替换或补充为 `rec:content:embedding:{id}`，召回层接口不变。

### 4.4 冷启动用户回退

冷启动用户定义：

- 未登录。
- 或 `rec:user:profile:{user_id}` 不存在。
- 或 `tag_count < 3`。
- 或最近 30 天没有正向行为。

回退策略：

1. 匿名用户：热榜 80% + 新内容 20%，不做兴趣召回。
2. 新注册用户：热榜 70% + 新内容 30%，并用 `type:article/type:video` 保持内容类型均衡。
3. 有少量行为用户：热榜 50% + 新内容 20% + 兴趣 30%，但兴趣召回不足时自动把权重转给热榜。

## 5. 缓存与分页策略

### 5.1 快照扩展

保留现有热榜快照，新增个性化快照：

```text
ZSET feed:rec:user:snap:{snapshot_id}
  member = content_id
  score  = rank_pos_score
TTL: 5m - 10m

HASH feed:rec:user:snapmeta:{snapshot_id}
  user_bucket = 0421
  variant_id  = exp_rec_v1_b
  config_hash = 7e3a9c
  created_at  = 1780191000
  recall_size = 500
TTL: same as snapshot
```

`rank_pos_score` 建议用稳定序号而不是最终分数：

```text
rank_pos_score = 1_000_000 - rank
```

这样翻页仍可沿用当前 Lua 的“cursor member(content_id)”语义，不受浮点分数相同影响。

### 5.2 缓存候选集 + 实时 rerank

推荐采用两级缓存：

1. 分桶候选缓存：按用户桶和实验版本缓存召回后的候选，TTL 2-5 分钟。
2. 用户快照缓存：用户首次请求时基于候选缓存实时精排/重排，生成稳定分页快照，TTL 5-10 分钟。

```text
ZSET feed:rec:candidate:{bucket}:{variant_id}:{config_hash}
  member = content_id
  score  = merge_score
TTL: 5m

ZSET feed:rec:user:snap:{snapshot_id}
  member = content_id
  score  = rank_pos_score
TTL: 10m
```

这样既能减少召回成本，又能避免“每页实时 rerank 导致分页抖动”。

### 5.3 snapshot_id 设计

现有热榜 `snapshot_id` 形如 `20260531153000`，继续支持。

个性化 `snapshot_id` 建议带前缀：

```text
rec:{user_bucket}:{variant_id}:{config_hash}:{unix_sec}:{rand6}
```

例：

```text
rec:0421:exp_rec_v1_b:7e3a9c:1780191000:a91f2c
```

`RecommendFeedLogic` 判断逻辑：

```go
func (r SnapshotResolver) Resolve(snapshotID string) SnapshotKind {
    if strings.HasPrefix(snapshotID, "rec:") {
        return SnapshotPersonalized
    }
    return SnapshotHot
}
```

返回协议不变：

```protobuf
message RecommendFeedRes {
  repeated ContentItem items = 1;
  int64 next_cursor = 2;   // 仍是最后一个 content_id
  bool has_more = 3;
  string snapshot_id = 4;
}
```

### 5.4 分页读取伪代码

```go
func (e *Engine) Recommend(ctx context.Context, req *contentpb.RecommendFeedReq) (*contentpb.RecommendFeedRes, error) {
    pageSize := clamp(int(req.GetPageSize()), 1, 50)

    if sid := req.GetSnapshotId(); strings.HasPrefix(sid, "rec:") {
        ids, next, hasMore, err := e.snapshots.Page(ctx, sid, req.GetCursor(), pageSize)
        return e.buildResponse(ctx, req, sid, ids, next, hasMore, err)
    }

    variant := e.experiments.Resolve(req.GetUserId())
    candidates, err := e.getOrBuildCandidates(ctx, req.GetUserId(), variant)
    if err != nil {
        return nil, err
    }

    ranked := e.rankPipeline.Run(ctx, candidates, variant)
    snapshotID, err := e.snapshots.SavePersonalized(ctx, req.GetUserId(), variant, ranked)
    if err != nil {
        return nil, err
    }

    ids, next, hasMore, err := e.snapshots.Page(ctx, snapshotID, req.GetCursor(), pageSize)
    return e.buildResponse(ctx, req, snapshotID, ids, next, hasMore, err)
}
```

## 6. 实验与开关设计

### 6.1 YAML 默认配置

`app/rpc/content/etc/content.yaml` 可新增：

```yaml
Recommend:
  Enabled: true
  TimeoutMs: 120
  SnapshotTTL: 600
  CandidateTTL: 300
  CandidateLimit: 500
  FallbackToHot: true

  Recall:
    Hot:
      Enabled: true
      Weight: 0.55
      Limit: 300
    NewContent:
      Enabled: true
      Weight: 0.20
      Limit: 100
      MaxAgeHours: 72
    Interest:
      Enabled: true
      Weight: 0.25
      Limit: 200
      TopTags: 8

  Rank:
    AlphaHot: 0.45
    BetaInterest: 0.30
    GammaFresh: 0.20
    DeltaQuality: 0.05
    SeenPenalty: 0.30

  Diversity:
    AuthorWindow: 5
    MaxSameAuthor: 1
    TypeWindow: 6
    MaxSameType: 4
    NewContentTopN: 20
    NewContentMinCount: 2
```

Go 配置结构：

```go
type RecommendConfig struct {
    Enabled        bool
    TimeoutMs      int
    SnapshotTTL    int
    CandidateTTL   int
    CandidateLimit int
    FallbackToHot  bool
    Recall         RecallConfig
    Rank           RankConfig
    Diversity      DiversityRule
}
```

### 6.2 Redis 动态开关

```text
HASH rec:flag:recommend
  enabled = true
  fallback_to_hot = true
  recall.hot.weight = 0.55
  recall.new_content.weight = 0.20
  recall.interest.weight = 0.25
  rank.alpha_hot = 0.45
  rank.beta_interest = 0.30
  rank.gamma_fresh = 0.20
  diversity.author_window = 5
  config_version = 20260531_01
```

加载策略：

- YAML 是默认值。
- Redis 覆盖 YAML，缓存 10 秒，防止每次请求都打 Redis。
- Redis 配置非法时忽略单项并打 warn log。

### 6.3 可实验变量

至少三类：

| 变量 | 示例 |
| --- | --- |
| 召回通道权重 | hot/new/interest = 0.55/0.20/0.25 vs 0.40/0.25/0.35 |
| 排序公式参数 | `alpha_hot`、`beta_interest`、`gamma_fresh` |
| 多样性力度 | `AuthorWindow`、`MaxSameAuthor`、`NewContentMinCount` |

实验配置：

```yaml
experiments:
  - id: exp_rec_v1
    layer: recommend_rank
    salt: zfeed_20260531
    enabled: true
    variants:
      - id: a
        traffic: 50
        overrides:
          recall.interest.weight: 0.20
          rank.beta_interest: 0.25
      - id: b
        traffic: 50
        overrides:
          recall.interest.weight: 0.35
          rank.beta_interest: 0.40
          diversity.new_content_min_count: 3
```

分流逻辑：

```go
func ResolveVariant(userID int64, exp Experiment) Variant {
    if userID <= 0 || !exp.Enabled {
        return exp.DefaultVariant()
    }
    h := murmur3.Sum32([]byte(fmt.Sprintf("%s:%d:%s", exp.ID, userID, exp.Salt)))
    bucket := int(h % 10000)
    cursor := 0
    for _, v := range exp.Variants {
        cursor += v.TrafficPermyriad
        if bucket < cursor {
            return v
        }
    }
    return exp.DefaultVariant()
}
```

### 6.4 实验埋点

事件类型：

- `exposure`: Feed 卡片曝光，推荐结果返回时先记服务端曝光；客户端真实曝光后可补前端曝光。
- `click`: 点击进入详情。
- `like/favorite/comment/unfavorite/follow`: 互动。
- `dwell`: 浏览停留时长。

Kafka topic：

```text
zfeed-rec-track
```

事件结构：

```go
type RecommendTrackEvent struct {
    EventID     string  `json:"event_id"`
    EventType   string  `json:"event_type"`
    UserID      int64   `json:"user_id,omitempty"`
    ContentID   int64   `json:"content_id"`
    RequestID   string  `json:"request_id"`
    SnapshotID  string  `json:"snapshot_id"`
    VariantID   string  `json:"variant_id"`
    Source      string  `json:"source"`
    Position    int     `json:"position"`
    FinalScore  float64 `json:"final_score,omitempty"`
    DwellMs     int64   `json:"dwell_ms,omitempty"`
    OccurredAt  int64   `json:"occurred_at"`
}
```

存储建议：

- 在线链路只写 Kafka，不同步写 MySQL。
- 本项目可用消费者按天聚合到 MySQL：`zfeed_rec_metric_daily`。
- 如果生产化，推荐改用 ClickHouse/Doris 做明细 OLAP；但当前技术栈约束下不强行引入。

## 7. 数据闭环与可观测性

### 7.1 核心业务指标

| 指标 | 计算方式 |
| --- | --- |
| CTR | `click_count / exposure_count` |
| 千次曝光互动数 IPM | `(like + favorite + comment) * 1000 / exposure_count` |
| 人均浏览时长 | `sum(dwell_ms) / active_user_count` |
| 新内容曝光占比 | `age < 24h exposure / total exposure` |
| 个性化覆盖率 | `personalized_success / recommend_requests` |
| 兜底率 | `fallback_to_hot / recommend_requests` |
| 快照命中率 | `snapshot_hit / snapshot_lookup` |
| 召回空结果率 | `empty_recall / recall_requests` |

### 7.2 Kafka 数据闭环

```text
RecommendFeed 返回
  -> async emit exposure to zfeed-rec-track
  -> ZADD rec:seen:{user_id} content_id now

Click / Like / Favorite / Comment / Dwell
  -> zfeed-rec-track or zfeed-user-action
  -> ProfileUpdater updates rec:user:profile:{user_id}
  -> count-rpc continues updating feed:hot:global:inc:{shard}
  -> offline job recalibrates tag index and cold-start score
```

离线/准实时任务：

- 每 5 分钟：消费行为事件，更新 `rec:user:profile:{user_id}`。
- 每 10 分钟：刷新 `rec:tag:index:{tag}` 中内容分数，加入最新热度和时间衰减。
- 每小时：清理过期新内容、过期 seen 记录、过期快照。
- 每天：聚合实验指标到 MySQL，用于 Grafana 或面试展示报表。

### 7.3 Prometheus 指标

建议在 `content-rpc` 新增低基数指标：

```text
zfeed_recommend_requests_total{mode,variant,result}
zfeed_recommend_stage_duration_seconds_bucket{stage,variant}
zfeed_recommend_recall_items_total{source,variant}
zfeed_recommend_fallback_total{reason}
zfeed_recommend_snapshot_total{kind,result}
zfeed_recommend_rerank_adjust_total{rule,variant}
zfeed_recommend_error_total{stage,variant}
zfeed_recommend_profile_total{result}
zfeed_recommend_track_emit_total{event_type,result}
zfeed_recommend_track_consume_total{event_type,variant,source,result}
zfeed_recommend_user_action_consume_total{event_type,result}
zfeed_recommend_track_consume_lag_seconds_bucket{event_type,source}
zfeed_recommend_user_action_consume_lag_seconds_bucket{event_type}
zfeed_user_action_outbox_total{action,result}
```

注意：

- `variant` 只允许实验组 ID，如 `a/b/control`。
- `source` 只允许推荐来源或互动来源，如 `hot/new_content/interest/recommend/interaction`。
- 不允许 `user_id/content_id/snapshot_id` 作为 label。

### 7.4 Grafana 面板

在现有 `deploy/grafana/dashboards/zfeed-overview.json` 基础上新增推荐分组：

- 推荐请求 QPS：`sum(rate(zfeed_recommend_requests_total[5m])) by (mode, variant)`
- 推荐 P95/P99：`histogram_quantile(0.99, sum(rate(zfeed_recommend_stage_duration_seconds_bucket{stage="total"}[5m])) by (le, variant))`
- 各阶段耗时：recall、feature_load、coarse_rank、fine_rank、rerank、build_items。
- 兜底率：`sum(rate(zfeed_recommend_fallback_total[5m])) / sum(rate(zfeed_recommend_requests_total[5m]))`
- 召回量：`sum(rate(zfeed_recommend_recall_items_total[5m])) by (source)`
- 快照命中率：`snapshot_hit / snapshot_lookup`。
- 实验 CTR/IPM：`zfeed_recommend_track_consume_total` 按 `variant` 聚合点击、互动和曝光事件。
- 新内容曝光占比：`new_content_exposure / total_exposure`。
- Kafka consumer lag：画像更新和埋点消费延迟。

### 7.5 Prometheus 告警

推荐迁移链路的告警规则放在 `deploy/prometheus/rules/zfeed-recommend-alerts.yml`，由现有 `rules/*.yml` 自动加载：

- 消费延迟：`zfeed_recommend_track_consume_lag_seconds_bucket` 和 `zfeed_recommend_user_action_consume_lag_seconds_bucket` p95 超过 60 秒持续 10 分钟。
- 消费错误率：`zfeed_recommend_track_consume_total{result=~"parse_error|profile_error|aggregate_error"}` 超过 1% 持续 10 分钟。
- user-action outbox 异常：`zfeed_user_action_outbox_total{result=~"retry|mark_failed"}` 出现持续 10 分钟。
- 推荐兜底率：`zfeed_recommend_fallback_total / zfeed_recommend_requests_total` 超过 20% 持续 15 分钟。

告警表达式继续保持低基数维度，只使用 `event_type`、`source`、`result` 等有限集合，不使用 `user_id`、`content_id`、`target_id`、`event_id` 或 `snapshot_id`。

## 8. 渐进式落地计划

### 阶段 1: 新内容冷启动 + 多样性重排

目标：

- 保留热榜主链路。
- 新增 `feed:rec:new:global`。
- 推荐结果在热榜基础上混入少量新内容，并做同作者/类型打散。

改动：

- `content-rpc` 新增 `recommend` 包：`NewContentRecall`、`CandidateMerge`、`DiversityRerank`。
- 发布成功后写 `feed:rec:new:global` 和 `feed:rec:new:meta:{content_id}`。
- `Recommend.Enabled=false` 默认关闭，灰度开启 `NewContent.Enabled=true`。

验收标准：

- `TestRecommendHotSnapshotE2E` 不变且通过。
- 开关关闭时响应与当前热榜路径一致。
- 开关开启且存在新内容时，前 20 条至少出现配置数量的新内容。
- 同作者连续刷屏明显减少。

回滚策略：

- Redis 设置 `HSET rec:flag:recommend enabled false`。
- 或仅关闭 `recall.new_content.enabled`、`diversity.enabled`。
- 热榜 Lua 和 `feed:hot:global` 不受影响。

### 阶段 2: 用户画像 + 兴趣召回 + 粗排

目标：

- 建立 `rec:user:profile:{user_id}`。
- 增加 `InterestRecall` 和粗排，候选规模控制在 500 -> 200。

改动：

- 新增 `zfeed-user-action` 事件结构；先从 like/favorite/comment Canal 或 interaction outbox 产生。
- 新增 `rec:content:tags:{content_id}`、`rec:tag:index:{tag}`。
- 新增 `ProfileUpdater` 消费者。
- `RecommendFeed` 登录用户开启兴趣召回。

验收标准：

- 用户点赞/收藏带标签内容后，画像标签 1 分钟内更新。
- 兴趣召回 source 占比可在日志或指标中看到。
- 冷启动用户仍稳定回退热榜 + 新内容。
- 粗排单元测试覆盖归一化、截断、已曝光降权。

回滚策略：

- 关闭 `recall.interest.enabled`。
- 保留画像更新消费者不影响推荐主链路。
- 画像 Redis key 可自然 TTL 过期。

### 阶段 3: A/B 实验 + 精排参数动态化 + 个性化快照

目标：

- 实现 `ExperimentResolver`。
- 精排公式参数可调。
- 个性化 snapshot 保证翻页稳定。

改动：

- `content.yaml` 增加 `Recommend` 配置。
- Redis `rec:flag:recommend` 支持动态覆盖。
- 新增 `feed:rec:candidate:{bucket}:{variant}:{config_hash}` 和 `feed:rec:user:snap:{snapshot_id}`。
- 埋点写 `zfeed-rec-track`。

验收标准：

- 同一个 `user_id` 稳定进入同一 variant。
- 同一 `snapshot_id` 翻页不重复、不漂移。
- 配置变更后新请求生效，旧 snapshot 仍按旧 config_hash 翻页。
- Grafana 能看到 variant 维度的请求量、兜底率、耗时。

回滚策略：

- `enabled=false` 一键回热榜。
- 单个实验 `enabled=false` 回默认 variant。
- 删除 `feed:rec:user:snap:*` 不影响热榜。

### 阶段 4: 数据闭环与服务边界演进

目标：

- 把推荐行为数据用于画像和策略优化。
- 如果 `content-rpc` 推荐逻辑膨胀，再抽 `recommend-rpc`。

改动：

- 新增日聚合表和离线任务。
- Grafana 展示 CTR/IPM/新内容曝光占比。
- 可选抽象服务边界：

```text
front-api -> content-rpc FeedService -> recommend-rpc RankService
                                  \-> FeedItemBuilder
```

验收标准：

- 每天能产出实验指标报表。
- 推荐链路各阶段指标完整，异常可定位到召回/排序/缓存/详情补全。
- 抽服务前后 API 协议不变。

回滚策略：

- `content-rpc` 保留本地推荐引擎实现。
- `recommend-rpc` 超时或错误时自动走本地热榜 fallback。

## Test Steps

1. 单元测试：
   - `MergeCandidates` 去重、权重合并、limit 截断。
   - `InterestScore` 空画像、空标签、正常点积。
   - `DiversityRerank` 同作者和内容类型滑窗约束。
   - `ResolveVariant` 同用户稳定分流、流量比例近似正确。
   - `SnapshotResolver` 正确区分 `rec:` 和热榜 snapshot。
2. 集成测试：
   - Redis 写入 `feed:rec:new:global` 后，推荐开启新内容通道能召回。
   - 写入 `rec:user:profile:{user_id}` 和 `rec:tag:index:{tag}` 后，兴趣召回返回匹配内容。
   - 个性化 snapshot 翻页不重复，`snapshot_id` 失效后自动生成新 snapshot 或回退热榜。
3. 回归测试：
   - `go test ./app/rpc/content/internal/logic/feed/...`
   - `go test ./pkg/hotrank/...`
   - Docker 栈启动后运行 `TestRecommendHotSnapshotE2E`，确认热榜基线不破坏。
4. 手工灰度：
   - `HSET rec:flag:recommend enabled false`，确认完全走热榜。
   - `HSET rec:flag:recommend enabled true recall.new_content.weight 0.2`，确认新内容混入。
   - 构造同作者多条内容，确认前 5 条不连续刷屏。

## Acceptance Criteria

- 现有热榜 API、请求/响应字段和 snapshot 翻页语义保持兼容。
- 个性化能力通过 Redis/YAML 开关可灰度、可回滚。
- 至少三路召回可独立开关，并在指标中能看到各自召回量。
- 推荐结果支持稳定分页，旧 snapshot 不受新配置影响。
- 用户画像可由互动事件增量更新，冷启动用户自动回退。
- Grafana 能观察请求量、延迟、兜底率、召回量、实验效果和新内容曝光占比。

## Implementation Progress

### 2026-06-06

已落地：

- `content-rpc` 新增 `internal/recommend` 包，包含候选合并、新内容召回、兴趣召回、画像标签、粗排、精排、多样性重排、个性化 snapshot、实验分流和运行时配置解析。
- `RecommendFeed` 保留热榜 fallback，并接入新内容和兴趣召回、候选合并、粗排、精排、多样性重排、个性化 snapshot 翻页。
- `PublishArticle` / `PublishVideo` 写入 `feed:rec:new:global`、`feed:rec:new:meta:{content_id}`、`rec:content:tags:{content_id}` 和 `rec:tag:index:{tag}`。
- `DeleteContent` 和 `rec.new.cleanup` 覆盖新内容池、meta、content tags 和 tag index 的清理。
- 推荐配置已加入 `content.yaml`，并支持 Redis `rec:flag:recommend` 覆盖全局开关、fallback、三路召回 enabled/weight/limit、粗排 limit、排序权重和多样性参数。
- Redis `rec:flag:recommend` 动态覆盖结果已加 10 秒进程内缓存，避免推荐请求每次读取 Redis。
- 推荐增强链路会按 `TimeoutMs` 派生带 deadline 的 context，并把该 context 传给 Redis/MySQL/item builder 路径。
- 新增推荐指标声明、低基数 label 单测、增强主链路 request/stage/recall/snapshot 记录和 Grafana 推荐面板。
- `ExperimentResolver` 已接入 `RecommendFeed`，同一 `user_id` 会稳定进入同一 variant，snapshot meta 会记录 variant 和 config hash。
- 个性化 snapshot 翻页已记录 lookup stage、hit/miss/error snapshot 事件、snapshot 请求结果和 snapshot_miss/snapshot_error 兜底原因。
- 热榜兜底已细分记录 disabled、cold_start、hot_error 和 build_error 原因，便于定位配置关闭、冷启动空结果、Redis 异常和详情补全失败。
- 推荐候选缓存已落地 `feed:rec:candidate:{bucket}:{variant}:{config_hash}`，增强链路会优先命中缓存并回写合并后的候选池。
- `rec:seen:{user_id}` 已接入排序降权和曝光回写，已曝光内容不会直接消失，但会按 `SeenPenalty` 被降权。
- 行为埋点已接入服务端曝光链路，`content-rpc` 新增 `internal/recommend/track` 事件模型、`KafkaProducer` 和默认 no-op 生产者，`RecommendFeed` 在热榜、个性化快照和增强成功返回时会逐条写出曝光事件。
- `ServiceContext` 已根据 `KqProducerConf` 注入 `zfeed-rec-track` 生产者，配置为空时自动回落到 no-op，避免本地和测试环境硬依赖 Kafka。
- `RecommendFeed` 的曝光写入已补 `zfeed_recommend_track_emit_total{event_type,result}` 观测，能区分曝光事件发射成功和失败。
- 行为埋点 click/dwell 上报入口已落地，`front-api /v1/feed/track` 薄转发、`content-rpc FeedService.EmitRecommendTrack`、事件类型校验、Kafka 生产者写入和对应测试都已接通。
- 兴趣召回画像读取已补 `zfeed_recommend_profile_total{result}` 观测，区分 disabled、skipped、miss、hit 和 error。
- 重排环节已补 `zfeed_recommend_rerank_adjust_total{rule,variant}` 观测，按 author_window 和 type_window 记录多样性重排调整次数。
- 推荐增强链路已补 `zfeed_recommend_error_total{stage,variant}` 观测，按 candidate_cache、recall、feature_load、seen_load、snapshot_save、snapshot_read、build_items 和 seen_write 定位阶段错误。
- 个性化 snapshot 翻页已读取 `feed:rec:user:snapmeta:{snapshot_id}` 的 variant 和 config_hash，旧 snapshot 翻页继续按生成时的实验版本记录请求与曝光，不受新 runtime 配置覆盖。
- click/dwell 客户端埋点上报成功后，会同步调用 `ApplyProfileEvent` 更新 `rec:user:profile:{user_id}`；画像更新失败只记录 `zfeed_recommend_error_total{stage="profile_update",variant}` 和日志，不影响埋点上报响应。
- 推荐客户端埋点白名单已扩展到 `like`、`favorite`、`comment`、`unlike`、`unfavorite`，这些事件写入 `zfeed-rec-track` 成功后会复用同一画像更新路径，按既有权重增量更新用户兴趣标签。
- `front-api` 点赞、取消点赞、收藏、取消收藏、评论写路径已在对应 interaction RPC 成功后复用 `FeedService.EmitRecommendTrack` 发出 `like`、`unlike`、`favorite`、`unfavorite`、`comment` 事件，source 固定为 `interaction`；埋点发射失败只记录日志，不影响原写操作响应。
- 已新增 `zfeed_rec_metric_daily` MySQL 表和 `DailyAggregator` 聚合写入组件，可按 `metric_date + variant_id + source` 累加 exposure、click、dwell、like、favorite、comment 和停留时长。
- `content-rpc` 已新增 `zfeed-rec-track` consumer，消费推荐埋点 JSON 后调用 `DailyAggregator` 写入日聚合表；consumer 配置已加入 `content.yaml`。
- `zfeed-rec-track` consumer 已在日聚合前复用 `ApplyProfileEvent` 更新 `rec:user:profile:{user_id}`，同一 `event_id` 重放依赖既有 profile 事件去重避免重复加权，后续 interaction outbox 或 Canal 只要投递同结构事件即可异步更新画像。
- `zfeed-rec-track` consumer 已兼容 interaction-rpc `zfeed-like` 原始事件，能将 `timestamp` 归一化为秒级 `occurred_at`，并把 `cancel_like` 映射为推荐画像使用的 `unlike`。
- `zfeed-rec-track` consumer 已兼容 `zfeed_favorite_event` 的新增收藏原始事件，能从 `event_id` 末段 UnixNano 推导秒级 `occurred_at`，并以 `Source=interaction` 写入画像和日聚合。
- `zfeed-rec-track` consumer 已兼容 `zfeed_favorite_event` 的取消收藏原始事件，把 `remove_favorite` 映射为推荐画像使用的 `unfavorite`，按 -1.5 权重撤回部分收藏偏好。
- `zfeed-rec-track` consumer 已兼容 `zfeed_comment` 正常未删除评论行，使用 `comment_{user_id}_{content_id}_{comment_id}` 派生幂等 `event_id`，并从 `created_at` 解析 `occurred_at`。
- `zfeed-rec-track` consumer 已兼容统一 user-action JSON 形态，支持通过 `action` 和 `target_id` 归一为既有 `track.Event`，后续 interaction 侧收敛事件生产者时无需改动画像更新和日聚合链路。
- `content-rpc` 已新增 `KqUserActionConsumerConf`，可独立订阅 `zfeed-user-action` topic，并复用同一个 `RecommendTrackConsumer` 写入画像更新和日聚合。
- `interaction-rpc` 已新增统一 user-action 事件模型、`zfeed_user_action_outbox`、Kafka producer 和后台 relay，可向 `zfeed-user-action` 发布 `action/target_id/source/occurred_at` 形态事件。
- `front-api` 已删除点赞、取消点赞、收藏、取消收藏、评论写路径里的同步推荐埋点兼容投递，互动画像统一由 interaction-rpc user-action outbox 异步驱动，避免同一互动行为重复加权。
- `interaction-rpc` 收藏和取消收藏写路径已在状态实际变化后调用 `UserActionProducer` 发出 `favorite` / `unfavorite` user-action；发送失败只记录日志，不回滚收藏业务。
- `interaction-rpc` 点赞和取消点赞写路径已在状态实际变化后调用 `UserActionProducer` 发出 `like` / `unlike` user-action；发送失败只记录日志，不回滚点赞业务。
- `interaction-rpc` 评论写路径已在评论事务提交成功后调用 `UserActionProducer` 发出 `comment` user-action；发送失败只记录日志，不回滚评论业务。
- 2026-06-15 同步点：interaction-rpc 点赞、收藏、评论写路径均已接入 `UserActionProducer`，front-api 兼容投递路径已清理；后续观察 `zfeed-user-action` 消费和画像权重稳定性。
- `content-rpc` 已新增 `zfeed_recommend_user_action_consume_total{event_type,result}`，按低基数 `event_type` 和 `success/parse_error/profile_error/aggregate_error` 记录 `zfeed-user-action` 消费结果，便于迁移后观察互动画像和日聚合链路。
- `interaction-rpc` 已新增 `zfeed_user_action_outbox_total{action,result}`，按低基数 `action` 和 `sent/retry/replayed/mark_failed` 记录 user-action outbox 发送、重试和回放结果，便于定位生产侧是否成功把互动事件交给 Kafka。
- 2026-06-15 观测同步点：user-action 迁移的生产侧 outbox 和消费侧 consumer 均已有低基数指标，后续以运行观察和数据校验为主。
- Grafana overview 已新增 `User Action Outbox Dispatch` 和 `Recommendation User Action Consume` 面板，分别按 `action/result`、`event_type/result` 展示 user-action 生产和消费速率，避免把 `user_id`、`content_id`、`target_id` 放入 PromQL 聚合维度。
- `content-rpc` 已新增 `zfeed_recommend_track_consume_total{event_type,variant,source,result}`，在 `zfeed-rec-track` / `zfeed-user-action` consumer 消费后记录曝光、点击、停留和互动事件的成功、解析失败、画像失败、聚合失败结果，Grafana 可直接按 `variant` 计算 CTR 和 IPM。
- Grafana overview 已新增 `Recommendation CTR` 和 `Recommendation IPM` 面板，基于 `zfeed_recommend_track_consume_total` 展示实验效果。
- `dwell` 画像更新已按 `dwell_ms >= 10000` 过滤，短停留仍会写入埋点和日聚合，但不会给兴趣画像加权。
- `ApplyProfileEvent` 已按 `_updated_at` 对既有 tag 权重执行 `exp(-hours_since_update/168)` 时间衰减，再叠加本次行为权重。
- 个性化 snapshot 已新增内容级召回来源归因，`feed:rec:user:snapsource:{snapshot_id}` 会按 `content_id` 保存主召回来源。服务端曝光事件优先写 `hot/new_content/interest`，老 snapshot 或缺失来源时回落为 `recommend`。
- 推荐候选缓存已补并行来源 Hash，`feed:rec:candidate:{bucket}:{variant}:{config_hash}:source` 与候选 ZSET 使用同一 TTL。缓存命中后会恢复候选主召回来源，避免新内容曝光占比在缓存路径被低估。
- Grafana overview 已新增 `Recommendation New Content Exposure Share` 面板，基于 `zfeed_recommend_track_consume_total{event_type="exposure",source="new_content",result="success"}` 除以成功曝光总量，按 `variant` 展示新内容曝光占比。
- `content-rpc` 已新增 `zfeed_recommend_track_consume_lag_seconds{event_type,source}` 和 `zfeed_recommend_user_action_consume_lag_seconds{event_type}` 直方图，按事件 `occurred_at` 到实际消费时间记录 Kafka 消费延迟；缺失 `occurred_at` 的历史或异常事件会跳过延迟观测，避免产生误导样本。
- Grafana overview 已新增 `Recommendation Track Consume Lag P95` 和 `Recommendation User Action Consume Lag P95` 面板，分别按 `event_type/source` 和 `event_type` 展示 p95 消费延迟，继续避免 `user_id`、`content_id`、`target_id` 等高基数标签。
- Prometheus 已新增 `zfeed-recommend-alerts.yml` 推荐迁移告警规则，覆盖推荐埋点消费延迟、user-action 消费延迟、埋点消费错误率、user-action outbox 重试/失败和推荐兜底率异常；规则测试会校验告警表达式和低基数约束。
- `content-rpc` 已新增 `rec.tag.refresh` XXL-Job，按公开已发布内容分页读取最近内容，复用 `hotrank.Formula` 计算互动热度衰减分，并叠加发布时间 freshness 衰减分作为 base score，再由 `recommend.RefreshContentTagIndex` 按已有 `rec:content:tags:{content_id}` 权重刷新 `rec:tag:index:{tag}` 分数并续 `TagIndexTTL`。
- `recommend.RefreshContentTagIndex` 已把标签索引刷新封装在 recommend 包内，cron 只负责调度、分页和加锁，避免把兴趣召回打分策略散落到任务层。
- `rec.tag.refresh` 已覆盖无互动新内容场景：即使 like/comment/favorite 均为 0，也会按发布时间 freshness 衰减刷新 tag index，防止发布时写入的高时间戳分数长期残留。

已验证：

- `go test ./app/rpc/content/internal/logic/feed -run 'TestRecommend(RuntimeFlagDisablesEnhancement|FineRankUsesConfiguredWeightsInMainPath)' -count=1`
- `go test ./app/front/internal/logic/feed -run 'TestEmitRecommendTrack' -count=1`
- `go test ./app/rpc/content/internal/recommend -run TestLoadRuntimeConfigMergesRedisOverrides -count=1`
- `go test ./app/rpc/content/internal/logic/feed -run TestRecommendRuntimeFlagDisablesHotRecallInEnhancedPath -count=1`
- `go test ./app/rpc/content/internal/logic/feed -run TestRecommendWithTimeoutUsesConfiguredBudget -count=1`
- `go test ./app/rpc/content/internal/logic/feed -run TestRecommendEnhancementRecordsCoreMetrics -count=1`
- `go test ./app/rpc/content/internal/recommend -run TestLoadRuntimeConfigUsesTenSecondCache -count=1`
- `go test ./app/rpc/content/internal/recommend -run 'Test(ApplyExperimentVariantOverridesRecommendConfig|ConfigHashChangesWithExperimentOverrides)' -count=1`
- `go test ./app/rpc/content/internal/logic/feed -run TestRecommendExperimentVariantWritesSnapshotMetaAndMetrics -count=1`
- `go test ./app/rpc/content/internal/logic/feed -run 'TestRecommendPersonalizedSnapshot(RecordsHitMetric|RecordsMissAndErrorMetrics)' -count=1`
- `go test ./app/rpc/content/internal/logic/feed -run TestRecommendHotFallbackRecordsReasonMetrics -count=1`
- `go test ./app/rpc/content/internal/recommend -run 'Test(BuildCandidateCacheKeyUsesBucketVariantAndConfigHash|CandidateCacheSaveAndLoad|LoadCandidateCacheMiss)' -count=1`
- `go test ./app/rpc/content/internal/logic/feed -run TestRecommendEnhancementUsesCandidateCache -count=1`
- `go test ./app/rpc/content/internal/recommend -run 'Test(RecordSeenContentsWritesRecentExposureSet|LoadSeenCountsReturnsOnlyRequestedIDs)' -count=1`
- `go test ./app/rpc/content/internal/logic/feed -run TestRecommendEnhancementAppliesAndRecordsSeen -count=1`
- `go test ./app/rpc/content/internal/logic/feed -run TestRecommendEnhancementEmitsExposureTrackEvents -count=1`
- `go test ./app/rpc/content/internal/logic/feed -run 'TestEmitRecommendTrack(EmitsClickAndDwellEvents|RejectsInvalidEvent)' -count=1`
- `go test ./app/rpc/content/internal/recommend/track -count=1`
- `go test ./app/rpc/content/internal/config -count=1`
- `go test ./app/rpc/content/internal/recommend/track ./app/rpc/content/internal/config ./app/rpc/content/internal/logic/feed -count=1`
- `go test ./app/rpc/content/internal/logic/feed -run 'TestRecommendMetric|TestRecommendExposureTrackEventsRecordEmitMetrics' -count=1`
- `go test ./app/rpc/content/internal/logic/feed -run 'TestRecommendMetric|TestRecommendInterestRecallRecordsProfileMetrics' -count=1`
- `go test ./app/rpc/content/internal/recommend ./app/rpc/content/internal/logic/feed -count=1`
- `go test ./app/rpc/content/internal/logic/feed -run 'TestRecommendMetric|TestRecommendEnhancementRecordsRecallErrorMetric' -count=1`
- `go test ./app/rpc/content/internal/recommend ./app/rpc/content/internal/logic/feed -run 'TestLoadPersonalizedSnapshotMetaReadsStoredVariantAndConfigHash|TestRecommendPersonalizedSnapshotUsesSnapshotMetaVariant|TestRecommendEnhancementUsesPersonalizedSnapshotForNextPage' -count=1`
- `go test ./app/rpc/content/internal/logic/feed -count=1`
- `go test ./app/rpc/content/internal/recommend -count=1`
- `go test ./app/rpc/content/internal/recommend ./app/rpc/content/internal/logic/feed -count=1`
- `go test ./app/front/internal/logic/feed ./app/rpc/content/internal/logic/feed ./app/rpc/content/internal/recommend/track -count=1`
- `go test ./app/rpc/content/internal/logic/content ./app/rpc/content/internal/cron/rec_new_cleanup -count=1`
- `go test ./app/rpc/content/internal/logic/feed -run 'TestEmitRecommendTrack|TestRecommendMetric' -count=1`
- `go test ./app/rpc/content/internal/recommend -run 'Test.*Profile|Test.*Tags' -count=1`
- `go test ./app/rpc/content/internal/logic/feed -run TestEmitRecommendTrack -count=1`
- `go test ./app/rpc/content/internal/recommend/track -run TestIsClientEventTypeAllowsInteractionEvents -count=1`
- `go test ./app/rpc/content/internal/recommend/track -run TestDailyAggregator -count=1`
- `go test ./app/rpc/content/internal/recommend/track -count=1`
- `go test ./app/rpc/content/internal/mq/consumer -count=1`
- `go test ./app/rpc/content/internal/config -count=1`
- `go test ./app/rpc/content ./app/rpc/content/internal/svc ./app/rpc/content/internal/mq/consumer -count=1`
- `go test ./app/front/internal/logic/interaction -run 'Test(Favorite|Comment)EmitsRecommendTrackAfterSuccess' -count=1`
- `go test ./app/rpc/content/internal/mq/consumer -run TestRecommendTrackConsumerAppliesProfileEvent -count=1`
- `go test ./app/rpc/content/internal/mq/consumer -count=1`
- `go test ./app/rpc/content/internal/logic/feed -run 'TestEmitRecommendTrack|TestRecommendMetric' -count=1`
- `go test ./app/rpc/content/internal/recommend -run 'Test.*Profile|Test.*Tags' -count=1`
- `go test ./app/rpc/content/internal/recommend/track -run TestIsClientEventTypeAllowsInteractionEvents/unlike -count=1`
- `go test ./app/rpc/content/internal/logic/feed -run TestEmitRecommendTrackUpdatesUserProfileAfterSuccessfulEmit/unlike -count=1`
- `go test ./app/front/internal/logic/interaction -run TestUnlikeEmitsRecommendTrackAfterSuccess -count=1`
- `go test ./app/rpc/content/internal/logic/feed -run 'TestEmitRecommendTrack(SkipsShortDwellProfileUpdate|UpdatesUserProfileAfterSuccessfulEmit/dwell)' -count=1`
- `go test ./app/rpc/content/internal/mq/consumer -run TestRecommendTrackConsumerAppliesProfileEvent -count=1`
- `go test ./app/rpc/content/internal/recommend -run TestApplyProfileEventDecaysExistingWeight -count=1`
- `go test ./app/rpc/content/internal/mq/consumer -run 'TestRecommendTrackConsumer(NormalizesInteractionLikeEvent|MapsCancelLikeToUnlike)' -count=1`
- `go test ./app/rpc/content/internal/mq/consumer -count=1`
- `go test ./app/rpc/content/internal/mq/consumer -run TestRecommendTrackConsumerNormalizesFavoriteEventRow -count=1`
- `go test ./app/rpc/content/internal/mq/consumer -run TestRecommendTrackConsumerNormalizesCommentRow -count=1`
- `go test ./app/rpc/content/internal/recommend/track -run TestIsClientEventTypeAllowsInteractionEvents/unfavorite -count=1`
- `go test ./app/rpc/content/internal/logic/feed -run 'TestEmitRecommendTrack(EmitsClientEvents|UpdatesUserProfileAfterSuccessfulEmit)/unfavorite' -count=1`
- `go test ./app/front/internal/logic/interaction -run TestRemoveFavoriteEmitsRecommendTrackAfterSuccess -count=1`
- `go test ./app/rpc/content/internal/mq/consumer -run TestRecommendTrackConsumerMapsRemoveFavoriteToUnfavorite -count=1`
- `go test ./app/rpc/content/internal/mq/consumer -run TestRecommendTrackConsumerNormalizesUserActionEvent -count=1`
- `go test ./app/rpc/content/internal/config ./app/rpc/content/internal/mq/consumer -run 'TestEnv|TestRecommendConsumerConfigsIncludesUserActionTopic' -count=1`
- `go test ./app/rpc/interaction/internal/config ./app/rpc/interaction/internal/mq/producer -run 'TestInteractionConfigLoadsWithEnv|Test(SendUserAction|DispatchDueUserActions)' -count=1`
- `go test ./app/front/internal/config ./app/front/internal/logic/interaction -run 'TestFrontConfigLoadsWithEnv|TestLikeSkipsRecommendTrackWhenDisabled' -count=1`
- `go test ./app/rpc/interaction/internal/logic/favorite -run 'TestFavoriteAndRemoveFavorite_UpdateDBAndCache|TestFavoriteIgnoresUserActionFailure' -count=1`
- `go test ./app/rpc/interaction/internal/logic/like -run 'Test(Like|Unlike)(EmitsUserAction|IgnoresUserAction)' -count=1`
- `go test ./app/rpc/interaction/internal/logic/like -count=1`
- `go test ./app/rpc/interaction/... -count=1`
- `go test ./app/rpc/interaction/internal/logic/comment -run 'TestComment(EmitsUserActionAfterSuccessfulCreate|IgnoresUserActionFailure)' -count=1`
- `go test ./app/rpc/interaction/internal/logic/comment -count=1`
- `go test ./app/front/internal/config -run TestFrontConfigLoadsWithEnv -count=1`
- `go test ./app/front/internal/config ./app/front/internal/logic/interaction -run 'TestFrontConfigLoadsWithEnv|TestLikeSkipsRecommendTrackWhenDisabled' -count=1`
- `go test ./app/front/... -count=1`
- `go test ./app/rpc/content/internal/mq/consumer -run 'TestRecommend(UserActionConsumeMetric|TrackConsumerRecordsUserActionConsumeMetrics)' -count=1`
- `go test ./app/rpc/content/internal/mq/consumer -count=1`
- `go test ./app/rpc/content/internal/config ./app/rpc/content/internal/mq/consumer -run 'TestEnv|TestRecommendConsumerConfigsIncludesUserActionTopic|TestRecommendTrackConsumer' -count=1`
- `go test ./app/rpc/content/... -count=1`
- `go test ./app/rpc/interaction/internal/mq/producer -run 'Test(UserActionOutboxMetric|SendUserActionRecordsOutboxMetrics|DispatchDueUserActionsRecordsOutboxMetrics)' -count=1`
- `go test ./app/rpc/interaction/internal/mq/producer -count=1`
- `go test ./app/rpc/interaction/... -count=1`
- `go test ./deploy/grafana/dashboards -run TestZFeedOverviewIncludesUserActionMigrationPanels -count=1`
- `go test ./app/rpc/content/internal/mq/consumer -run 'TestRecommendTrack(ConsumeMetric|ConsumerRecordsTrackConsumeMetrics)' -count=1`
- `go test ./deploy/grafana/dashboards -run 'TestZFeedOverviewIncludes(ExperimentEffect|UserActionMigration)Panels' -count=1`
- `go test ./app/rpc/content/internal/recommend -run 'TestPersonalizedSnapshotStoresContentSources|TestLoadPersonalizedSnapshotMetaReadsStoredVariantAndConfigHash|TestCandidateCachePreservesPrimarySources|TestCandidateCacheSaveAndLoad|TestLoadCandidateCacheMiss' -count=1`
- `go test ./app/rpc/content/internal/logic/feed -run TestRecommendEnhancementEmitsExposureTrackEvents -count=1`
- `go test ./deploy/grafana/dashboards -run TestZFeedOverviewIncludesExperimentEffectPanels -count=1`
- `go test ./app/front/internal/logic/interaction -run 'Test(Like|Unlike|Favorite|RemoveFavorite|Comment)DoesNotEmitRecommendTrackAfterSuccess|TestLikeDoesNotDependOnRecommendTrack' -count=1`
- `go test ./app/front/internal/config -run TestFrontConfigLoadsWithEnv -count=1`
- `go test ./app/rpc/content/internal/mq/consumer -run 'TestRecommend(UserActionConsumeMetricLabelsExcludeHighCardinalityIDs|TrackConsumerRecordsConsumeLagMetrics)' -count=1`
- `go test ./app/rpc/content/internal/mq/consumer -count=1`
- `go test ./deploy/grafana/dashboards -run TestZFeedOverviewIncludesRecommendationConsumeLagPanels -count=1`
- `go test ./deploy/grafana/dashboards -count=1`
- `go test ./deploy/prometheus/rules -run TestZfeedRecommendAlertsCoverMigrationRisks -count=1`
- `go test ./deploy/prometheus/rules -count=1`
- `go test ./app/rpc/content/internal/recommend -run 'TestRefreshContentTagIndex' -count=1`
- `go test ./app/rpc/content/internal/cron/rec_tag_refresh -run TestRunRefreshesTagIndexForFreshContentWithoutInteractions -count=1`
- `go test ./app/rpc/content/internal/cron/rec_tag_refresh -count=1`
- `go test ./app/rpc/content/internal/recommend -count=1`
- `go test ./app/rpc/content/internal/cron/rec_tag_refresh ./app/rpc/content/internal/cron/rec_new_cleanup -count=1`

剩余缺口：

- 行为埋点 `zfeed-rec-track` 已完成曝光事件模型、Kafka 生产者、主链路曝光写入、click/dwell/like/favorite/comment 客户端上报入口、画像同步更新，以及 content-rpc 日聚合 consumer。
- 画像更新已有 `ApplyProfileEvent`，推荐埋点入口已接入 click/dwell/like/favorite/comment/unlike/unfavorite，`zfeed-rec-track` consumer 也能异步更新画像；interaction-rpc `like/cancel_like`、`favorite/remove_favorite`、`comment` 原始事件和统一 user-action JSON 均已兼容，`content-rpc` 也能独立消费 `zfeed-user-action` 并记录消费结果和消费延迟指标，interaction-rpc 侧统一 outbox/producer 基础设施已就绪且已补 outbox 发送/回放指标，点赞/取消点赞、收藏/取消收藏、评论写路径均已接入，front-api 兼容投递路径已删除，`rec.tag.refresh` 也能周期刷新兴趣召回 tag index，并已覆盖无互动内容 freshness 衰减刷新，Grafana overview 和 Prometheus 告警覆盖 user-action 生产、消费速率、消费延迟、消费错误、实验 CTR/IPM 以及新内容曝光占比。后续需要继续观察迁移后的 `zfeed-user-action` 消费、画像增量和日聚合数据。

## Change Log

| Date       | Version | Description | Author |
|------------|---------|-------------|--------|
| 2026-05-31 | 1.0.0   | Initial draft | Codex |
| 2026-06-06 | 1.0.0   | Move spec under `specs/recommend/` entrypoint | Codex |
| 2026-06-06 | 1.0.0   | Record implementation progress, verification evidence, and remaining gaps | Codex |
| 2026-06-06 | 1.0.0   | Add runtime hot recall switch verification, timeout budget, and core recommend metrics progress | Codex |
| 2026-06-14 | 1.0.0   | Add 10-second in-process cache for Redis recommend runtime config | Codex |
| 2026-06-14 | 1.0.0   | Wire experiment resolver into recommend feed and snapshot metadata | Codex |
| 2026-06-14 | 1.0.0   | Sync current recommend implementation progress record | Codex |
| 2026-06-14 | 1.0.0   | Add personalized snapshot lookup metrics progress | Codex |
| 2026-06-14 | 1.0.0   | Add hot fallback reason metrics progress | Codex |
| 2026-06-14 | 1.0.0   | Add candidate cache progress | Codex |
| 2026-06-14 | 1.0.0   | Add seen penalty progress | Codex |
| 2026-06-14 | 1.0.0   | Record in-progress recommendation track event work | Codex |
| 2026-06-14 | 1.0.0   | Add recommendation exposure track emission and Kafka producer wiring | Codex |
| 2026-06-14 | 1.0.0   | Add recommendation track emit metric and tests | Codex |
| 2026-06-14 | 1.0.0   | Add recommend track click/dwell entry implementation and tests | Codex |
| 2026-06-14 | 1.0.0   | Add recommendation profile metric and interest recall tests | Codex |
| 2026-06-14 | 1.0.0   | Add recommendation rerank adjustment metric and diversity rerank tests | Codex |
| 2026-06-14 | 1.0.0   | Add recommendation enhancement error stage metric and tests | Codex |
| 2026-06-14 | 1.0.0   | Add personalized snapshot meta variant lookup and config hash isolation tests | Codex |
| 2026-06-15 | 1.0.0   | Update user profile from successful click/dwell recommend track events | Codex |
| 2026-06-15 | 1.0.0   | Allow like/favorite/comment recommend track events to update user profile | Codex |
| 2026-06-15 | 1.0.0   | Add recommendation daily metric table and aggregator | Codex |
| 2026-06-15 | 1.0.0   | Consume recommendation track events into daily metric aggregator | Codex |
| 2026-06-15 | 1.0.0   | Emit like recommendation track event from front-api like path | Codex |
| 2026-06-15 | 1.0.0   | Emit favorite and comment recommendation track events from front-api interaction paths | Codex |
| 2026-06-15 | 1.0.0   | Apply user profile updates from recommendation track consumer | Codex |
| 2026-06-15 | 1.0.0   | Add unlike recommendation track and profile decrement support | Codex |
| 2026-06-15 | 1.0.0   | Gate dwell profile updates at ten seconds | Codex |
| 2026-06-15 | 1.0.0   | Apply time decay to recommendation user profile weights | Codex |
| 2026-06-15 | 1.0.0   | Normalize interaction like raw events in recommendation track consumer | Codex |
| 2026-06-15 | 1.0.0   | Normalize favorite raw events in recommendation track consumer | Codex |
| 2026-06-15 | 1.0.0   | Normalize comment raw rows in recommendation track consumer | Codex |
| 2026-06-15 | 1.0.0   | Add unfavorite recommendation feedback support | Codex |
| 2026-06-15 | 1.0.0   | Normalize unified user-action events in recommendation track consumer | Codex |
| 2026-06-15 | 1.0.0   | Add dedicated user-action topic consumer config for content-rpc | Codex |
| 2026-06-15 | 1.0.0   | Add interaction user-action outbox producer infrastructure | Codex |
| 2026-06-15 | 1.0.0   | Add front-api recommend interaction track migration switch | Codex |
| 2026-06-15 | 1.0.0   | Emit favorite user-action events from interaction-rpc write paths | Codex |
| 2026-06-15 | 1.0.0   | Sync recommend implementation checkpoint after favorite user-action wiring | Codex |
| 2026-06-15 | 1.0.0   | Emit like user-action events from interaction-rpc write paths | Codex |
| 2026-06-15 | 1.0.0   | Emit comment user-action events from interaction-rpc write path | Codex |
| 2026-06-15 | 1.0.0   | Enable front-api recommend interaction track migration switch | Codex |
| 2026-06-15 | 1.0.0   | Add user-action consume metrics for recommendation migration observability | Codex |
| 2026-06-15 | 1.0.0   | Add user-action outbox metrics for recommendation migration observability | Codex |
| 2026-06-15 | 1.0.0   | Sync recommendation migration closeout progress checkpoint | Codex |
| 2026-06-15 | 1.0.0   | Add Grafana panels for user-action migration observability | Codex |
| 2026-06-15 | 1.0.0   | Add recommendation track consume metrics for CTR and IPM dashboards | Codex |
| 2026-06-15 | 1.0.0   | Add exposure source attribution and new-content exposure share dashboard | Codex |
| 2026-06-15 | 1.0.0   | Remove front-api fallback recommendation interaction track emission | Codex |
| 2026-06-15 | 1.0.0   | 同步推荐迁移进度和剩余观测缺口 | Codex |
| 2026-06-15 | 1.0.0   | Add recommendation consume lag metrics and Grafana panels | Codex |
| 2026-06-15 | 1.0.0   | Add recommendation migration Prometheus alerts | Codex |
| 2026-06-15 | 1.0.0   | Add periodic recommendation tag index refresh job | Codex |
| 2026-06-15 | 1.0.0   | 补充推荐标签索引无互动内容 freshness 衰减刷新记录 | Codex |
