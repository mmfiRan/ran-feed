package messageservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"ran-feed/app/rpc/im/im"
	"ran-feed/app/rpc/im/internal/common/consts"
	"ran-feed/app/rpc/im/internal/common/convert"
	"ran-feed/app/rpc/im/internal/repositories"
	"ran-feed/app/rpc/im/internal/svc"
	"ran-feed/pkg/errorx"
)

// PullMessagesLogic 按序号区间拉取消息
type PullMessagesLogic struct {
	ctx context.Context
	logx.Logger
	memberRepo   repositories.MemberRepository
	messageRepo  repositories.MessageRepository
	maxPullRange int
}

// NewPullMessagesLogic 构造
func NewPullMessagesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PullMessagesLogic {
	return &PullMessagesLogic{
		ctx:          ctx,
		Logger:       logx.WithContext(ctx),
		memberRepo:   svcCtx.MemberRepository,
		messageRepo:  svcCtx.MessageRepository,
		maxPullRange: svcCtx.Config.Message.MaxPullRange,
	}
}

// PullMessages 拉取 (seqBegin, seqEnd] 区间,过滤成员可见起点
func (l *PullMessagesLogic) PullMessages(in *im.PullMessagesReq) (*im.PullMessagesRes, error) {
	if in.UserId <= 0 || in.ConversationId <= 0 || in.SeqBegin < 0 || in.SeqEnd <= in.SeqBegin {
		return nil, errorx.NewMsg("参数错误")
	}
	if in.SeqEnd-in.SeqBegin > int64(l.maxPullRange) {
		return nil, errorx.NewMsg("拉取区间过大")
	}

	member, err := l.memberRepo.Get(l.ctx, in.ConversationId, in.UserId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询会话成员失败"))
	}
	if member == nil {
		return nil, consts.ErrNotMember
	}

	begin := max(in.SeqBegin, member.MinSeq)
	if begin >= in.SeqEnd {
		return &im.PullMessagesRes{}, nil
	}
	rows, err := l.messageRepo.ListBySeqRange(l.ctx, in.ConversationId, begin, in.SeqEnd)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("拉取消息失败"))
	}
	return &im.PullMessagesRes{Items: convert.MessagesFromModels(rows)}, nil
}
