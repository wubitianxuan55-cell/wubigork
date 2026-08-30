// Package costinquiry — 询价飞轮(四源归一询价库)。
//
// 信息价/OCR报价/供应商比价/手动询价四源归一为「询价库数据点」(cost_inquiry_records 表),
// 支撑 v4.2「询价飞轮」三件事:数据点存储(规格/地区/期数/有效期)、到期预警
// (valid_until 临近或已过期)、调差建议(成本库条目 vs 最新询价数据点,差幅显著时提示)。
// 与 costref/cost 同模式:显式、可编辑、可检索;包内 Open 自建表(正式迁移由父代理
// 收编进 db/schema.go,本包不触碰 schema.go)。
package costinquiry

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/gaea/gaea/internal/gaea/cost"
)

// Record 询价库数据点(四源归一:信息价/OCR报价/供应商比价/手动询价)。
type Record struct {
	ID         int64     `json:"id"`
	Title      string    `json:"title"`
	Spec       string    `json:"spec"`
	Unit       string    `json:"unit"`
	Price      float64   `json:"price"`
	Source     string    `json:"source"`   // 信息价/OCR报价/供应商比价/手动询价(默认 手动询价)
	Supplier   string    `json:"supplier"` // 供应商/来源点名(信息价=期数名)
	Region     string    `json:"region"`
	PriceDate  string    `json:"priceDate"`  // 价格时间/期数(如 2026-08 / 2026年第2期)
	ValidUntil string    `json:"validUntil"` // 有效期至 YYYY-MM-DD(空=长期有效)
	Note       string    `json:"note"`
	Status     string    `json:"status"` // 现行/已过期(默认 现行)
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// AdjustSuggestion 调差建议:成本库条目 vs 最新询价数据点。
type AdjustSuggestion struct {
	EntryName    string  `json:"entryName"`
	EntryTitle   string  `json:"entryTitle"`
	EntryPrice   float64 `json:"entryPrice"`
	LatestPrice  float64 `json:"latestPrice"`
	LatestDate   string  `json:"latestDate"`
	LatestSource string  `json:"latestSource"`
	Diff         float64 `json:"diff"`
	DiffPct      float64 `json:"diffPct"`
	Unit         string  `json:"unit"`
	// Level 是差幅分级（v4.6 询价异常检测）：正常(<5%) / 关注(5-15%) /
	// 异常(>15%)。沿用五算对比的同口径阈值常量（coststage 偏差特征）。
	Level string `json:"level"`
	// PredictedNext 是该条目询价序列线性回归的下一期预测价（v4.6 价格预测；
	// 序列 <2 个数据点时返回 0 = 无可预测）。
	PredictedNext float64 `json:"predictedNext,omitempty"`
	// PredictionNote 预测置信说明（点太少/趋势方向）。
	PredictionNote string `json:"predictionNote,omitempty"`
}

// Store 询价库存储(Hephaestus.db)。
type Store struct {
	db *sql.DB
}

// recordCols 表全列(查询共用)。
const recordCols = `id, title, spec, unit, price, source, supplier, region, price_date, valid_until, note, status, created_at, updated_at`

// createTableSQL §3 建表 SQL(幂等,正式迁移由父代理收编 db/schema.go)。
const createTableSQL = `
CREATE TABLE IF NOT EXISTS cost_inquiry_records (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  title       TEXT NOT NULL,
  spec        TEXT NOT NULL DEFAULT '',
  unit        TEXT NOT NULL DEFAULT '',
  price       REAL NOT NULL,
  source      TEXT NOT NULL DEFAULT '手动询价',
  supplier    TEXT NOT NULL DEFAULT '',
  region      TEXT NOT NULL DEFAULT '',
  price_date  TEXT NOT NULL DEFAULT '',
  valid_until TEXT NOT NULL DEFAULT '',
  note        TEXT NOT NULL DEFAULT '',
  status      TEXT NOT NULL DEFAULT '现行',
  created_at  TEXT NOT NULL,
  updated_at  TEXT NOT NULL
)`

// createIndexSQL 标题索引(幂等)。
const createIndexSQL = `CREATE INDEX IF NOT EXISTS idx_cost_inquiry_title ON cost_inquiry_records(title)`

// Open 打开询价库;gdb 为 nil 时返回不可用 store(Available()=false)。
// 幂等执行建表 SQL + 标题索引(重复 Open 安全)。
func Open(gdb *sql.DB) *Store {
	s := &Store{db: gdb}
	if gdb != nil {
		_, _ = gdb.Exec(createTableSQL)
		_, _ = gdb.Exec(createIndexSQL)
	}
	return s
}

// Available 报告存储是否可用。
func (s *Store) Available() bool { return s.db != nil }

// Save 新建/更新一条询价数据点:
//   - id<=0 新建:自动时间戳,Source/Status 空值填默认(手动询价/现行);
//   - id>0 更新:保留 created_at,刷新 updated_at。
func (s *Store) Save(r Record) (int64, error) {
	if s.db == nil {
		return 0, fmt.Errorf("cost inquiry store unavailable")
	}
	if strings.TrimSpace(r.Title) == "" {
		return 0, fmt.Errorf("询价记录需要标题")
	}
	if r.Source == "" {
		r.Source = "手动询价"
	}
	if r.Status == "" {
		r.Status = "现行"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if r.ID <= 0 {
		res, err := s.db.Exec(`
INSERT INTO cost_inquiry_records(title, spec, unit, price, source, supplier, region, price_date, valid_until, note, status, created_at, updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			r.Title, r.Spec, r.Unit, r.Price, r.Source, r.Supplier, r.Region,
			r.PriceDate, r.ValidUntil, r.Note, r.Status, now, now)
		if err != nil {
			return 0, err
		}
		return res.LastInsertId()
	}
	_, err := s.db.Exec(`
UPDATE cost_inquiry_records SET title=?, spec=?, unit=?, price=?, source=?, supplier=?, region=?, price_date=?, valid_until=?, note=?, status=?, updated_at=?
WHERE id=?`,
		r.Title, r.Spec, r.Unit, r.Price, r.Source, r.Supplier, r.Region,
		r.PriceDate, r.ValidUntil, r.Note, r.Status, now, r.ID)
	return r.ID, err
}

// UpsertBySourceKey 按 (title, spec, source, supplier) 幂等写入（v4.6 OCR
// 报价单自动入询价库飞轮）：同源同标题的数据点更新（价格/期数/有效期刷新，
// created_at 保留首见时间），避免重复导入同一报价单产生重复数据点。返回
// 数据点 id。与手动编辑（按 id 更新）互不干扰。
func (s *Store) UpsertBySourceKey(r Record) (int64, error) {
	if s.db == nil {
		return 0, fmt.Errorf("cost inquiry store unavailable")
	}
	if strings.TrimSpace(r.Title) == "" {
		return 0, fmt.Errorf("询价记录需要标题")
	}
	if r.Source == "" {
		r.Source = "OCR报价"
	}
	if r.Status == "" {
		r.Status = "现行"
	}
	var existingID int64
	err := s.db.QueryRow(`
SELECT id FROM cost_inquiry_records
WHERE title=? AND spec=? AND source=? AND supplier=? AND status='现行'
ORDER BY id LIMIT 1`,
		r.Title, r.Spec, r.Source, r.Supplier).Scan(&existingID)
	switch {
	case err == nil:
		r.ID = existingID
		return s.Save(r) // 更新：created_at 保留（Save 更新路径语义）
	case errors.Is(err, sql.ErrNoRows):
		return s.Save(r)
	default:
		return 0, err
	}
}

// List 检索询价数据点:关键词匹配 title/spec/supplier/region/note(词间 AND、字段间 OR,
// 大小写不敏感),按 updated_at 倒序(同秒按 id 倒序兜底,保证确定性);limit<=0 默认 100。
// 关键词过滤在 Go 侧做(cost.Search 同款),规避 modernc/sqlite 对长 OR LIKE 链的怪癖。
func (s *Store) List(query string, limit int) []Record {
	if s.db == nil {
		return nil
	}
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.Query(`
SELECT ` + recordCols + `
FROM cost_inquiry_records ORDER BY updated_at DESC, id DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Record
	for rows.Next() {
		if r, ok := scanRecord(rows.Scan); ok {
			out = append(out, r)
		}
	}
	q := strings.ToLower(strings.TrimSpace(query))
	if q != "" {
		terms := strings.Fields(q)
		filtered := out[:0]
		for _, r := range out {
			hay := strings.ToLower(r.Title + "\x00" + r.Spec + "\x00" + r.Supplier +
				"\x00" + r.Region + "\x00" + r.Note)
			ok := true
			for _, term := range terms {
				if !strings.Contains(hay, term) {
					ok = false
					break
				}
			}
			if ok {
				filtered = append(filtered, r)
			}
		}
		out = filtered
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// Get 按 id 读取数据点;不存在返回错误。
func (s *Store) Get(id int64) (*Record, error) {
	if s.db == nil {
		return nil, fmt.Errorf("cost inquiry store unavailable")
	}
	r, ok := scanRecord(func(dest ...any) error {
		return s.db.QueryRow(`SELECT `+recordCols+` FROM cost_inquiry_records WHERE id=?`, id).Scan(dest...)
	})
	if !ok {
		return nil, fmt.Errorf("询价记录 %d 不存在", id)
	}
	return &r, nil
}

// Delete 删除数据点。
func (s *Store) Delete(id int64) error {
	if s.db == nil {
		return nil
	}
	_, err := s.db.Exec("DELETE FROM cost_inquiry_records WHERE id=?", id)
	return err
}

// ListExpiring 到期预警:valid_until 非空且 <= today+days 的记录
// (按 valid_until 升序,已过期的在最前)。days<=0 时只返回已过期(<=today)的。
// valid_until 为 YYYY-MM-DD,解析失败视为空(不参与到期判断);today 用本地时区取日期部分。
func (s *Store) ListExpiring(days int) []Record {
	if s.db == nil {
		return nil
	}
	now := time.Now()
	deadlineStr := now.AddDate(0, 0, days).Format("2006-01-02")
	if days <= 0 {
		deadlineStr = now.Format("2006-01-02")
	}
	deadline, _ := time.Parse("2006-01-02", deadlineStr) // UTC 零点,与下方解析同日对齐
	rows, err := s.db.Query(`
SELECT ` + recordCols + `
FROM cost_inquiry_records WHERE valid_until != '' ORDER BY valid_until ASC, id ASC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Record
	for rows.Next() {
		r, ok := scanRecord(rows.Scan)
		if !ok {
			continue
		}
		vu, err := time.Parse("2006-01-02", r.ValidUntil)
		if err != nil {
			continue // 非法日期跳过(视为空)
		}
		if vu.After(deadline) {
			continue
		}
		out = append(out, r)
	}
	return out
}

// SuggestAdjustments 调差建议:对 entries 逐条按标题归一化(matchTitle)匹配询价数据点。
// 每条条目取匹配数据点中 price_date 最新的一条(price_date 空视为最旧);
// DiffPct = (LatestPrice-EntryPrice)/EntryPrice*100(四舍五入保留 4 位小数,消除浮点噪声),
// 仅 |DiffPct| > 2 时产出建议,按 |DiffPct| 降序(同幅按 EntryName 升序兜底)。
// 库空、entries 为空或无命中条目时返回 nil。
func (s *Store) SuggestAdjustments(entries []cost.Summary) []AdjustSuggestion {
	if s.db == nil || len(entries) == 0 {
		return nil
	}
	rows, err := s.db.Query(`SELECT ` + recordCols + ` FROM cost_inquiry_records`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	// 按归一化标题分组(同一标题可能多条数据点)。
	byTitle := map[string][]Record{}
	for rows.Next() {
		if r, ok := scanRecord(rows.Scan); ok {
			if r.Price <= 0 {
				continue // 无价数据点不参与比价
			}
			key := MatchTitle(r.Title)
			byTitle[key] = append(byTitle[key], r)
		}
	}
	if len(byTitle) == 0 {
		return nil // 库空(无有效数据点)
	}
	var out []AdjustSuggestion
	for _, e := range entries {
		if strings.TrimSpace(e.Title) == "" || e.Price <= 0 {
			continue // 无标题/无单价条目无法比价
		}
		recs := byTitle[MatchTitle(e.Title)]
		if len(recs) == 0 {
			continue
		}
		latest := newestByDate(recs)
		diff := latest.Price - e.Price
		diffPct := round4(diff / e.Price * 100)
		if math.Abs(diffPct) <= 2 {
			continue // 差幅不显著(<=2% 不提示)
		}
		sug := AdjustSuggestion{
			EntryName:    e.Name,
			EntryTitle:   e.Title,
			EntryPrice:   e.Price,
			LatestPrice:  latest.Price,
			LatestDate:   latest.PriceDate,
			LatestSource: latest.Source,
			Diff:         diff,
			DiffPct:      diffPct,
			Unit:         e.Unit,
			Level:        adjustLevel(diffPct),
		}
		// v4.6 价格预测：同标题询价序列（按 price_date 旧→新）线性回归下期价。
		if next, _, ok := PredictNext(pricesByDate(recs)); ok && next > 0 {
			sug.PredictedNext = round4(next)
			sug.PredictionNote = "基于询价序列线性回归（数据点越多越可信）"
		}
		out = append(out, sug)
	}
	if len(out) == 0 {
		return nil
	}
	sort.Slice(out, func(i, j int) bool {
		ai, aj := math.Abs(out[i].DiffPct), math.Abs(out[j].DiffPct)
		if ai != aj {
			return ai > aj
		}
		return out[i].EntryName < out[j].EntryName
	})
	return out
}

// adjustLevel 差幅分级（v4.6 询价异常检测）：|diffPct| <5 正常、5-15 关注、
// >15 异常——与 coststage 偏差特征同口径，前端按级着色。
func adjustLevel(diffPct float64) string {
	switch {
	case math.Abs(diffPct) > 15:
		return "异常"
	case math.Abs(diffPct) >= 5:
		return "关注"
	default:
		return "正常"
	}
}

// pricesByDate 返回按 price_date 升序（旧→新）的价格序列；price_date 空视为
// 最旧，时间无法解析的按字符串序（同为兜底，仅影响预测方向精度不影响安全）。
func pricesByDate(recs []Record) []float64 {
	sorted := append([]Record(nil), recs...)
	sort.SliceStable(sorted, func(i, j int) bool {
		pi, pj := sortableDate(sorted[i].PriceDate), sortableDate(sorted[j].PriceDate)
		if pi != pj {
			return pi < pj
		}
		return sorted[i].ID < sorted[j].ID
	})
	out := make([]float64, 0, len(sorted))
	for _, r := range sorted {
		if r.Price > 0 {
			out = append(out, r.Price)
		}
	}
	return out
}

// sortableDate 把 price_date（如 2026-08 / 2026年第2期 / 2026-08-15）转成可排序
// 字符串：前缀取前 7 位数字（YYYY-MM），不足补零；解析失败返回 ""（排最前）。
func sortableDate(s string) string {
	d := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			d = d*10 + int(c-'0')
		} else {
			break
		}
	}
	if d == 0 {
		return ""
	}
	return fmt.Sprintf("%08d", d)
}

// PredictNext 线性回归预测序列的下一期价格（v4.6 价格预测）。
// prices 按时间序（旧→新）；x = 0..n-1 索引，最小二乘拟合 y=a+bx，
// 返回 next = a + b*n（下一索引）与斜率 b（每期变化，单位=价格）。
// n<2 时返回 (最后价, 0, false)——数据点太少无可预测，调用方不展示。
func PredictNext(prices []float64) (next, slope float64, ok bool) {
	n := len(prices)
	if n == 0 {
		return 0, 0, false
	}
	if n == 1 {
		return prices[0], 0, false
	}
	var sx, sy, sxx, sxy float64
	for i, p := range prices {
		x := float64(i)
		sx += x
		sy += p
		sxx += x * x
		sxy += x * p
	}
	denom := float64(n)*sxx - sx*sx
	if denom == 0 {
		return prices[n-1], 0, false
	}
	slope = (float64(n)*sxy - sx*sy) / denom
	intercept := (sy - slope*sx) / float64(n)
	next = intercept + slope*float64(n)
	if next < 0 {
		next = 0
	}
	return round4(next), round4(slope), true
}

// MatchTitle 标题归一化(纯函数,导出供测试与外部复用):
// 去首尾空白 → 转小写 → 去全角空格(U+3000) → 去括号及括号内内容(中/英文括号,
// 循环到无括号,支持嵌套) → 去所有空白字符。
func MatchTitle(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "\u3000", "")
	for parenRe.MatchString(s) {
		s = parenRe.ReplaceAllString(s, "")
	}
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}

// parenRe 匹配完整括号对(不嵌套,配合循环处理嵌套),中/英文括号各一分支。
var parenRe = regexp.MustCompile(`\([^()]*\)|（[^（）]*）`)

// newestByDate 取 price_date 最新的一条;price_date 空视为最旧。
func newestByDate(recs []Record) Record {
	best := recs[0]
	for _, r := range recs[1:] {
		if priceDateNewer(r.PriceDate, best.PriceDate) {
			best = r
		}
	}
	return best
}

// priceDateNewer 报告 a 是否比 b 新;空视为最旧(任何非空都比空新)。
func priceDateNewer(a, b string) bool {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	if a == "" {
		return false
	}
	if b == "" {
		return true
	}
	at, aok := parsePriceDate(a)
	bt, bok := parsePriceDate(b)
	if aok && bok {
		return at.After(bt)
	}
	if aok != bok {
		return aok // 可解析的视为更新
	}
	return a > b // 均不可解析时按字符串比较(YYYY 前缀期数天然有序)
}

// parsePriceDate 尝试解析价格期数:YYYY-MM-DD / YYYY-MM / YYYY。
func parsePriceDate(s string) (time.Time, bool) {
	for _, layout := range []string{"2006-01-02", "2006-01", "2006"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// round4 四舍五入保留 4 位小数(消除 (diff/entry*100) 的浮点噪声,如 2.0000000000000004→2)。
func round4(v float64) float64 {
	return math.Round(v*1e4) / 1e4
}

// scanRecord 由 Scan 函数填充 Record,时间列 RFC3339 字符串解析为 UTC time.Time。
func scanRecord(scan func(dest ...any) error) (Record, bool) {
	var r Record
	var created, updated string
	if err := scan(&r.ID, &r.Title, &r.Spec, &r.Unit, &r.Price, &r.Source,
		&r.Supplier, &r.Region, &r.PriceDate, &r.ValidUntil, &r.Note, &r.Status,
		&created, &updated); err != nil {
		return r, false
	}
	r.CreatedAt, _ = time.Parse(time.RFC3339, created)
	r.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return r, true
}
