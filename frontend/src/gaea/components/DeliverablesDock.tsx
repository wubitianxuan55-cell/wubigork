import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ChevronRight, Copy, ExternalLink, FolderTree } from "../icons";
import { app } from "../lib/bridge";
import { deliverablePathKey, ensureTurnRegistry, invalidateTurnCaches } from "../lib/deliverablesTurn";
import { useUpdatedFilesStore } from "../lib/store";
import { openPaneFileOrPreview } from "../lib/paneFileOpen";
import { useToast } from "./Toast";
import { useT } from "../lib/i18n";

// DeliverablesDock — 交付文件停靠条（波3.5 实弹反馈：交互文件固定在窗口最
// 下方、输入框上方——消息流里的交付卡跟随正文走，长回合里翻找不方便；停靠
// 条常驻底部=固定查找入口）。
//
// 数据源 = 会话级权威产物登记表（GaeaDeliverableRegistry，updatedAt 倒序、
// 路径去重）；.py 过滤与 DeliverableCards 同口径（脚本属过程件）。
// 刷新信号：写后 updatedFilesStore.updatedAt + 30s TTL 重拉（registryCache
// 失效后 ensureTurnRegistry 自然重取）。
// 默认展开最近 6 条，「更多」展开全部；无条目不占位。整行点击开预览，
// 悬停外开/定位/复制路径。

function extOf(path: string): string {
  const m = /\.[^.\\/]+$/.exec(path);
  return m ? m[0].toLowerCase() : "";
}

function baseName(path: string): string {
  return path.split(/[\\/]/).pop() ?? path;
}

const DEFAULT_VISIBLE = 6;
const REFRESH_DEDUPE_MS = 30_000;

