package hotfeed

import (
	"context"
	"strconv"

	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

// Page 热榜一页 ResolvedSnapshotID 是实际命中的快照 由前端回传保证翻页在同一快照内
type Page struct {
	ContentIDs         []int64
	NextCursor         string
	HasMore            bool
	ResolvedSnapshotID string
}

// Query 读热榜一页 优先指定快照 缺失回退最新快照再回退主榜
// 第二个返回值表示是否读到了榜 没读到由调用方决定怎么兜底
func (f *Feed) Query(ctx context.Context, snapshotID, cursor string, pageSize int) (Page, bool, error) {
	preferredKey := ""
	if snapshotID != "" {
		preferredKey = rediskey.BuildHotFeedSnapshotKey(snapshotID)
	}

	res, err := f.redis.EvalCtx(
		ctx,
		queryZSetScript,
		[]string{
			preferredKey,
			rediskey.RedisFeedHotGlobalLatestKey,
			rediskey.RedisFeedHotGlobalSnapshotPrefix,
			rediskey.RedisFeedHotGlobalKey,
		},
		cursor,
		strconv.Itoa(pageSize),
		snapshotID,
	)
	if err != nil {
		return Page{}, false, errorx.Wrap(ctx, err, errorx.NewMsg("查询热榜索引失败"))
	}
	return parseQueryReply(res)
}

// Remove 把一条内容从主榜摘掉 删除或下架时调用
func (f *Feed) Remove(ctx context.Context, contentID int64) error {
	if contentID <= 0 {
		return nil
	}
	_, err := f.redis.ZremCtx(ctx, rediskey.RedisFeedHotGlobalKey, strconv.FormatInt(contentID, 10))
	return err
}

// parseQueryReply 解 query 脚本的回复 形状是 exists hasMore nextCursor snapshotID 后跟 content_id 列表
func parseQueryReply(res any) (Page, bool, error) {
	arr, ok := res.([]any)
	if !ok || len(arr) < 4 {
		return Page{}, false, errorx.NewMsg("查询热榜索引失败")
	}

	existsVal, _ := replyInt64(arr[0])
	if existsVal != 1 {
		return Page{}, false, nil
	}

	hasMoreVal, _ := replyInt64(arr[1])
	hasMore := hasMoreVal == 1

	nextCursor := ""
	if hasMore {
		nextCursor, _ = replyString(arr[2])
	}
	resolvedSnapshotID, _ := replyString(arr[3])

	ids := make([]int64, 0, len(arr)-4)
	for i := 4; i < len(arr); i++ {
		s, _ := replyString(arr[i])
		if s == "" {
			continue
		}
		id, perr := strconv.ParseInt(s, 10, 64)
		if perr != nil || id <= 0 {
			continue
		}
		ids = append(ids, id)
	}

	return Page{
		ContentIDs:         ids,
		NextCursor:         nextCursor,
		HasMore:            hasMore,
		ResolvedSnapshotID: resolvedSnapshotID,
	}, true, nil
}

// replyString 取 Eval 回复里的字符串 Lua 的 string 经 RESP 回来可能是 string 也可能是 []byte
func replyString(v any) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, true
	case []byte:
		return string(t), true
	default:
		return "", false
	}
}

// replyInt64 取 Eval 回复里的整数 Lua 的 number 和数字字符串都会出现 两种都认
func replyInt64(v any) (int64, bool) {
	switch t := v.(type) {
	case int64:
		return t, true
	case int:
		return int64(t), true
	case string:
		n, err := strconv.ParseInt(t, 10, 64)
		return n, err == nil
	case []byte:
		n, err := strconv.ParseInt(string(t), 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
}

// logQueryFallback 读不到榜时记一条 由调用方决定是否回源
func logQueryFallback(ctx context.Context, reason string) {
	logx.WithContext(ctx).Infof("热榜读未命中 转查库兜底 reason=%s", reason)
}
