package es

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/elastic/go-elasticsearch/v8"
)

// suggestName suggest DSL 里补全块的固定名字 解析响应时按此取
const suggestName = "suggest"

type esSuggestResponse struct {
	Suggest map[string][]struct {
		Options []struct {
			Text string `json:"text"`
		} `json:"options"`
	} `json:"suggest"`
}

// Suggest 用 completion suggester 按前缀补全 field 为 completion 字段 返回去重后的建议词
// prefix 为空返回空 不查询 ES 报错上抛由上层降级
func Suggest(ctx context.Context, client *elasticsearch.Client, index, field, prefix string, size int) ([]string, error) {
	if prefix == "" {
		return nil, nil
	}

	body := map[string]any{
		"_source": false,
		"suggest": map[string]any{
			suggestName: map[string]any{
				"prefix": prefix,
				"completion": map[string]any{
					"field":           field,
					"size":            size,
					"skip_duplicates": true,
				},
			},
		},
	}
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
		return nil, fmt.Errorf("ES 补全失败 index=%s status=%s", index, res.Status())
	}

	return parseSuggestOptions(res.Body)
}

// parseSuggestOptions 解析 suggest 响应体 抽出补全块的所有 option 文本
func parseSuggestOptions(r io.Reader) ([]string, error) {
	var parsed esSuggestResponse
	if err := json.NewDecoder(r).Decode(&parsed); err != nil {
		return nil, err
	}

	var texts []string
	for _, block := range parsed.Suggest[suggestName] {
		for _, opt := range block.Options {
			texts = append(texts, opt.Text)
		}
	}
	return texts, nil
}
