// CostGraphView.tsx — v4.8 成本知识图谱（造价数据库 · 知识图谱模块）。
// 后端 GaeaCostGraph 返回 JSON 串（CostGraphView），本组件 JSON.parse 后用
// 手写 SVG 环形布局渲染（零额外依赖，布局改自 WhisperGraphPanel 的同心圆环）：
//   · scope=tree（分类总览）：分类按路径深度分层环 + 项目最外环；
//   · scope=entry（条目展开）：以 focus（分类/项目）为中心 BFS 分环，沿
//     contains/references/benchmarks/suggests/notes/belongs_to 展开关联。
// 节点按 type 着色（design token 类，无裸 hex），点击节点弹 antd Modal 展示
// Meta 明细；节点/边硬上限截断时给出提示。
import { useCallback, useEffect, useMemo, useState } from "react";
import { Modal } from "antd";
import { ListTree, Loader, RefreshCw } from "../../icons";
import { app } from "../../lib/bridge";
import type { CostCategory, CostGraphNode, CostGraphView, CostProjectSummary } from "../../lib/types";
import { EmptyState } from "../EmptyState";

// ── 节点 type → 着色/中文标签（与 costref/graph.go 枚举对齐）──────────
const TYPE_COLORS: Record<string, { fill: string; dot: string; label: string }> = {
  category: { fill: "fill-sky-400", dot: "bg-sky-400", label: "分类" },
  entry: { fill: "fill-emerald-400", dot: "bg-emerald-400", label: "条目" },
  project: { fill: "fill-violet-400", dot: "bg-violet-400", label: "项目" },
  item: { fill: "fill-amber-400", dot: "bg-amber-400", label: "明细" },
  indicator: { fill: "fill-pink-400", dot: "bg-pink-400", label: "指标" },
  inquiry: { fill: "fill-rose-400", dot: "bg-rose-400", label: "询价" },
  note: { fill: "fill-teal-400", dot: "bg-teal-400", label: "笔记" },
};

function typeColor(type: string) {
  return TYPE_COLORS[type] ?? { fill: "fill-fg-faint", dot: "bg-bg-elev", label: type || "节点" };
}

// ── 边 type → 中文标签 ────────────────────────────────────────────────
const EDGE_LABELS: Record<string, string> = {
  belongs_to: "归属",
  contains: "包含",
  references: "引用",
  benchmarks: "对标",
  suggests: "调差",
  notes: "沉淀",
};

// Meta 键 → 中文（弹窗明细展示；未收录的键原样展示）。
const META_LABELS: Record<string, string> = {
  path: "分类路径",
  entries: "条目数",
  amount: "金额",
  name: "条目名",
  unit: "单位",
  source: "来源",
  status: "状态",
  region: "地区",
  spec: "规格",
  projectId: "项目 ID",
  projectType: "项目类型",
  items: "明细数",
  versions: "版本数",
  quantity: "数量",
  price: "单价",
  entryName: "引用条目",
  samples: "样本数",
  min: "最低",
  max: "最高",
  mean: "均值",
  p25: "P25",
  p75: "P75",
  supplier: "供应商",
  priceDate: "价格期",
  validUntil: "有效期至",
  confidence: "可信度",
  category: "分类",
  boundary: "适用边界",
  risk: "风险提示",
  evidence: "证据来源",
  matchedBy: "匹配方式",
};

const MATCHED_BY_LABELS: Record<string, string> = {
  entry_name: "条目名精确",
  title: "标题归一化",
};

// ── 环形布局（分层环 / BFS 中心环，零依赖）────────────────────────────
const SVG_W = 720;
const SVG_H = 560;
const CX = 360;
const CY = 278;
const R0 = 86; // 第一圈半径
const R_MAX = 248; // 最外圈半径上限

interface PlacedNode {
  node: CostGraphNode;
  x: number;
  y: number;
  r: number;
  depth: number;
}

interface PlacedEdge {
  x1: number;
  y1: number;
  x2: number;
  y2: number;
  type: string;
}

