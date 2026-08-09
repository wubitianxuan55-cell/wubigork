// Package crosslink 提供跨应用联动：从 xlsx 提取数据 → matplotlib 生成图表 PNG →
// 构建 docx/pptx 嵌入 spec（数据表格 + 图表）。数据源变化后重新导出即刷新，
// 实现「图表与数据保持同步」的联动语义。
package crosslink

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/xuri/excelize/v2"
)

const maxChartRows = 8

// ExtractChartData 从 xlsx 提取图表数据。
// range 形如 "A1:B6"（空 = 自动取前两列数据行）。
func ExtractChartData(path, sheet, rng string) (labels []string, values []float64, header []string, rows [][]string, err error) {
	f, err := excelize.OpenFile(path, excelize.Options{UnzipXMLSizeLimit: 1 << 30})
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("打开 xlsx 失败: %w", err)
	}
	defer f.Close()
	if sheet == "" {
		if list := f.GetSheetList(); len(list) > 0 {
			sheet = list[0]
		}
	}
	all, err := f.GetRows(sheet)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("读取工作表 %q 失败: %w", sheet, err)
	}
	if len(all) == 0 {
		return nil, nil, nil, nil, fmt.Errorf("工作表 %q 为空", sheet)
	}

	// 区域裁剪
	r1, c1, r2, c2 := 1, 1, len(all), 2
	if rng != "" {
		if r1, c1, r2, c2, err = parseRange(rng); err != nil {
			return nil, nil, nil, nil, err
		}
		if r2 > len(all) {
			r2 = len(all)
		}
	}
	if c2 > 2 {
		c2 = 2
	}
	slice := func(row []string, c int) string {
		if c-1 < len(row) {
			return strings.TrimSpace(row[c-1])
		}
		return ""
	}

	for i := r1; i <= r2; i++ {
		if i-1 >= len(all) {
			break
		}
		row := all[i-1]
		vals := make([]string, 0, c2-c1+1)
		for c := c1; c <= c2; c++ {
			vals = append(vals, slice(row, c))
		}
		rows = append(rows, vals)
	}
	if len(rows) == 0 {
		return nil, nil, nil, nil, fmt.Errorf("数据区域为空")
	}

	var dataRows [][]string
	if rng == "" {
		header = rows[0]
		dataRows = rows[1:]
	} else {
		// 显式区域：全部视为数据行（表头不强制）
		dataRows = rows
	}
	if len(dataRows) > maxChartRows {
		dataRows = dataRows[:maxChartRows]
	}
	labels = make([]string, 0, len(dataRows))
	values = make([]float64, 0, len(dataRows))
	for _, r := range dataRows {
		lbl := ""
		if len(r) > 0 {
			lbl = r[0]
		}
		if lbl == "" {
			lbl = strconv.Itoa(len(labels) + 1)
		}
		v := 0.0
		if len(r) > 1 {
			clean := strings.ReplaceAll(strings.ReplaceAll(r[1], ",", ""), " ", "")
			if parsed, e := strconv.ParseFloat(clean, 64); e == nil {
				v = parsed
			}
		}
		labels = append(labels, lbl)
		values = append(values, v)
	}
	if len(values) == 0 {
		return nil, nil, nil, nil, fmt.Errorf("数据区域没有数值列")
	}
	return labels, values, header, rows, nil
}

func parseRange(r string) (r1, c1, r2, c2 int, err error) {
	parts := strings.Split(r, ":")
	if len(parts) != 2 {
		return 0, 0, 0, 0, fmt.Errorf("区域格式应为 A1:B6，得到 %q", r)
	}
	c1, r1, err = excelize.CellNameToCoordinates(parts[0])
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("区域起点无效 %q", parts[0])
	}
	c2, r2, err = excelize.CellNameToCoordinates(parts[1])
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("区域终点无效 %q", parts[1])
	}
	return r1, c1, r2, c2, nil
}

// GenerateChartPNG 用 matplotlib 生成图表 PNG（stdin 传参，无窗口）。
func GenerateChartPNG(labels []string, values []float64, chartType, title, output string) error {
	python := findPython()
	if python == "" {
		return fmt.Errorf("未找到 Python（需要安装 Python 和 matplotlib）")
	}
	if chartType == "" {
		chartType = "bar"
	}
	input, _ := json.Marshal(map[string]interface{}{
		"labels": labels, "values": values, "chart_type": chartType,
		"title": title, "output": output,
	})
	if dir := dirOf(output); dir != "" {
		os.MkdirAll(dir, 0o755)
	}
	cmd := exec.Command(python, "-c", chartScript)
	hideWindow(cmd)
	cmd.Stdin = bytes.NewReader(input)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("图表生成失败: %v（%s）", err, truncate(stderr.String(), 300))
	}
	if _, err := os.Stat(output); err != nil {
		return fmt.Errorf("图表文件未生成: %w", err)
	}
	return nil
}

