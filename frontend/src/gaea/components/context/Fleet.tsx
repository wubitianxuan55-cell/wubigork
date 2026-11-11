// 编队页（v4.542 刀5，对标 dsh-context Fleet tab）：Agent 家族视图——摘要条、
// 协作泳道（派发→回报时间线）、协作任务（状态筛选+搜索）、事件检查（派发/
// 回报合并流）、Agent 网络径向图（复用 AgentRadial）。
// 数据源：GaeaAgentNetwork（谱系树）+ SubagentRuns（实时运行快照，共享单轮
// 询 store）。诚实子集：消息级跨 transcript 合并流、blocked-by 依赖链需新
// 后端面，本版不造（见拍板池）；家族消息计数无日志级聚合，摘要条不展示。
import { useCallback, useEffect, useMemo, useState } from "react";
import { useT } from "../../lib/i18n";
import { reloadSubagentRuns, subscribeSubagentRuns, type SubagentRunsMeta } from "../../lib/subagentRunsStore";
import { subscribeAgentNetwork } from "../../lib/agentNetworkStore";
import type { AgentNetwork } from "../../lib/types";
import type { SubagentRunView } from "../../lib/types/subagent";
import { AgentRadial } from "./AgentRadial";

type RunStatus = SubagentRunView["status"];

// STATUS_COLOR 与 dsh/子代理既有语义对齐：running 绿（脉冲在 AgentRadial）、
// queued 灰、completed 深绿、failed 红。
const STATUS_COLOR: Record<RunStatus, string> = {
  running: "var(--md-sys-color-success, #22c55e)",
  queued: "var(--md-sys-color-outline, #9e9e9e)",
  completed: "var(--md-sys-color-primary, var(--color-primary))",
  failed: "var(--md-sys-color-destructive, #ef4444)",
};

function shortTask(task: string, max = 24): string {
  const one = task.split("\n")[0] ?? task;
  return one.length > max ? `${one.slice(0, max)}…` : one;
}

export function Fleet({ running, sessionPath, onViewContext }: {
  running?: boolean;
  sessionPath?: string;
  onViewContext?: (ref: string) => void;
}) {
  const t = useT();
  const [net, setNet] = useState<AgentNetwork | null>(null);
  const [runs, setRuns] = useState<SubagentRunView[]>([]);
  const [meta, setMeta] = useState<SubagentRunsMeta>({ status: "loading", total: 0, running: 0 });

  useEffect(
    () => subscribeAgentNetwork((n) => setNet(n), { path: sessionPath }),
    [sessionPath],
  );
  useEffect(
    () => subscribeSubagentRuns(sessionPath ?? "", (r, m) => { setRuns(r); setMeta(m); }),
    [sessionPath],
  );

  const subagentRuns = useMemo(() => runs.filter((r) => (r.kind ?? "subagent") === "subagent"), [runs]);
  const toolCallsTotal = useMemo(() => runs.reduce((s, r) => s + (r.toolCalls ?? 0), 0), [runs]);

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto p-4" data-testid="ctx-fleet">
      {/* 摘要条 */}
      <div className="grid grid-cols-4 gap-2" data-testid="fleet-summary">
        <Summary label={t("fleet.kpiAgents")} value={String(1 + subagentRuns.length)} />
        <Summary label={t("fleet.kpiRunning")} value={String(meta.running)} accent={meta.running > 0} />
        <Summary label={t("fleet.kpiTasks")} value={String(subagentRuns.length)} />
        <Summary label={t("fleet.kpiToolCalls")} value={String(toolCallsTotal)} />
      </div>

      {/* Agent 网络（谱系径向图，复用实时拓扑组件） */}
      {net && (
        <section className="ctx-card p-3">
          <SectionTitle>{t("fleet.networkTitle")}</SectionTitle>
          <AgentRadial network={net} running={!!running} sessionPath={sessionPath} onViewContext={onViewContext} />
        </section>
      )}

      {/* 协作泳道 */}
      {subagentRuns.length > 0 && <SwimLanes runs={subagentRuns} />}

      {/* 协作任务 + 事件检查 */}
      {subagentRuns.length > 0 && (
        <div className="grid grid-cols-1 gap-3 min-[1100px]:grid-cols-2">
          <TaskPanel runs={subagentRuns} />
          <EventInspector runs={subagentRuns} />
        </div>
      )}

      {subagentRuns.length === 0 && meta.status !== "loading" && (
        <p className="py-6 text-center text-[12px] text-fg-dim" data-testid="fleet-empty">{t("fleet.empty")}</p>
      )}
    </div>
  );
}

