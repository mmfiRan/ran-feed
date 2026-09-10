package indexer

import (
	"math"
	"strconv"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/search/internal/es"
	"ran-feed/app/rpc/user/user"
)

// completionInput ES completion 字段的输入 input 为补全短语 weight 越大越靠前
type completionInput struct {
	Input  []string `json:"input"`
	Weight int      `json:"weight,omitempty"`
}

// ContentDoc 内容索引文档 id 类字段按 keyword 存字符串 published_at 毫秒
type ContentDoc struct {
	ContentID    string           `json:"content_id"`
	ContentType  int32            `json:"content_type"`
	Status       int32            `json:"status"`
	Visibility   int32            `json:"visibility"`
	AuthorID     string           `json:"author_id"`
	Title        string           `json:"title"`
	Description  string           `json:"description"`
	Body         string           `json:"body"`
	PublishedAt  int64            `json:"published_at"`
	HotScore     float64          `json:"hot_score"`
	IsDeleted    int32            `json:"is_deleted"`
	TitleSuggest *completionInput `json:"title_suggest,omitempty"`
}

// UserDoc 用户索引文档
type UserDoc struct {
	UserID          string           `json:"user_id"`
	Nickname        string           `json:"nickname"`
	Bio             string           `json:"bio"`
	Username        string           `json:"username"`
	Status          int32            `json:"status"`
	IsDeleted       int32            `json:"is_deleted"`
	NicknameSuggest *completionInput `json:"nickname_suggest,omitempty"`
}

// suggestField 构造 completion 输入 空文本返回 nil 使字段整体省略 避免 ES 拒绝空 input
func suggestField(text string, weight int) *completionInput {
	if text == "" {
		return nil
	}
	return &completionInput{Input: []string{text}, Weight: weight}
}

// hotScoreWeight hot_score 转 completion 权重 非正归零 上限截到 int32
func hotScoreWeight(score float64) int {
	if score <= 0 {
		return 0
	}
	if score > math.MaxInt32 {
		return math.MaxInt32
	}
	return int(score)
}

// ContentIndexItemToItem 由 content-rpc 索引投影映射为 ES 写入项 投影只含可索引内容 故 is_deleted 恒 0
func ContentIndexItemToItem(it *content.ContentIndexItem) es.IndexItem {
	doc := ContentDoc{
		ContentID:   strconv.FormatInt(it.ContentId, 10),
		ContentType: int32(it.ContentType),
		Status:      int32(it.Status),
		Visibility:  int32(it.Visibility),
		AuthorID:    strconv.FormatInt(it.AuthorId, 10),
		Title:       it.Title,
		Description: it.Description,
		Body:        it.Body,
		PublishedAt: it.PublishedAt.AsTime().UnixMilli(),
		HotScore:    it.HotScore,
		IsDeleted:   0,
	}
	doc.TitleSuggest = suggestField(it.Title, hotScoreWeight(it.HotScore))
	return es.IndexItem{ID: doc.ContentID, Version: it.Version, Doc: doc}
}

// UserIndexItemToItem 由 user-rpc 索引投影映射为 ES 写入项 投影只含可索引用户 故 is_deleted 恒 0
func UserIndexItemToItem(it *user.UserIndexItem) es.IndexItem {
	doc := UserDoc{
		UserID:    strconv.FormatInt(it.UserId, 10),
		Nickname:  it.Nickname,
		Bio:       it.Bio,
		Username:  it.Username,
		Status:    int32(it.Status),
		IsDeleted: 0,
	}
	// 用户暂无热度信号 weight 恒 0 后续可接粉丝数
	doc.NicknameSuggest = suggestField(it.Nickname, 0)
	return es.IndexItem{ID: doc.UserID, Version: it.Version, Doc: doc}
}
