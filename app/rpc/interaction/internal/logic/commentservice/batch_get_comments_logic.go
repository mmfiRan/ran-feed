package commentservicelogic

import (
	"context"
	"strconv"
	"time"

	"ran-feed/app/rpc/interaction/interaction"
	rediskey "ran-feed/app/rpc/interaction/internal/common/consts/redis"
	luautils "ran-feed/app/rpc/interaction/internal/common/utils/lua"
	"ran-feed/app/rpc/interaction/internal/repositories"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

type BatchGetCommentsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	commentRepo repositories.CommentRepository
}

func NewBatchGetCommentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchGetCommentsLogic {
	return &BatchGetCommentsLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		commentRepo: repositories.NewCommentRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *BatchGetCommentsLogic) BatchGetComments(in *interaction.BatchGetCommentsReq) (*interaction.BatchGetCommentsRes, error) {
	if in == nil {
		return nil, errorx.NewMsg("参数错误")
	}
	if len(in.CommentIds) == 0 {
		return &interaction.BatchGetCommentsRes{
			Comments: nil,
			MissIds:  nil,
		}, nil
	}
	// 限制单次批量数量，避免被滥用
	if len(in.CommentIds) > 200 {
		return nil, errorx.NewMsg("comment_ids过多")
	}

	// 去重+过滤非法ID，并保留原始顺序
	ordered := make([]int64, 0, len(in.CommentIds))
	seen := make(map[int64]struct{}, len(in.CommentIds))
	for _, id := range in.CommentIds {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ordered = append(ordered, id)
	}
	if len(ordered) == 0 {
		return &interaction.BatchGetCommentsRes{Comments: nil, MissIds: nil}, nil
	}

	keys := make([]string, 0, len(ordered))
	argv := make([]string, 0, len(ordered)+1)
	argv = append(argv, strconv.FormatInt(int64(rediskey.RedisCommentObjExpireSeconds), 10))
	for _, id := range ordered {
		keys = append(keys, rediskey.BuildCommentObjKey(strconv.FormatInt(id, 10)))
		argv = append(argv, strconv.FormatInt(id, 10))
	}
	args := make([]any, 0, len(argv))
	for _, v := range argv {
		args = append(args, v)
	}

	result, err := l.svcCtx.Redis.EvalCtx(l.ctx, luautils.BatchGetCommentObjsScript, keys, args...)
	if err != nil {
		l.Errorf("BatchGetComments redis lua失败: %v", err)
		return l.queryFromDBAndFill(ordered)
	}

	arr, ok := result.([]interface{})
	if !ok {
		return l.queryFromDBAndFill(ordered)
	}
	const chunkSize = 12
	if len(arr)%chunkSize != 0 {
		return l.queryFromDBAndFill(ordered)
	}

	cacheHit := make(map[int64]*interaction.CommentItem, len(ordered))
	miss := make([]int64, 0)
	for i := 0; i+chunkSize-1 < len(arr); i += chunkSize {
		id := parseInt64(arr[i])
		if id <= 0 {
			continue
		}
		contentID := parseInt64(arr[i+1])
		userID := parseInt64(arr[i+2])
		replyToUserID := parseInt64(arr[i+3])
		parentID := parseInt64(arr[i+4])
		rootID := parseInt64(arr[i+5])
		comment, _ := arr[i+6].(string)
		createdAt := parseInt64(arr[i+7])
		status := int32(parseInt64(arr[i+8]))
		userName, _ := arr[i+9].(string)
		userAvatar, _ := arr[i+10].(string)
		replyCount := parseInt64(arr[i+11])

		// 缓存对象缺失（全部关键字段为空）视为 miss
		if contentID == 0 && userID == 0 && comment == "" {
			miss = append(miss, id)
			continue
		}
		cacheHit[id] = &interaction.CommentItem{
			CommentId:     id,
			ContentId:     contentID,
			UserId:        userID,
			ReplyToUserId: replyToUserID,
			ParentId:      parentID,
			RootId:        rootID,
			Comment:       comment,
			CreatedAt:     createdAt,
			Status:        status,
			UserName:      userName,
			UserAvatar:    userAvatar,
			ReplyCount:    replyCount,
		}
	}

	// miss 再查DB
	dbMap := make(map[int64]*interaction.CommentItem)
	if len(miss) > 0 {
		rows, derr := l.commentRepo.ListByIDs(miss)
		if derr != nil {
			return nil, errorx.Wrap(l.ctx, derr, errorx.NewMsg("批量查询评论失败"))
		}
		for _, r := range rows {
			if r == nil {
				continue
			}
			isDeleted := r.IsDeleted == 1 || r.Status == commentStatusDeleted
			commentText := r.Comment
			status := r.Status
			userID := r.UserID
			if isDeleted {
				commentText = "该评论已删除"
				status = commentStatusDeleted
				userID = 0
			}
			dbMap[r.ID] = &interaction.CommentItem{
				CommentId:     r.ID,
				ContentId:     r.ContentID,
				UserId:        userID,
				ReplyToUserId: r.ReplyToUserID,
				ParentId:      r.ParentID,
				RootId:        r.RootID,
				Comment:       commentText,
				CreatedAt:     r.CreatedAt.Unix(),
				Status:        status,
			}
		}
	}

	if len(ordered) > 0 {
		rootIDs := make([]int64, 0)
		parentIDs := make([]int64, 0)
		for _, id := range ordered {
			var it *interaction.CommentItem
			if v, ok := cacheHit[id]; ok {
				it = v
			} else if v, ok := dbMap[id]; ok {
				it = v
			}
			if it == nil {
				continue
			}
			if it.ParentId == 0 {
				rootIDs = append(rootIDs, id)
			} else {
				parentIDs = append(parentIDs, id)
			}
		}

		rootCountMap := make(map[int64]int64)
		parentCountMap := make(map[int64]int64)
		if len(rootIDs) > 0 {
			var err error
			rootCountMap, err = l.commentRepo.BatchCountByRootIDs(rootIDs)
			if err != nil {
				return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询评论回复数失败"))
			}
		}
		if len(parentIDs) > 0 {
			var err error
			parentCountMap, err = l.commentRepo.BatchCountByParentIDs(parentIDs)
			if err != nil {
				return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询评论回复数失败"))
			}
		}

		for _, it := range cacheHit {
			if it == nil {
				continue
			}
			if it.ParentId == 0 {
				it.ReplyCount = rootCountMap[it.CommentId]
			} else {
				it.ReplyCount = parentCountMap[it.CommentId]
			}
		}
		for _, it := range dbMap {
			if it == nil {
				continue
			}
			if it.ParentId == 0 {
				it.ReplyCount = rootCountMap[it.CommentId]
			} else {
				it.ReplyCount = parentCountMap[it.CommentId]
			}
		}
	}
	// 异步回填缓存（仅obj，不更新idx），确保 reply_count 已更新
	if len(dbMap) > 0 {
		itemsToFill := make([]*interaction.CommentItem, 0, len(dbMap))
		for _, it := range dbMap {
			itemsToFill = append(itemsToFill, it)
		}
		fillCommentUsers(l.ctx, l.svcCtx, l.Logger, itemsToFill)
		threading.GoSafe(func() {
			l.fillObjCacheBestEffort(itemsToFill)
		})
	}

	respComments := make([]*interaction.CommentItem, 0, len(ordered))
	respMiss := make([]int64, 0)
	for _, id := range ordered {
		if it, ok := cacheHit[id]; ok {
			respComments = append(respComments, it)
			continue
		}
		if it, ok := dbMap[id]; ok {
			respComments = append(respComments, it)
			continue
		}
		respMiss = append(respMiss, id)
	}
	fillCommentUsers(l.ctx, l.svcCtx, l.Logger, respComments)
	return &interaction.BatchGetCommentsRes{Comments: respComments, MissIds: respMiss}, nil
}

