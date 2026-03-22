package favoriteservicelogic

import (
	"context"
	"math/rand"
	"strconv"
	"time"

	"ran-feed/app/rpc/count/count"
	"ran-feed/app/rpc/interaction/interaction"
	rediskey "ran-feed/app/rpc/interaction/internal/common/consts/redis"
	"ran-feed/app/rpc/interaction/internal/repositories"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	redislock "github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	// 缓存重建锁的过期时间（30秒）
	rebuildLockExpire = 30
)

type CacheResult int

const (
	CacheHit CacheResult = iota
	CacheMiss
	CacheError
)

type QueryFavoriteInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	favoriteRepo repositories.FavoriteRepository
}

func NewQueryFavoriteInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryFavoriteInfoLogic {
	return &QueryFavoriteInfoLogic{
		ctx:          ctx,
		svcCtx:       svcCtx,
		Logger:       logx.WithContext(ctx),
		favoriteRepo: repositories.NewFavoriteRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *QueryFavoriteInfoLogic) QueryFavoriteInfo(in *interaction.QueryFavoriteInfoReq) (*interaction.QueryFavoriteInfoRes, error) {
	favoriteCount, err := l.queryFavoriteCountFromCountRPC(in.ContentId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询收藏计数失败"))
	}

	// 未登录用户默认未收藏，只返回计数。
	if in.UserId <= 0 {
		return l.buildResp(in, favoriteCount, false), nil
	}

	scene := in.Scene.String()
	contentIDStr := strconv.FormatInt(in.ContentId, 10)
	isFavorited, cacheResult := l.queryIsFavoritedFromRedis(in.UserId, scene, contentIDStr)
	if cacheResult == CacheHit {
		return l.buildResp(in, favoriteCount, isFavorited), nil
	}

	// 缓存未命中或错误，尝试使用分布式锁重建 isFavorited 缓存
	isFavorited, err = l.rebuildFavoritedCacheWithLock(in.UserId, in.ContentId, scene, contentIDStr)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询是否收藏失败"))
	}

	return l.buildResp(in, favoriteCount, isFavorited), nil
}

func (l *QueryFavoriteInfoLogic) buildResp(in *interaction.QueryFavoriteInfoReq, favoriteCount int64, isFavorited bool) *interaction.QueryFavoriteInfoRes {
	return &interaction.QueryFavoriteInfoRes{
		ContentId:     in.ContentId,
		Scene:         in.Scene,
		FavoriteCount: favoriteCount,
		IsFavorited:   isFavorited,
	}
}

func (l *QueryFavoriteInfoLogic) queryFavoriteCountFromCountRPC(contentID int64) (int64, error) {
	resp, err := l.svcCtx.CountRpc.GetCount(l.ctx, &count.GetCountReq{
		BizType:    count.BizType_FAVORITE,
		TargetType: count.TargetType_CONTENT,
		TargetId:   contentID,
	})
	if err != nil {
		return 0, err
	}
	return resp.Value, nil
}

func (l *QueryFavoriteInfoLogic) queryIsFavoritedFromRedis(userID int64, scene string, contentIDStr string) (isFavorited bool, result CacheResult) {
	userIDStr := strconv.FormatInt(userID, 10)
	relKey := rediskey.BuildFavoriteRelKey(scene, userIDStr, contentIDStr)

	relVal, err := l.svcCtx.Redis.GetCtx(l.ctx, relKey)
	if err != nil {
		l.Errorf("查询收藏关系缓存失败: %v", err)
		return false, CacheError
	}
	if relVal == "" {
		return false, CacheMiss
	}
	// 约定：1=已收藏，0=未收藏
	if relVal == "1" {
		return true, CacheHit
	}
	if relVal == "0" {
		return false, CacheHit
	}
	return false, CacheError
}

func (l *QueryFavoriteInfoLogic) rebuildFavoritedCacheWithLock(userID, contentID int64, scene, contentIDStr string) (bool, error) {
	lockKey := rediskey.GetRedisPrefixKey("lock:rebuild:favorite", scene+":"+strconv.FormatInt(userID, 10)+":"+contentIDStr)
	lock := redislock.NewRedisLock(l.svcCtx.Redis, lockKey)
	lock.SetExpire(rebuildLockExpire)

	lockAcquired, err := lock.AcquireCtx(l.ctx)
	if err != nil {
		return false, err
	}

	if !lockAcquired {
		const (
			maxRetry    = 5
			baseSleepMs = 30
			jitterMs    = 50
		)
		for i := 0; i < maxRetry; i++ {
			select {
			case <-l.ctx.Done():
				return false, l.ctx.Err()
			default:
			}
			sleep := time.Duration(baseSleepMs+rand.Intn(jitterMs)) * time.Millisecond
			time.Sleep(sleep)

			isFavorited, cacheResult := l.queryIsFavoritedFromRedis(userID, scene, contentIDStr)
			if cacheResult == CacheHit {
				return isFavorited, nil
			}
		}

		// 未拿到锁且重试未命中，直接回源DB
		return l.queryIsFavoritedFromDB(userID, contentID)
	}

	defer func() {
		if releaseOk, releaseErr := lock.ReleaseCtx(l.ctx); !releaseOk || releaseErr != nil {
			l.Errorf("释放分布式锁失败: %v", releaseErr)
		}
	}()

	// 双重检查
	isFavorited, cacheResult := l.queryIsFavoritedFromRedis(userID, scene, contentIDStr)
	if cacheResult == CacheHit {
		return isFavorited, nil
	}

	// 回源DB并重建关系缓存
	isFavorited, err = l.queryIsFavoritedFromDB(userID, contentID)
	if err != nil {
		return false, err
	}
	l.rebuildFavoritedCache(userID, scene, contentIDStr, isFavorited)
	return isFavorited, nil
}

func (l *QueryFavoriteInfoLogic) queryIsFavoritedFromDB(userID, contentID int64) (bool, error) {
	return l.favoriteRepo.IsFavorited(userID, contentID)
}

func (l *QueryFavoriteInfoLogic) rebuildFavoritedCache(userID int64, scene, contentIDStr string, isFavorited bool) {
	userIDStr := strconv.FormatInt(userID, 10)
	relKey := rediskey.BuildFavoriteRelKey(scene, userIDStr, contentIDStr)

	value := "0"
	expireSeconds := rediskey.RedisFavoriteRelNegativeExpireSeconds
	if isFavorited {
		value = "1"
		expireSeconds = rediskey.RedisFavoriteRelExpireSeconds
	}

	if err := l.svcCtx.Redis.SetexCtx(l.ctx, relKey, value, expireSeconds); err != nil {
		l.Errorf("重建收藏关系缓存失败: %v", err)
	}
}
