package hotupdate

import (
	"encoding/json"

	"ran-feed/app/rpc/content/internal/common/component/hotfeed"
)

// Params XXL-JOB 参数
// 榜单参数直接内嵌 hotfeed.Options 缺省与校验都归组件 锁 TTL 属任务级 留在本层
type Params struct {
	hotfeed.Options
	LockTTL int `json:"lockTtl"`
}

// parseParams 解析 XXL-JOB 参数 缺省与不变量校验交 hotfeed 归一化
// 分片数与写入方不一致时直接报错 不静默跑一半
func parseParams(raw string) (Params, error) {
	p := Params{}
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &p)
	}
	if p.LockTTL <= 0 {
		p.LockTTL = defaultFullLockTTL
	}
	opts, err := p.Options.Normalize()
	if err != nil {
		return Params{}, err
	}
	p.Options = opts
	return p, nil
}
