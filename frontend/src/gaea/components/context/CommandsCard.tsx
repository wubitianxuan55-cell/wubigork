// 命令执行卡（v4.542 刀4，对标 dsh-context Command Executions）：bash 工具
// 推导的命令生命周期时间线。状态筛选/搜索收窄列表；程序分布 chip 点击回填
// 搜索框（dsh 同款交互）；诚实子集——exit 只认 bash JSON 信封、killed=回合
// 中断、ms 为秒级时间戳近似。
import { useEffect, useMemo, useState } from "react";
import { useT } from "../../lib/i18n";
import { app } from "../../lib/bridge";
import { fmtDuration } from "./cards";
import type {
  ContextCommandPage,
  ContextCommandRecord,
  ContextCommandStats,
  ContextNodeDetailView,
} from "../../lib/types/context";
import { NodeDetailPanel } from "./inspector";
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

type CommandDetailState = { state: "loading" } | { state: "ok"; detail: ContextNodeDetailView } | { state: "error" };

export function CommandsCard({
  commands,
  stats,
  sessionPath,
  fetchNodeDetail,
  focusProgram,
  fetchOlderCommands,
}: {
  commands: ContextCommandRecord[];
  stats?: ContextCommandStats;
  sessionPath?: string;
  fetchNodeDetail?: (seq: number) => Promise<ContextNodeDetailView>;
  focusProgram?: { program: string; tick: number } | null;
  fetchOlderCommands?: (beforeSeq: number, limit: number) => Promise<ContextCommandPage>;
}) {
  const t = useT();
  const [query, setQuery] = useState("");
  const [programQuery, setProgramQuery] = useState("");
  const [filter, setFilter] = useState<CmdFilter>("all");
  const [sort, setSort] = useState<CmdSort>("latest");
  const [pickedProgram, setPickedProgram] = useState<string | null>(null);
  const [openSeq, setOpenSeq] = useState<number | null>(null);
  const [details, setDetails] = useState<Map<number, CommandDetailState>>(() => new Map());
  const [olderCommands, setOlderCommands] = useState<ContextCommandRecord[]>([]);
  const [loadOlderState, setLoadOlderState] = useState<"idle" | "loading" | "error">("idle");
  const [atStart, setAtStart] = useState(false);
  const [serverTotal, setServerTotal] = useState(0);

  useEffect(() => {
    setOlderCommands([]);
    setLoadOlderState("idle");
    setAtStart(false);
    setServerTotal(0);
  }, [sessionPath]);

  useEffect(() => {
    if (focusProgram == null) return;
    setPickedProgram(focusProgram.program);
    setQuery(focusProgram.program);
    setFilter("all");
  }, [focusProgram]);

  const toggleDetail = (record: ContextCommandRecord) => {
    setOpenSeq((current) => (current === record.seq ? null : record.seq));
    const seq = record.resultSeq;
    if (seq == null || details.has(seq)) return;
    setDetails((current) => new Map(current).set(seq, { state: "loading" }));
    const read = fetchNodeDetail
      ? fetchNodeDetail(seq)
      : app.ContextNodeDetail(seq, sessionPath ?? "");
    read
      .then((detail) => setDetails((current) => new Map(current).set(seq, { state: "ok", detail })))
      .catch(() => setDetails((current) => new Map(current).set(seq, { state: "error" })));
  };

  const allCommands = useMemo(() => {
    const bySeq = new Map<number, ContextCommandRecord>();
    for (const command of olderCommands) bySeq.set(command.seq, command);
    for (const command of commands) bySeq.set(command.seq, command);
    return [...bySeq.values()].sort((a, b) => a.seq - b.seq);
  }, [commands, olderCommands]);

  const programs = useMemo(() => {
    const m = new Map<string, number>();
    for (const c of allCommands) {
      const p = commandProgram(c.cmd ?? "");
      m.set(p, (m.get(p) ?? 0) + 1);
    }
    return [...m.entries()].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]));
  }, [allCommands]);
  const shownPrograms = useMemo(() => {
    const q = programQuery.trim().toLowerCase();
    return programs.filter(([name]) => q === "" || name.toLowerCase().includes(q));
  }, [programs, programQuery]);

  const searched = useMemo(() => {
    const q = query.trim().toLowerCase();
    return allCommands.filter((c) => {
      if (pickedProgram && commandProgram(c.cmd ?? "") !== pickedProgram) return false;
      if (q && !(c.cmd ?? "").toLowerCase().includes(q)) return false;
      return true;
    });
  }, [allCommands, query, pickedProgram]);
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
    runs: stats?.runs ?? allCommands.length,
    failed: stats?.failed ?? allCommands.filter((c) => c.status === "failed").length,
    ms: stats?.ms ?? allCommands.reduce((sum, c) => sum + durationOf(c), 0),
    bg: stats?.bg ?? allCommands.filter((c) => c.background === true).length,
  }), [allCommands, stats]);
  const slowest = useMemo(
    () => allCommands.reduce<ContextCommandRecord | null>(
      (best, c) => (best === null || durationOf(c) > durationOf(best) ? c : best),
      null,
    ),
    [allCommands],
  );
  const repeated = useMemo(() => {
    const byCommand = new Map<string, number>();
    for (const command of allCommands) {
      const text = (command.cmd ?? "").trim();
      if (text) byCommand.set(text, (byCommand.get(text) ?? 0) + 1);
    }
    return [...byCommand.entries()].sort((a, b) => b[1] - a[1])[0] ?? null;
  }, [allCommands]);

  const totalKnown = Math.max(serverTotal, stats?.runs ?? 0, allCommands.length);
  const hasOlder = !atStart && totalKnown > allCommands.length;
  const loadOlder = async () => {
    if (loadOlderState === "loading" || !hasOlder) return;
    const cursor = allCommands[0]?.seq;
    if (cursor == null) return;
    setLoadOlderState("loading");
    try {
      const page = fetchOlderCommands
        ? await fetchOlderCommands(cursor, 200)
        : await app.ContextCommands(cursor, 200, sessionPath ?? "");
      const incoming = page.commands ?? [];
      if (!page.atStart && !incoming.some((command) => command.seq < cursor)) {
        setServerTotal(page.total);
        setLoadOlderState("error");
        return;
      }
      setOlderCommands((current) => {
        const bySeq = new Map<number, ContextCommandRecord>();
        for (const command of current) bySeq.set(command.seq, command);
        for (const command of incoming) bySeq.set(command.seq, command);
        return [...bySeq.values()].sort((a, b) => a.seq - b.seq);
      });
      setAtStart(page.atStart);
      setServerTotal(page.total);
      setLoadOlderState("idle");
    } catch {
      setLoadOlderState("error");
    }
  };

  if (allCommands.length === 0 && (stats?.runs ?? 0) === 0) return null;

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
                  <button
                    type="button"
                    aria-pressed={filter === "slow" && sort === "slowest"}
                    onClick={() => {
                      const active = filter === "slow" && sort === "slowest";
                      setFilter(active ? "all" : "slow");
                      setSort(active ? "latest" : "slowest");
                    }}
                    title={slowest.cmd}
                  >
                    <i data-tone="slow" />
                    <span>{t("contextview.cmdSlowest")}</span>
                    <em>{slowest.cmd || "—"}</em>
                    <b>{fmtDuration(durationOf(slowest), t)}</b>
                  </button>
                )}
                {repeated && repeated[1] > 1 && (
                  <button
                    type="button"
                    aria-pressed={query === repeated[0]}
                    onClick={() => {
                      setPickedProgram(null);
                      setQuery((current) => current === repeated[0] ? "" : repeated[0]);
                    }}
                    title={repeated[0]}
                  >
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
            {programs.length > 6 && (
              <input
                value={programQuery}
                onChange={(event) => setProgramQuery(event.target.value)}
                placeholder={t("contextview.cmdProgramSearch")}
                aria-label={t("contextview.cmdProgramSearch")}
                data-testid="ctx-cmd-program-search"
                className="ctx-cmd-program-search"
              />
            )}
            <div className="ctx-cmd-programs-list">
              {shownPrograms.map(([name, n], index) => (
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
              {shownPrograms.length === 0 && (
                <span className="ctx-cmd-program-empty">{t("contextview.cmdProgramNone")}</span>
              )}
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
              data-program={pickedProgram ?? undefined}
              className={`ctx-cmd-search${pickedProgram ? " ctx-cmd-search-active" : ""}`}
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
              <CommandRow
                key={c.seq}
                c={c}
                t={t}
                striped={index % 2 === 1}
                open={openSeq === c.seq}
                onToggle={() => toggleDetail(c)}
                detail={c.resultSeq != null ? details.get(c.resultSeq) : undefined}
              />
            ))}
            {filtered.length === 0 && (
              <li className="ctx-cmd-empty">{t("contextview.cmdNone")}</li>
            )}
            {(hasOlder || loadOlderState === "error" || (atStart && allCommands.length > commands.length)) && (
              <li className="ctx-cmd-loadmore" data-testid="ctx-cmd-loadmore">
                {hasOlder ? (
                  <button
                    type="button"
                    data-testid="ctx-cmd-load-older"
                    disabled={loadOlderState === "loading"}
                    onClick={() => { void loadOlder(); }}
                  >
                    {loadOlderState === "loading"
                      ? t("contextview.cmdLoadOlderLoading")
                      : loadOlderState === "error"
                        ? t("contextview.cmdLoadOlderRetry")
                        : t("contextview.cmdLoadOlder", { n: allCommands.length, total: totalKnown })}
                  </button>
                ) : (
                  <span>{t("contextview.cmdLoadedAll", { n: allCommands.length })}</span>
                )}
              </li>
            )}
          </ul>
        </div>
      </div>
    </section>
  );
}

