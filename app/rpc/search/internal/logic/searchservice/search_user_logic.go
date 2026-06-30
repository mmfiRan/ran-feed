package searchservicelogic

import (
	"context"
	"strconv"

	"ran-feed/app/rpc/search/internal/common/consts"
	"ran-feed/app/rpc/search/internal/es"
	"ran-feed/app/rpc/search/internal/svc"
	"ran-feed/app/rpc/search/search"
	"ran-feed/pkg/errorx"

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
	if in == nil || in.Keyword == "" {
		return &search.SearchUserRes{}, nil
	}

	from, size := pageToFromSize(in.Page, in.Size)
	result, err := es.Search(l.ctx, l.svcCtx.ES, es.IndexUser, l.buildQuery(in, from, size))
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("用户搜索失败"))
	}

	hits := make([]*search.UserHit, 0, len(result.Hits))
	for _, h := range result.Hits {
		userID, parseErr := strconv.ParseInt(h.ID, 10, 64)
		if parseErr != nil {
			l.Errorf("用户命中 id 解析失败 id=%s err=%v", h.ID, parseErr)
			continue
		}
		hits = append(hits, &search.UserHit{
			UserId: userID,
			Score:  h.Score,
		})
	}
	return &search.SearchUserRes{Hits: hits, Total: result.Total}, nil
}

// buildQuery 昵称^3 简介 多字段匹配 仅正常未删除 排序 _score
func (l *SearchUserLogic) buildQuery(in *search.SearchUserReq, from, size int) map[string]any {
	return map[string]any{
		"from": from,
		"size": size,
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{"multi_match": map[string]any{
						"query":    in.Keyword,
						"fields":   []string{"nickname^3", "bio"},
						"analyzer": "ik_smart",
					}},
				},
				"filter": []map[string]any{
					{"term": map[string]any{"status": consts.UserStatusNormal}},
					{"term": map[string]any{"is_deleted": 0}},
				},
			},
		},
		"sort": []map[string]any{
			{"_score": map[string]any{"order": "desc"}},
		},
	}
}
