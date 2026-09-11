package counterservicelogic

import (
	"context"

	"ran-feed/app/rpc/count/count"
	"ran-feed/app/rpc/count/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

// countBatchGetter 批量取数能力 抽接口便于测试注入
type countBatchGetter interface {
	BatchGetCount(in *count.BatchGetCountReq) (*count.BatchGetCountRes, error)
}

type BatchGetContentCountsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	batchGetter countBatchGetter
}

func NewBatchGetContentCountsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchGetContentCountsLogic {
	return &BatchGetContentCountsLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		batchGetter: NewBatchGetCountLogic(ctx, svcCtx),
	}
}

// BatchGetContentCounts 按内容ID批量取互动计数 复用 BatchGetCount 的缓存与批量取数
func (l *BatchGetContentCountsLogic) BatchGetContentCounts(in *count.BatchGetContentCountsReq) (*count.BatchGetContentCountsRes, error) {
	if in == nil {
		return nil, errorx.NewMsg("批量查询内容计数请求无效")
	}

	contentIDs := normalizeContentIDs(in.GetContentIds())
	if len(contentIDs) == 0 {
		return &count.BatchGetContentCountsRes{}, nil
	}

	resp, err := l.batchGetter.BatchGetCount(&count.BatchGetCountReq{
		Keys: buildContentCountKeys(contentIDs),
	})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("批量查询内容计数失败"))
	}

	countsByID := foldContentCounts(resp.GetItems())
	items := make([]*count.ContentCountsItem, 0, len(contentIDs))
	for _, contentID := range contentIDs {
		item := countsByID[contentID]
		// 无记录的内容补零值 保证返回与入参一一对应
		if item == nil {
			item = &count.ContentCountsItem{ContentId: contentID}
		}
		items = append(items, item)
	}

	return &count.BatchGetContentCountsRes{Items: items}, nil
}

// normalizeContentIDs 去重 过滤非正 保持入参顺序
func normalizeContentIDs(contentIDs []int64) []int64 {
	seen := make(map[int64]struct{}, len(contentIDs))
	ids := make([]int64, 0, len(contentIDs))
	for _, id := range contentIDs {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

// buildContentCountKeys 每个内容组三个计数键 点赞 收藏 评论
func buildContentCountKeys(contentIDs []int64) []*count.CountKey {
	keys := make([]*count.CountKey, 0, len(contentIDs)*3)
	for _, contentID := range contentIDs {
		keys = append(keys,
			&count.CountKey{BizType: count.BizType_BIZ_TYPE_LIKE, TargetType: count.TargetType_TARGET_TYPE_CONTENT, TargetId: contentID},
			&count.CountKey{BizType: count.BizType_BIZ_TYPE_FAVORITE, TargetType: count.TargetType_TARGET_TYPE_CONTENT, TargetId: contentID},
			&count.CountKey{BizType: count.BizType_BIZ_TYPE_COMMENT, TargetType: count.TargetType_TARGET_TYPE_CONTENT, TargetId: contentID},
		)
	}
	return keys
}

// foldContentCounts 扁平计数项折成 内容ID 到 三项计数 的映射
func foldContentCounts(items []*count.CountValueItem) map[int64]*count.ContentCountsItem {
	res := make(map[int64]*count.ContentCountsItem, len(items)/3+1)
	for _, item := range items {
		if item == nil || item.GetKey() == nil {
			continue
		}
		contentID := item.GetKey().GetTargetId()
		if contentID <= 0 {
			continue
		}
		entry, ok := res[contentID]
		if !ok {
			entry = &count.ContentCountsItem{ContentId: contentID}
			res[contentID] = entry
		}
		switch item.GetKey().GetBizType() {
		case count.BizType_BIZ_TYPE_LIKE:
			entry.LikeCount = item.GetValue()
		case count.BizType_BIZ_TYPE_FAVORITE:
			entry.FavoriteCount = item.GetValue()
		case count.BizType_BIZ_TYPE_COMMENT:
			entry.CommentCount = item.GetValue()
		}
	}
	return res
}
