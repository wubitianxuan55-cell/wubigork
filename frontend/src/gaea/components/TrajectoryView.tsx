import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Bot, ChevronDown, ChevronRight, Clock, Cpu, Eye, MessageSquare, Search, Shield, User, Wrench } from "../icons";
import { app } from "../lib/bridge";
import { List, useDynamicRowHeight, useListRef, type RowComponentProps } from "react-window";
import type {
  Trajectory, TrajectoryAssistantRec, TrajectoryHeaderRec, TrajectoryRecord,
  TrajectorySubagentRec, TrajectoryToolRec, TrajectoryTurn,
} from "../lib/types";
import { fmtTokens } from "../lib/stats";
import { formatElapsed } from "../lib/time";
import { useLiveReload } from "../hooks/useLiveReload";

// TrajectoryView.tsx — 轨迹事件账本（Why：长会话需按时间序审计每次请求/调用/
// 答复，v4.26 起子代理最终答复以 kind="subagent" 记录折叠进回合，前端需与
// 既有记录同视觉语言呈现；How：Trajectory() 拉取折叠记录 → 扁平行流（轮次头
// +记录行）→ react-window 动态行高虚拟渲染，记录行折叠态单行摘要、点击展开
// RecordInspector 看全文/参数/用量）。

const EMPTY: Trajectory = { ok: true, turns: [] };

function recordMatches(r: TrajectoryRecord, q: string): boolean {
  const hay = [r.user?.text, r.assistant?.text, r.assistant?.reasoning, r.tool?.name, r.tool?.args, r.tool?.output, r.tool?.err, r.ask?.question, r.approval?.subject, r.compact?.summary, r.subagent?.text, r.subagent?.ref].filter(Boolean).join("\n").toLowerCase();
  return hay.includes(q);
}

type FlatRow =
  | { kind: "turn"; key: string; turn: TrajectoryTurn; shown: number; open: boolean }
  | { kind: "between"; key: string }
  | { kind: "record"; key: string; record: TrajectoryRecord };

// 虚拟化行数据（rowProps 透传；open 由 openTurns/allCollapsed 在行内派生）。
interface TrajectoryRowData {
  rows: FlatRow[];
  openTurns: Set<number>;
  allCollapsed: boolean;
  onToggle: (turn: number) => void;
}

function BetweenTurnsHeader() {
  return (
    <div className="flex items-center gap-2 border-y border-dashed border-border-soft bg-warning/5 px-3 py-1.5">
      <span className="text-[10px] font-semibold text-warning uppercase tracking-wider">Between turns</span>
      <span className="text-[9px] text-fg-faint">轮次之间</span>
    </div>
  );
}

// 虚拟化行组件：react-window v2 动态行高模式下 style 不含 height，行高由
// ResizeObserver 实测（行内容变化自动重排；jsdom 无布局时回落 defaultRowHeight）。
function TrajectoryRow({ index, style, rows, openTurns, allCollapsed, onToggle }: RowComponentProps<TrajectoryRowData>) {
  const row = rows[index];
  if (!row) return null;
  return (
    <div style={style} className="bg-bg">
      {row.kind === "turn" ? (
        <TurnHeader
          turn={row.turn}
          shown={row.shown}
          open={allCollapsed ? openTurns.has(row.turn.turn) : !openTurns.has(row.turn.turn)}
          onToggle={() => onToggle(row.turn.turn)}
        />
      ) : row.kind === "between" ? (
        <BetweenTurnsHeader />
      ) : (
        <RecordRow record={row.record} />
      )}
    </div>
  );
}

function fmtTime(ts?: number): string {
  return ts ? new Date(ts * 1000).toLocaleTimeString() : "—";
}

