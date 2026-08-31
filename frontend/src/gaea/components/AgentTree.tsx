import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { ReactNode } from "react";
import { ChevronDown, ChevronRight } from "../icons";
import { app } from "../lib/bridge";
import type { AgentNetwork, AgentNode, SubagentRunView, SubagentTranscriptView } from "../lib/types";
import { fmtTokens } from "../lib/stats";

// AgentTree — 子代理树实时拓扑（v4.24 A1「分工面板工作台化」核心①）。
// 以 GaeaAgentNetwork 的嵌套 Children 渲染整棵子代理树（此前 AgentNetworkCard
// 只画两层，深层子树数据被丢弃）：
//  - root 折叠为「主 agent」行；一级子代理始终可见，更深层默认收起、可展开/收起；
//  - 新节点出现自动展开其父链（本次挂载生命周期内未见过的节点 id 即「新」，
//    首轮轮询只记基线不展开）；
//  - 节点行量化：状态色点（running 呼吸动画）/任务摘要/工具调用数/模型徽标/
//    耗时（running 显示实时已用时——组件内 1s tick；已完成显示总耗时）/
//    token 数（有则显）/错误数；
//  - 运行节点用 SubagentRuns 的 lastText/lastTool 富化为实时预览行
//    （按 ref 直等或任务摘要前缀双向匹配，与后端 enrichAgentNetwork 同口径；
//    匹配失败降级纯节点统计）；
//  - 下钻链：节点点击 → 详情卡（统计/回答）→「查看完整 transcript」→ 消息流
//    （#N 序号 + 搜索过滤沿用 AgentNetworkCard 渲染模式）→ 工具调用行可点击，
//    scrollIntoView 定位到 toolCallId 匹配的 tool 结果消息（v4.24 新增）。
// transcript 渲染为复制精简而非抽公共组件：AgentNetworkCard 本轮不允许改动，
// 无改动权就无抽取落点；单一消费方先内聚，后续允许动旧卡时再合并。

// 树展平条目：ancestors 供「新节点自动展开父链」用（含 root id，不含自身）。
interface FlatNode {
  node: AgentNode;
  depth: number;
  ancestors: string[];
}

function flattenTree(node: AgentNode, depth: number, ancestors: string[], out: FlatNode[]): void {
  out.push({ node, depth, ancestors });
  for (const child of node.children ?? []) {
    flattenTree(child, depth + 1, [...ancestors, node.id], out);
  }
}

// 节点 → 分工 meta 匹配：先 ref 直等（后端若以 ref 作节点 id），再任务摘要
// 前缀双向（与 AgentNetworkCard.selectNode / 后端富化同口径）。
function matchRunForNode(node: AgentNode, runs: SubagentRunView[]): SubagentRunView | null {
  for (const r of runs) {
    if (r.ref && node.id && r.ref === node.id) return r;
  }
  for (const r of runs) {
    if (node.task && r.task && (r.task.startsWith(node.task) || node.task.startsWith(r.task))) return r;
  }
  return null;
}

function statusColor(status: AgentNode["status"]): string {
  switch (status) {
    case "running":
      return "var(--gaea-glow)";
    case "error":
      return "var(--md-sys-color-destructive)";
    default:
      return "var(--md-sys-color-success)";
  }
}

function statusText(status: AgentNode["status"]): string {
  switch (status) {
    case "running":
      return "进行中";
    case "error":
      return "出错";
    default:
      return "已完成";
  }
}

// 耗时格式化（与 SubagentsPanel.fmtDuration 同语义：秒/分/小时三档）。
function fmtDur(ms: number): string {
  if (!Number.isFinite(ms) || ms <= 0) return "";
  const s = Math.max(1, Math.round(ms / 1000));
  if (s < 60) return `${s} 秒`;
  const m = Math.floor(s / 60);
  if (m < 60) return `${m} 分`;
  return `${Math.floor(m / 60)} 小时`;
}

