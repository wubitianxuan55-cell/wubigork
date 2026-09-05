// mindmap.ts — 思维导图视图纯函数层（M1，docs/gaea-office-mindmap-base-design-2026-09.md §3）。
// 权威格式 = markdown 大纲：首个 H1 为根，H2–H6 作次级根按级别嵌套，嵌套列表在
// 「最近标题」之下按缩进成树（列表优先口径，单测钉死）。文件里没有大纲时回退
// fallbackTitle 合成单节点根，视图层给空态提示。布局为纯数学（右向逻辑树），
// 无 DOM 测量，jsdom 可测。

export interface MindNode {
  id: string;
  text: string;
  /** 绝对深度：root=0，H2=1…H6=5；列表项 = 最近标题深度 + 1 + 缩进档。 */
  depth: number;
  children: MindNode[];
}

export interface MindmapParseResult {
  root: MindNode;
  /** 节点总数（含 root）。 */
  count: number;
  /** 触发节点上限被截断（继续消费行以维持围栏状态，但不再建节点）。 */
  truncated: boolean;
}

export const MINDMAP_MAX_NODES = 500;
const MAX_NODE_TEXT = 120;

interface Token {
  kind: "heading" | "list";
  level: number; // heading: 1–6；list: 缩进档（0 起）
  text: string;
}

function stripCheckbox(text: string): string {
  // 任务勾选框转可见符号，避免大纲里出现裸 "[x]" 噪音
  return text.replace(/^\[([ xX])\]\s*/, (_m, mark: string) => (mark.toLowerCase() === "x" ? "☑ " : "☐ "));
}

function toToken(rawLine: string): Token | null {
  const line = rawLine.replace(/\t/g, "  ");
  const h = /^(#{1,6})\s+(.+?)\s*#*\s*$/.exec(line);
  if (h) return { kind: "heading", level: h[1].length, text: h[2] };
  // 无序符号与「N./N)」要求后随空白（防误吃 "3.14 是 pi" 这类行）；
  // 中文顿号「N、」允许零空白（中文大纲习惯）
  const li = /^(\s*)(?:[-*+]\s+|\d+[.)]\s+|\d+、\s*)(.+)$/.exec(line);
  if (li) {
    const indent = li[1].length;
    return { kind: "list", level: Math.floor(indent / 2), text: li[2] };
  }
  return null;
}

interface WalkResult {
  y: number;
  item: MindLayoutNode;
  visibleChildren: WalkResult[];
}

