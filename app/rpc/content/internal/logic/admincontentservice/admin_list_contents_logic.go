package admincontentservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/common/utils"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/app/rpc/count/count"
	"ran-feed/app/rpc/user/client/userservice"
	"ran-feed/pkg/errorx"
	pkgutils "ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mr"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AdminListContentsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo repositories.ContentRepository
	articleRepo repositories.ArticleRepository
	videoRepo   repositories.VideoRepository
}

func NewAdminListContentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminListContentsLogic {
	return &AdminListContentsLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		articleRepo: repositories.NewArticleRepository(ctx, svcCtx.MysqlDb),
		videoRepo:   repositories.NewVideoRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *AdminListContentsLogic) AdminListContents(in *content.AdminListContentsReq) (*content.AdminListContentsRes, error) {

	offset, limit := pkgutils.NormalizePage(in.GetPage(), in.GetPageSize())
	rows, total, err := l.contentRepo.AdminPageContents(
		pkgutils.CastPtr[int32](in.Status),
		pkgutils.CastPtr[int32](in.ContentType),
		in.AuthorId,
		offset, limit,
	)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询内容列表失败"))
	}
	res := &content.AdminListContentsRes{
		Total:    total,
		Page:     in.GetPage(),
		PageSize: in.GetPageSize(),
	}
	if total == 0 {
		return res, nil
	}

	var (
		titles     map[int64]string
		countsByID map[int64]*count.ContentCountsItem
		usernames  map[int64]string
	)
	err = mr.Finish(
		func() error {
			titleMap, err := l.loadTitles(rows)
			if err != nil {
				return err
			}
			titles = titleMap
			return nil
		},
		func() error {
			countMap, err := l.loadCounts(rows)
			if err != nil {
				return err
			}
			countsByID = countMap
			return nil
		},
		func() error {
			usernameMap, err := l.loadUsernames(rows)
			if err != nil {
				return err
			}
			usernames = usernameMap
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	items := make([]*content.AdminContentItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, l.buildAdminContentItem(row, titles[row.ID], usernames[row.ID], countsByID[row.ID]))
	}
	res.Items = items
	return res, nil
}

// loadCounts 批量获取内容的互动计数
func (l *AdminListContentsLogic) loadCounts(rows []*model.RanFeedContent) (map[int64]*count.ContentCountsItem, error) {
	contentIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		contentIDs = append(contentIDs, row.ID)
	}

	resp, err := l.svcCtx.CountRpc.BatchGetContentCounts(l.ctx, &count.BatchGetContentCountsReq{
		ContentIds: contentIDs,
	})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询内容列表失败"))
	}

	countsByID := make(map[int64]*count.ContentCountsItem, len(contentIDs))
	for _, item := range resp.GetItems() {
		if item == nil || item.GetContentId() <= 0 {
			continue
		}
		countsByID[item.GetContentId()] = item
	}
	return countsByID, nil
}

// loadUsernames 批量取本页作者用户名
func (l *AdminListContentsLogic) loadUsernames(rows []*model.RanFeedContent) (map[int64]string, error) {
	authorIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		authorIDs = append(authorIDs, row.UserID)
	}

	resp, err := l.svcCtx.UserRpc.BatchGetUser(l.ctx, &userservice.BatchGetUserReq{
		UserIds: authorIDs,
	})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询内容列表失败"))
	}

	usernames := make(map[int64]string, len(authorIDs))
	for _, u := range resp.GetUsers() {
		if u == nil || u.GetUserId() <= 0 {
			continue
		}
		usernames[u.GetUserId()] = u.GetUsername()
	}
	return usernames, nil
}

// loadTitles 按类型分组批量取文章/视频标题
func (l *AdminListContentsLogic) loadTitles(rows []*model.RanFeedContent) (map[int64]string, error) {
	articleIDs := make([]int64, 0, len(rows))
	videoIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		switch content.ContentType(row.ContentType) {
		case content.ContentType_CONTENT_TYPE_ARTICLE:
			articleIDs = append(articleIDs, row.ID)
		case content.ContentType_CONTENT_TYPE_VIDEO:
			videoIDs = append(videoIDs, row.ID)
		}
	}

	var (
		articles map[int64]*model.RanFeedArticle
		videos   map[int64]*model.RanFeedVideo
	)
	err := mr.Finish(
		func() error {
			if len(articleIDs) == 0 {
				return nil
			}
			articleMap, err := l.articleRepo.BatchGetBriefByContentIDs(articleIDs)
			if err != nil {
				return err
			}
			articles = articleMap
			return nil
		},
		func() error {
			if len(videoIDs) == 0 {
				return nil
			}
			videoMap, err := l.videoRepo.BatchGetBriefByContentIDs(videoIDs)
			if err != nil {
				return err
			}
			videos = videoMap
			return nil
		},
	)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询内容列表失败"))
	}

	titles := make(map[int64]string, len(rows))
	for id, a := range articles {
		titles[id] = a.Title
	}
	for id, v := range videos {
		titles[id] = v.Title
	}
	return titles, nil
}

func (l *AdminListContentsLogic) buildAdminContentItem(row *model.RanFeedContent, title, username string, counts *count.ContentCountsItem) *content.AdminContentItem {
	item := &content.AdminContentItem{
		ContentId:     row.ID,
		ContentType:   utils.ContentTypeValue(row.ContentType),
		Status:        utils.ContentStatusValue(row.Status),
		Visibility:    utils.VisibilityValue(row.Visibility),
		AuthorId:      row.UserID,
		Username:      username,
		Title:         title,
		LikeCount:     counts.GetLikeCount(),
		FavoriteCount: counts.GetFavoriteCount(),
		CommentCount:  counts.GetCommentCount(),
		CreatedAt:     timestamppb.New(row.CreatedAt),
	}
	if row.PublishedAt != nil {
		item.PublishedAt = timestamppb.New(*row.PublishedAt)
	}
	return item
}
