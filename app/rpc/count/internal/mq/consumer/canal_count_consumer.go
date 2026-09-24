package consumer

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"

	"ran-feed/app/rpc/count/internal/entity/query"
	counterservicelogic "ran-feed/app/rpc/count/internal/logic/counterservice"
	"ran-feed/app/rpc/count/internal/mq/consumer/strategy"
	"ran-feed/app/rpc/count/internal/repositories"
	"ran-feed/app/rpc/count/internal/svc"
	"ran-feed/pkg/event/canal"
	"ran-feed/pkg/event/pipeline"

	// 空导入触发各表策略 init 注册
	_ "ran-feed/app/rpc/count/internal/mq/consumer/strategy/presence"
	_ "ran-feed/app/rpc/count/internal/mq/consumer/strategy/reset"
)

type CanalCountConsumer struct {
	ctx        context.Context
	svcContext *svc.ServiceContext
	logx.Logger
	countRepo     repositories.CountValueRepository
	bigVRepo      repositories.BigVRepository
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
		bigVRepo:      repositories.NewBigVRepository(ctx, svcContext.MysqlDb),
		deltaOperator: counterservicelogic.NewCountDeltaOperator(ctx, svcContext),
		consumerName:  "count.canal_consumer",
		strategies:    strategy.NewDefaultRegistry(),
	}
}

// Consume 五步管道 解析 路由 去重 落库 派发副作用 去重与落库同事务保证幂等 副作用在事务外
func (c *CanalCountConsumer) Consume(ctx context.Context, key, val string) error {
	logc.Infof(ctx, "收到canal消息: key=%s, val=%s", key, val)

	msg, err := canal.Parse(val)
	if err != nil {
		logc.Errorf(ctx, "解析canal消息失败: %v, val=%s", err, val)
		return err
	}

	tableStrategy, ok := c.strategies.Get(msg.Table())
	if !ok {
		logc.Infof(ctx, "跳过未监听表消息: table=%s", msg.RawTable)
		return nil
	}

	cs := newChangeSet()
	// 策略可选声明无关变更行 在落去重前跳过 例如 content 表的热榜分值更新与计数无关
	var filters []pipeline.RowFilter
	if skipper, ok := tableStrategy.(strategy.RowSkipper); ok {
		filters = append(filters, skipper.SkipRow)
	}
	err = pipeline.RunInTx(ctx, c.svcContext.MysqlDb.DB, c.consumerName, msg, val,
		func(ctx context.Context, tx *gorm.DB, meta pipeline.RowMeta, row, oldRow map[string]any) error {
			return c.processRow(ctx, query.Use(tx), meta, tableStrategy, row, oldRow, cs)
		}, filters...)
	if err != nil {
		return err
	}

	return c.dispatch(ctx, cs)
}

// processRow 单行处理 翻译 落库 去重由管道负责
func (c *CanalCountConsumer) processRow(ctx context.Context, tx *query.Query, meta pipeline.RowMeta, tableStrategy strategy.TableStrategy, row, oldRow map[string]any, cs *changeSet) error {
	for _, u := range tableStrategy.ExtractUpdates(ctx, meta.Op, row, oldRow) {
		applied, ownerID, err := c.applyUpdate(tx, u, meta.UpdatedAt)
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
