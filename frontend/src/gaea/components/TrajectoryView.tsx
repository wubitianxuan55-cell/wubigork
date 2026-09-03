import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Bot, ChevronDown, ChevronRight, Clock, Cpu, Eye, MessageSquare, Search, Shield, User, Wrench } from "../icons";
import { app } from "../lib/bridge";
import { useT } from "../lib/i18n";
import { List, useDynamicRowHeight, useListRef, type RowComponentProps } from "react-window";
import type {
  Trajectory, TrajectoryAssistantRec, TrajectoryHeaderRec, TrajectoryRecord,
  TrajectorySubagentRec, TrajectoryToolRec, TrajectoryTurn,
} from "../lib/types";
import type { DictKey } from "../locales/en";
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
  const t = useT();
  return (
    <div className="flex items-center gap-2 border-y border-dashed border-border-soft bg-warning/5 px-3 py-1.5">
      <span className="text-[10px] font-semibold text-warning uppercase tracking-wider">Between turns</span>
      <span className="text-[9px] text-fg-faint">{t("trajectory.betweenSub")}</span>
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
// ASSISTANT/TOOL/REQUEST HEADER/COMPACTION 为字面英文徽标；中文徽标走字典。
function KindBadge({ kind }: { kind: TrajectoryRecord["kind"] }) {
  const t = useT();
  const map: Record<TrajectoryRecord["kind"], { literal?: string; key?: DictKey; cls: string }> = {
    assistant: { literal: "ASSISTANT", cls: "bg-purple-500/15 text-purple-400" },
    tool: { literal: "TOOL", cls: "bg-orange-500/15 text-orange-400" },
    ask: { key: "trajectory.kindAsk", cls: "bg-blue-800/25 text-blue-300" },
    approval: { key: "trajectory.kindApproval", cls: "bg-amber-500/15 text-amber-400" },
    header: { literal: "REQUEST HEADER", cls: "bg-cyan-500/15 text-cyan-400" },
    compact: { literal: "COMPACTION", cls: "bg-yellow-500/15 text-yellow-400" },
    user: { key: "trajectory.kindUser", cls: "bg-emerald-500/15 text-emerald-400" },
    subagent: { key: "trajectory.kindSubagent", cls: "bg-accent/15 text-accent" },
  };
  const m = map[kind];
  const label = m.key ? t(m.key) : (m.literal ?? "");
  return <span className={`shrink-0 rounded px-1.5 py-0.5 text-[9px] font-semibold tracking-wide ${m.cls}`}>{label}</span>;
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
  const t = useT();
  const meta: string[] = [];
  if (record.durationMs) meta.push(t("trajectory.dur", { d: fmtDuration(record.durationMs) }));
  if (record.step) meta.push(t("trajectory.stepN", { n: record.step }));
  if (record.tool?.status === "running") meta.push(t("trajectory.statusRunning"));
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
      {record.kind === "compact" && <div className="text-warning/90">{record.compact?.summary || record.compact?.trigger || t("trajectory.compactFallback")}</div>}
      {record.kind === "subagent" && record.subagent && <SubagentDetail s={record.subagent} />}
    </div>
  );
}

function HeaderDetail({ h }: { h: TrajectoryHeaderRec }) {
  const t = useT();
  return (
    <div>
      <div className="flex flex-wrap gap-2 text-fg-dim tabular-nums font-mono">
        <span>{t("trajectory.toolItems", { n: h.toolCount })}</span>
        <span>≈{fmtTokens(h.tokens)}</span>
        {h.change && <span className="text-cyan-400">{h.change}</span>}
      </div>
      {h.system && <pre className="mt-1 max-h-40 overflow-auto rounded bg-bg p-1.5 text-[10px] text-fg-dim whitespace-pre-wrap">{h.system}</pre>}
    </div>
  );
}

