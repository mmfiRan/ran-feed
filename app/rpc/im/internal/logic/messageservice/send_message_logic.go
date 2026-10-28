package messageservicelogic

import (
	"context"
	"errors"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	"ran-feed/app/rpc/im/im"
	"ran-feed/app/rpc/im/internal/common/component/impush"
	"ran-feed/app/rpc/im/internal/common/component/imseq"
	"ran-feed/app/rpc/im/internal/common/consts"
	imEnum "ran-feed/app/rpc/im/internal/common/enums"
	"ran-feed/app/rpc/im/internal/entity/model"
	"ran-feed/app/rpc/im/internal/entity/query"
	"ran-feed/app/rpc/im/internal/repositories"
	"ran-feed/app/rpc/im/internal/svc"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/event/imevent"
	"ran-feed/pkg/snowflake"
)

// SendMessageLogic 发送消息
type SendMessageLogic struct {
	ctx context.Context
	logx.Logger
	conversationRepo repositories.ConversationRepository
	memberRepo       repositories.MemberRepository
	messageRepo      repositories.MessageRepository
	groupRepo        repositories.GroupRepository
	seq              *imseq.Allocator
	push             *impush.Notifier
}

// NewSendMessageLogic 构造
func NewSendMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendMessageLogic {
	return &SendMessageLogic{
		ctx:              ctx,
		Logger:           logx.WithContext(ctx),
		conversationRepo: svcCtx.ConversationRepository,
		memberRepo:       svcCtx.MemberRepository,
		messageRepo:      svcCtx.MessageRepository,
		groupRepo:        svcCtx.GroupRepository,
		seq:              svcCtx.Seq,
		push:             svcCtx.Push,
	}
}

// SendMessage 事务内定序并落库,提交后异步推送,client_msg_id 撞唯一键返回首次结果
func (l *SendMessageLogic) SendMessage(in *im.SendMessageReq) (*im.SendMessageRes, error) {
	contentType := imEnum.ContentTypeEnum(in.ContentType)
	if err := validateSendMessage(in, contentType); err != nil {
		return nil, err
	}

	conv, err := l.conversationRepo.GetByID(l.ctx, in.ConversationId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询会话失败"))
	}
	if conv == nil {
		return nil, consts.ErrSendNotMember
	}
	if err := ensureGroupWritable(l.ctx, l.groupRepo, conv); err != nil {
		return nil, err
	}
	member, err := l.memberRepo.Get(l.ctx, in.ConversationId, in.UserId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询会话成员失败"))
	}
	if member == nil {
		return nil, consts.ErrSendNotMember
	}

	messageID := snowflake.GenID()
	now := time.Now()
	var seq int64
	err = query.Q.Transaction(func(tx *query.Query) error {
		allocated, err := l.seq.Malloc(l.ctx, l.conversationRepo.WithTx(tx), in.ConversationId, messageID, in.UserId, now)
		if err != nil {
			return err
		}
		seq = allocated

		row := &model.RanFeedImMessage{
			ID:             messageID,
			ConversationID: in.ConversationId,
			Seq:            seq,
			ClientMsgID:    in.ClientMsgId,
			SenderID:       in.UserId,
			ContentType:    contentType.Int32(),
			Content:        in.Content,
			Status:         imEnum.MessageStatusNormal.Int32(),
			CreatedBy:      in.UserId,
			UpdatedBy:      in.UserId,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if err := l.messageRepo.WithTx(tx).Create(l.ctx, row); err != nil {
			return err
		}
		// 发送者自己的消息不计未读
		_, err = l.memberRepo.WithTx(tx).AdvanceReadSeq(l.ctx, in.ConversationId, in.UserId, seq)
		return err
	})
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return l.existingMessageResult(in.ConversationId, in.ClientMsgId, err)
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorx.NewMsg("会话不存在")
		}
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("发送消息失败"))
	}

	// 事务提交后推送消息帧,失败只记日志
	pushFrame(l.Logger, l.push, l.memberRepo, in.ConversationId, &imevent.Frame{
		Type:           imevent.TypeMessage,
		ConversationID: in.ConversationId,
		MessageID:      messageID,
		Seq:            seq,
		SenderID:       in.UserId,
		ContentType:    contentType.Int32(),
		Content:        in.Content,
		SendTime:       now.UnixMilli(),
	})
	return &im.SendMessageRes{
		MessageId: messageID,
		Seq:       seq,
		SendTime:  timestamppb.New(now),
	}, nil
}

// existingMessageResult 幂等命中回查首次落库结果
func (l *SendMessageLogic) existingMessageResult(conversationID int64, clientMsgID string, cause error) (*im.SendMessageRes, error) {
	existing, err := l.messageRepo.GetByClientMsgID(l.ctx, conversationID, clientMsgID)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("发送消息失败"))
	}
	if existing == nil {
		return nil, errorx.Wrap(l.ctx, cause, errorx.NewMsg("发送消息失败"))
	}
	return &im.SendMessageRes{
		MessageId: existing.ID,
		Seq:       existing.Seq,
		SendTime:  timestamppb.New(existing.CreatedAt),
	}, nil
}
