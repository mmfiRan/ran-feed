// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package audit

import (
	"context"

	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/admin/admin"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListOperationLogsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListOperationLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListOperationLogsLogic {
	return &ListOperationLogsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListOperationLogsLogic) ListOperationLogs(req *types.AdminOperationLogListReq) (resp *types.AdminOperationLogListRes, err error) {
	rpcRes, err := l.svcCtx.AdminRpc.ListOperationLogs(l.ctx, &admin.ListOperationLogsReq{
		AdminId:    req.AdminId,
		Action:     req.Action,
		TargetType: req.TargetType,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		Page:       req.Page,
		PageSize:   req.PageSize,
	})
	if err != nil {
		return nil, err
	}

	items := make([]types.AdminOperationLogItem, 0, len(rpcRes.GetItems()))
	for _, it := range rpcRes.GetItems() {
		items = append(items, types.AdminOperationLogItem{
			Id:         it.GetId(),
			AdminId:    it.GetAdminId(),
			Action:     it.GetAction(),
			TargetType: it.GetTargetType(),
			TargetId:   it.GetTargetId(),
			Result:     it.GetResult(),
			Ip:         it.GetIp(),
			CreatedAt:  it.GetCreatedAt(),
		})
	}

	return &types.AdminOperationLogListRes{
		Items: items,
		PageQueryResp: types.PageQueryResp{
			Page:     rpcRes.GetPage(),
			PageSize: rpcRes.GetPageSize(),
			Total:    rpcRes.GetTotal(),
		},
	}, nil
}
