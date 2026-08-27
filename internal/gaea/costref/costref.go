// Package costref 造价参考与复盘笔记（zaojia-database 蒸馏：案例指标 + 复盘经验）。
//
// 造价参考 = 对「已保存版本 / 已沉淀」测算项目的明细行做聚合：按科目（标题）或
// 一级分类给出 样本数/极值/分位数(P25/P75)/中位数/均值，供下次报价对标；
// 指标不落表，实时聚合避免双写。复盘笔记 = 结论/适用边界/风险提示/证据来源/
// 可信度/有效期/复核状态 + 引用计数，沉淀「判断」而非只堆数据。
package costref

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/costproject"
)

// Note 复盘笔记。
type Note struct {
	ID          int64
	Title       string
	Conclusion  string // 结论
	Boundary    string // 适用边界
	Risk        string // 风险提示
	Evidence    string // 证据来源（项目/版本/文件）
	Confidence  string // 可信度：高/中/低
	ValidUntil  string // 有效期至
	Status      string // 草稿 / 已确认
	Category    string // 成本分类（如 机械/材料/综合单价）
	ProjectType string // 项目类型
	Craft       string // 工艺类型
	RefCount    int    // 引用次数
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Indicator 造价参考指标（一组样本的价格统计）。
type Indicator struct {
	Key     string  // 科目标题 或 一级分类名
	Unit    string  // 样本最常见单位（按科目分组时才有意义）
	Samples int     // 样本数（参与统计的明细行数）
	Min     float64 // 极值（最低）
	Max     float64 // 极值（最高）
	Mean    float64 // 均值
	Median  float64 // 中位数（P50）
	P25     float64 // 下四分位
	P75     float64 // 上四分位
}

// Store 复盘笔记存储（Hephaestus.db）。
type Store struct {
	db *sql.DB
}

// Open 打开复盘笔记存储；gdb 为 nil 时返回不可用 store。
func Open(gdb *sql.DB) *Store { return &Store{db: gdb} }

// Available 报告存储是否可用。
func (s *Store) Available() bool { return s.db != nil }

// Save 新建/更新复盘笔记（id<=0 新建）。
func (s *Store) Save(n Note) (int64, error) {
	if s.db == nil {
		return 0, fmt.Errorf("cost ref store unavailable")
	}
	if strings.TrimSpace(n.Title) == "" {
		return 0, fmt.Errorf("复盘笔记需要标题")
	}
	if n.Status == "" {
		n.Status = "草稿"
	}
	if n.Confidence == "" {
		n.Confidence = "中"
	}
	now := time.Now().UTC()
	if n.ID <= 0 {
		res, err := s.db.Exec(`
INSERT INTO cost_review_notes(title, conclusion, boundary, risk, evidence, confidence, valid_until, status, category, project_type, craft, ref_count, created_at, updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			n.Title, n.Conclusion, n.Boundary, n.Risk, n.Evidence, n.Confidence,
			n.ValidUntil, n.Status, n.Category, n.ProjectType, n.Craft, n.RefCount,
			now.Format(time.RFC3339), now.Format(time.RFC3339))
		if err != nil {
			return 0, err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return 0, err
		}
		return id, nil
	}
	_, err := s.db.Exec(`
UPDATE cost_review_notes SET title=?, conclusion=?, boundary=?, risk=?, evidence=?, confidence=?, valid_until=?, status=?, category=?, project_type=?, craft=?, updated_at=?
WHERE id=?`,
		n.Title, n.Conclusion, n.Boundary, n.Risk, n.Evidence, n.Confidence,
		n.ValidUntil, n.Status, n.Category, n.ProjectType, n.Craft,
		now.Format(time.RFC3339), n.ID)
	return n.ID, err
}

// List 返回复盘笔记（按更新时间倒序），支持关键词（标题/结论/边界/风险/证据）与状态过滤。
func (s *Store) List(query, status string) []Note {
	if s.db == nil {
		return nil
	}
	conds := []string{"1=1"}
	var args []any
	if strings.TrimSpace(status) != "" && status != "all" {
		conds = append(conds, "status = ?")
		args = append(args, status)
	}
	q := strings.TrimSpace(query)
	if q != "" {
		conds = append(conds, `(title LIKE ? OR conclusion LIKE ? OR boundary LIKE ? OR risk LIKE ? OR evidence LIKE ?)`)
		like := "%" + q + "%"
		args = append(args, like, like, like, like, like)
	}
	rows, err := s.db.Query(`
SELECT id, title, conclusion, boundary, risk, evidence, confidence, valid_until, status, category, project_type, craft, ref_count, created_at, updated_at
FROM cost_review_notes WHERE `+strings.Join(conds, " AND ")+` ORDER BY updated_at DESC`, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Note
	for rows.Next() {
		var n Note
		var created, updated string
		if err := rows.Scan(&n.ID, &n.Title, &n.Conclusion, &n.Boundary, &n.Risk, &n.Evidence,
			&n.Confidence, &n.ValidUntil, &n.Status, &n.Category, &n.ProjectType, &n.Craft,
			&n.RefCount, &created, &updated); err != nil {
			continue
		}
		n.CreatedAt, _ = time.Parse(time.RFC3339, created)
		n.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
		out = append(out, n)
	}
	return out
}

// Delete 删除复盘笔记。
func (s *Store) Delete(id int64) error {
	if s.db == nil {
		return nil
	}
	_, err := s.db.Exec("DELETE FROM cost_review_notes WHERE id=?", id)
	return err
}

// BumpRef 引用次数 +1（供 agent/前端引用笔记时记录复用）。
func (s *Store) BumpRef(id int64) error {
	if s.db == nil {
		return nil
	}
	_, err := s.db.Exec("UPDATE cost_review_notes SET ref_count = ref_count + 1 WHERE id=?", id)
	return err
}

// ── 造价参考指标（纯函数，基于测算项目明细实时聚合）─────────────────────

// ComputeIndicators 按科目（title）或一级分类聚合项目明细行的价格统计。
// group 取值 "title"（默认）或 "category"；samples 为 nil 时返回 nil。
func ComputeIndicators(items []costproject.Item, group string) []Indicator {
	if len(items) == 0 {
		return nil
	}
	// 只统计有价明细（缺单价行不参与对标，对齐「缺单价标记」）。
	valid := items[:0]
	for _, i := range items {
		if i.Price > 0 && strings.TrimSpace(i.Title) != "" {
			valid = append(valid, i)
		}
	}
	if len(valid) == 0 {
		return nil
	}
	buckets := map[string][]float64{}
	units := map[string]map[string]int{}
	for _, i := range valid {
		var key string
		if group == "category" {
			key = firstCategory(i.CategoryPath)
			if key == "" {
				key = "未分类"
			}
		} else {
			key = strings.TrimSpace(i.Title)
		}
		if key == "" {
			continue
		}
		buckets[key] = append(buckets[key], i.Price)
		if group != "category" && strings.TrimSpace(i.Unit) != "" {
			if units[key] == nil {
				units[key] = map[string]int{}
			}
			units[key][strings.TrimSpace(i.Unit)]++
		}
	}
	out := make([]Indicator, 0, len(buckets))
	keys := make([]string, 0, len(buckets))
	for k := range buckets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		prices := buckets[k]
		sort.Float64s(prices)
		ind := Indicator{
			Key:     k,
			Samples: len(prices),
			Min:     prices[0],
			Max:     prices[len(prices)-1],
			Mean:    mean(prices),
			Median:  percentile(prices, 0.5),
			P25:     percentile(prices, 0.25),
			P75:     percentile(prices, 0.75),
		}
		if group != "category" {
			best := ""
			bestN := 0
			for u, n := range units[k] {
				if n > bestN {
					best, bestN = u, n
				}
			}
			ind.Unit = best
		}
		out = append(out, ind)
	}
	return out
}

func firstCategory(path string) string {
	for _, p := range strings.Split(path, "/") {
		if strings.TrimSpace(p) != "" {
			return strings.TrimSpace(p)
		}
	}
	return ""
}

func mean(v []float64) float64 {
	s := 0.0
	for _, x := range v {
		s += x
	}
	return s / float64(len(v))
}

// percentile 线性插值分位数（R-7，与 Excel PERCENTILE.INC 一致）。
func percentile(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return sorted[0]
	}
	rank := p * float64(n-1)
	lo := int(rank)
	hi := lo + 1
	if hi >= n {
		hi = n - 1
	}
	frac := rank - float64(lo)
	return sorted[lo] + frac*(sorted[hi]-sorted[lo])
}
