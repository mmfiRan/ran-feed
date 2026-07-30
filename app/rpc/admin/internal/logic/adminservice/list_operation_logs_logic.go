package adminservicelogic

import (
	"context"

	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/repositories"
	"ran-feed/app/rpc/admin/internal/svc"
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
	filter := repositories.OperationLogFilter{
		AdminID:     in.GetAdminId(),
		Action:      in.GetAction(),
		TargetType:  in.GetTargetType(),
		StartMillis: in.GetStartTime(),
		EndMillis:   in.GetEndTime(),
	}

	total, err := l.operationLogRepo.Count(filter)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("统计审计日志失败"))
	}
	res := &admin.ListOperationLogsRes{Total: total}
	if total == 0 {
		return res, nil
	}

	offset, limit := utils.NormalizePage(int(in.GetPage()), int(in.GetPageSize()))
	rows, err := l.operationLogRepo.List(filter, offset, limit)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询审计日志失败"))
	}

	items := make([]*admin.OperationLogItem, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		items = append(items, &admin.OperationLogItem{
			Id:         row.ID,
			AdminId:    row.AdminID,
			Action:     row.Action,
			TargetType: row.TargetType,
			TargetId:   row.TargetID,
			Result:     row.Result,
			Ip:         row.IP,
			CreatedAt:  row.CreatedAt.UnixMilli(),
		})
	}
	res.Items = items
	return res, nil
}
