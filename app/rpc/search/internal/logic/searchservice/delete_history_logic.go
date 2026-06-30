package searchservicelogic

import (
	"context"

	"ran-feed/app/rpc/search/internal/svc"
	"ran-feed/app/rpc/search/search"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteHistoryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteHistoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteHistoryLogic {
	return &DeleteHistoryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteHistoryLogic) DeleteHistory(in *search.DeleteHistoryReq) (*search.DeleteHistoryRes, error) {
	// todo: add your logic here and delete this line

	return &search.DeleteHistoryRes{}, nil
}
