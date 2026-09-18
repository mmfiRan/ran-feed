package searchservicelogic

import (
	"context"

	"ran-feed/app/rpc/search/internal/repositories"
	"ran-feed/app/rpc/search/internal/svc"
	"ran-feed/app/rpc/search/search"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/types/known/emptypb"
)

type RecordHistoryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	historyRepo repositories.SearchHistoryRepository
}

func NewRecordHistoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RecordHistoryLogic {
	return &RecordHistoryLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		historyRepo: repositories.NewSearchHistoryRepository(ctx, svcCtx.MysqlDb),
	}
}

// RecordHistory 记录一次搜索 匿名或空词忽略 去重提到最前
func (l *RecordHistoryLogic) RecordHistory(in *search.RecordHistoryReq) (*emptypb.Empty, error) {
	if in == nil || in.UserId <= 0 || in.Keyword == "" {
		return &emptypb.Empty{}, nil
	}

	if err := l.historyRepo.Upsert(in.UserId, in.Keyword); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("记录搜索历史失败"))
	}
	return &emptypb.Empty{}, nil
}
