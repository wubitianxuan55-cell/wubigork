package proposal

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// search_test.go — 索引优化批（2026-11-28）的检索单测：总数与翻页、分面计数、
// 文档顺序、多词放宽、限定在某标题之下、命中来源、语义路失败提示。

// seedWide 一份两章、每章三个二级小节的方案，另加一份单节方案。每个小节正文都含
// 「监测」；前一章的三个小节与另一份方案含「降水」，后一章的三个小节含「扬尘」。
func seedWide(t *testing.T, st *Store) (pidA, pidB int64) {
	t.Helper()
	var secs []saveSection
	secs = append(secs, saveSection{1, "第1章 施工", "监测方案总述。"})
	for i := 1; i <= 3; i++ {
		secs = append(secs, saveSection{2, fmt.Sprintf("1.%d 降水措施", i), fmt.Sprintf("第 1.%d 节：监测周边沉降，降水井布置。", i)})
	}
	secs = append(secs, saveSection{1, "第2章 安全", "安全监测要求。"})
	for i := 1; i <= 3; i++ {
		secs = append(secs, saveSection{2, fmt.Sprintf("2.%d 文明施工", i), fmt.Sprintf("第 2.%d 节：扬尘监测与围挡。", i)})
	}
	a := saveA(t, st, "宽方案", secs)
	b := saveA(t, st, "监测简报", []saveSection{{1, "概述", "监测简报正文，讲降水。"}})
	return a.ProposalID, b.ProposalID
}

func hitIDs(res *SearchResult) []int64 {
	var out []int64
	for _, h := range res.Hits {
		out = append(out, h.SectionID)
	}
	return out
}

func hitTitles(res *SearchResult) []string {
	var out []string
	for _, h := range res.Hits {
		out = append(out, h.Title)
	}
	return out
}

func mustSearch(t *testing.T, st *Store, in SearchInput) *SearchResult {
	t.Helper()
	res, err := st.Search(in)
	if err != nil {
		t.Fatalf("Search(%+v): %v", in, err)
	}
	return res
}

func TestSearchTotalAndPaging(t *testing.T) {
	st := newStore(t)
	seedWide(t, st)

	all := mustSearch(t, st, SearchInput{Query: "监测"})
	if len(all.Hits) != 9 || all.Total != 9 || all.Offset != 0 {
		t.Fatalf("默认一页应取全 9 条：len=%d total=%d", len(all.Hits), all.Total)
	}
	// 一页 4 条翻三页：页页相接、不重不漏；翻过头为空但总数照报。
	var got []int64
	for _, off := range []int{0, 4, 8} {
		res := mustSearch(t, st, SearchInput{Query: "监测", TopN: 4, Offset: off})
		if res.Offset != off || res.Total != 9 {
			t.Fatalf("第 %d 条起：offset=%d total=%d", off, res.Offset, res.Total)
		}
		got = append(got, hitIDs(res)...)
	}
	if len(got) != 9 {
		t.Fatalf("三页合计 %d 条，应为 9：%v", len(got), got)
	}
	seen := map[int64]bool{}
	for _, id := range got {
		if seen[id] {
			t.Fatalf("翻页出现重复：%d", id)
		}
		seen[id] = true
	}
	if res := mustSearch(t, st, SearchInput{Query: "监测", TopN: 4, Offset: 9}); len(res.Hits) != 0 || res.Total != 9 {
		t.Fatalf("翻过末页：%d total=%d", len(res.Hits), res.Total)
	}
	// 过滤后的总数只算过滤后的；指定的方案 / 标题不存在时为空。
	if res := mustSearch(t, st, SearchInput{Query: "监测", Level: 1}); res.Total != 3 || len(res.Hits) != 3 {
		t.Fatalf("层级过滤后总数：total=%d len=%d", res.Total, len(res.Hits))
	}
	if res := mustSearch(t, st, SearchInput{Query: "监测", Under: 99999}); res.Total != 0 || len(res.Hits) != 0 {
		t.Fatalf("标题不存在：%+v", res)
	}
}

