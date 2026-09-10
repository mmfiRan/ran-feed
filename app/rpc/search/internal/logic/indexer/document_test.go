package indexer

import (
	"testing"
	"time"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/user/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestContentIndexItemToItem(t *testing.T) {
	it := &content.ContentIndexItem{
		ContentId:   1,
		ContentType: content.ContentType_CONTENT_TYPE_ARTICLE,
		Status:      content.ContentStatus_CONTENT_STATUS_PUBLISHED,
		Visibility:  content.Visibility_VISIBILITY_PUBLIC,
		AuthorId:    100,
		Title:       "标题A",
		Description: "简介A",
		Body:        "正文A",
		PublishedAt: timestamppb.New(time.UnixMilli(1_700_000_000_000)),
		HotScore:    1.5,
		Version:     1_700_000_123_456,
	}

	item := ContentIndexItemToItem(it)

	// id 转字符串 version 透传投影 version
	assert.Equal(t, "1", item.ID)
	assert.Equal(t, int64(1_700_000_123_456), item.Version)

	doc := item.Doc.(ContentDoc)
	assert.Equal(t, "1", doc.ContentID)
	assert.Equal(t, int32(10), doc.ContentType)
	assert.Equal(t, int32(30), doc.Status)
	assert.Equal(t, int32(10), doc.Visibility)
	assert.Equal(t, "100", doc.AuthorID)
	assert.Equal(t, "标题A", doc.Title)
	assert.Equal(t, "简介A", doc.Description)
	assert.Equal(t, "正文A", doc.Body)
	assert.Equal(t, int64(1_700_000_000_000), doc.PublishedAt)
	assert.InDelta(t, 1.5, doc.HotScore, 1e-9)
	// 投影只含可索引内容 is_deleted 恒 0
	assert.Equal(t, int32(0), doc.IsDeleted)
}

func TestUserIndexItemToItem(t *testing.T) {
	it := &user.UserIndexItem{
		UserId:   10,
		Nickname: "爱丽丝",
		Bio:      "简介",
		Username: "alice",
		Status:   user.UserStatus_USER_STATUS_ACTIVE,
		Version:  1_700_000_123_456,
	}

	item := UserIndexItemToItem(it)

	assert.Equal(t, "10", item.ID)
	assert.Equal(t, int64(1_700_000_123_456), item.Version)

	doc := item.Doc.(UserDoc)
	assert.Equal(t, "10", doc.UserID)
	assert.Equal(t, "爱丽丝", doc.Nickname)
	assert.Equal(t, "简介", doc.Bio)
	assert.Equal(t, "alice", doc.Username)
	assert.Equal(t, int32(10), doc.Status)
	assert.Equal(t, int32(0), doc.IsDeleted)
	// 昵称补全填充 用户暂无热度 weight 恒 0
	require.NotNil(t, doc.NicknameSuggest)
	assert.Equal(t, []string{"爱丽丝"}, doc.NicknameSuggest.Input)
	assert.Equal(t, 0, doc.NicknameSuggest.Weight)
}

func TestContentIndexItemSuggest(t *testing.T) {
	// 有标题 weight 取 hot_score 取整
	item := ContentIndexItemToItem(&content.ContentIndexItem{
		ContentId: 1, ContentType: content.ContentType_CONTENT_TYPE_ARTICLE, Title: "露营装备测评", HotScore: 8.9,
	})
	doc := item.Doc.(ContentDoc)
	require.NotNil(t, doc.TitleSuggest)
	assert.Equal(t, []string{"露营装备测评"}, doc.TitleSuggest.Input)
	assert.Equal(t, 8, doc.TitleSuggest.Weight)

	// 空标题(异常视频)不产出补全字段 避免 ES 拒绝空 input
	noTitle := ContentIndexItemToItem(&content.ContentIndexItem{ContentId: 2, ContentType: content.ContentType_CONTENT_TYPE_VIDEO, Title: ""})
	assert.Nil(t, noTitle.Doc.(ContentDoc).TitleSuggest)

	// 负 hot_score 归零
	neg := ContentIndexItemToItem(&content.ContentIndexItem{ContentId: 3, ContentType: content.ContentType_CONTENT_TYPE_ARTICLE, Title: "x", HotScore: -5})
	assert.Equal(t, 0, neg.Doc.(ContentDoc).TitleSuggest.Weight)
}
