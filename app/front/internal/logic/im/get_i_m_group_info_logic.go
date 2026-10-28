package im

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"ran-feed/app/front/internal/common/consts"
	frontutils "ran-feed/app/front/internal/common/utils"
	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	impb "ran-feed/app/rpc/im/im"
	userpb "ran-feed/app/rpc/user/user"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"
)

// GetIMGroupInfoLogic 群信息
type GetIMGroupInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetIMGroupInfoLogic 构造
func NewGetIMGroupInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetIMGroupInfoLogic {
	return &GetIMGroupInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetIMGroupInfo 拉群信息并富化成员资料
func (l *GetIMGroupInfoLogic) GetIMGroupInfo(req *types.IMGetGroupInfoReq) (resp *types.IMGetGroupInfoRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil || userID <= 0 {
		return nil, consts.ErrUserNotLogin
	}

	rpcResp, err := l.svcCtx.ImGroupRpc.GetGroupInfo(l.ctx, &impb.GetGroupInfoReq{
		UserId:  userID,
		GroupId: req.GroupId,
	})
	if err != nil {
		return nil, err
	}
	if rpcResp == nil {
		return nil, errorx.NewMsg("查询群信息失败")
	}

	memberIDs := make([]int64, 0, len(rpcResp.Members))
	for _, m := range rpcResp.Members {
		if m != nil && m.UserId > 0 {
			memberIDs = append(memberIDs, m.UserId)
		}
	}
	userMap := make(map[int64]*userpb.UserInfo, len(memberIDs))
	if len(memberIDs) > 0 {
		userResp, uErr := l.svcCtx.UserRpc.BatchGetUser(l.ctx, &userpb.BatchGetUserReq{UserIds: memberIDs})
		if uErr != nil {
			l.Errorf("获取群成员用户信息失败[BatchGetUser]:memberIDs=%v err=%v", memberIDs, uErr)
			return nil, uErr
		}
		for _, u := range userResp.GetUsers() {
			if u != nil && u.UserId > 0 {
				userMap[u.UserId] = u
			}
		}
	}

	members := make([]types.IMGroupMember, 0, len(rpcResp.Members))
	for _, m := range rpcResp.Members {
		if m == nil {
			continue
		}
		item := types.IMGroupMember{UserId: m.UserId, Muted: m.Muted}
		if u := userMap[m.UserId]; u != nil {
			item.Nickname = u.Nickname
			item.Avatar = u.Avatar
		}
		if ts := m.GetJoinedAt(); ts != nil {
			item.JoinedAt = ts.AsTime().UnixMilli()
		}
		members = append(members, item)
	}

	return &types.IMGetGroupInfoRes{
		GroupId:     rpcResp.GroupId,
		Name:        rpcResp.Name,
		Avatar:      rpcResp.Avatar,
		OwnerId:     rpcResp.OwnerId,
		MemberCount: rpcResp.MemberCount,
		Status:      frontutils.ToEnumValue(rpcResp.Status),
		Members:     members,
	}, nil
}
