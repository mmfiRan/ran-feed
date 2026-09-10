package adminroleservicelogic

import (
	"context"
	"strings"

	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/repositories"
	"ran-feed/app/rpc/admin/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/types/known/emptypb"
)

type UpdateRoleLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	roleRepo repositories.AdminRoleRepository
}

func NewUpdateRoleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateRoleLogic {
	return &UpdateRoleLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		Logger:   logx.WithContext(ctx),
		roleRepo: repositories.NewAdminRoleRepository(ctx, svcCtx.MysqlDb),
	}
}

// UpdateRole 修改角色名与备注
func (l *UpdateRoleLogic) UpdateRole(in *admin.UpdateRoleReq) (*emptypb.Empty, error) {
	name := strings.TrimSpace(in.GetName())
	if in.GetId() <= 0 || name == "" {
		return nil, errorx.NewMsg("角色信息不能为空")
	}

	affected, err := l.roleRepo.UpdateProfile(in.GetId(), name, in.GetRemark(), in.GetOperatorId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("更新角色失败"))
	}
	if affected == 0 {
		return nil, errorx.NewMsg("角色不存在")
	}
	return &emptypb.Empty{}, nil
}
