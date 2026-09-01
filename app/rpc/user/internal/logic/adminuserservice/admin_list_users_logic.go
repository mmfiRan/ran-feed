package adminuserservicelogic

import (
	"context"

	"ran-feed/app/rpc/user/internal/entity/model"
	"ran-feed/app/rpc/user/internal/repositories"
	"ran-feed/app/rpc/user/internal/svc"
	"ran-feed/app/rpc/user/user"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminListUsersLogic struct {
	ctx      context.Context
	svcCtx   *svc.ServiceContext
	userRepo repositories.UserRepository
	logx.Logger
}

func NewAdminListUsersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminListUsersLogic {
	return &AdminListUsersLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		Logger:   logx.WithContext(ctx),
		userRepo: repositories.NewUserRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *AdminListUsersLogic) AdminListUsers(in *user.AdminListUsersReq) (*user.AdminListUsersRes, error) {
	status := int32(0)
	if in.Status != nil {
		status = int32(*in.Status)
	}
	keyword := ""
	if in.Keyword != nil {
		keyword = *in.Keyword
	}

	offset, pageSize := utils.NormalizePage(in.GetPage(), in.GetPageSize())
	rows, total, err := l.userRepo.AdminPageUsers(status, keyword, offset, pageSize)
	if err != nil {
		return nil, err
	}
	if total == 0 {
		return &user.AdminListUsersRes{
			Items:    []*user.AdminUserItem{},
			Total:    0,
			Page:     in.GetPage(),
			PageSize: in.GetPageSize(),
		}, nil
	}

	items := make([]*user.AdminUserItem, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		items = append(items, buildAdminUserItem(row))
	}

	return &user.AdminListUsersRes{
		Items:    items,
		Total:    uint32(total),
		Page:     in.GetPage(),
		PageSize: in.GetPageSize(),
	}, nil
}

func buildAdminUserItem(row *model.RanFeedUser) *user.AdminUserItem {
	return &user.AdminUserItem{
		UserId:    row.ID,
		Username:  row.Username,
		Nickname:  row.Nickname,
		Mobile:    row.Mobile,
		Avatar:    row.Avatar,
		Status:    user.UserStatus(row.Status),
		CreatedAt: row.CreatedAt.UnixMilli(),
	}
}
