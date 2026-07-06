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

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

const HandlerName = "hot.cold.update"

const (
	// 冷更窗口推导兜底 半衰期非正等异常时退化用此固定窗口
	defaultWindowDays = 15
	// 窗口自动推导允许的互动量级差 decades 取 5 即容忍 10^5 倍互动差距已很宽松
	// 窗口天数 = ceil(decades × halfLifeHours / (24 × log10(2))) 与竞争视界对齐
	windowDecades = 5
	// 自动推导窗口上限 防止半衰期被调超大时退化成全表扫描
	maxWindowDays = 90
	// 主榜候选池大小 裁剪线 远大于对外快照 留候补垫扛删除侵蚀
	defaultMainN = 5000
	// 重建后取前 TopN 作为对外可查询快照
	defaultTopN      = 2000
	defaultLockTTL   = 3600 // 1 小时
	defaultBatchSize = 500
	defaultPageSize  = 1000
	defaultShards    = 64
	// 与 fast_update 统一的默认半衰期
	defaultHalfLife = 24
	// 快照默认 1 小时过期，避免历史快照无限累积
	defaultSnapshotTTL = 3600
	// 冷更预约标志 TTL 秒 与锁 TTL 同量级 防异常未清标志永久挡快更
	defaultColdPendingTTL = 3600
	// 冷更挂牌后轮询抢写锁的最大等待秒数 只需等当前一轮快更自然结束
	defaultAcquireWaitSeconds = 60
	// 轮询抢锁间隔秒
	defaultAcquireRetryInterval = 3
	snapshotIDLayout            = "20060102150405"
	coldLockDateLayout          = "20060102"
)

type Params struct {
	WindowDays    int              `json:"windowDays"`
	MainN         int              `json:"mainN"`
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
	// 窗口缺省时从半衰期推导 与竞争视界对齐 半衰期调大窗口自动跟随 防静默砍榜
	// 显式传 windowDays 仍按传入值 不覆盖
	if p.WindowDays <= 0 {
		p.WindowDays = deriveWindowDays(p.HalfLifeHours)
	}

	logger := logx.WithContext(ctx)
	logger.Infof("热榜冷更开始 windowDays=%d mainN=%d topN=%d halfLife=%.1f shards=%d", p.WindowDays, p.MainN, p.TopN, p.HalfLifeHours, p.Shards)

	calculator := hotrank.AdditiveTime{
		Weights:       mergeWeights(p.Weights),
		HalfLifeHours: p.HalfLifeHours,
	}

	now := time.Now().UTC()
	// 日期锁 每日幂等去重 同一天重复触发只跑一次
	lockDate := now.Format(coldLockDateLayout)
	redisLock := redis.NewRedisLock(j.svc.Redis, rediskey.BuildHotFeedColdLockKey(lockDate))
	redisLock.SetExpire(p.LockTTL)
	locked, err := redisLock.AcquireCtx(ctx)
	if err != nil {
		return "", fmt.Errorf("抢当日冷更锁失败 %w", err)
	}
	if !locked {
		logger.Info("热榜冷更放弃 当日已执行")
		return "duplicate", nil
	}
	defer redisLock.Release()

	// 置冷更预约标志 带 TTL 快更见到即主动让路不抢写锁 保证冷更每日必跑不被饿死
	// 异常未清标志时 TTL 到期自动失效 快更最多让路到此为止 不会被永久挡住
	coldPendingTTL := p.LockTTL
	if coldPendingTTL <= 0 {
		coldPendingTTL = defaultColdPendingTTL
	}
	if err := j.svc.Redis.SetexCtx(ctx, rediskey.RedisFeedHotColdPendingKey, "1", coldPendingTTL); err != nil {
		return "", fmt.Errorf("置冷更让路标志失败 %w", err)
	}
	defer func() { _, _ = j.svc.Redis.DelCtx(context.Background(), rediskey.RedisFeedHotColdPendingKey) }()

	// 主榜写锁 与快更共享同一把 防止冷更重建期间快更并发写主榜或被删脏桶丢互动
	// 已挂预约标志 新快更都已让路 此处有界轮询只需等当前正在跑的那一轮快更自然结束
	// 超时仍拿不到 说明极端卡死 清标志退出返回 busy 由 xxljob 下次重调 不死等不拖垮任务
	writeLock := redis.NewRedisLock(j.svc.Redis, rediskey.BuildHotFeedWriteLockKey())
	writeLock.SetExpire(p.LockTTL)
	writeLocked, err := j.acquireWriteLockBounded(ctx, writeLock)
	if err != nil {
		return "", fmt.Errorf("抢主榜写锁失败 %w", err)
	}
	if !writeLocked {
		logger.Info("热榜冷更放弃 等主榜写锁超时")
		return "busy", nil
	}
	defer writeLock.ReleaseCtx(context.Background())

	// 冷更新用持久信源对账快更 补丢事件丢种脏 纠 count-rpc 与 DB 冗余计数漂移
	// 回收被裁内容 兜底 Redis 整体丢失
	rebuildKey := rediskey.RedisFeedHotGlobalRebuildKey
	// 清掉上轮可能残留的影子 key 防止脏数据并入本轮重建
	if _, err := j.svc.Redis.DelCtx(ctx, rebuildKey); err != nil {
		return "", fmt.Errorf("清理重建影子 key 失败 %w", err)
	}

	startTime := now.Add(-time.Duration(p.WindowDays) * 24 * time.Hour)
	if err := j.rebuildFromDB(ctx, calculator, startTime, now, p, rebuildKey); err != nil {
		return "", fmt.Errorf("从 DB 重建热榜失败 %w", err)
	}

	// 影子 key 重建条数 为空说明窗口内无内容 保留旧主榜不动 直接走后续清理
	rebuildCard, err := j.svc.Redis.ZcardCtx(ctx, rebuildKey)
	if err != nil {
		return "", fmt.Errorf("读重建影子 key 数量失败 %w", err)
	}
	logger.Infof("热榜冷更重建完成 count=%d", rebuildCard)
	if rebuildCard > 0 {
		if err := j.svc.Redis.PipelinedCtx(ctx, func(pipe redis.Pipeliner) error {
			pipe.Rename(ctx, rebuildKey, rediskey.RedisFeedHotGlobalKey)
			return nil
		}); err != nil {
			return "", fmt.Errorf("切换主榜失败 %w", err)
		}
	}

	// 重建完成后裁剪主榜到 MainN 候选池 取前 TopN 生成最新快照
	card, err := j.svc.Redis.ZcardCtx(ctx, rediskey.RedisFeedHotGlobalKey)
	if err != nil {
		return "", fmt.Errorf("读主榜数量失败 %w", err)
	}

	if card > 0 {
		snapshotID := now.Format(snapshotIDLayout)
		snapshotKey := rediskey.BuildHotFeedSnapshotKey(snapshotID)
		if _, err := j.svc.Redis.EvalCtx(ctx, luautils.RebuildHotSnapshotScript, []string{
			rediskey.RedisFeedHotGlobalKey,
			snapshotKey,
			rediskey.RedisFeedHotGlobalLatestKey,
		}, strconv.Itoa(p.MainN), strconv.Itoa(p.TopN), snapshotID, strconv.Itoa(defaultSnapshotTTL)); err != nil {
			return "", fmt.Errorf("重建快照失败 %w", err)
		}
		logger.Infof("热榜冷更快照重建完成 snapshotID=%s main=%d", snapshotID, card)
	} else {
		logger.Info("热榜冷更跳过快照 主榜为空")
	}

	// 清理所有脏集合桶 活跃和冻结 与旧版增量桶 避免冷更新后把旧脏数据再次合并
	// 冷更新是全量重算覆盖主榜的对账 完成后脏集合应清零 由快更在干净基础上重新积累
	for shard := 0; shard < p.Shards; shard++ {
		keys := []string{
			rediskey.BuildHotFeedDirtyKey(shard),
			rediskey.BuildHotFeedDirtyProcKey(shard),
			rediskey.BuildHotFeedIncKey(shard), // 旧记账格式 过渡期一并清理
		}
		for _, k := range keys {
			if _, err := j.svc.Redis.DelCtx(ctx, k); err != nil {
				return "", fmt.Errorf("清理脏集合桶失败 %w", err)
			}
		}
	}

	logger.Info("热榜冷更完成")
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

// acquireWriteLockBounded 有界轮询抢写锁 已挂冷更预约标志故新快更都让路
// 只需等当前正在跑的那一轮快更自然结束 间隔 defaultAcquireRetryInterval 秒重试
// 最多等 defaultAcquireWaitSeconds 秒 超时返回 false 由调用方清标志退出让调度重试
// ctx 取消时立即返回 不拖垮任务
func (j *HotColdUpdateJob) acquireWriteLockBounded(ctx context.Context, lock *redis.RedisLock) (bool, error) {
	deadline := time.Now().Add(defaultAcquireWaitSeconds * time.Second)
	for {
		locked, err := lock.AcquireCtx(ctx)
		if err != nil {
			return false, err
		}
		if locked {
			return true, nil
		}
		if time.Now().After(deadline) {
			return false, nil
		}
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(defaultAcquireRetryInterval * time.Second):
		}
	}
}

