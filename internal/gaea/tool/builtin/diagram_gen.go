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

func init() { tool.RegisterBuiltin(diagramGen{}) }

// diagramGen 生成框架图/流程图（matplotlib 确定性渲染，中文清晰）。
// 与 chart_gen（统计图表）互补：文字密集的结构图（架构图/流程图/框架图）
// 走本工具，避免文生图模型把中文渲染成乱码。
type diagramGen struct{}

func (diagramGen) Name() string { return "diagram_gen" }

func (diagramGen) Description() string {
	return "生成框架图/流程图（matplotlib，中文清晰）：架构图、层级框架图、业务流程/流程图。输入节点与连线，输出 PNG。文字密集的结构图请用本工具，不要用文生图（Flux/Z-Image-Turbo 会把中文画成乱码）。"
}

func (diagramGen) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "title":{"type":"string","description":"图表标题（中文可正常渲染）"},
  "kind":{"type":"string","enum":["framework","flow"],"description":"framework=分层框架图（按 level 分组横向排布）；flow=流程图（节点按顺序竖排 + 箭头连线）","default":"framework"},
  "nodes":{"type":"array","items":{"type":"object","properties":{"id":{"type":"string"},"label":{"type":"string"},"level":{"type":"integer","description":"framework 模式的分层号，0 为顶层"},"group":{"type":"string","description":"framework 模式的层名（同 level 节点建议同一 group）"}},"required":["id","label"]},"description":"节点列表"},
  "edges":{"type":"array","items":{"type":"object","properties":{"from":{"type":"string"},"to":{"type":"string"},"label":{"type":"string"}},"required":["from","to"]},"description":"连线（flow 模式必填；framework 模式可选，层间自动画箭头）"},
  "output":{"type":"string","description":"输出图片路径（.png）","default":"diagram.png"}
},
"required":["nodes"]
}`)
}

func (diagramGen) ReadOnly() bool { return false }

func (diagramGen) CompactDescription() string     { return compactDesc["diagram_gen"] }
func (diagramGen) CompactSchema() json.RawMessage { return compactSchema["diagram_gen"] }

var diagramScript = `
import json, sys, os
import logging, warnings
warnings.filterwarnings("ignore")
logging.getLogger("matplotlib").setLevel(logging.ERROR)
logging.getLogger("matplotlib.font_manager").setLevel(logging.ERROR)
try:
    import matplotlib
    matplotlib.use('Agg')
    import matplotlib.pyplot as plt
    from matplotlib.patches import FancyBboxPatch, FancyArrowPatch
    import matplotlib.font_manager as fm
except ImportError:
    print(json.dumps({"ok": False, "error": "matplotlib not installed. Run: pip install matplotlib"}))
    sys.exit(1)

# 中文字体：SimHei 优先，缺则 Microsoft YaHei / Noto Sans CJK SC
for font in ['SimHei', 'Microsoft YaHei', 'Noto Sans CJK SC', 'PingFang SC']:
    try:
        fm.findfont(font, fallback_to_default=False)
        plt.rcParams['font.sans-serif'] = [font]
        plt.rcParams['axes.unicode_minus'] = False
        break
    except Exception:
        continue

params = json.loads(sys.stdin.buffer.read().decode('utf-8'))
title = params.get('title', '')
kind = params.get('kind', 'framework')
nodes = params.get('nodes', [])
edges = params.get('edges', [])
output = params.get('output', 'diagram.png')

if not nodes:
    print(json.dumps({"ok": False, "error": "nodes 不能为空"}))
    sys.exit(1)

NAVY = '#1F3864'
BLUE = '#2E74B5'
GRAY = '#404040'
BG = '#D5E8F0'

def wrap_label(label, max_chars=14):
    label = str(label)
    if len(label) <= max_chars:
        return label
    lines = []
    while len(label) > max_chars:
        lines.append(label[:max_chars])
        label = label[max_chars:]
    lines.append(label)
    return '\n'.join(lines)

def draw_node(ax, x, y, w, h, label, fontsize=11):
    ax.add_patch(FancyBboxPatch((x, y), w, h, boxstyle="round,pad=0.02",
                                linewidth=1, edgecolor=GRAY, facecolor="white"))
    ax.text(x + w/2, y + h/2, wrap_label(label), fontsize=fontsize,
            ha="center", va="center", color=GRAY)

def draw_arrow(ax, x1, y1, x2, y2, label=""):
    ax.add_patch(FancyArrowPatch((x1, y1), (x2, y2), arrowstyle="-|>",
                                 mutation_scale=16, color="#7f7f7f", linewidth=1.4))
    if label:
        mx, my = (x1+x2)/2, (y1+y2)/2
        ax.text(mx, my, label, fontsize=9, ha="center", va="center",
                color="#7f7f7f", bbox=dict(facecolor="white", edgecolor="none", pad=0.8))

