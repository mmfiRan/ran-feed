package likeservicelogic

import (
	"context"
	"fmt"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"

	"ran-feed/app/rpc/interaction/interaction"
	rediskey "ran-feed/app/rpc/interaction/internal/common/consts/redis"
	luautils "ran-feed/app/rpc/interaction/internal/common/utils/lua"
	"ran-feed/app/rpc/interaction/internal/repositories"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/pkg/cache"
)

type BatchQueryIsLikedLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	likeRepo repositories.LikeRepository
}

func NewBatchQueryIsLikedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchQueryIsLikedLogic {
	return &BatchQueryIsLikedLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		Logger:   logx.WithContext(ctx),
		likeRepo: repositories.NewLikeRepository(ctx, svcCtx.MysqlDb),
	}
}

// cacheView Redis 缓存视图 描述某次查询的命中情况
// trusted=true 当且仅当 _full=1 蕴含 key 存在 业务上 _full 未置位都视为残缺需重建
type cacheView struct {
	trusted  bool           // _full=1 缓存可信
	minCID   int64          // 热区最小 cid
	likedMap map[int64]bool // 缓存命中的 cid → true 未命中不收录
}

// BatchQueryIsLiked 批量查询用户对一批 cid 的点赞态
func (l *BatchQueryIsLikedLogic) BatchQueryIsLiked(in *interaction.BatchQueryIsLikedReq) (*interaction.BatchQueryIsLikedRes, error) {
	if in == nil || len(in.LikeInfos) == 0 {
		return &interaction.BatchQueryIsLikedRes{
			IsLikedInfos: []*interaction.IsLikedInfo{},
		}, nil
	}

	out := buildDefaultOut(in.LikeInfos)
	if len(out.IsLikedInfos) == 0 || in.UserId == nil || *in.UserId <= 0 {
		return out, nil
	}
	userID := *in.UserId

	contentIDs := extractContentIDs(out.IsLikedInfos)
	if len(contentIDs) == 0 {
		return out, nil
	}

	userLikeKey := rediskey.BuildLikeUserKey(strconv.FormatInt(userID, 10))

	view, err := l.queryFromCache(userLikeKey, contentIDs)
	if err != nil {
		// 缓存读失败 直接回源 DB 保证可用性
		l.Errorf("批量查询点赞缓存失败 降级走 DB: user_id=%d err=%v", userID, err)
		return l.fillAllFromDB(out, userID, contentIDs)
	}

	if !view.trusted {
		rebuilt, rebuildErr := l.rebuildAndGetView(userID, userLikeKey)
		if rebuildErr != nil {
			l.Errorf("重建用户点赞缓存失败 降级走 DB: user_id=%d err=%v", userID, rebuildErr)
			return l.fillAllFromDB(out, userID, contentIDs)
		}
		view = rebuilt
	}

	dbQueryIDs := l.fillFromCacheView(out, view)
	if len(dbQueryIDs) > 0 {
		if err = l.fillColdFromDB(out, userID, dbQueryIDs, view.minCID); err != nil {
			l.Errorf("批量查询冷数据点赞 DB 失败: user_id=%d err=%v", userID, err)
		}
	}
	return out, nil
}

// queryFromCache 用单条 HMGET 一次性拉取 _full + _mincid + 所有指定 cid 的值
// 字段不存在时 HMGET 返回空字符串 业务字段值固定为 "1"
func (l *BatchQueryIsLikedLogic) queryFromCache(userLikeKey string, contentIDs []int64) (*cacheView, error) {
	uniqIDs := dedupContentIDs(contentIDs)
	if len(uniqIDs) == 0 {
		return &cacheView{}, nil
	}

	fields := buildHmgetFields(uniqIDs)
	vals, err := l.svcCtx.Redis.HmgetCtx(l.ctx, userLikeKey, fields...)
	if err != nil {
		return nil, err
	}
	return parseHmgetView(vals, uniqIDs), nil
}

