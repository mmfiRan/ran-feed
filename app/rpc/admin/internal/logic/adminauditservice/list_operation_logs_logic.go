package adminauditservicelogic

import (
	"context"

	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/common/utils"
	"ran-feed/app/rpc/admin/internal/repositories"
	"ran-feed/app/rpc/admin/internal/svc"
	"ran-feed/app/rpc/admin/internal/types"
	"ran-feed/pkg/errorx"
	pkgutils "ran-feed/pkg/utils"

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
		AdminID:  in.AdminId,
		Username: in.Username,
		Action:   in.Action,
		Status:   pkgutils.CastPtr[int32](in.Status),
	}
	if st := in.GetStartTime(); st != nil {
		ms := st.AsTime().UnixMilli()
		filter.StartMillis = &ms
	}
	if et := in.GetEndTime(); et != nil {
		ms := et.AsTime().UnixMilli()
		filter.EndMillis = &ms
	}

	offset, limit := pkgutils.NormalizePage(in.GetPage(), in.GetPageSize())
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
		if item := utils.BuildOperationLogItem(row); item != nil {
			items = append(items, item)
		}
	}
	res.Items = items
	return res, nil
}