export function parseMindmapOutline(text: string, fallbackTitle: string): MindmapParseResult {
  const root: MindNode = { id: "n0", text: fallbackTitle || "思维导图", depth: 0, children: [] };
  let count = 1;
  let truncated = false;
  let sawH1 = false;
  let lastHeadingDepth = -1;
  const stack: MindNode[] = [root];
  let idSeq = 1;
  let inFence = false;

  const attach = (node: MindNode): void => {
    while (stack.length > 1 && (stack[stack.length - 1] as MindNode).depth >= node.depth) stack.pop();
    (stack[stack.length - 1] as MindNode).children.push(node);
    stack.push(node);
  };

  for (const rawLine of text.split(/\r?\n/)) {
    if (/^\s*(```|~~~)/.test(rawLine)) {
      inFence = !inFence;
      continue;
    }
    if (inFence) continue;
    const token = toToken(rawLine);
    if (!token || token.text.trim() === "") continue;

    if (count >= MINDMAP_MAX_NODES) {
      truncated = true;
      continue;
    }

    if (token.kind === "heading") {
      if (token.level === 1 && !sawH1) {
        // 首个 H1 直接命名根（不新建节点）；H1 前若已有游离内容，收拢回根层
        root.text = token.text;
        sawH1 = true;
        stack.length = 1;
        lastHeadingDepth = 0;
        continue;
      }
      const depth = token.level === 1 ? 1 : token.level - 1;
      const node: MindNode = { id: `n${idSeq++}`, text: token.text, depth, children: [] };
      attach(node);
      lastHeadingDepth = depth;
    } else {
      const base = lastHeadingDepth < 0 ? 0 : lastHeadingDepth;
      const depth = base + 1 + token.level;
      const node: MindNode = { id: `n${idSeq++}`, text: token.text, depth, children: [] };
      attach(node);
    }
    const added = stack[stack.length - 1] as MindNode;
    added.text = stripCheckbox(added.text).slice(0, MAX_NODE_TEXT);
    count++;
  }
  return { root, count, truncated };
}

// ─── 布局（右向逻辑树）────────────────────────────────────────────

export const MM_NODE_W = 176;
export const MM_NODE_H = 30;
export const MM_GAP_X = 40;
export const MM_GAP_Y = 8;
const MM_PAD = 12;

export interface MindLayoutNode {
  id: string;
  text: string;
  depth: number;
  x: number;
  /** 节点顶边 y。 */
  y: number;
  w: number;
  h: number;
  hasChildren: boolean;
  collapsed: boolean;
  /** 折叠时的直接子节点数（badge 用）。 */
  hiddenChildren: number;
}

export interface MindLayout {
  nodes: MindLayoutNode[];
  /** 连线 path d（父右缘中点 → 子左缘中点三次贝塞尔）。 */
  edges: string[];
  width: number;
  height: number;
}

function edgeD(parent: MindLayoutNode, child: MindLayoutNode): string {
  const x1 = parent.x + parent.w;
  const y1 = parent.y + parent.h / 2;
  const x2 = child.x;
  const y2 = child.y + child.h / 2;
  const dx = (x2 - x1) / 2;
  return `M ${x1} ${y1} C ${x1 + dx} ${y1}, ${x2 - dx} ${y2}, ${x2} ${y2}`;
}

/** 折叠集合中的节点收起子树（badge 显直接子节点数）。布局永不稳定 → 组件 useMemo。 */
export function layoutMindmap(root: MindNode, collapsed: ReadonlySet<string>): MindLayout {
  let leafRow = 0;
  let maxDepth = 0;

  const walk = (node: MindNode): WalkResult => {
    maxDepth = Math.max(maxDepth, node.depth);
    const isCollapsed = collapsed.has(node.id);
    const x = node.depth * (MM_NODE_W + MM_GAP_X) + MM_PAD;
    const hasChildren = node.children.length > 0;
    if (isCollapsed || !hasChildren) {
      const y = leafRow * (MM_NODE_H + MM_GAP_Y);
      leafRow++;
      return {
        y,
        item: {
          id: node.id, text: node.text, depth: node.depth, x, y: y + MM_PAD,
          w: MM_NODE_W, h: MM_NODE_H, hasChildren, collapsed: isCollapsed,
          hiddenChildren: isCollapsed ? node.children.length : 0,
        },
        visibleChildren: [],
      };
    }
    const results = node.children.map(walk);
    const first = (results[0] as WalkResult).y;
    const last = (results[results.length - 1] as WalkResult).y;
    const y = first + (last - first) / 2;
    return {
      y,
      item: {
        id: node.id, text: node.text, depth: node.depth, x, y: y + MM_PAD,
        w: MM_NODE_W, h: MM_NODE_H, hasChildren, collapsed: false, hiddenChildren: 0,
      },
      visibleChildren: results,
    };
  };

  const nodes: MindLayoutNode[] = [];
  const edges: string[] = [];
  const emit = (r: WalkResult): void => {
    nodes.push(r.item);
    for (const child of r.visibleChildren) {
      edges.push(edgeD(r.item, child.item));
      emit(child);
    }
  };
  emit(walk(root));

  const width = (maxDepth + 1) * (MM_NODE_W + MM_GAP_X) - MM_GAP_X + MM_PAD * 2;
  const height = Math.max(leafRow * (MM_NODE_H + MM_GAP_Y) - MM_GAP_Y, MM_NODE_H) + MM_PAD * 2;
  return { nodes, edges, width, height };
}
