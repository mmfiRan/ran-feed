package snowflake

import (
	"sync"

	"github.com/sony/sonyflake/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

var (
	sf   *sonyflake.Sonyflake
	once sync.Once
)

// Init 初始化 Sonyflake
func Init() {
	once.Do(func() {
		var err error
		sf, err = sonyflake.New(sonyflake.Settings{})
		if err != nil || sf == nil {
			logx.Errorf("failed to initialize sonyflake: %v", err)
			panic("failed to initialize sonyflake")
		}
	})
}

// GenID 生成唯一ID
func GenID() int64 {
	if sf == nil {
		Init()
	}
	id, err := sf.NextID()
	if err != nil {
		logx.Errorf("failed to generate id: %v", err)
		return 0
	}
	return id
}
