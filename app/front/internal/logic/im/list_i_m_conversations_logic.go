// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package im

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"ran-feed/app/front/internal/common/consts"
	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	impb "ran-feed/app/rpc/im/im"
	userpb "ran-feed/app/rpc/user/user"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"
)

// ListIMConversationsLogic 会话列表
type ListIMConversationsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewListIMConversationsLogic 构造
func NewListIMConversationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListIMConversationsLogic {
	return &ListIMConversationsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ListIMConversations 拉会话列表并富化单聊对方资料
func (l *ListIMConversationsLogic) ListIMConversations(req *types.IMConversationListReq) (resp *types.IMConversationListRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil || userID <= 0 {
		return nil, consts.ErrUserNotLogin
	}

	cursorLastMsgAt, cursorID := decodeConversationCursor(req.Cursor)
	rpcResp, err := l.svcCtx.ImConversationRpc.ListConversations(l.ctx, &impb.ListConversationsReq{
		UserId:          userID,
		CursorLastMsgAt: cursorLastMsgAt,
		CursorId:        cursorID,
		PageSize:        req.PageSize,
	})
	if err != nil {
		return nil, err
	}
	if rpcResp == nil {
		return nil, errorx.NewMsg("查询会话列表失败")
	}

	peerIDs := collectPeerIDs(rpcResp.Items)
	userMap := make(map[int64]*userpb.UserInfo, len(peerIDs))
	if len(peerIDs) > 0 {
		userResp, uErr := l.svcCtx.UserRpc.BatchGetUser(l.ctx, &userpb.BatchGetUserReq{UserIds: peerIDs})
		if uErr != nil {
			l.Errorf("获取会话列表用户信息失败[BatchGetUser]:peerIDs=%v err=%v", peerIDs, uErr)
			return nil, uErr
		}
		for _, u := range userResp.GetUsers() {
			if u != nil && u.UserId > 0 {
				userMap[u.UserId] = u
			}
		}
	}

	items := make([]types.IMConversationItem, 0, len(rpcResp.Items))
	for _, it := range rpcResp.Items {
		if it == nil {
			continue
		}
		items = append(items, toIMConversationItem(it, userMap))
	}

	return &types.IMConversationListRes{
		List:    items,
		Cursor:  encodeConversationCursor(rpcResp.NextCursorLastMsgAt, rpcResp.NextCursorId),
		HasMore: rpcResp.HasMore,
	}, nil
}