function fmtDuration(ms?: number): string {
  if (!ms || ms <= 0) return "";
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

// 记录类型徽标（对齐效果图：ASSISTANT 紫 / TOOL 橙 / ask 深蓝）。
function KindBadge({ kind }: { kind: TrajectoryRecord["kind"] }) {
  const map: Record<TrajectoryRecord["kind"], { label: string; cls: string }> = {
    assistant: { label: "ASSISTANT", cls: "bg-purple-500/15 text-purple-400" },
    tool: { label: "TOOL", cls: "bg-orange-500/15 text-orange-400" },
    ask: { label: "提问", cls: "bg-blue-800/25 text-blue-300" },
    approval: { label: "审批", cls: "bg-amber-500/15 text-amber-400" },
    header: { label: "REQUEST HEADER", cls: "bg-cyan-500/15 text-cyan-400" },
    compact: { label: "COMPACTION", cls: "bg-yellow-500/15 text-yellow-400" },
    user: { label: "用户", cls: "bg-emerald-500/15 text-emerald-400" },
    subagent: { label: "子代理", cls: "bg-accent/15 text-accent" },
  };
  const m = map[kind];
  return <span className={`shrink-0 rounded px-1.5 py-0.5 text-[9px] font-semibold tracking-wide ${m.cls}`}>{m.label}</span>;
}

function KindIcon({ kind }: { kind: TrajectoryRecord["kind"] }) {
  switch (kind) {
    case "assistant": return <MessageSquare size={12} className="text-purple-400" />;
    case "tool": return <Wrench size={12} className="text-orange-400" />;
    case "ask": return <MessageSquare size={12} className="text-blue-300" />;
    case "approval": return <Shield size={12} className="text-amber-400" />;
    case "header": return <Cpu size={12} className="text-cyan-400" />;
    case "compact": return <Cpu size={12} className="text-yellow-400" />;
    case "subagent": return <Bot size={12} className="text-accent" />;
    default: return <User size={12} className="text-emerald-400" />;
  }
}

// ─── 记录详情（检查器） ──────────────────────────────────────
function RecordInspector({ record }: { record: TrajectoryRecord }) {
  const meta: string[] = [];
  if (record.durationMs) meta.push(`耗时 ${fmtDuration(record.durationMs)}`);
  if (record.step) meta.push(`步骤 ${record.step}`);
  if (record.tool?.status === "running") meta.push("运行中");
  return (
    <div className="mt-1 rounded-md border border-border-soft bg-bg-soft/50 p-2 text-[10px]">
      {meta.length > 0 && <div className="mb-1 flex flex-wrap gap-2 text-fg-faint tabular-nums font-mono">{meta.map((m) => <span key={m}>{m}</span>)}</div>}
      {record.kind === "header" && record.header && <HeaderDetail h={record.header} />}
      {record.kind === "assistant" && record.assistant && <AssistantDetail a={record.assistant} />}
      {record.kind === "tool" && record.tool && <ToolDetail t={record.tool} />}
      {record.kind === "ask" && <div className="text-fg">{record.ask?.question || "—"}</div>}
      {record.kind === "approval" && (
        <div className="text-fg-dim">{record.approval?.tool}{record.approval?.subject ? ` · ${record.approval.subject}` : ""}</div>
      )}
      {record.kind === "compact" && <div className="text-warning/90">{record.compact?.summary || record.compact?.trigger || "压缩"}</div>}
      {record.kind === "subagent" && record.subagent && <SubagentDetail s={record.subagent} />}
    </div>
  );
}

function HeaderDetail({ h }: { h: TrajectoryHeaderRec }) {
  return (
    <div>
      <div className="flex flex-wrap gap-2 text-fg-dim tabular-nums font-mono">
        <span>工具 {h.toolCount} 项</span>
        <span>≈{fmtTokens(h.tokens)}</span>
        {h.change && <span className="text-cyan-400">{h.change}</span>}
      </div>
      {h.system && <pre className="mt-1 max-h-40 overflow-auto rounded bg-bg p-1.5 text-[10px] text-fg-dim whitespace-pre-wrap">{h.system}</pre>}
    </div>
  );
}

function AssistantDetail({ a }: { a: TrajectoryAssistantRec }) {
  return (
    <div>
      {a.usage && (
        <div className="mb-1 flex flex-wrap gap-2 text-fg-faint tabular-nums font-mono">
          <span>↑{fmtTokens(a.usage.promptTokens ?? 0)}</span>
          <span>↓{fmtTokens(a.usage.completionTokens ?? 0)}</span>
          {a.usage.cacheHitTokens ? <span>缓存 {fmtTokens(a.usage.cacheHitTokens)}</span> : null}
        </div>
      )}
      {a.reasoning && <pre className="mt-1 rounded bg-bg p-1.5 text-[10px] text-fg-dim whitespace-pre-wrap border-l-2 border-info/40">{a.reasoning}</pre>}
      {a.text && <pre className="mt-1 rounded bg-bg p-1.5 text-[10px] text-fg whitespace-pre-wrap">{a.text}</pre>}
    </div>
  );
}

function ToolDetail({ t }: { t: TrajectoryToolRec }) {
  return (
    <div>
      <div className="text-fg font-mono">{t.name}</div>
      {t.args && <pre className="mt-1 max-h-40 overflow-auto rounded bg-bg p-1.5 text-[10px] text-fg-dim whitespace-pre-wrap">{t.args}</pre>}
      {t.output && <pre className={`mt-1 max-h-52 overflow-auto rounded bg-bg p-1.5 text-[10px] font-mono whitespace-pre-wrap ${t.status === "error" ? "text-err" : "text-fg-dim"}`}>{t.output}</pre>}
      {t.err && <div className="mt-1 text-[10px] text-err">{t.err}</div>}
      {t.truncated && <div className="mt-1 text-[9px] text-warning">结果已截断</div>}
      {t.parentId && <div className="mt-1 text-[9px] text-fg-faint font-mono">parentId: {t.parentId}</div>}
    </div>
  );
}

// SubagentDetail：子代理回投记录的展开详情——完整答复文本（后端展示级截断
// 2000 rune，全文在会话日志）+ transcript 引用 + 父 task 调用 ID（有则显示，
// 临时子代理无 ref 容错省略）。
function SubagentDetail({ s }: { s: TrajectorySubagentRec }) {
  return (
    <div>
      {s.ref && <div className="mt-1 text-[9px] text-fg-faint font-mono break-all">ref: {s.ref}</div>}
      {s.parentId && <div className="mt-1 text-[9px] text-fg-faint font-mono">parentId: {s.parentId}</div>}
      {s.text && <pre className="mt-1 rounded bg-bg p-1.5 text-[10px] text-fg whitespace-pre-wrap border-l-2 border-accent/40">{s.text}</pre>}
    </div>
  );
}

// ─── 单条记录行 ─────────────────────────────────────────────
function RecordRow({ record }: { record: TrajectoryRecord }) {
  const [open, setOpen] = useState(false);
  const badge = <KindBadge kind={record.kind} />;
  const summary = useMemo(() => {
    switch (record.kind) {
      case "user": return record.user?.text || "";
      case "header": return `${record.header?.toolCount ?? 0} 个工具 · ≈${fmtTokens(record.header?.tokens ?? 0)}`;
      case "assistant": return record.assistant?.text || (record.assistant?.reasoning ? "（仅推理）" : "");
      case "tool": {
        const t = record.tool!;
        const args = t.args ? t.args.slice(0, 80) + (t.args.length > 80 ? "…" : "") : "";
        const out = t.output ? ` → ${t.output.slice(0, 60)}${t.output.length > 60 ? "…" : ""}` : "";
        const err = t.err ? ` → ${t.err.slice(0, 60)}` : "";
        return `${t.name} ${args}${err || out}`;
      }
      case "compact": return `${record.compact?.trigger || "manual"}${record.compact?.summary ? ` — ${record.compact.summary.slice(0, 60)}` : ""}`;
      case "ask": return record.ask?.question || "";
      case "approval": return `${record.approval?.tool || ""}${record.approval?.subject ? ` · ${record.approval.subject}` : ""}`;
      // 子代理回投：折叠态弱存在感一行 = 答复文本（CSS truncate）+ ref（有则显示）
      case "subagent": return `${record.subagent?.text || "（无答复文本）"}${record.subagent?.ref ? ` · ${record.subagent.ref}` : ""}`;
      default: return "";
    }
  }, [record]);

  const canOpen = record.kind !== "user" || Boolean(record.user?.text && record.user.text.length > 80);
  const content = (
    <div className="flex min-w-0 flex-1 items-center gap-2 px-2 py-1.5">
      <KindIcon kind={record.kind} />
      {badge}
      <span className="truncate text-[10px] text-fg-dim">{summary}</span>
      <span className="ml-auto shrink-0 text-[9px] text-fg-faint tabular-nums font-mono">
        {fmtTime(record.ts)}
        {record.durationMs ? ` · ${fmtDuration(record.durationMs)}` : ""}
      </span>
    </div>
  );

  return (
    <div className="border-b border-border-soft/40 last:border-0">
      {canOpen ? (
        <button className="flex w-full items-center border-0 bg-transparent cursor-pointer hover:bg-bg-soft/40" onClick={() => setOpen(!open)}>
          {open ? <ChevronDown size={11} className="ml-1 shrink-0 text-fg-faint" /> : <ChevronRight size={11} className="ml-1 shrink-0 text-fg-faint" />}
          {content}
        </button>
      ) : (
        <div className="flex w-full items-center">{content}</div>
      )}
      {open && <div className="pl-6 pr-2 pb-1.5"><RecordInspector record={record} /></div>}
    </div>
  );
}

// ─── 轮次头（粗分割线；记录行由容器按扁平行流渲染） ─────────────
function TurnHeader({ turn, shown, open, onToggle }: {
  turn: TrajectoryTurn;
  shown: number;
  open: boolean;
  onToggle: () => void;
}) {
  return (
    <div id={`traj-turn-${turn.turn}`} className="scroll-mt-2">
      <button
        className="flex w-full items-center gap-2 border-y border-border-soft bg-bg-soft/60 px-3 py-1.5 text-left border-0 cursor-pointer"
        onClick={onToggle}
      >
        {open ? <ChevronDown size={12} className="text-fg-faint" /> : <ChevronRight size={12} className="text-fg-faint" />}
        <span className="text-[10px] font-semibold text-fg uppercase tracking-wider">第{turn.turn}轮</span>
        <span className="text-[9px] text-fg-faint tabular-nums font-mono">{fmtTime(turn.startedAt)}</span>
        {/* v4.31 轮级耗时：仅轮结束时（end 存在）且有 durationMs 才展示——
            当前 running 轮（无 end）维持现状不显示；旧后端无该字段也跳过。 */}
        {turn.end && turn.durationMs ? (
          <span className="text-[9px] text-fg-faint tabular-nums font-mono">用时 {formatElapsed(turn.durationMs / 1000)}</span>
        ) : null}
        {turn.end?.err && <span className="rounded bg-err/15 px-1 text-[9px] text-err">错误</span>}
        <span className="ml-auto text-[9px] text-fg-faint tabular-nums">{shown} 条记录</span>
      </button>
    </div>
  );
}

// ─── 轨迹概览（Overview 投影：每轮一根柱，点击跳转） ────────────
function OverviewBar({ turns, onJump }: { turns: TrajectoryTurn[]; onJump: (turn: number) => void }) {
  const maxRec = Math.max(1, ...turns.map((t) => t.records.length));
  const W = Math.max(turns.length * 9, 320);
  const H = 26;
  return (
    <div className="rounded-lg border border-border-soft bg-bg p-2.5">
      <div className="flex items-center justify-between">
        <div className="text-[10px] font-medium text-fg">轨迹概览</div>
        <div className="text-[9px] text-fg-faint tabular-nums">{turns.length} 轮 · 点击柱跳转</div>
      </div>
      <svg viewBox={`0 0 ${W} ${H}`} className="mt-1 w-full h-auto" style={{ width: W }}>
        {turns.map((t, i) => {
          const x = 2 + i * 9;
          const h = 8 + Math.round((t.records.length / maxRec) * 10);
          const tools = t.records.filter((r) => r.kind === "tool").length;
          const fill = t.end?.err ? "var(--color-err)" : tools > 0 ? "var(--color-accent)" : "var(--color-fg-faint)";
          return (
            <g key={t.turn} onClick={() => onJump(t.turn)} className="cursor-pointer">
              <rect x={x} y={H - h} width={5} height={h} rx={1} fill={fill} opacity={0.85}>
                <title>{`第${t.turn}轮 · ${t.records.length} 条记录 · ${tools} 工具调用${t.end?.err ? " · 报错" : ""}`}</title>
              </rect>
            </g>
          );
        })}
      </svg>
      <div className="mt-1 text-[9px] text-fg-faint">柱高 = 记录密度 · 高亮 = 含工具调用 · 红 = 该轮报错</div>
    </div>
  );
}

// ─── 容器：拉取 + 搜索 + 统计 + 布局 ────────────────────────
export function TrajectoryView({ running }: { running: boolean }) {
  const [trajectory, setTrajectory] = useState<Trajectory>(EMPTY);
  const [query, setQuery] = useState("");
  const [error, setError] = useState<string | null>(null);
  // 轮次展开态（缺省 = 展开；allCollapsed=true 时反转为「仅集合内展开」）。
  const [openTurns, setOpenTurns] = useState<Set<number>>(new Set());
  const [allCollapsed, setAllCollapsed] = useState(false);
  const listRef = useListRef();
  const rowHeights = useDynamicRowHeight({ defaultRowHeight: 29, key: "trajectory" });

  const load = useCallback(() => {
    app.Trajectory()
      // 老后端可能把空切片序列化成 null，按数组消费前统一归一化
      .then((t) => {
        setTrajectory({
          ...t,
          turns: (t.turns ?? []).map((tu) => ({ ...tu, records: tu.records ?? [] })),
          betweenTurns: t.betweenTurns ?? [],
        });
        setError(null);
      })
      .catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)));
  }, []);

  useEffect(() => { load(); }, [load]);
  // 运行中随事件流节流刷新 + 回合结束立即刷新（useLiveReload）。
  useLiveReload(running, load);

  const stats = useMemo(() => {
    const all = [...trajectory.turns.flatMap((t) => t.records), ...(trajectory.betweenTurns ?? [])];
    const calls = all.filter((r) => r.kind === "tool").length;
    const starts = all.map((r) => r.ts).filter(Boolean);
    const duration = starts.length > 1 ? Math.max(0, Math.max(...starts) - Math.min(...starts)) : 0;
    return { turns: trajectory.turns.length, calls, duration };
  }, [trajectory]);

  const isTurnOpen = (turn: number) => (allCollapsed ? openTurns.has(turn) : !openTurns.has(turn));
  const toggleTurn = (turn: number) => {
    setOpenTurns((prev) => {
      const next = new Set(prev);
      if (next.has(turn)) next.delete(turn);
      else next.add(turn);
      return next;
    });
  };
  const collapseAll = () => { setAllCollapsed(true); setOpenTurns(new Set()); };
  const expandAll = () => { setAllCollapsed(false); setOpenTurns(new Set()); };

  // 扁平行流：轮次头 + （展开的）记录行 + Between-turns；搜索命中轮次才保留。
  const flatRows = useMemo<FlatRow[]>(() => {
    const q = query.trim().toLowerCase();
    const rows: FlatRow[] = [];
    for (const t of trajectory.turns) {
      const shown = q ? t.records.filter((r) => recordMatches(r, q)) : t.records;
      if (q && shown.length === 0) continue;
      const open = isTurnOpen(t.turn);
      rows.push({ kind: "turn", key: `turn-${t.turn}`, turn: t, shown: shown.length, open });
      if (open) {
        for (const r of shown) rows.push({ kind: "record", key: `${r.kind}-${r.seq}`, record: r });
      }
    }
    if (trajectory.betweenTurns && trajectory.betweenTurns.length > 0) {
      rows.push({ kind: "between", key: "bt-header" });
      for (const r of trajectory.betweenTurns) {
        rows.push({ kind: "record", key: `bt-${r.kind}-${r.seq}`, record: r });
      }
    }
    return rows;
    // eslint-disable-next-line react-hooks/exhaustive-deps -- isTurnOpen 由 openTurns/allCollapsed 派生，已列入
  }, [trajectory, query, openTurns, allCollapsed]);

  const flatRowsRef = useRef(flatRows);
  flatRowsRef.current = flatRows;

  // 搜索词变化时回到列表顶部（过滤后的流从头看）。
  useEffect(() => {
    if (flatRows.length > 0) {
      listRef.current?.scrollToRow({ index: 0, align: "start", behavior: "instant" });
    }
  }, [query, flatRows.length, listRef]);

  const jumpToTurn = (turn: number) => {
    setOpenTurns((prev) => new Set(prev).add(turn)); // 展开目标轮
    const idx = flatRowsRef.current.findIndex((row) => row.kind === "turn" && row.turn.turn === turn);
    if (idx >= 0) listRef.current?.scrollToRow({ index: idx, align: "start", behavior: "smooth" });
  };

  const getRowKey = useCallback((index: number) => flatRows[index]?.key ?? String(index), [flatRows]);

  return (
    <div className="flex h-full flex-col gap-2 p-3">
      <div className="flex items-center gap-2">
        <div className="flex items-center gap-3 rounded-lg border border-border-soft bg-bg px-3 py-1.5 text-[10px] text-fg-dim">
          <span className="flex items-center gap-1"><Clock size={11} className="text-accent/70" />Duration {stats.duration ? fmtDuration(stats.duration * 1000) : "—"}</span>
          <span className="text-fg-faint">·</span>
          <span>Turns {stats.turns}</span>
          <span className="text-fg-faint">·</span>
          <span>Calls {stats.calls}</span>
        </div>
        <div className="flex items-center gap-1 text-[10px]">
          <button className="cursor-pointer rounded border-0 bg-transparent px-1.5 py-0.5 text-fg-dim hover:text-fg" onClick={collapseAll}>收起全部</button>
          <span className="text-fg-faint">·</span>
          <button className="cursor-pointer rounded border-0 bg-transparent px-1.5 py-0.5 text-fg-dim hover:text-fg" onClick={expandAll}>展开全部</button>
        </div>
        <div className="relative ml-auto w-56">
          <Search size={11} className="absolute left-2 top-1/2 -translate-y-1/2 text-fg-faint" />
          <input
            className="w-full rounded-md border border-border-soft bg-bg py-1 pl-6 pr-2 text-[10px] text-fg outline-none placeholder:text-fg-faint"
            placeholder="搜索"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </div>
      </div>

      {error && (
        <div className="rounded-lg border border-err/30 bg-del-bg px-3 py-2 text-[11px] text-err">
          轨迹视图加载失败：{error}
        </div>
      )}
      {trajectory.turns.length === 0 && (
        <div className="rounded-lg border border-border-soft bg-bg p-6 text-center text-[11px] text-fg-faint">
          暂无轨迹记录 —— 开始一轮办公任务后，这里会按时间顺序展示完整事件账本。
        </div>
      )}
      {trajectory.turns.length > 0 && (
        <OverviewBar turns={trajectory.turns} onJump={jumpToTurn} />
      )}
      <div className="min-h-0 flex-1">
        <List
          className="h-full"
          style={{ height: "100%" }}
          defaultHeight={600}
          rowCount={flatRows.length}
          rowHeight={rowHeights}
          rowKey={getRowKey}
          rowProps={{ rows: flatRows, openTurns, allCollapsed, onToggle: toggleTurn }}
          rowComponent={TrajectoryRow}
          overscanCount={12}
        />
      </div>
      <div className="flex items-center gap-1.5 text-[9px] text-fg-faint">
        <Eye size={10} /> 点击记录展开详情（工具参数/结果、请求头、用量与耗时）· 长会话按视口虚拟渲染
      </div>
    </div>
  );
}
