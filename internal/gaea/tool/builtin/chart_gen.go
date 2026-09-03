package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(chartGen{}) }

type chartGen struct{}

func (chartGen) Name() string { return "chart_gen" }

func (chartGen) Description() string {
	return "生成统计图表：单系列支持 bar 柱状图、line 折线图、pie 饼图、scatter 散点图、hbar 横向条形图、area 面积图、donut 环形图；多系列对比（多年份/多分组/多指标）传 series 数组，配合 grouped_bar 分组柱状、stacked_bar 堆叠柱状、bar 并列柱状、line 多折线。使用 Python matplotlib 生成 PNG/SVG 图片，通常几秒。需要系统中安装 Python 和 matplotlib。"
}

func (chartGen) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "labels":{"type":"array","items":{"type":"string"},"description":"类别标签列表（多系列时作为共用类别轴）"},
  "values":{"type":"array","items":{"type":"number"},"description":"单系列数据值（与 labels 等长；传 series 时可省略）"},
  "series":{"type":"array","description":"多系列数据（对比场景）：每组 {name, values}，values 与 labels 等长；配合 grouped_bar/stacked_bar/bar/line 使用","items":{"type":"object","properties":{"name":{"type":"string","description":"系列名（图例显示）"},"values":{"type":"array","items":{"type":"number"},"description":"该系列数据值，与 labels 等长"}},"required":["values"]}},
  "chart_type":{"type":"string","description":"图表类型：bar（柱状）、line（折线）、pie（饼图）、scatter（散点）、hbar（横向条形）、area（面积）、donut（环形）、grouped_bar（分组柱状，多系列）、stacked_bar（堆叠柱状，多系列）","default":"bar"},
  "title":{"type":"string","description":"图表标题"},
  "output":{"type":"string","description":"输出图片路径（.png 或 .svg）","default":"chart.png"},
  "xlabel":{"type":"string","description":"X轴标签"},
  "ylabel":{"type":"string","description":"Y轴标签"}
},
"required":["labels"]
}`)
}

func (chartGen) ReadOnly() bool { return false }

func (chartGen) CompactDescription() string     { return compactDesc["chart_gen"] }
func (chartGen) CompactSchema() json.RawMessage { return compactSchema["chart_gen"] }

type chartSeriesInput struct {
	Name   string    `json:"name,omitempty"`
	Values []float64 `json:"values"`
}

type chartInput struct {
	Labels    []string           `json:"labels"`
	Values    []float64          `json:"values"`
	Series    []chartSeriesInput `json:"series,omitempty"`
	ChartType string             `json:"chart_type,omitempty"`
	Title     string             `json:"title,omitempty"`
	Output    string             `json:"output,omitempty"`
	XLabel    string             `json:"xlabel,omitempty"`
	YLabel    string             `json:"ylabel,omitempty"`
}

// chartSeriesTypes 允许携带 series 多系列的类型；grouped_bar/stacked_bar 必须
// 多系列，bar/line 携带 series 时升级为并列柱状/多折线。
var chartSeriesTypes = map[string]bool{"grouped_bar": true, "stacked_bar": true, "bar": true, "line": true}

// chartSingleTypes 单系列（labels+values）合法类型白名单。
var chartSingleTypes = map[string]bool{"bar": true, "line": true, "pie": true, "scatter": true, "hbar": true, "area": true, "donut": true}

// parseChartInput 解析并校验参数。校验规则独立成纯函数供测试覆盖（真实渲染
// 依赖 Python+matplotlib，走独立冒烟测试）。
func parseChartInput(raw json.RawMessage) (chartInput, error) {
	var p chartInput
	if err := json.Unmarshal(raw, &p); err != nil {
		return p, fmt.Errorf("参数无效: %w", err)
	}
	if len(p.Labels) == 0 {
		return p, fmt.Errorf("labels 不能为空")
	}
	if p.ChartType == "" {
		p.ChartType = "bar"
	}
	multi := len(p.Series) > 0
	if multi && !chartSeriesTypes[p.ChartType] {
		return p, fmt.Errorf("chart_type=%s 不支持 series 多系列（请用 grouped_bar/stacked_bar/bar/line）", p.ChartType)
	}
	if multi {
		for i, s := range p.Series {
			if len(s.Values) != len(p.Labels) {
				return p, fmt.Errorf("series[%d] 的 values 长度（%d）与 labels（%d）不一致", i, len(s.Values), len(p.Labels))
			}
		}
		return p, nil
	}
	if p.ChartType == "grouped_bar" || p.ChartType == "stacked_bar" {
		return p, fmt.Errorf("chart_type=%s 需要 series 参数提供多系列数据", p.ChartType)
	}
	if !chartSingleTypes[p.ChartType] {
		return p, fmt.Errorf("未知 chart_type=%s（支持 bar/line/pie/scatter/hbar/area/donut/grouped_bar/stacked_bar）", p.ChartType)
	}
	if len(p.Values) == 0 {
		return p, fmt.Errorf("values 不能为空")
	}
	if len(p.Values) != len(p.Labels) {
		return p, fmt.Errorf("labels 和 values 长度不一致")
	}
	return p, nil
}

var chartScript = `
import json, sys, os
import logging, warnings
warnings.filterwarnings("ignore")
logging.getLogger("matplotlib").setLevel(logging.ERROR)
logging.getLogger("matplotlib.font_manager").setLevel(logging.ERROR)
try:
    import numpy as np
    import matplotlib
    matplotlib.use('Agg')
    import matplotlib.pyplot as plt
