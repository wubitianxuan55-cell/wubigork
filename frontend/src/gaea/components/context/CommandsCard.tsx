// 命令执行卡（v4.542 刀4，对标 dsh-context Command Executions）：bash 工具
// 推导的命令生命周期时间线。状态筛选/搜索收窄列表；程序分布 chip 点击回填
// 搜索框（dsh 同款交互）；诚实子集——exit 只认 bash JSON 信封、killed=回合
// 中断、ms 为秒级时间戳近似。
import { useMemo, useState } from "react";
import { useT } from "../../lib/i18n";
import { fmtDuration } from "./cards";
import type { ContextCommandRecord, ContextCommandStats } from "../../lib/types/context";
import type { Translator } from "../../lib/i18n";

type CmdStatus = ContextCommandRecord["status"];
type CmdFilter = "all" | "failed" | "slow" | "background";
type CmdSort = "latest" | "slowest";

const SLOW_MS = 10_000;

// STATUS_LABEL：不用 i18n（状态词与 dsh/终端语义对齐，mono 展示）。
const STATUS_COLOR: Record<CmdStatus, string> = {
  completed: "var(--md-sys-color-success, #22c55e)",
  failed: "var(--md-sys-color-destructive, #ef4444)",
  killed: "var(--md-sys-color-warning, #f59e0b)",
};

