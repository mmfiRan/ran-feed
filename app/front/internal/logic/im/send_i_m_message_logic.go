// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package im

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"ran-feed/app/front/internal/common/consts"
	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	impb "ran-feed/app/rpc/im/im"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"
)

// SendIMMessageLogic 发送消息
type SendIMMessageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewSendIMMessageLogic 构造
func NewSendIMMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendIMMessageLogic {
	return &SendIMMessageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// SendIMMessage 调 im-rpc 发送并返回 seq
func (l *SendIMMessageLogic) SendIMMessage(req *types.IMSendMessageReq) (resp *types.IMSendMessageRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil || userID <= 0 {
		return nil, consts.ErrUserNotLogin
	}

	rpcResp, err := l.svcCtx.ImMessageRpc.SendMessage(l.ctx, &impb.SendMessageReq{
		UserId:         userID,
		ConversationId: req.ConversationId,
		ClientMsgId:    req.ClientMsgId,
		ContentType:    impb.MessageContentType(req.ContentType),
		Content:        req.Content,
	})
	if err != nil {
		return nil, err
	}
	if rpcResp == nil {
		return nil, errorx.NewMsg("发送消息失败")
	}

	out := &types.IMSendMessageRes{
		MessageId: rpcResp.MessageId,
		Seq:       rpcResp.Seq,
	}
	if ts := rpcResp.GetSendTime(); ts != nil {
		out.SendTime = ts.AsTime().UnixMilli()
	}
	return out, nil
}
