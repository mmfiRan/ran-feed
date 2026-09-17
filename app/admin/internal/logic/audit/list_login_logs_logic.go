// Code scaffolded by goctl. Safe to edit.

package audit

import (
	"context"
	"time"

	"ran-feed/app/admin/internal/common/utils"
	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/admin/admin"
	pkgutils "ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ListLoginLogsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListLoginLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListLoginLogsLogic {
	return &ListLoginLogsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListLoginLogsLogic) ListLoginLogs(req *types.AdminLoginLogListReq) (resp *types.AdminLoginLogListRes, err error) {
	var startTime, endTime *timestamppb.Timestamp
	if st := req.StartTime; st != nil && *st > 0 {
		startTime = timestamppb.New(time.UnixMilli(*st))
	}
	if et := req.EndTime; et != nil && *et > 0 {
		endTime = timestamppb.New(time.UnixMilli(*et))
	}
	rpcRes, err := l.svcCtx.AdminAuditRpc.ListLoginLogs(l.ctx, &admin.ListLoginLogsReq{
		Username:  req.Username,
		Ip:        req.Ip,
		Status:    pkgutils.CastPtr[admin.LoginStatus](req.Status),
		StartTime: startTime,
		EndTime:   endTime,
		Page:      req.Page,
		PageSize:  req.PageSize,
	})
	if err != nil {
		return nil, err
	}

	items := make([]types.AdminLoginLogItem, 0, len(rpcRes.GetItems()))
	for _, it := range rpcRes.GetItems() {
		items = append(items, types.AdminLoginLogItem{
			Id:        it.GetId(),
			AdminId:   it.GetAdminId(),
			Username:  it.GetUsername(),
			Ip:        it.GetIp(),
			UserAgent: it.GetUserAgent(),
			Status:    utils.ToEnumValue(it.GetStatus()),
			Msg:       it.GetMsg(),
			CreatedAt: it.GetCreatedAt().AsTime().UnixMilli(),
		})
	}

	return &types.AdminLoginLogListRes{
		Items: items,
		PageQueryResp: types.PageQueryResp{
			Page:     rpcRes.GetPage(),
			PageSize: rpcRes.GetPageSize(),
			Total:    uint32(rpcRes.GetTotal()),
		},
	}, nil
}
