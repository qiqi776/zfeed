package likelogic

import (
	"context"
	"strconv"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchIsLikedLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	likeRepo repositories.LikeRepository
}

func NewBatchIsLikedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchIsLikedLogic {
	return &BatchIsLikedLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		Logger:   logx.WithContext(ctx),
		likeRepo: repositories.NewLikeRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *BatchIsLikedLogic) BatchIsLiked(in *interaction.BatchIsLikedReq) (*interaction.BatchIsLikedRes, error) {
	if in == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}

	normalized := normalizeInfos(in.GetLikeInfos())
	if len(normalized) == 0 {
		return &interaction.BatchIsLikedRes{
			IsLikedInfos: []*interaction.IsLikedInfo{},
		}, nil
	}

	likedMap := map[string]bool{}
	if in.GetUserId() > 0 {
		var err error
		likedMap, err = l.loadLikedMap(in.GetUserId(), normalized)
		if err != nil {
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询点赞信息失败"))
		}
	}

	items := make([]*interaction.IsLikedInfo, 0, len(normalized))
	for _, item := range normalized {
		items = append(items, &interaction.IsLikedInfo{
			ContentId: item.contentID,
			Scene:     item.scene,
			IsLiked:   likedMap[item.key()],
		})
	}

	return &interaction.BatchIsLikedRes{
		IsLikedInfos: items,
	}, nil
}

type normalizedLikeInfo struct {
	contentID int64
	scene     interaction.Scene
}

func (i normalizedLikeInfo) key() string {
	return likeTargetKey(i.scene, i.contentID)
}

func (i normalizedLikeInfo) repoTarget() repositories.LikeTarget {
	return repositories.LikeTarget{
		Scene:     int32(i.scene),
		ContentID: i.contentID,
	}
}

func normalizeInfos(items []*interaction.LikeInfo) []normalizedLikeInfo {
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

func (l *BatchIsLikedLogic) loadLikedMap(userID int64, infos []normalizedLikeInfo) (map[string]bool, error) {
	result := make(map[string]bool, len(infos))
	if userID <= 0 || len(infos) == 0 {
		return result, nil
	}

	userLikeKey := likeCacheKey(userID)
	fields := make([]string, 0, len(infos))
	for _, info := range infos {
		fields = append(fields, likeTargetKey(info.scene, info.contentID))
	}

	missingTargets := make([]normalizedLikeInfo, 0)
	cacheValues, err := l.svcCtx.Redis.HmgetCtx(l.ctx, userLikeKey, fields...)
	if err != nil {
		l.Errorf("batch query like relation cache failed, key=%s, err=%v", userLikeKey, err)
		missingTargets = append(missingTargets, infos...)
	} else {
		for index, info := range infos {
			if index < len(cacheValues) {
				if isLiked, ok := parseLikeCacheValue(cacheValues[index]); ok {
					if isLiked {
						result[info.key()] = true
					}
					continue
				}
				if cacheValues[index] == "" {
					missingTargets = append(missingTargets, info)
					continue
				}
			}
			missingTargets = append(missingTargets, info)
		}
	}

	if len(missingTargets) == 0 {
		return result, nil
	}

	targets := make([]repositories.LikeTarget, 0, len(missingTargets))
	for _, info := range missingTargets {
		targets = append(targets, info.repoTarget())
	}

	dbMap, err := l.likeRepo.BatchIsLiked(userID, targets)
	if err != nil {
		return nil, err
	}

	for _, info := range missingTargets {
		field := likeTargetKey(info.scene, info.contentID)
		mapKey := info.key()
		isLiked := dbMap[mapKey]
		if isLiked {
			result[mapKey] = true
		}

		if setErr := cacheLikeState(l.ctx, l.svcCtx.Redis, userLikeKey, field, isLiked); setErr != nil {
			l.Errorf("rebuild like relation cache failed, key=%s, field=%s, err=%v", userLikeKey, field, setErr)
		}
	}

	return result, nil
}