// commandProgram 与 Go 侧 commandProgram 同口径的客户端近似（程序分布 chip）。
function commandProgram(cmd: string): string {
  for (const tok of cmd.split(/\s+/).filter(Boolean)) {
    if (tok.includes("=") && !tok.startsWith("/") && !tok.startsWith("-")) continue;
    if (tok === "sudo" || tok === "nice" || tok === "nohup" || tok === "command") continue;
    const base = tok.replace(/^["']|["']$/g, "").split(/[\\/]/).pop();
    return base || tok;
  }
  return "other";
}

function durationOf(record: ContextCommandRecord): number {
  return record.end != null && record.end > record.start ? (record.end - record.start) * 1000 : 0;
}

export function CommandsCard({ commands, stats }: {
  commands: ContextCommandRecord[];
  stats?: ContextCommandStats;
}) {
  const t = useT();
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState<CmdFilter>("all");
  const [sort, setSort] = useState<CmdSort>("latest");
  const [pickedProgram, setPickedProgram] = useState<string | null>(null);

  const programs = useMemo(() => {
    const m = new Map<string, number>();
    for (const c of commands) {
      const p = commandProgram(c.cmd ?? "");
      m.set(p, (m.get(p) ?? 0) + 1);
    }
    return [...m.entries()].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0])).slice(0, 8);
  }, [commands]);

  const searched = useMemo(() => {
    const q = query.trim().toLowerCase();
    return commands.filter((c) => {
      if (pickedProgram && commandProgram(c.cmd ?? "") !== pickedProgram) return false;
      if (q && !(c.cmd ?? "").toLowerCase().includes(q)) return false;
      return true;
    });
  }, [commands, query, pickedProgram]);
  const counts = useMemo(() => ({
    all: searched.length,
    failed: searched.filter((c) => c.status === "failed").length,
    slow: searched.filter((c) => durationOf(c) >= SLOW_MS).length,
    background: searched.filter((c) => c.background === true).length,
  }), [searched]);
  const filtered = useMemo(() => {
    const selected = searched.filter((c) => {
      if (filter === "failed") return c.status === "failed";
      if (filter === "slow") return durationOf(c) >= SLOW_MS;
      if (filter === "background") return c.background === true;
      return true;
    });
    return selected.slice().sort((a, b) => (
      sort === "slowest"
        ? durationOf(b) - durationOf(a) || b.time - a.time
        : b.time - a.time
    ));
  }, [filter, searched, sort]);
  const tally = useMemo(() => ({
    runs: stats?.runs ?? commands.length,
    failed: stats?.failed ?? commands.filter((c) => c.status === "failed").length,
    ms: stats?.ms ?? commands.reduce((sum, c) => sum + durationOf(c), 0),
    bg: stats?.bg ?? commands.filter((c) => c.background === true).length,
  }), [commands, stats]);
  const slowest = useMemo(
    () => commands.reduce<ContextCommandRecord | null>(
      (best, c) => (best === null || durationOf(c) > durationOf(best) ? c : best),
      null,
    ),
    [commands],
  );
  const repeated = useMemo(() => {
    const byCommand = new Map<string, number>();
    for (const command of commands) {
      const text = (command.cmd ?? "").trim();
      if (text) byCommand.set(text, (byCommand.get(text) ?? 0) + 1);
    }
    return [...byCommand.entries()].sort((a, b) => b[1] - a[1])[0] ?? null;
  }, [commands]);

  if (commands.length === 0) return null;

  return (
    <section className="ctx-card p-3 ctx-cmd-card" data-testid="ctx-commands">
      <div className="ctx-cmd-head">
        <span className="ctx-card-title-text">{t("contextview.cmdTitle")}</span>
        <div className="ctx-cmd-filters" role="group" aria-label={t("contextview.cmdTitle")}>
          {(["all", "failed", "slow", "background"] as const).map((key) => (
            <button
              key={key}
              type="button"
              aria-pressed={filter === key}
              data-testid={`ctx-cmd-status-${key}`}
              data-tone={key}
              className="ctx-cmd-filter"
              onClick={() => setFilter((cur) => (cur === key ? "all" : key))}
            >
              {key === "all" ? t("contextview.cmdAll") : t(`contextview.cmdFilter${key[0].toUpperCase()}${key.slice(1)}` as Parameters<Translator>[0])}
              <b>{counts[key]}</b>
            </button>
          ))}
        </div>
      </div>
      <div className="ctx-cmd-body">
        <aside className="ctx-cmd-rail">
          <section className="ctx-cmd-rail-section">
            <h3 className="ctx-cmd-rail-title">{t("contextview.cmdSummary")}</h3>
            <div className="ctx-cmd-kpis">
              <button type="button" className="ctx-cmd-kpi" onClick={() => setFilter("all")}>
                <b>{tally.runs}</b><span>{t("contextview.cmdRunCount")}</span>
              </button>
              <button type="button" className="ctx-cmd-kpi" onClick={() => setFilter("all")} title={t("contextview.cmdMsNote")}>
                <b>{fmtDuration(tally.ms, t)}</b><span>{t("contextview.cmdTotalTime")}</span>
              </button>
              <button type="button" className="ctx-cmd-kpi" data-tone="failed" onClick={() => setFilter("failed")}>
                <b>{tally.failed}</b><span>{t("contextview.cmdFilterFailed")}</span>
              </button>
              <button type="button" className="ctx-cmd-kpi" data-tone="background" onClick={() => setFilter("background")}>
                <b>{tally.bg}</b><span>{t("contextview.cmdFilterBackground")}</span>
              </button>
            </div>
          </section>
          {(slowest || (repeated && repeated[1] > 1)) && (
            <section className="ctx-cmd-rail-section">
              <h3 className="ctx-cmd-rail-title">{t("contextview.cmdAttention")}</h3>
              <div className="ctx-cmd-attention">
                {slowest && durationOf(slowest) > 0 && (
                  <button type="button" onClick={() => { setFilter("slow"); setSort("slowest"); }} title={slowest.cmd}>
                    <i data-tone="slow" />
                    <span>{t("contextview.cmdSlowest")}</span>
                    <em>{slowest.cmd || "—"}</em>
                    <b>{fmtDuration(durationOf(slowest), t)}</b>
                  </button>
                )}
                {repeated && repeated[1] > 1 && (
                  <button type="button" onClick={() => setQuery(repeated[0])} title={repeated[0]}>
                    <i data-tone="repeat" />
                    <span>{t("contextview.cmdRepeats")}</span>
                    <em>{repeated[0]}</em>
                    <b>×{repeated[1]}</b>
                  </button>
                )}
              </div>
            </section>
          )}
          <section className="ctx-cmd-rail-section ctx-cmd-programs">
            <h3 className="ctx-cmd-rail-title">
              <span>{t("contextview.cmdProgramsN", { n: programs.length })}</span>
            </h3>
            <div className="ctx-cmd-programs-list">
              {programs.map(([name, n], index) => (
                <button
                  key={name}
                  type="button"
                  onClick={() => {
                    const next = pickedProgram === name ? null : name;
                    setPickedProgram(next);
                    setQuery(next ?? "");
                  }}
                  data-picked={pickedProgram === name || undefined}
                  className="ctx-cmd-program"
                  title={`${name} ×${n}`}
                >
                  <i data-rank={Math.min(index, 4)} />
                  <span>{name}</span>
                  <b>×{n}</b>
                </button>
              ))}
            </div>
          </section>
        </aside>
        <div className="ctx-cmd-main">
          <div className="ctx-cmd-controls">
            <input
              value={query}
              onChange={(event) => {
                setPickedProgram(null);
                setQuery(event.target.value);
              }}
              placeholder={t("contextview.cmdSearch")}
              aria-label={t("contextview.cmdSearch")}
              data-testid="ctx-cmd-search"
              className="ctx-cmd-search"
            />
            <div className="ctx-cmd-sort" role="group">
              {(["latest", "slowest"] as const).map((key) => (
                <button
                  key={key}
                  type="button"
                  aria-pressed={sort === key}
                  onClick={() => setSort(key)}
                >
                  {key === "latest" ? t("contextview.cmdSortLatest") : t("contextview.cmdSortSlowest")}
                </button>
              ))}
            </div>
          </div>
          <ul className="ctx-cmd-list" data-testid="ctx-commands-list">
            {filtered.map((c, index) => (
              <CommandRow key={c.seq} c={c} t={t} striped={index % 2 === 1} />
            ))}
            {filtered.length === 0 && (
              <li className="ctx-cmd-empty">{t("contextview.cmdNone")}</li>
            )}
          </ul>
        </div>
      </div>
    </section>
  );
}

