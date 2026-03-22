package hot_fast_update

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
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
	// 分片数：用于把增量写入压力打散到多个 hash key，避免单 key 热点
	defaultShards = 64
	// 热榜只维护前 N 个用于下游查询/快照
	defaultTopN = 5000
	// 分钟桶锁默认 5 分钟，防止同一时间窗重复执行
	defaultLockTTL = 300 // 5 分钟
	// 默认半衰期 24h：每过 24 小时分值衰减到原来的一半
	defaultHalfLifeHour = 24
	// 快照默认 1 小时过期，避免历史快照无限累积
	defaultSnapshotTTL  = 3600
	snapshotIDLayout    = "20060102150405"
	defaultBucketLayout = "200601021504"
)

type Params struct {
	Shards        int              `json:"shards"`
	TopN          int              `json:"topN"`
	LockTTL       int              `json:"lockTtl"`
	HalfLifeHours float64          `json:"halfLifeHours"`
	Weights       *hotrank.Weights `json:"weights"`
}

type HotFastUpdateJob struct {
	svc         *svc.ServiceContext
	contentRepo repositories.ContentRepository
}

// Register 注册热榜快速更新任务。
func Register(ctx context.Context, executor *xxljob.Executor, svcCtx *svc.ServiceContext) {
	job := &HotFastUpdateJob{
		svc:         svcCtx,
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
	}
	executor.RegisterTask(HandlerName, job.Run)
}

// Run 快速合并增量并刷新快照。
func (j *HotFastUpdateJob) Run(ctx context.Context, param xxljob.TriggerParam) (string, error) {

	p := parseParams(param.ExecutorParams)
	if p.Shards <= 0 {
		p.Shards = defaultShards
	}
	if p.TopN <= 0 {
		p.TopN = defaultTopN
	}
	if p.LockTTL <= 0 {
		p.LockTTL = defaultLockTTL
	}
	if p.HalfLifeHours <= 0 {
		p.HalfLifeHours = defaultHalfLifeHour
	}

	calculator := hotrank.ExpDecay{
		Weights:       mergeWeights(p.Weights),
		HalfLifeHours: p.HalfLifeHours,
	}

	// 使用“分钟桶 + 分布式锁”防重：同一分钟只允许一个实例合并增量
	bucket := time.Now().UTC().Format(defaultBucketLayout)
	lockKey := rediskey.BuildHotFeedFastLockKey(bucket)
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

	// 遍历每个分片增量桶，把原始行为增量转换成“本次应加分值”并合并进全局热榜
	// updatedScores 记录本轮被改动的内容，后续只把这些内容同步到 DB，减少写放大
	updatedScores := make(map[int64]float64)
	for shard := 0; shard < p.Shards; shard++ {
		incKey := rediskey.BuildHotFeedIncKey(shard)
		now := time.Now().UTC()
		items, err := j.svc.Redis.HgetallCtx(ctx, incKey)
		if err != nil {
			return "", err
		}
		if len(items) == 0 {
			continue
		}

		// deltaMap: member(contentID) -> deltaScore
		// 这里先在 Go 里统一口径算分，再交给 Lua 原子合并，避免脚本里重复实现公式
		deltaMap := make(map[string]float64, len(items))
		for itemID, raw := range items {
			delta, err := computeDelta(raw, calculator, now)
			if err != nil {
				return "", err
			}
			if delta == 0 {
				continue
			}
			deltaMap[itemID] = delta
		}

		if len(deltaMap) == 0 {
			continue
		}

		if err := j.mergeIncAtomic(ctx, incKey, deltaMap, updatedScores); err != nil {
			return "", err
		}
	}

	// 读取当前 TopN，用于落库和快照
	pairs, err := j.svc.Redis.ZrevrangeWithScoresByFloatCtx(ctx, rediskey.RedisFeedHotGlobalKey, 0, int64(p.TopN-1))
	if err != nil {
		return "", err
	}

	if err := j.flushHotScoresTopN(ctx, pairs, updatedScores); err != nil {
		return "", err
	}

	// 刷新快照：把当前全局热榜裁剪成一个“可复用快照”，并更新 latest 指针
	if len(pairs) > 0 {
		snapshotID := time.Now().UTC().Format(snapshotIDLayout)
		snapshotKey := rediskey.BuildHotFeedSnapshotKey(snapshotID)
		if _, err := j.svc.Redis.EvalCtx(ctx, luautils.RebuildHotSnapshotScript, []string{
			rediskey.RedisFeedHotGlobalKey,
			snapshotKey,
			rediskey.RedisFeedHotGlobalLatestKey,
		}, strconv.FormatInt(int64(p.TopN), 10), snapshotID, strconv.Itoa(defaultSnapshotTTL)); err != nil {
			return "", err
		}
	}

	return "ok", nil
}

