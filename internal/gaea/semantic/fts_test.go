package semantic

// 阶段九 9.2 混合召回测试：FTS 路（MATCH/LIKE 降级/惰性重建）+ 三路融合排序
// （可解释权重复算+时序 tie-break+确定性）。

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func ftsTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, schema := range []string{`
CREATE TABLE IF NOT EXISTS semantic_vectors (
  kind       TEXT NOT NULL,
  id         TEXT NOT NULL,
  vec        TEXT NOT NULL DEFAULT '[]',
  doc        TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (kind, id)
);`, `
CREATE VIRTUAL TABLE IF NOT EXISTS semantic_fts USING fts5(
  doc,
  kind UNINDEXED,
  id UNINDEXED
);`} {
		if _, err := db.Exec(schema); err != nil {
			t.Fatal(err)
		}
	}
	return Open(db)
}

func seedFTS(t *testing.T, s *Store) {
	t.Helper()
	rows := []struct{ kind, id, doc, at string }{
		{"cost", "HP300 高频液压振动锤", "HP300 高频液压振动锤 打桩机械 台班 800 元", "2026-10-01T10:00:00+08:00"},
		{"knowledge", "振动锤选型要点", "振动锤选型考虑地质与桩径，锤击能量匹配。", "2026-10-02T10:00:00+08:00"},
		{"office", "项目-土方开挖方案", "土方开挖放坡 1:0.5，机械开挖人工清底。", "2026-10-03T10:00:00+08:00"},
	}
	for _, r := range rows {
		if _, err := s.db.Exec(`INSERT INTO semantic_vectors(kind,id,vec,doc,updated_at) VALUES(?,?,?,?,?)`,
			r.kind, r.id, "[]", r.doc, r.at); err != nil {
			t.Fatal(err)
		}
	}
}

func TestEnsureFTS_LazyRebuild(t *testing.T) {
	s := ftsTestStore(t)
	seedFTS(t, s)

	// 首次：向量 3 行 FTS 0 行 → 重建
	rebuilt, err := s.EnsureFTS()
	if err != nil || !rebuilt {
		t.Fatalf("首次应重建: rebuilt=%v err=%v", rebuilt, err)
	}
	// 二次：一致 → 不重建
	rebuilt, err = s.EnsureFTS()
	if err != nil || rebuilt {
		t.Fatalf("一致不应重建: rebuilt=%v err=%v", rebuilt, err)
	}
}

func TestFTSSearch_MatchAndLikeFallback(t *testing.T) {
	s := ftsTestStore(t)
	seedFTS(t, s)
	if _, err := s.EnsureFTS(); err != nil {
		t.Fatal(err)
	}

	// 精确词命中（FTS MATCH，英文/数字词 unicode61 可切）
	hits, err := s.FTSSearch("HP300", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].ID != "HP300 高频液压振动锤" {
		t.Fatalf("精确词应命中: %+v", hits)
	}

	// 中文查询：unicode61 不切 CJK，MATCH 零命中 → LIKE 降级仍可命中
	hits, err = s.FTSSearch("土方开挖", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("中文 LIKE 降级应命中")
	}
	found := false
	for _, h := range hits {
		if h.ID == "项目-土方开挖方案" {
			found = true
		}
		if h.UpdatedAt == "" {
			t.Fatalf("LIKE 路 updated_at 应带出: %+v", h)
		}
	}
	if !found {
		t.Fatalf("土方方案应在列: %+v", hits)
	}

	// 零命中查询：空集不报错
	hits, err = s.FTSSearch("量子纠缠", 10)
	if err != nil || len(hits) != 0 {
		t.Fatalf("零命中应空集: %+v err=%v", hits, err)
	}
}

func TestFuseOrdered_WeightsAndTieBreak(t *testing.T) {
	// 可解释性：final 用常量权重复算可复现
	in := []FuseInput{
		{ID: "向量单路", Kind: "k", VecScore: 0.9},
		{ID: "全文单路", Kind: "k", FTScore: 1.0},
		{ID: "名称命中", Kind: "k", NameHit: true},
		{ID: "三路齐中", Kind: "k", VecScore: 0.9, FTScore: 1.0, NameHit: true},
	}
	out := FuseOrdered(in)
	if out[0].ID != "三路齐中" {
		t.Fatalf("三路齐中应第一: %+v", out[0])
	}
	// 复算：final = 0.55*vec + 0.30*ft + 0.15*name
	want := 0.55*0.9 + 0.30*1.0 + 0.15*1.0
	if diff := out[0].Final - want; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("加权分复算不符: %v want %v", out[0].Final, want)
	}
	// 权重和=1
	if FuseWeightVec+FuseWeightFT+FuseWeightName != 1.0 {
		t.Fatalf("权重和应=1")
	}
}

func TestFuseOrdered_RecencyTieBreak(t *testing.T) {
	// 同分：updated_at 新者优先（RFC3339 字典序=时间序）
	in := []FuseInput{
		{ID: "旧条目", Kind: "k", VecScore: 0.8, UpdatedAtRFC3339: "2026-09-01T00:00:00+08:00"},
		{ID: "新条目", Kind: "k", VecScore: 0.8, UpdatedAtRFC3339: "2026-10-01T00:00:00+08:00"},
	}
	out := FuseOrdered(in)
	if out[0].ID != "新条目" {
		t.Fatalf("同分新者优先: %+v", out)
	}
}

func TestFuseSemantic_MergeAndNormalize(t *testing.T) {
	// 两路合并去重 + FTS 分按最大值归一 + 名称路判定
	vecHits := []Hit{
		{Kind: "cost", ID: "向量命中条目", Score: 0.7},
		{Kind: "cost", ID: "两路齐中", Score: 0.5},
	}
	ftHits := []FTSHit{
		{Kind: "cost", ID: "两路齐中", Score: 5, UpdatedAt: "2026-10-01T00:00:00+08:00"},
		{Kind: "cost", ID: "全文命中条目", Score: 2},
	}
	out := FuseSemantic("两路齐中", vecHits, ftHits, map[string]string{}, 10)
	if len(out) != 3 {
		t.Fatalf("去重后应 3 条（两路重复合一）: %+v", out)
	}
	var both *FusedHit
	for i := range out {
		if out[i].ID == "两路齐中" {
			both = &out[i]
		}
	}
	if both == nil || both.FTScore != 1.0 { // 5/5=归一满
		t.Fatalf("FTS 归一不符: %+v", both)
	}
	if !both.NameHit { // 名称含查询词
		t.Fatalf("名称路应命中: %+v", both)
	}
	if both.VecScore != 0.5 {
		t.Fatalf("向量分: %+v", both)
	}
}