except ImportError:
    print(json.dumps({"ok": False, "error": "matplotlib not installed. Run: pip install matplotlib"}))
    sys.exit(1)

# 尝试加载中文字体
for font in ['Microsoft YaHei', 'SimHei', 'WenQuanYi Micro Hei', 'Noto Sans CJK SC', 'PingFang SC', 'Arial Unicode MS']:
    try:
        plt.rcParams['font.sans-serif'] = [font]
        plt.rcParams['axes.unicode_minus'] = False
        break
    except:
        continue

params = json.loads(sys.stdin.buffer.read().decode('utf-8'))
labels = params.get('labels', [])
values = params.get('values', [])
series = params.get('series', [])
ctype = params.get('chart_type', 'bar')
title = params.get('title', '')
output = params.get('output', 'chart.png')
xlabel = params.get('xlabel', '')
ylabel = params.get('ylabel', '')

PALETTE = ['#4C8BF5', '#F6A609', '#34A853', '#EA4335', '#9B59B6', '#00BCD4', '#FF7043', '#8D6E63', '#7986CB', '#66BB6A']

fig, ax = plt.subplots(figsize=(10, 6))
multi = len(series) > 0
is_pieish = ctype in ('pie', 'donut')

if not is_pieish:
    ax.spines['top'].set_visible(False)
    ax.spines['right'].set_visible(False)
    ax.set_axisbelow(True)
    ax.grid(axis='y', alpha=0.25, linewidth=0.6)

def legend(ax):
    ax.legend(frameon=False, fontsize=9, loc='best')

if multi and ctype in ('grouped_bar', 'stacked_bar', 'bar'):
    n = len(series)
    x = np.arange(len(labels))
    if ctype == 'stacked_bar':
        bottom = np.zeros(len(labels))
        for i, s in enumerate(series):
            vals = np.array(s.get('values'), dtype=float)
            ax.bar(x, vals, bottom=bottom, width=0.6, color=PALETTE[i % len(PALETTE)],
                   label=s.get('name') or ('S%d' % (i + 1)), edgecolor='white')
            bottom += vals
    else:
        w = 0.8 / n
        for i, s in enumerate(series):
            offs = x - 0.4 + w * (i + 0.5)
            ax.bar(offs, s.get('values'), width=w * 0.92, color=PALETTE[i % len(PALETTE)],
                   label=s.get('name') or ('S%d' % (i + 1)), edgecolor='white')
    ax.set_xticks(x)
    ax.set_xticklabels(labels, rotation=45, ha='right')
    legend(ax)
elif multi and ctype == 'line':
    x = np.arange(len(labels))
    for i, s in enumerate(series):
        ax.plot(x, s.get('values'), marker='o', linewidth=2, markersize=5,
                color=PALETTE[i % len(PALETTE)], label=s.get('name') or ('S%d' % (i + 1)))
    ax.set_xticks(x)
    ax.set_xticklabels(labels, rotation=45, ha='right')
    legend(ax)
elif ctype == 'bar':
    ax.bar(labels, values, color=PALETTE[0], edgecolor='white')
    for i, v in enumerate(values):
        ax.text(i, v + max(values) * 0.01, str(v), ha='center', fontsize=9)
    ax.set_xticks(range(len(labels)))
    ax.set_xticklabels(labels, rotation=45, ha='right')
elif ctype == 'line':
    ax.plot(range(len(labels)), values, marker='o', linewidth=2, color=PALETTE[0], markersize=6)
    for i, v in enumerate(values):
        ax.text(i, v, str(v), ha='center', va='bottom', fontsize=9)
    ax.set_xticks(range(len(labels)))
    ax.set_xticklabels(labels, rotation=45, ha='right')
elif ctype == 'pie':
    colors = [PALETTE[i % len(PALETTE)] for i in range(len(values))]
    wedges, texts, autotexts = ax.pie(values, labels=labels, autopct='%1.1f%%',
                                      colors=colors, startangle=90, counterclock=False)
    for t in autotexts:
        t.set_fontsize(9)