function CommandRow({ c, t, striped, open, onToggle, detail }: {
  c: ContextCommandRecord;
  t: Translator;
  striped: boolean;
  open: boolean;
  onToggle: () => void;
  detail?: CommandDetailState;
}) {
  const dur = durationOf(c);
  return (
    <li
      className={`ctx-cmd-row${striped ? " ctx-cmd-row-alt" : ""}${open ? " ctx-cmd-row-open" : ""}`}
      data-testid={`ctx-cmd-row-${c.seq}`}
    >
      <button
        type="button"
        className="ctx-cmd-row-button"
        aria-expanded={open}
        onClick={onToggle}
      >
        <span className="ctx-cmd-row-state" data-status={c.status} aria-hidden>{c.status === "completed" ? "✓" : c.status === "failed" ? "✕" : "!"}</span>
        <span
          className="ctx-cmd-dot"
          style={{ background: STATUS_COLOR[c.status] }}
          title={c.status}
          data-testid={`ctx-cmd-dot-${c.status}`}
        />
        <span className="min-w-0 flex-1">
          <span className="ctx-cmd-row-main">
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
          </span>
          <span className="ctx-cmd-command" title={c.cmd}>
            {c.cmd || "—"}
          </span>
          {c.err && (
            <span className="ctx-cmd-error">✗ error</span>
          )}
        </span>
        {c.resultSeq != null && <span className="ctx-cmd-chev" aria-hidden>{open ? "▾" : "▸"}</span>}
      </button>
      {open && (
        <div className="ctx-cmd-detail" data-testid="ctx-cmd-detail">
          {c.resultSeq == null ? (
            <div className="px-2 py-1 text-[9.5px] text-fg-faint">{t("contextview.cmdNoResult")}</div>
          ) : detail?.state === "loading" || detail == null ? (
            <div className="px-2 py-1 text-[9.5px] text-fg-faint">{t("contextview.detailLoading")}</div>
          ) : detail.state === "error" ? (
            <div className="px-2 py-1 text-[9.5px] text-err">{t("contextview.detailFail")}</div>
          ) : (
            <NodeDetailPanel d={detail.detail} />
          )}
        </div>
      )}
    </li>
  );
}
