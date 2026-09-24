package counterservicelogic

import (
	"context"
	"encoding/json"
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

type GetUserProfileCountsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	countRepo repositories.CountValueRepository
}

func NewGetUserProfileCountsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserProfileCountsLogic {
	return &GetUserProfileCountsLogic{
		ctx:       ctx,
		svcCtx:    svcCtx,
		Logger:    logx.WithContext(ctx),
		countRepo: repositories.NewCountValueRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *GetUserProfileCountsLogic) GetUserProfileCounts(in *count.GetUserProfileCountsReq) (*count.GetUserProfileCountsRes, error) {
	if in == nil || in.UserId <= 0 {
		return nil, errorx.NewMsg("查询用户主页计数请求无效")
	}

	cacheKey := buildUserProfileCountsCacheKey(in.UserId)
	if cacheValue, cacheResult := l.queryFromCache(cacheKey); cacheResult == cacheHit {
		return cacheValue, nil
	}

	value, err := l.rebuildCacheWithLock(in.UserId, cacheKey)
	if err != nil {
		return nil, err
	}
	return value, nil
}

type userProfileCountsCache struct {
	FollowingCount int64 `json:"following_count"`
	FollowedCount  int64 `json:"followed_count"`
	LikeCount      int64 `json:"like_count"`
	FavoriteCount  int64 `json:"favorite_count"`
}

func (l *GetUserProfileCountsLogic) queryFromCache(cacheKey string) (*count.GetUserProfileCountsRes, cacheQueryResult) {
	cacheStr, err := l.svcCtx.Redis.GetCtx(l.ctx, cacheKey)
	if err != nil {
		l.Errorf("查询用户主页计数缓存失败: key=%s, err=%v", cacheKey, err)
		return nil, cacheError
	}
	if cacheStr == "" {
		return nil, cacheMiss
	}

	var cached userProfileCountsCache
	if err := json.Unmarshal([]byte(cacheStr), &cached); err != nil {
		l.Errorf("解析用户主页计数缓存失败: key=%s, value=%s, err=%v", cacheKey, cacheStr, err)
		return nil, cacheError
	}
	return &count.GetUserProfileCountsRes{
		FollowingCount: cached.FollowingCount,
		FollowedCount:  cached.FollowedCount,
		LikeCount:      cached.LikeCount,
		FavoriteCount:  cached.FavoriteCount,
	}, cacheHit
}

func (l *GetUserProfileCountsLogic) rebuildCacheWithLock(userID int64, cacheKey string) (*count.GetUserProfileCountsRes, error) {
	lockKey := rediskey.GetRedisPrefixKey(rediskey.RedisUserProfileCountsRebuildLockPrefix, strconv.FormatInt(userID, 10))
	lock := redislock.NewRedisLock(l.svcCtx.Redis, lockKey)
	lock.SetExpire(rebuildLockExpireSeconds)

	lockAcquired, err := lock.AcquireCtx(l.ctx)
	if err != nil {
		l.Errorf("获取用户主页计数重建锁失败，降级查库: lock_key=%s, err=%v", lockKey, err)
		return l.queryFromDB(userID)
	}

	if !lockAcquired {
		// 别人正在重建 不阻塞请求 直接回源查库
		return l.queryFromDB(userID)
	}

	defer func() {
		if releaseOk, releaseErr := lock.ReleaseCtx(l.ctx); !releaseOk || releaseErr != nil {
			l.Errorf("释放用户主页计数重建锁失败: lock_key=%s, err=%v", lockKey, releaseErr)
		}
	}()

	if value, cacheResult := l.queryFromCache(cacheKey); cacheResult == cacheHit {
		return value, nil
	}

	value, err := l.queryFromDB(userID)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(userProfileCountsCache{
		FollowingCount: value.FollowingCount,
		FollowedCount:  value.FollowedCount,
		LikeCount:      value.LikeCount,
		FavoriteCount:  value.FavoriteCount,
	})
	if err != nil {
		l.Errorf("序列化用户主页计数缓存失败: key=%s, err=%v", cacheKey, err)
		return value, nil
	}
	if cacheErr := l.svcCtx.Redis.SetexCtx(l.ctx, cacheKey, string(payload), countCacheExpireSecondsWithJitter()); cacheErr != nil {
		l.Errorf("重建用户主页计数缓存失败: key=%s, err=%v", cacheKey, cacheErr)
	}

	return value, nil
}

func (l *GetUserProfileCountsLogic) queryFromDB(userID int64) (*count.GetUserProfileCountsRes, error) {
	likeCount, err := l.countRepo.SumByOwner(countenum.BizTypeLike.Int32(), countenum.TargetTypeContent.Int32(), userID)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询用户点赞数失败"))
	}
	favoriteCount, err := l.countRepo.SumByOwner(countenum.BizTypeFavorite.Int32(), countenum.TargetTypeContent.Int32(), userID)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询用户收藏数失败"))
	}

	followingCount := int64(0)
	followedCount := int64(0)
	if row, err := l.countRepo.Get(countenum.BizTypeFollowing.Int32(), countenum.TargetTypeUser.Int32(), userID); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关注数失败"))
	} else if row != nil {
		followingCount = row.Value
	}
	if row, err := l.countRepo.Get(countenum.BizTypeFollowed.Int32(), countenum.TargetTypeUser.Int32(), userID); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询被关注数失败"))
	} else if row != nil {
		followedCount = row.Value
	}

	return &count.GetUserProfileCountsRes{
		FollowingCount: followingCount,
		FollowedCount:  followedCount,
		LikeCount:      likeCount,
		FavoriteCount:  favoriteCount,
	}, nil
}
