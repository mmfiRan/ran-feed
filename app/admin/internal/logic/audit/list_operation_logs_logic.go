// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package audit

import (
	"context"
	"time"

	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/admin/admin"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	var startTime, endTime *timestamppb.Timestamp
	if req.StartTime > 0 {
		startTime = timestamppb.New(time.UnixMilli(req.StartTime))
	}
	if req.EndTime > 0 {
		endTime = timestamppb.New(time.UnixMilli(req.EndTime))
	}
	rpcRes, err := l.svcCtx.AdminAuditRpc.ListOperationLogs(l.ctx, &admin.ListOperationLogsReq{
		AdminId:    req.AdminId,
		Action:     req.Action,
		TargetType: req.TargetType,
		StartTime:  startTime,
		EndTime:    endTime,
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
			CreatedAt:  it.GetCreatedAt().AsTime().UnixMilli(),
		})
	}

	return &types.AdminOperationLogListRes{
		Items: items,
		PageQueryResp: types.PageQueryResp{
			Page:     rpcRes.GetPage(),
			PageSize: rpcRes.GetPageSize(),
			Total:    uint32(rpcRes.GetTotal()),
		},
	}, nil
}
