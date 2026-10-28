// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package im

import (
	"context"
	"encoding/json"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"ran-feed/app/front/internal/common/consts"
	"ran-feed/app/front/internal/common/sse"
	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	"ran-feed/pkg/event/imevent"
	"ran-feed/pkg/utils"
)

// StreamIMLogic IM 实时流
type StreamIMLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewStreamIMLogic 构造
func NewStreamIMLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StreamIMLogic {
	return &StreamIMLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// StreamIM 注册 conn 后阻塞转发帧直到断连
func (l *StreamIMLogic) StreamIM(req *types.IMStreamReq, client chan<- *types.IMStreamRes) error {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil || userID <= 0 {
		return consts.ErrUserNotLogin
	}

	conn := sse.NewConn(l.ctx.Done())
	l.svcCtx.IMConnManager.Add(userID, conn)
	defer l.svcCtx.IMConnManager.Remove(userID, conn)

	select {
	case client <- &types.IMStreamRes{Type: imevent.TypeConnected}:
	case <-l.ctx.Done():
		return nil
	}

	ticker := time.NewTicker(sse.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-l.ctx.Done():
			return nil
		case <-ticker.C:
			select {
			case client <- &types.IMStreamRes{Type: imevent.TypeHeartbeat}:
			case <-l.ctx.Done():
				return nil
			}
		case payload, ok := <-conn.Send:
			if !ok {
				return nil
			}
			var frame imevent.Frame
			if err := json.Unmarshal(payload, &frame); err != nil {
				l.Errorf("im sse 帧解析失败 userID=%d err=%v", userID, err)
				continue
			}
			select {
			case client <- toIMStreamRes(frame):
			case <-l.ctx.Done():
				return nil
			}
		}
	}
}

// toIMStreamRes 帧字段按非零值映射,未设置的字段保持 nil
func toIMStreamRes(frame imevent.Frame) *types.IMStreamRes {
	out := &types.IMStreamRes{Type: frame.Type}
	if frame.ConversationID > 0 {
		v := frame.ConversationID
		out.ConversationId = &v
	}
	if frame.MessageID > 0 {
		v := frame.MessageID
		out.MessageId = &v
	}
	if frame.Seq > 0 {
		v := frame.Seq
		out.Seq = &v
	}
	if frame.SenderID > 0 {
		v := frame.SenderID
		out.SenderId = &v
	}
	if frame.ContentType != 0 {
		v := frame.ContentType
		out.ContentType = &v
	}
	if frame.Content != "" {
		v := frame.Content
		out.Content = &v
	}
	if frame.SendTime > 0 {
		v := frame.SendTime
		out.SendTime = &v
	}
	return out
}