func (l *BatchGetCommentsLogic) queryFromDBAndFill(ids []int64) (*interaction.BatchGetCommentsRes, error) {
	rows, err := l.commentRepo.ListByIDs(ids)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("批量查询评论失败"))
	}
	dbMap := make(map[int64]*interaction.CommentItem, len(rows))
	for _, r := range rows {
		if r == nil {
			continue
		}
		dbMap[r.ID] = &interaction.CommentItem{
			CommentId:     r.ID,
			ContentId:     r.ContentID,
			UserId:        r.UserID,
			ReplyToUserId: r.ReplyToUserID,
			ParentId:      r.ParentID,
			RootId:        r.RootID,
			Comment:       r.Comment,
			CreatedAt:     r.CreatedAt.Unix(),
			Status:        r.Status,
		}
	}
	if len(ids) > 0 {
		rootIDs := make([]int64, 0)
		parentIDs := make([]int64, 0)
		for _, it := range dbMap {
			if it == nil {
				continue
			}
			if it.ParentId == 0 {
				rootIDs = append(rootIDs, it.CommentId)
			} else {
				parentIDs = append(parentIDs, it.CommentId)
			}
		}

		rootCountMap := make(map[int64]int64)
		parentCountMap := make(map[int64]int64)
		if len(rootIDs) > 0 {
			var err error
			rootCountMap, err = l.commentRepo.BatchCountByRootIDs(rootIDs)
			if err != nil {
				return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询评论回复数失败"))
			}
		}
		if len(parentIDs) > 0 {
			var err error
			parentCountMap, err = l.commentRepo.BatchCountByParentIDs(parentIDs)
			if err != nil {
				return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询评论回复数失败"))
			}
		}

		for _, it := range dbMap {
			if it == nil {
				continue
			}
			if it.ParentId == 0 {
				it.ReplyCount = rootCountMap[it.CommentId]
			} else {
				it.ReplyCount = parentCountMap[it.CommentId]
			}
		}
	}

	comments := make([]*interaction.CommentItem, 0, len(ids))
	miss := make([]int64, 0)
	for _, id := range ids {
		if it, ok := dbMap[id]; ok {
			comments = append(comments, it)
		} else {
			miss = append(miss, id)
		}
	}

	itemsToFill := make([]*interaction.CommentItem, 0, len(dbMap))
	for _, it := range dbMap {
		itemsToFill = append(itemsToFill, it)
	}
	fillCommentUsers(l.ctx, l.svcCtx, l.Logger, itemsToFill)
	threading.GoSafe(func() {
		l.fillObjCacheBestEffort(itemsToFill)
	})

	fillCommentUsers(l.ctx, l.svcCtx, l.Logger, comments)
	return &interaction.BatchGetCommentsRes{Comments: comments, MissIds: miss}, nil
}

