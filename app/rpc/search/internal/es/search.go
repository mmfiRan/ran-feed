package es

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
)

// Hit 一条命中 只含 _id 排序分与高亮 展示字段由 front 富化
type Hit struct {
	ID        string
	Score     float64
	Highlight map[string][]string
}

// FirstHighlight 取某字段首个高亮片段 无则空串
func (h Hit) FirstHighlight(field string) string {
	if frags := h.Highlight[field]; len(frags) > 0 {
		return frags[0]
	}
	return ""
}

// SearchResult 查询结果 Total 为总命中数 Hits 为当前页
type SearchResult struct {
	Total int64
	Hits  []Hit
}

type esSearchResponse struct {
	Hits struct {
		Total struct {
			Value int64 `json:"value"`
		} `json:"total"`
		Hits []struct {
			ID        string              `json:"_id"`
			Score     float64             `json:"_score"`
			Highlight map[string][]string `json:"highlight"`
		} `json:"hits"`
	} `json:"hits"`
}

// Search 执行查询 body 为 query DSL 解析命中返回 ranked IDs
func Search(ctx context.Context, client *elasticsearch.Client, index string, body any) (*SearchResult, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	res, err := client.Search(
		client.Search.WithContext(ctx),
		client.Search.WithIndex(index),
		client.Search.WithBody(bytes.NewReader(data)),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, fmt.Errorf("ES 查询失败 index=%s status=%s", index, res.Status())
	}

	var parsed esSearchResponse
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	result := &SearchResult{Total: parsed.Hits.Total.Value}
	for _, h := range parsed.Hits.Hits {
		result.Hits = append(result.Hits, Hit{
			ID:        h.ID,
			Score:     h.Score,
			Highlight: h.Highlight,
		})
	}
	return result, nil
}
