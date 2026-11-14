import { useMemo, useState } from "react";
import { ChevronRight, Copy, ExternalLink, FolderTree } from "../icons";
import { app } from "../lib/bridge";
import { deliverablePathKey } from "../lib/deliverablesTurn";
import { fileBadge, fileCardKind } from "../lib/fileCard";
import { useUpdatedFilesStore } from "../lib/store";
import { openPaneFileOrPreview } from "../lib/paneFileOpen";
import { useToast } from "./Toast";
import { useT } from "../lib/i18n";

// DeliverablesDock — 交付文件停靠条（波3.5 实弹反馈：交互文件固定在窗口最
// 下方、输入框上方——消息流里的交付卡跟随正文走，长回合里翻找不方便；停靠
// 条常驻底部=固定查找入口）。
//
// 数据源 = App 的 sessionDeliverables（与右侧「会话产物」同源，按首现顺序、
// 路径去重）；.py 过滤与 DeliverableCards 同口径（脚本属过程件）。
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
export function DeliverablesDock({ items }: { items: ReadonlyArray<{ path: string }> }) {
  const updatedAt = useUpdatedFilesStore((s) => s.updatedAt);
  const toast = useToast();
  const t = useT();
  const [expanded, setExpanded] = useState(false);
  const [collapsedUI, setCollapsedUI] = useState(true);
  // 与右侧「会话产物」共用 App 的同一份会话派生列表：文本引用 + 写工具
  // 显式登记一次算清，既能在会话切换时即时更新，也不会出现正文/右栏已有
  // 多项文件、输入框上方却因另一份登记表只显示 3 项的数量漂移。
  const entries = useMemo(
    () => items.filter((item) => extOf(item.path) !== ".py"),
    [items],
  );

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
    () => entries.slice(0, expanded ? undefined : DEFAULT_VISIBLE),
    [entries, expanded],
  );
  const hiddenCount = entries.length - visible.length;
  if (entries.length === 0) return null;

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
