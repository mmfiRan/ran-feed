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
)

type ListOperationLogsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	operationLogRepo repositories.OperationLogRepository
}

func NewListOperationLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListOperationLogsLogic {
	return &ListOperationLogsLogic{
		ctx:              ctx,
		svcCtx:           svcCtx,
		Logger:           logx.WithContext(ctx),
		operationLogRepo: repositories.NewOperationLogRepository(ctx, svcCtx.MysqlDb),
	}
}

// ListOperationLogs 按条件分页查操作审计日志 先统计总数为0直接返回
func (l *ListOperationLogsLogic) ListOperationLogs(in *admin.ListOperationLogsReq) (*admin.ListOperationLogsRes, error) {
	filter := types.OperationLogFilter{
		AdminID: in.GetAdminId(),
		Action:  in.GetAction(),
		Status:  int32(in.GetStatus()),
	}
	if st := in.GetStartTime(); st != nil {
		filter.StartMillis = st.AsTime().UnixMilli()
	}
	if et := in.GetEndTime(); et != nil {
		filter.EndMillis = et.AsTime().UnixMilli()
	}

	offset, limit := utils.NormalizePage(in.GetPage(), in.GetPageSize())
	rows, total, err := l.operationLogRepo.Page(filter, offset, limit)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询审计日志失败"))
	}
	res := &admin.ListOperationLogsRes{
		Total:    total,
		Page:     in.GetPage(),
		PageSize: in.GetPageSize(),
	}
	if total == 0 {
		return res, nil
	}

	items := make([]*admin.OperationLogItem, 0, len(rows))
	for _, row := range rows {
		if item := logichelper.BuildOperationLogItem(row); item != nil {
			items = append(items, item)
		}
	}
	res.Items = items
	return res, nil
}
