package adminuserservicelogic

import (
	"context"

	"ran-feed/app/rpc/user/internal/common/utils"
	"ran-feed/app/rpc/user/internal/entity/model"
	"ran-feed/app/rpc/user/internal/repositories"
	"ran-feed/app/rpc/user/internal/svc"
	"ran-feed/app/rpc/user/user"
	pkgutils "ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	offset, pageSize := pkgutils.NormalizePage(in.GetPage(), in.GetPageSize())
	rows, total, err := l.userRepo.AdminPageUsers(
		pkgutils.CastPtr[int32](in.Status),
		in.Username,
		in.Nickname,
		offset, pageSize,
	)
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
		Total:    total,
		Page:     in.GetPage(),
		PageSize: in.GetPageSize(),
	}, nil
}

func buildAdminUserItem(row *model.RanFeedUser) *user.AdminUserItem {
	return &user.AdminUserItem{
		UserId:    row.ID,
		Username:  row.Username,
		Nickname:  row.Nickname,
		Mobile:    pkgutils.Deref(row.Mobile),
		Avatar:    row.Avatar,
		Status:    utils.UserStatusValue(row.Status),
		CreatedAt: timestamppb.New(row.CreatedAt),
	}
}
