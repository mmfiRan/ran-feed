package xxljob

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/threading"
	"github.com/zeromicro/go-zero/core/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type TaskHandler func(ctx context.Context, param TriggerParam) (string, error)

type TaskMiddleWare func(next TaskHandler) TaskHandler

type runningTask struct {
	cancel context.CancelFunc
	jobID  int64
}

type taskSlot struct {
	mu      sync.Mutex
	running *runningTask
}

type taskRunner struct {
	handlers    map[string]TaskHandler
	middlewares []TaskMiddleWare
	slots       map[string]*taskSlot
	mu          sync.RWMutex
}

func newTaskRunner() *taskRunner {
	return &taskRunner{
		handlers: make(map[string]TaskHandler),
		slots:    make(map[string]*taskSlot),
	}
}

func (r *taskRunner) register(name string, h TaskHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[name] = h
	if _, ok := r.slots[name]; !ok {
		r.slots[name] = &taskSlot{}
	}
}

func (r *taskRunner) use(mw TaskMiddleWare) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.middlewares = append(r.middlewares, mw)
}

func (r *taskRunner) get(name string) (TaskHandler, *taskSlot, bool) {
	r.mu.RLock()
	h, ok := r.handlers[name]
	s := r.slots[name]
	mws := append([]TaskMiddleWare(nil), r.middlewares...)
	r.mu.RUnlock()
	if !ok {
		return nil, nil, false
	}
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h, s, true
}

func (r *taskRunner) start(ctx context.Context, param TriggerParam, client *AdminClient, logger Logger) error {
	h, slot, ok := r.get(param.ExecutorHandler)
	if !ok {
		return errors.New("handler not found")
	}
	strategy := BlockStrategy(param.ExecutorBlockStrategy)

	slot.mu.Lock()
	if slot.running != nil {
		switch strategy {
		case BlockCover:
			slot.running.cancel()
		case BlockDiscard:
			slot.mu.Unlock()
			return errors.New("discard later")
		default:
			slot.mu.Unlock()
			return errors.New("serial execution")
		}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if param.ExecutorTimeout > 0 {
		var timeoutCancel context.CancelFunc
		ctx, timeoutCancel = context.WithTimeout(ctx, time.Duration(param.ExecutorTimeout)*time.Second)
		defer timeoutCancel()
	}
	ctx, cancel := context.WithCancel(ctx)
	slot.running = &runningTask{cancel: cancel, jobID: param.JobID}
	slot.mu.Unlock()

	threading.GoSafe(func() {
		startedAt := time.Now()
		// 每次执行起一个根 span 触发外部为 xxl admin 视为 server 端
		// 任务 ctx 来自 admin HTTP 请求 不经 trace 拦截器 无 span 会导致所有日志缺 traceId
		// 在此起 span 让 begin/fail/finish/callback 全部日志带同一 traceId 且下游 DB/RPC span 正确嵌套
		var span oteltrace.Span
		ctx, span = trace.TracerFromContext(ctx).Start(
			ctx,
			"xxljob/"+param.ExecutorHandler,
			oteltrace.WithSpanKind(oteltrace.SpanKindServer),
		)
		defer span.End()
		logInfo(
			ctx,
			logger,
			"xxljob: execute begin handler=%s jobId=%d logId=%d timeout=%ds params=%q",
			param.ExecutorHandler,
			param.JobID,
			param.LogID,
			param.ExecutorTimeout,
			shortenForLog(param.ExecutorParams, 256),
		)
		defer func() {
			slot.mu.Lock()
			slot.running = nil
			slot.mu.Unlock()
		}()
		defer func() {
			if rec := recover(); rec != nil {
				msg := fmt.Sprintf("panic: %v", rec)
				logError(
					ctx,
					logger,
					"xxljob: execute panic handler=%s jobId=%d logId=%d cost=%s err=%s",
					param.ExecutorHandler,
					param.JobID,
					param.LogID,
					time.Since(startedAt),
					msg,
				)
				if client != nil && param.LogID != 0 {
					if err := client.Callback(context.Background(), []HandleCallbackParam{{
						LogID:       param.LogID,
						LogDateTime: param.LogDateTime,
						HandleCode:  FailCode,
						HandleMsg:   msg,
					}}); err != nil {
						logError(ctx, logger, "xxljob: callback panic result failed handler=%s jobId=%d logId=%d err=%v", param.ExecutorHandler, param.JobID, param.LogID, err)
					}
				}
			}
		}()

		msg, err := h(ctx, param)
		code := SuccessCode
		if err != nil {
			code = FailCode
			if msg == "" {
				msg = err.Error()
			}
			logError(
				ctx,
				logger,
				"xxljob: execute failed handler=%s jobId=%d logId=%d cost=%s err=%v msg=%q",
				param.ExecutorHandler,
				param.JobID,
				param.LogID,
				time.Since(startedAt),
				err,
				shortenForLog(msg, 256),
			)
		} else {
			logInfo(
				ctx,
				logger,
				"xxljob: execute finished handler=%s jobId=%d logId=%d cost=%s msg=%q",
				param.ExecutorHandler,
				param.JobID,
				param.LogID,
				time.Since(startedAt),
				shortenForLog(msg, 256),
			)
		}
		if client != nil && param.LogID != 0 {
			if err := client.Callback(context.Background(), []HandleCallbackParam{
				{
					LogID:       param.LogID,
					LogDateTime: param.LogDateTime,
					HandleCode:  code,
					HandleMsg:   msg,
				},
			}); err != nil {
				logError(ctx, logger, "xxljob: callback failed handler=%s jobId=%d logId=%d code=%d err=%v", param.ExecutorHandler, param.JobID, param.LogID, code, err)
			} else {
				logInfo(ctx, logger, "xxljob: callback success handler=%s jobId=%d logId=%d code=%d", param.ExecutorHandler, param.JobID, param.LogID, code)
			}
		}
	})

	return nil
}

func (r *taskRunner) kill(handler string) bool {
	r.mu.RLock()
	slot := r.slots[handler]
	r.mu.RUnlock()
	if slot == nil {
		return false
	}
	slot.mu.Lock()
	defer slot.mu.Unlock()
	if slot.running == nil {
		return false
	}
	slot.running.cancel()
	slot.running = nil
	return true
}

func (r *taskRunner) idle(handler string) bool {
	r.mu.RLock()
	slot := r.slots[handler]
	r.mu.RUnlock()
	if slot == nil {
		return true
	}
	slot.mu.Lock()
	defer slot.mu.Unlock()
	return slot.running == nil
}
