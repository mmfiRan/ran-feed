// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package notification

import (
	"context"
	"sync"

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

// notificationEnrichConcurrency 审核通知逐条取详情的并发上限
const notificationEnrichConcurrency = 8

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

	actorIDs, _ := collectRefIDs(rpcResp.Items)
	reviewContentIDs, publicContentIDs := splitReviewContentIDs(rpcResp.Items)

	var (
		userMap   map[int64]*userpb.UserInfo
		publicMap map[int64]*contentpb.ContentItem
		reviewMap map[int64]*contentpb.ContentItem
	)
	err = mr.Finish(
		func() error {
			userMap = make(map[int64]*userpb.UserInfo)
			if len(actorIDs) == 0 {
				return nil
			}
			userResp, uErr := l.svcCtx.UserRpc.BatchGetUser(l.ctx, &userpb.BatchGetUserReq{UserIds: actorIDs})
			if uErr != nil {
				l.Errorf("获取通知列表填充用户信息失败[BatchGetUser]:actorIDs=%v err=%v", actorIDs, uErr)
				return uErr
			}
			for _, u := range userResp.GetUsers() {
				if u != nil && u.UserId > 0 {
					userMap[u.UserId] = u
				}
			}
			return nil
		},
		func() error {
			publicMap = make(map[int64]*contentpb.ContentItem)
			if len(publicContentIDs) == 0 {
				return nil
			}
			contentResp, cErr := l.svcCtx.FeedRpc.BatchGetContentItems(l.ctx, &contentpb.BatchGetContentItemsReq{
				ContentIds: publicContentIDs,
				ViewerId:   userID,
			})
			if cErr != nil {
				l.Errorf("获取通知列表填充内容信息失败[BatchGetContentItems]:contentIDs=%v err=%v", publicContentIDs, cErr)
				return cErr
			}
			for _, c := range contentResp.GetItems() {
				if c != nil && c.ContentId > 0 {
					publicMap[c.ContentId] = c
				}
			}
			return nil
		},
		func() error {
			reviewMap = l.fillReviewContents(reviewContentIDs, userID)
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	contentMap := make(map[int64]*contentpb.ContentItem, len(publicMap)+len(reviewMap))
	for id, c := range publicMap {
		contentMap[id] = c
	}
	for id, c := range reviewMap {
		contentMap[id] = c
	}

	return &types.NotificationListRes{
		Items:      assembleNotificationItems(rpcResp.Items, userMap, contentMap),
		NextCursor: encodeCursor(rpcResp.NextCursorUpdatedAt, rpcResp.NextCursorId),
		HasMore:    rpcResp.HasMore,
	}, nil
}

// fillReviewContents 审核通知按作者本人视角逐条取详情 只取标题封面
// 内容已被删除取不到属正常 记日志跳过不报错 避免整页通知加载失败
func (l *ListNotificationsLogic) fillReviewContents(contentIDs []int64, viewerID int64) map[int64]*contentpb.ContentItem {
	out := make(map[int64]*contentpb.ContentItem, len(contentIDs))
	if len(contentIDs) == 0 {
		return out
	}
	var mu sync.Mutex
	mr.ForEach(func(source chan<- int64) {
		for _, id := range contentIDs {
			source <- id
		}
	}, func(id int64) {
		resp, cErr := l.svcCtx.ContentRpc.GetContentDetail(l.ctx, &contentpb.GetContentDetailReq{
			ContentId: id,
			ViewerId:  &viewerID,
		})
		if cErr != nil {
			l.Errorf("获取审核通知内容失败[GetContentDetail]:contentID=%d err=%v", id, cErr)
			return
		}
		if resp == nil || resp.Detail == nil {
			return
		}
		mu.Lock()
		out[id] = &contentpb.ContentItem{
			ContentId: resp.Detail.ContentId,
			Title:     resp.Detail.Title,
			CoverUrl:  resp.Detail.CoverUrl,
		}
		mu.Unlock()
	}, mr.WithWorkers(notificationEnrichConcurrency))

	return out
}
