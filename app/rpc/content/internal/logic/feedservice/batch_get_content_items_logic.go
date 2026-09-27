package feedservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/common/component/contentresolver"
	"ran-feed/app/rpc/content/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchGetContentItemsLogic struct {
	ctx context.Context
	logx.Logger
	resolver *contentresolver.Resolver
}

func NewBatchGetContentItemsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchGetContentItemsLogic {
	return &BatchGetContentItemsLogic{
		ctx:      ctx,
		Logger:   logx.WithContext(ctx),
		resolver: svcCtx.ContentResolver,
	}
}

// BatchGetContentItems 按给定 id 批量富化 复用 feed 拼装链 仅取公开内容
func (l *BatchGetContentItemsLogic) BatchGetContentItems(in *content.BatchGetContentItemsReq) (*content.BatchGetContentItemsRes, error) {
	if in == nil || len(in.ContentIds) == 0 {
		return &content.BatchGetContentItemsRes{Items: []*content.ContentItem{}}, nil
	}

	entries, err := l.resolver.Resolve(l.ctx, in.ContentIds, in.ViewerId, true)
	if err != nil {
		return nil, err
	}
	return &content.BatchGetContentItemsRes{Items: buildContentItems(entries)}, nil
}
