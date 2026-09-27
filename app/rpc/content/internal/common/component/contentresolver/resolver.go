package contentresolver

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/common/component/contentcache"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/interaction/client/likeservice"
	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/user/client/userservice"
	"ran-feed/app/rpc/user/user"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/mr"
)

// Resolver feed 读路径公共入口 走二级缓存取内容详情 再旁挂作者与点赞
type Resolver struct {
	cache       *contentcache.Cache
	contentRepo repositories.ContentRepository
	articleRepo repositories.ArticleRepository
	videoRepo   repositories.VideoRepository
	userRpc     userservice.UserService
	likesRpc    likeservice.LikeService
}

func New(
	cache *contentcache.Cache,
	contentRepo repositories.ContentRepository,
	articleRepo repositories.ArticleRepository,
	videoRepo repositories.VideoRepository,
	userRpc userservice.UserService,
	likesRpc likeservice.LikeService,
) *Resolver {
	return &Resolver{
		cache:       cache,
		contentRepo: contentRepo,
		articleRepo: articleRepo,
		videoRepo:   videoRepo,
		userRpc:     userRpc,
		likesRpc:    likesRpc,
	}
}

// Resolve 按 ids 顺序解析成 Entry 列表 返回顺序是入参顺序的子序 publicOnly 为真时只留 PUBLIC 已删和未发布的 id 直接剔除
func (r *Resolver) Resolve(ctx context.Context, ids []int64, viewerID int64, publicOnly bool) ([]*Entry, error) {
	details, err := r.resolveDetails(ctx, ids, publicOnly)
	if err != nil {
		return nil, err
	}
	if len(details) == 0 {
		return []*Entry{}, nil
	}

	userMap, likedMap, likeCountMap, err := r.loadAuthorsAndLikes(ctx, details, viewerID)
	if err != nil {
		return nil, err
	}

	entries := make([]*Entry, 0, len(details))
	for _, d := range details {
		e := &Entry{
			Detail:    d,
			IsLiked:   likedMap[d.ContentID],
			LikeCount: likeCountMap[d.ContentID],
		}
		if u, ok := userMap[d.AuthorID]; ok && u != nil {
			e.AuthorName = u.Nickname
			e.AuthorAvatar = u.Avatar
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// resolveDetails 走二级缓存按 ids 顺序拿详情
func (r *Resolver) resolveDetails(ctx context.Context, ids []int64, publicOnly bool) ([]*contentcache.Detail, error) {
	detailMap, err := r.cache.BatchGet(ctx, ids, func(missIDs []int64) (map[int64]*contentcache.Detail, error) {
		return r.loadDetails(ctx, missIDs)
	})
	if err != nil {
		return nil, err
	}

	details := make([]*contentcache.Detail, 0, len(ids))
	for _, id := range ids {
		d, ok := detailMap[id]
		if !ok || d == nil {
			continue
		}
		if publicOnly && d.Visibility != int32(content.Visibility_VISIBILITY_PUBLIC) {
			continue
		}
		details = append(details, d)
	}
	return details, nil
}

// loadDetails 缓存没命中时回源 content 主表和它的文章 视频子表
func (r *Resolver) loadDetails(ctx context.Context, missIDs []int64) (map[int64]*contentcache.Detail, error) {
	contentMap, err := r.contentRepo.BatchGetPublishedByIDs(ctx, missIDs)
	if err != nil {
		return nil, errorx.Wrap(ctx, err, errorx.NewMsg("查询内容失败"))
	}
	if len(contentMap) == 0 {
		return map[int64]*contentcache.Detail{}, nil
	}

	articleIDs := make([]int64, 0)
	videoIDs := make([]int64, 0)
	for _, row := range contentMap {
		switch content.ContentType(row.ContentType) {
		case content.ContentType_CONTENT_TYPE_ARTICLE:
			articleIDs = append(articleIDs, row.ID)
		case content.ContentType_CONTENT_TYPE_VIDEO:
			videoIDs = append(videoIDs, row.ID)
		}
	}

	articleMap, err := r.articleRepo.BatchGetBriefByContentIDs(ctx, articleIDs)
	if err != nil {
		return nil, errorx.Wrap(ctx, err, errorx.NewMsg("查询文章摘要失败"))
	}
	videoMap, err := r.videoRepo.BatchGetBriefByContentIDs(ctx, videoIDs)
	if err != nil {
		return nil, errorx.Wrap(ctx, err, errorx.NewMsg("查询视频摘要失败"))
	}

	res := make(map[int64]*contentcache.Detail, len(contentMap))
	for id, row := range contentMap {
		title := ""
		coverURL := ""
		switch content.ContentType(row.ContentType) {
		case content.ContentType_CONTENT_TYPE_ARTICLE:
			if a, ok := articleMap[row.ID]; ok && a != nil {
				title = a.Title
				coverURL = a.Cover
			}
		case content.ContentType_CONTENT_TYPE_VIDEO:
			if v, ok := videoMap[row.ID]; ok && v != nil {
				title = v.Title
				coverURL = v.CoverURL
			}
		}
		publishedAt := int64(0)
		if row.PublishedAt != nil {
			publishedAt = row.PublishedAt.Unix()
		}
		res[id] = &contentcache.Detail{
			ContentID:   row.ID,
			ContentType: row.ContentType,
			AuthorID:    row.UserID,
			Title:       title,
			CoverURL:    coverURL,
			PublishedAt: publishedAt,
			Visibility:  row.Visibility,
		}
	}
	return res, nil
}

// loadAuthorsAndLikes 并行查作者与点赞信息 点赞跟观察者相关 不进缓存
func (r *Resolver) loadAuthorsAndLikes(ctx context.Context, details []*contentcache.Detail, viewerID int64) (map[int64]*user.UserInfo, map[int64]bool, map[int64]int64, error) {
	authorIDs := make([]int64, 0, len(details))
	authorSeen := make(map[int64]struct{}, len(details))
	likeInfos := make([]*likeservice.LikeInfo, 0, len(details))

	for _, d := range details {
		if _, ok := authorSeen[d.AuthorID]; !ok {
			authorSeen[d.AuthorID] = struct{}{}
			authorIDs = append(authorIDs, d.AuthorID)
		}
		switch content.ContentType(d.ContentType) {
		case content.ContentType_CONTENT_TYPE_ARTICLE:
			likeInfos = append(likeInfos, &likeservice.LikeInfo{
				ContentId: d.ContentID,
				Scene:     interaction.Scene_SCENE_ARTICLE,
			})
		case content.ContentType_CONTENT_TYPE_VIDEO:
			likeInfos = append(likeInfos, &likeservice.LikeInfo{
				ContentId: d.ContentID,
				Scene:     interaction.Scene_SCENE_VIDEO,
			})
		}
	}

	var (
		userMap      map[int64]*user.UserInfo
		likedMap     map[int64]bool
		likeCountMap map[int64]int64
	)

	err := mr.Finish(
		func() error {
			userMap = map[int64]*user.UserInfo{}
			if len(authorIDs) == 0 {
				return nil
			}
			resp, err := r.userRpc.BatchGetUser(ctx, &userservice.BatchGetUserReq{
				UserIds: authorIDs,
			})
			if err != nil {
				return err
			}
			if resp == nil {
				return nil
			}
			for _, u := range resp.Users {
				if u == nil {
					continue
				}
				userMap[u.UserId] = u
			}
			return nil
		},
		func() error {
			likedMap = map[int64]bool{}
			likeCountMap = map[int64]int64{}
			if len(likeInfos) == 0 {
				return nil
			}
			resp, err := r.likesRpc.BatchQueryLikeInfo(ctx, &likeservice.BatchQueryLikeInfoReq{
				UserId:    viewerID,
				LikeInfos: likeInfos,
			})
			if err != nil {
				return err
			}
			if resp == nil {
				return nil
			}
			for _, info := range resp.LikeInfos {
				if info == nil {
					continue
				}
				likeCountMap[info.ContentId] = info.LikeCount
				if info.IsLiked {
					likedMap[info.ContentId] = true
				}
			}
			return nil
		},
	)
	if err != nil {
		return nil, nil, nil, err
	}
	return userMap, likedMap, likeCountMap, nil
}
