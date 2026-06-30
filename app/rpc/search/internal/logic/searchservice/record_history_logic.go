package searchservicelogic

import (
	"context"

	"ran-feed/app/rpc/search/internal/svc"
	"ran-feed/app/rpc/search/search"

	"github.com/zeromicro/go-zero/core/logx"
)

type RecordHistoryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRecordHistoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RecordHistoryLogic {
	return &RecordHistoryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RecordHistoryLogic) RecordHistory(in *search.RecordHistoryReq) (*search.RecordHistoryRes, error) {
	// todo: add your logic here and delete this line

	return &search.RecordHistoryRes{}, nil
}
