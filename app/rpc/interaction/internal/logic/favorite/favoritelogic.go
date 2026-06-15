package favoritelogic

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"zfeed/app/rpc/interaction/interaction"
	rediskey "zfeed/app/rpc/interaction/internal/common/consts/redis"
	luautils "zfeed/app/rpc/interaction/internal/common/utils/lua"
	"zfeed/app/rpc/interaction/internal/do"
	"zfeed/app/rpc/interaction/internal/model"
	"zfeed/app/rpc/interaction/internal/mq/event"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type FavoriteLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	favoriteRepo      repositories.FavoriteRepository
	favoriteEventRepo repositories.FavoriteEventRepository
	contentRepo       repositories.ContentRepository
	commentRepo       repositories.CommentRepository
}

func NewFavoriteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FavoriteLogic {
	return &FavoriteLogic{
		ctx:               ctx,
		svcCtx:            svcCtx,
		Logger:            logx.WithContext(ctx),
		favoriteRepo:      repositories.NewFavoriteRepository(ctx, svcCtx.MysqlDb),
		favoriteEventRepo: repositories.NewFavoriteEventRepository(ctx, svcCtx.MysqlDb),
		contentRepo:       repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		commentRepo:       repositories.NewCommentRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *FavoriteLogic) Favorite(in *interaction.FavoriteReq) (*interaction.FavoriteRes, error) {
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

	var rowID int64
	var changed bool
	err = l.svcCtx.MysqlDb.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		favoriteRepo := l.favoriteRepo.WithTx(tx)
		favoriteEventRepo := l.favoriteEventRepo.WithTx(tx)

		existingRow, err := favoriteRepo.GetByUserAndTarget(in.GetUserId(), int32(in.GetScene()), in.GetContentId())
		if err != nil {
			return err
		}
		if existingRow != nil {
			rowID = existingRow.ID
			return nil
		}

		if err := favoriteRepo.Upsert(&do.FavoriteDO{
			UserID:        in.GetUserId(),
			Scene:         int32(in.GetScene()),
			ContentID:     in.GetContentId(),
			ContentUserID: contentUserID,
			Status:        repositories.FavoriteStatusActive,
			CreatedBy:     in.GetUserId(),
			UpdatedBy:     in.GetUserId(),
		}); err != nil {
			return err
		}

		now := time.Now()
		row, err := favoriteRepo.GetByUserAndTarget(in.GetUserId(), int32(in.GetScene()), in.GetContentId())
		if err != nil {
			return err
		}
		if row != nil {
			rowID = row.ID
		}
		changed = true

		return favoriteEventRepo.Create(&model.ZfeedFavoriteEvent{
			EventID:       fmt.Sprintf("favorite_%d_%d_%d", in.GetUserId(), in.GetContentId(), now.UnixNano()),
			EventType:     "favorite",
			Scene:         int32(in.GetScene()),
			UserID:        in.GetUserId(),
			ContentID:     in.GetContentId(),
			ContentUserID: contentUserID,
			CreatedAt:     now,
		})
	})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("收藏失败"))
	}

	scene := in.GetScene().String()
	contentIDStr := strconv.FormatInt(in.GetContentId(), 10)
	userIDStr := strconv.FormatInt(in.GetUserId(), 10)

	relKey := rediskey.BuildFavoriteRelKey(scene, userIDStr, contentIDStr)
	if _, delErr := l.svcCtx.Redis.DelCtx(l.ctx, relKey); delErr != nil {
		l.Errorf("delete favorite relation cache failed: %v", delErr)
	}

	if changed {
		emitUserAction(
			l.ctx,
			l.Logger,
			l.svcCtx.UserActionProducer,
			event.UserActionFavorite,
			in.GetUserId(),
			in.GetContentId(),
			contentUserID,
			in.GetScene(),
		)
	}

	if !shouldUpdateFavoriteFeed(in.GetScene()) || rowID <= 0 {
		return &interaction.FavoriteRes{}, nil
	}

	favKey := rediskey.BuildUserFavoriteFeedKey(userIDStr)
	if _, evalErr := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.AddUserFavoriteIfExistsScript,
		[]string{favKey},
		strconv.FormatInt(rowID, 10),
		contentIDStr,
		strconv.FormatInt(5000, 10),
	); evalErr != nil {
		l.Errorf("update favorite feed cache failed, user_id=%d, content_id=%d, err=%v", in.GetUserId(), in.GetContentId(), evalErr)
	}

	return &interaction.FavoriteRes{}, nil
}

func resolveFavoriteTargetOwner(
	contentRepo repositories.ContentRepository,
	commentRepo repositories.CommentRepository,
	scene interaction.Scene,
	contentID int64,
) (int64, error) {
	switch scene {
	case interaction.Scene_ARTICLE, interaction.Scene_VIDEO:
		return contentRepo.GetAuthorID(contentID)
	case interaction.Scene_COMMENT:
		commentDO, err := commentRepo.GetByID(contentID)
		if err != nil {
			return 0, err
		}
		if commentDO == nil {
			return 0, nil
		}
		return commentDO.UserID, nil
	default:
		return 0, nil
	}
}

func shouldUpdateFavoriteFeed(scene interaction.Scene) bool {
	return scene == interaction.Scene_ARTICLE || scene == interaction.Scene_VIDEO
}