// 节点耗时：优先节点 firstTs/lastTs（后端权威），缺省回落匹配 run 的
// createdAt/updatedAt（ISO 串前端可算）。running 显示实时已用时（live=true
// 走 1s tick），已完成/出错显示总耗时。
function nodeDuration(node: AgentNode, run: SubagentRunView | null, nowMs: number): { label: string; live: boolean } {
  const nodeStart = node.firstTs !== undefined ? node.firstTs * 1000 : NaN;
  const runStart = run ? Date.parse(run.createdAt) : NaN;
  const start = Number.isFinite(nodeStart) ? nodeStart : runStart;
  if (node.status === "running") {
    if (!Number.isFinite(start)) return { label: "", live: false };
    return { label: `已用 ${fmtDur(nowMs - start)}`, live: true };
  }
  const nodeEnd = node.lastTs !== undefined ? node.lastTs * 1000 : NaN;
  const runEnd = run ? Date.parse(run.updatedAt) : NaN;
  const end = Number.isFinite(nodeEnd) ? nodeEnd : runEnd;
  if (!Number.isFinite(start) || !Number.isFinite(end) || end < start) return { label: "", live: false };
  return { label: fmtDur(end - start), live: false };
}

const iconBtn =
  "flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-(color:--md-sys-color-text-secondary) cursor-pointer hover:text-(color:--md-sys-color-text) hover:bg-(color:--md-sys-color-surface-container-high) transition-colors";

