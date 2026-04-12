package likeservicelogic

import (
	"context"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"

	"zfeed/app/rpc/interaction/interaction"
	rediskey "zfeed/app/rpc/interaction/internal/common/consts/redis"
	luautils "zfeed/app/rpc/interaction/internal/common/utils/lua"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/pkg/errorx"
)

type LikeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLikeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LikeLogic {
	return &LikeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *LikeLogic) Like(in *interaction.LikeReq) (*interaction.LikeRes, error) {
	if in == nil || in.GetUserId() <= 0 || in.GetContentId() <= 0 || in.GetContentUserId() <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}
	if in.GetScene() == interaction.Scene_SCENE_UNKNOWN {
		return nil, errorx.NewBadRequest("场景参数错误")
	}

	changed, err := l.processLike(in.GetUserId(), in.GetContentId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("点赞处理失败"))
	}
	if changed {
		scene := in.GetScene().String()
		threading.GoSafe(func() {
			l.publishLikeEvent(in.GetUserId(), in.GetContentId(), in.GetContentUserId(), scene)
		})
	}

	return &interaction.LikeRes{}, nil
}

func (l *LikeLogic) processLike(userID, contentID int64) (changed bool, err error) {
	contentIDStr := strconv.FormatInt(contentID, 10)
	userIDStr := strconv.FormatInt(userID, 10)
	userLikeKey := rediskey.BuildLikeUserKey(userIDStr)

	resultVal, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.LikeUserHashScript,
		[]string{userLikeKey},
		contentIDStr,
		strconv.FormatInt(rediskey.RedisLikeExpireSeconds, 10),
	)
	if err != nil {
		return false, err
	}

	arr, ok := resultVal.([]interface{})
	if !ok || len(arr) < 2 {
		return false, errorx.NewMsg("解析点赞脚本返回值失败")
	}

	changedVal, _ := arr[0].(int64)
	return changedVal == 1, nil
}

func (l *LikeLogic) publishLikeEvent(userID, contentID, contentUserID int64, scene string) {
	l.svcCtx.LikeProducer.SendLikeEvent(l.ctx, userID, contentID, contentUserID, scene)
}
