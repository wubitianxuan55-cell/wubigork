import {
  createElement,
  useEffect,
  useId,
  useRef,
  type ComponentType,
  type KeyboardEvent,
} from "react";
import { Dropdown, Popover, Switch } from "antd";
import type { MenuProps } from "antd";
import { usePaneTabsStore } from "../lib/paneTabs";
import { usePaneEnabledStore } from "../lib/paneEnabled";
import { normalizePreviewPath } from "../lib/officeTurnProjection";
import { WORKSPACE_TABS, type WorkspaceTabId } from "../lib/workspaceTabs";
import { getWorkspaceRegistration, type WorkspacePanelContext } from "../lib/sidebarRegistry";
import { FileTypeIcon } from "../lib/fileIcon";
import { confirmDiscardEditor } from "../lib/editorDrafts";
import { FilePreview } from "./FilePreview";
import { BrowserPane } from "./BrowserPane";
import { ChevronRight, PanelRightClose, Plus, Settings } from "../icons";
import "../../gaea/components/workspace-pane.css";

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

const WORKSPACE_TAB_COPY: Record<WorkspaceTabId, string> = {
  files: "浏览作品文件与目录",
  deliverables: "查看交付物与文件变更",
  browser: "观察网页操作与截图",
  git: "状态、暂存与提交",
  ui: "模型生成的可交互面板",
  gallery: "封面、场景插图与角色立绘",
};

function WorkspacePanelManager({
  enabled,
  setEnabled,
}: {
  enabled: Record<WorkspaceTabId, boolean>;
  setEnabled: (id: WorkspaceTabId, value: boolean) => void;
}) {
  return (
    <div className="workspace-manager" role="group" aria-label="管理工作台面板">
      <div className="workspace-manager__head">
        <strong>工作台面板</strong>
        <span>{WORKSPACE_TABS.filter((tab) => enabled[tab.id]).length}/{WORKSPACE_TABS.length} 已启用</span>
      </div>
      <div className="workspace-manager__list">
        {WORKSPACE_TABS.map((tab) => (
          <label key={tab.id} className="workspace-manager__row">
            <span className="workspace-manager__icon" aria-hidden="true">
              <tab.icon size={13} />
            </span>
            <span className="workspace-manager__copy">
              <strong>{tab.label}</strong>
              <span>{WORKSPACE_TAB_COPY[tab.id]}</span>
            </span>
            <Switch
              size="small"
              checked={enabled[tab.id]}
              aria-label={`启用 ${tab.label}`}
              data-testid={`tab-enable-${tab.id}`}
              onChange={(value) => setEnabled(tab.id, value)}
            />
          </label>
        ))}
      </div>
      <p className="workspace-manager__note">停用只隐藏入口，已经打开的页签不会被强制关闭。</p>
    </div>
  );
}

function WorkspaceIllustration() {
  return (
    <svg className="workspace-illustration" viewBox="0 0 112 78" aria-hidden="true" focusable="false">
      <path className="workspace-illustration__route" d="M17 61C27 43 43 29 62 25c16-4 29-1 38 8" />
      <circle className="workspace-illustration__node" cx="17" cy="61" r="3" />
      <circle className="workspace-illustration__node workspace-illustration__node--accent" cx="99" cy="33" r="3" />
      <rect className="workspace-illustration__window" x="22" y="10" width="62" height="45" rx="7" />
      <path className="workspace-illustration__hairline" d="M22 20h62" />
      <circle className="workspace-illustration__dot" cx="29" cy="15" r="1.4" />
      <circle className="workspace-illustration__dot" cx="34" cy="15" r="1.4" />
      <circle className="workspace-illustration__dot" cx="39" cy="15" r="1.4" />
      <rect className="workspace-illustration__paper" x="42" y="28" width="52" height="36" rx="6" />
      <path className="workspace-illustration__muted" d="M50 37h25M50 42h31M50 47h20" />
      <rect className="workspace-illustration__bar" x="51" y="53" width="5" height="5" rx="1.5" />
      <rect className="workspace-illustration__bar workspace-illustration__bar--mid" x="59" y="50" width="5" height="8" rx="1.5" />
      <rect className="workspace-illustration__bar workspace-illustration__bar--accent" x="67" y="48" width="5" height="10" rx="1.5" />
      <path className="workspace-illustration__tick" d="M84 28v-9M84 19l-4 4M84 19l4 4" />
      <path className="workspace-illustration__spark" d="M88 12h7M91.5 8.5v7" />
    </svg>
  );
}