export function DeliverablesDock() {
  const updatedAt = useUpdatedFilesStore((s) => s.updatedAt);
  const toast = useToast();
  const t = useT();
  const [entries, setEntries] = useState<{ path: string; tool: string; updatedAt: number }[] | null>(null);
  const [expanded, setExpanded] = useState(false);
  const [collapsedUI, setCollapsedUI] = useState(false);

  const refresh = useCallback(async () => {
    try {
      const view = await ensureTurnRegistry(Date.now());
      const list = (view?.entries ?? [])
        .filter((e) => extOf(e.path) !== ".py")
        .map((e) => ({ path: e.path, tool: e.tool, updatedAt: e.updatedAt }));
      setEntries(list);
    } catch {
      setEntries((cur) => cur ?? []);
    }
  }, []);

  // 首拉 + 写后刷新（30s 去重；updatedAt 变化=有新写盘）。
  const lastRefreshRef = useRef(0);
  useEffect(() => {
    const now = Date.now();
    if (now - lastRefreshRef.current < REFRESH_DEDUPE_MS && entries !== null) return;
    lastRefreshRef.current = now;
    void refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps -- entries 只用于去重判定
  }, [updatedAt, refresh]);

  // 会话切换（登记表随会话走）：强制失效重拉。
  useEffect(() => {
    invalidateTurnCaches();
    lastRefreshRef.current = 0;
    void refresh();
  }, [refresh]);

  const copyPath = async (path: string) => {
    try {
      await navigator.clipboard.writeText(path);
      toast.show(t("deliver.copyPathDone"), "info");
    } catch {
      toast.show(t("deliver.copyFail"), "warn");
    }
  };

  const iconBtn =
    "flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-(color:--md-sys-color-text-secondary) cursor-pointer hover:text-(color:--md-sys-color-text) hover:bg-(color:--md-sys-color-surface-container-high) transition-colors";

  const visible = useMemo(
    () => (entries ?? []).slice(0, expanded ? undefined : DEFAULT_VISIBLE),
    [entries, expanded],
  );
  const hiddenCount = (entries?.length ?? 0) - visible.length;
  if (entries === null || entries.length === 0) return null;

  return (
    <div
      data-testid="deliverables-dock"
      className="mx-2 mb-1 rounded-lg overflow-hidden"
      style={{
        border: "1px solid color-mix(in srgb, var(--md-sys-color-outline-variant) 55%, transparent)",
        background: "color-mix(in srgb, var(--md-sys-color-surface-container) 35%, transparent)",
      }}
    >
      <button
        type="button"
        className="flex items-center gap-1.5 w-full select-none px-3 py-1.5 cursor-pointer hover:bg-(color:--md-sys-color-surface-container-high)/50"
        onClick={() => setCollapsedUI((v) => !v)}
        aria-expanded={!collapsedUI}
      >
        <span className="text-[10px] uppercase tracking-wider font-medium" style={{ color: "var(--md-sys-color-text-secondary)" }}>
          {t("deliver.title")}
        </span>
        <span
          className="rounded-full px-1.5 py-px text-[9px] font-mono leading-none"
          style={{ color: "var(--md-sys-color-text-secondary)", border: "1px solid var(--md-sys-color-outline-variant)" }}
        >
          {entries.length}
        </span>
        <ChevronRight
          size={12}
          className={`ml-auto transition-transform ${collapsedUI ? "" : "rotate-90"}`}
          style={{ color: "var(--md-sys-color-text-secondary)" }}
        />
      </button>
      {!collapsedUI && (
        <div className="flex flex-col">
          {visible.map(({ path, updatedAt: at }, idx) => {
            const updated = updatedAt[path] != null;
            return (
              <div
                key={deliverablePathKey(path)}
                className="group flex items-center gap-2.5 px-3 py-1.5 transition-colors duration-150 hover:bg-(color:--md-sys-color-surface-container-high)"
                style={{ borderTop: idx > 0 ? "1px solid color-mix(in srgb, var(--md-sys-color-outline-variant) 40%, transparent)" : undefined }}
              >
                <button
                  type="button"
                  onClick={() => openPaneFileOrPreview(path)}
                  title={t("msg.clickPreview", { path })}
                  className="min-w-0 flex-1 text-left cursor-pointer rounded-md px-1 py-0.5 -mx-1 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-(color:--gaea-glow)/40"
                >
                  <span className="flex items-center gap-1.5 min-w-0">
                    <span className="truncate text-[12.5px] font-medium leading-tight" style={{ color: "var(--md-sys-color-text)" }}>
                      {baseName(path)}
                    </span>
                    {updated && (
                      <span
                        className="shrink-0 rounded-full px-1.5 py-px text-[9px] leading-none"
                        style={{
                          color: "var(--md-sys-color-success)",
                          background: "color-mix(in srgb, var(--md-sys-color-success) 12%, transparent)",
                          border: "1px solid var(--md-sys-color-success)",
                        }}
                      >
                        {t("deliver.updated")}
                      </span>
                    )}
                  </span>
                  <span
                    className="block truncate text-[10px] font-mono leading-tight mt-px"
                    style={{ color: "var(--md-sys-color-text-secondary)" }}
                  >
                    {path}
                  </span>
                </button>
                <span className="shrink-0 text-[10px] text-fg-faint/70 tabular-nums">
                  {at > 0 ? new Date(at * 1000).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }) : ""}
                </span>
                <div className="shrink-0 flex items-center gap-0.5 opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 transition-opacity">
                  <button type="button" className={iconBtn} onClick={() => void copyPath(path)} title={t("deliver.copyPath")} aria-label={t("deliver.copyPath")}>
                    <Copy size={12} />
                  </button>
                  <button type="button" className={iconBtn} onClick={() => void app.OpenWorkspacePath(path).catch(() => {})} title={t("deliver.openExternal")} aria-label={t("deliver.openExternal")}>
                    <ExternalLink size={12} />
                  </button>
                  <button type="button" className={iconBtn} onClick={() => void app.RevealWorkspacePath(path).catch(() => {})} title={t("deliver.reveal")} aria-label={t("deliver.reveal")}>
                    <FolderTree size={12} />
                  </button>
                </div>
              </div>
            );
          })}
          {hiddenCount > 0 && (
            <button
              type="button"
              className="px-3 py-1.5 text-[11px] text-left cursor-pointer hover:bg-(color:--md-sys-color-surface-container-high)/50"
              style={{ color: "var(--md-sys-color-text-secondary)", borderTop: "1px solid color-mix(in srgb, var(--md-sys-color-outline-variant) 40%, transparent)" }}
              onClick={() => setExpanded(true)}
            >
              … 还有 {hiddenCount} 个（点击展开全部）
            </button>
          )}
        </div>
      )}
    </div>
  );
}
