// KnowledgePanel 拆分——左栏单条知识条目行（page variant）（P3 结构版2 次批巨文件拆分）。
// 内容自 KnowledgePanel.tsx 原文件 renderEntryRow 整体迁出，逐字节照搬；行为零变化。
import type { KnowledgeSummary } from "../../lib/types";
import { highlightText, phaseOf } from "./utils";

// 左栏：单条知识条目（page variant 紧凑行；激活 = 主色容器 + 左缘光条）
export function EntryRow({ entry, active, selected, normalizedQuery, onToggle, onSelect }: {
  entry: KnowledgeSummary;
  active: boolean;
  selected: boolean;
  normalizedQuery: string;
  onToggle: (name: string) => void;
  onSelect: (name: string) => void;
}) {
  const phaseVal = phaseOf(entry);
  return (
    <div className="relative">
      <div
        className={`flex items-start gap-1.5 px-2 py-1.5 rounded-lg transition-colors ${active ? "" : "hover:bg-bg-soft"}`}
        style={active ? {
          background: "var(--color-primary-container, var(--md-sys-color-primary-container))",
          boxShadow: "var(--v3-glow-faint)",
        } : undefined}
      >
        {active && (
          <span aria-hidden="true" className="absolute left-0 top-2 bottom-2 w-[3px] rounded-r" style={{ background: "var(--gaea-glow)", boxShadow: "0 0 8px var(--gaea-glow)" }} />
        )}
        <input
          type="checkbox"
          checked={selected}
          onChange={() => onSelect(entry.name)}
          title="多选（批量删除/改状态）"
          aria-label={`选择 ${entry.title}`}
          className="mt-1.5 shrink-0 cursor-pointer"
        />
        <button type="button" onClick={() => onToggle(entry.name)} aria-expanded={active}
          className="flex-1 min-w-0 text-left flex flex-col gap-0.5 cursor-pointer">
          <span className="flex items-start gap-1.5">
            <span className="flex-1 text-[12.5px] font-medium leading-snug truncate text-fg">
              {normalizedQuery ? highlightText(entry.title, normalizedQuery) : entry.title}
            </span>
            <span className="shrink-0 text-[9.5px] font-medium px-1.5 py-0.5 rounded-full"
              style={{
                background: "var(--accent-soft, color-mix(in srgb, var(--accent, var(--md-sys-color-primary)) 12%, transparent))",
                color: "var(--accent, var(--md-sys-color-primary))",
              }}>
              {entry.category}
            </span>
          </span>
          <span className="flex items-center gap-1.5 text-[10px] text-fg-faint">
            {phaseVal && <span>{phaseVal}</span>}
            {phaseVal && <span aria-hidden="true">·</span>}
            {entry.status && <span>{entry.status}</span>}
            {entry.updatedAt && <span className="ml-auto tabular-nums">{new Date(entry.updatedAt).toLocaleDateString()}</span>}
          </span>
          {entry.tags.length > 0 && (
            <span className="flex flex-wrap gap-1">
              {entry.tags.slice(0, 3).map((tag) => (
                <span key={tag} className="text-[9.5px] text-fg-faint px-1.5 py-0.5 rounded-full bg-bg-soft">
                  {normalizedQuery ? highlightText(tag, normalizedQuery) : tag}
                </span>
              ))}
              {entry.tags.length > 3 && <span className="text-[9.5px] text-fg-faint">+{entry.tags.length - 3}</span>}
            </span>
          )}
        </button>
      </div>
    </div>
  );
}