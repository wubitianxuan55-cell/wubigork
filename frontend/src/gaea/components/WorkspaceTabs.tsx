import { WORKSPACE_TABS, type WorkspaceTabId } from "../lib/workspaceTabs";

// 右侧面板 Tab 按钮条：由 lib/workspaceTabs.ts 清单驱动渲染。
// 激活态样式与 App.tsx 原手写按钮一致（text-accent + border-accent，
// redesign.css 用 [class*="text-accent"] 收复 Tailwind 类冲突）。

export function WorkspaceTabs({
  active,
  onChange,
}: {
  active: WorkspaceTabId;
  onChange: (tab: WorkspaceTabId) => void;
}) {
  return (
    <div className="workspace-tabs flex items-center border-b border-border-soft overflow-hidden shrink" role="tablist" aria-label="右侧面板">
      {WORKSPACE_TABS.map((tab) => {
        const Icon = tab.icon;
        const isActive = active === tab.id;
        return (
          <button
            key={tab.id}
            role="tab"
            aria-selected={isActive}
            className={`flex items-center gap-1 px-3 py-2 text-xs bg-transparent border-0 border-b-2 cursor-pointer transition-[color,border-color] duration-[var(--dur-base)] hover:text-fg text-fg-dim border-transparent ${isActive ? "text-accent border-accent" : ""}`}
            onClick={() => onChange(tab.id)}
            title={tab.label}
          >
            <Icon size={13} />
            <span>{tab.label}</span>
          </button>
        );
      })}
    </div>
  );
}
