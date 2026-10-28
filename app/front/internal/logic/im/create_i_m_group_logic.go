package im

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"ran-feed/app/front/internal/common/consts"
	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	impb "ran-feed/app/rpc/im/im"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"
)

// CreateIMGroupLogic 创建群
type CreateIMGroupLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewCreateIMGroupLogic 构造
func NewCreateIMGroupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateIMGroupLogic {
	return &CreateIMGroupLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// CreateIMGroup 调 im-rpc 建群
func (l *CreateIMGroupLogic) CreateIMGroup(req *types.IMCreateGroupReq) (resp *types.IMCreateGroupRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil || userID <= 0 {
		return nil, consts.ErrUserNotLogin
	}
	memberIDs, err := parseIMUserIDs(req.MemberIds)
	if err != nil {
		return nil, err
	}

	rpcResp, err := l.svcCtx.ImGroupRpc.CreateGroup(l.ctx, &impb.CreateGroupReq{
		UserId:    userID,
		Name:      req.Name,
		Avatar:    req.Avatar,
		MemberIds: memberIDs,
	})
	if err != nil {
		return nil, err
	}
	if rpcResp == nil {
		return nil, errorx.NewMsg("创建群失败")
	}
	return &types.IMCreateGroupRes{
		GroupId:        rpcResp.GroupId,
		ConversationId: rpcResp.ConversationId,
	}, nil
}
