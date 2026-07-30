package notification

import (
	"strconv"
	"strings"

	"ran-feed/app/front/internal/types"
	contentpb "ran-feed/app/rpc/content/content"
	notifypb "ran-feed/app/rpc/notification/notification"
	userpb "ran-feed/app/rpc/user/user"
)

// encodeCursor 通知列表复合游标 编码为 "{updated_at_millis}:{id}" 字符串
// 前端当黑盒回传 svr 侧再解码 首页返回空串
func encodeCursor(updatedAtMillis, id int64) string {
	if updatedAtMillis <= 0 || id <= 0 {
		return ""
	}
	return strconv.FormatInt(updatedAtMillis, 10) + ":" + strconv.FormatInt(id, 10)
}

// decodeCursor 解 encodeCursor 出的串 空串或非法格式视为首页返 (0, 0)
// 非法输入不报错 直接首页 避免翻页因客户端错传死循环
func decodeCursor(cursor string) (int64, int64) {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return 0, 0
	}
	parts := strings.SplitN(cursor, ":", 2)
	if len(parts) != 2 {
		return 0, 0
	}
	ts, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, 0
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, 0
	}
	return ts, id
}

// parseIDs 输入 string ids 转 int64 过滤空/非法/<=0 保序去重
func parseIDs(raw []string) []int64 {
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(raw))
	out := make([]int64, 0, len(raw))
	for _, s := range raw {
		id, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// collectRefIDs 从原始行去重收集 actor_ids 与 content_ids 供批量富化
// 空 items 返 (nil, nil) actor/content <=0 跳过 关注类 content_id 为 0
func collectRefIDs(items []*notifypb.NotificationItem) (actorIDs, contentIDs []int64) {
	if len(items) == 0 {
		return nil, nil
	}
	seenActor := make(map[int64]struct{})
	seenContent := make(map[int64]struct{})
	for _, it := range items {
		if it == nil {
			continue
		}
		if it.ActorId > 0 {
			if _, ok := seenActor[it.ActorId]; !ok {
				seenActor[it.ActorId] = struct{}{}
				actorIDs = append(actorIDs, it.ActorId)
			}
		}
		if it.ContentId > 0 {
			if _, ok := seenContent[it.ContentId]; !ok {
				seenContent[it.ContentId] = struct{}{}
				contentIDs = append(contentIDs, it.ContentId)
			}
		}
	}
	return actorIDs, contentIDs
}

// nilSafeInt64 >0 转指针 nil 则返回 nil 用于 optional 字段映射
func nilSafeInt64(v int64) *int64 {
	if v > 0 {
		return &v
	}
	return nil
}

// nilSafeString 非空转指针 空串返回 nil 用于 optional 字段映射
func nilSafeString(v string) *string {
	if v != "" {
		return &v
	}
	return nil
}

// assembleNotificationItems 按 rpc 出的原始行顺序 富化组装 front 层 NotificationItem
// - actor 找不到照样返 空 nickname/avatar 但 user_id 仍带回
// - content 找不到(已删/下架/私密) Content 置 nil 但保留通知本身
// - 关注类(content_id=0) Content 恒 nil
func assembleNotificationItems(
	items []*notifypb.NotificationItem,
	userMap map[int64]*userpb.UserInfo,
	contentMap map[int64]*contentpb.ContentItem,
) []types.NotificationItem {
	out := make([]types.NotificationItem, 0, len(items))
	for _, it := range items {
		if it == nil {
			continue
		}
		actor := types.NotificationActor{UserId: it.ActorId}
		if u := userMap[it.ActorId]; u != nil {
			actor.Nickname = u.Nickname
			actor.Avatar = u.Avatar
		}
		var content *types.NotificationContent
		if it.ContentId > 0 {
			if c := contentMap[it.ContentId]; c != nil {
				content = &types.NotificationContent{
					ContentId: c.ContentId,
					Title:     c.Title,
					CoverUrl:  c.CoverUrl,
				}
			}
		}
		out = append(out, types.NotificationItem{
			Id:        it.Id,
			Type:      int32(it.NotifyType),
			Actor:     actor,
			AggCount:  it.AggCount,
			Content:   content,
			CommentId: nilSafeInt64(it.CommentId),
			Snippet:   nilSafeString(it.Snippet),
			IsRead:    it.IsRead,
			UpdatedAt: it.UpdatedAt,
		})
	}
	return out
}