func findPython() string {
	cands := []string{"python", "python3"}
	if runtime.GOOS != "windows" {
		cands = []string{"python3", "python"}
	}
	for _, c := range cands {
		if _, err := exec.LookPath(c); err == nil {
			return c
		}
	}
	return ""
}

func hideWindow(cmd *exec.Cmd) {
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
}

func dirOf(p string) string {
	i := strings.LastIndexAny(p, `/\`)
	if i < 0 {
		return ""
	}
	return p[:i]
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// BuildDocxSpec 构建 docx 嵌入 spec：标题 + 图表图片 + 数据表格。
func BuildDocxSpec(title, pngPath string, header []string, rows [][]string) map[string]interface{} {
	blocks := []map[string]interface{}{
		{"type": "heading1", "text": title},
		{"type": "paragraph", "text": "数据图表（由 Excel 数据自动生成，数据源更新后重新导出即同步）"},
		{"type": "image", "path": pngPath},
	}
	if len(header) > 0 {
		data := [][]string{}
		for _, r := range rows[1:] {
			data = append(data, r)
		}
		blocks = append(blocks, map[string]interface{}{"type": "table", "header": header, "rows": data})
	}
	return map[string]interface{}{
		"title": title, "cover": true, "toc": false,
		"header": "", "footer": "第 {page} 页", "font": "微软雅黑", "template": "报告",
		"blocks": blocks,
	}
}

// BuildPptxSpec 构建 pptx 嵌入 spec：图表页 + 数据明细页。
func BuildPptxSpec(title, pngPath string, header []string, rows [][]string) map[string]interface{} {
	var dataPoints []string
	for i, r := range rows {
		if i == 0 && len(header) > 0 {
			dataPoints = append(dataPoints, "表头："+strings.Join(r, " ｜ "))
			continue
		}
		dataPoints = append(dataPoints, strings.Join(r, " ｜ "))
		if len(dataPoints) >= 12 {
			break
		}
	}
	return map[string]interface{}{
		"title": title, "subtitle": "数据图表（基于 Excel 数据）", "author": "gaea",
		"slides": []map[string]interface{}{
			{"title": "图表", "image": pngPath},
			{"title": "数据明细", "points": dataPoints},
		},
	}
}

var chartScript = `
import json, sys, os, logging, warnings
warnings.filterwarnings("ignore")
logging.getLogger("matplotlib").setLevel(logging.ERROR)
try:
    import matplotlib
    matplotlib.use("Agg")
    import matplotlib.pyplot as plt
except ImportError:
    print(json.dumps({"ok": False, "error": "matplotlib not installed"}))
    sys.exit(1)
for font in ["SimHei", "Microsoft YaHei", "WenQuanYi Micro Hei", "Noto Sans CJK SC", "Arial Unicode MS"]:
    try:
        plt.rcParams["font.sans-serif"] = [font]
        plt.rcParams["axes.unicode_minus"] = False
        break
    except Exception:
        continue
params = json.loads(sys.stdin.buffer.read().decode("utf-8"))
labels = params.get("labels", [])
values = params.get("values", [])
ctype = params.get("chart_type", "bar")
title = params.get("title", "")
output = params.get("output", "chart.png")
fig, ax = plt.subplots(figsize=(10, 6))
if ctype == "pie":
    ax.pie(values, labels=labels, autopct="%1.1f%%", startangle=90)
    ax.axis("equal")
elif ctype == "line":
    ax.plot(labels, values, marker="o", linewidth=2, color="#4A90D9", markersize=6)
    for i, v in enumerate(values):
        ax.text(i, v, str(v), ha="center", va="bottom", fontsize=9)
elif ctype == "scatter":
    ax.scatter(range(len(values)), values, color="#4A90D9", s=60)
    ax.set_xticks(range(len(labels)))
    ax.set_xticklabels(labels)
else:
    ax.bar(labels, values, color="#4A90D9", edgecolor="white")
    for i, v in enumerate(values):
        ax.text(i, v + max(values) * 0.01, str(v), ha="center", fontsize=9)
if title:
    ax.set_title(title)
plt.tight_layout()
plt.savefig(output, dpi=150, bbox_inches="tight")
plt.close()
print(json.dumps({"ok": True, "output": output}))
`
