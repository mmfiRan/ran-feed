package hotfeed

import (
	"fmt"

	"ran-feed/pkg/consts"
	"ran-feed/pkg/hotrank"
)

const (
	// 主榜候选池大小 裁剪线 远大于对外快照 留候补垫扛删除侵蚀
	defaultMainN = 5000
	// 重建后取前 TopN 作为对外可查询快照
	defaultTopN = 2000
	// 默认半衰期小时
	defaultHalfLifeHour = 24
	// 落库与回查计数批大小
	defaultBatchSize = 500
	// 全量扫库分页大小
	defaultPageSize = 1000
	// 冻结桶 SSCAN 每批拉取数量 防单个大 key 阻塞
	defaultScanBatch = 1000
	// 快照默认 1 小时过期 避免历史快照无限累积
	defaultSnapshotTTL = 3600
	// 窗口推导兜底 半衰期非正等异常时退化用此固定窗口
	defaultWindowDays = 15
	// 窗口自动推导允许的互动量级差 decades 取 5 即容忍 10 的 5 次方倍互动差距已很宽松
	windowDecades = 5
	// 自动推导窗口上限 防半衰期被调超大时退化成全表扫描
	maxWindowDays = 90
)

// Options 单次跑批的榜单参数 零值字段由 Normalize 按内置默认补齐
type Options struct {
	Shards        int
	MainN         int
	TopN          int
	HalfLifeHours float64
	Weights       *hotrank.Weights
	BatchSize     int
	PageSize      int
	WindowDays    int
}

// Normalize 补默认值并校验榜单不变量 幂等 可重复调用
// 分片数必须与 count 侧写入分片一致 否则部分分片永远不被处理 宁可不跑也不静默跑一半
func (o Options) Normalize() (Options, error) {
	if o.Shards <= 0 {
		o.Shards = consts.HotDirtyShards
	}
	if o.Shards != consts.HotDirtyShards {
		return Options{}, fmt.Errorf("热榜分片数 %d 与约定 %d 不一致", o.Shards, consts.HotDirtyShards)
	}
	if o.TopN <= 0 {
		o.TopN = defaultTopN
	}
	if o.MainN <= 0 {
		o.MainN = defaultMainN
	}
	// 主榜候选池必须不小于对外快照 否则没有候补垫 退化成主榜等于快照
	if o.MainN < o.TopN {
		o.MainN = o.TopN
	}
	if o.BatchSize <= 0 {
		o.BatchSize = defaultBatchSize
	}
	if o.PageSize <= 0 {
		o.PageSize = defaultPageSize
	}
	if o.HalfLifeHours <= 0 {
		o.HalfLifeHours = defaultHalfLifeHour
	}
	// 窗口缺省时从半衰期推导 与竞争视界对齐 显式传 WindowDays 不覆盖
	if o.WindowDays <= 0 {
		o.WindowDays = deriveWindowDays(o.HalfLifeHours)
	}
	return o, nil
}
