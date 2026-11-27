package workcost

import (
	"context"
	"testing"
)

// 仅字面匹配（SearchQuery.Exact，2026-11-27 AI 反馈 #8）：常规关键词路会按同义
// 说法经向量路召回；exact 只认字段里含查询词的行，按编码稳定排序、全部交回
// （条数与翻页交给调用方），过滤条件照常生效。
func TestSearch_ExactKeywordOnly(t *testing.T) {
	s, gdb, cleanup := searchStore(t)
	defer cleanup()
	saveSearchFixtures(t, s)
	emb := &bagEmbedder{buckets: searchBuckets}
	h := NewSearcher(s, buildIndex(t, gdb, s, emb), func() SearchEmbedder { return emb }, nil)
	ctx := context.Background()

	// 常规关键词路：「挖机」是「挖掘机」的同义说法，经向量路召回。
	if hits, err := h.Resources(ctx, SearchQuery{Text: "挖机"}); err != nil || len(hits) != 1 || hits[0].Item.Title != "单斗挖掘机 挖土" {
		t.Fatalf("常规检索应经向量路召回挖掘机：%v %+v", err, hits)
	}
	// 仅字面匹配：字段里没有「挖机」两个字，就不召回。
	if hits, err := h.Resources(ctx, SearchQuery{Text: "挖机", Exact: true}); err != nil || len(hits) != 0 {
		t.Fatalf("exact 不该混入语义近似：%v %+v", err, hits)
	}
	// 含「机」的全列出来、按编码升序，不受 Limit 截断。
	hits, err := h.Resources(ctx, SearchQuery{Text: "机", Exact: true, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 {
		t.Fatalf("含「机」的资源应全部交回（2 条）：%+v", hits)
	}
	for i, h2 := range hits {
		if h2.Via != ViaKeyword || (i > 0 && hits[i-1].Item.Code > h2.Item.Code) {
			t.Errorf("应标关键词命中并按编码升序：%+v", hits)
		}
	}
	// 过滤条件照常生效。
	if hits, err = h.Resources(ctx, SearchQuery{Text: "机", Exact: true, Kind: KindMaterial}); err != nil || len(hits) != 0 {
		t.Fatalf("材料类里没有含「机」的：%v %+v", err, hits)
	}
	// 定额同理；编码精确命中仍走短路。
	if qhits, err := h.Quotas(ctx, SearchQuery{Text: "挖", Exact: true}); err != nil || len(qhits) != 1 || qhits[0].Item.Code != "SZ-TF-001" {
		t.Fatalf("定额字面匹配「挖」应只有 1 条：%v %+v", err, qhits)
	}
	if qhits, err := h.Quotas(ctx, SearchQuery{Text: "JZ-QT-001", Exact: true}); err != nil || len(qhits) != 1 || qhits[0].Via != ViaCode {
		t.Fatalf("编码精确命中应短路：%v %+v", err, qhits)
	}
}