if kind == "framework":
    # 按 level 分组；同 level 节点横向排布，层间画聚合箭头
    groups = {}
    order = []
    for n in nodes:
        lv = int(n.get('level', 0))
        if lv not in groups:
            groups[lv] = []
            order.append(lv)
        groups[lv].append(n)
    order.sort()
    nlevel = len(order)
    fig_h = max(5, 1.2 + nlevel * 1.7)
    fig, ax = plt.subplots(figsize=(12, fig_h))
    ax.set_xlim(0, 12)
    ax.set_ylim(0, fig_h)
    ax.axis('off')
    top = fig_h - 0.7
    lane_h = 0.9
    for idx, lv in enumerate(order):
        y = top - idx * 1.7
        group_name = groups[lv][0].get('group', '')
        # 层背景条
        ax.add_patch(FancyBboxPatch((0.35, y - lane_h - 0.1), 11.3, lane_h + 0.2,
                                    boxstyle="round,pad=0.02", linewidth=1,
                                    edgecolor=BLUE, facecolor=BG))
        if group_name:
            ax.text(0.55, y - lane_h/2, str(group_name), fontsize=12, fontweight="bold",
                    color=NAVY, va="center", rotation=90)
        items = groups[lv]
        n = len(items)
        w = min(2.8, (10.8 - (n-1)*0.3) / n)
        total = n*w + (n-1)*0.3
        x0 = (12-total)/2
        for i, node in enumerate(items):
            x = x0 + i*(w+0.3)
            draw_node(ax, x, y - lane_h, w, lane_h, node.get('label', node.get('id', '')))
        if idx < nlevel - 1:
            draw_arrow(ax, 6, y - lane_h - 0.22, 6, y - 1.7 + lane_h + 0.22)
elif kind == "flow":
    n = len(nodes)
    lane_h = 1.0
    gap = 1.3
    fig_h = max(4, 1.5 + n * gap)
    fig, ax = plt.subplots(figsize=(12, fig_h))
    ax.set_xlim(0, 12)
    ax.set_ylim(0, fig_h)
    ax.axis('off')
    by_id = {str(n.get('id')): n for n in nodes}
    pos = {}
    top = fig_h - 0.9
    for i, node in enumerate(nodes):
        y = top - i * gap
        pos[str(node.get('id'))] = (6, y)
        draw_node(ax, 3.6, y - lane_h/2, 4.8, lane_h, node.get('label', node.get('id', '')), fontsize=12)
    for e in edges:
        frm, to = str(e.get('from')), str(e.get('to'))
        if frm in pos and to in pos:
            x1, y1 = pos[frm]
            x2, y2 = pos[to]
            draw_arrow(ax, x1, y1 - lane_h/2 - 0.08, x2, y2 + lane_h/2 + 0.08, e.get('label', ''))
else:
    print(json.dumps({"ok": False, "error": "kind 仅支持 framework/flow"}))
    sys.exit(1)

if title:
    ax.set_title(title, fontsize=15, fontweight="bold", color=NAVY)
plt.tight_layout()
plt.savefig(output, dpi=150, bbox_inches="tight")
plt.close()

size_bytes = os.path.getsize(output)
print(json.dumps({"ok": True, "output": output, "size_bytes": size_bytes, "nodes": len(nodes)}))
`

func (diagramGen) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Title  string `json:"title,omitempty"`
		Kind   string `json:"kind,omitempty"`
		Nodes  []struct {
			ID    string `json:"id"`
			Label string `json:"label"`
			Level int    `json:"level,omitempty"`
			Group string `json:"group,omitempty"`
		} `json:"nodes"`
		Edges []struct {
			From  string `json:"from"`
			To    string `json:"to"`
			Label string `json:"label,omitempty"`
		} `json:"edges,omitempty"`
		Output string `json:"output,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	if len(p.Nodes) == 0 {
		return "", fmt.Errorf("nodes 不能为空")
	}
	if p.Kind == "" {
		p.Kind = "framework"
	}
	if p.Kind != "framework" && p.Kind != "flow" {
		return "", fmt.Errorf("kind 仅支持 framework/flow")
	}
	for i := range p.Nodes {
		if strings.TrimSpace(p.Nodes[i].ID) == "" {
			p.Nodes[i].ID = fmt.Sprintf("n%d", i+1)
		}
		if strings.TrimSpace(p.Nodes[i].Label) == "" {
			p.Nodes[i].Label = p.Nodes[i].ID
		}
	}
	if p.Output == "" {
		p.Output = "diagram.png"
	}
	if dir := filepath.Dir(p.Output); dir != "." {
		os.MkdirAll(dir, 0755)
	}

	// 查找 Python：Windows 优先 python（python3 常被商店别名劫持）
	candidates := []string{"python", "python3"}
	if runtime.GOOS != "windows" {
		candidates = []string{"python3", "python"}
	}
	var py string
	for _, c := range candidates {
		if _, err := exec.LookPath(c); err == nil {
			py = c
			break
		}
	}
	if py == "" {
		return "", fmt.Errorf("未找到 Python，请先安装 Python")
	}

	payload, _ := json.Marshal(p)
	cmd := exec.CommandContext(ctx, py, "-c", diagramScript)
	cmd.Stdin = strings.NewReader(string(payload))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("diagram_gen 执行失败: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	var res map[string]interface{}
	if err := json.Unmarshal(out, &res); err != nil {
		return "", fmt.Errorf("diagram_gen 输出解析失败: %s", strings.TrimSpace(string(out)))
	}
	if ok, _ := res["ok"].(bool); !ok {
		return "", fmt.Errorf("diagram_gen: %v", res["error"])
	}
	outJSON, _ := json.Marshal(res)
	return string(outJSON), nil
}