// 节点半径：按金额量级（Val）温和放大，空值给基础半径。
function nodeRadius(type: string, val: number): number {
  const base = type === "project" || type === "category" ? 11 : 8;
  if (val <= 0) return base;
  return base + Math.min(11, Math.sqrt(val) / 4);
}

function ringRadius(depth: number, maxDepth: number): number {
  if (maxDepth <= 1) return R0;
  const gap = Math.min(90, (R_MAX - R0) / (maxDepth - 1));
  return Math.min(R_MAX, R0 + (depth - 1) * gap);
}

// 均分圆环：同环节点均匀布角，奇偶环错开起始角减少径向重叠。
function ringPositions(ids: string[], depth: number, maxDepth: number): Map<string, { x: number; y: number }> {
  const out = new Map<string, { x: number; y: number }>();
  const r = ringRadius(depth, maxDepth);
  const n = Math.max(1, ids.length);
  ids.forEach((id, i) => {
    const angle = (depth % 2 === 0 ? 0 : Math.PI / ids.length) + (i / n) * Math.PI * 2;
    out.set(id, { x: CX + Math.cos(angle) * r, y: CY + Math.sin(angle) * r });
  });
  return out;
}

// tree 布局：分类按 meta.path 深度分层，项目（及其他类型）落最外环。
function layoutTree(view: CostGraphView): { placed: PlacedNode[]; edges: PlacedEdge[] } | null {
  const nodes = view.nodes ?? [];
  if (nodes.length === 0) return null;
  const depthOf = (n: CostGraphNode) => {
    if (n.type === "category" && n.meta?.path) return n.meta.path.split("/").length;
    return 1;
  };
  const catDepths = nodes.filter((n) => n.type === "category").map(depthOf);
  const hasOuter = nodes.some((n) => n.type !== "category");
  const maxDepth = Math.max(1, ...catDepths) + (hasOuter ? 1 : 0);

  const byDepth = new Map<number, string[]>();
  for (const n of nodes) {
    const d = n.type === "category" ? depthOf(n) : Math.max(1, ...catDepths, 1) + 1;
    if (!byDepth.has(d)) byDepth.set(d, []);
    byDepth.get(d)!.push(n.id);
  }
  const positions = new Map<string, { x: number; y: number }>();
  for (const [d, ids] of byDepth) {
    for (const [id, p] of ringPositions(ids, d, maxDepth)) positions.set(id, p);
  }
  return toPlaced(nodes, view.edges ?? [], positions);
}

// entry 布局：以 focus 节点为中心 BFS 分环（无向邻接），孤立节点落最外环。
function layoutEntry(view: CostGraphView, focus: string): { placed: PlacedNode[]; edges: PlacedEdge[] } | null {
  const nodes = view.nodes ?? [];
  if (nodes.length === 0) return null;
  const center =
    nodes.find((n) => n.id === "proj:" + focus) ??
    nodes.find((n) => n.type === "category" && n.meta?.path === focus) ??
    nodes.find((n) => n.id === focus) ??
    nodes[0];

  const adj = new Map<string, Set<string>>();
  for (const e of view.edges ?? []) {
    if (!adj.has(e.source)) adj.set(e.source, new Set());
    if (!adj.has(e.target)) adj.set(e.target, new Set());
    adj.get(e.source)!.add(e.target);
    adj.get(e.target)!.add(e.source);
  }
  const depth = new Map<string, number>([[center.id, 0]]);
  const queue: string[] = [center.id];
  while (queue.length > 0) {
    const cur = queue.shift()!;
    for (const nb of adj.get(cur) ?? []) {
      if (!depth.has(nb)) {
        depth.set(nb, (depth.get(cur) ?? 0) + 1);
        queue.push(nb);
      }
    }
  }
  const maxBfs = Math.max(0, ...[...depth.values()]);
  const maxDepth = Math.max(1, maxBfs);
  const byDepth = new Map<number, string[]>();
  for (const n of nodes) {
    const d = Math.min(depth.get(n.id) ?? maxBfs + 1, maxBfs + 1);
    if (!byDepth.has(d)) byDepth.set(d, []);
    byDepth.get(d)!.push(n.id);
  }
  const positions = new Map<string, { x: number; y: number }>();
  for (const [d, ids] of byDepth) {
    if (d === 0) {
      positions.set(ids[0], { x: CX, y: CY });
      for (const extra of ids.slice(1)) {
        // 中心环理论只有一个节点，防御多余
        const p = ringPositions([extra], 1, maxDepth).get(extra)!;
        positions.set(extra, p);
      }
      continue;
    }
    for (const [id, p] of ringPositions(ids, d, maxDepth)) positions.set(id, p);
  }
  return toPlaced(nodes, view.edges ?? [], positions);
}

