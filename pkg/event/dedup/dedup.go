// Package dedup 消费者行级幂等去重
package dedup

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

// tableName 跨服务公共去重表
const tableName = "ran_feed_mq_consume_dedup"

// row 去重表行
type row struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement:true"`
	Consumer  string    `gorm:"column:consumer"`
	EventID   string    `gorm:"column:event_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

// TableName 指定表名
func (*row) TableName() string {
	return tableName
}

// Gate 幂等去重
type Gate struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Gate {
	return &Gate{
		db: db,
	}
}

// InsertIfAbsent 插入成功返回 true 首次处理 命中唯一键返回 false 已处理过
func (g *Gate) InsertIfAbsent(ctx context.Context, consumer, eventID string) (bool, error) {
	if consumer == "" || eventID == "" {
		return false, nil
	}

	err := g.db.WithContext(ctx).Create(&row{
		Consumer: consumer,
		EventID:  eventID,
	}).Error
	if err == nil {
		return true, nil
	}
	if isDuplicate(err) {
		return false, nil
	}
	return false, err
}

func (g *Gate) Exists(ctx context.Context, consumer, eventID string) (bool, error) {
	if consumer == "" || eventID == "" {
		return false, nil
	}

	var cnt int64
	err := g.db.WithContext(ctx).
		Model(&row{}).
		Where("consumer = ? AND event_id = ?", consumer, eventID).
		Count(&cnt).Error
	if err != nil {
		return false, err
	}
	return cnt > 0, nil
}

// isDuplicate 唯一键冲突判定 哨兵靠 pkg/orm 开的 TranslateError 错误串兜底未开该选项的连接
func isDuplicate(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "duplicate")
}
