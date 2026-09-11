// Package presence 出现消失型计数策略 互动新增即加一 取消即减一
// 覆盖 like favorite comment follow 由判活谓词与目标映射两个轴配置 共用同一套增量算法
package presence

import (
	"context"
	"strings"

	"github.com/zeromicro/go-zero/core/logc"

	"ran-feed/app/rpc/count/count"
	countenum "ran-feed/app/rpc/count/internal/common/enums"
	"ran-feed/app/rpc/count/internal/mq/consumer/strategy"
	"ran-feed/pkg/enums"
)

// presenceCounterStrategy 把一行互动变更映射为若干计数目标上的同一增量
type presenceCounterStrategy struct {
	tableName string
	isActive  func(row map[string]interface{}) bool                               // 轴A 判活谓词
	targetsOf func(ctx context.Context, row map[string]interface{}) []countTarget // 轴B 行到计数目标
}

// countTarget 一条增量要落到的计数对象
type countTarget struct {
	bizType    count.BizType
	targetType count.TargetType
	targetID   int64
	ownerID    int64
}

func (s *presenceCounterStrategy) TableName() string {
	return s.tableName
}

func (s *presenceCounterStrategy) ExtractUpdates(ctx context.Context, op string, row map[string]interface{}, oldRow map[string]interface{}) []strategy.Update {
	delta := presenceDelta(op, s.isActive, row, oldRow)
	if delta == 0 {
		return nil
	}
	targets := s.targetsOf(ctx, row)
	if len(targets) == 0 {
		return nil
	}
	updates := make([]strategy.Update, 0, len(targets))
	for _, t := range targets {
		if t.targetID <= 0 {
			continue
		}
		updates = append(updates, strategy.Update{
			BizType:    t.bizType,
			TargetType: t.targetType,
			TargetID:   t.targetID,
			Delta:      delta,
			OwnerID:    t.ownerID,
		})
	}
	return updates
}

// presenceDelta op 加 判活谓词得出增量 新增活跃加一 删除前活跃减一 更新看活跃翻转
func presenceDelta(op string, isActive func(map[string]interface{}) bool, row, oldRow map[string]interface{}) int64 {
	switch strings.ToUpper(strings.TrimSpace(op)) {
	case "INSERT":
		if isActive(row) {
			return 1
		}
	case "DELETE":
		if isActive(beforeView(row, oldRow)) {
			return -1
		}
	case "UPDATE":
		return boolDelta(isActive(beforeView(row, oldRow)), isActive(row))
	}
	return 0
}

// beforeView 还原变更前整行 canal 的 Old 只带改动列 用其覆盖新行得到旧视图
func beforeView(row, oldRow map[string]interface{}) map[string]interface{} {
	if len(oldRow) == 0 {
		return row
	}
	merged := make(map[string]interface{}, len(row))
	for k, v := range row {
		merged[k] = v
	}
	for k, v := range oldRow {
		merged[k] = v
	}
	return merged
}

func boolDelta(before, after bool) int64 {
	if !before && after {
		return 1
	}
	if before && !after {
		return -1
	}
	return 0
}

// statusActive status 为正常即有效
func statusActive(row map[string]interface{}) bool {
	v, ok := strategy.ParseInt64(row["status"])
	if !ok {
		return false
	}
	return countenum.RecordStatus(int32(v)).IsActive()
}

// statusActiveNotDeleted status 正常且未逻辑删除
func statusActiveNotDeleted(row map[string]interface{}) bool {
	if !statusActive(row) {
		return false
	}
	v, ok := strategy.ParseInt64(row["is_deleted"])
	if !ok {
		return true
	}
	return !enums.IsDeleted(int32(v)).IsDel()
}

// alwaysActive 无状态字段的表恒计数 如 favorite
func alwaysActive(map[string]interface{}) bool {
	return true
}

// contentTargets 行映射到单个内容计数 取 content_id 与作者 content_user_id
func contentTargets(bizType count.BizType) func(ctx context.Context, row map[string]interface{}) []countTarget {
	return func(ctx context.Context, row map[string]interface{}) []countTarget {
		contentID, ok := strategy.ParseInt64(row["content_id"])
		if !ok || contentID <= 0 {
			logc.Errorf(ctx, "canal消息缺少有效content_id biz=%d row=%v", bizType, row)
			return nil
		}
		ownerID, ok := strategy.ParseInt64(row["content_user_id"])
		if !ok || ownerID <= 0 {
			logc.Errorf(ctx, "canal消息缺少有效content_user_id biz=%d row=%v", bizType, row)
			ownerID = 0
		}
		return []countTarget{{
			bizType:    bizType,
			targetType: count.TargetType_TARGET_TYPE_CONTENT,
			targetID:   contentID,
			ownerID:    ownerID,
		}}
	}
}

// followTargets 一行关注同时影响关注者的关注数与被关注者的粉丝数
func followTargets(ctx context.Context, row map[string]interface{}) []countTarget {
	userID, ok := strategy.ParseInt64(row["user_id"])
	if !ok || userID <= 0 {
		logc.Errorf(ctx, "canal消息缺少有效user_id row=%v", row)
		return nil
	}
	followUserID, ok := strategy.ParseInt64(row["follow_user_id"])
	if !ok || followUserID <= 0 {
		logc.Errorf(ctx, "canal消息缺少有效follow_user_id row=%v", row)
		return nil
	}
	return []countTarget{
		{bizType: count.BizType_BIZ_TYPE_FOLLOWING, targetType: count.TargetType_TARGET_TYPE_USER, targetID: userID},
		{bizType: count.BizType_BIZ_TYPE_FOLLOWED, targetType: count.TargetType_TARGET_TYPE_USER, targetID: followUserID},
	}
}
