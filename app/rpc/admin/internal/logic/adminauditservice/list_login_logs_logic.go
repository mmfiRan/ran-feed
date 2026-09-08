package adminauditservicelogic

import (
	"context"

	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/common/logichelper"
	"ran-feed/app/rpc/admin/internal/repositories"
	"ran-feed/app/rpc/admin/internal/svc"
	"ran-feed/app/rpc/admin/internal/types"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ListLoginLogsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	loginLogRepo repositories.LoginLogRepository
}

func NewListLoginLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListLoginLogsLogic {
	return &ListLoginLogsLogic{
		ctx:          ctx,
		svcCtx:       svcCtx,
		Logger:       logx.WithContext(ctx),
		loginLogRepo: repositories.NewLoginLogRepository(ctx, svcCtx.MysqlDb),
	}
}

// ListLoginLogs 按条件分页查登录日志 先统计总数为0直接返回
func (l *ListLoginLogsLogic) ListLoginLogs(in *admin.ListLoginLogsReq) (*admin.ListLoginLogsRes, error) {
	filter := types.LoginLogFilter{
		Username: in.GetUsername(),
		IP:       in.GetIp(),
		Status:   int32(in.GetStatus()),
	}
	if st := in.GetStartTime(); st != nil {
		filter.StartMillis = st.AsTime().UnixMilli()
	}
	if et := in.GetEndTime(); et != nil {
		filter.EndMillis = et.AsTime().UnixMilli()
	}

	offset, limit := utils.NormalizePage(in.GetPage(), in.GetPageSize())
	rows, total, err := l.loginLogRepo.Page(filter, offset, limit)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询登录日志失败"))
	}
	res := &admin.ListLoginLogsRes{
		Total:    total,
		Page:     in.GetPage(),
		PageSize: in.GetPageSize(),
	}
	if total == 0 {
		return res, nil
	}

	items := make([]*admin.LoginLogItem, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		items = append(items, &admin.LoginLogItem{
			Id:        row.ID,
			AdminId:   row.AdminID,
			Username:  row.Username,
			Ip:        row.IP,
			UserAgent: row.UserAgent,
			Status:    logichelper.LoginStatusValue(row.Status),
			Msg:       row.Msg,
			CreatedAt: timestamppb.New(row.CreatedAt),
		})
	}
	res.Items = items
	return res, nil
}
