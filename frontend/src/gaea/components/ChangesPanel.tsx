import { Diff, FileText } from "../icons";
import { useCompact } from "../hooks/useCompact";
import type { SessionChange } from "../lib/changes";

// ── 文件变更面板（Kun 可观察性精华）─────────────────────────────────
// 汇总本会话中 Agent 写/改过的文件（write_file / edit_file / move_file 等），
// 点击行直接打开预览，方便验收前快速核对改动范围。

function relPath(path: string, cwd?: string): string {
  const p = path.replace(/\\/g, "/");
  const base = (cwd || "").replace(/\\/g, "/").replace(/\/+$/, "");
  if (base && p.startsWith(base + "/")) return p.slice(base.length + 1);
  return p;
}

export function ChangesPanel({
  changes,
  cwd,
  onOpenFile,
}: {
  changes: SessionChange[];
  cwd?: string;
  onOpenFile: (path: string) => void;
}) {
  const compact = useCompact();
  const totalChanges = changes.reduce((sum, c) => sum + c.count, 0);
  const sorted = [...changes].sort((a, b) => (b.lastTouched ?? 0) - (a.lastTouched ?? 0));
  if (changes.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 px-6 text-center">
        <span className="w-10 h-10 rounded-xl bg-accent/8 border border-accent/15 text-accent flex items-center justify-center mb-3">
          <Diff size={18} />
        </span>
        <div className="text-fg-dim text-[12.5px] font-medium">本会话暂无文件变更</div>
        <div className="text-fg-faint text-[11px] mt-1 leading-snug max-w-[220px]">
          Agent 写入或修改工作区文件后，会在这里汇总，点击可打开预览
        </div>
      </div>
    );
  }
  return (
    <div className="flex flex-col py-1">
      <div className="flex items-center gap-1.5 px-3 pt-1.5 pb-2 text-fg-faint text-[11px] font-medium tracking-[0.02em]">
        <Diff size={12} className="text-accent" />
        文件变更
        <span className="ml-auto font-mono text-[10px] text-fg-faint/60">
          {changes.length} 个文件 · {totalChanges} 次
        </span>
      </div>
      <div className="flex flex-col gap-px">
        {sorted.map((c) => {
          const name = c.path.split(/[\\/]/).filter(Boolean).pop() || c.path;
          const rel = relPath(c.path, cwd);
          return (
            <button
              key={c.path}
              className="group flex items-center gap-2.5 px-3 py-2 text-left rounded-lg bg-transparent border-0 cursor-pointer transition-colors hover:bg-sidebar-hover"
              onClick={() => onOpenFile(c.path)}
              title={`打开 ${rel}`}
            >
              <span className="w-7 h-7 rounded-md bg-bg-soft/70 border border-border-soft text-fg-faint flex items-center justify-center shrink-0 group-hover:text-accent group-hover:border-accent/25 transition-colors">
                <FileText size={13} />
              </span>
              <span className="min-w-0 flex-1">
                <span className={`block truncate text-fg-dim font-medium ${compact ? "text-[11.5px]" : "text-[12.5px]"}`}>
                  {name}
                </span>
                <span className="block truncate text-fg-faint/70 font-mono text-[10px]">{rel}</span>
              </span>
              <span className="shrink-0 rounded-full bg-accent/10 text-accent font-mono text-[10px] px-1.5 py-px">
                {c.count} 次
              </span>
            </button>
          );
        })}
      </div>
    </div>
  );
}
