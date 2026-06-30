package indexer

import (
	"strconv"
	"time"

	"ran-feed/app/rpc/search/internal/common/consts"
	"ran-feed/app/rpc/search/internal/entity/model"
	"ran-feed/app/rpc/search/internal/es"
	"ran-feed/app/rpc/search/internal/repositories"
)

// ContentDoc 内容索引文档 id 类字段按 keyword 存字符串 published_at 毫秒
type ContentDoc struct {
	ContentID   string  `json:"content_id"`
	ContentType int32   `json:"content_type"`
	Status      int32   `json:"status"`
	Visibility  int32   `json:"visibility"`
	AuthorID    string  `json:"author_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Body        string  `json:"body"`
	PublishedAt int64   `json:"published_at"`
	HotScore    float64 `json:"hot_score"`
	IsDeleted   int32   `json:"is_deleted"`
}

// UserDoc 用户索引文档
type UserDoc struct {
	UserID    string `json:"user_id"`
	Nickname  string `json:"nickname"`
	Bio       string `json:"bio"`
	Username  string `json:"username"`
	Status    int32  `json:"status"`
	IsDeleted int32  `json:"is_deleted"`
}

// Assembler 回源组装内容文档 正文散在 article/video 子表 故按 content_id 回读拼整篇
type Assembler struct {
	articleRepo repositories.ArticleRepository
	videoRepo   repositories.VideoRepository
}

func NewAssembler(articleRepo repositories.ArticleRepository, videoRepo repositories.VideoRepository) *Assembler {
	return &Assembler{
		articleRepo: articleRepo,
		videoRepo:   videoRepo,
	}
}

// AssembleContentDocs 由内容主表行回源 article/video 组装文档 文章取标题简介正文 视频只有标题
func (a *Assembler) AssembleContentDocs(contents []*model.RanFeedContent) ([]es.IndexItem, error) {
	if len(contents) == 0 {
		return nil, nil
	}

	ids := make([]int64, 0, len(contents))
	for _, c := range contents {
		if c != nil {
			ids = append(ids, c.ID)
		}
	}

	articles, err := a.articleRepo.GetByContentIDs(ids)
	if err != nil {
		return nil, err
	}
	videos, err := a.videoRepo.GetByContentIDs(ids)
	if err != nil {
		return nil, err
	}

	items := make([]es.IndexItem, 0, len(contents))
	for _, c := range contents {
		if c == nil {
			continue
		}
		doc := ContentDoc{
			ContentID:   strconv.FormatInt(c.ID, 10),
			ContentType: c.ContentType,
			Status:      c.Status,
			Visibility:  c.Visibility,
			AuthorID:    strconv.FormatInt(c.UserID, 10),
			PublishedAt: millis(c.PublishedAt),
			HotScore:    c.HotScore,
			IsDeleted:   c.IsDeleted,
		}
		switch c.ContentType {
		case consts.ContentTypeArticle:
			if art := articles[c.ID]; art != nil {
				doc.Title = art.Title
				if art.Description != nil {
					doc.Description = *art.Description
				}
				doc.Body = art.Content
			}
		case consts.ContentTypeVideo:
			if vid := videos[c.ID]; vid != nil {
				doc.Title = vid.Title
			}
		}
		items = append(items, es.IndexItem{
			ID:      doc.ContentID,
			Version: c.UpdatedAt.UnixMilli(),
			Doc:     doc,
		})
	}
	return items, nil
}

// AssembleUserDocs 组装用户文档 无子表回源
func AssembleUserDocs(users []*model.RanFeedUser) []es.IndexItem {
	items := make([]es.IndexItem, 0, len(users))
	for _, u := range users {
		if u == nil {
			continue
		}
		items = append(items, es.IndexItem{
			ID:      strconv.FormatInt(u.ID, 10),
			Version: u.UpdatedAt.UnixMilli(),
			Doc: UserDoc{
				UserID:    strconv.FormatInt(u.ID, 10),
				Nickname:  u.Nickname,
				Bio:       u.Bio,
				Username:  u.Username,
				Status:    u.Status,
				IsDeleted: u.IsDeleted,
			},
		})
	}
	return items
}

func millis(t *time.Time) int64 {
	if t == nil {
		return 0
	}
	return t.UnixMilli()
}
