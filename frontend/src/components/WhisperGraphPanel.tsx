// WhisperGraphPanel.tsx — 轻语关系图谱面板（v4.3b）
/* eslint-disable react-refresh/only-export-components -- 组件 + currentPersonalityId 工具同文件（面板与集成页共用，参照 Toast.tsx 模式） */
// 以实体为中心查询 hermes.db 记忆图谱子图（WhisperGraphSubgraph），用
// 手写 SVG + 同心圆环形布局渲染邻接图（不引入额外依赖）；底部提供
// 「轻语先开口」主动关心评估（WhisperProactiveNow，仅评估展示，实际推送
// 由定时器或用户确认后触发）。
import { useCallback, useEffect, useMemo, useState } from "react";
import { Modal } from "antd";
import { ShareAltOutlined, SoundOutlined } from "@ant-design/icons";
import { app } from "../gaea/lib/bridge";
import type {
  WhisperGraphEdge,
  WhisperGraphNode,
  WhisperProactiveResult,
  WhisperSubgraph,
} from "../gaea/lib/types";
import { useToast } from "../gaea/components/Toast";
import { EmptyState } from "../gaea/components/EmptyState";
import { Loader } from "../gaea/icons";
import { BACKEND_EVENTS, subscribeForSpace } from "../events";

// ── 当前轻语人格 ID（与聊天页同源：localStorage，缺省 gaea） ──────────
const PERSONALITY_KEY = "gaea_whisper_personality";
const LEGACY_PERSONALITY_KEY = "wubigrok_whisper_personality";

/** 读取当前轻语人格 ID（聊天板块切换人格时写入的同一键）。 */
export function currentPersonalityId(): string {
  try {
    return (
      window.localStorage.getItem(PERSONALITY_KEY) ??
      window.localStorage.getItem(LEGACY_PERSONALITY_KEY) ??
      "gaea"
    );
  } catch {
    return "gaea";
  }
}

// ── 节点按 type 着色（type 空 → 统一色）；权重高亮描边 ──────────────
const TYPE_FILLS = [
  "fill-sky-400",
  "fill-pink-400",
  "fill-amber-400",
  "fill-emerald-400",
  "fill-violet-400",
  "fill-rose-400",
  "fill-teal-400",
  "fill-indigo-400",
];

function typeFill(type: string): string {
  if (!type) return "fill-fg-faint"; // type 空 → 统一色
  let h = 0;
  for (const ch of type) h = (h * 31 + ch.charCodeAt(0)) >>> 0;
  return TYPE_FILLS[h % TYPE_FILLS.length];
}

// 权重描边：≥0.8 琥珀高亮；其余默认细描边。中心节点用 accent 强调。
function nodeStrokeClass(isCenter: boolean, weight: number): string {
  if (isCenter) return "stroke-accent";
  if (weight >= 0.8) return "stroke-amber-300";
  return "stroke-fg-faint";
}

function nodeStrokeWidth(isCenter: boolean, weight: number): number {
  if (isCenter) return 3;
  return 1 + Math.min(2.5, weight * 2);
}

// v4.9 图谱情绪维度：边按情绪着色（正面绿 / 负面红 / 中性灰；空=默认细灰）。
function edgeStrokeClass(emotion?: string): string {
  switch (emotion) {
    case "正面":
      return "stroke-emerald-400/70";
    case "负面":
      return "stroke-rose-400/70";
    case "中性":
      return "stroke-fg-faint/50";
    default:
      return "stroke-fg-faint/60";
  }
}

// ── 主动关心消息类型 → 中文徽标（其余原文） ────────────────────────
const MESSAGE_TYPE_LABELS: Record<string, string> = {
  check_in: "关怀问候",
  miss_you: "想念",
  time_aware: "时间感知",
  playful_nudge: "俏皮戳一戳",
  // v4.3c 后续小步：生日祝福（定时推送 ticker 构造的主动消息类型）
  birthday: "生日祝福",
};

// ── 环形布局（BFS 深度 → 同心圆，cos/sin 定位，零依赖） ────────────
interface PlacedNode {
  node: WhisperGraphNode;
  x: number;
  y: number;
  depth: number;
}

interface PlacedEdge {
  edge: WhisperGraphEdge;
  x1: number;
  y1: number;
  x2: number;
  y2: number;
}

interface LayoutResult {
  placed: PlacedNode[];
  edges: PlacedEdge[];
  centerId: string;
}

const SVG_W = 700;
const SVG_H = 560;
const CX = 350;
const CY = 280;
const R0 = 100; // 第一圈半径
const RING = 90; // 圈间距

