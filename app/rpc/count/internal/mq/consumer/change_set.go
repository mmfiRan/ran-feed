package consumer

import (
	"ran-feed/app/rpc/count/count"
	"ran-feed/app/rpc/count/internal/mq/consumer/strategy"
)

// countKey 计数缓存失效用的类型化键 取代字符串拼接再解析
type countKey struct {
	bizType    count.BizType
	targetType count.TargetType
	targetID   int64
}

// changeSet 累积一条消息落库后衍生的副作用 待失效缓存与待标脏内容
type changeSet struct {
	counts   map[countKey]struct{} // 待失效的计数缓存
	users    map[int64]struct{}    // 待失效的用户主页计数缓存
	contents map[int64]struct{}    // 待登记热榜脏集合的内容
}

func newChangeSet() *changeSet {
	return &changeSet{
		counts:   make(map[countKey]struct{}),
		users:    make(map[int64]struct{}),
		contents: make(map[int64]struct{}),
	}
}

// record 登记一条已落库 Update 的全部衍生影响 ownerID 为 ResetToZero 解析后的归属
func (s *changeSet) record(u strategy.Update, ownerID int64) {
	if u.TargetID <= 0 {
		return
	}
	s.counts[countKey{u.BizType, u.TargetType, u.TargetID}] = struct{}{}

	switch u.TargetType {
	case count.TargetType_TARGET_TYPE_CONTENT:
		if ownerID > 0 {
			s.users[ownerID] = struct{}{}
		}
		s.contents[u.TargetID] = struct{}{}
	case count.TargetType_TARGET_TYPE_USER:
		s.users[u.TargetID] = struct{}{}
	}
}

func (s *changeSet) empty() bool {
	return len(s.counts) == 0 && len(s.users) == 0 && len(s.contents) == 0
}
