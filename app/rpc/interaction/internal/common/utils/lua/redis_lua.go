package lua

import _ "embed"

// LikeUserHashScript 用户维度点赞 HASH 写入脚本
//
//go:embed like_user_hash.lua
var LikeUserHashScript string

// CancelLikeUserHashScript 用户维度取消点赞 HASH 脚本
//
//go:embed cancel_like_user_hash.lua
var CancelLikeUserHashScript string

// RebuildLikeUserHashScript 用户维度点赞 HASH 原子重建脚本
//
//go:embed rebuild_like_user_hash.lua
var RebuildLikeUserHashScript string

// BatchGetCommentObjsScript 批量获取评论对象 HASH
//
//go:embed batch_get_comment_objs.lua
var BatchGetCommentObjsScript string

// UpdateCommentObjScript 更新评论对象 HASH
//
//go:embed update_comment_obj.lua
var UpdateCommentObjScript string
