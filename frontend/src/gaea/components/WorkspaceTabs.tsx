import { useEffect, useState } from "react";
import { Settings } from "../icons";
import { WORKSPACE_TABS, type WorkspaceTabId } from "../lib/workspaceTabs";

// 右侧面板 Tab 按钮条（v4.27 扁平化）：
//   一级平铺 = 文件 / 产物 / 变更 / 任务 / 分工（无二级标签）。
// 由 lib/workspaceTabs.ts 清单驱动渲染；激活态与 App.tsx 的 rightTab 保持一致。
// 激活态样式与 App.tsx 原手写按钮一致（text-accent + border-accent，
// redesign.css 用 [class*="text-accent"] 收复 Tailwind 类冲突）。
// C6（蒸馏 dsh-better-sidebar badge）：Tab 支持计数角标（99+ 封顶），由 App
// 传入（v4.27 扁平化后按面板 id 下发，如任务/分工共享运行计数）；角标走
// 语义色令牌，不硬编码色值。
// v4.23（蒸馏 dsh-better-sidebar「声明式设置 + 侧边卡片」）：
//   条尾齿轮 → 弹层卡片列表，每 tab 一张卡（名称 + 启用开关），停用即从
//   tab 条隐藏；至少保留一个启用面板（最后一个开关锁定）。文案跟随本文件
//   既有现状用硬编码中文（未接 i18n）。

/** 角标渲染上限（对齐插件 Sidebar badge 的 99+ 封顶）。 */
const BADGE_CAP = 99;

function BadgePill({ count }: { count: number }) {
  const text = count > BADGE_CAP ? `${BADGE_CAP}+` : String(count);
  return (
    <span
      aria-label={`${count} 项进行中`}
      className="ml-1 inline-flex items-center justify-center min-w-4 h-4 px-1 rounded-full text-[9px] leading-none font-medium"
      style={{ background: "var(--gaea-glow)", color: "var(--color-on-primary)" }}
    >
      {text}
    </span>
  );
}

// 开关滑块（蒸馏 better-sidebar 品牌开关）：轨道 + 圆钮，令牌化配色。
function ToggleSwitch({ on, locked }: { on: boolean; locked: boolean }) {
  return (
    <span
      aria-hidden
      className={`inline-flex h-4 w-7 shrink-0 items-center rounded-full p-0.5 transition-colors duration-[var(--dur-fast)] ${on ? "justify-end" : "justify-start"}`}
      style={{
        background: on ? "var(--md-sys-color-primary)" : "var(--md-sys-color-outline-variant, var(--border))",
        opacity: locked ? 0.5 : undefined,
      }}
    >
      <span
        className="h-3 w-3 rounded-full"
        style={{ background: on ? "var(--color-on-primary)" : "var(--fg)" }}
      />
    </span>
  );
}

/** 声明式设置弹层：每 tab 一张卡（名称 + 启用开关，学 better-sidebar 侧边卡片）。 */
function TabSettingsPopup({
  enabledTabs,
  onToggle,
  onClose,
}: {
  enabledTabs: ReadonlySet<WorkspaceTabId>;
  onToggle?: (id: WorkspaceTabId, next: boolean) => void;
  onClose: () => void;
}) {
  // Esc 关闭（App 全局 Esc 收敛预览/抽屉，两者叠加无害）
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [onClose]);
  const enabledCount = enabledTabs.size;
  return (
    <>
      {/* 透明遮罩：点击弹层外部即关闭 */}
      <div className="fixed inset-0 z-10 cursor-default" onClick={onClose} aria-hidden />
      <div
        role="dialog"
        aria-label="面板显示设置"
        data-testid="workspace-tabs-settings"
        className="absolute right-1 top-[calc(100%-2px)] z-20 w-56 rounded-lg border border-border-soft bg-bg-elev p-1.5"
        style={{ boxShadow: "var(--ds-shadow-dropdown)" }}
      >
        <div className="px-1.5 pb-1 pt-0.5 text-[10px] text-fg-faint">面板显示设置</div>
        <div className="flex flex-col">
          {WORKSPACE_TABS.map((tab) => {
            const Icon = tab.icon;
            const on = enabledTabs.has(tab.id);
            // 至少保留一个启用面板：最后剩下的开关锁定（better-sidebar 同款约束）
            const locked = on && enabledCount <= 1;
            return (
              <div
                key={tab.id}
                data-settings-card={tab.id}
                className="flex items-center gap-2 rounded-md px-1.5 py-1 hover:bg-(color:--md-sys-color-surface-container-high)"
              >
                <Icon size={12} aria-hidden style={{ color: "var(--gaea-glow)" }} />
                <span className="text-[11px] text-fg">{tab.label}</span>
                <button
                  type="button"
                  role="switch"
                  aria-checked={on}
                  aria-label={`${tab.label}面板${on ? "（启用中）" : "（已停用）"}`}
                  data-tabswitch={tab.id}
                  disabled={locked}
                  className="ml-auto inline-flex items-center border-0 bg-transparent p-0 cursor-pointer disabled:cursor-default"
                  onClick={() => onToggle?.(tab.id, !on)}
                >
                  <ToggleSwitch on={on} locked={locked} />
                </button>
              </div>
            );
          })}
        </div>
        <div className="mt-0.5 border-t border-border-soft px-1.5 pt-1 text-[10px] text-fg-faint">
          停用的面板从标签条隐藏，至少保留一个
        </div>
      </div>
    </>
  );
}

