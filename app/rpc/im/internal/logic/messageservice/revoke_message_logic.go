package messageservicelogic

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/types/known/emptypb"

	"ran-feed/app/rpc/im/im"
	"ran-feed/app/rpc/im/internal/common/component/impush"
	"ran-feed/app/rpc/im/internal/common/consts"
	imEnum "ran-feed/app/rpc/im/internal/common/enums"
	"ran-feed/app/rpc/im/internal/repositories"
	"ran-feed/app/rpc/im/internal/svc"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/event/imevent"
)

// RevokeMessageLogic 撤回消息
type RevokeMessageLogic struct {
	ctx context.Context
	logx.Logger
	conversationRepo repositories.ConversationRepository
	memberRepo       repositories.MemberRepository
	messageRepo      repositories.MessageRepository
	groupRepo        repositories.GroupRepository
	push             *impush.Notifier
	revokeWindow     time.Duration
}

// NewRevokeMessageLogic 构造
func NewRevokeMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevokeMessageLogic {
	return &RevokeMessageLogic{
		ctx:              ctx,
		Logger:           logx.WithContext(ctx),
		conversationRepo: svcCtx.ConversationRepository,
		memberRepo:       svcCtx.MemberRepository,
		messageRepo:      svcCtx.MessageRepository,
		groupRepo:        svcCtx.GroupRepository,
		push:             svcCtx.Push,
		revokeWindow:     time.Duration(svcCtx.Config.Message.RevokeWindowSeconds) * time.Second,
	}
}

// RevokeMessage 仅发送者本人在窗口内可撤回,不删行不回收 seq
func (l *RevokeMessageLogic) RevokeMessage(in *im.RevokeMessageReq) (*emptypb.Empty, error) {
	if in.UserId <= 0 || in.ConversationId <= 0 || in.Seq <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	conv, err := l.conversationRepo.GetByID(l.ctx, in.ConversationId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询会话失败"))
	}
	if conv == nil {
		return nil, consts.ErrNotMember
	}
	if err := ensureGroupWritable(l.ctx, l.groupRepo, conv); err != nil {
		return nil, err
	}
	member, err := l.memberRepo.Get(l.ctx, in.ConversationId, in.UserId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询会话成员失败"))
	}
	if member == nil {
		return nil, consts.ErrNotMember
	}

	msg, err := l.messageRepo.GetBySeq(l.ctx, in.ConversationId, in.Seq)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询消息失败"))
	}
	if msg == nil {
		return nil, errorx.NewMsg("消息不存在")
	}
	if msg.SenderID != in.UserId {
		return nil, errorx.NewMsg("无权撤回该消息")
	}
	if imEnum.MessageStatusEnum(msg.Status) == imEnum.MessageStatusRevoked {
		return &emptypb.Empty{}, nil
	}
	if l.revokeWindow > 0 && time.Since(msg.CreatedAt) > l.revokeWindow {
		return nil, consts.ErrRevokeWindow
	}

	affected, err := l.messageRepo.Revoke(l.ctx, in.ConversationId, in.Seq, in.UserId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("撤回消息失败"))
	}
	if affected == 0 {
		latest, rErr := l.messageRepo.GetBySeq(l.ctx, in.ConversationId, in.Seq)
		if rErr != nil {
			return nil, errorx.Wrap(l.ctx, rErr, errorx.NewMsg("撤回消息失败"))
		}
		if latest == nil || imEnum.MessageStatusEnum(latest.Status) != imEnum.MessageStatusRevoked {
			return nil, errorx.NewMsg("撤回消息失败")
		}
	}

	// 事务外推送撤回帧,失败只记日志
	pushFrame(l.Logger, l.push, l.memberRepo, in.ConversationId, &imevent.Frame{
		Type:           imevent.TypeRevoke,
		ConversationID: in.ConversationId,
		Seq:            in.Seq,
	})
	return &emptypb.Empty{}, nil
}
