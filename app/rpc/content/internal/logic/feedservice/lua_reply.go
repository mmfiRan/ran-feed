package feedservicelogic

import "strconv"

// scoredID feed zset 成员 member 为 content_id score 为 published_at 毫秒
type scoredID struct {
	id    int64
	score int64
}

// parseZSetReply 解析 query lua 返回 [exists hasMore nextCursor member1 score1 member2 score2 ...]
// exists 为假时 ok 仍为真表示调用成功只是缓存不存在 ok 为假表示返回格式非法
func parseZSetReply(res any) (items []scoredID, nextCursor string, hasMore, exists, ok bool) {
	arr, isArr := res.([]any)
	if !isArr || len(arr) < 3 {
		return nil, "", false, false, false
	}
	existsVal, _ := luaReplyInt64(arr[0])
	if existsVal != 1 {
		return nil, "", false, false, true
	}
	hasMoreVal, _ := luaReplyInt64(arr[1])
	hasMore = hasMoreVal == 1
	if hasMore {
		if s, sok := luaReplyString(arr[2]); sok {
			nextCursor = s
		}
	}
	items = make([]scoredID, 0, (len(arr)-3)/2)
	for i := 3; i+1 < len(arr); i += 2 {
		memberStr, _ := luaReplyString(arr[i])
		scoreStr, _ := luaReplyString(arr[i+1])
		id, idErr := strconv.ParseInt(memberStr, 10, 64)
		scoreF, scoreErr := strconv.ParseFloat(scoreStr, 64)
		if idErr != nil || id <= 0 || scoreErr != nil {
			continue
		}
		items = append(items, scoredID{id: id, score: int64(scoreF)})
	}
	return items, nextCursor, hasMore, true, true
}

func luaReplyString(v any) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, true
	case []byte:
		return string(t), true
	default:
		return "", false
	}
}

func luaReplyInt64(v any) (int64, bool) {
	switch t := v.(type) {
	case int64:
		return t, true
	case int:
		return int64(t), true
	case string:
		n, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			return 0, false
		}
		return n, true
	case []byte:
		n, err := strconv.ParseInt(string(t), 10, 64)
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}
