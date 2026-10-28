package conversationservicelogic

import (
	"context"
	"errors"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"

	"ran-feed/app/rpc/im/im"
	"ran-feed/app/rpc/im/internal/common/consts"
	imEnum "ran-feed/app/rpc/im/internal/common/enums"
	"ran-feed/app/rpc/im/internal/entity/model"
	"ran-feed/app/rpc/im/internal/entity/query"
	"ran-feed/app/rpc/im/internal/repositories"
	"ran-feed/app/rpc/im/internal/svc"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/snowflake"
)

// OpenConversationLogic 打开或创建单聊会话
type OpenConversationLogic struct {
	ctx context.Context
	logx.Logger
	conversationRepo repositories.ConversationRepository
	memberRepo       repositories.MemberRepository
}

// NewOpenConversationLogic 构造
func NewOpenConversationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OpenConversationLogic {
	return &OpenConversationLogic{
		ctx:              ctx,
		Logger:           logx.WithContext(ctx),
		conversationRepo: svcCtx.ConversationRepository,
		memberRepo:       svcCtx.MemberRepository,
	}
}

// OpenConversation 单聊会话不存在则创建,依赖 uk_biz_key 保证幂等
func (l *OpenConversationLogic) OpenConversation(in *im.OpenConversationReq) (*im.OpenConversationRes, error) {
	if in.UserId <= 0 || in.PeerUserId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	if in.UserId == in.PeerUserId {
		return nil, errorx.NewMsg("不能与自己创建会话")
	}

	conv, member, err := l.openOrCreate(in.UserId, in.PeerUserId)
	if err != nil {
		return nil, err
	}
	return &im.OpenConversationRes{
		ConversationId: conv.ID,
		MaxSeq:         conv.MaxSeq,
		ReadSeq:        member.ReadSeq,
		MinSeq:         member.MinSeq,
	}, nil
}

// openOrCreate 先查后建,并发撞唯一键时回滚重读并补齐成员
func (l *OpenConversationLogic) openOrCreate(userID, peerID int64) (*model.RanFeedImConversation, *model.RanFeedImConversationMember, error) {
	bizKey := consts.SingleConversationBizKey(userID, peerID)
	now := time.Now()
	conv := &model.RanFeedImConversation{
		ID:        snowflake.GenID(),
		ConvType:  imEnum.ConvTypeSingle.Int32(),
		BizKey:    bizKey,
		OwnerID:   0,
		MaxSeq:    0,
		LastMsgID: 0,
		LastMsgAt: &now,
		CreatedBy: userID,
		UpdatedBy: userID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := query.Q.Transaction(func(tx *query.Query) error {
		convRepo := l.conversationRepo.WithTx(tx)
		existing, err := convRepo.GetByBizKey(l.ctx, bizKey)
		if err != nil {
			return err
		}
		if existing != nil {
			conv = existing
			return l.ensureMembers(l.ctx, l.memberRepo.WithTx(tx), existing.ID, userID, peerID, now)
		}
		if err := convRepo.Create(l.ctx, conv); err != nil {
			return err
		}
		return l.ensureMembers(l.ctx, l.memberRepo.WithTx(tx), conv.ID, userID, peerID, now)
	})
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		// 并发创建撞 uk_biz_key,事务已回滚,重读赢家并补齐成员
		conv, err = l.conversationRepo.GetByBizKey(l.ctx, bizKey)
		if err != nil {
			return nil, nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("创建会话失败"))
		}
		if conv == nil {
			return nil, nil, errorx.NewMsg("创建会话失败")
		}
		if err = l.ensureMembers(l.ctx, l.memberRepo, conv.ID, userID, peerID, now); err != nil {
			return nil, nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("创建会话失败"))
		}
	}
	if err != nil {
		return nil, nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("创建会话失败"))
	}

	member, err := l.memberRepo.Get(l.ctx, conv.ID, userID)
	if err != nil {
		return nil, nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询会话成员失败"))
	}
	if member == nil {
		return nil, nil, errorx.NewMsg("会话成员不存在")
	}
	return conv, member, nil
}

// ensureMembers 幂等写双方成员记录
func (l *OpenConversationLogic) ensureMembers(ctx context.Context, repo repositories.MemberRepository, conversationID, userID, peerID int64, now time.Time) error {
	rows := make([]*model.RanFeedImConversationMember, 0, 2)
	for _, uid := range []int64{userID, peerID} {
		rows = append(rows, &model.RanFeedImConversationMember{
			ID:             snowflake.GenID(),
			ConversationID: conversationID,
			UserID:         uid,
			ReadSeq:        0,
			MinSeq:         0,
			Muted:          imEnum.MutedNo.Int32(),
			CreatedBy:      userID,
			UpdatedBy:      userID,
			CreatedAt:      now,
			UpdatedAt:      now,
		})
	}
	return repo.Upsert(ctx, rows)
}