export function WorkspacePane({
  context,
  badges,
  onActiveViewChange,
}: {
  context: WorkspacePanelContext;
  /** 视图级计数角标（deliverables=新产物）；0/缺省不显示。 */
  badges?: Partial<Record<WorkspaceTabId, number>>;
  /** 激活视图变化通知（App 用于角标已读/命令面板重排等）。 */
  onActiveViewChange?: (viewId: WorkspaceTabId | null) => void;
}) {
  const tabs = usePaneTabsStore((s) => s.tabs);
  const active = usePaneTabsStore((s) => s.active);
  // v4.524 可选卡红线：入口按启用集收敛（cad 默认关；store 为欢迎网格/＋下拉/
  // 命令面板三入口的响应式单源）。已开 tab 不因停用强关——启用集只管入口。
  const enabled = usePaneEnabledStore((s) => s.enabled);
  const setEnabled = usePaneEnabledStore((s) => s.setEnabled);
  const enabledTabs = WORKSPACE_TABS.filter((t) => enabled[t.id]);
  const disabledTabs = WORKSPACE_TABS.filter((t) => !enabled[t.id]);
  // U4 写后预览实时跟随：文件 tab 的 FilePreview 消费 reloadTicks 总线
  // （App 从 office 写类工具成功回执派生，800ms 防抖合并连写后递增）。
  const reloadTicks = usePaneTabsStore((s) => s.reloadTicks);
  const activeTab = tabs.find((t) => t.id === active) ?? tabs[tabs.length - 1];
  const activeTabId = activeTab?.id;
  const paneId = useId();
  const stripRef = useRef<HTMLDivElement>(null);
  const tabRefs = useRef<Record<string, HTMLButtonElement | null>>({});

  const api = usePaneTabsStore.getState;
  const openView = (viewId: WorkspaceTabId, title: string) => api().openView(viewId, title);
  const closeTab = (id: string) => {
    const tab = tabs.find((item) => item.id === id);
    confirmDiscardEditor(tab?.kind === "file" ? tab.path : undefined, () => api().close(id));
  };
  const activateTab = (id: string) => {
    if (id === active) return;
    confirmDiscardEditor(activeTab?.kind === "file" ? activeTab.path : undefined, () => api().activate(id));
  };

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

  const manageContent = <WorkspacePanelManager enabled={enabled} setEnabled={setEnabled} />;

  useEffect(() => {
    if (!activeTabId) return;
    tabRefs.current[activeTabId]?.scrollIntoView?.({ block: "nearest", inline: "nearest" });
  }, [activeTabId]);

  // ── 欢迎卡片态（无任何 tab） ───────────────────────────────
  if (tabs.length === 0) {
    const featuredTab = enabledTabs[0];
    const secondaryTabs = enabledTabs.slice(1);
    return (
      <section data-workspace-pane className="workspace-shell workspace-shell--home" aria-label="工作台">
        <header className="workspace-home__head">
          <div className="workspace-home__topline">
            <span className="workspace-home__eyebrow">STORY WORKSPACE</span>
            <div className="workspace-home__actions">
              <Popover content={manageContent} trigger="click" placement="bottomRight">
                <button
                  type="button"
                  data-testid="tabs-manage"
                  className="workspace-icon-button"
                  aria-label="管理工作台面板"
                  title="管理工作台面板"
                >
                  <Settings size={14} aria-hidden />
                </button>
              </Popover>
              <button
                type="button"
                className="workspace-icon-button"
                aria-label="收起工作台"
                title="收起工作台"
                onClick={() => context.onClosePanel?.()}
              >
                <PanelRightClose size={14} aria-hidden />
              </button>
            </div>
          </div>
          <div className="workspace-home__hero">
            <div className="workspace-home__heading">
              <h2>工作台</h2>
              <p>选择要打开的工作台页面</p>
            </div>
            <WorkspaceIllustration />
          </div>
        </header>

        <div className="workspace-home__scroll">
          {featuredTab && (
            <button
              type="button"
              data-welcome-card={featuredTab.id}
              className="workspace-launch-feature"
              aria-label={`打开${featuredTab.label}`}
              onClick={() => openView(featuredTab.id, featuredTab.label)}
            >
              <span className="workspace-launch-feature__icon" aria-hidden="true">
                <featuredTab.icon size={19} />
              </span>
              <span className="workspace-launch-feature__copy">
                <strong>{featuredTab.label}</strong>
                <span>{WORKSPACE_TAB_COPY[featuredTab.id]}</span>
              </span>
              <span className="workspace-launch-feature__action" aria-hidden="true">
                <ChevronRight size={14} />
              </span>
            </button>
          )}

          {secondaryTabs.length > 0 && (
            <div className="workspace-launch-grid">
              {secondaryTabs.map((tab) => {
                const count = badgeOf(tab.id);
                return (
                  <button
                    key={tab.id}
                    type="button"
                    data-welcome-card={tab.id}
                    className="workspace-launch-card"
                    aria-label={`打开${tab.label}`}
                    onClick={() => openView(tab.id, tab.label)}
                  >
                    <span className="workspace-launch-card__top">
                      <span className="workspace-launch-card__icon" aria-hidden="true">
                        <tab.icon size={15} />
                      </span>
                      {count > 0 && <span className="workspace-launch-badge">{count > 99 ? "99+" : count}</span>}
                    </span>
                    <strong>{tab.label}</strong>
                    <span>{WORKSPACE_TAB_COPY[tab.id]}</span>
                  </button>
                );
              })}
            </div>
          )}

          <footer className="workspace-home__foot">
            <span>{enabledTabs.length} 个面板可用</span>
            <span>{disabledTabs.length > 0 ? `${disabledTabs.length} 个已隐藏` : "全部面板已显示"}</span>
          </footer>
        </div>
      </section>
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

  const activeIndex = Math.max(0, tabs.findIndex((tab) => tab.id === activeTab?.id));
  const panelId = `${paneId}-panel`;
  const tabIdOf = (index: number) => `${paneId}-tab-${index}`;

  const onTabKeyDown = (event: KeyboardEvent<HTMLElement>, index: number) => {
    if (!["ArrowLeft", "ArrowRight", "Home", "End"].includes(event.key)) return;
    event.preventDefault();
    let nextIndex = index;
    if (event.key === "Home") nextIndex = 0;
    else if (event.key === "End") nextIndex = tabs.length - 1;
    else if (event.key === "ArrowLeft") nextIndex = index <= 0 ? tabs.length - 1 : index - 1;
    else nextIndex = index >= tabs.length - 1 ? 0 : index + 1;
    const next = tabs[nextIndex];
    if (!next) return;
    activateTab(next.id);
    tabRefs.current[next.id]?.focus();
  };

  return (
    <section data-workspace-pane className="workspace-shell workspace-shell--active" aria-label="工作台">
      <div className="workspace-tabbar">
        <div
          ref={stripRef}
          className="workspace-tabbar__scroll"
          role="tablist"
          aria-label="工作台标签页"
          onKeyDown={(event) => {
            const target = event.target;
            if (target instanceof HTMLButtonElement && target.getAttribute("role") === "tab") {
              const index = tabs.findIndex((tab) => tab.id === target.getAttribute("data-tab-id"));
              if (index >= 0) onTabKeyDown(event, index);
            }
          }}
        >
          {tabs.map((tab, index) => {
            const isActive = tab.id === activeTab?.id;
            const icon =
              tab.kind === "view"
                ? (() => {
                    const def = WORKSPACE_TABS.find((d) => d.id === tab.viewId);
                    return def ? createElement(def.icon, { size: 13 }) : null;
                  })()
                : createElement(FileTypeIcon as ComponentType<{ name: string; size?: number }>, {
                    name: (tab.path ?? "").split(/[\\/]/).pop() ?? tab.title,
                    size: 13,
                  });
            const count = tab.kind === "view" && tab.viewId ? badgeOf(tab.viewId as WorkspaceTabId) : 0;
            return (
              <div
                key={tab.id}
                data-pane-tab={tab.id}
                data-active={isActive}
                data-tab-kind={tab.kind}
                data-view-id={tab.viewId ?? ""}
                className="workspace-tab"
              >
                <button
                  ref={(node) => { tabRefs.current[tab.id] = node; }}
                  type="button"
                  role="tab"
                  id={tabIdOf(index)}
                  data-tab-id={tab.id}
                  aria-label={tab.kind === "file" ? `文件 ${tab.title}` : tab.title}
                  aria-selected={isActive}
                  aria-controls={panelId}
                  tabIndex={isActive ? 0 : -1}
                  title={tab.kind === "file" ? tab.path : tab.title}
                  className="workspace-tab__main"
                  onClick={() => activateTab(tab.id)}
                >
                  <span className="workspace-tab__icon">{icon}</span>
                  <span className="workspace-tab__title">{tab.title}</span>
                  {count > 0 && (
                    <span className="workspace-tab__badge" title={`${count} 个新项目`}>
                      {count > 99 ? "99+" : count}
                    </span>
                  )}
                </button>
                <button
                  type="button"
                  aria-label={`关闭 ${tab.title}`}
                  title={`关闭 ${tab.title}`}
                  className="workspace-tab__close"
                  onClick={() => closeTab(tab.id)}
                >
                  <svg width="9" height="9" viewBox="0 0 10 10" aria-hidden>
                    <path d="M1 1l8 8M9 1L1 9" stroke="currentColor" strokeWidth="1.4" />
                  </svg>
                </button>
              </div>
            );
          })}
        </div>

        <div className="workspace-tabbar__actions">
          <Dropdown
            trigger={["click"]}
            menu={{
              items: [
                ...enabledTabs.map((tab) => ({
                  key: tab.id,
                  label: tab.label,
                  icon: createElement(tab.icon, { size: 13 }),
                })),
                ...(disabledTabs.length > 0
                  ? [
                      { type: "divider" as const },
                      ...disabledTabs.map((tab) => ({
                        key: tab.id,
                        label: `${tab.label}（未启用）`,
                        icon: createElement(tab.icon, { size: 13 }),
                      })),
                    ]
                  : []),
              ] as MenuProps["items"],
              onClick: ({ key }) => {
                const def = WORKSPACE_TABS.find((tab) => tab.id === key);
                if (!def) return;
                if (!enabled[def.id]) setEnabled(def.id, true);
                openView(def.id, def.label);
              },
            }}
          >
            <button type="button" title="新建标签页" aria-label="新建标签页" className="workspace-icon-button">
              <Plus size={14} aria-hidden />
            </button>
          </Dropdown>
          <Popover content={manageContent} trigger="click" placement="bottomRight">
            <button type="button" title="管理工作台面板" aria-label="管理工作台面板" className="workspace-icon-button">
              <Settings size={14} aria-hidden />
            </button>
          </Popover>
          <button
            type="button"
            title="收起工作台"
            aria-label="收起工作台"
            className="workspace-icon-button"
            onClick={() => context.onClosePanel?.()}
          >
            <PanelRightClose size={14} aria-hidden />
          </button>
        </div>
      </div>

      <div id={panelId} role="tabpanel" aria-labelledby={tabIdOf(activeIndex)} className="workspace-shell__body">
        <div className="workspace-shell__content">{content}</div>
      </div>
    </section>
  );
}
