package es

import _ "embed"

// 索引名 演进改 mapping 时建新版索引 reindex 后 alias 切换 代码只认这两个名
const (
	IndexContent = "ran-feed-content"
	IndexUser    = "ran-feed-user"
)

// 自动补全 completion 字段名 与 mapping 对齐
const (
	FieldTitleSuggest    = "title_suggest"
	FieldNicknameSuggest = "nickname_suggest"
)

// contentMapping 内容索引 title^3 description^2 body^1 中文分词 ik body 不入 _source 控体积
//
//go:embed content_mapping.json
var contentMapping string

// userMapping 用户索引 nickname^3 bio^1 nickname 带 keyword 子字段供精确匹配
//
//go:embed user_mapping.json
var userMapping string
