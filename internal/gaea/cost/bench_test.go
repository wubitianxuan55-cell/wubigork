package cost

// 刀D（v4.247）性能基线：cost_search 的固定成本。逐行相关子查询
// (SELECT COUNT(*) FROM cost_entry_components ...) 随库规模 N 次索引探测；
// BM25 每查询对关键词命中子集从零建倒排。

import (
	"fmt"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

// benchStore 预置 n 条条目（1/3 带组件明细）的 TempDir 隔离库（勿打真实用户库）。
func benchStore(b *testing.B, n int) (*Store, func()) {
	b.Helper()
	dir := b.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		b.Fatal("GetDatabase nil")
	}
	s := Open(gdb)
	closeDB := func() { db.CloseDatabase(dir) }
	for i := 0; i < n; i++ {
		e := Entry{
			Name:      fmt.Sprintf("entry-%04d", i),
			Title:     fmt.Sprintf("液压振动锤 %d 型 台班/材料 混合条目", i%7),
			Code:      fmt.Sprintf("A1-%02d-%04d", i%13, i),
			Category:  "材料",
			Unit:      "台班",
			Price:     float64(100 + i%900),
			Spec:      fmt.Sprintf("规格 %d：功率/吨位等长规格文本", i),
			Source:    "benchmark",
			Region:    "成都",
			PriceDate: "2026-08",
			PriceType: "市场价",
			Status:    "现行",
		}
		if i%3 == 0 {
			for k := 0; k < 4; k++ {
				e.Components = append(e.Components, Component{
					Kind: "材料", Title: fmt.Sprintf("comp-%d-%d", i, k),
					Quantity: 1, Price: 10, Amount: 10,
				})
			}
		}
		if err := s.Save(e); err != nil {
			b.Fatal(err)
		}
	}
	return s, closeDB
}

// BenchmarkSearch BM25 路径（有查询词：子串过滤+子集重建倒排+排序）。
func BenchmarkSearch(b *testing.B) {
	for _, n := range []int{500, 2000} {
		b.Run(fmt.Sprintf("entries=%d", n), func(b *testing.B) {
			s, closeDB := benchStore(b, n)
			defer closeDB()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if got := s.Search("液压振动锤 台班", "", ""); len(got) == 0 {
					b.Fatal("no hits")
				}
			}
		})
	}
}

// BenchmarkSearchNoQuery 空查询（List 主路径：纯 SQL + 逐行 COUNT 子查询）。
func BenchmarkSearchNoQuery(b *testing.B) {
	for _, n := range []int{500, 2000} {
		b.Run(fmt.Sprintf("entries=%d", n), func(b *testing.B) {
			s, closeDB := benchStore(b, n)
			defer closeDB()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if got := s.List(); len(got) != n {
					b.Fatalf("got %d want %d", len(got), n)
				}
			}
		})
	}
}
