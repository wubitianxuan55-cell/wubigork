package builtin

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"

	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(csvParse{}) }

type csvParse struct{}

func (csvParse) Name() string { return "csv_parse" }

func (csvParse) Description() string {
	return "解析CSV文件，返回结构化JSON（表头+行数据+列统计）。支持自动编码检测（UTF-8/GBK）、分隔符自动检测，可限制行数和指定编码。"
}

func (csvParse) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "path":{"type":"string","description":"CSV文件路径"},
  "delimiter":{"type":"string","description":"列分隔符（不指定则自动检测：逗号/制表符/分号/竖线）"},
  "has_header":{"type":"boolean","description":"文件第一行是否为表头（默认true）"},
  "encoding":{"type":"string","description":"文件编码：utf-8、gbk、gb2312、gb18030（不指定则自动检测）"},
  "limit":{"type":"integer","description":"读取行数上限（默认1000）"}
},
"required":["path"]
}`)
}

func (csvParse) ReadOnly() bool { return true }

func (csvParse) CompactDescription() string     { return compactDesc["csv_parse"] }
func (csvParse) CompactSchema() json.RawMessage { return compactSchema["csv_parse"] }

type csvResult struct {
	Headers  []string           `json:"headers,omitempty"`
	Rows     [][]string         `json:"rows"`
	RowCount int                `json:"row_count"`
	ColCount int                `json:"col_count"`
	Stats    map[string]colStat `json:"stats,omitempty"`
	Encoding string             `json:"encoding"`
	Delim    string             `json:"delimiter"`
}

type colStat struct {
	Type    string   `json:"type"`
	Count   int      `json:"count"`
	NonNull int      `json:"non_null"`
	Min     *float64 `json:"min,omitempty"`
	Max     *float64 `json:"max,omitempty"`
	Mean    *float64 `json:"mean,omitempty"`
	StdDev  *float64 `json:"std_dev,omitempty"`
}

type csvParams struct {
	Path      string `json:"path"`
	Delimiter string `json:"delimiter,omitempty"`
	HasHeader *bool  `json:"has_header,omitempty"`
	Encoding  string `json:"encoding,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

func (csvParse) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p csvParams
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	if p.Path == "" {
		return "", fmt.Errorf("path 不能为空")
	}

	// 读取文件头用于检测
	raw, err := os.ReadFile(p.Path)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	if len(raw) == 0 {
		return `{"rows":[],"row_count":0,"col_count":0}`, nil
	}

	// 编码检测
	encName, reader := detectEncoding(bytes.NewReader(raw), p.Encoding)

	// 读取全部内容（通过编码转换）
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("解码失败: %w", err)
	}

	content := string(decoded)
	// 检测分隔符
	delim := detectDelimiter(content, p.Delimiter)

	// 解析 CSV
	r := csv.NewReader(strings.NewReader(content))
	r.Comma = delim
	r.LazyQuotes = true
	r.FieldsPerRecord = -1

	allRows, err := r.ReadAll()
	if err != nil {
		return "", fmt.Errorf("读取CSV失败: %w", err)
	}
	if len(allRows) == 0 {
		return fmt.Sprintf(`{"rows":[],"row_count":0,"col_count":0,"encoding":"%s","delimiter":"%s"}`, encName, string(delim)), nil
	}

	// 限制行数
	limit := p.Limit
	if limit <= 0 {
		limit = 1000
	}
	if len(allRows) > limit+1 { // +1 for header
		allRows = allRows[:limit+1]
	}

	hasHeader := true
	if p.HasHeader != nil {
		hasHeader = *p.HasHeader
	}

	var headers []string
	var dataRows [][]string
	if hasHeader {
		headers = allRows[0]
		dataRows = allRows[1:]
	} else {
		dataRows = allRows
	}

	colCount := 0
	for _, row := range dataRows {
		if len(row) > colCount {
			colCount = len(row)
		}
	}

	// 列统计
	stats := make(map[string]colStat)
	if colCount > 0 {
		for ci := 0; ci < colCount; ci++ {
			colName := fmt.Sprintf("col_%d", ci)
			if ci < len(headers) {
				colName = headers[ci]
			}
			cs := colStat{Count: len(dataRows)}
			var nums []float64
			for _, row := range dataRows {
				if ci < len(row) {
					val := strings.TrimSpace(row[ci])
					if val != "" {
						cs.NonNull++
						if n, err := strconv.ParseFloat(val, 64); err == nil {
							nums = append(nums, n)
						}
					}
				}
			}
			if len(nums) > 0 {
				cs.Type = "numeric"
				min, max := nums[0], nums[0]
				sum := 0.0
				for _, n := range nums {
					if n < min {
						min = n
					}
					if n > max {
						max = n
					}
					sum += n
				}
				mean := sum / float64(len(nums))
				cs.Min = &min
				cs.Max = &max
				cs.Mean = &mean
				if len(nums) > 1 {
					var sqSum float64
					for _, n := range nums {
						d := n - mean
						sqSum += d * d
					}
					std := math.Sqrt(sqSum / float64(len(nums)))
					cs.StdDev = &std
				}
			} else {
				cs.Type = "string"
			}
			stats[colName] = cs
		}
	}

	res := csvResult{
		Headers:  headers,
		Rows:     dataRows,
		RowCount: len(dataRows),
		ColCount: colCount,
		Stats:    stats,
		Encoding: encName,
		Delim:    string(delim),
	}
	out, _ := json.MarshalIndent(res, "", "  ")
	return string(out), nil
}

// detectEncoding 检测文件编码并返回解码 Reader
func detectEncoding(r io.Reader, hint string) (string, io.Reader) {
	head, _ := io.ReadAll(io.LimitReader(r, 4096))
	combined := append([]byte(nil), head...)

	if hint != "" {
		hint = strings.ToLower(hint)
		switch hint {
		case "gbk", "gb2312", "gb18030":
			return hint, transform.NewReader(io.MultiReader(bytes.NewReader(combined), r), simplifiedchinese.GBK.NewDecoder())
		default:
			return hint, io.MultiReader(bytes.NewReader(combined), r)
		}
	}

	// 自动检测：尝试 UTF-8
	if utf8.Valid(combined) {
		return "utf-8", io.MultiReader(bytes.NewReader(combined), r)
	}
	// 默认按 GBK 处理（中文环境常见）
	return "gbk", transform.NewReader(io.MultiReader(bytes.NewReader(combined), r), simplifiedchinese.GBK.NewDecoder())
}

// detectDelimiter 自动检测分隔符
func detectDelimiter(content string, hint string) rune {
	if hint != "" {
		runes := []rune(hint)
		if len(runes) == 1 {
			return runes[0]
		}
		return ','
	}

	// 取前 2000 字符统计候选分隔符
	sample := content
	if len(sample) > 2000 {
		sample = sample[:2000]
	}

	type cand struct {
		r   rune
		cnt int
	}
	candidates := []cand{{',', 0}, {'\t', 0}, {';', 0}, {'|', 0}}
	firstLines := strings.SplitN(sample, "\n", 5)

	for _, line := range firstLines {
		for i := range candidates {
			candidates[i].cnt += strings.Count(line, string(candidates[i].r))
		}
	}

	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.cnt > best.cnt {
			best = c
		}
	}
	if best.cnt > 0 {
		return best.r
	}
	return ','
}
