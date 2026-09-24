package contentservicelogic

import (
	"context"
	"strconv"

	"ran-feed/app/rpc/content/content"
	contentEnum "ran-feed/app/rpc/content/internal/common/enums"
	contentutils "ran-feed/app/rpc/content/internal/common/utils"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/app/rpc/count/count"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mr"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type MyContentListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepository repositories.ContentRepository
	articleRepository repositories.ArticleRepository
	videoRepository   repositories.VideoRepository
	reviewRepository  repositories.ContentReviewRepository
}

func NewMyContentListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MyContentListLogic {
	return &MyContentListLogic{
		ctx:               ctx,
		svcCtx:            svcCtx,
		Logger:            logx.WithContext(ctx),
		contentRepository: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		articleRepository: repositories.NewArticleRepository(ctx, svcCtx.MysqlDb),
		videoRepository:   repositories.NewVideoRepository(ctx, svcCtx.MysqlDb),
		reviewRepository:  repositories.NewContentReviewRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *MyContentListLogic) MyContentList(in *content.MyContentListReq) (*content.MyContentListRes, error) {
	pageSize := utils.ClampPageSize(in.PageSize)

	var cursorID int64
	if in.Cursor != "" {
		cursorID, _ = strconv.ParseInt(in.Cursor, 10, 64)
	}

	rows, err := l.contentRepository.MyContentPage(
		in.UserId,
		utils.CastPtr[int32](in.Status),
		utils.CastPtr[int32](in.ContentType),
		cursorID,
		pageSize+1,
	)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询我的内容失败"))
	}

	hasMore := len(rows) > pageSize
	if hasMore {
		rows = rows[:pageSize]
	}
	if len(rows) == 0 {
		return &content.MyContentListRes{
			Items: []*content.MyContentItem{},
		}, nil
	}

	items, err := l.assembleItems(rows)
	if err != nil {
		return nil, err
	}

	nextCursor := ""
	if hasMore {
		nextCursor = strconv.FormatInt(rows[len(rows)-1].ID, 10)
	}

	return &content.MyContentListRes{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (l *MyContentListLogic) assembleItems(rows []*model.RanFeedContent) ([]*content.MyContentItem, error) {
	articleIDs := make([]int64, 0, len(rows))
	videoIDs := make([]int64, 0, len(rows))
	rejectedIDs := make([]int64, 0)
	takenDownIDs := make([]int64, 0)
	allIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		allIDs = append(allIDs, row.ID)
		switch content.ContentType(row.ContentType) {
		case content.ContentType_CONTENT_TYPE_ARTICLE:
			articleIDs = append(articleIDs, row.ID)
		case content.ContentType_CONTENT_TYPE_VIDEO:
			videoIDs = append(videoIDs, row.ID)
		}
		switch content.ContentStatus(row.Status) {
		case content.ContentStatus_CONTENT_STATUS_REJECTED:
			rejectedIDs = append(rejectedIDs, row.ID)
		case content.ContentStatus_CONTENT_STATUS_TAKEN_DOWN:
			takenDownIDs = append(takenDownIDs, row.ID)
		}
	}

	// 标题封面 不可见原因 计数
	var (
		articleMap map[int64]*model.RanFeedArticle
		videoMap   map[int64]*model.RanFeedVideo
		reasonMap  map[int64]string
		countMap   map[int64]*count.ContentCountsItem
	)
	if err := mr.Finish(
		func() error {
			m, err := l.articleRepository.BatchGetBriefByContentIDs(articleIDs)
			if err != nil {
				return err
			}
			articleMap = m
			return nil
		},
		func() error {
			m, err := l.videoRepository.BatchGetBriefByContentIDs(videoIDs)
			if err != nil {
				return err
			}
			videoMap = m
			return nil
		},
		func() error {
			reasonMap = make(map[int64]string, len(rejectedIDs)+len(takenDownIDs))
			// 被拒取最新拒绝理由 下架取最新下架原因 二者按当前状态分别取 互不串
			if m, err := l.reviewRepository.LatestReasonByContentIDs(
				rejectedIDs, []contentEnum.ReviewDecisionEnum{contentEnum.ReviewDecisionReject}); err != nil {
				return err
			} else {
				for id, r := range m {
					reasonMap[id] = r
				}
			}
			if m, err := l.reviewRepository.LatestReasonByContentIDs(
				takenDownIDs, []contentEnum.ReviewDecisionEnum{contentEnum.ReviewDecisionTakenDown}); err != nil {
				return err
			} else {
				for id, r := range m {
					reasonMap[id] = r
				}
			}
			return nil
		},
		func() error {
			countMap = l.loadCounts(allIDs)
			return nil
		},
	); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询我的内容失败"))
	}

	items := make([]*content.MyContentItem, 0, len(rows))
	for _, row := range rows {
		item := &content.MyContentItem{
			ContentId:   row.ID,
			ContentType: contentutils.ContentTypeValue(row.ContentType),
			Status:      contentutils.ContentStatusValue(row.Status),
			Visibility:  contentutils.VisibilityValue(row.Visibility),
			CreatedAt:   timestamppb.New(row.CreatedAt),
		}
		if row.PublishedAt != nil {
			item.PublishedAt = timestamppb.New(*row.PublishedAt)
		}
		switch content.ContentType(row.ContentType) {
		case content.ContentType_CONTENT_TYPE_ARTICLE:
			if a := articleMap[row.ID]; a != nil {
				item.Title = a.Title
				item.CoverUrl = a.Cover
			}
		case content.ContentType_CONTENT_TYPE_VIDEO:
			if v := videoMap[row.ID]; v != nil {
				item.Title = v.Title
				item.CoverUrl = v.CoverURL
			}
		}
		switch content.ContentStatus(row.Status) {
		case content.ContentStatus_CONTENT_STATUS_REJECTED, content.ContentStatus_CONTENT_STATUS_TAKEN_DOWN:
			item.StatusReason = reasonMap[row.ID]
		}
		if c := countMap[row.ID]; c != nil {
			item.LikeCount = c.GetLikeCount()
			item.FavoriteCount = c.GetFavoriteCount()
			item.CommentCount = c.GetCommentCount()
		}
		items = append(items, item)
	}
	return items, nil
}

// loadCounts 计数是次要展示 拉取失败降级为零值不阻断
func (l *MyContentListLogic) loadCounts(contentIDs []int64) map[int64]*count.ContentCountsItem {
	res := make(map[int64]*count.ContentCountsItem, len(contentIDs))
	if len(contentIDs) == 0 {
		return res
	}
	resp, err := l.svcCtx.CountRpc.BatchGetContentCounts(l.ctx, &count.BatchGetContentCountsReq{
		ContentIds: contentIDs,
	})
	if err != nil {
		l.Errorf("我的内容拉取计数失败 降级零值 err=%v", err)
		return res
	}
	for _, item := range resp.GetItems() {
		if item == nil {
			continue
		}
		res[item.GetContentId()] = item
	}
	return res
}
