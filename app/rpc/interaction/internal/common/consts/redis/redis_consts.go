package redis

const (
	// RedisInteractionLikePrefix 点赞用户集合前缀 like:action:{scene}:{content_id}
	RedisInteractionLikePrefix = "like:action"
	// RedisLikeUserPrefix 用户维度点赞HASH前缀 like:user:{user_id}
	RedisLikeUserPrefix = "like:user"
	// RedisInteractionLikeCountPrefix 点赞计数前缀 like:count:{scene}:{content_id}
	RedisInteractionLikeCountPrefix = "like:count"

	// RedisLikeUserHashCapacity 用户维度点赞热数据容量上限（超过则进入冷数据，回源DB）
	RedisLikeUserHashCapacity = 10000
	// RedisLikeUserHashMetaFieldMinCid 用户维度点赞HASH元信息：热区最小content_id（小于该值视为冷数据）
	RedisLikeUserHashMetaFieldMinCid = "_mincid"
	// RedisLikeUserHashMetaFieldExpireAt 用户维度点赞HASH元信息：逻辑过期时间戳（本期保留字段，不做过期重建）
	RedisLikeUserHashMetaFieldExpireAt = "_expire_at"
	// RedisLikeUserHashMetaFieldSize 用户维度点赞HASH元信息：热区当前容量（可选，用于快速判断是否达到上限）
	RedisLikeUserHashMetaFieldSize = "_size"
	// RedisLikeUserHashMetaFieldVersion 用户维度点赞HASH元信息：版本号（预留，便于后续结构演进）
	RedisLikeUserHashMetaFieldVersion = "_ver"
	// RedisLikeUserHashMetaPrefix 用户维度点赞HASH元信息field前缀（避免与content_id冲突）
	RedisLikeUserHashMetaPrefix = "_"
	// RedisLikeExpireSeconds 点赞缓存过期时间（秒）；0 表示不过期
	RedisLikeExpireSeconds                = 5 * 24 * 60 * 60
	RedisFavoriteRelExpireSeconds         = 24 * 60 * 60
	RedisFavoriteRelNegativeExpireSeconds = 10 * 60
	RedisFavoriteRelPrefix                = "favorite:rel"
	RedisFeedUserFavoritePrefix           = "feed:user:favorite"

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

func BuildFavoriteRelKey(scene string, userId string, contentId string) string {
	return GetRedisPrefixKey(GetRedisPrefixKey(RedisFavoriteRelPrefix, scene), userId+":"+contentId)
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
