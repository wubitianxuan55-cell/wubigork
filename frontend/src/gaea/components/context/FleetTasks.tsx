import { useMemo, useState, type CSSProperties } from "react";
import {
  ArrowUpRight,
  Bot,
  ChevronDown,
  ClipboardList,
  Clock,
  Wrench,
} from "../../icons";
import { useT } from "../../lib/i18n";
import { fmtTokens } from "../../lib/stats";
import type { FamilyAgentRowView } from "../../lib/types/context";
import type { SubagentRunView } from "../../lib/types/subagent";
import { agentHue } from "./fleetVisuals";
import "./fleet-tasks.css";

type RunStatus = SubagentRunView["status"];
type PhaseFilter = "all" | RunStatus;

const STATUS_ORDER: RunStatus[] = ["running", "queued", "completed", "failed", "aborted"];

export interface FleetTasksProps {
  runs: SubagentRunView[];
  agents: FamilyAgentRowView[];
  selectedAgent: string | null;
  onSelectAgent: (ref: string | null) => void;
  onViewContext?: (ref: string) => void;
}

interface RunGroup {
  ref: string;
  runs: SubagentRunView[];
}

function taskTitle(task: string, max = 58): string {
  const first = task.split("\n")[0]?.trim() ?? task;
  return first.length > max ? `${first.slice(0, max)}…` : first;
}

function avatarLabel(task: string): string {
  return task.replace(/\s+/g, "").slice(0, 2).toUpperCase() || "A";
}

function statusLabel(status: RunStatus, t: ReturnType<typeof useT>): string {
  if (status === "running") return t("fleet.stRunning");
  if (status === "queued") return t("fleet.stQueued");
  if (status === "completed") return t("fleet.stCompleted");
  return t("fleet.stFailed");
}

function fmtTime(iso: string): string {
  const date = new Date(iso);
  return Number.isFinite(date.getTime())
    ? date.toLocaleString([], {
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
    })
    : "—";
}