function toPlaced(
  nodes: CostGraphNode[],
  edges: CostGraphView["edges"],
  positions: Map<string, { x: number; y: number }>,
): { placed: PlacedNode[]; edges: PlacedEdge[] } | null {
  const placed: PlacedNode[] = [];
  for (const node of nodes) {
    const p = positions.get(node.id);
    if (!p) continue;
    placed.push({ node, x: p.x, y: p.y, r: nodeRadius(node.type, node.val), depth: 0 });
  }
  if (placed.length === 0) return null;
  const layoutEdges: PlacedEdge[] = [];
  for (const e of edges) {
    const f = positions.get(e.source);
    const t = positions.get(e.target);
    if (!f || !t) continue;
    layoutEdges.push({ x1: f.x, y1: f.y, x2: t.x, y2: t.y, type: e.type });
  }
  return { placed, edges: layoutEdges };
}

function shortLabel(name: string): string {
  return name.length > 9 ? `${name.slice(0, 9)}…` : name;
}

const fmtMoney = (v: number) => "¥" + new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 }).format(v);
const ghostBtn =
  "inline-flex items-center gap-1 px-2.5 h-7 rounded-lg border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors text-[11.5px]";

/**
 * CostGraphView 成本知识图谱：分类/条目/项目/明细/指标/询价/笔记关联图。
 */
