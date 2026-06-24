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

// RebuildLikeUserHashScript 用户维度点赞HASH原子重建脚本
//
//go:embed rebuild_like_user_hash.lua
var RebuildLikeUserHashScript string

// BatchGetCommentObjsScript 批量获取评论对象HASH
//
//go:embed batch_get_comment_objs.lua
var BatchGetCommentObjsScript string

// UpdateCommentObjScript 更新评论对象HASH
//
//go:embed update_comment_obj.lua
var UpdateCommentObjScript string