// ─── transcript 消息流（精简自 AgentNetworkCard；新增工具行点击定位） ───
function TranscriptViewer({ transcript }: { transcript: SubagentTranscriptView }) {
  const [query, setQuery] = useState("");
  const [locateIdx, setLocateIdx] = useState<number | null>(null);
  const listRef = useRef<HTMLDivElement | null>(null);

  // 搜索过滤（正文/推理/工具调用/工具结果全字段），命中带原索引保证 #N 稳定。
  const shown = useMemo(() => {
    const all = transcript.messages.map((m, idx) => ({ m, idx }));
    const q = query.trim().toLowerCase();
    if (!q) return all;
    return all.filter(({ m }) =>
      [m.content, m.reasoning, m.name, m.toolCallId, ...(m.toolCalls ?? []).map((tc) => `${tc.name} ${tc.arguments}`)]
        .filter(Boolean)
        .join("\n")
        .toLowerCase()
        .includes(q),
    );
  }, [transcript, query]);

  // 搜索命中 → 自动定位第一条命中消息（与 AgentNetworkCard 的 data-msg-hit 同语义）。
  useEffect(() => {
    if (!query.trim()) return;
    setLocateIdx(shown.length > 0 ? shown[0].idx : null);
  }, [query, shown]);

  // 定位滚动：jsdom 无布局，单测以 scrollIntoView stub 调用断言。
  useEffect(() => {
    if (locateIdx === null) return;
    listRef.current
      ?.querySelector(`[data-msg-idx="${locateIdx}"]`)
      ?.scrollIntoView({ behavior: "smooth", block: "start" });
  }, [locateIdx]);

  // v4.24 新增：工具调用行点击 → 找到 toolCallId 匹配的 tool 结果消息 → 定位。
  const locateToolCall = useCallback(
    (toolCallId: string) => {
      const idx = transcript.messages.findIndex((m) => m.toolCallId === toolCallId);
      if (idx >= 0) setLocateIdx(idx);
    },
    [transcript],
  );

  return (
    <div
      data-testid="agent-transcript"
      className="mt-1.5 flex flex-col rounded-md px-2 py-1.5"
      style={{ background: "var(--md-sys-color-surface-container)" }}
    >
      <div className="mb-1 flex items-center gap-1.5">
        <span className="min-w-0 flex-1 truncate text-[10px] font-medium" style={{ color: "var(--md-sys-color-text)" }}>
          {transcript.task || transcript.ref}
        </span>
        <input
          className="w-28 rounded-md border-0 px-1.5 py-0.5 text-[10px] outline-none"
          style={{ background: "var(--md-sys-color-surface-container-high)", color: "var(--md-sys-color-text)" }}
          placeholder="搜索消息"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <span className="shrink-0 font-mono text-[9px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
          {shown.length}/{transcript.messages.length} 条
        </span>
      </div>
      <div ref={listRef} className="flex max-h-64 flex-col gap-1 overflow-y-auto">
        {shown.length === 0 && (
          <div className="py-2 text-[10px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
            没有匹配的消息
          </div>
        )}
        {shown.map(({ m, idx }, i) => (
          <div
            key={idx}
            data-msg-idx={idx}
            data-msg-hit={query.trim() && i === 0 ? "true" : undefined}
            data-located={locateIdx === idx ? "true" : undefined}
            className="rounded-md px-1.5 py-1 text-[10px]"
            style={{
              background: locateIdx === idx
                ? "color-mix(in srgb, var(--gaea-glow) 10%, transparent)"
                : "var(--md-sys-color-surface-container-high)",
            }}
          >
            <div className="flex items-center gap-1.5" style={{ color: "var(--md-sys-color-text-secondary)" }}>
              <span className="shrink-0 font-mono tabular-nums">#{idx + 1}</span>
              <span
                className="shrink-0 rounded px-1 text-[9px] font-medium"
                style={{ background: "color-mix(in srgb, var(--md-sys-color-text) 8%, transparent)", color: "var(--md-sys-color-text)" }}
              >
                {m.role.toUpperCase()}
              </span>
              {m.name && <span className="font-mono">{m.name}</span>}
              {m.toolCallId && <span className="truncate font-mono" title={`工具结果 ${m.toolCallId}`}>{m.toolCallId}</span>}
            </div>
            {m.reasoning && (
              <pre className="mt-0.5 whitespace-pre-wrap break-words font-mono text-[9.5px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                {m.reasoning}
              </pre>
            )}
            {m.toolCalls && m.toolCalls.length > 0 && (
              <div className="mt-0.5 flex flex-col items-stretch">
                {m.toolCalls.map((tc, j) => (
                  <button
                    key={j}
                    type="button"
                    data-testid="agent-transcript-toolcall"
                    className="mt-px flex cursor-pointer items-center gap-1 rounded border-0 px-1 py-px text-left font-mono text-[9.5px] transition-colors"
                    style={{
                      background: "color-mix(in srgb, var(--gaea-glow) 8%, transparent)",
                      color: "var(--md-sys-color-text-secondary)",
                    }}
                    title="点击定位该工具调用的结果消息"
                    onClick={() => locateToolCall(tc.id)}
                  >
                    <span aria-hidden>⚙</span>
                    <span className="truncate">{tc.name} {tc.arguments}</span>
                  </button>
                ))}
              </div>
            )}
            {m.content && (
              <pre className="mt-0.5 whitespace-pre-wrap break-words font-mono text-[9.5px]" style={{ color: "var(--md-sys-color-text)" }}>
                {m.content}
              </pre>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

// ─── 子代理树 ──────────────────────────────────────────────
export function AgentTree({ network, runs, sessionPath }: {
  network: AgentNetwork;
  runs: SubagentRunView[];
  sessionPath?: string;
}) {
  const root = network.root;

  // 展平一次供逻辑用（匹配表/新节点检测/hasRunning），渲染走递归保结构。
  const flat = useMemo(() => {
    const out: FlatNode[] = [];
    flattenTree(root, 0, [], out);
    return out;
  }, [root]);

  // 节点 → 分工 meta 匹配表：单轮轮询算全表，避免逐节点重复扫描。
  const runByNodeId = useMemo(() => {
    const map = new Map<string, SubagentRunView | null>();
    for (const f of flat) map.set(f.node.id, matchRunForNode(f.node, runs));
    return map;
  }, [flat, runs]);

  // 展开态：root 的直接子级恒可见（depth<=1），更深层按展开集合放行。
  const [expanded, setExpanded] = useState<Set<string>>(() => new Set<string>());
  const toggleNode = useCallback((id: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  // 新节点出现自动展开其父链：本次挂载未见过的节点 id 即「新」；
  // 首轮轮询只记基线不展开（挂载时已存在的子代理不算新）。
  const seenIdsRef = useRef<Set<string> | null>(null);
  useEffect(() => {
    const ids = new Set(flat.map((f) => f.node.id));
    const seen = seenIdsRef.current;
    seenIdsRef.current = ids;
    if (!seen) return;
    const fresh = flat.filter((f) => !seen.has(f.node.id));
    if (fresh.length === 0) return;
    setExpanded((prev) => {
      const next = new Set(prev);
      for (const f of fresh) {
        for (const a of f.ancestors) next.add(a);
      }
      return next;
    });
  }, [flat]);

  // 详情选中态 + transcript 下钻状态（切换节点即清空，防串卡）。
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [transcript, setTranscript] = useState<SubagentTranscriptView | null>(null);
  const [transcriptNodeId, setTranscriptNodeId] = useState<string | null>(null);
  const [transcriptBusy, setTranscriptBusy] = useState(false);
  const selectNode = useCallback((id: string) => {
    setSelectedId((prev) => (prev === id ? null : id));
    setTranscript(null);
    setTranscriptNodeId(null);
  }, []);

  // running 节点存在时启动 1s tick：实时已用时走秒（无运行节点零开销）。
  const hasRunning = useMemo(() => flat.some((f) => f.node.status === "running"), [flat]);
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!hasRunning) return;
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, [hasRunning]);

  // 查看完整 transcript：拉取该子代理的 transcript JSONL 投影（再点收起）。
  const viewTranscript = useCallback(
    (nodeId: string, ref: string) => {
      if (!sessionPath) return;
      if (transcriptNodeId === nodeId && transcript?.ref === ref) {
        setTranscript(null);
        setTranscriptNodeId(null);
        return;
      }
      setTranscriptBusy(true);
      app
        .SubagentTranscript(sessionPath, ref)
        .then((v) => {
          setTranscript(v);
          setTranscriptNodeId(nodeId);
        })
        .catch(() => {
          setTranscript(null);
          setTranscriptNodeId(null);
        })
        .finally(() => setTranscriptBusy(false));
    },
    [sessionPath, transcript, transcriptNodeId],
  );

  const renderNode = (f: FlatNode): ReactNode => {
    const node = f.node;
    const run = runByNodeId.get(node.id) ?? null;
    const dur = nodeDuration(node, run, now);
    const title = node.kind === "root" ? "主 agent" : node.task || node.name;
    const childEntries = node.children ?? [];
    const hasChildren = childEntries.length > 0;
    const isExpanded = expanded.has(node.id);
    const isSelected = selectedId === node.id;
    // transcript 引用：优先匹配 run 的 ref；后端若以 ref 作节点 id 也可直用。
    const transcriptRef = run?.ref ?? (node.id.startsWith("sa_") ? node.id : null);
    const dotCls = `inline-block h-1.5 w-1.5 shrink-0 rounded-full${node.status === "running" ? " animate-pulse" : ""}`;
    return (
      <div
        key={node.id}
        data-testid="agent-node"
        data-node-id={node.id}
        className="flex flex-col rounded-[var(--radius-md)] transition-all duration-200"
        style={{
          background: "var(--md-sys-color-surface-container)",
          border: `1px solid ${isSelected ? "color-mix(in srgb, var(--gaea-glow) 40%, transparent)" : "var(--md-sys-color-outline-variant)"}`,
          marginLeft: f.depth * 10,
        }}
      >
        <div className="flex items-center gap-0.5 py-1 pr-1" style={{ paddingLeft: 4 }}>
          {hasChildren ? (
            <button
              type="button"
              className={iconBtn}
              data-testid="agent-node-toggle"
              aria-label={`${isExpanded ? "收起" : "展开"} ${title}`}
              aria-expanded={isExpanded}
              onClick={() => toggleNode(node.id)}
            >
              {isExpanded ? <ChevronDown size={12} aria-hidden /> : <ChevronRight size={12} aria-hidden />}
            </button>
          ) : (
            <span className="w-6 shrink-0" aria-hidden />
          )}
          <button
            type="button"
            data-testid="agent-node-row"
            onClick={() => selectNode(node.id)}
            className="flex min-w-0 flex-1 cursor-pointer items-center gap-1.5 border-0 bg-transparent p-0 text-left"
            title={`${statusText(node.status)} · ${title}`}
          >
            <span className={dotCls} aria-hidden style={{ background: statusColor(node.status) }} />
            <span className="min-w-0 flex-1 truncate text-[11.5px] font-medium" style={{ color: "var(--md-sys-color-text)" }}>
              {title}
            </span>
            {node.errors > 0 && (
              <span className="shrink-0 text-[9.5px]" style={{ color: "var(--md-sys-color-destructive)" }}>错 {node.errors}</span>
            )}
            <span className="shrink-0 font-mono text-[9.5px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
              ⚙{node.toolCalls}
            </span>
            {(node.model || run?.model) && (
              <span className="shrink-0 rounded px-1 py-px font-mono text-[9px]" style={{ background: "var(--md-sys-color-surface-container-high)" }}>
                {node.model || run?.model}
              </span>
            )}
            {dur.label && (
              <span
                className="shrink-0 font-mono text-[9.5px]"
                style={{ color: dur.live ? "var(--gaea-glow)" : "var(--md-sys-color-text-secondary)" }}
              >
                {dur.label}
              </span>
            )}
            {node.tokens > 0 && (
              <span className="shrink-0 font-mono text-[9.5px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                ≈{fmtTokens(node.tokens)}
              </span>
            )}
          </button>
        </div>

        {/* C2 活动行迁入树内：运行节点的实时预览（匹配失败时整块省略 = 纯节点统计降级） */}
        {node.status === "running" && run && (run.lastText || run.lastTool) && (
          <div
            className="mx-1 mb-1 flex flex-col gap-px rounded-md px-1.5 py-1 text-[10px] leading-relaxed"
            style={{
              background: "color-mix(in srgb, var(--gaea-glow) 6%, transparent)",
              border: "1px solid color-mix(in srgb, var(--gaea-glow) 16%, transparent)",
            }}
          >
            {run.lastText && (
              <span className="truncate" title={run.lastText} style={{ color: "var(--md-sys-color-text)" }}>
                <span className="mr-1 inline-block h-1 w-1 rounded-full align-middle animate-pulse" style={{ background: "var(--gaea-glow)" }} aria-hidden />
                正在：{run.lastText}
              </span>
            )}
            {run.lastTool && (
              <span className="truncate font-mono" title={run.lastTool} style={{ color: "var(--md-sys-color-text-secondary)" }}>
                ⚙ {run.lastTool}
              </span>
            )}
          </div>
        )}

        {/* 详情卡（沿用展开卡片模式）：统计/回答/transcript 下钻 */}
        {isSelected && (
          <div
            data-testid="agent-detail"
            className="mx-1 mb-1 flex flex-col gap-1 rounded-md px-2 py-1.5 text-[10.5px]"
            style={{ background: "color-mix(in srgb, var(--md-sys-color-text) 5%, transparent)" }}
          >
            <div className="flex items-center gap-2">
              <span className="min-w-0 flex-1 truncate font-medium" style={{ color: "var(--md-sys-color-text)" }}>{title}</span>
              <span className="shrink-0" style={{ color: statusColor(node.status) }}>{statusText(node.status)}</span>
              <button
                type="button"
                className="shrink-0 cursor-pointer border-0 bg-transparent p-0 text-[10px]"
                style={{ color: "var(--md-sys-color-text-secondary)" }}
                onClick={() => selectNode(node.id)}
              >
                关闭
              </button>
            </div>
            <div className="flex flex-wrap gap-x-2 gap-y-0.5" style={{ color: "var(--md-sys-color-text-secondary)" }}>
              {(node.model || run?.model) && <span className="font-mono">{node.model || run?.model}</span>}
              <span className="tabular-nums">工具 {node.toolCalls}</span>
              {node.errors > 0 && <span className="tabular-nums" style={{ color: "var(--md-sys-color-destructive)" }}>错误 {node.errors}</span>}
              {node.tokens > 0 && <span className="font-mono tabular-nums">≈{fmtTokens(node.tokens)}</span>}
              {dur.label && <span className="font-mono tabular-nums">{dur.label}</span>}
              {run && <span className="font-mono">{run.ref.slice(0, 24)}…</span>}
            </div>
            {run?.lastText && (
              <div className="truncate" title={run.lastText} style={{ color: "var(--md-sys-color-text)" }}>最后活动：{run.lastText}</div>
            )}
            {run?.lastTool && (
              <div className="truncate font-mono" title={run.lastTool} style={{ color: "var(--md-sys-color-text-secondary)" }}>⚙ {run.lastTool}</div>
            )}
            {run?.answer && (
              <div
                className="whitespace-pre-wrap break-words rounded-md px-1.5 py-1 text-[10.5px] leading-relaxed"
                style={{ background: "var(--md-sys-color-surface-container-high)", color: "var(--md-sys-color-text)" }}
              >
                {run.answer}
              </div>
            )}
            {transcriptRef && sessionPath && (
              <button
                type="button"
                data-testid="agent-detail-transcript"
                className="cursor-pointer self-start rounded-md border-0 px-2 py-1 text-[10px] transition-colors"
                style={{ background: "var(--md-sys-color-surface-container-high)", color: "var(--gaea-glow)" }}
                onClick={() => viewTranscript(node.id, transcriptRef)}
              >
                {transcriptBusy && transcriptNodeId !== node.id
                  ? "读取中…"
                  : transcriptNodeId === node.id && transcript
                    ? "收起完整 transcript"
                    : "查看完整 transcript"}
              </button>
            )}
            {transcriptNodeId === node.id && transcript && (
              <TranscriptViewer key={transcript.ref} transcript={transcript} />
            )}
          </div>
        )}

        {/* 子树：root 直接子级恒可见，更深层按展开态渲染（收起即不挂载） */}
        {(f.depth === 0 || isExpanded) &&
          childEntries.map((child) => renderNode({ node: child, depth: f.depth + 1, ancestors: [...f.ancestors, node.id] }))}
      </div>
    );
  };

  return (
    <div className="flex flex-col gap-1" data-testid="agent-tree">
      {renderNode({ node: root, depth: 0, ancestors: [] })}
    </div>
  );
}
