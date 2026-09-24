// Package outbox_reconcile content 域事件链对账 补跑无幂等记录的 outbox 事件并清理保留期外数据
package outbox_reconcile

import (
	"context"
	"fmt"
	"time"

	"ran-feed/app/rpc/content/internal/mq/consumer"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/event/contentevent"
	"ran-feed/pkg/event/dedup"
	"ran-feed/pkg/xxljob"

	"github.com/zeromicro/go-zero/core/logx"
)

const HandlerName = "content.outbox.reconcile"

const (
	// scanWindow 扫描窗口 覆盖 kafka 抖动与短时积压
	scanWindow = time.Hour
	// inFlightBuffer 在途缓冲 跳过最近这段避免与正在消费的事件重复
	inFlightBuffer = 2 * time.Minute
	// outboxRetention 发件箱保留期
	outboxRetention = 30 * 24 * time.Hour
	// dedupRetention 去重表保留期 须长于扫描窗口
	dedupRetention = 7 * 24 * time.Hour
	// reconcileBatch 单批扫描条数
	reconcileBatch = 500
)

// OutboxReconcileJob 对账作业 整条事件链唯一的可靠性保证 丢失最多延迟一个调度周期被补回
type OutboxReconcileJob struct {
	logx.Logger
	outboxRepo repositories.ContentOutboxRepository
	consumer   *consumer.ContentEventConsumer
	dedupGate  *dedup.Gate
}

func Register(ctx context.Context, executor *xxljob.Executor, svcCtx *svc.ServiceContext) {
	job := &OutboxReconcileJob{
		Logger:     logx.WithContext(ctx),
		outboxRepo: repositories.NewContentOutboxRepository(ctx, svcCtx.MysqlDb),
		consumer:   consumer.NewContentEventConsumer(ctx, svcCtx),
		dedupGate:  dedup.New(svcCtx.MysqlDb.DB),
	}
	executor.RegisterTask(HandlerName, job.Run)
}

// Run 补跑窗口内漏消费事件 再按保留期清理两表 清去重表须先于清 outbox 保证保留期约束
func (j *OutboxReconcileJob) Run(ctx context.Context, _ xxljob.TriggerParam) (string, error) {
	now := time.Now()
	from, to := now.Add(-scanWindow), now.Add(-inFlightBuffer)

	replayed, err := j.replayMissed(ctx, from, to)
	if err != nil {
		return "", err
	}
	if _, err := j.dedupGate.DeleteBefore(ctx, now.Add(-dedupRetention)); err != nil {
		return "", fmt.Errorf("清理去重表失败 %w", err)
	}
	if _, err := j.outboxRepo.DeleteEventsBefore(ctx, now.Add(-outboxRetention)); err != nil {
		return "", fmt.Errorf("清理发件箱失败 %w", err)
	}

	j.Infof("事件对账完成 replayed=%d", replayed)
	return fmt.Sprintf("ok replayed=%d", replayed), nil
}

// replayMissed id 游标分页补跑窗口内无幂等记录的事件 解析或补跑失败只记日志下轮重试
func (j *OutboxReconcileJob) replayMissed(ctx context.Context, from, to time.Time) (int, error) {
	total := 0
	afterID := int64(0)
	for {
		rows, err := j.outboxRepo.ListUnconsumedEvents(ctx, consumer.FeedConsumerName, from, to, afterID, reconcileBatch)
		if err != nil {
			return total, err
		}
		if len(rows) == 0 {
			return total, nil
		}
		for _, row := range rows {
			if row == nil {
				continue
			}
			afterID = row.ID
			evt, uerr := contentevent.UnmarshalContentEvent(row.Payload)
			if uerr != nil {
				j.Errorf("对账解析事件失败跳过 eid=%s err=%v", row.EventID, uerr)
				continue
			}
			if rerr := j.consumer.Replay(ctx, row.EventID, evt); rerr != nil {
				j.Errorf("对账补跑失败 eid=%s err=%v", row.EventID, rerr)
				continue
			}
			total++
		}
		if len(rows) < reconcileBatch {
			return total, nil
		}
	}
}
