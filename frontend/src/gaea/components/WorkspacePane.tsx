import { createElement, useEffect, type ComponentType } from "react";
import { Dropdown } from "antd";
import { usePaneTabsStore } from "../lib/paneTabs";
import { normalizePreviewPath } from "../lib/officeTurnProjection";
import { WORKSPACE_TABS, type WorkspaceTabId } from "../lib/workspaceTabs";
import { getWorkspaceRegistration, type WorkspacePanelContext } from "../lib/sidebarRegistry";
import { FileTypeIcon } from "../lib/fileIcon";
import { FilePreview } from "./FilePreview";
import { BrowserPane } from "./BrowserPane";

// WorkspacePane — 右栏工作台 pane（对标 dsh-better-sidebar）。
//
// 语义：
//   1. tabs 为空 = 欢迎卡片（点卡片才开对应视图 tab）；
//   2. 点「文件」卡片 → view:files tab（资源管理器视图本身只是其中一个 tab）；
//   3. 资源管理器里点文件 → file:<path> tab（文件浏览/编辑），与资源管理器
//      并存于同一 tab 条；
//   4. 产物/任务/浏览器同样是视图 tab（复用 sidebarRegistry 现有渲染器）；
//   5. 关闭全部 tab 回到欢迎卡片态。
//
// 文件 tab 内容复用 FilePreview（embedded：docx/xlsx/md 等能力原样保留）。

const TAB_STRIP_BTN =
  "relative flex items-center gap-1 pl-2 pr-1 py-1 rounded-t-md border-0 bg-transparent cursor-pointer text-[11px] select-none shrink-0 transition-colors";
const TAB_STRIP_ACTIVE = "text-accent bg-accent/10";
const TAB_STRIP_IDLE = "text-fg-dim hover:bg-bg-soft";