func parseParams(raw string) Params {
	if raw == "" {
		return Params{}
	}
	var p Params
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return Params{}
	}
	return p
}

type ActionCounts struct {
	Like     int64 `json:"like"`
	Comment  int64 `json:"comment"`
	Favorite int64 `json:"favorite"`
	TS       int64 `json:"ts"`
}

func mergeWeights(w *hotrank.Weights) hotrank.Weights {
	if w == nil {
		return hotrank.DefaultWeights()
	}
	base := hotrank.DefaultWeights()
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

func computeDelta(raw string, calculator hotrank.ExpDecay, now time.Time) (float64, error) {
	// 兼容老格式：直接是数值字符串
	// 老格式已是“分值增量”，这里只做统一精度处理
	if v, err := strconv.ParseFloat(raw, 64); err == nil {
		return math.Round(v*1000) / 1000, nil
	}

	// 新格式：按行为计数 + 时间衰减计算增量分值。
	var payload ActionCounts
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return 0, err
	}
	if payload.Like == 0 && payload.Comment == 0 && payload.Favorite == 0 {
		return 0, nil
	}
	eventTime := now
	if payload.TS > 0 {
		eventTime = time.Unix(payload.TS, 0).UTC()
	}

	// weighted = like*w1 + comment*w2 + favorite*w3
	// delta    = ln(1 + weighted) * exp(-ln2 * ageHours / halfLifeHours)
	weighted := float64(payload.Like)*calculator.Weights.Like +
		float64(payload.Comment)*calculator.Weights.Comment +
		float64(payload.Favorite)*calculator.Weights.Favorite
	if weighted <= 0 {
		return 0, nil
	}
	ageHours := now.Sub(eventTime).Hours()
	if ageHours < 0 {
		ageHours = 0
	}
	decay := 1.0
	if calculator.HalfLifeHours > 0 {
		decay = math.Exp(-math.Ln2 * ageHours / calculator.HalfLifeHours)
	}

	score := math.Log1p(weighted) * decay
	return math.Round(score*1000) / 1000, nil
}

func (j *HotFastUpdateJob) flushHotScoresTopN(ctx context.Context, pairs []redis.FloatPair, scores map[int64]float64) error {
	if len(pairs) == 0 || len(scores) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(pairs))
	values := make([]float64, 0, len(pairs))
	for _, pair := range pairs {
		id, err := strconv.ParseInt(pair.Key, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid item_id=%s", pair.Key)
		}
		if _, ok := scores[id]; !ok {
			continue
		}
		ids = append(ids, id)
		values = append(values, pair.Score)
	}
	if len(ids) == 0 {
		return nil
	}
	return j.batchUpdateHotScore(ctx, ids, values)
}

func (j *HotFastUpdateJob) batchUpdateHotScore(ctx context.Context, ids []int64, scores []float64) error {
	const batchSize = 500
	for start := 0; start < len(ids); start += batchSize {
		end := start + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		// 仅更新本次变更且进入 topN 的内容
		if err := j.contentRepo.BatchUpdateHotScores(ids[start:end], scores[start:end], time.Now()); err != nil {
			return err
		}
	}
	return nil
}

func (j *HotFastUpdateJob) mergeIncAtomic(ctx context.Context, incKey string, deltaMap map[string]float64, updatedScores map[int64]float64) error {

	args := make([]interface{}, 0, 1+len(deltaMap)*2)
	args = append(args, "3")
	for member, delta := range deltaMap {
		args = append(args, member, strconv.FormatFloat(delta, 'f', 6, 64))
	}

	_, err := j.svc.Redis.EvalCtx(ctx, luautils.MergeHotIncScript, []string{
		incKey,
		rediskey.RedisFeedHotGlobalKey,
	}, args...)
	if err != nil {
		return err
	}

	// 读取最新分值用于 DB 同步（只同步本次发生变化的内容）
	for itemID := range deltaMap {
		score, err := j.svc.Redis.ZscoreByFloatCtx(ctx, rediskey.RedisFeedHotGlobalKey, itemID)
		if err != nil {
			return err
		}
		id, err := strconv.ParseInt(itemID, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid item_id=%s", itemID)
		}
		updatedScores[id] = score
	}
	return nil
}
