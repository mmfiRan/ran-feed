package indexer

import (
	"testing"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/user/user"

	"github.com/stretchr/testify/assert"
)

func TestContentIndexItemToItem(t *testing.T) {
	it := &content.ContentIndexItem{
		ContentId:   1,
		ContentType: content.ContentType_ARTICLE,
		Status:      content.ContentStatus_PUBLISHED,
		Visibility:  content.Visibility_PUBLIC,
		AuthorId:    100,
		Title:       "标题A",
		Description: "简介A",
		Body:        "正文A",
		PublishedAt: 1_700_000_000_000,
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
}