// rebuildAndGetView 通过分布式锁重建 全集群同 userKey 同时刻只一个实例执行 rebuild
// 拿不到锁的实例轮询 cache 等别人重建好 超时返回 ErrLockBusy
func (l *BatchQueryIsLikedLogic) rebuildAndGetView(userID int64, userLikeKey string) (*cacheView, error) {
	lockKey := cache.BuildLockKey(userLikeKey)
	return cache.DoWithLock(
		l.svcCtx.LikeUserRebuildLocker,
		l.ctx,
		lockKey,
		func(ctx context.Context) (*cacheView, bool, error) {
			view, err := l.probeCache(ctx, userLikeKey)
			if err != nil {
				return nil, false, err
			}
			return view, view.trusted, nil
		},
		func(_ context.Context) (*cacheView, error) {
			return l.doRebuild(userID, userLikeKey)
		},
	)
}

// doRebuild 真正执行重建 DB topN + 原子 Lua 写回
func (l *BatchQueryIsLikedLogic) doRebuild(userID int64, userLikeKey string) (view *cacheView, err error) {
	defer func() {
		if r := recover(); r != nil {
			l.Errorf("重建用户点赞缓存 panic: user_id=%d recovered=%v", userID, r)
			err = fmt.Errorf("rebuild user like hash panic: %v", r)
		}
	}()

	bgCtx := context.WithoutCancel(l.ctx)

	cids, err := l.likeRepo.QueryUserLikedTopN(userID, rediskey.RedisLikeUserHashCapacity)
	if err != nil {
		return nil, err
	}

	if err := l.evalRebuildScript(bgCtx, userLikeKey, cids); err != nil {
		return nil, err
	}

	return buildRebuiltView(cids), nil
}

// probeCache 探测当前缓存元信息 用于分布式锁的双检与等待轮询
// 只拉 _full 和 _mincid 不查具体 cid
func (l *BatchQueryIsLikedLogic) probeCache(ctx context.Context, userLikeKey string) (*cacheView, error) {
	vals, err := l.svcCtx.Redis.HmgetCtx(ctx, userLikeKey, "_full", "_mincid")
	if err != nil {
		return nil, err
	}
	return parseHmgetView(vals, nil), nil
}

// evalRebuildScript 调用原子重建 Lua DEL + HSET 业务字段 + _mincid + _full=1
func (l *BatchQueryIsLikedLogic) evalRebuildScript(ctx context.Context, userLikeKey string, cids []int64) error {
	args := make([]any, 0, len(cids)+2)
	args = append(args, strconv.FormatInt(rediskey.RedisLikeExpireSeconds, 10))
	if len(cids) > 0 {
		// topN 是降序 最小 cid 在末尾 用其当 _mincid
		args = append(args, strconv.FormatInt(cids[len(cids)-1], 10))
	} else {
		// 用户无任何点赞 写一个标记 _full=1 的空 hash 避免反复重建
		args = append(args, "")
	}
	for _, cid := range cids {
		args = append(args, strconv.FormatInt(cid, 10))
	}
	_, err := l.svcCtx.Redis.EvalCtx(
		ctx,
		luautils.RebuildLikeUserHashScript,
		[]string{userLikeKey},
		args...,
	)
	return err
}

// fillFromCacheView 按缓存视图填充结果 返回需回源 DB 的冷数据 cid 列表
func (l *BatchQueryIsLikedLogic) fillFromCacheView(out *interaction.BatchQueryIsLikedRes, view *cacheView) []int64 {
	dbQueryIDs := make([]int64, 0)
	for _, item := range out.IsLikedInfos {
		cid := item.ContentId
		if cid <= 0 {
			continue
		}
		if view.minCID > 0 && cid < view.minCID {
			dbQueryIDs = append(dbQueryIDs, cid)
			continue
		}
		item.IsLiked = view.likedMap[cid]
	}
	return dbQueryIDs
}

// fillColdFromDB 冷数据回源 DB 仅填充 cid < minCID 的项
func (l *BatchQueryIsLikedLogic) fillColdFromDB(out *interaction.BatchQueryIsLikedRes, userID int64, dbQueryIDs []int64, minCID int64) error {
	dbLikedMap, err := l.likeRepo.BatchIsLiked(userID, dbQueryIDs)
	if err != nil {
		return err
	}
	for _, item := range out.IsLikedInfos {
		cid := item.ContentId
		if cid > 0 && minCID > 0 && cid < minCID {
			item.IsLiked = dbLikedMap[cid]
		}
	}
	return nil
}

