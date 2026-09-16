// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package user

import (
	"context"

	"ran-feed/app/admin/internal/common/utils"
	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/user/user"
	pkgutils "ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListUsersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListUsersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListUsersLogic {
	return &ListUsersLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListUsersLogic) ListUsers(req *types.CUserListReq) (resp *types.CUserListRes, err error) {
	in := &user.AdminListUsersReq{
		Page:     req.Page,
		PageSize: req.PageSize,
		Status:   pkgutils.CastPtr[user.UserStatus](req.Status),
		Username: req.Username,
		Nickname: req.Nickname,
	}

	rpcRes, err := l.svcCtx.UserAdminRpc.AdminListUsers(l.ctx, in)
	if err != nil {
		return nil, err
	}

	items := make([]types.CUserItem, 0, len(rpcRes.GetItems()))
	for _, it := range rpcRes.GetItems() {
		items = append(items, types.CUserItem{
			UserId:    it.GetUserId(),
			Username:  it.GetUsername(),
			Nickname:  it.GetNickname(),
			Mobile:    it.GetMobile(),
			Avatar:    it.GetAvatar(),
			Status:    utils.ToEnumValue(it.GetStatus()),
			CreatedAt: it.GetCreatedAt().AsTime().UnixMilli(),
		})
	}

	return &types.CUserListRes{
		Items: items,
		PageQueryResp: types.PageQueryResp{
			Page:     rpcRes.GetPage(),
			PageSize: rpcRes.GetPageSize(),
			Total:    uint32(rpcRes.GetTotal()),
		},
	}, nil
}
