// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package search

import (
	"context"
	"strings"

	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/search/search"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type SearchContentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSearchContentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchContentLogic {
	return &SearchContentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SearchContentLogic) SearchContent(req *types.SearchContentReq) (resp *types.SearchContentRes, err error) {
	resp = &types.SearchContentRes{Items: []types.SearchContentItem{}}
	if req == nil || strings.TrimSpace(req.Keyword) == "" {
		return resp, nil
	}

	viewerID := utils.GetContextUserIdWithDefault(l.ctx)

	// search-rpc 只排序 出 ranked content_ids 与高亮
	searchRes, err := l.svcCtx.SearchRpc.SearchContent(l.ctx, &search.SearchContentReq{
		Keyword:     req.Keyword,
		ContentType: search.ContentType(req.ContentType),
		Page:        req.Page,
		Size:        req.Size,
	})
	if err != nil {
		return nil, err
	}
	if searchRes == nil || len(searchRes.Hits) == 0 {
		return resp, nil
	}

	ids := make([]int64, 0, len(searchRes.Hits))
	highlightMap := make(map[int64]*search.ContentHit, len(searchRes.Hits))
	for _, h := range searchRes.Hits {
		ids = append(ids, h.ContentId)
		highlightMap[h.ContentId] = h
	}

	// content-rpc 复用 feed 拼装链富化 按 id 顺序返回
	enriched, err := l.svcCtx.FeedRpc.BatchGetContentItems(l.ctx, &content.BatchGetContentItemsReq{
		ContentIds: ids,
		ViewerId:   viewerID,
	})
	if err != nil {
		return nil, err
	}

	resp.Items = assembleSearchContentItems(enriched.Items, highlightMap)
	resp.Total = searchRes.Total

	recordSearchHistory(l.svcCtx, viewerID, req.Keyword)
	return resp, nil
}
