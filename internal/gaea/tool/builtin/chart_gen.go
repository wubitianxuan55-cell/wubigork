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
	return "生成统计图表：支持柱状图、折线图、饼图、散点图。使用 Python matplotlib 生成 PNG/SVG 图片，通常几秒。需要系统中安装 Python 和 matplotlib。"
}

func (chartGen) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "labels":{"type":"array","items":{"type":"string"},"description":"数据标签列表"},
  "values":{"type":"array","items":{"type":"number"},"description":"数据值列表"},
  "chart_type":{"type":"string","description":"图表类型：bar（柱状图）、line（折线图）、pie（饼图）、scatter（散点图）","default":"bar"},
  "title":{"type":"string","description":"图表标题"},
  "output":{"type":"string","description":"输出图片路径（.png 或 .svg）","default":"chart.png"},
  "xlabel":{"type":"string","description":"X轴标签"},
  "ylabel":{"type":"string","description":"Y轴标签"}
},
"required":["labels","values"]
}`)
}

func (chartGen) ReadOnly() bool { return false }

func (chartGen) CompactDescription() string     { return compactDesc["chart_gen"] }
func (chartGen) CompactSchema() json.RawMessage { return compactSchema["chart_gen"] }

var chartScript = `
import json, sys, os
import logging, warnings
warnings.filterwarnings("ignore")
logging.getLogger("matplotlib").setLevel(logging.ERROR)
logging.getLogger("matplotlib.font_manager").setLevel(logging.ERROR)
try:
    import matplotlib
    matplotlib.use('Agg')
    import matplotlib.pyplot as plt
except ImportError:
    print(json.dumps({"ok": False, "error": "matplotlib not installed. Run: pip install matplotlib"}))
    sys.exit(1)

# 尝试加载中文字体
for font in ['SimHei', 'Microsoft YaHei', 'WenQuanYi Micro Hei', 'Noto Sans CJK SC', 'PingFang SC', 'Arial Unicode MS']:
    try:
        plt.rcParams['font.sans-serif'] = [font]
        plt.rcParams['axes.unicode_minus'] = False
        break
    except:
        continue

params = json.loads(sys.stdin.buffer.read().decode('utf-8'))
labels = params.get('labels', [])
values = params.get('values', [])
ctype = params.get('chart_type', 'bar')
title = params.get('title', '')
output = params.get('output', 'chart.png')
xlabel = params.get('xlabel', '')
ylabel = params.get('ylabel', '')

fig, ax = plt.subplots(figsize=(10, 6))

if ctype == 'bar':
    ax.bar(labels, values, color='#4A90D9', edgecolor='white')
    for i, v in enumerate(values):
        ax.text(i, v + max(values)*0.01, str(v), ha='center', fontsize=9)
elif ctype == 'line':
    ax.plot(labels, values, marker='o', linewidth=2, color='#4A90D9', markersize=6)
    for i, v in enumerate(values):
        ax.text(i, v, str(v), ha='center', va='bottom', fontsize=9)
elif ctype == 'pie':
    colors = ['#4A90D9', '#7EC8E3', '#F5A623', '#D0021B', '#7ED321', '#B8E986', '#F8E71C', '#9B9B9B']
    wedges, texts, autotexts = ax.pie(values, labels=labels, autopct='%%1.1f%%%%', colors=colors[:len(values)])
    for t in autotexts:
        t.set_fontsize(9)
elif ctype == 'scatter':
    x = list(range(len(values)))
    ax.scatter(x, values, color='#4A90D9', s=60, alpha=0.7)
    if labels:
        ax.set_xticks(x)
        ax.set_xticklabels(labels, rotation=45, ha='right')

if title:
    ax.set_title(title, fontsize=14, fontweight='bold')
if xlabel:
    ax.set_xlabel(xlabel)
if ylabel:
    ax.set_ylabel(ylabel)

if ctype != 'pie':
    plt.xticks(rotation=45, ha='right')
plt.tight_layout()
plt.savefig(output, dpi=150, bbox_inches='tight')
plt.close()

size_bytes = os.path.getsize(output)
print(json.dumps({"ok": True, "output": output, "size_bytes": size_bytes}))
`

func (chartGen) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Labels    []string  `json:"labels"`
		Values    []float64 `json:"values"`
		ChartType string    `json:"chart_type,omitempty"`
		Title     string    `json:"title,omitempty"`
		Output    string    `json:"output,omitempty"`
		XLabel    string    `json:"xlabel,omitempty"`
		YLabel    string    `json:"ylabel,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	if len(p.Labels) == 0 || len(p.Values) == 0 {
		return "", fmt.Errorf("labels 和 values 不能为空")
	}
	if len(p.Labels) != len(p.Values) {
		return "", fmt.Errorf("labels 和 values 长度不一致")
	}
	if p.ChartType == "" {
		p.ChartType = "bar"
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

	return tool.WrapText(fmt.Sprintf("✅ 图表已生成: %s（%d 字节，类型: %s）\n标题: %s\n数据点: %d",
		result.Output, result.SizeBytes, p.ChartType, p.Title, len(p.Labels))), nil
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
