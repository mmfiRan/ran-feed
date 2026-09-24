package hot_update

import (
	"context"
	"strconv"

	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	luautils "ran-feed/app/rpc/content/internal/common/utils/lua"
	"ran-feed/pkg/snowflake"

	"github.com/zeromicro/go-zero/core/logx"
)

// newSnapshotID 全局唯一快照 id 雪花号
// 不用秒级时间戳 避免同秒两模式生成同一 key 导致快照残缺
func newSnapshotID() string {
	return strconv.FormatInt(snowflake.GenID(), 10)
}

// refreshSnapshot 原子裁剪主榜到 mainN 候选池 取前 topN 重建快照 切 latest 指针
// 裁剪与快照合并在同一 Lua 内执行 主榜留 mainN 候补垫 对外快照只取 topN
func (j *Job) refreshSnapshot(ctx context.Context, mainN, topN int) error {
	card, err := j.svc.Redis.ZcardCtx(ctx, rediskey.RedisFeedHotGlobalKey)
	if err != nil {
		return err
	}
	if card == 0 {
		logx.WithContext(ctx).Info("热榜跳过快照 主榜为空")
		return nil
	}
	snapshotID := newSnapshotID()
	snapshotKey := rediskey.BuildHotFeedSnapshotKey(snapshotID)
	if _, err = j.svc.Redis.EvalCtx(ctx, luautils.RebuildHotSnapshotScript, []string{
		rediskey.RedisFeedHotGlobalKey,
		snapshotKey,
		rediskey.RedisFeedHotGlobalLatestKey,
	}, strconv.Itoa(mainN), strconv.Itoa(topN), snapshotID, strconv.Itoa(defaultSnapshotTTL)); err != nil {
		return err
	}
	logx.WithContext(ctx).Infof("热榜快照重建完成 snapshotID=%s main=%d", snapshotID, card)
	return nil
}
