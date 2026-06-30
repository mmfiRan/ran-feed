package feedservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchGetContentItemsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBatchGetContentItemsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchGetContentItemsLogic {
	return &BatchGetContentItemsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// BatchGetContentItems 按给定 id 批量富化 复用 feed 拼装链 仅取公开内容
func (l *BatchGetContentItemsLogic) BatchGetContentItems(in *content.BatchGetContentItemsReq) (*content.BatchGetContentItemsRes, error) {
	if in == nil || len(in.ContentIds) == 0 {
		return &content.BatchGetContentItemsRes{Items: []*content.ContentItem{}}, nil
	}

	resolver := newContentDetailResolver(l.ctx, l.svcCtx)
	items, err := resolver.assembleItems(in.ContentIds, in.ViewerId, true)
	if err != nil {
		return nil, err
	}
	return &content.BatchGetContentItemsRes{Items: items}, nil
}