// fillAllFromDB 缓存完全不可用时 整批走 DB 兜底
func (l *BatchQueryIsLikedLogic) fillAllFromDB(out *interaction.BatchQueryIsLikedRes, userID int64, contentIDs []int64) (*interaction.BatchQueryIsLikedRes, error) {
	dbLikedMap, err := l.likeRepo.BatchIsLiked(userID, contentIDs)
	if err != nil {
		// DB 也失败 返回 default false 但记日志 不阻塞调用方
		l.Errorf("缓存降级后 DB 批量查询点赞也失败: user_id=%d err=%v", userID, err)
		return out, nil
	}
	for _, item := range out.IsLikedInfos {
		if item.ContentId > 0 {
			item.IsLiked = dbLikedMap[item.ContentId]
		}
	}
	return out, nil
}

// buildDefaultOut 入参默认未点赞的结果集 保留 nil 过滤后顺序
func buildDefaultOut(infos []*interaction.LikeInfo) *interaction.BatchQueryIsLikedRes {
	out := &interaction.BatchQueryIsLikedRes{
		IsLikedInfos: make([]*interaction.IsLikedInfo, 0, len(infos)),
	}
	for _, info := range infos {
		if info == nil {
			continue
		}
		out.IsLikedInfos = append(out.IsLikedInfos, &interaction.IsLikedInfo{
			ContentId: info.ContentId,
			Scene:     info.Scene,
			IsLiked:   false,
		})
	}
	return out
}

// extractContentIDs 抽取所有 cid
func extractContentIDs(infos []*interaction.IsLikedInfo) []int64 {
	cids := make([]int64, 0, len(infos))
	for _, item := range infos {
		cids = append(cids, item.ContentId)
	}
	return cids
}

// dedupContentIDs 过滤非法 cid 并按首次出现顺序去重
func dedupContentIDs(contentIDs []int64) []int64 {
	uniq := make([]int64, 0, len(contentIDs))
	seen := make(map[int64]struct{}, len(contentIDs))
	for _, cid := range contentIDs {
		if cid <= 0 {
			continue
		}
		if _, ok := seen[cid]; ok {
			continue
		}
		seen[cid] = struct{}{}
		uniq = append(uniq, cid)
	}
	return uniq
}

// buildHmgetFields 拼装 HMGET 参数 顺序固定 _full _mincid 业务 cid...
// 与 parseHmgetView 的解析顺序一一对应
func buildHmgetFields(uniqIDs []int64) []string {
	fields := make([]string, 0, len(uniqIDs)+2)
	fields = append(fields, "_full", "_mincid")
	for _, cid := range uniqIDs {
		fields = append(fields, strconv.FormatInt(cid, 10))
	}
	return fields
}

// parseHmgetView 把 HMGET 返回的 []string 翻译为 cacheView
// uniqIDs 传 nil 表示仅探测模式 不构造 likedMap
func parseHmgetView(vals []string, uniqIDs []int64) *cacheView {
	view := &cacheView{
		likedMap: make(map[int64]bool, len(uniqIDs)),
	}
	if len(vals) > 0 && vals[0] == "1" {
		view.trusted = true
	}
	if len(vals) > 1 && vals[1] != "" {
		if parsed, err := strconv.ParseInt(vals[1], 10, 64); err == nil {
			view.minCID = parsed
		}
	}
	for i, cid := range uniqIDs {
		idx := i + 2
		if idx >= len(vals) {
			break
		}
		if vals[idx] == "1" {
			view.likedMap[cid] = true
		}
	}
	return view
}

// buildRebuiltView 把重建拿到的 topN cids 翻译成缓存视图
// 全量都是已点赞 minCID = 末尾元素（topN 是降序）
func buildRebuiltView(cids []int64) *cacheView {
	view := &cacheView{
		trusted:  true,
		likedMap: make(map[int64]bool, len(cids)),
	}
	if len(cids) > 0 {
		view.minCID = cids[len(cids)-1]
	}
	for _, cid := range cids {
		view.likedMap[cid] = true
	}
	return view
}
