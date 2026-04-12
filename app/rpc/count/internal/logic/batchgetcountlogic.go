package logic

import (
	"context"
	"strconv"
	"strings"

	"zfeed/app/rpc/count/count"
	redisconsts "zfeed/app/rpc/count/internal/common/consts/redis"
	"zfeed/app/rpc/count/internal/repositories"
	"zfeed/app/rpc/count/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchGetCountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	countRepo repositories.CountValueRepository
}

func NewBatchGetCountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchGetCountLogic {
	return &BatchGetCountLogic{
		ctx:       ctx,
		svcCtx:    svcCtx,
		Logger:    logx.WithContext(ctx),
		countRepo: repositories.NewCountValueRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *BatchGetCountLogic) BatchGetCount(in *count.BatchGetCountReq) (*count.BatchGetCountRes, error) {
	if in == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}
	if len(in.GetKeys()) == 0 {
		return &count.BatchGetCountRes{}, nil
	}

	infos := make([]batchCountKeyInfo, 0, len(in.GetKeys()))
	uniqueInfoByMapKey := make(map[string]batchCountKeyInfo, len(in.GetKeys()))
	uniqueKeys := make([]string, 0, len(in.GetKeys()))
	for _, key := range in.GetKeys() {
		if key == nil || key.GetTargetId() <= 0 {
			return nil, errorx.NewBadRequest("参数错误")
		}
		if key.GetBizType() == count.BizType_BIZ_TYPE_UNKNOWN || key.GetTargetType() == count.TargetType_TARGET_TYPE_UNKNOWN {
			return nil, errorx.NewBadRequest("参数错误")
		}

		info := batchCountKeyInfo{
			key:      key,
			cacheKey: buildCountValueCacheKey(key.GetBizType(), key.GetTargetType(), key.GetTargetId()),
			mapKey:   buildCountValueMapKey(key.GetBizType(), key.GetTargetType(), key.GetTargetId()),
		}
		infos = append(infos, info)
		if _, exists := uniqueInfoByMapKey[info.mapKey]; !exists {
			uniqueInfoByMapKey[info.mapKey] = info
			uniqueKeys = append(uniqueKeys, info.mapKey)
		}
	}

	valueByMapKey, missMapKeys := l.batchLoadFromCache(uniqueInfoByMapKey, uniqueKeys)
	if len(missMapKeys) > 0 {
		dbValueByMapKey, err := l.batchLoadFromDB(uniqueInfoByMapKey, missMapKeys)
		if err != nil {
			return nil, err
		}
		for mapKey, value := range dbValueByMapKey {
			valueByMapKey[mapKey] = value
		}
		l.batchWriteCache(uniqueInfoByMapKey, dbValueByMapKey)
	}

	items := make([]*count.CountValueItem, 0, len(infos))
	for _, info := range infos {
		items = append(items, &count.CountValueItem{
			Key:   info.key,
			Value: valueByMapKey[info.mapKey],
		})
	}
	return &count.BatchGetCountRes{Items: items}, nil
}

type batchCountKeyInfo struct {
	key      *count.CountKey
	cacheKey string
	mapKey   string
}

