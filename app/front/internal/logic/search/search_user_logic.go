// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package search

import (
	"context"
	"strings"
	"sync"

	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/search/search"
	"ran-feed/app/rpc/user/user"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

type SearchUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSearchUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchUserLogic {
	return &SearchUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SearchUserLogic) SearchUser(req *types.SearchUserReq) (resp *types.SearchUserRes, err error) {
	resp = &types.SearchUserRes{Items: []types.SearchUserItem{}}
	if req == nil || strings.TrimSpace(req.Keyword) == "" {
		return resp, nil
	}

	viewerID := utils.GetContextUserIdWithDefault(l.ctx)

	searchRes, err := l.svcCtx.SearchRpc.SearchUser(l.ctx, &search.SearchUserReq{
		Keyword: req.Keyword,
		Cursor:  req.Cursor,
		Size:    req.Size,
	})
	if err != nil {
		return nil, err
	}
	if searchRes == nil || len(searchRes.Hits) == 0 {
		return resp, nil
	}

	ids := make([]int64, 0, len(searchRes.Hits))
	for _, h := range searchRes.Hits {
		ids = append(ids, h.UserId)
	}

	usersRes, err := l.svcCtx.UserRpc.BatchGetUser(l.ctx, &user.BatchGetUserReq{UserIds: ids})
	if err != nil {
		return nil, err
	}
	userMap := make(map[int64]*user.UserInfo, len(usersRes.Users))
	for _, u := range usersRes.Users {
		if u != nil {
			userMap[u.UserId] = u
		}
	}

	followMap := l.loadFollowSummaries(ids, viewerID)

	// 按搜索排序拼装
	items := make([]types.SearchUserItem, 0, len(ids))
	for _, id := range ids {
		u := userMap[id]
		if u == nil {
			continue
		}
		item := types.SearchUserItem{
			UserId:   u.UserId,
			Nickname: u.Nickname,
			Avatar:   u.Avatar,
			Bio:      u.Bio,
		}
		if fs := followMap[id]; fs != nil {
			item.IsFollowed = fs.IsFollowing
			item.FollowerCount = fs.FollowerCount
		}
		items = append(items, item)
	}
	resp.Items = items
	resp.Total = searchRes.Total
	resp.NextCursor = searchRes.NextCursor

	recordSearchHistory(l.svcCtx, viewerID, req.Keyword)
	return resp, nil
}

// loadFollowSummaries 并行取每个用户的是否已关注与粉丝数 单个失败只记日志不阻断
func (l *SearchUserLogic) loadFollowSummaries(ids []int64, viewerID int64) map[int64]*interaction.GetFollowSummaryRes {
	res := make(map[int64]*interaction.GetFollowSummaryRes, len(ids))
	var viewerPtr *int64
	if viewerID > 0 {
		viewerPtr = &viewerID
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, id := range ids {
		id := id
		wg.Add(1)
		threading.GoSafe(func() {
			defer wg.Done()
			summary, err := l.svcCtx.FollowRpc.GetFollowSummary(l.ctx, &interaction.GetFollowSummaryReq{
				UserId:   id,
				ViewerId: viewerPtr,
			})
			if err != nil {
				logx.WithContext(l.ctx).Errorf("取关注摘要失败 userID=%d err=%v", id, err)
				return
			}
			mu.Lock()
			res[id] = summary
			mu.Unlock()
		})
	}
	wg.Wait()
	return res
}
