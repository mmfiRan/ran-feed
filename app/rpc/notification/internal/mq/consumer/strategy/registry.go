// Package strategy 通知域每表策略基础设施
// 复用 count presence 翻转判定的思路 但输出物是 NotifyEvent 而非计数 Delta
// 每张源表(like/favorite/comment/follow)一个 TableStrategy 通过 init 自注册进 Registry
package strategy

import (
	"context"

	notifyenum "ran-feed/app/rpc/notification/internal/common/enums"
	"ran-feed/pkg/event/registry"
)

// PersistAction 决定 event 如何落库
type PersistAction int

const (
	// PersistAggregate 聚合 upsert(LIKE_FAVORITE) 命中 uk_recipient_aggkey → agg_count+1 + re-surface
	// FOLLOW 亦复用该动作 意在依赖 uk 收敛「取关→再关注」到同一行并 re-surface
	// FOLLOW 上 agg_count 无产品含义 客户端渲染忽略即可
	PersistAggregate PersistAction = iota
	// PersistInsertOne 单条 insert(COMMENT_REPLY) agg_key=CR:{comment_id} 天然唯一 无冲突
	PersistInsertOne
)

// NotifyEvent 策略输出的中间物 由 consumer 落库同事务
// recipient/actor/notifyType/aggKey 必填 其余按类型选择性填
type NotifyEvent struct {
	RecipientID int64
	ActorID     int64
	NotifyType  notifyenum.NotifyTypeEnum
	AggKey      string
	Action      PersistAction
	ContentID   int64
	CommentID   int64
	Snippet     string
}

// TableStrategy 定义某张源表在 Canal 消息中翻译为 NotifyEvent 的策略
type TableStrategy interface {
	TableName() string
	// ExtractEvents 输入 canal op(INSERT/UPDATE/DELETE) 与前后行状态 返回可落库的通知事件
	// 内部负责激活态翻转判定与 actor==recipient 自我过滤 空返回代表本行不产通知
	ExtractEvents(ctx context.Context, op string, row map[string]interface{}, oldRow map[string]interface{}) []NotifyEvent
}

// Registry 管理 table 到 strategy 的映射
type Registry = registry.Registry[TableStrategy]

var factories []func() TableStrategy

// RegisterFactory 子包通过 init 注册各自表策略工厂
func RegisterFactory(factory func() TableStrategy) {
	if factory == nil {
		return
	}
	factories = append(factories, factory)
}

// NewDefaultRegistry 实例化所有已注册的表策略
func NewDefaultRegistry() *Registry {
	strategies := make([]TableStrategy, 0, len(factories))
	for _, f := range factories {
		if f == nil {
			continue
		}
		if s := f(); s != nil {
			strategies = append(strategies, s)
		}
	}
	return registry.New(strategies...)
}
