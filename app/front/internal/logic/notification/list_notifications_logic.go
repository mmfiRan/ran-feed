// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package notification

import (
	"context"
	"ran-feed/app/front/internal/common/consts"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mr"

	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	contentpb "ran-feed/app/rpc/content/content"
	notifypb "ran-feed/app/rpc/notification/notification"
	userpb "ran-feed/app/rpc/user/user"
	"ran-feed/pkg/utils"
)

type ListNotificationsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListNotificationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListNotificationsLogic {
	return &ListNotificationsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ListNotifications 通知列表
func (l *ListNotificationsLogic) ListNotifications(req *types.NotificationListReq) (resp *types.NotificationListRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil || userID <= 0 {
		return nil, consts.ErrUserNotLogin
	}

	cursorTS, cursorID := decodeCursor(req.Cursor)
	rpcResp, err := l.svcCtx.NotificationRpc.ListNotifications(l.ctx, &notifypb.ListNotificationsReq{
		RecipientId:     userID,
		CursorUpdatedAt: cursorTS,
		CursorId:        cursorID,
		PageSize:        req.PageSize,
		TypeFilter:      notifypb.NotifyType(req.Type),
	})
	if err != nil {
		return nil, err
	}

	actorIDs, contentIDs := collectRefIDs(rpcResp.Items)

	userMap := make(map[int64]*userpb.UserInfo)
	contentMap := make(map[int64]*contentpb.ContentItem)
	_ = mr.Finish(
		func() error {
			if len(actorIDs) == 0 {
				return nil
			}
			userResp, uErr := l.svcCtx.UserRpc.BatchGetUser(l.ctx, &userpb.BatchGetUserReq{UserIds: actorIDs})
			if uErr != nil {
				l.Errorf("获取通知列表填充用户信息失败[BatchGetUser]:actorIDs=%v err=%v", actorIDs, uErr)
				return nil
			}
			for _, u := range userResp.GetUsers() {
				if u != nil && u.UserId > 0 {
					userMap[u.UserId] = u
				}
			}
			return nil
		},
		func() error {
			if len(contentIDs) == 0 {
				return nil
			}
			contentResp, cErr := l.svcCtx.FeedRpc.BatchGetContentItems(l.ctx, &contentpb.BatchGetContentItemsReq{
				ContentIds: contentIDs,
				ViewerId:   userID,
			})
			if cErr != nil {
				l.Errorf("获取通知列表填充内容信息失败[BatchGetContentItems]：contentIDs=%v err=%v", contentIDs, cErr)
				return nil
			}
			for _, c := range contentResp.GetItems() {
				if c != nil && c.ContentId > 0 {
					contentMap[c.ContentId] = c
				}
			}
			return nil
		},
	)

	return &types.NotificationListRes{
		Items:      assembleNotificationItems(rpcResp.Items, userMap, contentMap),
		NextCursor: encodeCursor(rpcResp.NextCursorUpdatedAt, rpcResp.NextCursorId),
		HasMore:    rpcResp.HasMore,
	}, nil
}
