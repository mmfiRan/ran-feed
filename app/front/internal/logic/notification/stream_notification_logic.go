// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package notification

import (
	"context"
	"encoding/json"
	"time"

	"ran-feed/app/front/internal/common/sse"
	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

// heartbeatInterval 心跳间隔 定期发帧探活并防代理层因空闲断连
const heartbeatInterval = 15 * time.Second

type StreamNotificationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewStreamNotificationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StreamNotificationLogic {
	return &StreamNotificationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// StreamNotification SSE 长连接主循环 注册 conn 后阻塞转发信号帧到 client 直到断连
func (l *StreamNotificationLogic) StreamNotification(req *types.NotifyStreamReq, client chan<- *types.NotifyStreamRes) error {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return err
	}

	// 挂 conn 到 ConnManager 请求结束自动摘除
	conn := sse.NewConn(l.ctx.Done())
	l.svcCtx.NotifyConnManager.Add(userID, conn)
	defer l.svcCtx.NotifyConnManager.Remove(userID, conn)

	// 首帧 connected 让前端确认通道就绪
	select {
	case client <- &types.NotifyStreamRes{Type: "connected"}:
	case <-l.ctx.Done():
		return nil
	}

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-l.ctx.Done():
			return nil
		case <-ticker.C:
			// 心跳保活
			select {
			case client <- &types.NotifyStreamRes{Type: "heartbeat"}:
			case <-l.ctx.Done():
				return nil
			}
		case payload, ok := <-conn.Send:
			// broadcaster 侧关闭 send 视为断连
			if !ok {
				return nil
			}
			var frame sse.SSEFrame
			if err := json.Unmarshal(payload, &frame); err != nil {
				l.Errorf("sse 帧解析失败 userID=%d err=%v", userID, err)
				continue
			}
			select {
			case client <- &types.NotifyStreamRes{Type: frame.Type, Unread: frame.Unread}:
			case <-l.ctx.Done():
				return nil
			}
		}
	}
}
