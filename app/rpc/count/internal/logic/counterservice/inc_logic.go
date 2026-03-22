package counterservicelogic

import (
	"context"
	"time"

	"ran-feed/app/rpc/count/count"
	"ran-feed/app/rpc/count/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type IncLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	deltaOperator *CountDeltaOperator
}

func NewIncLogic(ctx context.Context, svcCtx *svc.ServiceContext) *IncLogic {
	return &IncLogic{
		ctx:           ctx,
		svcCtx:        svcCtx,
		Logger:        logx.WithContext(ctx),
		deltaOperator: NewCountDeltaOperator(ctx, svcCtx),
	}
}

func (l *IncLogic) Inc(in *count.IncReq) (*count.IncRes, error) {
	if in == nil {
		return nil, errorx.NewMsg("新增计数请求无效")
	}
	if in.BizType == count.BizType_BIZ_TYPE_UNKNOWN ||
		in.TargetType == count.TargetType_TARGET_TYPE_UNKNOWN ||
		in.TargetId <= 0 {
		return nil, errorx.NewMsg("新增计数请求无效")
	}

	err := l.deltaOperator.UpdateDeltaOnly(
		in.BizType,
		in.TargetType,
		in.TargetId,
		1,
		time.Now(),
	)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("更新计数失败"))
	}

	l.deltaOperator.InvalidateCountCache(in.BizType, in.TargetType, in.TargetId)

	return &count.IncRes{}, nil
}
