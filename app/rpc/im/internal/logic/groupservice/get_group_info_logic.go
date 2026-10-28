package groupservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/types/known/timestamppb"

	"ran-feed/app/rpc/im/im"
	"ran-feed/app/rpc/im/internal/common/consts"
	imEnum "ran-feed/app/rpc/im/internal/common/enums"
	"ran-feed/app/rpc/im/internal/repositories"
	"ran-feed/app/rpc/im/internal/svc"
	"ran-feed/pkg/enums"
	"ran-feed/pkg/errorx"
)

// GetGroupInfoLogic 群信息与成员列表
type GetGroupInfoLogic struct {
	ctx context.Context
	logx.Logger
	conversationRepo repositories.ConversationRepository
	memberRepo       repositories.MemberRepository
	groupRepo        repositories.GroupRepository
}

// NewGetGroupInfoLogic 构造
func NewGetGroupInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGroupInfoLogic {
	return &GetGroupInfoLogic{
		ctx:              ctx,
		Logger:           logx.WithContext(ctx),
		conversationRepo: svcCtx.ConversationRepository,
		memberRepo:       svcCtx.MemberRepository,
		groupRepo:        svcCtx.GroupRepository,
	}
}

// GetGroupInfo 仅群成员可查看,已解散群仍可查看
func (l *GetGroupInfoLogic) GetGroupInfo(in *im.GetGroupInfoReq) (*im.GetGroupInfoRes, error) {
	if in.UserId <= 0 || in.GroupId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	conv, err := l.conversationRepo.GetByBizKey(l.ctx, consts.GroupConversationBizKey(in.GroupId))
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询群会话失败"))
	}
	if conv == nil {
		return nil, errorx.NewMsg("群不存在")
	}
	member, err := l.memberRepo.Get(l.ctx, conv.ID, in.UserId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询群成员失败"))
	}
	if member == nil {
		return nil, consts.ErrNotMember
	}
	group, err := l.groupRepo.GetByID(l.ctx, in.GroupId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询群信息失败"))
	}
	if group == nil {
		return nil, errorx.NewMsg("群不存在")
	}

	members, err := l.memberRepo.ListByConversation(l.ctx, conv.ID)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询群成员失败"))
	}
	out := make([]*im.GroupMember, 0, len(members))
	for _, m := range members {
		out = append(out, &im.GroupMember{
			UserId:   m.UserID,
			JoinedAt: timestamppb.New(m.CreatedAt),
			Muted:    imEnum.MutedEnum(m.Muted) == imEnum.MutedYes,
		})
	}

	return &im.GetGroupInfoRes{
		GroupId:     group.ID,
		Name:        group.Name,
		Avatar:      group.Avatar,
		OwnerId:     group.OwnerID,
		MemberCount: group.MemberCount,
		Status:      enums.ToCommonPB(imEnum.GroupStatusEnum(group.Status)),
		Members:     out,
	}, nil
}
