package messageservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"

	"ran-feed/app/rpc/im/internal/common/component/impush"
	"ran-feed/app/rpc/im/internal/common/consts"
	"ran-feed/app/rpc/im/internal/repositories"
	"ran-feed/pkg/event/imevent"
)

// pushFrame 事务外的实时下行,脱离请求 ctx 独立超时,失败只记日志
func pushFrame(logger logx.Logger, push *impush.Notifier, memberRepo repositories.MemberRepository, conversationID int64, frame *imevent.Frame) {
	threading.GoSafe(func() {
		ctx, cancel := context.WithTimeout(context.Background(), consts.PushTimeout)
		defer cancel()

		targets, err := activePushTargets(ctx, memberRepo, conversationID)
		if err != nil {
			logger.Errorf("im 推送查询接收方失败 conversationID=%d err=%v", conversationID, err)
			return
		}
		if err := push.Notify(ctx, targets, frame); err != nil {
			logger.Errorf("im 推送到 im:push 失败 conversationID=%d err=%v", conversationID, err)
		}
	})
}
