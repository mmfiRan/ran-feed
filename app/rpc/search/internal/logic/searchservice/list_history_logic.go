package searchservicelogic

import (
	"context"

	"ran-feed/app/rpc/search/internal/repositories"
	"ran-feed/app/rpc/search/internal/svc"
	"ran-feed/app/rpc/search/search"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

// defaultHistoryLimit 历史记录默认返回条数
const defaultHistoryLimit = 10

type ListHistoryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	historyRepo repositories.SearchHistoryRepository
}

func NewListHistoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListHistoryLogic {
	return &ListHistoryLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		historyRepo: repositories.NewSearchHistoryRepository(ctx, svcCtx.MysqlDb),
	}
}

// ListHistory 取最近 limit 条 匿名返回空
func (l *ListHistoryLogic) ListHistory(in *search.ListHistoryReq) (*search.ListHistoryRes, error) {
	if in == nil || in.UserId <= 0 {
		return &search.ListHistoryRes{}, nil
	}

	limit := int(in.Limit)
	if limit <= 0 {
		limit = defaultHistoryLimit
	}

	rows, err := l.historyRepo.ListRecent(in.UserId, limit)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询搜索历史失败"))
	}

	items := make([]*search.HistoryItem, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		items = append(items, &search.HistoryItem{
			Keyword:   row.Keyword,
			UpdatedAt: row.UpdatedAt.UnixMilli(),
		})
	}
	return &search.ListHistoryRes{
		Items: items,
	}, nil
}
