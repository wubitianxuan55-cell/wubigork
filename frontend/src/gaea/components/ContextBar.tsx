import { fmtTokens } from "../lib/stats";

// ─── ContextBar（顶栏上下文用量进度条）──────────────────────────

export function ContextBar({ label, used, window: win, color }: { label: string; used: number; window: number; color: string }) {
  const pct = win > 0 ? Math.round((used / win) * 100) : 0;
  const barColor = pct > 80 ? "bg-err" : pct > 60 ? "bg-warning" : color;
  return (
    <div className="flex items-center gap-1">
      <span className="text-fg-faint text-[9px] shrink-0 w-6">{label}</span>
      <div className="flex-1 h-1.5 bg-border/40 rounded-full overflow-hidden min-w-[60px]">
        <div className={`h-full rounded-full transition-all duration-500 ${barColor}`}
          style={{ width: `${Math.min(pct, 100)}%` }} />
      </div>
      <span className="text-fg-dim font-mono tabular-nums text-[9px] shrink-0 w-7 text-right">{pct}%</span>
      <span className="text-fg-faint font-mono tabular-nums text-[8px] shrink-0">{fmtTokens(used)}/{fmtTokens(win)}</span>
    </div>
  );
}
