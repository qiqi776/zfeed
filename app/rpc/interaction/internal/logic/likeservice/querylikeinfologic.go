package likeservicelogic

import (
	"context"
	"strconv"

	"zfeed/app/rpc/interaction/interaction"
	rediskey "zfeed/app/rpc/interaction/internal/common/consts/redis"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type QueryLikeInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	likeRepo repositories.LikeRepository
}

func NewQueryLikeInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryLikeInfoLogic {
	return &QueryLikeInfoLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		Logger:   logx.WithContext(ctx),
		likeRepo: repositories.NewLikeRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *QueryLikeInfoLogic) QueryLikeInfo(in *interaction.QueryLikeInfoReq) (*interaction.QueryLikeInfoRes, error) {
	if in == nil || in.GetContentId() <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}
	if in.GetScene() == interaction.Scene_SCENE_UNKNOWN {
		return nil, errorx.NewBadRequest("场景参数错误")
	}

	likeCount, err := l.likeRepo.CountByContentID(in.GetContentId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询点赞信息失败"))
	}

	isLiked := false
	if in.GetUserId() > 0 {
		isLiked, err = l.queryIsLiked(in.GetUserId(), in.GetContentId())
		if err != nil {
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询点赞信息失败"))
		}
	}

	return &interaction.QueryLikeInfoRes{
		LikeCount: likeCount,
		IsLiked:   isLiked,
		ContentId: in.GetContentId(),
		Scene:     in.GetScene(),
	}, nil
}

func (l *QueryLikeInfoLogic) queryIsLiked(userID, contentID int64) (bool, error) {
	userLikeKey := rediskey.BuildLikeUserKey(strconv.FormatInt(userID, 10))
	contentIDStr := strconv.FormatInt(contentID, 10)

	if exists, err := l.svcCtx.Redis.HexistsCtx(l.ctx, userLikeKey, contentIDStr); err == nil {
		if exists {
			return true, nil
		}
	} else {
		l.Errorf("query like relation cache failed, key=%s, field=%s, err=%v", userLikeKey, contentIDStr, err)
	}

	isLiked, err := l.likeRepo.IsLiked(userID, contentID)
	if err != nil {
		return false, err
	}
	if isLiked {
		if setErr := l.svcCtx.Redis.HsetCtx(l.ctx, userLikeKey, contentIDStr, "1"); setErr != nil {
			l.Errorf("rebuild like relation cache failed, key=%s, field=%s, err=%v", userLikeKey, contentIDStr, setErr)
		}
	}

	return isLiked, nil
}
