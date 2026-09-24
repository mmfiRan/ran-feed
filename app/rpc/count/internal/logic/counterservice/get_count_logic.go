package counterservicelogic

import (
	"context"
	"strconv"

	"ran-feed/app/rpc/count/count"
	rediskey "ran-feed/app/rpc/count/internal/common/consts/redis"
	countenum "ran-feed/app/rpc/count/internal/common/enums"
	"ran-feed/app/rpc/count/internal/repositories"
	"ran-feed/app/rpc/count/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	redislock "github.com/zeromicro/go-zero/core/stores/redis"
)

type GetCountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	countRepo repositories.CountValueRepository
}

func NewGetCountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCountLogic {
	return &GetCountLogic{
		ctx:       ctx,
		svcCtx:    svcCtx,
		Logger:    logx.WithContext(ctx),
		countRepo: repositories.NewCountValueRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *GetCountLogic) GetCount(in *count.GetCountReq) (*count.GetCountRes, error) {
	if in == nil {
		return nil, errorx.NewMsg("查询计数请求无效")
	}
	if in.BizType == count.BizType_BIZ_TYPE_UNSPECIFIED ||
		in.TargetType == count.TargetType_TARGET_TYPE_UNSPECIFIED {
		return nil, errorx.NewMsg("查询计数请求无效")
	}

	cacheKey := buildCountValueCacheKey(countenum.BizTypeEnum(in.BizType), countenum.TargetTypeEnum(in.TargetType), in.TargetId)

	cacheValue, cacheResult := l.queryFromCache(cacheKey)
	if cacheResult == cacheHit {
		return &count.GetCountRes{
			Value: cacheValue,
		}, nil
	}

	value, err := l.rebuildCacheWithLock(in, cacheKey)
	if err != nil {
		return nil, err
	}
	return &count.GetCountRes{
		Value: value,
	}, nil
}

type cacheQueryResult int

const (
	cacheHit cacheQueryResult = iota
	cacheMiss
	cacheError
	rebuildLockExpireSeconds = 30
)

func (l *GetCountLogic) queryFromCache(cacheKey string) (int64, cacheQueryResult) {
	cacheStr, err := l.svcCtx.Redis.GetCtx(l.ctx, cacheKey)
	if err != nil {
		l.Errorf("查询计数缓存失败: key=%s, err=%v", cacheKey, err)
		return 0, cacheError
	}
	if cacheStr == "" {
		return 0, cacheMiss
	}

	value, parseErr := strconv.ParseInt(cacheStr, 10, 64)
	if parseErr != nil {
		l.Errorf("解析计数缓存失败: key=%s, value=%s, err=%v", cacheKey, cacheStr, parseErr)
		return 0, cacheError
	}
	return value, cacheHit
}

func (l *GetCountLogic) rebuildCacheWithLock(in *count.GetCountReq, cacheKey string) (int64, error) {
	lockKey := rediskey.GetRedisPrefixKey(rediskey.RedisCountRebuildLockPrefix,
		strconv.FormatInt(int64(in.BizType), 10)+":"+
			strconv.FormatInt(int64(in.TargetType), 10)+":"+
			strconv.FormatInt(in.TargetId, 10))
	lock := redislock.NewRedisLock(l.svcCtx.Redis, lockKey)
	lock.SetExpire(rebuildLockExpireSeconds)

	lockAcquired, err := lock.AcquireCtx(l.ctx)
	if err != nil {
		// 锁获取失败直接降级查库，保证可用性
		l.Errorf("获取计数重建锁失败，降级查库: lock_key=%s, err=%v", lockKey, err)
		return l.queryFromDB(in)
	}

	if !lockAcquired {
		// 别人正在重建 不阻塞请求 直接回源查库
		return l.queryFromDB(in)
	}

	defer func() {
		if releaseOk, releaseErr := lock.ReleaseCtx(l.ctx); !releaseOk || releaseErr != nil {
			l.Errorf("释放计数重建锁失败: lock_key=%s, err=%v", lockKey, releaseErr)
		}
	}()

	// 双重检查，避免重复回源
	if value, cacheResult := l.queryFromCache(cacheKey); cacheResult == cacheHit {
		return value, nil
	}

	value, err := l.queryFromDB(in)
	if err != nil {
		return 0, err
	}

	if cacheErr := l.svcCtx.Redis.SetexCtx(
		l.ctx,
		cacheKey,
		strconv.FormatInt(value, 10),
		countCacheExpireSecondsWithJitter(),
	); cacheErr != nil {
		l.Errorf("重建计数缓存失败: key=%s, value=%d, err=%v", cacheKey, value, cacheErr)
	}

	return value, nil
}

func (l *GetCountLogic) queryFromDB(in *count.GetCountReq) (int64, error) {
	row, err := l.countRepo.Get(int32(in.BizType), int32(in.TargetType), in.TargetId)
	if err != nil {
		return 0, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询计数失败"))
	}
	if row == nil {
		return 0, nil
	}
	return row.Value, nil
}
