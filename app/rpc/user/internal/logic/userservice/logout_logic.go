package userservicelogic

import (
	"context"

	"ran-feed/app/rpc/user/internal/common/utils/session"
	"ran-feed/app/rpc/user/internal/svc"
	"ran-feed/app/rpc/user/user"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/types/known/emptypb"
)

type LogoutLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLogoutLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LogoutLogic {
	return &LogoutLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *LogoutLogic) Logout(in *user.LogoutReq) (*emptypb.Empty, error) {
	if in == nil {
		return nil, errorx.NewMsg("参数错误")
	}

	if err := session.RemoveSession(l.ctx, l.svcCtx.Redis, in.GetUserId(), in.GetToken()); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("退出登录失败"))
	}

	return &emptypb.Empty{}, nil
}
