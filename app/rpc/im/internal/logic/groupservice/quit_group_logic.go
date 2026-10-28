package groupservicelogic

import (
	"context"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/types/known/emptypb"

	"ran-feed/app/rpc/im/im"
	"ran-feed/app/rpc/im/internal/common/consts"
	"ran-feed/app/rpc/im/internal/entity/query"
	"ran-feed/app/rpc/im/internal/repositories"
	"ran-feed/app/rpc/im/internal/svc"
	"ran-feed/pkg/errorx"
)

// QuitGroupLogic 退出群聊
type QuitGroupLogic struct {
	ctx context.Context
	logx.Logger
	conversationRepo repositories.ConversationRepository
	memberRepo       repositories.MemberRepository
	groupRepo        repositories.GroupRepository
}

// NewQuitGroupLogic 构造
func NewQuitGroupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QuitGroupLogic {
	return &QuitGroupLogic{
		ctx:              ctx,
		Logger:           logx.WithContext(ctx),
		conversationRepo: svcCtx.ConversationRepository,
		memberRepo:       svcCtx.MemberRepository,
		groupRepo:        svcCtx.GroupRepository,
	}
}

// QuitGroup 群主退群即解散,普通成员退群置成员记录删除
func (l *QuitGroupLogic) QuitGroup(in *im.QuitGroupReq) (*emptypb.Empty, error) {
	if in.UserId <= 0 || in.GroupId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	err := query.Q.Transaction(func(tx *query.Query) error {
		groupRepo := l.groupRepo.WithTx(tx)
		group, err := groupRepo.GetByIDForUpdate(l.ctx, in.GroupId)
		if err != nil {
			return err
		}
		if group == nil {
			return errorx.NewMsg("群不存在")
		}

		conv, err := l.conversationRepo.WithTx(tx).GetByBizKey(l.ctx, consts.GroupConversationBizKey(in.GroupId))
		if err != nil {
			return err
		}
		if conv == nil {
			return errorx.NewMsg("群会话不存在")
		}

		memberRepo := l.memberRepo.WithTx(tx)
		member, err := memberRepo.Get(l.ctx, conv.ID, in.UserId)
		if err != nil {
			return err
		}
		if member == nil {
			return consts.ErrNotMember
		}

		if group.OwnerID == in.UserId {
			return groupRepo.Dissolve(l.ctx, in.GroupId, in.UserId)
		}

		if _, err := memberRepo.MarkLeft(l.ctx, conv.ID, in.UserId); err != nil {
			return err
		}
		active, err := memberRepo.ListByConversation(l.ctx, conv.ID)
		if err != nil {
			return err
		}
		return groupRepo.UpdateMemberCount(l.ctx, in.GroupId, int32(len(active)), in.UserId)
	})
	if err != nil {
		var bizErr *errorx.BizError
		if errors.As(err, &bizErr) {
			return nil, bizErr
		}
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("退出群聊失败"))
	}
	return &emptypb.Empty{}, nil
}
