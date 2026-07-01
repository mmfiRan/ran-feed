// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package search

import (
	"context"
	"strings"

	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	"ran-feed/app/rpc/search/search"

	"github.com/zeromicro/go-zero/core/logx"
)

// suggestTypeName 补全项类型枚举转前端友好字符串
func suggestTypeName(t search.SuggestType) string {
	switch t {
	case search.SuggestType_SUGGEST_CONTENT:
		return "content"
	case search.SuggestType_SUGGEST_USER:
		return "user"
	default:
		return ""
	}
}

type SuggestLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSuggestLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SuggestLogic {
	return &SuggestLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SuggestLogic) Suggest(req *types.SuggestReq) (resp *types.SuggestRes, err error) {
	resp = &types.SuggestRes{Items: []types.SuggestItem{}}
	if req == nil || strings.TrimSpace(req.Keyword) == "" {
		return resp, nil
	}

	// search-rpc 跨内容/用户两索引前缀补全 只出建议词与类型 不富化
	suggestRes, err := l.svcCtx.SearchRpc.Suggest(l.ctx, &search.SuggestReq{
		Keyword: req.Keyword,
		Size:    req.Size,
	})
	if err != nil {
		return nil, err
	}

	for _, it := range suggestRes.Items {
		resp.Items = append(resp.Items, types.SuggestItem{
			Text: it.Text,
			Type: suggestTypeName(it.Type),
		})
	}
	return resp, nil
}
