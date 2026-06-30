package searchservicelogic

import (
	"context"

	"ran-feed/app/rpc/search/internal/svc"
	"ran-feed/app/rpc/search/search"

	"github.com/zeromicro/go-zero/core/logx"
)

type SearchContentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSearchContentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchContentLogic {
	return &SearchContentLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SearchContentLogic) SearchContent(in *search.SearchContentReq) (*search.SearchContentRes, error) {
	// todo: add your logic here and delete this line

	return &search.SearchContentRes{}, nil
}
