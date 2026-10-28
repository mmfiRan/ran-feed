package im

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"ran-feed/app/front/internal/common/consts"
	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	impb "ran-feed/app/rpc/im/im"
	"ran-feed/pkg/utils"
)

// QuitIMGroupLogic 退出群聊
type QuitIMGroupLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewQuitIMGroupLogic 构造
func NewQuitIMGroupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QuitIMGroupLogic {
	return &QuitIMGroupLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// QuitIMGroup 调 im-rpc 退群
func (l *QuitIMGroupLogic) QuitIMGroup(req *types.IMQuitGroupReq) (resp *types.IMQuitGroupRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil || userID <= 0 {
		return nil, consts.ErrUserNotLogin
	}

	if _, err := l.svcCtx.ImGroupRpc.QuitGroup(l.ctx, &impb.QuitGroupReq{
		UserId:  userID,
		GroupId: req.GroupId,
	}); err != nil {
		return nil, err
	}
	return &types.IMQuitGroupRes{}, nil
}