func (l *BatchGetCommentsLogic) fillObjCacheBestEffort(items []*interaction.CommentItem) {
	for _, c := range items {
		if c == nil || c.CommentId <= 0 {
			continue
		}
		objKey := rediskey.BuildCommentObjKey(strconv.FormatInt(c.CommentId, 10))
		createdAt := c.CreatedAt
		if createdAt <= 0 {
			createdAt = time.Now().Unix()
		}
		_, err := l.svcCtx.Redis.EvalCtx(
			context.Background(),
			luautils.UpdateCommentObjScript,
			[]string{objKey},
			strconv.FormatInt(int64(rediskey.RedisCommentObjExpireSeconds), 10),
			strconv.FormatInt(c.CommentId, 10),
			strconv.FormatInt(c.ContentId, 10),
			strconv.FormatInt(c.UserId, 10),
			strconv.FormatInt(c.ReplyToUserId, 10),
			strconv.FormatInt(c.ParentId, 10),
			strconv.FormatInt(c.RootId, 10),
			c.Comment,
			strconv.FormatInt(createdAt, 10),
			strconv.FormatInt(int64(c.Status), 10),
			c.UserName,
			c.UserAvatar,
			strconv.FormatInt(c.ReplyCount, 10),
		)
		if err != nil {
			l.Errorf("回填评论对象缓存失败: %v, comment_id=%d", err, c.CommentId)
		}
	}
}
