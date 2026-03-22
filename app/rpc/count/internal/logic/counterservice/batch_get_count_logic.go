package counterservicelogic

import (
	"context"
	"strconv"

	"ran-feed/app/rpc/count/count"
	"ran-feed/app/rpc/count/internal/repositories"
	"ran-feed/app/rpc/count/internal/svc"
	"ran-feed/pkg/errorx"

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
		return nil, errorx.NewMsg("批量查询计数请求无效")
	}
	if len(in.Keys) == 0 {
		return &count.BatchGetCountRes{}, nil
	}

	infos := make([]batchCountKeyInfo, 0, len(in.Keys))
	uniqueInfoByMapKey := make(map[string]batchCountKeyInfo, len(in.Keys))
	uniqueKeys := make([]string, 0, len(in.Keys))
	for _, key := range in.Keys {
		if key == nil ||
			key.BizType == count.BizType_BIZ_TYPE_UNKNOWN ||
			key.TargetType == count.TargetType_TARGET_TYPE_UNKNOWN ||
			key.TargetId <= 0 {
			return nil, errorx.NewMsg("批量查询计数请求无效")
		}

		info := batchCountKeyInfo{
			key:      key,
			cacheKey: buildCountValueCacheKey(key.BizType, key.TargetType, key.TargetId),
			mapKey:   buildCountValueMapKey(key.BizType, key.TargetType, key.TargetId),
		}
		infos = append(infos, info)

		if _, exists := uniqueInfoByMapKey[info.mapKey]; !exists {
			uniqueInfoByMapKey[info.mapKey] = info
			uniqueKeys = append(uniqueKeys, info.mapKey)
		}
	}

	valueByMapKey, missMapKeys := l.batchLoadFromCache(uniqueInfoByMapKey, uniqueKeys)
	if len(missMapKeys) > 0 {
		dbValueByMapKey := l.batchLoadFromDB(uniqueInfoByMapKey, missMapKeys)
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

type batchGroup struct {
	bizType    count.BizType
	targetType count.TargetType
}

func (l *BatchGetCountLogic) batchLoadFromCache(uniqueInfoByMapKey map[string]batchCountKeyInfo, uniqueKeys []string) (map[string]int64, []string) {
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
		l.Errorf("批量查询计数缓存失败: keys=%v, err=%v", cacheKeys, err)
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
			l.Errorf("解析批量计数缓存失败: key=%s, value=%s, err=%v",
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
) map[string]int64 {
	valueByMapKey := make(map[string]int64, len(missMapKeys))
	if len(missMapKeys) == 0 {
		return valueByMapKey
	}

	groupIDs := make(map[batchGroup][]int64)
	groupMapKey := make(map[batchGroup]map[int64]string)
	for _, mapKey := range missMapKeys {
		info := uniqueInfoByMapKey[mapKey]
		group := batchGroup{
			bizType:    info.key.BizType,
			targetType: info.key.TargetType,
		}
		groupIDs[group] = append(groupIDs[group], info.key.TargetId)
		if _, ok := groupMapKey[group]; !ok {
			groupMapKey[group] = make(map[int64]string)
		}
		groupMapKey[group][info.key.TargetId] = mapKey
	}

	for group, targetIDs := range groupIDs {
		rows, err := l.countRepo.BatchGet(int32(group.bizType), int32(group.targetType), targetIDs)
		if err != nil {
			l.Errorf("批量查询计数DB失败: bizType=%d, targetType=%d, targetIDs=%v, err=%v",
				group.bizType, group.targetType, targetIDs, err)
			for _, targetID := range targetIDs {
				valueByMapKey[groupMapKey[group][targetID]] = 0
			}
			continue
		}

		for _, targetID := range targetIDs {
			mapKey := groupMapKey[group][targetID]
			if row, ok := rows[targetID]; ok && row != nil {
				valueByMapKey[mapKey] = row.Value
				continue
			}
			valueByMapKey[mapKey] = 0
		}
	}

	return valueByMapKey
}

func (l *BatchGetCountLogic) batchWriteCache(uniqueInfoByMapKey map[string]batchCountKeyInfo, valueByMapKey map[string]int64) {
	for mapKey, value := range valueByMapKey {
		info, ok := uniqueInfoByMapKey[mapKey]
		if !ok {
			continue
		}
		if err := l.svcCtx.Redis.SetexCtx(
			l.ctx,
			info.cacheKey,
			strconv.FormatInt(value, 10),
			countCacheExpireSecondsWithJitter(),
		); err != nil {
			l.Errorf("批量回填计数缓存失败: key=%s, value=%d, err=%v", info.cacheKey, value, err)
		}
	}
}
