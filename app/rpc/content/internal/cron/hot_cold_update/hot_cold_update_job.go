package hot_cold_update

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"ran-feed/app/rpc/content/content"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	luautils "ran-feed/app/rpc/content/internal/common/utils/lua"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/hotrank"
	"ran-feed/pkg/xxljob"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

const HandlerName = "hot.cold.update"

const (
	// 冷更新窗口：只重算最近 windowDays 的内容，避免全库扫描
	defaultWindowDays = 15
	// 重建后保留 TopN 作为对外可查询快照
	defaultTopN      = 5000
	defaultLockTTL   = 3600 // 1 小时
	defaultBatchSize = 500
	defaultPageSize  = 1000
	defaultShards    = 64
	// 与 fast_update 统一的默认半衰期
	defaultHalfLife = 24
	// 快照默认 1 小时过期，避免历史快照无限累积
	defaultSnapshotTTL = 3600
	snapshotIDLayout   = "20060102150405"
	coldLockDateLayout = "20060102"
)

type Params struct {
	WindowDays    int              `json:"windowDays"`
	TopN          int              `json:"topN"`
	LockTTL       int              `json:"lockTtl"`
	HalfLifeHours float64          `json:"halfLifeHours"`
	Weights       *hotrank.Weights `json:"weights"`
	BatchSize     int              `json:"batchSize"`
	PageSize      int              `json:"pageSize"`
	Shards        int              `json:"shards"`
}

type HotColdUpdateJob struct {
	svc         *svc.ServiceContext
	contentRepo repositories.ContentRepository
}

// Register 注册热榜慢更新任务。
func Register(ctx context.Context, executor *xxljob.Executor, svcCtx *svc.ServiceContext) {
	job := &HotColdUpdateJob{
		svc:         svcCtx,
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
	}
	executor.RegisterTask(HandlerName, job.Run)
}

