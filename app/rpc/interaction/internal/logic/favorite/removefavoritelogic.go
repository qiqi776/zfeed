package favoritelogic

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"zfeed/app/rpc/interaction/interaction"
	rediskey "zfeed/app/rpc/interaction/internal/common/consts/redis"
	"zfeed/app/rpc/interaction/internal/model"
	"zfeed/app/rpc/interaction/internal/mq/event"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type RemoveFavoriteLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	favoriteRepo      repositories.FavoriteRepository
	favoriteEventRepo repositories.FavoriteEventRepository
	contentRepo       repositories.ContentRepository
	commentRepo       repositories.CommentRepository
}

func NewRemoveFavoriteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveFavoriteLogic {
	return &RemoveFavoriteLogic{
		ctx:               ctx,
		svcCtx:            svcCtx,
		Logger:            logx.WithContext(ctx),
		favoriteRepo:      repositories.NewFavoriteRepository(ctx, svcCtx.MysqlDb),
		favoriteEventRepo: repositories.NewFavoriteEventRepository(ctx, svcCtx.MysqlDb),
		contentRepo:       repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		commentRepo:       repositories.NewCommentRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *RemoveFavoriteLogic) RemoveFavorite(in *interaction.RemoveFavoriteReq) (*interaction.RemoveFavoriteRes, error) {
	if in == nil || in.GetUserId() <= 0 || in.GetContentId() <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}
	if in.GetScene() == interaction.Scene_SCENE_UNKNOWN {
		return nil, errorx.NewBadRequest("场景参数错误")
	}

	contentUserID, err := resolveFavoriteTargetOwner(l.contentRepo, l.commentRepo, in.GetScene(), in.GetContentId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询内容作者失败"))
	}
	if contentUserID <= 0 {
		return nil, errorx.NewNotFound("内容不存在")
	}

	var removed bool
	if err := l.svcCtx.MysqlDb.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		favoriteRepo := l.favoriteRepo.WithTx(tx)
		favoriteEventRepo := l.favoriteEventRepo.WithTx(tx)

		row, err := favoriteRepo.GetByUserAndTarget(in.GetUserId(), int32(in.GetScene()), in.GetContentId())
		if err != nil {
			return err
		}
		if row == nil {
			return nil
		}
		contentUserID = row.ContentUserID

		if _, err := favoriteRepo.DeleteByUserAndTarget(in.GetUserId(), int32(in.GetScene()), in.GetContentId()); err != nil {
			return err
		}
		removed = true

		now := time.Now()
		return favoriteEventRepo.Create(&model.ZfeedFavoriteEvent{
			EventID:       fmt.Sprintf("remove_favorite_%d_%d_%d", in.GetUserId(), in.GetContentId(), now.UnixNano()),
			EventType:     "remove_favorite",
			Scene:         int32(in.GetScene()),
			UserID:        in.GetUserId(),
			ContentID:     in.GetContentId(),
			ContentUserID: contentUserID,
			CreatedAt:     now,
		})
	}); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("取消收藏失败"))
	}

	scene := in.GetScene().String()
	contentIDStr := strconv.FormatInt(in.GetContentId(), 10)
	userIDStr := strconv.FormatInt(in.GetUserId(), 10)

	relKey := rediskey.BuildFavoriteRelKey(scene, userIDStr, contentIDStr)
	if _, delErr := l.svcCtx.Redis.DelCtx(l.ctx, relKey); delErr != nil {
		l.Errorf("delete favorite relation cache failed: %v", delErr)
	}

	if removed {
		emitUserAction(
			l.ctx,
			l.Logger,
			l.svcCtx.UserActionProducer,
			event.UserActionUnfavorite,
			in.GetUserId(),
			in.GetContentId(),
			contentUserID,
			in.GetScene(),
		)
	}

	if shouldUpdateFavoriteFeed(in.GetScene()) {
		favKey := rediskey.BuildUserFavoriteFeedKey(userIDStr)
		if _, zremErr := l.svcCtx.Redis.ZremCtx(l.ctx, favKey, contentIDStr); zremErr != nil {
			l.Errorf("remove favorite feed cache failed, user_id=%d, content_id=%d, err=%v", in.GetUserId(), in.GetContentId(), zremErr)
		}
	}

	return &interaction.RemoveFavoriteRes{}, nil
}