export function WorkspaceTabs({
  active,
  onChange,
  badges,
  enabledTabs,
  onToggleTab,
}: {
  active: WorkspaceTabId;
  onChange: (tab: WorkspaceTabId) => void;
  /** Tab 计数角标：>0 才渲染，99+ 封顶（C6；v4.27 起按面板 id 下发）。 */
  badges?: Partial<Record<WorkspaceTabId, number>>;
  /** 声明式设置启用的面板集合；缺省 = 全部启用（旧调用方兼容）。 */
  enabledTabs?: ReadonlySet<WorkspaceTabId>;
  /** 设置卡开关回调（受控：开关状态由 App 经 enabledTabs 下发）。 */
  onToggleTab?: (id: WorkspaceTabId, next: boolean) => void;
}) {
  const [settingsOpen, setSettingsOpen] = useState(false);
  // 按启用集过滤：停用的面板从 Tab 条隐藏；enabledTabs 缺省时不过滤。
  const visibleTabs = enabledTabs
    ? WORKSPACE_TABS.filter((t) => enabledTabs.has(t.id))
    : WORKSPACE_TABS;
  return (
    <div className="workspace-tabs relative shrink" role="tablist" aria-label="右侧面板">
      {/* 一级 Tab 条 + 条尾声明式设置齿轮 */}
      <div className="flex items-center border-b border-border-soft overflow-hidden">
        {visibleTabs.map((tab) => {
          const Icon = tab.icon;
          const isActive = active === tab.id;
          const badge = badges?.[tab.id];
          return (
            <button
              key={tab.id}
              role="tab"
              aria-selected={isActive}
              data-paneltab={tab.id}
              className={`flex items-center gap-1 px-3 py-2 text-xs bg-transparent border-0 border-b-2 cursor-pointer transition-[color,border-color] duration-[var(--dur-base)] hover:text-fg text-fg-dim border-transparent ${isActive ? "text-accent border-accent" : ""}`}
              onClick={() => onChange(tab.id)}
              title={tab.label}
            >
              <Icon size={13} />
              <span>{tab.label}</span>
              {typeof badge === "number" && badge > 0 && !isActive && <BadgePill count={badge} />}
            </button>
          );
        })}
        <button
          type="button"
          data-testid="workspace-tabs-gear"
          aria-label="面板显示设置"
          aria-expanded={settingsOpen}
          title="面板显示设置"
          className={`ml-auto flex items-center justify-center w-6 h-6 mr-1 rounded-md border-0 bg-transparent cursor-pointer transition-colors duration-[var(--dur-fast)] hover:bg-(color:--md-sys-color-surface-container-high) ${settingsOpen ? "text-accent" : "text-fg-faint hover:text-fg"}`}
          onClick={() => setSettingsOpen((o) => !o)}
        >
          <Settings size={12} />
        </button>
      </div>
      {settingsOpen && (
        <TabSettingsPopup
          enabledTabs={enabledTabs ?? new Set(WORKSPACE_TABS.map((t) => t.id))}
          onToggle={onToggleTab}
          onClose={() => setSettingsOpen(false)}
        />
      )}
    </div>
  );
}