function Summary({ label, value, accent }: { label: string; value: string; accent?: boolean }) {
  return (
    <div className="rounded-md border border-border-soft bg-panel-2 px-2 py-1.5">
      <div className="text-[10.5px] text-fg-dim">{label}</div>
      <div className={`font-mono text-[15px] font-semibold ${accent ? "text-(color:--color-success)" : "text-fg"}`}>{value}</div>
    </div>
  );
}

function SectionTitle({ children }: { children: React.ReactNode }) {
  return <h3 className="m-0 mb-1.5 text-[12.5px] font-medium text-fg">{children}</h3>;
}

// ─── 协作泳道 ────────────────────────────────────────────────
// 每个子代理一条道：派发（createdAt）→ 收尾（updatedAt）的 span 画在同一条
// 全局时间轴上；父代理道标出每次派发点。任务编号认同一件事：悬停联动同
// 任务（简化为同 lane 高亮）。
function SwimLanes({ runs }: { runs: SubagentRunView[] }) {
  const t = useT();
  const [hovered, setHovered] = useState<string | null>(null);
  const times = runs.flatMap((r) => [Date.parse(r.createdAt), Date.parse(r.updatedAt || r.createdAt)]).filter((x) => Number.isFinite(x));
  if (times.length === 0) return null;
  const min = Math.min(...times);
  const max = Math.max(...times, min + 1);
  const pos = (ts: number): number => ((ts - min) / (max - min)) * 100;
  // 父代理道：派发点 = 各 run.createdAt（0 偏移开窗给 hover 空间）。
  return (
    <section className="ctx-card p-3" data-testid="fleet-lanes">
      <SectionTitle>{t("fleet.lanesTitle")}</SectionTitle>
      <div className="flex flex-col gap-1">
        <Lane
          name={t("fleet.laneParent")}
          color="var(--md-sys-color-primary, var(--color-primary))"
          marks={runs.map((r) => ({ at: pos(Date.parse(r.createdAt)), title: `${t("fleet.evDispatch")}: ${shortTask(r.task, 60)}` }))}
          span={null}
          dim={hovered !== null}
        />
        {runs.map((r) => {
          const start = pos(Date.parse(r.createdAt));
          const end = pos(Date.parse(r.updatedAt || r.createdAt));
          const dim = hovered !== null && hovered !== r.ref;
          return (
            <div key={r.ref} onMouseEnter={() => setHovered(r.ref)} onMouseLeave={() => setHovered(null)}>
              <Lane
                name={shortTask(r.task)}
                color={STATUS_COLOR[r.status]}
                marks={[]}
                span={{ start, end, status: r.status }}
                dim={dim}
                title={`${r.task} · ${t("fleet.laneSpanTip", { from: fmtClock(r.createdAt), to: fmtClock(r.updatedAt || r.createdAt) })}`}
              />
            </div>
          );
        })}
      </div>
    </section>
  );
}