elif ctype == 'donut':
    colors = [PALETTE[i % len(PALETTE)] for i in range(len(values))]
    wedges, texts, autotexts = ax.pie(values, labels=labels, autopct='%1.1f%%',
                                      colors=colors, startangle=90, counterclock=False,
                                      wedgeprops={'width': 0.42, 'edgecolor': 'white'})
    for t in autotexts:
        t.set_fontsize(9)
elif ctype == 'hbar':
    y = range(len(labels))
    ax.barh(y, values, color=PALETTE[0], edgecolor='white')
    ax.set_yticks(list(y))
    ax.set_yticklabels(labels)
    ax.invert_yaxis()
    ax.grid(axis='y', visible=False)
    ax.grid(axis='x', alpha=0.25, linewidth=0.6)
    for i, v in enumerate(values):
        ax.text(v + max(values) * 0.01, i, str(v), va='center', fontsize=9)
elif ctype == 'area':
    x = range(len(labels))
    ax.plot(x, values, marker='o', linewidth=2, color=PALETTE[0], markersize=5)
    ax.fill_between(x, values, color=PALETTE[0], alpha=0.18)
    ax.set_xticks(list(x))
    ax.set_xticklabels(labels, rotation=45, ha='right')
elif ctype == 'scatter':
    x = list(range(len(values)))
    ax.scatter(x, values, color=PALETTE[0], s=60, alpha=0.7)
    if labels:
        ax.set_xticks(x)
        ax.set_xticklabels(labels, rotation=45, ha='right')

if title:
    ax.set_title(title, fontsize=14, fontweight='bold')
if xlabel:
    ax.set_xlabel(xlabel)
if ylabel:
    ax.set_ylabel(ylabel)

plt.tight_layout()
plt.savefig(output, dpi=150, bbox_inches='tight')
plt.close()

size_bytes = os.path.getsize(output)
print(json.dumps({"ok": True, "output": output, "size_bytes": size_bytes}))
`

func (chartGen) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	p, err := parseChartInput(args)
	if err != nil {
		return "", err
	}
	if p.Output == "" {
		p.Output = "chart.png"
	}
	// 确保输出目录存在
	if dir := filepath.Dir(p.Output); dir != "." {
		os.MkdirAll(dir, 0755)
	}

	// 查找 Python：Windows 优先 python（python3 常被商店别名劫持）
	candidates := []string{"python", "python3"}
	if runtime.GOOS != "windows" {
		candidates = []string{"python3", "python"}
	}
	python, err := lookPathFirst(candidates)
	if err != nil {
		return "", fmt.Errorf("未找到 Python（需要安装 Python 和 matplotlib）")
	}

	input := map[string]interface{}{
		"labels":     p.Labels,
		"values":     p.Values,
		"series":     p.Series,
		"chart_type": p.ChartType,
		"title":      p.Title,
		"output":     p.Output,
		"xlabel":     p.XLabel,
		"ylabel":     p.YLabel,
	}
	inputJSON, _ := json.Marshal(input)

	cmd := exec.CommandContext(ctx, python, "-c", chartScript)
	hideBashWindow(cmd) // Windows: 防止弹出 cmd 黑框
	cmd.Stdin = strings.NewReader(string(inputJSON))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Python 执行失败: %w\n输出: %s", err, string(output))
	}

	var result struct {
		OK        bool   `json:"ok"`
		Error     string `json:"error"`
		Output    string `json:"output"`
		SizeBytes int64  `json:"size_bytes"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return "", fmt.Errorf("解析结果失败: %w\n输出: %s", err, string(output))
	}
	if !result.OK {
		return "", fmt.Errorf("图表生成失败: %s", result.Error)
	}

	if len(p.Series) > 0 {
		return tool.WrapText(fmt.Sprintf("✅ 图表已生成: %s（%d 字节，类型: %s）\n标题: %s\n系列: %d · 类别: %d",
			result.Output, result.SizeBytes, p.ChartType, p.Title, len(p.Series), len(p.Labels))), nil
	}
	return tool.WrapText(fmt.Sprintf("✅ 图表已生成: %s（%d 字节，类型: %s）\n标题: %s\n数据点: %d",
		result.Output, result.SizeBytes, p.ChartType, p.Title, len(p.Values))), nil
}

// lookPathFirst returns the first executable found among candidates.
func lookPathFirst(candidates []string) (string, error) {
	var lastErr error
	for _, c := range candidates {
		if p, err := exec.LookPath(c); err == nil {
			return p, nil
		} else {
			lastErr = err
		}
	}
	return "", lastErr
}
