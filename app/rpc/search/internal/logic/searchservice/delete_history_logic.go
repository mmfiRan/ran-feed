package searchservicelogic

import (
	"context"

	"ran-feed/app/rpc/search/internal/repositories"
	"ran-feed/app/rpc/search/internal/svc"
	"ran-feed/app/rpc/search/search"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteHistoryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	historyRepo repositories.SearchHistoryRepository
}

func NewDeleteHistoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteHistoryLogic {
	return &DeleteHistoryLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		historyRepo: repositories.NewSearchHistoryRepository(ctx, svcCtx.MysqlDb),
	}
}

// DeleteHistory all 为 true 清空 否则删单条 匿名忽略
func (l *DeleteHistoryLogic) DeleteHistory(in *search.DeleteHistoryReq) (*search.DeleteHistoryRes, error) {
	if in == nil || in.UserId <= 0 {
		return &search.DeleteHistoryRes{}, nil
	}

	var err error
	if in.All {
		err = l.historyRepo.Clear(in.UserId)
	} else {
		err = l.historyRepo.DeleteOne(in.UserId, in.Keyword)
	}
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("删除搜索历史失败"))
	}
	return &search.DeleteHistoryRes{}, nil
}