func TestSearchFacets(t *testing.T) {
	st := newStore(t)
	pid, _ := seedWide(t, st)
	ch1 := sectionIDByTitle(t, st, pid, "第1章 施工")

	// 不要分面时不算。
	if res := mustSearch(t, st, SearchInput{Query: "监测", TopN: 3}); res.Total != 9 || res.ByProposal != nil || res.ByChapter != nil {
		t.Fatalf("未要分面：total=%d %+v %+v", res.Total, res.ByProposal, res.ByChapter)
	}
	// 全库：按方案计数（不受条数上限影响）；没限定方案就没有「章」。
	res := mustSearch(t, st, SearchInput{Query: "监测", TopN: 3, Facets: true})
	byName := map[string]int{}
	for _, f := range res.ByProposal {
		byName[f.Label] = f.Count
	}
	if len(res.Hits) != 3 || byName["宽方案"] != 8 || byName["监测简报"] != 1 || res.ByChapter != nil {
		t.Fatalf("按方案分页：%d 条 %+v 章=%+v", len(res.Hits), res.ByProposal, res.ByChapter)
	}
	// 限定方案：分面里方案计数仍是两份都列（不然恒等于本方案、没法切过去），章按标题序。
	res = mustSearch(t, st, SearchInput{Query: "监测", Proposal: "宽方案", TopN: 3, Facets: true})
	if res.Total != 8 || len(res.ByProposal) != 2 || len(res.ByChapter) != 2 ||
		res.ByChapter[0].Key != fmt.Sprint(ch1) || res.ByChapter[0].Label != "第1章 施工" || res.ByChapter[0].Count != 4 ||
		res.ByChapter[1].Label != "第2章 安全" || res.ByChapter[1].Count != 4 {
		t.Fatalf("限定方案的分面：total=%d 方案=%+v 章=%+v", res.Total, res.ByProposal, res.ByChapter)
	}
	// 限定到第1章之下：命中只剩这一章（含章标题自己的正文），章分面仍是整册的。
	res = mustSearch(t, st, SearchInput{Query: "监测", Proposal: "宽方案", Under: ch1, Facets: true})
	if res.Total != 4 || len(res.Hits) != 4 || len(res.ByChapter) != 2 || res.ByChapter[1].Count != 4 {
		t.Fatalf("限定章：total=%d hits=%v 章=%+v", res.Total, hitTitles(res), res.ByChapter)
	}
	for _, h := range res.Hits {
		if h.ProposalName != "宽方案" || (h.Title != "第1章 施工" && h.Title[0] != '1') {
			t.Fatalf("限定章里混进别处命中：%s", h.TitlePath)
		}
	}
	// 标题范围 + 层级：只剩该章下的二级小节。
	if res := mustSearch(t, st, SearchInput{Query: "监测", Under: ch1, Level: 2}); res.Total != 3 || len(res.Hits) != 3 {
		t.Fatalf("限定章 + 层级：total=%d len=%d", res.Total, len(res.Hits))
	}
}

func TestSearchSortDoc(t *testing.T) {
	st := newStore(t)
	seedWide(t, st)
	// 文档顺序：方案按清单序（最近更新的在前），方案内按标题序。
	want := []string{"概述", "第1章 施工", "1.1 降水措施", "1.2 降水措施", "1.3 降水措施", "第2章 安全", "2.1 文明施工", "2.2 文明施工", "2.3 文明施工"}
	res := mustSearch(t, st, SearchInput{Query: "监测", Sort: SortDoc})
	if res.Mode != "keyword" || res.Total != 9 || len(res.Hits) != 9 {
		t.Fatalf("文档顺序：mode=%q total=%d len=%d", res.Mode, res.Total, len(res.Hits))
	}
	got := hitTitles(res)
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("文档顺序第 %d 条 = %s, want %s（全序 %v）", i, got[i], want[i], got)
		}
	}
	if res.Hits[0].ProposalName != "监测简报" {
		t.Fatalf("最近更新的方案应排在前：%+v", res.Hits[0])
	}
	// 文档顺序下翻页接着标题序走。
	res = mustSearch(t, st, SearchInput{Query: "监测", Sort: SortDoc, TopN: 3, Offset: 3})
	if len(res.Hits) != 3 || res.Hits[0].Title != "1.2 降水措施" || res.Hits[2].Title != "第2章 安全" {
		t.Fatalf("文档顺序翻页：%v", hitTitles(res))
	}
	// 标题路 + 文档顺序：只在本份方案里按标题序列标题含「降水」的节点。
	res = mustSearch(t, st, SearchInput{Query: "降水", Scope: ScopeTitle, Sort: SortDoc, Proposal: "宽方案"})
	if res.Mode != "title" || len(res.Hits) != 3 || res.Hits[0].Title != "1.1 降水措施" || res.Hits[2].Title != "1.3 降水措施" {
		t.Fatalf("标题路文档顺序：mode=%q %v", res.Mode, hitTitles(res))
	}
}

