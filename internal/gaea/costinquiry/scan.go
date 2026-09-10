// Package costinquiry — 库级异常扫描（刀路池第 4 项）：询价库自身的
// 数据质量体检。与 SuggestAdjustments 的单点调差互补——那边对的是
// 「成本条目 vs 询价」，这边对的是「询价库内部自洽」。
// 只读；全部检查基于既有归一化/排序辅助（MatchTitle/sortableDate/
// parsePriceDate），不新建检测基建。
package costinquiry

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ScanFinding 一条库级扫描发现。
type ScanFinding struct {
	Kind     string  `json:"kind"`     // 离散/跳变/过期/陈旧
	Severity string  `json:"severity"` // 关注/异常（与调差建议同口径词表）
	Title    string  `json:"title"`    // 涉及的标题（组用归一化标题，单条用原标题）
	Detail   string  `json:"detail"`   // 人话描述（含关键数值，UI 直接展示）
	RefIDs   []int64 `json:"refIds"`   // 涉及记录 id（供前端定位复核）
}

// 扫描阈值（与 adjustLevel 的 5%/15% 分档各自独立：这里是库内自洽检查，
// 阈值放宽到「值得人工看一眼」的量级）。
const (
	spreadWatch  = 1.5 // 同标题 max/min ≥1.5 关注
	spreadAlarm  = 2.0 // ≥2.0 异常
	jumpWatchPct = 30  // 相邻期跳变 ≥30% 关注
	jumpAlarmPct = 50  // ≥50% 异常
	staleDays    = 365 // 最新期数超过一年未更新
)

// ScanAnomalies 库级异常扫描（只读）：
//  1. 离散——同标题数据点 max/min ≥1.5（同名不同规格混归一或录入有误）；
//  2. 跳变——同标题按期数排序后相邻两点涨幅 ≥30%（行情剧变或录入错误）；
//  3. 过期——valid_until 已过而状态仍为「现行」（该转已过期）；
//  4. 陈旧——标题最新可解析期数超过一年未更新（调差/预测可信度下降）。
//
// 库空或无发现返回 nil；Price<=0 与空标题数据点不参与。
func (s *Store) ScanAnomalies() []ScanFinding {
	if s.db == nil {
		return nil
	}
	rows, err := s.db.Query(`SELECT ` + recordCols + ` FROM cost_inquiry_records`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	byTitle := map[string][]Record{}
	var out []ScanFinding
	for rows.Next() {
		r, ok := scanRecord(rows.Scan)
		if !ok {
			continue
		}
		// ③ 过期未标记：逐条检查（不依赖分组，单条也能报）。
		if t, err := time.Parse("2006-01-02", r.ValidUntil); err == nil {
			if t.Before(time.Now()) && strings.TrimSpace(r.Status) == "现行" {
				out = append(out, ScanFinding{
					Kind:     "过期",
					Severity: "关注",
					Title:    r.Title,
					Detail:   fmt.Sprintf("有效期至 %s 已过，状态仍为「现行」，调差比价时会被当成有效价", r.ValidUntil),
					RefIDs:   []int64{r.ID},
				})
			}
		}
		if strings.TrimSpace(r.Title) == "" || r.Price <= 0 {
			continue
		}
		key := MatchTitle(r.Title)
		byTitle[key] = append(byTitle[key], r)
	}
	if len(byTitle) == 0 && len(out) == 0 {
		return nil
	}

	for key, recs := range byTitle {
		// ① 离散：max/min 比值。
		minP, maxP := recs[0].Price, recs[0].Price
		for _, r := range recs[1:] {
			if r.Price < minP {
				minP = r.Price
			}
			if r.Price > maxP {
				maxP = r.Price
			}
		}
		if minP > 0 && len(recs) >= 2 {
			if ratio := maxP / minP; ratio >= spreadWatch {
				severity := "关注"
				if ratio >= spreadAlarm {
					severity = "异常"
				}
				out = append(out, ScanFinding{
					Kind:     "离散",
					Severity: severity,
					Title:    key,
					Detail: fmt.Sprintf("%d 个数据点价差 %.1f 倍（最低 ¥%.2f ～ 最高 ¥%.2f）——同名可能不同规格或录入有误",
						len(recs), ratio, minP, maxP),
					RefIDs: recordIDs(recs),
				})
			}
		}

		// ② 跳变：按期数排序后相邻两点涨幅（取最大一处）。
		if len(recs) >= 2 {
			sorted := append([]Record(nil), recs...)
			sort.SliceStable(sorted, func(i, j int) bool {
				pi, pj := sortableDate(sorted[i].PriceDate), sortableDate(sorted[j].PriceDate)
				if pi != pj {
					return pi < pj
				}
				return sorted[i].ID < sorted[j].ID
			})
			worstJ, worstPct := -1, 0.0
			for j := 1; j < len(sorted); j++ {
				if sorted[j-1].Price <= 0 {
					continue
				}
				pct := (sorted[j].Price - sorted[j-1].Price) / sorted[j-1].Price * 100
				if pct >= jumpWatchPct && pct > worstPct {
					worstJ, worstPct = j, pct
				}
			}
			if worstJ > 0 {
				severity := "关注"
				if worstPct >= jumpAlarmPct {
					severity = "异常"
				}
				a, b := sorted[worstJ-1], sorted[worstJ]
				out = append(out, ScanFinding{
					Kind:     "跳变",
					Severity: severity,
					Title:    key,
					Detail: fmt.Sprintf("%s ¥%.2f → %s ¥%.2f，相邻期跳变 +%.0f%%——行情剧变或录入错误",
						dateOrDash(a.PriceDate), a.Price, dateOrDash(b.PriceDate), b.Price, worstPct),
					RefIDs: []int64{a.ID, b.ID},
				})
			}
		}

		// ④ 陈旧：最新可解析期数超过一年。
		var newestT time.Time
		newestOK := false
		for _, r := range recs {
			if t, ok := parsePriceDate(strings.TrimSpace(r.PriceDate)); ok {
				if !newestOK || t.After(newestT) {
					newestT, newestOK = t, true
				}
			}
		}
		if newestOK && time.Since(newestT) > staleDays*24*time.Hour {
			out = append(out, ScanFinding{
				Kind:     "陈旧",
				Severity: "关注",
				Title:    key,
				Detail: fmt.Sprintf("最新数据点期数 %s，超过一年未更新——调差与预测可信度下降",
					newestT.Format("2006-01-02")),
				RefIDs: recordIDs(recs),
			})
		}
	}

	if len(out) == 0 {
		return nil
	}
	// 异常在前；同类按标题（稳定可预期，前端测试也好锚定）。
	kindOrder := map[string]int{"离散": 0, "跳变": 1, "过期": 2, "陈旧": 3}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Severity != out[j].Severity {
			return out[i].Severity == "异常"
		}
		if kindOrder[out[i].Kind] != kindOrder[out[j].Kind] {
			return kindOrder[out[i].Kind] < kindOrder[out[j].Kind]
		}
		return out[i].Title < out[j].Title
	})
	return out
}

// recordIDs 收集记录 id 列表。
func recordIDs(recs []Record) []int64 {
	ids := make([]int64, 0, len(recs))
	for _, r := range recs {
		ids = append(ids, r.ID)
	}
	return ids
}

// dateOrDash 期数展示兜底（空显示 —）。
func dateOrDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}
