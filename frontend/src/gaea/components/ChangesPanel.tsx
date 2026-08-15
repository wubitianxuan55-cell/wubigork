import { Diff, FileText } from "../icons";
import { useCompact } from "../hooks/useCompact";
import type { SessionChange } from "../lib/changes";

// ── 文件变更面板（Kun 可观察性精华）─────────────────────────────────
// 汇总本会话中 Agent 写/改过的文件（write_file / edit_file / move_file 等），
// 点击行直接打开预览，方便验收前快速核对改动范围。
// v3「星枢」面板语言：v3-panel-head 细条头部；行 hover 柔光、图标/计数走令牌。

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
        <span
          className="w-10 h-10 rounded-[var(--radius-md)] flex items-center justify-center mb-3"
          style={{
            background: "color-mix(in srgb, var(--gaea-glow) 9%, transparent)",
            border: "1px solid color-mix(in srgb, var(--gaea-glow) 20%, transparent)",
            color: "var(--gaea-glow)",
          }}
        >
          <Diff size={18} aria-hidden />
        </span>
        <div className="text-(color:--md-sys-color-text-secondary) text-[12.5px] font-medium">
          本会话暂无文件变更
        </div>
        <div
          className="mt-1 text-[11px] leading-snug max-w-[220px]"
          style={{ color: "var(--md-sys-color-text-secondary)" }}
        >
          Agent 写入或修改工作区文件后，会在这里汇总，点击可打开预览
        </div>
      </div>
    );
  }
  return (
    <div className="flex flex-col py-1">
      {/* v3 细条头部：标题 + 汇总计数 */}
      <div className="v3-panel-head">
        <Diff size={12} aria-hidden style={{ color: "var(--gaea-glow)" }} />
        <span className="v3-panel-title">文件变更</span>
        <span className="v3-panel-spacer" />
        <span className="font-mono text-[10px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
          {changes.length} 个文件 · {totalChanges} 次
        </span>
      </div>
      <div className="flex flex-col gap-px px-1.5 pt-1.5 pb-1">
        {sorted.map((c) => {
          const name = c.path.split(/[\\/]/).filter(Boolean).pop() || c.path;
          const rel = relPath(c.path, cwd);
          return (
            <button
              key={c.path}
              className="group flex items-center gap-2.5 px-2.5 py-2 text-left rounded-[var(--radius-md)] border border-transparent bg-transparent cursor-pointer transition-all duration-200 hover:bg-(color:--md-sys-color-surface-container-high) hover:shadow-[var(--v3-glow-faint)]"
              onClick={() => onOpenFile(c.path)}
              title={`打开 ${rel}`}
              aria-label={`打开 ${rel}`}
            >
              <span
                className="w-7 h-7 rounded-md flex items-center justify-center shrink-0 transition-colors"
                style={{
                  background: "color-mix(in srgb, var(--gaea-glow) 8%, transparent)",
                  border: "1px solid var(--md-sys-color-outline-variant)",
                  color: "var(--md-sys-color-text-secondary)",
                }}
              >
                <FileText size={13} aria-hidden />
              </span>
              <span className="min-w-0 flex-1">
                <span
                  className={`block truncate text-fg-dim font-medium ${
                    compact ? "text-[11.5px]" : "text-[12.5px]"
                  }`}
                >
                  {name}
                </span>
                <span
                  className="block truncate font-mono text-[10px]"
                  style={{ color: "var(--md-sys-color-text-secondary)" }}
                >
                  {rel}
                </span>
              </span>
              <span
                className="shrink-0 rounded-full font-mono text-[10px] px-1.5 py-px"
                style={{
                  background: "color-mix(in srgb, var(--gaea-glow) 10%, transparent)",
                  color: "var(--gaea-glow)",
                  border: "1px solid color-mix(in srgb, var(--gaea-glow) 26%, transparent)",
                }}
              >
                {c.count} 次
              </span>
            </button>
          );
        })}
      </div>
    </div>
  );
}
