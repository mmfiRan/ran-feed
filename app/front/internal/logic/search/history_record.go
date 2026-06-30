package search

import (
	"context"
	"strings"
	"time"

	"ran-feed/app/front/internal/svc"
	"ran-feed/app/rpc/search/search"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

// recordSearchHistory 搜索时隐式记录历史 登录用户非空词才记 异步脱离请求 ctx 非致命
func recordSearchHistory(svcCtx *svc.ServiceContext, userID int64, keyword string) {
	if userID <= 0 || strings.TrimSpace(keyword) == "" {
		return
	}

	bgCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	threading.GoSafe(func() {
		defer cancel()
		if _, err := svcCtx.SearchRpc.RecordHistory(bgCtx, &search.RecordHistoryReq{
			UserId:  userID,
			Keyword: keyword,
		}); err != nil {
			logx.WithContext(bgCtx).Errorf("记录搜索历史失败 userID=%d keyword=%s err=%v", userID, keyword, err)
		}
	})
}