export function CostGraphView() {
  const [scope, setScope] = useState<"tree" | "entry">("tree");
  const [focus, setFocus] = useState("");
  const [graph, setGraph] = useState<CostGraphView | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<CostGraphNode | null>(null);
  const [catPaths, setCatPaths] = useState<string[]>([]);
  const [projects, setProjects] = useState<CostProjectSummary[]>([]);

  // focus 选择器数据：分类树铺平为路径 + 项目列表（加载失败静默，仅少选项）。
  useEffect(() => {
    (async () => {
      try {
        const cats = (await app.CostCategories()) ?? [];
        const paths: string[] = [];
        const walk = (nodes: CostCategory[], parent: string) => {
          for (const c of nodes) {
            const p = parent ? `${parent}/${c.name}` : c.name;
            paths.push(p);
            walk(c.children ?? [], p);
          }
        };
        walk(cats, "");
        setCatPaths(paths);
      } catch {
        setCatPaths([]);
      }
      try {
        setProjects((await app.CostProjectList()) ?? []);
      } catch {
        setProjects([]);
      }
    })();
  }, []);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const raw = await app.CostGraph(scope, scope === "entry" ? focus : "", 300);
      setGraph(JSON.parse(raw) as CostGraphView);
    } catch (e) {
      setGraph(null);
      setError(String(e));
    } finally {
      setLoading(false);
    }
  }, [scope, focus]);

  useEffect(() => {
    // entry 展开需要 focus；未选择时给出引导，不发空查询。
    if (scope === "entry" && !focus) {
      setGraph(null);
      setError(null);
      setLoading(false);
      return;
    }
    void load();
  }, [focus, load, scope]);

  const layout = useMemo(() => {
    if (!graph || (graph.nodes ?? []).length === 0) return null;
    return scope === "entry" ? layoutEntry(graph, focus) : layoutTree(graph);
  }, [focus, graph, scope]);

  const stats = graph?.stats;
  const showLegend = Boolean(layout && (graph?.nodes ?? []).length > 0);
  const moneyVal = selected ? selected.type !== "note" : false;

  return (
    <div className="h-full flex flex-col min-h-0 text-[12.5px]">
      {/* 顶栏：标题 + scope 切换 + focus 选择器 + 刷新 */}
      <div className="shrink-0 flex items-center gap-3 px-5 h-12 border-b border-border-soft/60">
        <span className="text-fg font-semibold text-[13px] flex items-center gap-1.5">
          <ListTree size={14} className="text-accent" /> 知识图谱
        </span>
        <span className="text-[11px] text-fg-faint hidden xl:inline">
          分类·条目·项目·明细·指标·询价·笔记 关联总览
        </span>
        <div className="ml-auto flex items-center gap-1.5">
          <div className="flex items-center rounded-lg border border-border bg-bg p-0.5 text-[11px]">
            <button
              type="button"
              className={`px-2.5 h-6 rounded-md transition-colors ${scope === "tree" ? "bg-accent text-white" : "text-fg-faint hover:text-fg"}`}
              onClick={() => setScope("tree")}
              title="分类树聚合总览（每分类一个节点，金额=子树合计）"
            >
              分类总览
            </button>
            <button
              type="button"
              className={`px-2.5 h-6 rounded-md transition-colors ${scope === "entry" ? "bg-accent text-white" : "text-fg-faint hover:text-fg"}`}
              onClick={() => setScope("entry")}
              title="以分类/项目为中心展开条目与关联"
            >
              条目展开
            </button>
          </div>
          {scope === "entry" && (
            <select
              aria-label="展开范围"
              value={focus}
              onChange={(e) => setFocus(e.target.value)}
              className="px-2 h-7 rounded-lg border border-border bg-bg text-fg text-[11.5px] outline-none focus:border-accent max-w-52"
            >
              <option value="">选择分类或项目…</option>
              {catPaths.length > 0 && (
                <optgroup label="分类">
                  {catPaths.map((p) => (
                    <option key={p} value={p}>
                      {p}
                    </option>
                  ))}
                </optgroup>
              )}
              {projects.length > 0 && (
                <optgroup label="项目">
                  {projects.map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.name}
                    </option>
                  ))}
                </optgroup>
              )}
            </select>
          )}
          <button type="button" className={ghostBtn} onClick={() => void load()} title="刷新" disabled={scope === "entry" && !focus}>
            <RefreshCw size={12} />
          </button>
        </div>
      </div>

      {/* 规模与截断提示条 */}
      <div className="shrink-0 flex items-center gap-2 px-5 h-8 border-b border-border-soft/40 text-[10.5px] text-fg-faint">
        {stats ? (
          <>
            <span className="tabular-nums">
              节点 {stats.nodeCount} · 边 {stats.edgeCount}
            </span>
            {stats.truncated && (
              <span className="inline-flex items-center px-1.5 py-px rounded bg-amber-400/15 text-amber-400 font-medium">
                已达节点上限，仅展示部分子图
              </span>
            )}
          </>
        ) : (
          <span>{scope === "entry" && !focus ? "选择分类或项目后展开关联图谱" : "加载中…"}</span>
        )}
        <span className="ml-auto">点击节点查看明细</span>
      </div>

      {/* 图谱渲染区 */}
      <div className="flex-1 min-h-0 overflow-y-auto">
        {loading ? (
          <div className="flex items-center justify-center h-64 text-fg-faint text-[11.5px] gap-1.5">
            <Loader size={14} className="animate-spin" /> 组图计算中…
          </div>
        ) : error ? (
          <div role="alert" className="m-4 px-3 py-3 rounded-lg border border-err/40 bg-err/10 text-fg-dim text-[11.5px]">
            <span className="text-err font-medium">图谱加载失败：{error}</span>
          </div>
        ) : !graph || (graph.nodes ?? []).length === 0 ? (
          <EmptyState message="暂无图谱数据——先在成本库/测算项目里沉淀一些条目" />
        ) : layout ? (
          <div className="p-2">
            <svg viewBox={`0 0 ${SVG_W} ${SVG_H}`} className="w-full h-auto block" role="img" aria-label="成本知识图谱">
              {/* 边：细线 + 类型标签（边多时省略标签防糊） */}
              {layout.edges.map((e, i) => (
                <g key={`edge-${i}`}>
                  <line x1={e.x1} y1={e.y1} x2={e.x2} y2={e.y2} className="stroke-fg-faint/50" strokeWidth={1} />
                  {layout.edges.length <= 40 && (
                    <text
                      x={(e.x1 + e.x2) / 2}
                      y={(e.y1 + e.y2) / 2 - 3}
                      textAnchor="middle"
                      className="fill-fg-faint"
                      fontSize={8.5}
                    >
                      {EDGE_LABELS[e.type] ?? e.type}
                    </text>
                  )}
                </g>
              ))}
              {/* 节点：圆（type 着色）+ 名称；点击弹 Meta 明细 */}
              {layout.placed.map((p) => (
                <g
                  key={p.node.id}
                  className="cursor-pointer"
                  onClick={() => setSelected(p.node)}
                  aria-label={`节点 ${p.node.name}`}
                >
                  <title>{`${p.node.name}（${typeColor(p.node.type).label}）${p.node.desc ? " · " + p.node.desc : ""}`}</title>
                  <circle
                    cx={p.x}
                    cy={p.y}
                    r={p.r}
                    className={`${typeColor(p.node.type).fill} stroke-border`}
                    strokeWidth={1}
                  />
                  <text x={p.x} y={p.y + p.r + 11} textAnchor="middle" className="fill-fg" fontSize={10}>
                    {shortLabel(p.node.name)}
                  </text>
                </g>
              ))}
            </svg>
            {/* 图例 */}
            {showLegend && (
              <div className="flex flex-wrap items-center gap-x-3 gap-y-1 px-3 pb-2 text-[10.5px] text-fg-faint">
                {Object.entries(TYPE_COLORS).map(([type, c]) => {
                  const count = graph?.stats.countsByType?.[type] ?? 0;
                  if (count === 0) return null;
                  return (
                    <span key={type} className="inline-flex items-center gap-1">
                      <span className={`w-2 h-2 rounded-full ${c.dot}`} />
                      {c.label} {count}
                    </span>
                  );
                })}
              </div>
            )}
          </div>
        ) : null}
      </div>

      {/* 节点明细弹窗 */}
      <Modal
        title={
          selected ? (
            <span className="flex items-center gap-2">
              <span className={`w-2.5 h-2.5 rounded-full ${typeColor(selected.type).dot}`} />
              {selected.name}
            </span>
          ) : null
        }
        open={!!selected}
        onCancel={() => setSelected(null)}
        footer={null}
        width={440}
        destroyOnHidden
        transitionName=""
        maskTransitionName=""
      >
        {selected && (
          <div className="space-y-2 text-[12px]">
            <div className="flex items-center gap-2">
              <span className="px-1.5 py-0.5 rounded bg-bg-elev text-fg-dim text-[10.5px]">
                {typeColor(selected.type).label}
              </span>
              {selected.desc && <span className="text-fg-dim">{selected.desc}</span>}
              <span className="ml-auto tabular-nums text-fg font-medium">
                {moneyVal ? fmtMoney(selected.val) : String(selected.val)}
              </span>
            </div>
            {selected.meta && Object.keys(selected.meta).length > 0 && (
              <div className="rounded-lg border border-border overflow-hidden">
                {Object.entries(selected.meta)
                  .filter(([, v]) => v !== "")
                  .map(([k, v]) => (
                    <div key={k} className="flex items-start gap-2 px-2.5 py-1.5 border-b border-border-soft/40 last:border-0">
                      <span className="text-fg-faint shrink-0 w-20">{META_LABELS[k] ?? k}</span>
                      <span className="text-fg-dim break-all">
                        {k === "matchedBy"
                          ? MATCHED_BY_LABELS[v] ?? v
                          : k === "amount" || k === "price"
                            ? fmtMoney(Number(v))
                            : v}
                      </span>
                    </div>
                  ))}
              </div>
            )}
            <div className="text-[10.5px] text-fg-faint">
              节点 ID {selected.id}
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}

export default CostGraphView;
