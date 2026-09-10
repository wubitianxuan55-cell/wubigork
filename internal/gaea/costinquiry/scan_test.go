package costinquiry

import (
	"strings"
	"testing"
	"time"
)

// findScan 按类型+标题取一条发现（断言辅助）。
func findScan(fs []ScanFinding, kind, title string) *ScanFinding {
	for i := range fs {
		if fs[i].Kind == kind && fs[i].Title == title {
			return &fs[i]
		}
	}
	return nil
}

// TestScanAnomalies 四类发现：离散（max/min 比值定级）/跳变（相邻期涨幅，
// 异常优先排序）/过期未标记（逐条不依赖分组）/陈旧（最新期数超一年）。
func TestScanAnomalies(t *testing.T) {
	s := newTestStore(t)
	// c25 组：两点价差 1.6 倍（离散·关注），相邻期 +60%（跳变·异常）。
	id1 := mustSave(t, s, Record{Title: "C25 混凝土", Price: 100, PriceDate: "2026-01"})
	id2 := mustSave(t, s, Record{Title: "c25混凝土", Price: 160, PriceDate: "2026-03"})
	// 单点旧期数：陈旧（2024-06 距今超过一年）。
	mustSave(t, s, Record{Title: "中砂", Price: 80, PriceDate: "2024-06"})
	// 有效期已过仍「现行」（Save 默认 Status=现行）：过期。
	mustSave(t, s, Record{Title: "木材", Price: 1200, ValidUntil: "2020-01-01"})
	// 无价与零价数据点：不参与任何检查。
	mustSave(t, s, Record{Title: "废料", Price: 0, PriceDate: "2026-08"})

	fs := s.ScanAnomalies()
	if len(fs) != 4 {
		t.Fatalf("应恰 4 条发现（离散+跳变+过期+陈旧），got %d: %+v", len(fs), fs)
	}
	// 异常优先排序：跳变（异常）排第一。
	if fs[0].Kind != "跳变" || fs[0].Severity != "异常" {
		t.Fatalf("首条应为跳变·异常: %+v", fs[0])
	}
	if fs[0].RefIDs == nil || len(fs[0].RefIDs) != 2 || fs[0].RefIDs[0] != id1 || fs[0].RefIDs[1] != id2 {
		t.Fatalf("跳变应指向相邻两点: %+v", fs[0])
	}
	// 离散：1.6 倍=关注，明细带倍数与区间。
	sp := findScan(fs, "离散", "c25混凝土")
	if sp == nil || sp.Severity != "关注" {
		t.Fatalf("离散发现缺失或定级错误: %+v", sp)
	}
	if sp.Detail == "" || !strings.Contains(sp.Detail, "1.6 倍") {
		t.Fatalf("离散明细应含倍数: %+v", sp)
	}
	// 过期：木材，逐条报，refIds 单条。
	ep := findScan(fs, "过期", "木材")
	if ep == nil || ep.Severity != "关注" || len(ep.RefIDs) != 1 {
		t.Fatalf("过期发现缺失: %+v", ep)
	}
	// 陈旧：中砂，明细含期数。
	st := findScan(fs, "陈旧", "中砂")
	if st == nil || !strings.Contains(st.Detail, "2024-06-01") {
		t.Fatalf("陈旧发现缺失或期数错: %+v", st)
	}
	// 废料（零价）不应出现在任何发现里。
	for _, f := range fs {
		if f.Title == "废料" {
			t.Fatalf("零价数据点不应产生发现: %+v", f)
		}
	}
}

// TestScanAnomalies_Clean 干净库：单点新价、无过期、跳变在阈值内 → 无发现。
func TestScanAnomalies_Clean(t *testing.T) {
	s := newTestStore(t)
	recent := time.Now().AddDate(0, -1, 0).Format("2006-01")
	mustSave(t, s, Record{Title: "水泥", Price: 100, PriceDate: recent})
	mustSave(t, s, Record{Title: "水泥", Price: 104, PriceDate: recent})
	// +4% 在 30% 跳变阈值内、1.04 倍在 1.5 离散阈值内、期数新鲜。
	if fs := s.ScanAnomalies(); len(fs) != 0 {
		t.Fatalf("干净库不应有发现: %+v", fs)
	}
	// 有效期未到的记录不算过期。
	mustSave(t, s, Record{Title: "碎石", Price: 60, ValidUntil: "2099-12-31"})
	if fs := s.ScanAnomalies(); len(fs) != 0 {
		t.Fatalf("未到期不应报过期: %+v", fs)
	}
}

// TestScanAnomalies_Empty 空库与不可用存储。
func TestScanAnomalies_Empty(t *testing.T) {
	s := newTestStore(t)
	if fs := s.ScanAnomalies(); fs != nil {
		t.Fatalf("空库应返回 nil: %+v", fs)
	}
	if fs := Open(nil).ScanAnomalies(); fs != nil {
		t.Fatalf("不可用存储应返回 nil: %+v", fs)
	}
}
