import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ChevronRight, Copy, ExternalLink, FolderTree } from "../icons";
import { app } from "../lib/bridge";
import { deliverablePathKey, ensureTurnRegistry, invalidateTurnCaches } from "../lib/deliverablesTurn";
import { fileBadge, fileCardKind } from "../lib/fileCard";
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
// 默认收起为单行摘要；展开显示最近 6 张紧凑成果卡，「更多」展开全部。
// 无条目不占位。卡片点击开预览，悬停外开/定位/复制路径。

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
  const [collapsedUI, setCollapsedUI] = useState(true);

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
      className="office-delivery-dock"
    >
      <button
        type="button"
        className="office-delivery-dock__head"
        onClick={() => setCollapsedUI((v) => !v)}
        aria-expanded={!collapsedUI}
      >
        <ChevronRight
          size={12}
          className="office-delivery-dock__chevron"
          data-open={collapsedUI ? "false" : "true"}
          aria-hidden
        />
        <span className="office-delivery-dock__title">
          {t("deliver.title")}
        </span>
        <span className="office-delivery-dock__count">
          {entries.length}
        </span>
        <span className="office-delivery-dock__hint">
          {collapsedUI ? "点击展开" : "点击收起"}
        </span>
      </button>
      {!collapsedUI && (
        <div className="office-delivery-dock__body">
          <div className="office-delivery-dock__grid">
          {visible.map(({ path }) => {
            const updated = updatedAt[path] != null;
            const kind = fileCardKind(path);
            return (
              <div
                key={deliverablePathKey(path)}
                className="office-file-card office-file-card--compact group"
                data-kind={kind}
              >
                <button
                  type="button"
                  onClick={() => openPaneFileOrPreview(path)}
                  title={`${t("msg.clickPreview", { path })}\n${path}`}
                  aria-label={`${baseName(path)} — ${path}`}
                  className="office-file-card__open"
                >
                  <span className="office-file-card__badge" aria-hidden>
                    {fileBadge(path, kind)}
                  </span>
                  <span className="office-file-card__copy">
                    <span className="office-file-card__name">{baseName(path)}</span>
                    {updated && (
                      <span className="office-file-card__meta">
                        <span className="office-file-card__chip" data-tone="success">
                          {t("deliver.updated")}
                        </span>
                      </span>
                    )}
                  </span>
                </button>
                <div className="office-file-card__actions">
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
          </div>
          {hiddenCount > 0 && (
            <button
              type="button"
              className="office-delivery-dock__more"
              onClick={() => setExpanded(true)}
            >
              还有 {hiddenCount} 个成果
            </button>
          )}
        </div>
      )}
    </div>
  );
}
