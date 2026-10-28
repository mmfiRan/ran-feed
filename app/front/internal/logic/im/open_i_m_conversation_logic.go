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

// OpenIMConversationLogic 打开或创建单聊会话
type OpenIMConversationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewOpenIMConversationLogic 构造
func NewOpenIMConversationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OpenIMConversationLogic {
	return &OpenIMConversationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// OpenIMConversation 调 im-rpc 幂等开会话
func (l *OpenIMConversationLogic) OpenIMConversation(req *types.OpenIMConversationReq) (resp *types.OpenIMConversationRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil || userID <= 0 {
		return nil, consts.ErrUserNotLogin
	}

	rpcResp, err := l.svcCtx.ImConversationRpc.OpenConversation(l.ctx, &impb.OpenConversationReq{
		UserId:     userID,
		PeerUserId: req.PeerUserId,
	})
	if err != nil {
		return nil, err
	}
	if rpcResp == nil {
		return nil, errorx.NewMsg("打开会话失败")
	}
	return &types.OpenIMConversationRes{
		ConversationId: rpcResp.ConversationId,
		MaxSeq:         rpcResp.MaxSeq,
		ReadSeq:        rpcResp.ReadSeq,
		MinSeq:         rpcResp.MinSeq,
	}, nil
}
