package followservicelogic

import (
	"context"
	"time"

	contentservice "zfeed/app/rpc/content/contentservice"
	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/do"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
	userpb "zfeed/app/rpc/user/user"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

const (
	backfillFollowInboxLimit   = 20
	backfillFollowInboxTimeout = 3 * time.Second
)

type FollowUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	followRepo repositories.FollowRepository
}

func NewFollowUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FollowUserLogic {
	return &FollowUserLogic{
		ctx:        ctx,
		svcCtx:     svcCtx,
		Logger:     logx.WithContext(ctx),
		followRepo: repositories.NewFollowRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *FollowUserLogic) FollowUser(in *interaction.FollowUserReq) (*interaction.FollowUserRes, error) {
	if in == nil || in.GetUserId() <= 0 || in.GetFollowUserId() <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	if in.GetUserId() == in.GetFollowUserId() {
		return nil, errorx.NewMsg("不能关注自己")
	}

	if err := l.ensureTargetUserExists(in.GetFollowUserId()); err != nil {
		return nil, err
	}

	err := l.followRepo.Upsert(&do.FollowDO{
		UserID:       in.GetUserId(),
		FollowUserID: in.GetFollowUserId(),
		Status:       repositories.FollowStatusFollow,
		CreatedBy:    in.GetUserId(),
		UpdatedBy:    in.GetUserId(),
	})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("关注失败"))
	}

	if l.svcCtx.ContentRpc != nil {
		threading.GoSafe(func() {
			ctx, cancel := context.WithTimeout(context.Background(), backfillFollowInboxTimeout)
			defer cancel()

			_, callErr := l.svcCtx.ContentRpc.BackfillFollowInbox(ctx, &contentservice.BackfillFollowInboxReq{
				FollowerId: in.GetUserId(),
				FolloweeId: in.GetFollowUserId(),
				Limit:      backfillFollowInboxLimit,
			})
			if callErr != nil {
				l.Errorf("backfill follow inbox failed, follower_id=%d, followee_id=%d, err=%v", in.GetUserId(), in.GetFollowUserId(), callErr)
			}
		})
	}

	return &interaction.FollowUserRes{IsFollowed: true}, nil
}

func (l *FollowUserLogic) ensureTargetUserExists(userID int64) error {
	if l.svcCtx.UserRpc == nil || userID <= 0 {
		return nil
	}

	resp, err := l.svcCtx.UserRpc.GetUser(l.ctx, &userpb.GetUserReq{UserId: userID})
	if err != nil {
		return errorx.Wrap(l.ctx, err, errorx.NewMsg("查询用户失败"))
	}
	if resp.GetUserInfo() == nil || resp.GetUserInfo().GetUserId() <= 0 {
		return errorx.NewMsg("用户不存在")
	}
	return nil
}
