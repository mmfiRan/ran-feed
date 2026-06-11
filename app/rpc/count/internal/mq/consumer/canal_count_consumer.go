package consumer

import (
	"context"
	"encoding/json"
	"time"

	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/core/logx"

	"ran-feed/app/rpc/count/internal/entity/query"
	counterservicelogic "ran-feed/app/rpc/count/internal/logic/counterservice"
	"ran-feed/app/rpc/count/internal/mq/consumer/strategy"
	"ran-feed/app/rpc/count/internal/repositories"
	"ran-feed/app/rpc/count/internal/svc"

	// 空导入触发各表策略 init 注册
	_ "ran-feed/app/rpc/count/internal/mq/consumer/strategy/presence"
	_ "ran-feed/app/rpc/count/internal/mq/consumer/strategy/reset"
)

type CanalCountConsumer struct {
	ctx        context.Context
	svcContext *svc.ServiceContext
	logx.Logger
	countRepo     repositories.CountValueRepository
	dedupRepo     repositories.MqConsumeDedupRepository
	deltaOperator *counterservicelogic.CountDeltaOperator
	consumerName  string
	strategies    *strategy.Registry
}

func NewCanalCountConsumer(ctx context.Context, svcContext *svc.ServiceContext) *CanalCountConsumer {
	return &CanalCountConsumer{
		ctx:           ctx,
		svcContext:    svcContext,
		Logger:        logx.WithContext(ctx),
		countRepo:     repositories.NewCountValueRepository(ctx, svcContext.MysqlDb),
		dedupRepo:     repositories.NewMqConsumeDedupRepository(ctx, svcContext.MysqlDb),
		deltaOperator: counterservicelogic.NewCountDeltaOperator(ctx, svcContext),
		consumerName:  "count.canal_consumer",
		strategies:    strategy.NewDefaultRegistry(),
	}
}

// Consume 五步管道 解析 路由 去重 落库 派发副作用 去重与落库同事务保证幂等 副作用在事务外
func (c *CanalCountConsumer) Consume(ctx context.Context, key, val string) error {
	logc.Infof(ctx, "收到canal消息: key=%s, val=%s", key, val)

	var msg canalMessage
	if err := json.Unmarshal([]byte(val), &msg); err != nil {
		logc.Errorf(ctx, "解析canal消息失败: %v, val=%s", err, val)
		return err
	}

	tableStrategy, ok := c.strategies.Get(msg.table())
	if !ok {
		logc.Infof(ctx, "跳过未监听表消息: table=%s", msg.Table)
		return nil
	}

	eventID := msg.eventID(val)
	if eventID == "" {
		logc.Errorf(ctx, "canal消息event_id为空: table=%s", msg.Table)
		return nil
	}

	meta := rowMeta{
		table:     msg.table(),
		op:        msg.op(),
		eventID:   eventID,
		updatedAt: msg.updatedAt(),
		strategy:  tableStrategy,
	}

	cs := newChangeSet()
	err := query.Q.Transaction(func(tx *query.Query) error {
		for i, row := range msg.Data {
			if row == nil {
				continue
			}
			if err := c.processRow(ctx, tx, meta, i, row, msg.oldRow(i), cs); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return c.dispatch(ctx, cs)
}

// rowMeta 一条消息内逐行处理共享的元信息
type rowMeta struct {
	table     string
	op        string
	eventID   string
	updatedAt time.Time
	strategy  strategy.TableStrategy
}

// processRow 单行处理 去重 翻译 落库 去重命中已处理则跳过
func (c *CanalCountConsumer) processRow(ctx context.Context, tx *query.Query, meta rowMeta, idx int, row, oldRow map[string]interface{}, cs *changeSet) error {
	rowEventID := rowEventID(meta.eventID, meta.table, meta.op, row, idx)
	inserted, err := c.dedupRepo.WithTx(tx).InsertIfAbsent(c.consumerName, rowEventID)
	if err != nil {
		return err
	}
	if !inserted {
		logc.Infof(ctx, "canal消息行已处理，跳过: rowEventId=%s, table=%s", rowEventID, meta.table)
		return nil
	}

	for _, u := range meta.strategy.ExtractUpdates(ctx, meta.op, row, oldRow) {
		applied, ownerID, err := c.applyUpdate(tx, u, meta.updatedAt)
		if err != nil {
			return err
		}
		if applied {
			cs.record(u, ownerID)
		}
	}
	return nil
}

// applyUpdate 把一条 Update 落库 区分增量与清零两种动作 返回是否真的改动与归属 owner
func (c *CanalCountConsumer) applyUpdate(tx *query.Query, u strategy.Update, updatedAt time.Time) (bool, int64, error) {
	if u.TargetID <= 0 {
		return false, 0, nil
	}
	repo := c.countRepo.WithTx(tx)

	if u.Action == strategy.UpdateActionResetToZero {
		rowVal, err := repo.Get(int32(u.BizType), int32(u.TargetType), u.TargetID)
		if err != nil {
			return false, 0, err
		}
		if rowVal == nil || rowVal.Value <= 0 {
			return false, 0, nil
		}
		ownerID := u.OwnerID
		if ownerID <= 0 {
			ownerID = rowVal.OwnerID
		}
		if err := c.deltaOperator.UpdateDeltaOnlyWithRepoAndOwner(
			repo, u.BizType, u.TargetType, u.TargetID, ownerID, -rowVal.Value, updatedAt,
		); err != nil {
			return false, 0, err
		}
		return true, ownerID, nil
	}

	if u.Delta == 0 {
		return false, 0, nil
	}
	if err := c.deltaOperator.UpdateDeltaOnlyWithRepoAndOwner(
		repo, u.BizType, u.TargetType, u.TargetID, u.OwnerID, u.Delta, updatedAt,
	); err != nil {
		return false, 0, err
	}
	return true, u.OwnerID, nil
}