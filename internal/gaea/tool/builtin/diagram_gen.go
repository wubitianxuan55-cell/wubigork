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
	return "生成框架图/流程图（matplotlib，中文清晰）：架构图、层级框架图、业务流程/流程图。flow 模式按 edges 拓扑自动分层，支持分支/汇合（layout=vertical 自上而下分层，默认；horizontal 从左到右按列展开），检测到循环依赖时报参数错误；节点文字过长自动折行。输入节点与连线，输出 PNG。文字密集的结构图请用本工具，不要用文生图（Flux/Z-Image-Turbo 会把中文画成乱码）。"
}

func (diagramGen) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "title":{"type":"string","description":"图表标题（中文可正常渲染）"},
  "kind":{"type":"string","enum":["framework","flow"],"description":"framework=分层框架图（按 level 分组横向排布）；flow=流程图（按 edges 拓扑分层，入度 0 的节点在最上层，支持分支/汇合，存在循环依赖时报错）","default":"framework"},
  "layout":{"type":"string","enum":["vertical","horizontal"],"description":"flow 模式布局方向：vertical=自上而下分层（默认）；horizontal=从左到右按列展开","default":"vertical"},
  "nodes":{"type":"array","items":{"type":"object","properties":{"id":{"type":"string"},"label":{"type":"string","description":"节点文字（过长自动折行）"},"level":{"type":"integer","description":"framework 模式的分层号，0 为顶层"},"group":{"type":"string","description":"framework 模式的层名（同 level 节点建议同一 group）"}},"required":["id","label"]},"description":"节点列表"},
  "edges":{"type":"array","items":{"type":"object","properties":{"from":{"type":"string"},"to":{"type":"string"},"label":{"type":"string"}},"required":["from","to"]},"description":"连线（flow 模式决定拓扑分层，支持分支/汇合；framework 模式可选，层间自动画箭头）"},
  "output":{"type":"string","description":"输出图片路径（.png）","default":"diagram.png"}
},
"required":["nodes"]
}`)
}

func (diagramGen) ReadOnly() bool { return false }

func (diagramGen) CompactDescription() string     { return compactDesc["diagram_gen"] }
func (diagramGen) CompactSchema() json.RawMessage { return compactSchema["diagram_gen"] }

// wrapLabel 将过长节点 label 按 rune 计数折行（每行约 maxChars 字），
// 返回以 \n 连接的多行文本；Python 端按行数自适应节点框高度。
func wrapLabel(label string, maxChars int) string {
	if maxChars <= 0 {
		return label
	}
	runes := []rune(label)
	if len(runes) <= maxChars {
		return label
	}
	var b strings.Builder
	for i, r := range runes {
		if i > 0 && i%maxChars == 0 {
			b.WriteByte('\n')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// computeFlowLevels 计算 flow 模式的拓扑分层（纯函数，便于单测）：
// 入度 0 的节点 level=0，其余节点 level = 所有前驱 level 的最大值 + 1；
// 引用未知节点的边忽略；检测到环（含自环）时返回参数错误。
func computeFlowLevels(ids []string, edges [][2]string) (map[string]int, error) {
	known := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		known[id] = struct{}{}
	}
	adj := make(map[string][]string, len(ids))
	indeg := make(map[string]int, len(ids))
	for _, id := range ids {
		indeg[id] = 0
	}
	for _, e := range edges {
		if _, ok := known[e[0]]; !ok {
			continue
		}
		if _, ok := known[e[1]]; !ok {
			continue
		}
		adj[e[0]] = append(adj[e[0]], e[1])
		indeg[e[1]]++
	}
	// Kahn 拓扑 + 最长路径分层：maxPred 在节点出队时已汇聚全部前驱的最大 level+1
	maxPred := make(map[string]int, len(ids))
	done := make(map[string]bool, len(ids))
	queue := make([]string, 0, len(ids))
	for _, id := range ids {
		if indeg[id] == 0 {
			queue = append(queue, id)
		}
	}
	levels := make(map[string]int, len(ids))
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		done[cur] = true
		lv := maxPred[cur]
		levels[cur] = lv
		for _, next := range adj[cur] {
			if lv+1 > maxPred[next] {
				maxPred[next] = lv + 1
			}
			indeg[next]--
			if indeg[next] == 0 {
				queue = append(queue, next)
			}
		}
	}
	if len(done) < len(known) {
		stuck := make([]string, 0, len(ids)-len(done))
		for _, id := range ids {
			if !done[id] {
				stuck = append(stuck, id)
			}
		}
		return nil, fmt.Errorf("flow 节点存在循环依赖: %s", strings.Join(stuck, ", "))
	}
	return levels, nil
}

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
layout = params.get('layout', 'vertical')
nodes = params.get('nodes', [])
edges = params.get('edges', [])
levels = params.get('levels', {})   # flow 模式：Go 端拓扑分层结果 id -> level
output = params.get('output', 'diagram.png')

if not nodes:
    print(json.dumps({"ok": False, "error": "nodes 不能为空"}))
    sys.exit(1)

# 调色板与 chart_gen 对齐
PALETTE = ['#4C8BF5', '#F6A609', '#34A853', '#EA4335', '#9B59B6',
           '#00BCD4', '#FF7043', '#8D6E63', '#7986CB', '#66BB6A']
NAVY = '#1F3864'
TEXT = '#33383F'

def theme(i):
    return PALETTE[i % len(PALETTE)]

def mix_white(hex_color, alpha):
    """主题色叠白底（color-mix 效果），返回淡版色作节点底色。"""
    h = hex_color.lstrip('#')
    r, g, b = int(h[0:2], 16), int(h[2:4], 16), int(h[4:6], 16)
    mix = lambda c: int(round(255 * (1 - alpha) + c * alpha))
    return '#%02x%02x%02x' % (mix(r), mix(g), mix(b))

def nlines(s):
    """Go 端已折行（\n 分隔），此处只数行数供节点框自适应高度。"""
    return max(1, str(s).count('\n') + 1)

def draw_node(ax, x, y, w, h, label, edge_color, fontsize=11):
    ax.add_patch(FancyBboxPatch((x, y), w, h, boxstyle="round,pad=0.02",
                                linewidth=1.2, edgecolor=edge_color,
                                facecolor=mix_white(edge_color, 0.15)))
    ax.text(x + w/2, y + h/2, label, fontsize=fontsize,
            ha="center", va="center", color=TEXT)

def draw_arrow(ax, x1, y1, x2, y2, label=""):
    ax.add_patch(FancyArrowPatch((x1, y1), (x2, y2), arrowstyle="-|>",
                                 mutation_scale=16, color="#7f7f7f", linewidth=1.4))
    if label:
        mx, my = (x1+x2)/2, (y1+y2)/2
        ax.text(mx, my, label, fontsize=9, ha="center", va="center",
                color="#7f7f7f", bbox=dict(facecolor="white", edgecolor="none", pad=0.8))

if kind == "framework":
    # 按 level 分组；同 level 节点横向排布，层间画聚合箭头
    # （布局行为与历史版本一致，仅配色/折行/行高自适应升级）
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
    # 层高按该层最大行数自适应
    lanes = []
    for lv in order:
        ml = max(nlines(nd.get('label', nd.get('id', ''))) for nd in groups[lv])
        lanes.append(max(0.9, 0.40 + 0.32 * ml))
    fig_h = max(5, 1.2 + sum(lanes) + 0.8 * nlevel)
    fig, ax = plt.subplots(figsize=(12, fig_h))
    ax.set_xlim(0, 12)
    ax.set_ylim(0, fig_h)
    ax.axis('off')
    top = fig_h - 0.7
    y = top
    for idx, lv in enumerate(order):
        lane_h = lanes[idx]
        color = theme(idx)
        group_name = groups[lv][0].get('group', '')
        # 层背景条（主题色淡版）
        ax.add_patch(FancyBboxPatch((0.35, y - lane_h - 0.1), 11.3, lane_h + 0.2,
                                    boxstyle="round,pad=0.02", linewidth=1,
                                    edgecolor=color, facecolor=mix_white(color, 0.08)))
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
            draw_node(ax, x, y - lane_h, w, lane_h, node.get('label', node.get('id', '')), color)
        if idx < nlevel - 1:
            draw_arrow(ax, 6, y - lane_h - 0.22, 6, y - (lane_h + 0.8) + lanes[idx+1] + 0.22)
        y -= lane_h + 0.8
elif kind == "flow":
    # 按 Go 端拓扑分层结果铺排：层 = 拓扑推进方向，层内节点横向（vertical）/纵向（horizontal）均布
    horiz = (layout == 'horizontal')
    lv_of = {str(k): int(v) for k, v in (levels or {}).items()}
    by_level = {}
    for nd in nodes:
        by_level.setdefault(lv_of.get(str(nd.get('id')), 0), []).append(nd)
    order = sorted(by_level)

    CROSS_GAP = 0.5   # 层内节点间距
    DEPTH_GAP = 1.1   # 层间距（留箭头空间）
    layers = []
    max_cnt = 1
    for lv in order:
        items = by_level[lv]
        max_cnt = max(max_cnt, len(items))
        hs = [max(0.9, 0.40 + 0.32 * nlines(nd.get('label', nd.get('id', '')))) for nd in items]
        layers.append({'items': items, 'hs': hs})

    def extent(sizes, gap):
        return (sum(sizes) + gap * (len(sizes) - 1)) if sizes else 0.0

    if horiz:
        node_w = max(2.8, min(3.8, 34.0 / max(1, len(order))))
        for l in layers:
            l['cs'] = l['hs']                        # cross 向 = 节点高
            l['ds'] = [node_w] * len(l['items'])     # depth 向 = 节点宽
    else:
        # 层内节点多时加宽画布，保证折行文本放得下
        fig_w = max(12.0, 1.2 + max_cnt * 3.4 + (max_cnt - 1) * CROSS_GAP)
        for l in layers:
            cnt = len(l['items'])
            bw = min(4.8, (fig_w - 1.2 - (cnt - 1) * CROSS_GAP) / cnt)
            l['cs'] = [bw] * cnt                     # cross 向 = 节点宽
            l['ds'] = l['hs']                        # depth 向 = 节点高

    cross_total = max(extent(l['cs'], CROSS_GAP) for l in layers)
    depth_total = sum(max(l['ds']) for l in layers) + DEPTH_GAP * max(0, len(layers) - 1)
    if horiz:
        fig_w = max(7.0, 1.2 + depth_total)
        fig_h = max(5.0, cross_total + 1.6)
    else:
        fig_h = max(4.0, depth_total + 1.6)

    fig, ax = plt.subplots(figsize=(fig_w, fig_h))
    ax.set_xlim(0, fig_w)
    ax.set_ylim(0, fig_h)
    ax.axis('off')

    # 层内 cross 居中、层间 depth 推进，映射回 (x, y)
    place = {}
    d0 = 0.0
    x_off = (fig_w - cross_total) / 2.0
    y_off = (fig_h + cross_total) / 2.0
    for li, l in enumerate(layers):
        color = theme(li)
        c_cur = (cross_total - extent(l['cs'], CROSS_GAP)) / 2.0
        for nd, cs, ds in zip(l['items'], l['cs'], l['ds']):
            cid = str(nd.get('id'))
            label = nd.get('label', nd.get('id', ''))
            if horiz:
                x = 0.6 + d0 + ds / 2.0
                y = y_off - (c_cur + cs / 2.0)
                w, h = ds, cs
            else:
                x = x_off + c_cur + cs / 2.0
                y = fig_h - 0.8 - (d0 + ds / 2.0)
                w, h = cs, ds
            draw_node(ax, x - w/2, y - h/2, w, h, label, color, fontsize=12 if w >= 3.3 else 11)
            place[cid] = (x, y, w, h)
            c_cur += cs + CROSS_GAP
        d0 += max(l['ds']) + DEPTH_GAP

    for e in edges:
        a = place.get(str(e.get('from')))
        b = place.get(str(e.get('to')))
        if not a or not b or a is b:
            continue
        if horiz:
            x1, y1 = a[0] + a[2]/2 + 0.06, a[1]
            x2, y2 = b[0] - b[2]/2 - 0.06, b[1]
        else:
            x1, y1 = a[0], a[1] - a[3]/2 - 0.06
            x2, y2 = b[0], b[1] + b[3]/2 + 0.06
        draw_arrow(ax, x1, y1, x2, y2, e.get('label', ''))
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
		Layout string `json:"layout,omitempty"`
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
		Levels map[string]int `json:"levels,omitempty"`
		Output string         `json:"output,omitempty"`
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
	if p.Layout == "" {
		p.Layout = "vertical"
	}
	if p.Layout != "vertical" && p.Layout != "horizontal" {
		return "", fmt.Errorf("layout 仅支持 vertical/horizontal")
	}
	for i := range p.Nodes {
		if strings.TrimSpace(p.Nodes[i].ID) == "" {
			p.Nodes[i].ID = fmt.Sprintf("n%d", i+1)
		}
		if strings.TrimSpace(p.Nodes[i].Label) == "" {
			p.Nodes[i].Label = p.Nodes[i].ID
		}
		p.Nodes[i].Label = wrapLabel(p.Nodes[i].Label, 14)
	}
	if p.Kind == "flow" {
		ids := make([]string, len(p.Nodes))
		for i := range p.Nodes {
			ids[i] = p.Nodes[i].ID
		}
		pairs := make([][2]string, len(p.Edges))
		for i := range p.Edges {
			pairs[i] = [2]string{p.Edges[i].From, p.Edges[i].To}
		}
		levels, err := computeFlowLevels(ids, pairs)
		if err != nil {
			return "", err
		}
		p.Levels = levels
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
	hideBashWindow(cmd) // Windows: 防止弹出 cmd 黑框
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
