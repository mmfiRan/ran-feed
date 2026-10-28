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

// PullIMMessagesLogic 按 seq 区间拉取消息
type PullIMMessagesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewPullIMMessagesLogic 构造
func NewPullIMMessagesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PullIMMessagesLogic {
	return &PullIMMessagesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// PullIMMessages 拉取 (seq_begin, seq_end] 区间
func (l *PullIMMessagesLogic) PullIMMessages(req *types.IMPullMessagesReq) (resp *types.IMPullMessagesRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil || userID <= 0 {
		return nil, consts.ErrUserNotLogin
	}
	var seqBegin int64
	if req.SeqBegin != nil {
		seqBegin = *req.SeqBegin
	}
	if req.SeqEnd <= seqBegin {
		return nil, errorx.NewMsg("参数错误")
	}

	rpcResp, err := l.svcCtx.ImMessageRpc.PullMessages(l.ctx, &impb.PullMessagesReq{
		UserId:         userID,
		ConversationId: req.ConversationId,
		SeqBegin:       seqBegin,
		SeqEnd:         req.SeqEnd,
	})
	if err != nil {
		return nil, err
	}
	if rpcResp == nil {
		return nil, errorx.NewMsg("拉取消息失败")
	}

	items := make([]types.IMMessagePreview, 0, len(rpcResp.Items))
	for _, msg := range rpcResp.Items {
		if msg == nil {
			continue
		}
		items = append(items, toIMMessagePreview(msg))
	}
	return &types.IMPullMessagesRes{List: items}, nil
}