function CommandRow({ c, t, striped }: { c: ContextCommandRecord; t: Translator; striped: boolean }) {
  const dur = durationOf(c);
  return (
    <li className={`ctx-cmd-row${striped ? " ctx-cmd-row-alt" : ""}`}>
      <span className="ctx-cmd-row-state" data-status={c.status} aria-hidden>{c.status === "completed" ? "✓" : c.status === "failed" ? "✕" : "!"}</span>
      <span
        className="ctx-cmd-dot"
        style={{ background: STATUS_COLOR[c.status] }}
        title={c.status}
        data-testid={`ctx-cmd-dot-${c.status}`}
      />
      <div className="min-w-0 flex-1">
        <div className="ctx-cmd-row-main">
          <span className="ctx-cmd-stamp">{t("contextview.cmdTurnStep", { turn: c.turn, step: c.step })}</span>
          {c.background && (
            <span className="ctx-cmd-badge" title="run_in_background">bg</span>
          )}
          {c.exit != null && c.exit !== 0 && (
            <span className="ctx-cmd-exit" title={t("contextview.cmdExitNote")}>exit {c.exit}</span>
          )}
          {c.truncated && <span className="text-warning">{t("contextview.detailClamped")}</span>}
          <span className="ctx-cmd-duration">
            {dur > 0 ? fmtDuration(dur, t) : (c.status === "killed" ? "—" : "")}
          </span>
        </div>
        <div className="ctx-cmd-command" title={c.cmd}>
          {c.cmd || "—"}
        </div>
        {c.err && (
          <div className="ctx-cmd-error">✗ error</div>
        )}
      </div>
    </li>
  );
}
