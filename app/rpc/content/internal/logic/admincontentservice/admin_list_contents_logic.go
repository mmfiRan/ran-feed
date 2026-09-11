package admincontentservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/common/logichelper"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/app/rpc/count/count"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
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

	statusFilter := optionalStatus(in)
	typeFilter := optionalContentType(in)
	authorFilter := optionalAuthorID(in)

	offset, limit := utils.NormalizePage(in.GetPage(), in.GetPageSize())
	rows, total, err := l.contentRepo.AdminPageContents(statusFilter, typeFilter, authorFilter, offset, limit)
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

	titles, err := l.loadTitles(rows)
	if err != nil {
		return nil, err
	}

	countsByID, err := l.loadCounts(rows)
	if err != nil {
		return nil, err
	}

	items := make([]*content.AdminContentItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, buildAdminContentItem(row, titles[row.ID], countsByID[row.ID]))
	}
	res.Items = items
	return res, nil
}

// loadCounts 一次批量取本页内容的互动计数 由 count 服务提供
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

	titles := make(map[int64]string, len(rows))
	if len(articleIDs) > 0 {
		articles, err := l.articleRepo.BatchGetBriefByContentIDs(articleIDs)
		if err != nil {
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询内容列表失败"))
		}
		for id, a := range articles {
			titles[id] = a.Title
		}
	}
	if len(videoIDs) > 0 {
		videos, err := l.videoRepo.BatchGetBriefByContentIDs(videoIDs)
		if err != nil {
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询内容列表失败"))
		}
		for id, v := range videos {
			titles[id] = v.Title
		}
	}
	return titles, nil
}

func buildAdminContentItem(row *model.RanFeedContent, title string, counts *count.ContentCountsItem) *content.AdminContentItem {
	item := &content.AdminContentItem{
		ContentId:     row.ID,
		ContentType:   logichelper.ContentTypeValue(row.ContentType),
		Status:        logichelper.ContentStatusValue(row.Status),
		Visibility:    logichelper.VisibilityValue(row.Visibility),
		AuthorId:      row.UserID,
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

func optionalStatus(in *content.AdminListContentsReq) *int32 {
	if in.Status == nil {
		return nil
	}
	v := int32(in.GetStatus())
	return &v
}

func optionalContentType(in *content.AdminListContentsReq) *int32 {
	if in.ContentType == nil {
		return nil
	}
	v := int32(in.GetContentType())
	return &v
}

func optionalAuthorID(in *content.AdminListContentsReq) *int64 {
	if in.AuthorId == nil {
		return nil
	}
	v := in.GetAuthorId()
	return &v
}
