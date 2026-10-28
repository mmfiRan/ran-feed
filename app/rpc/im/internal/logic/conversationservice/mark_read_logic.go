package conversationservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/types/known/emptypb"

	"ran-feed/app/rpc/im/im"
	"ran-feed/app/rpc/im/internal/common/consts"
	"ran-feed/app/rpc/im/internal/repositories"
	"ran-feed/app/rpc/im/internal/svc"
	"ran-feed/pkg/errorx"
)

// MarkReadLogic 上报已读水位
type MarkReadLogic struct {
	ctx context.Context
	logx.Logger
	conversationRepo repositories.ConversationRepository
	memberRepo       repositories.MemberRepository
}

// NewMarkReadLogic 构造
func NewMarkReadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MarkReadLogic {
	return &MarkReadLogic{
		ctx:              ctx,
		Logger:           logx.WithContext(ctx),
		conversationRepo: svcCtx.ConversationRepository,
		memberRepo:       svcCtx.MemberRepository,
	}
}

// MarkRead 只增不减推进水位,低于当前水位的上报静默成功
func (l *MarkReadLogic) MarkRead(in *im.MarkReadReq) (*emptypb.Empty, error) {
	if in.UserId <= 0 || in.ConversationId <= 0 || in.ReadSeq < 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	member, err := l.memberRepo.Get(l.ctx, in.ConversationId, in.UserId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询会话成员失败"))
	}
	if member == nil {
		return nil, consts.ErrNotMember
	}
	conv, err := l.conversationRepo.GetByID(l.ctx, in.ConversationId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询会话失败"))
	}
	if conv == nil {
		return nil, consts.ErrNotMember
	}

	target := in.ReadSeq
	if target > conv.MaxSeq {
		target = conv.MaxSeq
	}
	if target <= member.ReadSeq {
		return &emptypb.Empty{}, nil
	}
	if _, err := l.memberRepo.AdvanceReadSeq(l.ctx, in.ConversationId, in.UserId, target); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("更新已读水位失败"))
	}
	return &emptypb.Empty{}, nil
}
