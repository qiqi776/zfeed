package contentlogic

import (
	"context"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"

	contentpb "zfeed/app/rpc/content/content"
	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	luautils "zfeed/app/rpc/content/internal/common/utils/lua"
	"zfeed/app/rpc/content/internal/repositories"
	"zfeed/app/rpc/content/internal/svc"
	followservice "zfeed/app/rpc/interaction/client/followservice"
	"zfeed/pkg/errorx"
)

type CleanupFollowInboxLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo repositories.ContentRepository
}

func NewCleanupFollowInboxLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CleanupFollowInboxLogic {
	return &CleanupFollowInboxLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *CleanupFollowInboxLogic) CleanupFollowInbox(in *contentpb.CleanupFollowInboxReq) (*contentpb.CleanupFollowInboxRes, error) {
	if in == nil || in.GetFollowerId() <= 0 || in.GetFolloweeId() <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}

	inboxKey := redisconsts.BuildFollowInboxKey(in.GetFollowerId())
	members, err := l.svcCtx.Redis.ZrangeCtx(l.ctx, inboxKey, 0, -1)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关注收件箱失败"))
	}
	if len(members) == 0 {
		return &contentpb.CleanupFollowInboxRes{}, nil
	}

	contentIDs, memberByID := parseInboxMembers(members)
	if len(contentIDs) == 0 {
		return &contentpb.CleanupFollowInboxRes{}, nil
	}

	authorByID, err := l.contentRepo.BatchGetAuthorsByIDs(contentIDs)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关注内容失败"))
	}

	removeMembers := make([]string, 0)
	for _, contentID := range contentIDs {
		if authorByID[contentID] != in.GetFolloweeId() {
			continue
		}
		removeMembers = append(removeMembers, memberByID[contentID])
	}
	if len(removeMembers) == 0 {
		return &contentpb.CleanupFollowInboxRes{}, nil
	}

	stillFollowing, err := l.isStillFollowing(in.GetFollowerId(), in.GetFolloweeId())
	if err != nil {
		return nil, err
	}
	if stillFollowing {
		return &contentpb.CleanupFollowInboxRes{Skipped: true}, nil
	}

	removed, err := l.removeInboxMembers(inboxKey, removeMembers)
	if err != nil {
		return nil, err
	}

	return &contentpb.CleanupFollowInboxRes{RemovedCount: int32(removed)}, nil
}

func parseInboxMembers(members []string) ([]int64, map[int64]string) {
	contentIDs := make([]int64, 0, len(members))
	memberByID := make(map[int64]string, len(members))
	seen := make(map[int64]struct{}, len(members))

	for _, member := range members {
		if member == "" {
			continue
		}
		contentID, err := strconv.ParseInt(member, 10, 64)
		if err != nil || contentID <= 0 {
			continue
		}
		if _, ok := seen[contentID]; ok {
			continue
		}
		seen[contentID] = struct{}{}
		contentIDs = append(contentIDs, contentID)
		memberByID[contentID] = member
	}
	return contentIDs, memberByID
}

func (l *CleanupFollowInboxLogic) isStillFollowing(followerID, followeeID int64) (bool, error) {
	if l.svcCtx.FollowRpc == nil {
		return false, errorx.NewMsg("查询关注关系失败")
	}

	resp, err := l.svcCtx.FollowRpc.BatchQueryFollowing(l.ctx, &followservice.BatchQueryFollowingReq{
		UserId:        followerID,
		FollowUserIds: []int64{followeeID},
	})
	if err != nil {
		return false, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关注关系失败"))
	}
	if resp == nil {
		return false, nil
	}
	for _, item := range resp.GetItems() {
		if item != nil && item.GetUserId() == followeeID {
			return item.GetIsFollowing(), nil
		}
	}
	return false, nil
}

func (l *CleanupFollowInboxLogic) removeInboxMembers(inboxKey string, members []string) (int64, error) {
	args := make([]any, 0, len(members))
	for _, member := range members {
		if member == "" {
			continue
		}
		args = append(args, member)
	}
	if len(args) == 0 {
		return 0, nil
	}

	result, err := l.svcCtx.Redis.EvalCtx(l.ctx, luautils.CleanupFollowInboxZSetScript, []string{inboxKey}, args...)
	if err != nil {
		return 0, errorx.Wrap(l.ctx, err, errorx.NewMsg("清理关注收件箱失败"))
	}

	removed, _ := result.(int64)
	return removed, nil
}
