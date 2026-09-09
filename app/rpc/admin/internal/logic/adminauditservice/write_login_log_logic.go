package adminauditservicelogic

import (
	"context"

	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/entity/model"
	"ran-feed/app/rpc/admin/internal/repositories"
	"ran-feed/app/rpc/admin/internal/svc"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/snowflake"

	"github.com/zeromicro/go-zero/core/logx"
)

type WriteLoginLogLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	loginLogRepo repositories.LoginLogRepository
}

func NewWriteLoginLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WriteLoginLogLogic {
	return &WriteLoginLogLogic{
		ctx:          ctx,
		svcCtx:       svcCtx,
		Logger:       logx.WithContext(ctx),
		loginLogRepo: repositories.NewLoginLogRepository(ctx, svcCtx.MysqlDb),
	}
}

// WriteLoginLog 写登录日志
func (l *WriteLoginLogLogic) WriteLoginLog(in *admin.WriteLoginLogReq) (*admin.WriteLoginLogRes, error) {

	row := &model.RanFeedLoginLog{
		ID:        snowflake.GenID(),
		AdminID:   in.GetAdminId(),
		Username:  in.GetUsername(),
		IP:        in.GetIp(),
		UserAgent: in.GetUserAgent(),
		Status:    int32(in.GetStatus()),
		Msg:       in.GetMsg(),
		CreatedBy: in.GetAdminId(),
		UpdatedBy: in.GetAdminId(),
	}

	logID, err := l.loginLogRepo.Create(row)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("写登录日志失败"))
	}

	return &admin.WriteLoginLogRes{
		LogId: logID,
	}, nil
}