// Run 慢更新热度（每日/冷更新）
func (j *HotColdUpdateJob) Run(ctx context.Context, param xxljob.TriggerParam) (string, error) {
	p := parseParams(param.ExecutorParams)
	if p.WindowDays <= 0 {
		p.WindowDays = defaultWindowDays
	}
	if p.TopN <= 0 {
		p.TopN = defaultTopN
	}
	if p.LockTTL <= 0 {
		p.LockTTL = defaultLockTTL
	}
	if p.BatchSize <= 0 {
		p.BatchSize = defaultBatchSize
	}
	if p.PageSize <= 0 {
		p.PageSize = defaultPageSize
	}
	if p.Shards <= 0 {
		p.Shards = defaultShards
	}
	if p.HalfLifeHours <= 0 {
		p.HalfLifeHours = defaultHalfLife
	}

	calculator := hotrank.ExpDecay{
		Weights:       mergeWeights(p.Weights),
		HalfLifeHours: p.HalfLifeHours,
	}

	now := time.Now().UTC()
	lockDate := now.Format(coldLockDateLayout)
	lockKey := rediskey.BuildHotFeedColdLockKey(lockDate)
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

	// 冷更新采用“先清空全局榜，再按窗口重建”的策略：
	// 可以把 fast_update 期间积累的误差（例如未连续衰减）一次性纠正回来
	if _, err := j.svc.Redis.DelCtx(ctx, rediskey.RedisFeedHotGlobalKey); err != nil {
		return "", err
	}

	startTime := now.Add(-time.Duration(p.WindowDays) * 24 * time.Hour)
	if err := j.rebuildFromDB(ctx, calculator, startTime, now, p); err != nil {
		return "", err
	}

	// 重建完成后生成最新快照
	pairs, err := j.svc.Redis.ZrevrangeWithScoresByFloatCtx(ctx, rediskey.RedisFeedHotGlobalKey, 0, int64(p.TopN-1))
	if err != nil {
		return "", err
	}

	if len(pairs) > 0 {
		snapshotID := now.Format(snapshotIDLayout)
		snapshotKey := rediskey.BuildHotFeedSnapshotKey(snapshotID)
		if _, err := j.svc.Redis.EvalCtx(ctx, luautils.RebuildHotSnapshotScript, []string{
			rediskey.RedisFeedHotGlobalKey,
			snapshotKey,
			rediskey.RedisFeedHotGlobalLatestKey,
		}, strconv.FormatInt(int64(p.TopN), 10), snapshotID, strconv.Itoa(defaultSnapshotTTL)); err != nil {
			return "", err
		}
	}

	// 清理所有增量桶，避免冷更新后把旧增量再次合并
	for shard := 0; shard < p.Shards; shard++ {
		incKey := rediskey.BuildHotFeedIncKey(shard)
		if _, err := j.svc.Redis.DelCtx(ctx, incKey); err != nil {
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

func (j *HotColdUpdateJob) rebuildFromDB(ctx context.Context, calculator hotrank.ExpDecay, startTime, now time.Time, p Params) error {
	// 基于内容ID倒序游标分页，全量扫描窗口内“已发布且公开”的内容
	cursorID := int64(0)
	for {
		rows, err := j.contentRepo.ListColdUpdateContents(
			int32(content.ContentStatus_PUBLISHED),
			int32(content.Visibility_PUBLIC),
			startTime,
			cursorID,
			p.PageSize,
		)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}

		ids := make([]int64, 0, len(rows))               // DB 批量更新 hot_score
		scores := make([]float64, 0, len(rows))          // 对应分值
		redisArgs := make([]interface{}, 0, len(rows)*2) // zadd 参数：score, member...

		for _, row := range rows {
			if row == nil || row.PublishedAt == nil {
				continue
			}
			// 冷更新分值是“时点重算值”，不是增量。
			score := calcScore(calculator, row, now)
			ids = append(ids, row.ID)
			scores = append(scores, score)
			redisArgs = append(redisArgs, score, strconv.FormatInt(row.ID, 10))
		}

		if len(ids) > 0 {
			if err := j.batchUpdateHotScore(ctx, ids, scores, p.BatchSize); err != nil {
				return err
			}
			if _, err := j.svc.Redis.EvalCtx(ctx, luautils.RebuildHotFeedZSetScript, []string{
				rediskey.RedisFeedHotGlobalKey,
			}, redisArgs...); err != nil {
				return err
			}
		}

		// 下一页继续向更小 ID 扫描
		cursorID = rows[len(rows)-1].ID
		if len(rows) < p.PageSize {
			return nil
		}
	}
}

func calcScore(calculator hotrank.Calculator, row *model.RanFeedContent, now time.Time) float64 {
	publishedAt := now
	if row.PublishedAt != nil {
		publishedAt = row.PublishedAt.UTC()
	}

	// 这里保留 Calculator 接口入参，便于未来替换公式实现
	exp, ok := calculator.(hotrank.ExpDecay)
	if !ok {
		return 0
	}

	// 与 fast_update 完全一致的基础公式，保证快慢任务口径统一
	weighted := float64(row.LikeCount)*exp.Weights.Like +
		float64(row.CommentCount)*exp.Weights.Comment +
		float64(row.FavoriteCount)*exp.Weights.Favorite
	if weighted <= 0 {
		return 0
	}

	ageHours := now.Sub(publishedAt).Hours()
	if ageHours < 0 {
		ageHours = 0
	}
	decay := 1.0
	if exp.HalfLifeHours > 0 {
		decay = math.Exp(-math.Ln2 * ageHours / exp.HalfLifeHours)
	}

	score := math.Log1p(weighted) * decay
	return math.Round(score*1000) / 1000
}

func (j *HotColdUpdateJob) batchUpdateHotScore(ctx context.Context, ids []int64, scores []float64, batchSize int) error {
	if len(ids) == 0 {
		return nil
	}
	if len(ids) != len(scores) {
		return fmt.Errorf("ids and scores length mismatch")
	}
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}
	for start := 0; start < len(ids); start += batchSize {
		end := start + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		if err := j.contentRepo.BatchUpdateHotScores(ids[start:end], scores[start:end], time.Now()); err != nil {
			return err
		}
	}
	return nil
}
