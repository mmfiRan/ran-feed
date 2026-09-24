package es

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ran-feed/pkg/snowflake"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/zeromicro/go-zero/core/logx"
)

// EnsureIndices 幂等确保两个别名可用 别名不存在则建物理索引并把别名指过去
func EnsureIndices(ctx context.Context, client *elasticsearch.Client) error {
	indices := []struct {
		name    string
		mapping string
	}{
		{IndexContent, contentMapping},
		{IndexUser, userMapping},
	}
	for _, idx := range indices {
		if err := ensureAlias(ctx, client, idx.name, idx.mapping); err != nil {
			return err
		}
	}
	return nil
}

// MappingFor 取别名对应索引的 mapping
func MappingFor(alias string) string {
	if alias == IndexUser {
		return userMapping
	}
	return contentMapping
}

// RebuildIndexName 生成新物理索引名 别名加时间与雪花号 全局唯一
func RebuildIndexName(alias string) string {
	return fmt.Sprintf("%s-%d-%d", alias, time.Now().Unix(), snowflake.GenID())
}

// IndexExists 索引或别名是否存在
func IndexExists(ctx context.Context, client *elasticsearch.Client, name string) (bool, error) {
	res, err := client.Indices.Exists([]string{name}, client.Indices.Exists.WithContext(ctx))
	if err != nil {
		return false, err
	}
	defer res.Body.Close()
	if res.StatusCode == 404 {
		return false, nil
	}
	if res.IsError() {
		return false, fmt.Errorf("检查索引失败 name=%s status=%s", name, res.Status())
	}
	return true, nil
}

// AliasIndices 列出别名当前指向的物理索引 别名不存在返回空
func AliasIndices(ctx context.Context, client *elasticsearch.Client, alias string) ([]string, error) {
	res, err := client.Indices.GetAlias(
		client.Indices.GetAlias.WithContext(ctx),
		client.Indices.GetAlias.WithName(alias),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode == 404 {
		return nil, nil
	}
	if res.IsError() {
		return nil, fmt.Errorf("查询别名失败 alias=%s status=%s", alias, res.Status())
	}
	var parsed map[string]any
	if derr := json.NewDecoder(res.Body).Decode(&parsed); derr != nil {
		return nil, derr
	}
	names := make([]string, 0, len(parsed))
	for name := range parsed {
		names = append(names, name)
	}
	return names, nil
}

// CreatePhysicalIndex 建物理索引 已存在则跳过
func CreatePhysicalIndex(ctx context.Context, client *elasticsearch.Client, name, mapping string) error {
	exists, err := IndexExists(ctx, client, name)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	res, err := client.Indices.Create(name,
		client.Indices.Create.WithContext(ctx),
		client.Indices.Create.WithBody(strings.NewReader(mapping)),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("创建索引失败 index=%s status=%s", name, res.Status())
	}
	logx.WithContext(ctx).Infof("ES 物理索引已创建 index=%s", name)
	return nil
}

// SwitchAlias 原子把别名迁到新索引 只对本就挂着该别名的索引发 remove
func SwitchAlias(ctx context.Context, client *elasticsearch.Client, alias, newIndex string, olds []string) error {
	actions := make([]map[string]any, 0, len(olds)+1)
	for _, old := range olds {
		actions = append(actions, map[string]any{
			"remove": map[string]any{"index": old, "alias": alias},
		})
	}
	actions = append(actions, map[string]any{
		"add": map[string]any{"index": newIndex, "alias": alias},
	})
	body, err := json.Marshal(map[string]any{"actions": actions})
	if err != nil {
		return err
	}
	res, err := client.Indices.UpdateAliases(
		strings.NewReader(string(body)),
		client.Indices.UpdateAliases.WithContext(ctx),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("切换别名失败 alias=%s index=%s status=%s", alias, newIndex, res.Status())
	}
	return nil
}

// DeletePhysicalIndex 删物理索引 不存在视为成功
func DeletePhysicalIndex(ctx context.Context, client *elasticsearch.Client, name string) error {
	res, err := client.Indices.Delete([]string{name}, client.Indices.Delete.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() && res.StatusCode != 404 {
		return fmt.Errorf("删除索引失败 index=%s status=%s", name, res.Status())
	}
	return nil
}

// RebuildIndex 以新物理索引重建别名指向的索引 全程旧索引可读
// 成功则切别名并删旧索引 加载或切换失败则清理新索引 旧索引与别名保持不动
func RebuildIndex(ctx context.Context, client *elasticsearch.Client, alias string, load func(ctx context.Context, index string) (int, error)) (int, error) {
	physical := RebuildIndexName(alias)
	if err := CreatePhysicalIndex(ctx, client, physical, MappingFor(alias)); err != nil {
		return 0, err
	}

	total, err := load(ctx, physical)
	if err != nil {
		_ = DeletePhysicalIndex(context.WithoutCancel(ctx), client, physical)
		return 0, err
	}

	olds, err := AliasIndices(ctx, client, alias)
	if err != nil {
		_ = DeletePhysicalIndex(context.WithoutCancel(ctx), client, physical)
		return 0, err
	}
	if err := SwitchAlias(ctx, client, alias, physical, olds); err != nil {
		_ = DeletePhysicalIndex(context.WithoutCancel(ctx), client, physical)
		return 0, err
	}

	for _, old := range olds {
		if derr := DeletePhysicalIndex(context.WithoutCancel(ctx), client, old); derr != nil {
			logx.WithContext(ctx).Errorf("删除旧索引失败 index=%s err=%v", old, derr)
		}
	}
	logx.WithContext(ctx).Infof("ES 索引重建完成 alias=%s index=%s count=%d", alias, physical, total)
	return total, nil
}

// ensureAlias 别名不存在则建物理索引并指过去 已存在跳过
// 同名具体索引存在但未挂别名时只告警 需人工删除后由本服务重建别名
func ensureAlias(ctx context.Context, client *elasticsearch.Client, alias, mapping string) error {
	exists, err := IndexExists(ctx, client, alias)
	if err != nil {
		return err
	}
	if exists {
		olds, aerr := AliasIndices(ctx, client, alias)
		if aerr != nil {
			return aerr
		}
		if len(olds) == 0 {
			logx.WithContext(ctx).Errorf("ES 索引名已被具体索引用占用且未挂别名 需删除该索引后由本服务重建 index=%s", alias)
		}
		return nil
	}

	physical := RebuildIndexName(alias)
	if err := CreatePhysicalIndex(ctx, client, physical, mapping); err != nil {
		return err
	}
	return SwitchAlias(ctx, client, alias, physical, nil)
}
