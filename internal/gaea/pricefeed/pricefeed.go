// Package pricefeed 成本库价格源：定时/手动抓取固定网站的价格表，解析为
// 成本条目候选（名称/规格/单位/价格），与既有 cost_entries 匹配后产出
// 「新增 / 更新（含差额与环比）/ 无变化」待确认结果。遵循「无确认不写库」：
// 本包只抓取与解析，写回 cost_entries 由调用方在用户确认后执行。
package pricefeed

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/net/html"

	"github.com/gaea/gaea/internal/gaea/cost"
	fileenc "github.com/gaea/gaea/internal/gaea/fileutil/encoding"
)

const (
	fetchTimeout = 30 * time.Second
	maxBodyBytes = 4 << 20
)

// Source 是价格源订阅配置。
type Source struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	URL            string            `json:"url"`
	Parser         string            `json:"parser"` // sc_table：造价信息网价格表
	FrequencyHours int               `json:"frequencyHours"` // 0=仅手动
	Area           string            `json:"area"`           // 地区过滤（如 成都市区）
	Headers        map[string]string `json:"headers,omitempty"` // 自定义请求头（Cookie 等）
	Enabled        bool              `json:"enabled"`
	LastFetchAt    string            `json:"lastFetchAt"`
	CreatedAt      string            `json:"createdAt"`
}

// Row 是解析出的一行价格数据。
type Row struct {
	Title string
	Spec  string
	Unit  string
	Price float64
	Tax   string
}

// Candidate 是与既有成本条目匹配后的待确认结果。
type Candidate struct {
	Title         string  `json:"title"`
	Spec          string  `json:"spec"`
	Unit          string  `json:"unit"`
	Price         float64 `json:"price"`
	Tax           string  `json:"tax"`
	ExistingName  string  `json:"existingName"`
	ExistingPrice float64 `json:"existingPrice"`
	Status        string  `json:"status"` // 更新 / 无变化 / 新增
	Diff          float64 `json:"diff"`
	DiffPct       float64 `json:"diffPct"`
	Anomaly       bool    `json:"anomaly"`       // 偏离历史价格区间（价格异常）
	AnomalyReason string  `json:"anomalyReason"` // 如 "单期跳幅 +22.5%"
}

// Result 是一次抓取的完整结果。
type Result struct {
	SourceID   string
	SourceName string
	URL        string
	Period     string
	FetchedAt  string
	Rows       int
	Candidates []Candidate
}