export function FleetTasks({
  runs,
  agents,
  selectedAgent,
  onSelectAgent,
  onViewContext,
}: FleetTasksProps) {
  const t = useT();
  const [phase, setPhase] = useState<PhaseFilter>("all");
  const [query, setQuery] = useState("");
  const [expanded, setExpanded] = useState<string | null>(null);
  const statsByRef = useMemo(
    () => new Map(agents.map((agent) => [agent.ref, agent])),
    [agents],
  );
  const orderedRuns = useMemo(
    () => [...runs].sort((a, b) => Date.parse(a.createdAt) - Date.parse(b.createdAt)),
    [runs],
  );
  const hueByRef = useMemo(() => {
    const hues = new Map<string, string>();
    for (const run of orderedRuns) {
      if (!hues.has(run.ref)) hues.set(run.ref, agentHue(run.ref, hues.size));
    }
    return hues;
  }, [orderedRuns]);
  const scopedRuns = useMemo(
    () => selectedAgent === null || selectedAgent === "main"
      ? orderedRuns
      : orderedRuns.filter((run) => run.ref === selectedAgent),
    [orderedRuns, selectedAgent],
  );
  const counts = useMemo(() => {
    const next: Record<PhaseFilter, number> = {
      all: scopedRuns.length,
      running: 0,
      queued: 0,
      completed: 0,
      failed: 0,
      aborted: 0,
    };
    for (const run of scopedRuns) next[run.status] += 1;
    return next;
  }, [scopedRuns]);
  const needle = query.trim().toLowerCase();
  const filtered = useMemo(() => {
    return scopedRuns.filter((run) => {
      if (phase !== "all" && run.status !== phase) return false;
      if (needle === "") return true;
      return [
        run.task,
        run.answer,
        run.model,
        run.lastTool,
        run.ref,
      ].some((value) => value?.toLowerCase().includes(needle));
    });
  }, [needle, phase, scopedRuns]);
  const groups = useMemo(() => {
    const map = new Map<string, RunGroup>();
    for (const run of filtered) {
      const group = map.get(run.ref) ?? { ref: run.ref, runs: [] };
      group.runs.push(run);
      map.set(run.ref, group);
    }
    return [...map.values()];
  }, [filtered]);

  return (
    <section className="ctx-card fleet-tasks-panel" data-testid="fleet-tasks">
      <div className="fleet-tasks-head">
        <span className="fleet-tasks-title">
          <ClipboardList size={14} aria-hidden />
          {t("fleet.tasksTitle")}
        </span>
        <span className="fleet-tasks-meta">
          {t("fleet.workSummary", { shown: filtered.length, total: scopedRuns.length })}
        </span>
        <label className="fleet-tasks-search">
          <span className="sr-only">{t("fleet.taskSearch")}</span>
          <input
            type="search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={t("fleet.taskSearch")}
          />
        </label>
      </div>

      <div className="fleet-tasks-filters" role="group" aria-label={t("fleet.tasksTitle")}>
        <button
          type="button"
          aria-pressed={phase === "all"}
          data-testid="fleet-task-status-all"
          onClick={() => setPhase("all")}
        >
          {t("fleet.filterAll")}
          <b>{counts.all}</b>
        </button>
        {STATUS_ORDER.map((status) => (
          <button
            key={status}
            type="button"
            aria-pressed={phase === status}
            data-testid={`fleet-task-status-${status}`}
            data-status={status}
            onClick={() => setPhase(status)}
          >
            {statusLabel(status, t)}
            <b>{counts[status]}</b>
          </button>
        ))}
      </div>

      <div className="fleet-tasks-body">
        {groups.length === 0 ? (
          <div className="fleet-tasks-empty">
            <ClipboardList size={20} aria-hidden />
            <span>{t("fleet.none")}</span>
          </div>
        ) : groups.map((group) => {
          const stats = statsByRef.get(group.ref);
          const groupRun = group.runs[0];
          const hue = hueByRef.get(group.ref) ?? agentHue(group.ref);
          const groupSelected = selectedAgent === group.ref;
          return (
            <section
              key={group.ref}
              className="fleet-task-group"
              data-owner={group.ref}
              style={{ "--fleet-task-hue": hue } as CSSProperties}
            >
              <div className={`fleet-task-group-head${groupSelected ? " is-selected" : ""}`}>
                <button
                  type="button"
                  className="fleet-task-owner"
                  aria-pressed={groupSelected}
                  onClick={() => onSelectAgent(groupSelected ? null : group.ref)}
                >
                  <span className="fleet-task-avatar" aria-hidden>
                    {avatarLabel(groupRun?.task || group.ref)}
                  </span>
                  <span className="fleet-task-owner-copy">
                    <b title={groupRun?.task}>{taskTitle(groupRun?.task || group.ref, 32)}</b>
                    <small>{t("fleet.roleSub")} · {group.ref}</small>
                  </span>
                </button>
                <span className="fleet-task-owner-live">
                  <i data-status={groupRun?.status} aria-hidden />
                  {groupRun ? statusLabel(groupRun.status, t) : t("fleet.none")}
                </span>
                {stats && (
                  <span className="fleet-task-owner-stats">
                    {fmtTokens(stats.tokens)} tok
                    {stats.cost !== undefined ? ` · ¥${stats.cost.toFixed(2)}` : ""}
                    {` · ${stats.messages} ${t("fleet.kpiMessages")}`}
                  </span>
                )}
                <span className="fleet-task-group-count">
                  {group.runs.length}
                </span>
              </div>

              <ol className="fleet-task-list">
                {group.runs.map((run) => {
                  const open = expanded === run.ref;
                  return (
                    <li key={run.ref} className={`fleet-task-item is-${run.status}${open ? " is-open" : ""}`}>
                      <button
                        type="button"
                        className="fleet-task-toggle"
                        aria-expanded={open}
                        aria-label={t(open ? "fleet.taskCollapse" : "fleet.taskExpand")}
                        onClick={() => setExpanded(open ? null : run.ref)}
                      >
                        <span className="fleet-task-status" data-status={run.status}>
                          <i aria-hidden />
                          {statusLabel(run.status, t)}
                        </span>
                        <span className="fleet-task-name" title={run.task}>{taskTitle(run.task)}</span>
                        <span className="fleet-task-tail">
                          {run.model && <span className="fleet-task-model">{run.model}</span>}
                          <span className="fleet-task-tools">
                            <Wrench size={10} aria-hidden />
                            {run.toolCalls}
                          </span>
                          <ChevronDown
                            size={11}
                            className="fleet-task-chevron"
                            aria-hidden
                          />
                        </span>
                      </button>
                      <div className="fleet-task-subline">
                        <Clock size={10} aria-hidden />
                        <span>{fmtTime(run.createdAt)}</span>
                        <span>→</span>
                        <span>{fmtTime(run.updatedAt)}</span>
                        {run.lastTool && <span className="fleet-task-last-tool">{run.lastTool}</span>}
                      </div>
                      {open && (
                        <div className="fleet-task-detail">
                          <div className="fleet-task-detail-label">
                            <Bot size={11} aria-hidden />
                            {t("fleet.summary")}
                          </div>
                          <p>{run.answer || run.lastText || t("fleet.none")}</p>
                          {run.followUpError && (
                            <p className="fleet-task-error">{run.followUpError}</p>
                          )}
                          {onViewContext && (
                            <button
                              type="button"
                              className="fleet-task-open"
                              onClick={() => onViewContext(run.ref)}
                            >
                              <ArrowUpRight size={11} aria-hidden />
                              {t("fleet.open")}
                            </button>
                          )}
                        </div>
                      )}
                    </li>
                  );
                })}
              </ol>
            </section>
          );
        })}
      </div>
    </section>
  );
}
