package searchservicelogic

import (
	"context"

	"ran-feed/app/rpc/search/internal/svc"
	"ran-feed/app/rpc/search/search"

	"github.com/zeromicro/go-zero/core/logx"
)

type SearchUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSearchUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchUserLogic {
	return &SearchUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SearchUserLogic) SearchUser(in *search.SearchUserReq) (*search.SearchUserRes, error) {
	// todo: add your logic here and delete this line

	return &search.SearchUserRes{}, nil
}