// deriveWindowDays 从半衰期推导冷更窗口 与加法时间模型的竞争视界对齐
// 时间项每天增量 = 1/(halfLifeHours/24×log10(2)) 个 decade 容忍 windowDecades 个量级差
// 窗口天数 = ceil(windowDecades × halfLifeHours / (24 × log10(2)))
// 半衰期非正时退化为固定兜底窗口 推导值封顶 maxWindowDays 防全表扫描
func deriveWindowDays(halfLifeHours float64) int {
	if halfLifeHours <= 0 {
		return defaultWindowDays
	}
	days := int(math.Ceil(windowDecades * halfLifeHours / (24 * math.Log10(2))))
	if days < 1 {
		days = 1
	}
	if days > maxWindowDays {
		days = maxWindowDays
	}
	return days
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

func (j *HotColdUpdateJob) rebuildFromDB(ctx context.Context, calculator hotrank.AdditiveTime, startTime, now time.Time, p Params, targetKey string) error {
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
			return fmt.Errorf("分页拉取冷更内容失败 %w", err)
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
				return fmt.Errorf("批量落库 hot_score 失败 %w", err)
			}
			if _, err := j.svc.Redis.EvalCtx(ctx, luautils.RebuildHotFeedZSetScript, []string{
				targetKey,
			}, redisArgs...); err != nil {
				return fmt.Errorf("写重建影子 key 失败 %w", err)
			}
		}

		// 下一页继续向更小 ID 扫描
		cursorID = rows[len(rows)-1].ID
		if len(rows) < p.PageSize {
			return nil
		}
	}
}

func calcScore(calculator hotrank.AdditiveTime, row *model.RanFeedContent, now time.Time) float64 {
	publishedAt := now
	if row.PublishedAt != nil {
		publishedAt = row.PublishedAt.UTC()
	}
	weighted := calculator.Weighted(row.LikeCount, row.CommentCount, row.FavoriteCount)

	// 与 fast_update 完全一致的加法时间项公式 保证快慢任务口径统一
	return calculator.Score(weighted, publishedAt)
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
