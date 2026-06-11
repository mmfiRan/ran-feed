package redis

const (
	// RedisInteractionLikePrefix 点赞用户集合前缀 like:action:{scene}:{content_id}
	RedisInteractionLikePrefix = "like:action"
	// RedisLikeUserPrefix 用户维度点赞HASH前缀 like:user:{user_id}
	RedisLikeUserPrefix = "like:user"
	// RedisInteractionLikeCountPrefix 点赞计数前缀 like:count:{scene}:{content_id}
	RedisInteractionLikeCountPrefix = "like:count"

	// RedisLikeUserHashCapacity 用户维度点赞热数据容量上限（超过则进入冷数据，回源DB）
	RedisLikeUserHashCapacity = 5000
	// RedisLikeUserHashMetaFieldMinCid 用户维度点赞HASH元信息：热区最小content_id（小于该值视为冷数据）
	RedisLikeUserHashMetaFieldMinCid = "_mincid"
	// RedisLikeUserHashMetaFieldFull 用户维度点赞HASH元信息：完整性标记（"1"=完整热区，可信；缺失=残缺，需回源/交下游）
	RedisLikeUserHashMetaFieldFull = "_full"
	// RedisLikeExpireSeconds 点赞缓存过期时间（秒）；0 表示不过期
	RedisLikeExpireSeconds      = 5 * 24 * 60 * 60
	RedisFeedUserFavoritePrefix = "feed:user:favorite"

	// RedisCommentObjPrefix 评论对象缓存前缀 comment:obj:{comment_id}
	RedisCommentObjPrefix = "comment:obj"
	// RedisCommentIdxContentPrefix 一级评论索引前缀 comment:idx:content:{content_id}
	RedisCommentIdxContentPrefix = "comment:idx:content"
	// RedisCommentIdxRootPrefix 评论回复索引前缀 comment:idx:root:{root_id}
	RedisCommentIdxRootPrefix = "comment:idx:root"
	// RedisCommentObjExpireSeconds 评论对象缓存过期时间：24小时
	RedisCommentObjExpireSeconds = 24 * 60 * 60
	// RedisCommentIdxExpireSeconds 评论索引过期时间：20分钟
	RedisCommentIdxExpireSeconds = 20 * 60
	// RedisCommentIdxKeepLatestN 评论索引保留最新N条
	RedisCommentIdxKeepLatestN = 10000
)

func GetRedisPrefixKey(prefix string, id string) string {
	return prefix + ":" + id
}

func BuildLikeKey(scene string, contentId string) string {
	return GetRedisPrefixKey(GetRedisPrefixKey(RedisInteractionLikePrefix, scene), contentId)
}

func BuildLikeUserKey(userId string) string {
	return GetRedisPrefixKey(RedisLikeUserPrefix, userId)
}

func BuildLikeCountKey(scene string, contentId string) string {
	return GetRedisPrefixKey(GetRedisPrefixKey(RedisInteractionLikeCountPrefix, scene), contentId)
}

func BuildUserFavoriteFeedKey(userId string) string {
	return GetRedisPrefixKey(RedisFeedUserFavoritePrefix, userId)
}

func BuildCommentObjKey(commentId string) string {
	return GetRedisPrefixKey(RedisCommentObjPrefix, commentId)
}

func BuildCommentIdxContentKey(contentId string) string {
	return GetRedisPrefixKey(RedisCommentIdxContentPrefix, contentId)
}

func BuildCommentIdxRootKey(rootId string) string {
	return GetRedisPrefixKey(RedisCommentIdxRootPrefix, rootId)
}
