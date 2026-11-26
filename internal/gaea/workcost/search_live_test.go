package workcost

// search_live_test.go — 真实库同义词评测（默认跳过）：拿真实造价库的副本 + 本机 bge-m3 建索引，
// 对比「纯关键词（LIKE）」与「混合检索」在同一批同义说法上的召回。
//
// 跑法（三项都给才跑；库给**副本**目录，不动活库）：
//
//	GAEA_COST_EVAL_DIR=<含 Hephaestus.db 的目录> GAEA_COST_EVAL_BASE=http://127.0.0.1:8080 \
//	  go test ./internal/gaea/workcost/ -run TestLiveCostSearchEval -v -count=1
//
// 每个查询打印一行：关键词命中数、两者 top-8 里各命中几个期望条目、混合检索的 top-8（标 K=关键词 /
// S=语义）。没有标注数据，期望条目是照着库名人工点的同义词对应项——评测看的是「名称里不含查询词的
// 那些对应项，关键词路一个都搜不到时，向量路能不能召回」。

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/retrieval"
	"github.com/gaea/gaea/internal/gaea/vecindex"
)

// liveReranker 把 retrieval.Reranker 适配成检索器要的形状。
type liveReranker struct{ r *retrieval.Reranker }

func (l liveReranker) Rerank(ctx context.Context, query string, docs []string, topN int) ([]int, error) {
	scored, err := l.r.Rerank(ctx, query, docs, topN)
	if err != nil {
		return nil, err
	}
	out := make([]int, 0, len(scored))
	for _, s := range scored {
		out = append(out, s.Index)
	}
	return out, nil
}

type liveCase struct {
	kind   string
	query  string
	expect []string // 期望命中的编码（照着库名人工点的同义词对应项）
}

var liveCases = []liveCase{
	{IndexKindResource, "挖机", []string{"EXC", "EXC-2", "EXC-3", "EXC-4"}},
	{IndexKindResource, "翻斗车", []string{"DMP", "DMP-2", "DMP-3", "DMP-CD"}},
	{IndexKindResource, "铲车", []string{"LDR", "LDR-2", "LDR-3", "LDR-4"}},
	{IndexKindResource, "砼", []string{"C20", "C20-2", "C30", "CONC-P6", "MXR-CD"}},
	{IndexKindResource, "吊车", []string{"CRN", "CRN-2", "CRN-3"}},
	{IndexKindResource, "excavator", []string{"EXC", "EXC-2", "EXC-3", "EXC-4"}},
	{IndexKindResource, "dump truck", []string{"DMP", "DMP-2", "DMP-3", "DMP-CD"}},
	{IndexKindResource, "防渗膜", []string{"HDPE", "HDPE-2"}},
	{IndexKindQuota, "渣土外运", []string{"D-BJ21A", "D-70", "D-BJ17B"}},
	{IndexKindQuota, "办公室搭建", []string{"D-BJ10A", "D-BJ10B", "D-PT08"}},
	{IndexKindQuota, "基坑排水", []string{"D-BJ16B", "D-BJ16A"}},
	{IndexKindQuota, "扬尘治理", []string{"D-39", "D-39-2", "D-BJ13A", "D-PT19", "D-BJ26A"}},
	{IndexKindQuota, "洗车槽", []string{"D-38", "D-38-2", "D-PT12", "D-BJ08A", "D-WP03A"}},
	{IndexKindQuota, "污泥脱水", []string{"D-63", "D-67"}},
	{IndexKindQuota, "地面硬化", []string{"D-BJ05A", "D-BJ06A", "D-PT10"}},
	{IndexKindBill, "渣土外运", []string{"0410010071684", "04B101", "04B100", "04B1689"}},
	{IndexKindBill, "泥水分离设备", []string{"04B095", "04B096"}},
	{IndexKindBill, "工地围挡", []string{"TL43"}},
	{IndexKindBill, "喷淋降扬尘", []string{"TL39"}},
}

func liveHit[T any](hits []Hit[T], key func(T) string) (codes []string, marks []string) {
	for _, h := range hits {
		codes = append(codes, key(h.Item))
		mark := "S" // 语义：名称里不含查询词，向量路按意思召回
		if h.Via != ViaSemantic {
			mark = "K"
		}
		marks = append(marks, mark)
	}
	return codes, marks
}

func liveRecall(codes, expect []string) int {
	n := 0
	for _, e := range expect {
		for _, c := range codes {
			if c == e {
				n++
				break
			}
		}
	}
	return n
}