// Fetch 抓取并解析价格源，与成本库匹配后返回待确认候选。
// store 可为 nil（跳过既有匹配，全部按「新增」处理）。
func Fetch(ctx context.Context, src Source, store *cost.Store) (*Result, error) {
	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	for k, v := range src.Headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: fetchTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("抓取失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("抓取失败 HTTP %d（站点可能需要浏览器验证或 Cookie）", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	text := decodeHTML(body, resp.Header.Get("Content-Type"))
	rows, err := parseTable(text)
	if err != nil {
		return nil, err
	}
	return &Result{
		SourceID:   src.ID,
		SourceName: src.Name,
		URL:        src.URL,
		Period:     periodFromURL(src.URL),
		FetchedAt:  time.Now().UTC().Format(time.RFC3339),
		Rows:       len(rows),
		Candidates: matchRows(rows, store),
	}, nil
}

// decodeHTML 按内容编码解码为 UTF-8（utf-8 直接可用，否则探测 GBK 等）。
func decodeHTML(body []byte, contentType string) string {
	if utf8.Valid(body) {
		return string(body)
	}
	if strings.Contains(strings.ToLower(contentType), "charset=utf-8") && !utf8.Valid(body) {
		// 页面声明 utf-8 但字节非法时仍尝试探测
	}
	enc, _ := fileenc.Detect(body)
	return string(fileenc.Decode(body, enc))
}

// parseTable 解析造价信息网价格表：找包含「名称/规格/单位」的表头行，
// 之后的数据行取名称/规格/单位/含税，价格取首个可解析的地区报价列。
func parseTable(text string) ([]Row, error) {
	doc, err := html.Parse(strings.NewReader(text))
	if err != nil {
		return nil, fmt.Errorf("HTML 解析失败: %w", err)
	}

	var rows [][]string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			rows = append(rows, rowCells(n))
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	if len(rows) == 0 {
		return nil, fmt.Errorf("未识别到价格表（页面结构变化或需要登录）")
	}

	// 找表头行并映射列。
	headerIdx := -1
	titleCol, specCol, unitCol, taxCol := -1, -1, -1, -1
	for i, r := range rows {
		if hasKeyword(r, "名称") && (hasKeyword(r, "规格") || hasKeyword(r, "单位")) {
			headerIdx = i
			titleCol = colByKeyword(r, "名称", "材料", "品名")
			specCol = colByKeyword(r, "规格", "型号")
			unitCol = colByKeyword(r, "单位")
			taxCol = colByKeyword(r, "含税", "税")
			break
		}
	}
	if headerIdx < 0 || titleCol < 0 {
		return nil, fmt.Errorf("未识别到价格表（缺少名称/规格/单位表头）")
	}

	priceStart := unitCol + 1
	if taxCol > unitCol {
		priceStart = taxCol + 1
	}

	var out []Row
	for _, r := range rows[headerIdx+1:] {
		if len(r) <= titleCol || strings.TrimSpace(cellAt(r, titleCol)) == "" {
			continue
		}
		if isAreaHeaderRow(r) {
			continue
		}
		price := 0.0
		for c := priceStart; c < len(r); c++ {
			if p, ok := parsePrice(cellAt(r, c)); ok {
				price = p
				break
			}
		}
		if price <= 0 {
			continue
		}
		out = append(out, Row{
			Title: strings.TrimSpace(cellAt(r, titleCol)),
			Spec:  trimCell(specCol, r),
			Unit:  trimCell(unitCol, r),
			Price: price,
			Tax:   trimCell(taxCol, r),
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("价格表无有效数据行")
	}
	return out, nil
}

func rowCells(n *html.Node) []string {
	var cells []string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && (c.Data == "td" || c.Data == "th") {
			cells = append(cells, strings.TrimSpace(nodeText(c)))
		}
	}
	return cells
}

func nodeText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		if x.Type == html.TextNode {
			b.WriteString(x.Data)
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

func hasKeyword(row []string, kw string) bool {
	for _, c := range row {
		if strings.Contains(strings.ToLower(c), strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

func colByKeyword(row []string, kws ...string) int {
	best := -1
	bestLen := 0
	for i, c := range row {
		low := strings.ToLower(c)
		for _, kw := range kws {
			if strings.Contains(low, strings.ToLower(kw)) && len(kw) > bestLen {
				best = i
				bestLen = len(kw)
			}
		}
	}
	return best
}

func cellAt(row []string, idx int) string {
	if idx >= 0 && idx < len(row) {
		return strings.TrimSpace(row[idx])
	}
	return ""
}

func trimCell(idx int, row []string) string {
	v := cellAt(row, idx)
	if v == "—" || v == "-" {
		return ""
	}
	return v
}

// isAreaHeaderRow 判断是否为地区表头行（成都市区/天府新区…，非数据行）。
func isAreaHeaderRow(row []string) bool {
	if len(row) < 3 {
		return false
	}
	areaish := 0
	for _, c := range row {
		if strings.Contains(c, "区") || strings.Contains(c, "市") || strings.Contains(c, "县") {
			areaish++
		}
	}
	return areaish >= len(row)/2
}

// parsePrice 解析价格文本：去掉 ￥/元/千分位，支持 "￥3181.00"、"3,200 元"。
func parsePrice(s string) (float64, bool) {
	if !strings.ContainsAny(s, "0123456789") {
		return 0, false
	}
	clean := strings.NewReplacer(",", "", "，", "", "￥", "", "¥", "", "元", "", " ", "", "\u00a0", "").Replace(strings.TrimSpace(s))
	if clean == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(clean, 64)
	if err != nil || v <= 0 {
		return 0, false
	}
	return round2(v), true
}

// matchRows 把解析行与成本库匹配，产出「更新/无变化/新增」候选。
func matchRows(rows []Row, store *cost.Store) []Candidate {
	byTitle := map[string]cost.Summary{}
	byName := map[string]cost.Summary{}
	if store != nil && store.Available() {
		for _, s := range store.List() {
			if t := strings.ToLower(strings.TrimSpace(s.Title)); t != "" {
				byTitle[t] = s
			}
			if n := strings.TrimSpace(s.Name); n != "" {
				byName[n] = s
			}
		}
	}

	out := make([]Candidate, 0, len(rows))
	for _, r := range rows {
		c := Candidate{Title: r.Title, Spec: r.Spec, Unit: r.Unit, Price: r.Price, Tax: r.Tax}
		if e, ok := byTitle[strings.ToLower(strings.TrimSpace(r.Title))]; ok {
			c.ExistingName = e.Name
			c.ExistingPrice = e.Price
		} else if e, ok := byName[cost.SlugName(r.Title)]; ok {
			c.ExistingName = e.Name
			c.ExistingPrice = e.Price
		}
		switch {
		case c.ExistingName == "":
			c.Status = "新增"
		case c.Price == c.ExistingPrice:
			c.Status = "无变化"
		default:
			c.Status = "更新"
			c.Diff = round2(c.Price - c.ExistingPrice)
			if c.ExistingPrice > 0 {
				c.DiffPct = round2(c.Diff / c.ExistingPrice * 100)
			}
		}
		out = append(out, c)
	}
	return out
}

func periodFromURL(raw string) string {
	if u, err := url.Parse(raw); err == nil {
		if p := u.Query().Get("period"); p != "" {
			return p
		}
	}
	return ""
}

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}

// anomalyJumpPct 是单期跳幅的异常阈值（±20%）。
const anomalyJumpPct = 20.0

// DetectAnomalies 对「更新」候选做价格异常识别：对比该条目最近的历史价格
// （无历史时用现价），单期跳幅超过 ±20% 标记异常并给出原因。新增/无变化
// 不判异常。history 返回某条目的价格历史（新→旧，由调用方提供）。
func DetectAnomalies(cands []Candidate, history func(name string) []History) []Candidate {
	if history == nil {
		return cands
	}
	for i := range cands {
		c := &cands[i]
		if c.Status != "更新" || c.ExistingName == "" {
			continue
		}
		prev := c.ExistingPrice
		// history 最新一条是最近发布价，作为跳幅基准（现价可能已过时）。
		if h := history(c.ExistingName); len(h) > 0 && h[0].Price > 0 {
			prev = h[0].Price
		}
		if prev <= 0 {
			continue
		}
		jump := (c.Price - prev) / prev * 100
		if jump >= anomalyJumpPct || jump <= -anomalyJumpPct {
			c.Anomaly = true
			c.AnomalyReason = fmt.Sprintf("单期跳幅 %+.1f%%（基准 ¥%s）", jump, strconv.FormatFloat(prev, 'f', 2, 64))
		}
	}
	return cands
}
