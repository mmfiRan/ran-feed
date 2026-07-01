package searchservicelogic

import (
	"context"
	"sync"

	"ran-feed/app/rpc/search/internal/es"
	"ran-feed/app/rpc/search/internal/svc"
	"ran-feed/app/rpc/search/search"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

type SuggestLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSuggestLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SuggestLogic {
	return &SuggestLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Suggest 前缀补全 并行查内容标题与用户昵称两索引 合并成带 type 的建议 单索引失败仅该组为空不整体报错
func (l *SuggestLogic) Suggest(in *search.SuggestReq) (*search.SuggestRes, error) {
	res := &search.SuggestRes{Items: []*search.SuggestItem{}}
	if in == nil || in.Keyword == "" {
		return res, nil
	}
	size := normalizeSize(in.Size)

	var (
		wg           sync.WaitGroup
		contentTexts []string
		userTexts    []string
	)
	wg.Add(2)
	threading.GoSafe(func() {
		defer wg.Done()
		contentTexts = l.suggestOne(es.IndexContent, es.FieldTitleSuggest, in.Keyword, size)
	})
	threading.GoSafe(func() {
		defer wg.Done()
		userTexts = l.suggestOne(es.IndexUser, es.FieldNicknameSuggest, in.Keyword, size)
	})
	wg.Wait()

	for _, t := range contentTexts {
		res.Items = append(res.Items, &search.SuggestItem{
			Text: t,
			Type: search.SuggestType_SUGGEST_CONTENT,
		})
	}
	for _, t := range userTexts {
		res.Items = append(res.Items, &search.SuggestItem{
			Text: t,
			Type: search.SuggestType_SUGGEST_USER,
		})
	}
	return res, nil
}

// suggestOne 查单个索引的补全 失败只记日志返回空 保证按键接口稳定
func (l *SuggestLogic) suggestOne(index, field, prefix string, size int) []string {
	texts, err := es.Suggest(l.ctx, l.svcCtx.ES, index, field, prefix, size)
	if err != nil {
		l.Errorf("补全失败 index=%s prefix=%s err=%v", index, prefix, err)
		return nil
	}
	return texts
}
