import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ChevronDown, ChevronRight, Clock, Cpu, Eye, MessageSquare, Search, Shield, User, Wrench } from "../icons";
import { app } from "../lib/bridge";
import type {
  Trajectory, TrajectoryAssistantRec, TrajectoryHeaderRec, TrajectoryRecord,
  TrajectoryToolRec, TrajectoryTurn,
} from "../lib/types";
import { fmtTokens } from "../lib/stats";

const EMPTY: Trajectory = { ok: true, turns: [] };

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

// ─── 轮次区段（粗分割线） ───────────────────────────────────
function TurnSection({ turn, query }: { turn: TrajectoryTurn; query: string }) {
  const [open, setOpen] = useState(true);
  const records = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return turn.records;
    return turn.records.filter((r) => {
      const hay = [r.user?.text, r.assistant?.text, r.assistant?.reasoning, r.tool?.name, r.tool?.args, r.tool?.output, r.tool?.err, r.ask?.question, r.approval?.subject, r.compact?.summary].filter(Boolean).join("\n").toLowerCase();
      return hay.includes(q);
    });
  }, [turn, query]);

  if (records.length === 0 && query) return null;
  return (
    <div>
      <button
        className="flex w-full items-center gap-2 border-y border-border-soft bg-bg-soft/60 px-3 py-1.5 text-left border-0 cursor-pointer"
        onClick={() => setOpen(!open)}
      >
        {open ? <ChevronDown size={12} className="text-fg-faint" /> : <ChevronRight size={12} className="text-fg-faint" />}
        <span className="text-[10px] font-semibold text-fg uppercase tracking-wider">第{turn.turn}轮</span>
        <span className="text-[9px] text-fg-faint tabular-nums font-mono">{fmtTime(turn.startedAt)}</span>
        {turn.end?.err && <span className="rounded bg-err/15 px-1 text-[9px] text-err">错误</span>}
        <span className="ml-auto text-[9px] text-fg-faint tabular-nums">{records.length} 条记录</span>
      </button>
      {open && <div className="bg-bg">{records.map((r) => <RecordRow key={`${r.kind}-${r.seq}`} record={r} />)}</div>}
    </div>
  );
}

// ─── 容器：拉取 + 搜索 + 统计 + 布局 ────────────────────────
export function TrajectoryView({ running }: { running: boolean }) {
  const [trajectory, setTrajectory] = useState<Trajectory>(EMPTY);
  const [query, setQuery] = useState("");
  const [error, setError] = useState<string | null>(null);
  const runningRef = useRef(running);

  const load = useCallback(() => {
    app.Trajectory()
      .then((t) => { setTrajectory(t); setError(null); })
      .catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)));
  }, []);

  useEffect(() => { load(); }, [load]);
  useEffect(() => {
    if (runningRef.current && !running) load();
    runningRef.current = running;
  }, [running, load]);

  const stats = useMemo(() => {
    const all = [...trajectory.turns.flatMap((t) => t.records), ...(trajectory.betweenTurns ?? [])];
    const calls = all.filter((r) => r.kind === "tool").length;
    const starts = all.map((r) => r.ts).filter(Boolean);
    const duration = starts.length > 1 ? Math.max(0, Math.max(...starts) - Math.min(...starts)) : 0;
    return { turns: trajectory.turns.length, calls, duration };
  }, [trajectory]);

  return (
    <div className="flex h-full flex-col gap-2 overflow-y-auto p-3">
      <div className="flex items-center gap-2">
        <div className="flex items-center gap-3 rounded-lg border border-border-soft bg-bg px-3 py-1.5 text-[10px] text-fg-dim">
          <span className="flex items-center gap-1"><Clock size={11} className="text-accent/70" />Duration {stats.duration ? fmtDuration(stats.duration * 1000) : "—"}</span>
          <span className="text-fg-faint">·</span>
          <span>Turns {stats.turns}</span>
          <span className="text-fg-faint">·</span>
          <span>Calls {stats.calls}</span>
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
      {trajectory.turns.map((t) => <TurnSection key={t.turn} turn={t} query={query} />)}
      {trajectory.betweenTurns && trajectory.betweenTurns.length > 0 && (
        <div>
          <div className="flex items-center gap-2 border-y border-dashed border-border-soft bg-warning/5 px-3 py-1.5">
            <span className="text-[10px] font-semibold text-warning uppercase tracking-wider">Between turns</span>
            <span className="text-[9px] text-fg-faint">轮次之间</span>
          </div>
          <div className="bg-bg">
            {trajectory.betweenTurns.map((r) => <RecordRow key={`bt-${r.kind}-${r.seq}`} record={r} />)}
          </div>
        </div>
      )}
      <div className="flex items-center gap-1.5 text-[9px] text-fg-faint">
        <Eye size={10} /> 点击记录展开详情（工具参数/结果、请求头、用量与耗时）
      </div>
    </div>
  );
}
