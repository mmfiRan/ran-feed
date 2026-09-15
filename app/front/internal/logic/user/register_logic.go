// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/types/known/timestamppb"
	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	"ran-feed/app/rpc/user/user"
)

type RegisterLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterLogic) Register(req *types.RegisterReq) (resp *types.RegisterRes, err error) {

	rpcResp, err := l.svcCtx.UserRpc.Register(l.ctx, &user.RegisterReq{
		Mobile:   *req.Mobile,
		Password: *req.Password,
		Nickname: *req.Nickname,
		Avatar:   *req.Avatar,
		Bio:      req.Bio,
		Email:    *req.Email,
		Gender:   user.Gender(*req.Gender),
		Birthday: timestamppb.New(time.Unix(*req.Birthday, 0)),
	})
	if err != nil {
		return nil, err
	}

	return &types.RegisterRes{
		UserId:    rpcResp.UserId,
		Token:     rpcResp.Token,
		ExpiredAt: rpcResp.ExpiredAt.AsTime().Unix(),
	}, nil
}
