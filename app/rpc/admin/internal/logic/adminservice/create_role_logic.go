package adminservicelogic

import (
	"context"
	"strings"

	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/entity/model"
	"ran-feed/app/rpc/admin/internal/repositories"
	"ran-feed/app/rpc/admin/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateRoleLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	roleRepo repositories.AdminRoleRepository
}

func NewCreateRoleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateRoleLogic {
	return &CreateRoleLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		Logger:   logx.WithContext(ctx),
		roleRepo: repositories.NewAdminRoleRepository(ctx, svcCtx.MysqlDb),
	}
}

// CreateRole 建角色 code 唯一 撞码则拒
func (l *CreateRoleLogic) CreateRole(in *admin.CreateRoleReq) (*admin.CreateRoleRes, error) {
	code := strings.TrimSpace(in.GetCode())
	name := strings.TrimSpace(in.GetName())
	if code == "" || name == "" {
		return nil, errorx.NewMsg("参数错误")
	}

	existing, err := l.roleRepo.GetByCode(code)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询角色失败"))
	}
	if existing != nil {
		return nil, errorx.NewMsg("角色码已存在")
	}

	id, err := l.roleRepo.Create(&model.RanFeedAdminRole{
		Code:      code,
		Name:      name,
		Remark:    in.GetRemark(),
		CreatedBy: in.GetOperatorId(),
		UpdatedBy: in.GetOperatorId(),
	})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("创建角色失败"))
	}
	return &admin.CreateRoleRes{Id: id}, nil
}