func TestSearchRelaxMultiTerms(t *testing.T) {
	st := newStore(t)
	seedWide(t, st)
	// 「降水」与「扬尘」从不在同一节出现：全含零命中，放宽为任一词并注明。
	res := mustSearch(t, st, SearchInput{Query: "降水 扬尘"})
	if !res.Relaxed || res.Total != 7 || len(res.Hits) != 7 {
		t.Fatalf("多词放宽：relaxed=%v total=%d len=%d", res.Relaxed, res.Total, len(res.Hits))
	}
	// 全含有结果就不放宽；单词查询零命中也不算放宽。
	if res := mustSearch(t, st, SearchInput{Query: "监测 降水"}); res.Relaxed || res.Total != 4 {
		t.Fatalf("全含有结果：relaxed=%v total=%d", res.Relaxed, res.Total)
	}
	if res := mustSearch(t, st, SearchInput{Query: "库里没有的词"}); res.Relaxed || res.Total != 0 {
		t.Fatalf("单词零命中：relaxed=%v total=%d", res.Relaxed, res.Total)
	}
	// 放宽后的分面口径跟着放宽后的命中集走。
	res = mustSearch(t, st, SearchInput{Query: "降水 扬尘", Facets: true})
	sum := 0
	for _, f := range res.ByProposal {
		sum += f.Count
	}
	if !res.Relaxed || sum != 7 {
		t.Fatalf("放宽后分面：%+v", res.ByProposal)
	}
	// 长短词混合放宽：短词走不了全文索引，整组改走 LIKE。
	res = mustSearch(t, st, SearchInput{Query: "扬尘 监测简报"})
	if !res.Relaxed || res.Total != 4 {
		t.Fatalf("长短词放宽：relaxed=%v total=%d", res.Relaxed, res.Total)
	}
}

func TestSearchUnderAppliesToSemanticRoad(t *testing.T) {
	st := newStore(t)
	pid := ingestMD(t, st)
	ctx := context.Background()
	em := &fakeEmbedder{dim: 32}
	if _, _, err := st.EmbedPending(ctx, em, "fake-embed", 0, 5*time.Second); err != nil {
		t.Fatal(err)
	}
	sem := &Semantic{Embed: em, Model: "fake-embed"}
	q := "降水的时候旁边房子沉了怎么办" // 全文路零命中，只走语义路
	ch1 := sectionIDByTitle(t, st, pid, "第1章 项目实施方案")
	ch4 := sectionIDByTitle(t, st, pid, "第4章 安全保障方案")

	// 语义召回只按方案收窄：限定到哪一章之下必须在这里再筛一道，否则会混进别章。
	for _, tc := range []struct {
		under  int64
		prefix string
	}{
		{ch1, "第1章 项目实施方案"}, {ch4, "第4章 安全保障方案"},
	} {
		res, err := st.HybridSearch(ctx, SearchInput{Query: q, Under: tc.under, TopN: 30, Facets: true}, sem)
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Hits) == 0 {
			t.Fatalf("限定标题「%s」：应有语义命中", tc.prefix)
		}
		for _, h := range res.Hits {
			if !strings.HasPrefix(h.TitlePath, tc.prefix) {
				t.Fatalf("限定「%s」之下混进别处命中：%s", tc.prefix, h.TitlePath)
			}
			if h.Source != "semantic" {
				t.Fatalf("只被语义路召回的命中来源应为 semantic：%s source=%q", h.Title, h.Source)
			}
		}
		// 分面按全文路计数：纯语义命中没有关键词可数，章分面为空；总数不少于本页。
		if len(res.ByChapter) != 0 || res.Total < len(res.Hits) {
			t.Fatalf("限定「%s」：章分面=%+v total=%d len=%d", tc.prefix, res.ByChapter, res.Total, len(res.Hits))
		}
	}
}

