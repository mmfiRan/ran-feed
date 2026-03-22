// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	"ran-feed/app/rpc/user/user"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.LoginReq) (resp *types.LoginRes, err error) {

	rpcResp, err := l.svcCtx.UserRpc.Login(l.ctx, &user.LoginReq{
		Mobile:   *req.Mobile,
		Password: *req.Password,
	})
	if err != nil {
		return nil, err
	}

	return &types.LoginRes{
		UserId:    rpcResp.UserId,
		Token:     rpcResp.Token,
		ExpiredAt: rpcResp.ExpiredAt,
		Nickname:  rpcResp.Nickname,
		Avatar:    rpcResp.Avatar,
	}, nil
}
