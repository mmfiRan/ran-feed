package snowflake

import (
	"fmt"
	"sync"

	"github.com/sony/sonyflake/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

// machineID 默认 0 表示用本机 IP 推导
var machineID uint16

// SetMachineID 显式设置机器 ID 同机多进程各给不同值避免同毫秒碰撞
func SetMachineID(id uint16) {
	if id == 0 {
		panic("snowflake: 机器 ID 不能为 0")
	}
	machineID = id
}

// sf 懒初始化 且只初始化一次
var sf = sync.OnceValue(func() *sonyflake.Sonyflake {
	opts := sonyflake.Settings{}
	if machineID != 0 {
		opts.MachineID = func() (int, error) {
			return int(machineID), nil
		}
	}
	s, err := sonyflake.New(opts)
	if err != nil {
		panic(fmt.Sprintf("snowflake: 初始化失败 %v", err))
	}
	return s
})

// GenID 生成唯一ID 失败即 panic 快速失败
func GenID() int64 {
	id, err := sf().NextID()
	if err != nil {
		logx.Errorf("snowflake: 生成 ID 失败 %v", err)
	}
	return id
}
