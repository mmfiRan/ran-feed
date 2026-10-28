package im

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"ran-feed/app/front/internal/common/consts"
	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	impb "ran-feed/app/rpc/im/im"
	"ran-feed/pkg/utils"
)

// RevokeIMMessageLogic 撤回消息
type RevokeIMMessageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewRevokeIMMessageLogic 构造
func NewRevokeIMMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevokeIMMessageLogic {
	return &RevokeIMMessageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// RevokeIMMessage 调 im-rpc 撤回
func (l *RevokeIMMessageLogic) RevokeIMMessage(req *types.IMRevokeMessageReq) (resp *types.IMRevokeMessageRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil || userID <= 0 {
		return nil, consts.ErrUserNotLogin
	}

	if _, err := l.svcCtx.ImMessageRpc.RevokeMessage(l.ctx, &impb.RevokeMessageReq{
		UserId:         userID,
		ConversationId: req.ConversationId,
		Seq:            req.Seq,
	}); err != nil {
		return nil, err
	}
	return &types.IMRevokeMessageRes{}, nil
}
