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

const STATUS_ORDER: CmdStatus[] = ["completed", "failed", "killed"];

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

export function CommandsCard({ commands, stats }: {
  commands: ContextCommandRecord[];
  stats?: ContextCommandStats;
}) {
  const t = useT();
  const [query, setQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<CmdStatus | null>(null);
  const [pickedProgram, setPickedProgram] = useState<string | null>(null);

  const programs = useMemo(() => {
    const m = new Map<string, number>();
    for (const c of commands) {
      const p = commandProgram(c.cmd ?? "");
      m.set(p, (m.get(p) ?? 0) + 1);
    }
    return [...m.entries()].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0])).slice(0, 8);
  }, [commands]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return commands.filter((c) => {
      if (statusFilter && c.status !== statusFilter) return false;
      if (pickedProgram && commandProgram(c.cmd ?? "") !== pickedProgram) return false;
      if (q && !(c.cmd ?? "").toLowerCase().includes(q)) return false;
      return true;
    });
  }, [commands, query, statusFilter, pickedProgram]);

  if (commands.length === 0) return null;

  return (
    <section className="ctx-card p-3" data-testid="ctx-commands">
      <div className="flex flex-wrap items-center gap-1.5">
        <span className="text-[12.5px] font-semibold text-fg">{t("contextview.cmdTitle")}</span>
        {stats && (
          <span className="flex items-center gap-1.5 font-mono text-[10.5px] text-fg-dim">
            <span data-testid="ctx-cmd-runs">{t("contextview.cmdRuns", { n: stats.runs })}</span>
            {stats.failed > 0 && <span style={{ color: STATUS_COLOR.failed }}>{t("contextview.cmdFailed", { n: stats.failed })}</span>}
            {(stats.bg ?? 0) > 0 && <span>{t("contextview.cmdBg", { n: stats.bg ?? 0 })}</span>}
            {!!stats.ms && stats.ms > 0 && <span title={t("contextview.cmdMsNote")}>{fmtDuration(stats.ms, t)}</span>}
          </span>
        )}
        <span className="ml-auto flex items-center gap-1">
          {(["all", ...STATUS_ORDER] as const).map((s) => (
            <button
              key={s}
              type="button"
              onClick={() => setStatusFilter(s === "all" ? null : s)}
              data-testid={`ctx-cmd-status-${s}`}
              className={`rounded border-0 px-1.5 py-0.5 text-[10.5px] cursor-pointer ${
                (s === "all" ? statusFilter === null : statusFilter === s) ? "bg-accent/15 text-accent" : "text-fg-dim hover:text-fg"
              }`}
            >
              {s === "all" ? t("contextview.cmdAll") : s}
            </button>
          ))}
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t("contextview.cmdSearch")}
            data-testid="ctx-cmd-search"
            className="w-36 rounded border border-border-soft bg-transparent px-1.5 py-0.5 text-[10.5px] text-fg placeholder:text-fg-faint focus:outline-none focus:ring-1 focus:ring-accent/40"
          />
        </span>
      </div>
      {/* 程序分布（全局口径，不随筛选收窄——dsh 同款；点击回填搜索+高亮） */}
      <div className="mt-1.5 flex flex-wrap items-center gap-1">
        <span className="text-[10px] text-fg-faint">{t("contextview.cmdPrograms")}</span>
        {programs.map(([name, n]) => (
          <button
            key={name}
            type="button"
            onClick={() => setPickedProgram((cur) => (cur === name ? null : name))}
            data-picked={pickedProgram === name || undefined}
            className={`cursor-pointer rounded-full border-0 px-1.5 py-px font-mono text-[10px] ${
              pickedProgram === name ? "bg-accent/20 text-accent" : "bg-bg-soft text-fg-dim hover:text-fg"
            }`}
            title={`${name} ×${n}`}
          >
            {name} <span className="opacity-70">×{n}</span>
          </button>
        ))}
      </div>
      <ul className="m-0 mt-1.5 max-h-64 list-none overflow-y-auto p-0" data-testid="ctx-commands-list">
        {filtered.map((c) => (
          <CommandRow key={c.seq} c={c} t={t} />
        ))}
        {filtered.length === 0 && (
          <li className="py-3 text-center text-[11px] text-fg-dim">{t("contextview.cmdNone")}</li>
        )}
      </ul>
    </section>
  );
}

function CommandRow({ c, t }: { c: ContextCommandRecord; t: Translator }) {
  const dur = c.end != null && c.end > c.start ? (c.end - c.start) * 1000 : 0;
  return (
    <li className="ctx-row flex items-start gap-1.5 px-2 py-1.5 text-[10px]">
      <span
        className="mt-1 h-2 w-2 shrink-0 rounded-full"
        style={{ background: STATUS_COLOR[c.status] }}
        title={c.status}
        data-testid={`ctx-cmd-dot-${c.status}`}
      />
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-1.5">
          <span className="font-mono text-fg">{t("contextview.cmdTurnStep", { turn: c.turn, step: c.step })}</span>
          {c.background && (
            <span className="rounded bg-bg-soft px-1 font-mono text-[9.5px] text-fg-dim" title="run_in_background">bg</span>
          )}
          {c.exit != null && c.exit !== 0 && (
            <span className="font-mono" style={{ color: STATUS_COLOR.failed }} title={t("contextview.cmdExitNote")}>exit {c.exit}</span>
          )}
          {c.truncated && <span className="text-warning">{t("contextview.detailClamped")}</span>}
          <span className="ml-auto font-mono tabular-nums text-fg-dim">
            {dur > 0 ? fmtDuration(dur, t) : (c.status === "killed" ? "—" : "")}
          </span>
        </div>
        <div className="mt-0.5 break-all font-mono text-fg-dim" title={c.cmd}>
          {c.cmd || "—"}
        </div>
        {c.err && (
          <div className="mt-0.5 font-mono" style={{ color: STATUS_COLOR.failed }}>✗ error</div>
        )}
      </div>
    </li>
  );
}