function layoutGraph(
  graph: WhisperSubgraph,
  centerEntity: string,
  hops: number,
): LayoutResult | null {
  const nodes = graph.nodes ?? [];
  const edges = graph.edges ?? [];
  if (nodes.length === 0) return null;

  const byId = new Map(nodes.map((n) => [n.id, n]));
  const adj = new Map<string, Set<string>>();
  for (const e of edges) {
    if (!byId.has(e.from) || !byId.has(e.to)) continue;
    if (!adj.has(e.from)) adj.set(e.from, new Set());
    if (!adj.has(e.to)) adj.set(e.to, new Set());
    adj.get(e.from)!.add(e.to);
    adj.get(e.to)!.add(e.from);
  }

  const center =
    nodes.find((n) => n.name === centerEntity) ??
    nodes.find((n) => n.id === centerEntity) ??
    nodes[0];
  const centerId = center.id;

  // BFS 深度（无向）
  const depth = new Map<string, number>();
  depth.set(centerId, 0);
  const queue: string[] = [centerId];
  while (queue.length > 0) {
    const cur = queue.shift()!;
    const d = depth.get(cur)!;
    for (const nb of adj.get(cur) ?? []) {
      if (!depth.has(nb)) {
        depth.set(nb, d + 1);
        queue.push(nb);
      }
    }
  }

  // 按深度分组，逐圈均分角度（奇偶圈错开起始角避免径向重叠）
  const byDepth = new Map<number, string[]>();
  for (const n of nodes) {
    if (n.id === centerId) continue;
    const d = Math.min(depth.get(n.id) ?? hops, Math.max(1, hops));
    if (!byDepth.has(d)) byDepth.set(d, []);
    byDepth.get(d)!.push(n.id);
  }
  const positions = new Map<string, { x: number; y: number; depth: number }>();
  positions.set(centerId, { x: CX, y: CY, depth: 0 });
  for (const [d, ids] of byDepth) {
    const n = ids.length;
    ids.forEach((id, i) => {
      const angle = (d % 2 === 0 ? 0 : Math.PI / 4) + (i / Math.max(1, n)) * Math.PI * 2;
      const r = R0 + (d - 1) * RING;
      positions.set(id, {
        x: CX + Math.cos(angle) * r,
        y: CY + Math.sin(angle) * r,
        depth: d,
      });
    });
  }

  const placed: PlacedNode[] = nodes
    .map((node) => {
      const p = positions.get(node.id);
      return p ? { node, x: p.x, y: p.y, depth: p.depth } : null;
    })
    .filter((p): p is PlacedNode => p !== null);

  const placedEdges: PlacedEdge[] = edges
    .map((edge) => {
      const f = positions.get(edge.from);
      const t = positions.get(edge.to);
      if (!f || !t) return null;
      return { edge, x1: f.x, y1: f.y, x2: t.x, y2: t.y };
    })
    .filter((e): e is PlacedEdge => e !== null);

  return { placed, edges: placedEdges, centerId };
}

function shortLabel(name: string): string {
  return name.length > 10 ? `${name.slice(0, 10)}…` : name;
}

/**
 * WhisperGraphPanel 轻语关系图谱面板。
 * open/personalityId/onClose 由外层（轻语记忆库页）控制。
 */
