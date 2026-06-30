package es

import (
	"context"
	"fmt"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/zeromicro/go-zero/core/logx"
)

// EnsureIndices 幂等建索引 不存在则按 mapping 创建 已存在跳过
func EnsureIndices(ctx context.Context, client *elasticsearch.Client) error {
	indices := []struct {
		name    string
		mapping string
	}{
		{IndexContent, contentMapping},
		{IndexUser, userMapping},
	}
	for _, idx := range indices {
		if err := ensureIndex(ctx, client, idx.name, idx.mapping); err != nil {
			return err
		}
	}
	return nil
}

func ensureIndex(ctx context.Context, client *elasticsearch.Client, name, mapping string) error {
	existsRes, err := client.Indices.Exists([]string{name}, client.Indices.Exists.WithContext(ctx))
	if err != nil {
		return err
	}
	defer existsRes.Body.Close()
	if existsRes.StatusCode == 200 {
		return nil
	}

	createRes, err := client.Indices.Create(name,
		client.Indices.Create.WithContext(ctx),
		client.Indices.Create.WithBody(strings.NewReader(mapping)),
	)
	if err != nil {
		return err
	}
	defer createRes.Body.Close()
	if createRes.IsError() {
		return fmt.Errorf("创建索引失败 index=%s status=%s", name, createRes.Status())
	}
	logx.WithContext(ctx).Infof("ES 索引已创建 index=%s", name)
	return nil
}
