package lua

import _ "embed"

// LikeUserHashScript 用户维度点赞HASH写入脚本
//
//go:embed like_user_hash.lua
var LikeUserHashScript string

// CancelLikeUserHashScript 用户维度取消点赞HASH脚本
//
//go:embed cancel_like_user_hash.lua
var CancelLikeUserHashScript string

// QueryIsLikedUserHashScript 用户维度查询是否点赞HASH脚本
//
//go:embed query_is_liked_user_hash.lua
var QueryIsLikedUserHashScript string

// QueryIsLikedUserHashBatchScript 用户维度批量查询是否点赞HASH脚本
//
//go:embed query_is_liked_user_hash_batch.lua
var QueryIsLikedUserHashBatchScript string

// UpdateCommentCacheScript 评论缓存更新脚本
//
//go:embed update_comment_cache.lua
var UpdateCommentCacheScript string

// QueryCommentListScript 评论列表查询脚本
//
//go:embed query_comment_list.lua
var QueryCommentListScript string

// BatchGetCommentObjsScript 批量获取评论对象HASH
//
//go:embed batch_get_comment_objs.lua
var BatchGetCommentObjsScript string

// UpdateCommentObjScript 更新评论对象HASH
//
//go:embed update_comment_obj.lua
var UpdateCommentObjScript string

// AddUserFavoriteIfExistsScript 用户收藏列表ZSET存在时追加Lua脚本
//
//go:embed add_user_favorite_if_exists.lua
var AddUserFavoriteIfExistsScript string
