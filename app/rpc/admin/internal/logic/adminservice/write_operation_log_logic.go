package adminservicelogic

import (
	"context"

	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/entity/model"
	"ran-feed/app/rpc/admin/internal/repositories"
	"ran-feed/app/rpc/admin/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type WriteOperationLogLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	operationLogRepo repositories.OperationLogRepository
}

func NewWriteOperationLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WriteOperationLogLogic {
	return &WriteOperationLogLogic{
		ctx:              ctx,
		svcCtx:           svcCtx,
		Logger:           logx.WithContext(ctx),
		operationLogRepo: repositories.NewOperationLogRepository(ctx, svcCtx.MysqlDb),
	}
}

// WriteOperationLog 落一条后台操作审计日志
func (l *WriteOperationLogLogic) WriteOperationLog(in *admin.WriteOperationLogReq) (*admin.WriteOperationLogRes, error) {
	if in == nil || in.GetAdminId() <= 0 || in.GetAction() == "" {
		return nil, errorx.NewMsg("参数错误")
	}

	row := &model.RanFeedOperationLog{
		AdminID:    in.GetAdminId(),
		Action:     in.GetAction(),
		TargetType: in.GetTargetType(),
		TargetID:   in.GetTargetId(),
		Result:     in.GetResult(),
		IP:         in.GetIp(),
		CreatedBy:  in.GetAdminId(),
		UpdatedBy:  in.GetAdminId(),
	}
	if payload := in.GetPayload(); payload != "" {
		row.Payload = &payload
	}

	logID, err := l.operationLogRepo.Create(row)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("写操作日志失败"))
	}

	return &admin.WriteOperationLogRes{
		LogId: logID,
	}, nil
}
