package counterservicelogic

import (
	"context"
	"time"

	"ran-feed/app/rpc/count/count"
	"ran-feed/app/rpc/count/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type DecLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	deltaOperator *CountDeltaOperator
}

func NewDecLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DecLogic {
	return &DecLogic{
		ctx:           ctx,
		svcCtx:        svcCtx,
		Logger:        logx.WithContext(ctx),
		deltaOperator: NewCountDeltaOperator(ctx, svcCtx),
	}
}

func (l *DecLogic) Dec(in *count.DecReq) (*count.DecRes, error) {
	if in == nil {
		return nil, errorx.NewMsg("减少计数请求无效")
	}
	if in.BizType == count.BizType_BIZ_TYPE_UNKNOWN ||
		in.TargetType == count.TargetType_TARGET_TYPE_UNKNOWN ||
		in.TargetId <= 0 {
		return nil, errorx.NewMsg("减少计数请求无效")
	}

	if err := l.deltaOperator.UpdateDeltaOnly(
		in.BizType,
		in.TargetType,
		in.TargetId,
		-1,
		time.Now(),
	); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("更新计数失败"))
	}

	l.deltaOperator.InvalidateCountCache(in.BizType, in.TargetType, in.TargetId)

	return &count.DecRes{}, nil
}
