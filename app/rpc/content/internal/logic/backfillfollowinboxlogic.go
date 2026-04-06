package logic

import (
	"context"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"
	zredis "github.com/zeromicro/go-zero/core/stores/redis"

	contentpb "zfeed/app/rpc/content/content"
	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	luautils "zfeed/app/rpc/content/internal/common/utils/lua"
	"zfeed/app/rpc/content/internal/repositories"
	"zfeed/app/rpc/content/internal/svc"
	"zfeed/pkg/errorx"
)

const (
	backfillFollowInboxDefaultLimit = 20
	backfillFollowInboxMaxLimit     = 50
)

type BackfillFollowInboxLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo repositories.ContentRepository
}

type inboxCandidate struct {
	score  string
	member string
}

func NewBackfillFollowInboxLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BackfillFollowInboxLogic {
	return &BackfillFollowInboxLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *BackfillFollowInboxLogic) BackfillFollowInbox(in *contentpb.BackfillFollowInboxReq) (*contentpb.BackfillFollowInboxRes, error) {
	if in == nil || in.GetFollowerId() <= 0 || in.GetFolloweeId() <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	limit := int(in.GetLimit())
	if limit <= 0 {
		limit = backfillFollowInboxDefaultLimit
	}
	if limit > backfillFollowInboxMaxLimit {
		limit = backfillFollowInboxMaxLimit
	}

	candidates, err := l.loadFolloweeLatest(in.GetFolloweeId(), limit)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return &contentpb.BackfillFollowInboxRes{AddedCount: 0}, nil
	}

	addedCount, err := l.updateInbox(redisconsts.BuildFollowInboxKey(in.GetFollowerId()), candidates)
	if err != nil {
		return nil, err
	}

	return &contentpb.BackfillFollowInboxRes{AddedCount: int32(addedCount)}, nil
}

func (l *BackfillFollowInboxLogic) loadFolloweeLatest(followeeID int64, limit int) ([]inboxCandidate, error) {
	publishKey := redisconsts.BuildUserPublishKey(followeeID)
	exists, err := l.svcCtx.Redis.ExistsCtx(l.ctx, publishKey)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询发布流失败"))
	}
	if exists {
		pairs, err := l.svcCtx.Redis.ZrevrangeWithScoresByFloatCtx(l.ctx, publishKey, 0, int64(limit-1))
		if err != nil {
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询发布流失败"))
		}
		return buildCandidatesFromPairs(pairs), nil
	}

	ids, err := l.contentRepo.ListLatestPublishedIDsByAuthor(followeeID, limit)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询发布流失败"))
	}

	candidates := make([]inboxCandidate, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		value := strconv.FormatInt(id, 10)
		candidates = append(candidates, inboxCandidate{
			score:  value,
			member: value,
		})
	}
	return candidates, nil
}

func buildCandidatesFromPairs(pairs []zredis.FloatPair) []inboxCandidate {
	if len(pairs) == 0 {
		return nil
	}

	result := make([]inboxCandidate, 0, len(pairs))
	for _, pair := range pairs {
		if pair.Key == "" {
			continue
		}

		score := pair.Key
		if _, err := strconv.ParseInt(pair.Key, 10, 64); err != nil {
			score = strconv.FormatInt(int64(pair.Score), 10)
		}

		result = append(result, inboxCandidate{
			score:  score,
			member: pair.Key,
		})
	}

	return result
}

func (l *BackfillFollowInboxLogic) updateInbox(inboxKey string, candidates []inboxCandidate) (int, error) {
	args := make([]any, 0, 1+len(candidates)*2)
	args = append(args, strconv.Itoa(redisconsts.RedisFollowInboxKeepLatestN))
	for _, candidate := range candidates {
		if candidate.member == "" || candidate.score == "" {
			continue
		}
		args = append(args, candidate.score, candidate.member)
	}

	result, err := l.svcCtx.Redis.EvalCtx(l.ctx, luautils.BackfillFollowInboxZSetScript, []string{inboxKey}, args...)
	if err != nil {
		return 0, errorx.Wrap(l.ctx, err, errorx.NewMsg("回填关注收件箱失败"))
	}

	addedCount, _ := result.(int64)
	return int(addedCount), nil
}