func TestHybridSearchPagingSourcesAndFailure(t *testing.T) {
	st := newStore(t)
	ingestMD(t, st)
	ctx := context.Background()
	em := &fakeEmbedder{dim: 32}
	if _, _, err := st.EmbedPending(ctx, em, "fake-embed", 0, 5*time.Second); err != nil {
		t.Fatal(err)
	}
	sem := &Semantic{Embed: em, Model: "fake-embed", Rerank: fakeReranker{available: true}}

	// 命中来源：纯全文路是 keyword；混合检索里两路都命中的是 both。
	kw := mustSearch(t, st, SearchInput{Query: "边坡失稳", TopN: 10})
	for _, h := range kw.Hits {
		if h.Source != "keyword" {
			t.Fatalf("全文路命中来源：%q", h.Source)
		}
	}
	res, err := st.HybridSearch(ctx, SearchInput{Query: "边坡失稳", TopN: 10, Facets: true}, sem)
	if err != nil || res.Mode != "hybrid" || res.Total < kw.Total || len(res.ByProposal) != 1 || res.Offset != 0 {
		t.Fatalf("混合检索分面：err=%v mode=%q total=%d（全文路 %d）方案=%+v", err, res.Mode, res.Total, kw.Total, res.ByProposal)
	}
	// 混合检索的命中来源三样都可能：两路都命中 both、只被语义路召回 semantic、
	// 语义路没召回的纯全文命中 keyword；至少要有一条 both。
	sawBoth := false
	for _, h := range res.Hits {
		if h.Source != "both" && h.Source != "semantic" && h.Source != "keyword" {
			t.Fatalf("混合检索命中来源：%q", h.Source)
		}
		sawBoth = sawBoth || h.Source == "both"
	}
	if !sawBoth {
		t.Fatalf("应有全文与语义都命中的条目：%+v", res.Hits)
	}

	// 偏移超过候选池：不再调本地模型，直接按全文路翻页。
	cold := &fakeEmbedder{dim: 32}
	page, err := st.HybridSearch(ctx, SearchInput{Query: "扬尘", Offset: rerankPool, TopN: 10}, &Semantic{Embed: cold, Model: "fake-embed"})
	if err != nil || cold.calls != 0 || page.Mode != "keyword" || page.Total != 2 || page.Offset != rerankPool {
		t.Fatalf("深翻页：err=%v calls=%d mode=%q total=%d", err, cold.calls, page.Mode, page.Total)
	}

	// 本地 embedding 挂了：退成纯全文检索，不报错，回执注明原因与总数。
	down := &fakeEmbedder{dim: 32, err: errors.New("device lost on vk::Queue::submit")}
	res, err = st.HybridSearch(ctx, SearchInput{Query: "边坡失稳", TopN: 10, Facets: true}, &Semantic{Embed: down, Model: "fake-embed"})
	if err != nil || res.Mode != "keyword" || len(res.Hits) != 2 || res.Total != 2 || len(res.ByProposal) != 1 ||
		!strings.Contains(res.SearchError, "device lost") || !strings.Contains(res.Note, "全文路") {
		t.Fatalf("语义路失败降级：err=%v mode=%q total=%d searchError=%q note=%q", err, res.Mode, res.Total, res.SearchError, res.Note)
	}
}
