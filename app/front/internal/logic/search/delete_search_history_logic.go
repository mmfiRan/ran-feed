// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package search

import (
	"context"

	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	"ran-feed/app/rpc/search/search"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteSearchHistoryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteSearchHistoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteSearchHistoryLogic {
	return &DeleteSearchHistoryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DeleteSearchHistory 带 keyword 删单条 all 为 true 清空 匿名直接返回成功
func (l *DeleteSearchHistoryLogic) DeleteSearchHistory(req *types.DeleteSearchHistoryReq) (resp *types.DeleteSearchHistoryRes, err error) {
	viewerID := utils.GetContextUserIdWithDefault(l.ctx)
	if viewerID <= 0 {
		return &types.DeleteSearchHistoryRes{
			Success: true,
		}, nil
	}

	if _, err = l.svcCtx.SearchRpc.DeleteHistory(l.ctx, &search.DeleteHistoryReq{
		UserId:  viewerID,
		Keyword: req.Keyword,
		All:     req.All,
	}); err != nil {
		return nil, err
	}
	return &types.DeleteSearchHistoryRes{
		Success: true,
	}, nil
}
