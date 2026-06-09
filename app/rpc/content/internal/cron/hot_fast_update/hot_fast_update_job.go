package hot_fast_update

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	luautils "ran-feed/app/rpc/content/internal/common/utils/lua"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/hotrank"
	"ran-feed/pkg/xxljob"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

const HandlerName = "hot.fast.update"

const (
	// 脏集合分片数 互动按 contentID 取模 shards 打散 避免单 key 热点
	defaultShards = 32
	// 主榜候选池大小 裁剪线 远大于对外快照
	defaultMainN = 5000
	// 对外快照大小 用户实际翻页读这个
	defaultTopN = 2000
	// 分钟桶锁默认 5 分钟 防止同一时间窗重复执行
	defaultLockTTL = 300
	// 默认半衰期 24h
	defaultHalfLifeHour = 24
	// 快照默认 1 小时过期 避免历史快照无限累积
	defaultSnapshotTTL = 3600
	// SSCAN 每批拉取数量
	defaultScanBatch = 1000
	// 回查计数和落库批大小
	defaultBatchSize = 500

	snapshotIDLayout = "20060102150405"
)

type Params struct {
	Shards        int              `json:"shards"`
	MainN         int              `json:"mainN"`
	TopN          int              `json:"topN"`
	LockTTL       int              `json:"lockTtl"`
	HalfLifeHours float64          `json:"halfLifeHours"`
	Weights       *hotrank.Weights `json:"weights"`
}

type HotFastUpdateJob struct {
	svc         *svc.ServiceContext
	contentRepo repositories.ContentRepository
}

// Register 注册热榜快速更新任务
func Register(ctx context.Context, executor *xxljob.Executor, svcCtx *svc.ServiceContext) {
	job := &HotFastUpdateJob{
		svc:         svcCtx,
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
	}
	executor.RegisterTask(HandlerName, job.Run)
}

// Run 快更 冻结脏集合 回查计数总量算全分 覆盖主榜 刷新快照
func (j *HotFastUpdateJob) Run(ctx context.Context, param xxljob.TriggerParam) (string, error) {
	p := parseParams(param.ExecutorParams)

	calculator := hotrank.AdditiveTime{
		Weights:       mergeWeights(p.Weights),
		HalfLifeHours: p.HalfLifeHours,
	}

	// 冷更优先 见到冷更预约标志主动让路 不去抢写锁 避免快更连续抢锁把冷更饿死
	// 让掉的这轮互动原样留在活跃脏桶 冷更跑完或下一轮快更照常处理 不丢
	pending, err := j.svc.Redis.ExistsCtx(ctx, rediskey.RedisFeedHotColdPendingKey)
	if err != nil {
		return "", err
	}
	if pending {
		return "yield", nil
	}

	lockKey := rediskey.BuildHotFeedWriteLockKey()
	redisLock := redis.NewRedisLock(j.svc.Redis, lockKey)
	redisLock.SetExpire(p.LockTTL)
	locked, err := redisLock.AcquireCtx(ctx)
	if err != nil {
		return "", err
	}
	if !locked {
		return "duplicate", nil
	}
	defer redisLock.ReleaseCtx(context.Background())

	// 逐分片冻结脏集合并收集脏 ID 双缓冲 处理期间新互动堆进活跃桶
	dirtyIDs, err := j.collectDirtyIDs(ctx, p.Shards)
	if err != nil {
		return "", err
	}

	// 回查计数总量算全分 ZADD 覆盖主榜
	if len(dirtyIDs) > 0 {
		if err = j.recomputeAndOverwrite(ctx, calculator, dirtyIDs, p.MainN); err != nil {
			return "", err
		}
	}

	// 裁剪主榜到 MainN 候选池 取前 TopN 建快照 切 latest 指针
	if err = j.refreshSnapshot(ctx, p.MainN, p.TopN); err != nil {
		return "", err
	}
	// 清理已处理的冻结桶
	if err = j.cleanupProcShards(ctx, p.Shards); err != nil {
		return "", err
	}

	return "ok", nil
}

func parseParams(raw string) Params {
	p := Params{}
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &p)
	}
	if p.Shards <= 0 {
		p.Shards = defaultShards
	}
	if p.TopN <= 0 {
		p.TopN = defaultTopN
	}
	if p.MainN <= 0 {
		p.MainN = defaultMainN
	}
	// 主榜候选池必须 >= 对外快照 否则没有候补垫 退化成主榜=快照
	if p.MainN < p.TopN {
		p.MainN = p.TopN
	}
	if p.LockTTL <= 0 {
		p.LockTTL = defaultLockTTL
	}
	if p.HalfLifeHours <= 0 {
		p.HalfLifeHours = defaultHalfLifeHour
	}
	return p
}

func mergeWeights(w *hotrank.Weights) hotrank.Weights {
	base := hotrank.DefaultWeights()
	if w == nil {
		return base
	}
	if w.Like > 0 {
		base.Like = w.Like
	}
	if w.Comment > 0 {
		base.Comment = w.Comment
	}
	if w.Favorite > 0 {
		base.Favorite = w.Favorite
	}
	return base
}

// refreshSnapshot 原子裁剪主榜到 mainN 候选池 取前 topN 重建快照 切 latest 指针
// 裁剪与快照合并在同一 Lua 内执行 主榜留 mainN 候补垫 对外快照只取 topN
func (j *HotFastUpdateJob) refreshSnapshot(ctx context.Context, mainN, topN int) error {
	card, err := j.svc.Redis.ZcardCtx(ctx, rediskey.RedisFeedHotGlobalKey)
	if err != nil {
		return err
	}
	if card == 0 {
		return nil
	}
	snapshotID := time.Now().UTC().Format(snapshotIDLayout)
	snapshotKey := rediskey.BuildHotFeedSnapshotKey(snapshotID)
	_, err = j.svc.Redis.EvalCtx(ctx, luautils.RebuildHotSnapshotScript, []string{
		rediskey.RedisFeedHotGlobalKey,
		snapshotKey,
		rediskey.RedisFeedHotGlobalLatestKey,
	}, strconv.Itoa(mainN), strconv.Itoa(topN), snapshotID, strconv.Itoa(defaultSnapshotTTL))
	return err
}

// cleanupProcShards 删除本轮已处理的冻结桶
func (j *HotFastUpdateJob) cleanupProcShards(ctx context.Context, shards int) error {
	for shard := 0; shard < shards; shard++ {
		if _, err := j.svc.Redis.DelCtx(ctx, rediskey.BuildHotFeedDirtyProcKey(shard)); err != nil {
			return err
		}
	}
	return nil
}
