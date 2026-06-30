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

type ListSearchHistoryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListSearchHistoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListSearchHistoryLogic {
	return &ListSearchHistoryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ListSearchHistory 取当前登录用户最近搜索历史 匿名返回空
func (l *ListSearchHistoryLogic) ListSearchHistory() (resp *types.ListSearchHistoryRes, err error) {
	resp = &types.ListSearchHistoryRes{Items: []types.SearchHistoryItem{}}

	viewerID := utils.GetContextUserIdWithDefault(l.ctx)
	if viewerID <= 0 {
		return resp, nil
	}

	historyRes, err := l.svcCtx.SearchRpc.ListHistory(l.ctx, &search.ListHistoryReq{UserId: viewerID})
	if err != nil {
		return nil, err
	}

	items := make([]types.SearchHistoryItem, 0, len(historyRes.Items))
	for _, it := range historyRes.Items {
		if it == nil {
			continue
		}
		items = append(items, types.SearchHistoryItem{
			Keyword:   it.Keyword,
			UpdatedAt: it.UpdatedAt,
		})
	}
	resp.Items = items
	return resp, nil
}
