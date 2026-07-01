package feedservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

// maxIndexPageSize 全量重建单页上限 防止调用方传入过大 limit
const maxIndexPageSize = 500

type ListContentForIndexLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListContentForIndexLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListContentForIndexLogic {
	return &ListContentForIndexLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ListContentForIndex 全量重建 按 id 游标扫可索引内容 组装原始投影 空返回表示扫完
func (l *ListContentForIndexLogic) ListContentForIndex(in *content.ListContentForIndexReq) (*content.ListContentForIndexRes, error) {
	if in == nil {
		return &content.ListContentForIndexRes{Items: []*content.ContentIndexItem{}}, nil
	}

	limit := int(in.Limit)
	if limit <= 0 || limit > maxIndexPageSize {
		limit = maxIndexPageSize
	}

	contentRepo := repositories.NewContentRepository(l.ctx, l.svcCtx.MysqlDb)
	rows, err := contentRepo.ScanIndexableByIDCursor(in.Cursor, limit)
	if err != nil {
		return nil, err
	}

	items, err := assembleContentIndexItems(l.ctx, l.svcCtx, rows)
	if err != nil {
		return nil, err
	}
	return &content.ListContentForIndexRes{Items: items}, nil
}
