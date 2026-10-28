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

// InviteIMGroupMembersLogic 邀请成员入群
type InviteIMGroupMembersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewInviteIMGroupMembersLogic 构造
func NewInviteIMGroupMembersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *InviteIMGroupMembersLogic {
	return &InviteIMGroupMembersLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// InviteIMGroupMembers 调 im-rpc 邀请成员
func (l *InviteIMGroupMembersLogic) InviteIMGroupMembers(req *types.IMInviteGroupMembersReq) (resp *types.IMInviteGroupMembersRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil || userID <= 0 {
		return nil, consts.ErrUserNotLogin
	}
	memberIDs, err := parseIMUserIDs(req.MemberIds)
	if err != nil {
		return nil, err
	}

	if _, err := l.svcCtx.ImGroupRpc.InviteGroupMembers(l.ctx, &impb.InviteGroupMembersReq{
		UserId:    userID,
		GroupId:   req.GroupId,
		MemberIds: memberIDs,
	}); err != nil {
		return nil, err
	}
	return &types.IMInviteGroupMembersRes{}, nil
}
