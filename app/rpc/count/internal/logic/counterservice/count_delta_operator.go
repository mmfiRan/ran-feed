package counterservicelogic

import (
	"context"
	"time"

	"ran-feed/app/rpc/count/count"
	"ran-feed/app/rpc/count/internal/repositories"
	"ran-feed/app/rpc/count/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

// CountDeltaOperator 封装计数增减与缓存失效的通用逻辑。
type CountDeltaOperator struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	countRepo repositories.CountValueRepository
}

const delayedCacheInvalidateDelay = 200 * time.Millisecond

func NewCountDeltaOperator(ctx context.Context, svcCtx *svc.ServiceContext) *CountDeltaOperator {
	return &CountDeltaOperator{
		ctx:       ctx,
		svcCtx:    svcCtx,
		Logger:    logx.WithContext(ctx),
		countRepo: repositories.NewCountValueRepository(ctx, svcCtx.MysqlDb),
	}
}

// UpdateDeltaOnly 仅更新DB，不处理缓存
func (o *CountDeltaOperator) UpdateDeltaOnly(bizType count.BizType, targetType count.TargetType, targetID int64, delta int64, updatedAt time.Time) error {
	return o.UpdateDeltaOnlyWithRepo(o.countRepo, bizType, targetType, targetID, delta, updatedAt)
}

// UpdateDeltaOnlyWithRepo 允许调用方传入带事务的repo
func (o *CountDeltaOperator) UpdateDeltaOnlyWithRepo(repo repositories.CountValueRepository, bizType count.BizType, targetType count.TargetType, targetID int64, delta int64, updatedAt time.Time) error {
	if bizType == count.BizType_BIZ_TYPE_UNKNOWN || targetType == count.TargetType_TARGET_TYPE_UNKNOWN || targetID <= 0 || delta == 0 {
		return nil
	}
	_, err := repo.UpdateDelta(int32(bizType), int32(targetType), targetID, delta, updatedAt)
	return err
}

// UpdateDeltaOnlyWithOwner 仅更新DB（携带owner_id），不处理缓存
func (o *CountDeltaOperator) UpdateDeltaOnlyWithOwner(
	bizType count.BizType,
	targetType count.TargetType,
	targetID int64,
	ownerID int64,
	delta int64,
	updatedAt time.Time,
) error {
	return o.UpdateDeltaOnlyWithRepoAndOwner(o.countRepo, bizType, targetType, targetID, ownerID, delta, updatedAt)
}

// UpdateDeltaOnlyWithRepoAndOwner 允许调用方传入带事务的repo，并携带owner_id
func (o *CountDeltaOperator) UpdateDeltaOnlyWithRepoAndOwner(
	repo repositories.CountValueRepository,
	bizType count.BizType,
	targetType count.TargetType,
	targetID int64,
	ownerID int64,
	delta int64,
	updatedAt time.Time,
) error {
	if bizType == count.BizType_BIZ_TYPE_UNKNOWN || targetType == count.TargetType_TARGET_TYPE_UNKNOWN || targetID <= 0 || delta == 0 {
		return nil
	}
	if ownerID <= 0 {
		_, err := repo.UpdateDelta(int32(bizType), int32(targetType), targetID, delta, updatedAt)
		return err
	}
	_, err := repo.UpdateDeltaWithOwner(int32(bizType), int32(targetType), targetID, ownerID, delta, updatedAt)
	return err
}

// InvalidateCountCache 旁路缓存策略：写成功后删除缓存
func (o *CountDeltaOperator) InvalidateCountCache(bizType count.BizType, targetType count.TargetType, targetID int64) {
	if bizType == count.BizType_BIZ_TYPE_UNKNOWN || targetType == count.TargetType_TARGET_TYPE_UNKNOWN || targetID <= 0 {
		return
	}
	cacheKey := buildCountValueCacheKey(bizType, targetType, targetID)
	if _, err := o.svcCtx.Redis.DelCtx(o.ctx, cacheKey); err != nil {
		o.Errorf("删除计数缓存失败: key=%s, biz_type=%d, target_type=%d, target_id=%d, err=%v",
			cacheKey, bizType, targetType, targetID, err)
	}

	go func(cacheKey string, bizType count.BizType, targetType count.TargetType, targetID int64) {
		time.Sleep(delayedCacheInvalidateDelay)
		if _, err := o.svcCtx.Redis.DelCtx(context.Background(), cacheKey); err != nil {
			o.Errorf("延迟删除计数缓存失败: key=%s, biz_type=%d, target_type=%d, target_id=%d, err=%v",
				cacheKey, bizType, targetType, targetID, err)
		}
	}(cacheKey, bizType, targetType, targetID)
}
