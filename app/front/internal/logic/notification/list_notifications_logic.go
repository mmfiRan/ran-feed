// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package notification

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	contentpb "ran-feed/app/rpc/content/content"
	notifypb "ran-feed/app/rpc/notification/notification"
	userpb "ran-feed/app/rpc/user/user"
	"ran-feed/pkg/errorx"
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

// ListNotifications 通知列表 读时富化(N8):notification-rpc 取原始行→UserRpc+FeedRpc 批量富化→front 层组装
func (l *ListNotificationsLogic) ListNotifications(req *types.NotificationListReq) (resp *types.NotificationListRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil || userID <= 0 {
		return nil, errorx.NewMsg("未登录")
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
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询通知列表失败"))
	}

	actorIDs, contentIDs := collectRefIDs(rpcResp.Items)

	// 并行富化 actor 与 content 单侧失败退化为空 map 不阻断整个列表
	userMap := make(map[int64]*userpb.UserInfo)
	contentMap := make(map[int64]*contentpb.ContentItem)
	if len(actorIDs) > 0 {
		userResp, uErr := l.svcCtx.UserRpc.BatchGetUser(l.ctx, &userpb.BatchGetUserReq{UserIds: actorIDs})
		if uErr != nil {
			logx.WithContext(l.ctx).Errorf("BatchGetUser 富化失败 actorIDs=%v err=%v", actorIDs, uErr)
		} else {
			for _, u := range userResp.GetUsers() {
				if u != nil && u.UserId > 0 {
					userMap[u.UserId] = u
				}
			}
		}
	}
	if len(contentIDs) > 0 {
		contentResp, cErr := l.svcCtx.FeedRpc.BatchGetContentItems(l.ctx, &contentpb.BatchGetContentItemsReq{
			ContentIds: contentIDs,
			ViewerId:   userID,
		})
		if cErr != nil {
			logx.WithContext(l.ctx).Errorf("BatchGetContentItems 富化失败 contentIDs=%v err=%v", contentIDs, cErr)
		} else {
			for _, c := range contentResp.GetItems() {
				if c != nil && c.ContentId > 0 {
					contentMap[c.ContentId] = c
				}
			}
		}
	}

	return &types.NotificationListRes{
		Items:      assembleNotificationItems(rpcResp.Items, userMap, contentMap),
		NextCursor: encodeCursor(rpcResp.NextCursorUpdatedAt, rpcResp.NextCursorId),
		HasMore:    rpcResp.HasMore,
	}, nil
}