function Lane({ name, color, marks, span, dim, title }: {
  name: string;
  color: string;
  marks: { at: number; title: string }[];
  span: { start: number; end: number; status: RunStatus } | null;
  dim?: boolean;
  title?: string;
}) {
  return (
    <div className={`flex items-center gap-2 ${dim ? "opacity-40" : "opacity-100"} transition-opacity`} title={title}>
      <span className="w-40 shrink-0 truncate text-[10.5px] text-fg-dim">{name}</span>
      <div className="relative h-3.5 flex-1 rounded bg-bg-soft">
        {marks.map((m, i) => (
          <span
            key={i}
            title={m.title}
            className="absolute top-0 h-full w-0.5 -translate-x-1/2"
            style={{ left: `${m.at}%`, background: color }}
          />
        ))}
        {span && (
          <span
            className="absolute top-0.5 h-2.5 rounded-full"
            style={{
              left: `${span.start}%`,
              width: `${Math.max(span.end - span.start, 1.5)}%`,
              background: STATUS_COLOR[span.status],
              opacity: 0.8,
            }}
          />
        )}
      </div>
    </div>
  );
}

function fmtClock(iso: string): string {
  const d = new Date(iso);
  return Number.isFinite(d.getTime()) ? d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }) : "—";
}

// ─── 协作任务 ────────────────────────────────────────────────
function TaskPanel({ runs }: { runs: SubagentRunView[] }) {
  const t = useT();
  const [status, setStatus] = useState<RunStatus | "all">("all");
  const [q, setQ] = useState("");
  const filtered = runs.filter((r) => {
    if (status !== "all" && r.status !== status) return false;
    const query = q.trim().toLowerCase();
    if (query && !(`${r.task} ${r.answer ?? ""}`.toLowerCase().includes(query))) return false;
    return true;
  });
  return (
    <section className="ctx-card p-3" data-testid="fleet-tasks">
      <div className="mb-1.5 flex flex-wrap items-center gap-1">
        <SectionTitle>{t("fleet.tasksTitle")}</SectionTitle>
        <span className="ml-auto flex items-center gap-1">
          {(["all", "running", "queued", "completed", "failed"] as const).map((s) => (
            <button
              key={s}
              type="button"
              aria-pressed={status === s}
              data-testid={`fleet-task-status-${s}`}
              className={`cursor-pointer rounded border-0 px-1.5 py-0.5 text-[10px] ${status === s ? "bg-accent/15 text-accent" : "text-fg-dim hover:text-fg"}`}
              onClick={() => setStatus(s)}
            >
              {s === "all"
                ? t("fleet.filterAll")
                : s === "running"
                  ? t("fleet.stRunning")
                  : s === "queued"
                    ? t("fleet.stQueued")
                    : s === "completed"
                      ? t("fleet.stCompleted")
                      : t("fleet.stFailed")}
            </button>
          ))}
          <input
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder={t("fleet.taskSearch")}
            className="w-28 rounded border border-border-soft bg-transparent px-1.5 py-0.5 text-[10px] text-fg placeholder:text-fg-faint focus:outline-none focus:ring-1 focus:ring-accent/40"
          />
        </span>
      </div>
      <ul className="m-0 flex max-h-64 list-none flex-col gap-1.5 overflow-y-auto p-0">
        {filtered.map((r) => (
          <li key={r.ref} className="rounded-md border border-border-soft bg-panel-2 p-1.5 text-[10.5px]">
            <div className="flex items-center gap-1.5">
              <span className="h-2 w-2 shrink-0 rounded-full" style={{ background: STATUS_COLOR[r.status] }} title={r.status} />
              <span className="min-w-0 flex-1 truncate text-fg" title={r.task}>{shortTask(r.task, 40)}</span>
              {r.model && <span className="shrink-0 font-mono text-[9.5px] text-fg-dim">{r.model}</span>}
              <span className="shrink-0 font-mono text-[9.5px] text-fg-dim">×{r.toolCalls}</span>
            </div>
            {r.answer && <div className="mt-0.5 line-clamp-2 text-[10px] text-fg-dim">{r.answer}</div>}
          </li>
        ))}
        {filtered.length === 0 && <li className="py-3 text-center text-[11px] text-fg-dim">{t("fleet.none")}</li>}
      </ul>
    </section>
  );
}