func TestLiveCostSearchEval(t *testing.T) {
	dir, base := os.Getenv("GAEA_COST_EVAL_DIR"), os.Getenv("GAEA_COST_EVAL_BASE")
	if dir == "" || base == "" {
		t.Skip("默认跳过：设 GAEA_COST_EVAL_DIR（库副本目录）与 GAEA_COST_EVAL_BASE（模型地址）才跑")
	}
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase 返回 nil（副本目录里没有 Hephaestus.db？）")
	}
	defer db.CloseDatabase(dir)
	s := Open(gdb)
	vs := vecindex.Open(gdb)
	defer vecindex.Forget(gdb)
	emb := retrieval.NewEmbedder(base, "bge-m3")
	rr := retrieval.New(base, "bge-reranker-v2-m3")

	// ── 建索引：计时，算吞吐 ──
	ctx := context.Background()
	var docs int
	var build time.Duration
	for _, kind := range IndexKinds {
		start := time.Now()
		st, err := vs.Apply(ctx, kind, "bge-m3", s.IndexSources()[kind], emb, 128)
		if err != nil {
			t.Fatalf("建索引 %s: %v", kind, err)
		}
		build += time.Since(start)
		docs += st.Total
		t.Logf("建索引 %-8s 文档 %4d 条  新嵌 %4d  耗时 %6.2fs", kind, st.Total, st.Embedded, time.Since(start).Seconds())
	}
	t.Logf("建索引合计 %d 条 / %.2fs ≈ %.0f 条/秒", docs, build.Seconds(), float64(docs)/build.Seconds())

	kw := NewSearcher(s, nil, nil, nil) // 纯关键词
	hy := NewSearcher(s, vs, func() SearchEmbedder { return emb }, func() SearchReranker { return liveReranker{r: rr} })
	vecOnly := NewSearcher(s, vs, func() SearchEmbedder { return emb }, nil) // 向量路但不精排（量精排的开销）

	var sumKW, sumHY, zeroKW int
	var tKW, tVec, tFull time.Duration
	fmt.Println("\n| 类目 | 查询词 | 关键词命中 | 关键词 top8 命中期望 | 混合 top8 命中期望 | 混合 top8（K=关键词 S=语义） |")
	fmt.Println("|---|---|---:|---:|---:|---|")
	for _, c := range liveCases {
		q := SearchQuery{Text: c.query, Limit: 8}
		var kwCodes, hyCodes, hyMarks []string
		var err error
		switch c.kind {
		case IndexKindQuota:
			a, e1 := kw.Quotas(ctx, q)
			b, e2 := hy.Quotas(ctx, q)
			err = e1
			if err == nil {
				err = e2
			}
			kwCodes, _ = liveHit(a, func(x Quota) string { return x.Code })
			hyCodes, hyMarks = liveHit(b, func(x Quota) string { return x.Code })
		case IndexKindResource:
			a, e1 := kw.Resources(ctx, q)
			b, e2 := hy.Resources(ctx, q)
			err = e1
			if err == nil {
				err = e2
			}
			kwCodes, _ = liveHit(a, func(x Resource) string { return x.Code })
			hyCodes, hyMarks = liveHit(b, func(x Resource) string { return x.Code })
		default:
			a, e1 := kw.BillItems(ctx, q)
			b, e2 := hy.BillItems(ctx, q)
			err = e1
			if err == nil {
				err = e2
			}
			kwCodes, _ = liveHit(a, func(x BillItem) string { return x.Code })
			hyCodes, hyMarks = liveHit(b, func(x BillItem) string { return x.Code })
		}
		if err != nil {
			t.Fatalf("%s %q: %v", c.kind, c.query, err)
		}
		// 延迟拆开量（同一查询各跑一遍）：纯关键词 / 向量路不精排 / 完整混合含精排
		q0 := SearchQuery{Text: c.query, Limit: 8}
		t0 := time.Now()
		switch c.kind {
		case IndexKindQuota:
			_, _ = kw.Quotas(ctx, q0)
		case IndexKindResource:
			_, _ = kw.Resources(ctx, q0)
		default:
			_, _ = kw.BillItems(ctx, q0)
		}
		tKW += time.Since(t0)
		t0 = time.Now()
		switch c.kind {
		case IndexKindQuota:
			_, _ = vecOnly.Quotas(ctx, q0)
		case IndexKindResource:
			_, _ = vecOnly.Resources(ctx, q0)
		default:
			_, _ = vecOnly.BillItems(ctx, q0)
		}
		tVec += time.Since(t0)
		t0 = time.Now()
		switch c.kind {
		case IndexKindQuota:
			_, _ = hy.Quotas(ctx, q0)
		case IndexKindResource:
			_, _ = hy.Resources(ctx, q0)
		default:
			_, _ = hy.BillItems(ctx, q0)
		}
		tFull += time.Since(t0)
		rk, rh := liveRecall(kwCodes, c.expect), liveRecall(hyCodes, c.expect)
		sumKW += rk
		sumHY += rh
		if rk == 0 && rh > 0 {
			zeroKW++
		}
		shown := make([]string, len(hyCodes))
		for i := range hyCodes {
			shown[i] = hyMarks[i] + ":" + hyCodes[i]
		}
		fmt.Printf("| %s | %s | %d | %d/%d | %d/%d | %s |\n", c.kind, c.query, len(kwCodes), rk, len(c.expect), rh, len(c.expect), strings.Join(shown, " "))
	}
	total := 0
	for _, c := range liveCases {
		total += len(c.expect)
	}
	fmt.Printf("\n合计：期望条目 %d 个；关键词 top8 命中 %d（%.0f%%）；混合 top8 命中 %d（%.0f%%）；关键词一条都对不上、混合能召回的查询 %d/%d\n",
		total, sumKW, 100*float64(sumKW)/float64(total), sumHY, 100*float64(sumHY)/float64(total), zeroKW, len(liveCases))
	n := float64(len(liveCases))
	fmt.Printf("延迟（单次查询平均）：纯关键词 %.0f 毫秒；向量路不精排 %.0f 毫秒（含查询词向量化）；完整混合含精排 %.0f 毫秒。库规模 %d 条 / 1024 维\n",
		float64(tKW.Microseconds())/1000/n, float64(tVec.Microseconds())/1000/n, float64(tFull.Microseconds())/1000/n, docs)
}
