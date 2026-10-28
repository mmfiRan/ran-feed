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
	"ran-feed/pkg/utils"
)

// MarkIMReadLogic 上报已读水位
type MarkIMReadLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewMarkIMReadLogic 构造
func NewMarkIMReadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MarkIMReadLogic {
	return &MarkIMReadLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// MarkIMRead 调 im-rpc 推进已读水位
func (l *MarkIMReadLogic) MarkIMRead(req *types.IMMarkReadReq) (resp *types.IMMarkReadRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil || userID <= 0 {
		return nil, consts.ErrUserNotLogin
	}

	if _, err := l.svcCtx.ImConversationRpc.MarkRead(l.ctx, &impb.MarkReadReq{
		UserId:         userID,
		ConversationId: req.ConversationId,
		ReadSeq:        req.ReadSeq,
	}); err != nil {
		return nil, err
	}
	return &types.IMMarkReadRes{}, nil
}