// ─── 事件检查 ────────────────────────────────────────────────
// 派发/回报合并流（诚实子集：来自 runs 元数据的两类事件；transcript 级消息
// 合并流挂拍板池）。
function EventInspector({ runs }: { runs: SubagentRunView[] }) {
  const t = useT();
  const [kind, setKind] = useState<"all" | "dispatch" | "report">("all");
  const [q, setQ] = useState("");
  type Ev = { ts: number; kind: "dispatch" | "report"; run: SubagentRunView };
  const events = useMemo<Ev[]>(() => {
    const out: Ev[] = [];
    for (const r of runs) {
      const c = Date.parse(r.createdAt);
      if (Number.isFinite(c)) out.push({ ts: c, kind: "dispatch", run: r });
      const u = Date.parse(r.updatedAt || r.createdAt);
      if (Number.isFinite(u) && r.status !== "running" && r.status !== "queued") out.push({ ts: u, kind: "report", run: r });
    }
    out.sort((a, b) => b.ts - a.ts);
    return out;
  }, [runs]);
  const filtered = events.filter((e) => {
    if (kind !== "all" && e.kind !== kind) return false;
    const query = q.trim().toLowerCase();
    if (query && !(`${e.run.task} ${e.run.answer ?? ""}`.toLowerCase().includes(query))) return false;
    return true;
  });
  return (
    <section className="ctx-card p-3" data-testid="fleet-events">
      <div className="mb-1.5 flex flex-wrap items-center gap-1">
        <SectionTitle>{t("fleet.eventsTitle")}</SectionTitle>
        <span className="ml-auto flex items-center gap-1">
          {(["all", "dispatch", "report"] as const).map((k) => (
            <button
              key={k}
              type="button"
              aria-pressed={kind === k}
              className={`cursor-pointer rounded border-0 px-1.5 py-0.5 text-[10px] ${kind === k ? "bg-accent/15 text-accent" : "text-fg-dim hover:text-fg"}`}
              onClick={() => setKind(k)}
            >
              {k === "all" ? t("fleet.filterAll") : k === "dispatch" ? t("fleet.evDispatch") : t("fleet.evReport")}
            </button>
          ))}
          <input
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder={t("fleet.taskSearch")}
            className="w-28 rounded border border-border-soft bg-transparent px-1.5 py-0.5 text-[10px] text-fg placeholder:text-fg-faint focus:outline-none focus:ring-1 focus:ring-accent/40"
          />
        </span>
      </div>
      <ul className="m-0 flex max-h-64 list-none flex-col overflow-y-auto p-0">
        {filtered.map((e, i) => (
          <li key={`${e.run.ref}-${e.kind}-${i}`} className="ctx-row flex items-start gap-1.5 px-1.5 py-1 text-[10.5px]">
            <span
              className="mt-1 h-2 w-2 shrink-0 rounded-sm"
              style={{ background: e.kind === "dispatch" ? "var(--md-sys-color-primary, var(--color-primary))" : STATUS_COLOR[e.run.status] }}
            />
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-1.5">
                <span className="font-medium text-fg-dim">{e.kind === "dispatch" ? t("fleet.evDispatch") : t("fleet.evReport")}</span>
                <span className="min-w-0 truncate text-fg">{shortTask(e.run.task, 32)}</span>
                <span className="ml-auto shrink-0 font-mono text-[9.5px] text-fg-dim">{fmtClock(new Date(e.ts).toISOString())}</span>
              </div>
              {e.kind === "report" && e.run.answer && (
                <div className="mt-0.5 line-clamp-2 text-[10px] text-fg-dim">{e.run.answer}</div>
              )}
            </div>
          </li>
        ))}
        {filtered.length === 0 && <li className="py-3 text-center text-[11px] text-fg-dim">{t("fleet.none")}</li>}
      </ul>
    </section>
  );
}
