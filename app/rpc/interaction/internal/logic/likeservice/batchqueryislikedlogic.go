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

type BatchQueryIsLikedLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	likeRepo repositories.LikeRepository
}

func NewBatchQueryIsLikedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchQueryIsLikedLogic {
	return &BatchQueryIsLikedLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		Logger:   logx.WithContext(ctx),
		likeRepo: repositories.NewLikeRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *BatchQueryIsLikedLogic) BatchQueryIsLiked(in *interaction.BatchQueryIsLikedReq) (*interaction.BatchQueryIsLikedRes, error) {
	if in == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}

	normalized := normalizeLikeInfos(in.GetLikeInfos())
	if len(normalized) == 0 {
		return &interaction.BatchQueryIsLikedRes{
			IsLikedInfos: []*interaction.IsLikedInfo{},
		}, nil
	}

	contentIDs := make([]int64, 0, len(normalized))
	for _, item := range normalized {
		contentIDs = append(contentIDs, item.contentID)
	}

	likedMap := map[int64]bool{}
	if in.GetUserId() > 0 {
		var err error
		likedMap, err = l.loadBatchLikedState(in.GetUserId(), contentIDs)
		if err != nil {
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询点赞信息失败"))
		}
	}

	items := make([]*interaction.IsLikedInfo, 0, len(normalized))
	for _, item := range normalized {
		items = append(items, &interaction.IsLikedInfo{
			ContentId: item.contentID,
			Scene:     item.scene,
			IsLiked:   likedMap[item.contentID],
		})
	}

	return &interaction.BatchQueryIsLikedRes{
		IsLikedInfos: items,
	}, nil
}

type normalizedLikeInfo struct {
	contentID int64
	scene     interaction.Scene
}

func normalizeLikeInfos(items []*interaction.LikeInfo) []normalizedLikeInfo {
	result := make([]normalizedLikeInfo, 0, len(items))
	seen := make(map[string]struct{}, len(items))

	for _, item := range items {
		if item == nil || item.GetContentId() <= 0 || item.GetScene() == interaction.Scene_SCENE_UNKNOWN {
			continue
		}

		key := strconv.FormatInt(item.GetContentId(), 10) + ":" + item.GetScene().String()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		result = append(result, normalizedLikeInfo{
			contentID: item.GetContentId(),
			scene:     item.GetScene(),
		})
	}

	return result
}

func (l *BatchQueryIsLikedLogic) loadBatchLikedState(userID int64, contentIDs []int64) (map[int64]bool, error) {
	result := make(map[int64]bool, len(contentIDs))
	if userID <= 0 || len(contentIDs) == 0 {
		return result, nil
	}

	userLikeKey := rediskey.BuildLikeUserKey(strconv.FormatInt(userID, 10))
	fields := make([]string, 0, len(contentIDs))
	for _, contentID := range contentIDs {
		fields = append(fields, strconv.FormatInt(contentID, 10))
	}

	missingIDs := make([]int64, 0)
	cacheValues, err := l.svcCtx.Redis.HmgetCtx(l.ctx, userLikeKey, fields...)
	if err != nil {
		l.Errorf("batch query like relation cache failed, key=%s, err=%v", userLikeKey, err)
		missingIDs = append(missingIDs, contentIDs...)
	} else {
		for index, contentID := range contentIDs {
			if index < len(cacheValues) && cacheValues[index] == "1" {
				result[contentID] = true
				continue
			}
			missingIDs = append(missingIDs, contentID)
		}
	}

	if len(missingIDs) == 0 {
		return result, nil
	}

	dbMap, err := l.likeRepo.BatchIsLiked(userID, missingIDs)
	if err != nil {
		return nil, err
	}

	for _, contentID := range missingIDs {
		if !dbMap[contentID] {
			continue
		}
		result[contentID] = true
		if setErr := l.svcCtx.Redis.HsetCtx(l.ctx, userLikeKey, strconv.FormatInt(contentID, 10), "1"); setErr != nil {
			l.Errorf("rebuild like relation cache failed, key=%s, content_id=%d, err=%v", userLikeKey, contentID, setErr)
		}
	}

	return result, nil
}