export function WhisperGraphPanel({
  open,
  personalityId,
  onClose,
}: {
  open: boolean;
  personalityId: string;
  onClose: () => void;
}) {
  const toast = useToast();
  const [entityInput, setEntityInput] = useState("");
  const [hopsInput, setHopsInput] = useState("1");
  const [graph, setGraph] = useState<WhisperSubgraph | null>(null);
  const [queriedEntity, setQueriedEntity] = useState("");
  const [hasQueried, setHasQueried] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [proactive, setProactive] = useState<WhisperProactiveResult | null>(null);
  const [proactiveBusy, setProactiveBusy] = useState(false);

  // 打开时重置上一次查询/评估结果（关闭即卸载内容，重新打开保持干净）
  useEffect(() => {
    if (!open) return;
    setGraph(null);
    setQueriedEntity("");
    setHasQueried(false);
    setError(null);
    setProactive(null);
  }, [open]);

  // v4.3c 后续小步：订阅轻语主动关心定时推送（gaea-whisper-proactive，play
  // 空间过滤），收到当前人格的推送即显示为「轻语先开口」气泡（与手动评估
  // 同款渲染，messageType/promptHint 由后端 ticker 构造，含生日祝福）。
  useEffect(() => {
    if (!open) return;
    return subscribeForSpace(BACKEND_EVENTS.WHISPER_PROACTIVE, (data) => {
      const payload = data as {
        personalityID?: string;
        messageType?: string;
        promptHint?: string;
      } | null;
      if (!payload) return;
      if (payload.personalityID && payload.personalityID !== personalityId) return;
      setProactive({
        shouldSend: true,
        messageType: payload.messageType,
        promptHint: payload.promptHint,
      });
    }, "play");
  }, [open, personalityId]);

  const clampHops = useCallback((raw: string): number => {
    const n = Number.parseInt(raw, 10);
    if (Number.isNaN(n)) return 1;
    return Math.min(3, Math.max(1, n));
  }, []);

  const handleQuery = useCallback(
    async (entityOverride?: string) => {
      const entity = (entityOverride ?? entityInput).trim();
      if (!entity) {
        toast.show("请输入要查询的实体名称", "warn");
        return;
      }
      const hops = clampHops(hopsInput);
      setLoading(true);
      setError(null);
      try {
        const g = await app.WhisperGraphSubgraph(personalityId, entity, hops);
        setGraph(g);
        setQueriedEntity(entity);
        setHasQueried(true);
      } catch (e) {
        setError(String(e));
        toast.show(`关系图谱查询失败：${String(e)}`, "warn");
      } finally {
        setLoading(false);
      }
    },
    [clampHops, entityInput, hopsInput, personalityId, toast],
  );

  const handleProactive = useCallback(async () => {
    setProactiveBusy(true);
    try {
      const r = await app.WhisperProactiveNow(personalityId);
      if (!r.shouldSend) {
        toast.show("现在没有合适的时机，轻语选择安静陪伴", "info");
        setProactive(null);
      } else {
        setProactive(r);
      }
    } catch (e) {
      toast.show(`主动关心评估失败：${String(e)}`, "warn");
    } finally {
      setProactiveBusy(false);
    }
  }, [personalityId, toast]);

  const layout = useMemo<LayoutResult | null>(() => {
    if (!graph || !hasQueried) return null;
    return layoutGraph(graph, queriedEntity, clampHops(hopsInput));
  }, [clampHops, graph, hasQueried, hopsInput, queriedEntity]);

  const proactiveLabel = proactive?.messageType
    ? MESSAGE_TYPE_LABELS[proactive.messageType] ?? proactive.messageType
    : "主动关心";

  return (
    <Modal
      title={
        <span className="flex items-center gap-2">
          <ShareAltOutlined className="text-pink-400" />
          关系图谱
        </span>
      }
      open={open}
      onCancel={onClose}
      footer={null}
      width={760}
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
    >
      <div className="space-y-3">
        {/* ── 搜索/查询区 ── */}
        <div className="flex items-center gap-2">
          <input
            value={entityInput}
            onChange={(e) => setEntityInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") void handleQuery();
            }}
            aria-label="实体名"
            placeholder="实体名，如「阿黎」"
            className="w-56 px-3 h-8 rounded-lg border border-border bg-bg text-fg text-[12px] placeholder:text-fg-faint outline-none focus:border-accent transition-colors"
          />
          <label className="text-fg-faint text-[11px]">跳数</label>
          <input
            type="number"
            min={1}
            max={3}
            value={hopsInput}
            onChange={(e) => setHopsInput(e.target.value)}
            aria-label="跳数"
            placeholder="1-3"
            className="w-16 px-2 h-8 rounded-lg border border-border bg-bg text-fg text-[12px] placeholder:text-fg-faint outline-none focus:border-accent transition-colors"
          />
          <button
            className="inline-flex items-center gap-1 px-3 h-8 rounded-lg bg-accent text-accent-fg text-[12px] font-medium hover:brightness-110 transition-all disabled:opacity-40"
            onClick={() => void handleQuery()}
            disabled={loading}
          >
            <ShareAltOutlined style={{ fontSize: 12 }} />
            查询
          </button>
          {loading && (
            <span className="inline-flex items-center gap-1 text-fg-faint text-[11px]">
              <Loader size={12} className="animate-spin" />
              查询中…
            </span>
          )}
        </div>

        {/* ── 图谱渲染区 ── */}
        <div className="rounded-lg border border-border bg-bg-soft/40 min-h-[300px]">
          {!hasQueried ? (
            <EmptyState message="输入实体并点击「查询」，查看轻语记忆中的关系图谱" />
          ) : error ? (
            <div
              role="alert"
              className="m-3 px-3 py-3 rounded-lg border border-err/40 bg-err/10 text-fg-dim text-[11.5px]"
            >
              <span className="text-err font-medium">图谱查询失败：{error}</span>
            </div>
          ) : !graph || (graph.nodes ?? []).length === 0 ? (
            <EmptyState message="图谱暂无关系——多聊几次、让轻语记住更多约定与回忆" />
          ) : layout ? (
            <>
              {/* 情绪图例（v4.9 图谱情绪维度） */}
              <div className="flex items-center gap-3 px-3 pt-2 text-[10.5px] text-fg-faint">
                <span className="inline-flex items-center gap-1">
                  <i className="w-2 h-2 rounded-full bg-emerald-400 inline-block" />
                  正面
                </span>
                <span className="inline-flex items-center gap-1">
                  <i className="w-2 h-2 rounded-full bg-rose-400 inline-block" />
                  负面
                </span>
                <span className="inline-flex items-center gap-1">
                  <i className="w-2 h-2 rounded-full bg-fg-faint/60 inline-block" />
                  中性
                </span>
                <span className="inline-flex items-center gap-1">
                  <i className="w-2.5 h-0 border-t-2 border-dashed border-amber-400 inline-block" />
                  因果
                </span>
              </div>
              <svg viewBox={`0 0 ${SVG_W} ${SVG_H}`} className="w-full h-auto block">
              {/* 边：线 + 关系类型标签（中点） */}
              {layout.edges.map((pe, i) => (
                <g key={`edge-${i}`}>
                    <line
                      x1={pe.x1}
                      y1={pe.y1}
                      x2={pe.x2}
                      y2={pe.y2}
                      className={pe.edge.type === "因果" ? "stroke-amber-400/80" : edgeStrokeClass(pe.edge.emotionLabel)}
                      strokeWidth={1}
                      strokeDasharray={pe.edge.type === "因果" ? "4 2" : undefined}
                    />
                  {pe.edge.type && (
                    <text
                      x={(pe.x1 + pe.x2) / 2}
                      y={(pe.y1 + pe.y2) / 2 - 4}
                      textAnchor="middle"
                      className="fill-fg-faint"
                      fontSize={8}
                    >
                      {pe.edge.type}
                    </text>
                  )}
                </g>
              ))}
              {/* 节点：圆（type 着色 + 权重描边）+ 实体名；点击以该节点为中心重新查询 */}
              {layout.placed.map((p) => {
                const isCenter = p.node.id === layout.centerId;
                const r = isCenter ? 32 : 20 + Math.min(10, p.node.weight * 8);
                return (
                  <g
                    key={p.node.id}
                    className="cursor-pointer"
                    aria-label={`以「${p.node.name}」为中心重新查询`}
                    onClick={() => {
                      setEntityInput(p.node.name);
                      void handleQuery(p.node.name);
                    }}
                  >
                    <circle
                      cx={p.x}
                      cy={p.y}
                      r={r}
                      className={`${typeFill(p.node.type)} ${nodeStrokeClass(isCenter, p.node.weight)}`}
                      strokeWidth={nodeStrokeWidth(isCenter, p.node.weight)}
                    />
                    <text
                      x={p.x}
                      y={p.y + r + 12}
                      textAnchor="middle"
                      className={isCenter ? "fill-fg font-semibold" : "fill-fg"}
                      fontSize={10.5}
                    >
                      {shortLabel(p.node.name)}
                    </text>
                  </g>
                );
              })}
              </svg>
            </>
          ) : null}
        </div>

        {/* ── 主动关心区 ── */}
        <div className="pt-3 border-t border-border">
          <div className="flex items-center gap-2">
            <button
              className="inline-flex items-center gap-1 px-3 h-8 rounded-lg border border-border text-fg-dim hover:text-fg hover:border-accent hover:bg-bg-soft transition-colors text-[12px] disabled:opacity-40"
              onClick={() => void handleProactive()}
              disabled={proactiveBusy}
            >
              <SoundOutlined style={{ fontSize: 12 }} />
              轻语先开口
            </button>
            <span className="text-fg-faint text-[11px]">
              主动关心评估：轻语依据最近的记忆与情绪判断是否该先开口
            </span>
          </div>
          {proactiveBusy && (
            <div className="mt-2 flex items-center gap-1 text-fg-faint text-[11px]">
              <Loader size={12} className="animate-spin" />
              评估中…
            </div>
          )}
          {proactive && (
            <div className="mt-2 p-2.5 rounded-lg border border-border bg-bg-soft/60">
              <span className="inline-flex items-center px-1.5 py-0.5 rounded bg-pink-400/20 text-pink-300 text-[10.5px] font-medium">
                {proactiveLabel}
              </span>
              {proactive.promptHint && (
                <div className="mt-1.5 text-fg-dim text-[12.5px] leading-relaxed">
                  {proactive.promptHint}
                </div>
              )}
              <div className="mt-1 text-fg-faint text-[10.5px]">
                手动评估结果；定时推送到达时也会在这里显示（可在设置中关闭）
              </div>
            </div>
          )}
        </div>
      </div>
    </Modal>
  );
}
