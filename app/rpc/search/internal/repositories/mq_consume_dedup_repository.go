package repositories

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"ran-feed/app/rpc/search/internal/entity/model"
	"ran-feed/app/rpc/search/internal/entity/query"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/logx"
)

type MqConsumeDedupRepository interface {
	InsertIfAbsent(consumer, eventID string) (bool, error)
}

type mqConsumeDedupRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	logx.Logger
}

func NewMqConsumeDedupRepository(ctx context.Context, db *orm.DB) MqConsumeDedupRepository {
	return &mqConsumeDedupRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

// InsertIfAbsent 插入成功返回 true 已存在返回 false 充当幂等闸
func (r *mqConsumeDedupRepositoryImpl) InsertIfAbsent(consumer, eventID string) (bool, error) {
	if consumer == "" || eventID == "" {
		return false, nil
	}

	record := &model.RanFeedMqConsumeDedup{
		Consumer: consumer,
		EventID:  eventID,
	}

	err := query.Q.RanFeedMqConsumeDedup.WithContext(r.ctx).Create(record)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return false, nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
		return false, nil
	}
	return false, err
}
