package searchservicelogic

import (
	"context"

	"ran-feed/app/rpc/search/internal/svc"
	"ran-feed/app/rpc/search/search"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListHistoryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListHistoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListHistoryLogic {
	return &ListHistoryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListHistoryLogic) ListHistory(in *search.ListHistoryReq) (*search.ListHistoryRes, error) {
	// todo: add your logic here and delete this line

	return &search.ListHistoryRes{}, nil
}
