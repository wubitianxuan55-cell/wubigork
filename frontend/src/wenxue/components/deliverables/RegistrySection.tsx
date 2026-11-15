// DeliverablesPanel 拆分——权威产物登记表（P3 结构版2 次批巨文件拆分）。
// 内容自 DeliverablesPanel.tsx 原文件登记表区块整体迁出，逐字节照搬；行为零变化。
import { Table } from "../../icons";
import type { DeliverableEntry } from "../../lib/types";
import { useT } from "../../lib/i18n";
import { fmtRegistryTime } from "./shared";

// v4.24 C1 权威产物登记表：后端从事件日志折叠的写类/生成类落盘登记，
// 只读展示（含启发式漏登的非常规扩展名产物）。无会话路径或后端
// Available=false（legacy 会话）整节收起（准入判断在父级组件）。
export function RegistrySection({ open, onToggleOpen, total, entries, onOpen }: {
  open: boolean;
  onToggleOpen: () => void;
  total: number;
  entries: DeliverableEntry[];
  onOpen: (path: string) => void;
}) {
  const t = useT();
  return (
    <div className="shrink-0 border-t border-(color:--md-sys-color-outline-variant)" data-testid="deliverable-registry">
      <button
        type="button"
        className="w-full flex items-center gap-2 px-3 py-2 text-[11px] cursor-pointer hover:bg-(color:--md-sys-color-surface-container-high)"
        onClick={onToggleOpen}
        aria-expanded={open}
      >
        <Table size={13} aria-hidden style={{ color: "var(--md-sys-color-warning)" }} />
        <span className="font-medium" style={{ color: "var(--md-sys-color-text)" }}>{t("deliverPanel.registryTitle")}</span>
        <span className="truncate text-[10px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
          {total > entries.length
            ? t("deliverPanel.registryCountPartial", { shown: entries.length, total })
            : t("deliverPanel.registryCount", { n: total })}
        </span>
        <span className="v3-panel-spacer" />
        <span className="text-[10px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>{open ? t("common.collapse") : t("common.expand")}</span>
      </button>
      {open && (
        <div className="max-h-44 overflow-y-auto px-2 pb-2 flex flex-col gap-1">
          {entries.map((e) => (
            <div
              key={e.path}
              data-testid="deliverable-registry-row"
              className="flex items-center gap-1.5 px-1.5 py-1 rounded-md bg-(color:--md-sys-color-surface-container) border border-(color:--md-sys-color-outline-variant)"
            >
              <span className="shrink-0 font-mono text-[9px] px-1 py-px rounded" style={{
                color: "var(--md-sys-color-warning)",
                background: "color-mix(in srgb, var(--md-sys-color-warning) 12%, transparent)",
              }} title={t("deliverPanel.toolBadgeTitle", { tool: e.tool })}>
                {e.tool}
              </span>
              <button
                type="button"
                className="min-w-0 flex-1 text-left cursor-pointer truncate font-mono text-[10px]"
                style={{ color: "var(--md-sys-color-text)" }}
                title={t("msg.clickPreview", { path: e.path })}
                onClick={() => onOpen(e.path)}
              >
                {e.path}
              </button>
              <span className="shrink-0 text-[9px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                {e.turn > 0 ? t("deliverPanel.turnN", { n: e.turn }) : t("deliverPanel.turnOut")}
              </span>
              {e.touches > 1 && (
                <span className="shrink-0 text-[9px] font-mono" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                  ×{e.touches}
                </span>
              )}
              <span className="shrink-0 text-[9px] font-mono tabular-nums" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                {fmtRegistryTime(e.updatedAt)}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}