function AssistantDetail({ a }: { a: TrajectoryAssistantRec }) {
  const t = useT();
  return (
    <div>
      {a.usage && (
        <div className="mb-1 flex flex-wrap gap-2 text-fg-faint tabular-nums font-mono">
          <span>↑{fmtTokens(a.usage.promptTokens ?? 0)}</span>
          <span>↓{fmtTokens(a.usage.completionTokens ?? 0)}</span>
          {a.usage.cacheHitTokens ? <span>{t("trajectory.cache", { tokens: fmtTokens(a.usage.cacheHitTokens) })}</span> : null}
        </div>
      )}
      {a.reasoning && <pre className="mt-1 rounded bg-bg p-1.5 text-[10px] text-fg-dim whitespace-pre-wrap border-l-2 border-info/40">{a.reasoning}</pre>}
      {a.text && <pre className="mt-1 rounded bg-bg p-1.5 text-[10px] text-fg whitespace-pre-wrap">{a.text}</pre>}
    </div>
  );
}

function ToolDetail({ t: rec }: { t: TrajectoryToolRec }) {
  const t = useT();
  return (
    <div>
      <div className="text-fg font-mono">{rec.name}</div>
      {rec.args && <pre className="mt-1 max-h-40 overflow-auto rounded bg-bg p-1.5 text-[10px] text-fg-dim whitespace-pre-wrap">{rec.args}</pre>}
      {rec.output && <pre className={`mt-1 max-h-52 overflow-auto rounded bg-bg p-1.5 text-[10px] font-mono whitespace-pre-wrap ${rec.status === "error" ? "text-err" : "text-fg-dim"}`}>{rec.output}</pre>}
      {rec.err && <div className="mt-1 text-[10px] text-err">{rec.err}</div>}
      {rec.truncated && <div className="mt-1 text-[9px] text-warning">{t("trajectory.outputTruncated")}</div>}
      {rec.parentId && <div className="mt-1 text-[9px] text-fg-faint font-mono">parentId: {rec.parentId}</div>}
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
  const t = useT();
  const [open, setOpen] = useState(false);
  const badge = <KindBadge kind={record.kind} />;
  const summary = useMemo(() => {
    switch (record.kind) {
      case "user": return record.user?.text || "";
      case "header": return t("trajectory.toolSummary", { n: record.header?.toolCount ?? 0, tokens: fmtTokens(record.header?.tokens ?? 0) });
      case "assistant": return record.assistant?.text || (record.assistant?.reasoning ? t("trajectory.reasoningOnly") : "");
      case "tool": {
        const rec = record.tool!;
        const args = rec.args ? rec.args.slice(0, 80) + (rec.args.length > 80 ? "…" : "") : "";
        const out = rec.output ? ` → ${rec.output.slice(0, 60)}${rec.output.length > 60 ? "…" : ""}` : "";
        const err = rec.err ? ` → ${rec.err.slice(0, 60)}` : "";
        return `${rec.name} ${args}${err || out}`;
      }
      case "compact": return `${record.compact?.trigger || "manual"}${record.compact?.summary ? ` — ${record.compact.summary.slice(0, 60)}` : ""}`;
      case "ask": return record.ask?.question || "";
      case "approval": return `${record.approval?.tool || ""}${record.approval?.subject ? ` · ${record.approval.subject}` : ""}`;
      // 子代理回投：折叠态弱存在感一行 = 答复文本（CSS truncate）+ ref（有则显示）
      case "subagent": return `${record.subagent?.text || t("trajectory.noReply")}${record.subagent?.ref ? ` · ${record.subagent.ref}` : ""}`;
      default: return "";
    }
  }, [record, t]);

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
  const t = useT();
  return (
    <div id={`traj-turn-${turn.turn}`} className="scroll-mt-2">
      <button
        className="flex w-full items-center gap-2 border-y border-border-soft bg-bg-soft/60 px-3 py-1.5 text-left border-0 cursor-pointer"
        onClick={onToggle}
      >
        {open ? <ChevronDown size={12} className="text-fg-faint" /> : <ChevronRight size={12} className="text-fg-faint" />}
        <span className="text-[10px] font-semibold text-fg uppercase tracking-wider">{t("trajectory.turnN", { n: turn.turn })}</span>
        <span className="text-[9px] text-fg-faint tabular-nums font-mono">{fmtTime(turn.startedAt)}</span>
        {/* v4.31 轮级耗时：仅轮结束时（end 存在）且有 durationMs 才展示——
            当前 running 轮（无 end）维持现状不显示；旧后端无该字段也跳过。 */}
        {turn.end && turn.durationMs ? (
          <span className="text-[9px] text-fg-faint tabular-nums font-mono">{t("trajectory.turnElapsed", { t: formatElapsed(turn.durationMs / 1000) })}</span>
        ) : null}
        {turn.end?.err && <span className="rounded bg-err/15 px-1 text-[9px] text-err">{t("trajectory.turnError")}</span>}
        <span className="ml-auto text-[9px] text-fg-faint tabular-nums">{t("trajectory.recordCount", { n: shown })}</span>
      </button>
    </div>
  );
}

// ─── 轨迹概览（Overview 投影：每轮一根柱，点击跳转） ────────────
function OverviewBar({ turns, onJump }: { turns: TrajectoryTurn[]; onJump: (turn: number) => void }) {
  const t = useT();
  const maxRec = Math.max(1, ...turns.map((t) => t.records.length));
  const W = Math.max(turns.length * 9, 320);
  const H = 26;
  return (
    <div className="rounded-lg border border-border-soft bg-bg p-2.5">
      <div className="flex items-center justify-between">
        <div className="text-[10px] font-medium text-fg">{t("trajectory.overviewTitle")}</div>
        <div className="text-[9px] text-fg-faint tabular-nums">{t("trajectory.overviewMeta", { n: turns.length })}</div>
      </div>
      <svg viewBox={`0 0 ${W} ${H}`} className="mt-1 w-full h-auto" style={{ width: W }}>
        {turns.map((tr, i) => {
          const x = 2 + i * 9;
          const h = 8 + Math.round((tr.records.length / maxRec) * 10);
          const tools = tr.records.filter((r) => r.kind === "tool").length;
          const fill = tr.end?.err ? "var(--color-err)" : tools > 0 ? "var(--color-accent)" : "var(--color-fg-faint)";
          return (
            <g key={tr.turn} onClick={() => onJump(tr.turn)} className="cursor-pointer">
              <rect x={x} y={H - h} width={5} height={h} rx={1} fill={fill} opacity={0.85}>
                <title>{`${t("trajectory.barTitle", { turn: tr.turn, records: tr.records.length, tools })}${tr.end?.err ? t("trajectory.barErr") : ""}`}</title>
              </rect>
            </g>
          );
        })}
      </svg>
      <div className="mt-1 text-[9px] text-fg-faint">{t("trajectory.overviewLegend")}</div>
    </div>
  );
}

// ─── 容器：拉取 + 搜索 + 统计 + 布局 ────────────────────────
export function TrajectoryView({ running }: { running: boolean }) {
  const t = useT();
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
          <button className="cursor-pointer rounded border-0 bg-transparent px-1.5 py-0.5 text-fg-dim hover:text-fg" onClick={collapseAll}>{t("trajectory.collapseAll")}</button>
          <span className="text-fg-faint">·</span>
          <button className="cursor-pointer rounded border-0 bg-transparent px-1.5 py-0.5 text-fg-dim hover:text-fg" onClick={expandAll}>{t("trajectory.expandAll")}</button>
        </div>
        <div className="relative ml-auto w-56">
          <Search size={11} className="absolute left-2 top-1/2 -translate-y-1/2 text-fg-faint" />
          <input
            className="w-full rounded-md border border-border-soft bg-bg py-1 pl-6 pr-2 text-[10px] text-fg outline-none placeholder:text-fg-faint"
            placeholder={t("trajectory.searchPh")}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </div>
      </div>

      {error && (
        <div className="rounded-lg border border-err/30 bg-del-bg px-3 py-2 text-[11px] text-err">
          {t("trajectory.loadFail", { msg: error })}
        </div>
      )}
      {trajectory.turns.length === 0 && (
        <div className="rounded-lg border border-border-soft bg-bg p-6 text-center text-[11px] text-fg-faint">
          {t("trajectory.empty")}
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
        <Eye size={10} /> {t("trajectory.footerHint")}
      </div>
    </div>
  );
}