export function WorkspacePane({
  context,
  badges,
  onActiveViewChange,
}: {
  context: WorkspacePanelContext;
  /** 视图级计数角标（tasks=运行中、deliverables=新产物）；0/缺省不显示。 */
  badges?: Partial<Record<WorkspaceTabId, number>>;
  /** 激活视图变化通知（App 用于角标已读/命令面板重排等）。 */
  onActiveViewChange?: (viewId: WorkspaceTabId | null) => void;
}) {
  const tabs = usePaneTabsStore((s) => s.tabs);
  const active = usePaneTabsStore((s) => s.active);
  // U4 写后预览实时跟随：文件 tab 的 FilePreview 消费 reloadTicks 总线
  // （App 从 office 写类工具成功回执派生，800ms 防抖合并连写后递增）。
  const reloadTicks = usePaneTabsStore((s) => s.reloadTicks);
  const activeTab = tabs.find((t) => t.id === active) ?? tabs[tabs.length - 1];

  const api = usePaneTabsStore.getState;
  const openView = (viewId: WorkspaceTabId, title: string) => api().openView(viewId, title);
  const closeTab = (id: string) => api().close(id);
  const activateTab = (id: string) => api().activate(id);

  // 激活变化 → 通知外层（产物/任务角标已读、命令面板重排）
  useEffect(() => {
    const viewId = activeTab?.kind === "view" && activeTab.viewId
      ? (activeTab.viewId as WorkspaceTabId)
      : activeTab?.kind === "file"
        ? "files"
        : null;
    onActiveViewChange?.(viewId);
  }, [activeTab?.id, activeTab?.kind, activeTab?.viewId, onActiveViewChange]);

  const badgeOf = (viewId: WorkspaceTabId): number => {
    const n = badges?.[viewId];
    return typeof n === "number" && n > 0 ? n : 0;
  };

  // ── 欢迎卡片态（无任何 tab） ───────────────────────────────
  if (tabs.length === 0) {
    return (
      <div data-workspace-pane className="flex flex-col h-full min-h-0">
        <div className="px-3 pt-2.5 pb-1 text-[11px] text-fg-dim">
          选择要打开的工作台页面
        </div>
        <div className="grid grid-cols-2 gap-2 px-3 py-2">
          {WORKSPACE_TABS.map((tab) => {
            const count = badgeOf(tab.id);
            return (
              <button
                key={tab.id}
                type="button"
                data-welcome-card={tab.id}
                className="flex flex-col items-start gap-2 rounded-xl border border-border-soft bg-bg-elev px-3 py-2.5 text-left cursor-pointer transition-colors hover:bg-bg-soft"
                onClick={() => openView(tab.id, tab.label)}
              >
                <span className="flex w-full items-center gap-1.5">
                  <tab.icon size={15} aria-hidden style={{ color: "var(--gaea-glow)" }} />
                  {count > 0 && (
                    <span className="ml-auto inline-flex min-w-4 h-4 items-center justify-center rounded-full px-1 text-[9px] font-medium"
                      style={{ background: "var(--gaea-glow)", color: "var(--color-on-primary)" }}
                    >
                      {count > 99 ? "99+" : count}
                    </span>
                  )}
                </span>
                <span className="text-[12px] font-medium text-fg">{tab.label}</span>
              </button>
            );
          })}
        </div>
      </div>
    );
  }

  // ── 内容渲染：视图 tab / 文件 tab ─────────────────────────
  let content: React.ReactNode = null;
  if (activeTab?.kind === "file" && activeTab.path) {
    content = (
      <FilePreview
        relPath={activeTab.path}
        onClose={() => closeTab(activeTab.id)}
        onBackToFiles={() => openView("files", "文件")}
        embedded
        reloadSignal={reloadTicks[normalizePreviewPath(activeTab.path)] ?? 0}
      />
    );
  } else if (activeTab?.kind === "view" && activeTab.viewId === "browser") {
    content = (
      <BrowserPane
        url={activeTab.meta?.url}
        onUrlChange={(u) => api().updateTabUrl(activeTab.id, u)}
      />
    );
  } else if (activeTab?.kind === "view" && activeTab.viewId) {
    // 视图 tab 统一走注册表渲染器（files=资源管理器，点文件开 pane 文件 tab；
    // 产物/任务/浏览器复用各自面板）
    content = getWorkspaceRegistration(activeTab.viewId as WorkspaceTabId).render(context);
  }

  return (
    <div data-workspace-pane className="flex flex-col h-full min-h-0">
      {/* pane tab 条：视图 tab 与文件 tab 混排（better-sidebar 语义） */}
      <div
        className="flex items-stretch gap-0.5 px-1.5 pt-1 overflow-x-auto shrink-0 border-b border-border-soft"
        role="tablist"
        aria-label="工作台标签页"
      >
        {tabs.map((tab) => {
          const isActive = tab.id === activeTab?.id;
          const icon =
            tab.kind === "view"
              ? (() => {
                  const def = WORKSPACE_TABS.find((d) => d.id === tab.viewId);
                  return def ? createElement(def.icon, { size: 12 }) : null;
                })()
              : createElement(FileTypeIcon as ComponentType<{ name: string; size?: number }>, {
                  name: (tab.path ?? "").split(/[\\/]/).pop() ?? tab.title,
                  size: 12,
                });
          const count =
            tab.kind === "view" && tab.viewId ? badgeOf(tab.viewId as WorkspaceTabId) : 0;
          return (
            <div
              key={tab.id}
              role="tab"
              aria-selected={isActive}
              tabIndex={0}
              title={tab.kind === "file" ? tab.path : tab.title}
              data-pane-tab={tab.id}
              className={`${TAB_STRIP_BTN} ${isActive ? TAB_STRIP_ACTIVE : TAB_STRIP_IDLE}`}
              onClick={() => activateTab(tab.id)}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  activateTab(tab.id);
                }
              }}
            >
              {icon}
              <span className="truncate max-w-[120px]">{tab.title}</span>
              {count > 0 && (
                <span
                  className="inline-flex min-w-3.5 h-3.5 items-center justify-center rounded-full px-1 text-[8px] font-medium"
                  style={{ background: "var(--gaea-glow)", color: "var(--color-on-primary)" }}
                >
                  {count > 99 ? "99+" : count}
                </span>
              )}
              <button
                type="button"
                aria-label={`关闭 ${tab.title}`}
                title={`关闭 ${tab.title}`}
                className="flex items-center justify-center w-4 h-4 rounded border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors shrink-0"
                onClick={(e) => {
                  e.stopPropagation();
                  closeTab(tab.id);
                }}
              >
                <svg width="9" height="9" viewBox="0 0 10 10" aria-hidden>
                  <path d="M1 1l8 8M9 1L1 9" stroke="currentColor" strokeWidth="1.4" />
                </svg>
              </button>
            </div>
          );
        })}
        <Dropdown
          trigger={["click"]}
          menu={{
            items: WORKSPACE_TABS.map((t) => ({ key: t.id, label: t.label })),
            onClick: ({ key }) => {
              const def = WORKSPACE_TABS.find((t) => t.id === key);
              if (def) openView(def.id, def.label);
            },
          }}
        >
          <button
            type="button"
            title="新建标签页"
            aria-label="新建标签页"
            className="shrink-0 flex items-center justify-center w-5 h-5 my-1 ml-0.5 rounded border border-dashed border-border-strong bg-transparent text-fg-faint cursor-pointer hover:text-accent hover:border-accent transition-colors"
          >
            <span aria-hidden>＋</span>
          </button>
        </Dropdown>
      </div>
      <div className="flex-1 min-h-0 overflow-hidden">{content}</div>
    </div>
  );
}