func (l *BatchGetCountLogic) batchLoadFromCache(
	uniqueInfoByMapKey map[string]batchCountKeyInfo,
	uniqueKeys []string,
) (map[string]int64, []string) {
	valueByMapKey := make(map[string]int64, len(uniqueInfoByMapKey))
	if len(uniqueKeys) == 0 {
		return valueByMapKey, nil
	}

	cacheKeys := make([]string, 0, len(uniqueKeys))
	for _, mapKey := range uniqueKeys {
		cacheKeys = append(cacheKeys, uniqueInfoByMapKey[mapKey].cacheKey)
	}

	cacheValues, err := l.svcCtx.Redis.MgetCtx(l.ctx, cacheKeys...)
	if err != nil {
		l.Errorf("mget count cache failed, keys=%v, err=%v", cacheKeys, err)
		return valueByMapKey, uniqueKeys
	}

	missMapKeys := make([]string, 0, len(uniqueKeys))
	for i, mapKey := range uniqueKeys {
		if i >= len(cacheValues) || cacheValues[i] == "" {
			missMapKeys = append(missMapKeys, mapKey)
			continue
		}

		value, parseErr := strconv.ParseInt(cacheValues[i], 10, 64)
		if parseErr != nil {
			l.Errorf("parse batch count cache failed, key=%s, value=%s, err=%v",
				uniqueInfoByMapKey[mapKey].cacheKey, cacheValues[i], parseErr)
			missMapKeys = append(missMapKeys, mapKey)
			continue
		}
		valueByMapKey[mapKey] = value
	}

	return valueByMapKey, missMapKeys
}

func (l *BatchGetCountLogic) batchLoadFromDB(
	uniqueInfoByMapKey map[string]batchCountKeyInfo,
	missMapKeys []string,
) (map[string]int64, error) {
	valueByMapKey := make(map[string]int64, len(missMapKeys))
	if len(missMapKeys) == 0 {
		return valueByMapKey, nil
	}

	groupedTargetIDs := make(map[string][]int64)
	groupKeyMap := make(map[string]map[int64]string)
	for _, mapKey := range missMapKeys {
		info := uniqueInfoByMapKey[mapKey]
		groupKey := fmtGroupKey(info.key.GetBizType(), info.key.GetTargetType())
		groupedTargetIDs[groupKey] = append(groupedTargetIDs[groupKey], info.key.GetTargetId())
		if _, ok := groupKeyMap[groupKey]; !ok {
			groupKeyMap[groupKey] = make(map[int64]string)
		}
		groupKeyMap[groupKey][info.key.GetTargetId()] = mapKey
	}

	for groupKey, targetIDs := range groupedTargetIDs {
		bizType, targetType := parseGroupKey(groupKey)
		rows, err := l.countRepo.BatchGet(int32(bizType), int32(targetType), targetIDs)
		if err != nil {
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("批量查询计数失败"))
		}
		for _, targetID := range targetIDs {
			mapKey := groupKeyMap[groupKey][targetID]
			value := int64(0)
			if row, ok := rows[targetID]; ok && row != nil {
				value = row.Value
			}
			valueByMapKey[mapKey] = value
		}
	}

	return valueByMapKey, nil
}

func (l *BatchGetCountLogic) batchWriteCache(
	uniqueInfoByMapKey map[string]batchCountKeyInfo,
	valueByMapKey map[string]int64,
) {
	for mapKey, value := range valueByMapKey {
		info, ok := uniqueInfoByMapKey[mapKey]
		if !ok {
			continue
		}
		if err := l.svcCtx.Redis.SetexCtx(
			l.ctx,
			info.cacheKey,
			strconv.FormatInt(value, 10),
			countCacheExpireSecondsWithJitter(redisconsts.RedisCountValueExpireSeconds),
		); err != nil {
			l.Errorf("set batch count cache failed, key=%s, err=%v", info.cacheKey, err)
		}
	}
}

func fmtGroupKey(bizType count.BizType, targetType count.TargetType) string {
	return strconv.FormatInt(int64(bizType), 10) + ":" + strconv.FormatInt(int64(targetType), 10)
}

func parseGroupKey(groupKey string) (count.BizType, count.TargetType) {
	parts := strings.Split(groupKey, ":")
	if len(parts) != 2 {
		return count.BizType_BIZ_TYPE_UNKNOWN, count.TargetType_TARGET_TYPE_UNKNOWN
	}
	bizType, _ := strconv.ParseInt(parts[0], 10, 32)
	targetType, _ := strconv.ParseInt(parts[1], 10, 32)
	return count.BizType(bizType), count.TargetType(targetType)
}
