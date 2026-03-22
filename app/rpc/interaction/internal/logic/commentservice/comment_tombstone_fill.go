package commentservicelogic

import "ran-feed/app/rpc/interaction/interaction"

const commentStatusDeleted int32 = 20

// fillCommentTombstones 给缓存缺失且无法回填的评论补墓碑占位
func fillCommentTombstones(items []*interaction.CommentItem, missIDs []int64, defaultContentID int64) {
	if len(items) == 0 || len(missIDs) == 0 {
		return
	}
	missSet := make(map[int64]struct{}, len(missIDs))
	for _, id := range missIDs {
		if id <= 0 {
			continue
		}
		missSet[id] = struct{}{}
	}
	if len(missSet) == 0 {
		return
	}

	for _, c := range items {
		if c == nil || c.CommentId <= 0 {
			continue
		}
		if _, ok := missSet[c.CommentId]; !ok {
			continue
		}
		// 仅在无有效内容时补墓碑，避免覆盖正常数据
		if c.Comment == "" && c.UserId == 0 && c.Status == 0 {
			c.Comment = "该评论已删除"
			c.Status = commentStatusDeleted
		}
		if c.ContentId == 0 && defaultContentID > 0 {
			c.ContentId = defaultContentID
		}
	}
}
