package content

import (
	"ran-feed/app/rpc/content/content"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// visibilityOf 可见性未传返回 UNSPECIFIED 由 RPC 层按新建或编辑决定默认
func visibilityOf(p *int32) content.Visibility {
	if p != nil {
		return content.Visibility(*p)
	}
	return content.Visibility_VISIBILITY_UNSPECIFIED
}

// tsUnix pb 时间转秒 nil 返回 0
func tsUnix(ts *timestamppb.Timestamp) int64 {
	if ts == nil {
		return 0
	}
	return ts.AsTime().Unix()
}
