package es

import (
	"bytes"
	"context"
	"encoding/json"
	"sync/atomic"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esutil"
)

// IndexItem 一条待写文档 Version 取源行 updated_at 毫秒作 external version
type IndexItem struct {
	ID      string
	Version int64
	Doc     any
}

// DeleteRef 一条待删文档 Version 取事件时间毫秒 防旧删除覆盖新文档
type DeleteRef struct {
	ID      string
	Version int64
}

// BulkUpsert 批量 index 文档 external version 防乱序覆盖 返回失败条数 仅 BulkIndexer 致命错误才返回 err
func BulkUpsert(ctx context.Context, client *elasticsearch.Client, index string, items []IndexItem) (int, error) {
	if len(items) == 0 {
		return 0, nil
	}

	bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Client: client,
		Index:  index,
	})
	if err != nil {
		return 0, err
	}

	var failed int64
	for _, it := range items {
		body, err := json.Marshal(it.Doc)
		if err != nil {
			atomic.AddInt64(&failed, 1)
			continue
		}
		version := it.Version
		addErr := bi.Add(ctx, esutil.BulkIndexerItem{
			Action:      "index",
			DocumentID:  it.ID,
			Version:     &version,
			VersionType: "external",
			Body:        bytes.NewReader(body),
			OnFailure: func(_ context.Context, _ esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, _ error) {
				// external version 冲突说明已有更新文档 跳过不算失败
				if res.Status == 409 {
					return
				}
				atomic.AddInt64(&failed, 1)
			},
		})
		if addErr != nil {
			atomic.AddInt64(&failed, 1)
		}
	}

	if err := bi.Close(ctx); err != nil {
		return int(atomic.LoadInt64(&failed)), err
	}
	return int(atomic.LoadInt64(&failed)), nil
}

// BulkDelete 批量删文档 external version 防旧删覆盖新文档 404 不存在与 409 冲突均不算失败
func BulkDelete(ctx context.Context, client *elasticsearch.Client, index string, refs []DeleteRef) (int, error) {
	if len(refs) == 0 {
		return 0, nil
	}

	bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Client: client,
		Index:  index,
	})
	if err != nil {
		return 0, err
	}

	var failed int64
	for _, ref := range refs {
		version := ref.Version
		addErr := bi.Add(ctx, esutil.BulkIndexerItem{
			Action:      "delete",
			DocumentID:  ref.ID,
			Version:     &version,
			VersionType: "external",
			OnFailure: func(_ context.Context, _ esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, _ error) {
				if res.Status == 404 || res.Status == 409 {
					return
				}
				atomic.AddInt64(&failed, 1)
			},
		})
		if addErr != nil {
			atomic.AddInt64(&failed, 1)
		}
	}

	if err := bi.Close(ctx); err != nil {
		return int(atomic.LoadInt64(&failed)), err
	}
	return int(atomic.LoadInt64(&failed)), nil
}
