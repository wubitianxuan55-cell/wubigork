import { WORKSPACE_GROUPS, groupOfTab, type WorkspaceGroupId, type WorkspaceTabId } from "../lib/workspaceTabs";

// 右侧面板 Tab 按钮条（v3.0.8 收敛为两级）：
//   第一级 = 4 个主 Tab（文件 / 成果 / 运行 / 分析，按域分组）；
//   第二级 = 当前组内的子面板小 Tab（如「文件」组下：文件 / 资料）。
// 由 lib/workspaceTabs.ts 清单驱动渲染；激活态与 App.tsx 的 rightTab 保持一致。
// 激活态样式与 App.tsx 原手写按钮一致（text-accent + border-accent，
// redesign.css 用 [class*="text-accent"] 收复 Tailwind 类冲突）。
// C6（蒸馏 dsh-better-sidebar badge）：主 Tab 支持计数角标（99+ 封顶），
// 由 App 传入；角标走语义色令牌，不硬编码色值。

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

export function WorkspaceTabs({
  active,
  onChange,
  badges,
}: {
  active: WorkspaceTabId;
  onChange: (tab: WorkspaceTabId) => void;
  /** 主 Tab（组）计数角标：>0 才渲染，99+ 封顶（C6）。 */
  badges?: Partial<Record<WorkspaceGroupId, number>>;
}) {
  const group = groupOfTab(active);
  return (
    <div className="workspace-tabs shrink" role="tablist" aria-label="右侧面板">
      {/* 第一级：主 Tab（分组） */}
      <div className="flex items-center border-b border-border-soft overflow-hidden">
        {WORKSPACE_GROUPS.map((g) => {
          const Icon = g.icon;
          const isActive = group.id === g.id;
          const badge = badges?.[g.id];
          return (
            <button
              key={g.id}
              role="tab"
              aria-selected={isActive}
              data-grouptab={g.id}
              className={`flex items-center gap-1 px-3 py-2 text-xs bg-transparent border-0 border-b-2 cursor-pointer transition-[color,border-color] duration-[var(--dur-base)] hover:text-fg text-fg-dim border-transparent ${isActive ? "text-accent border-accent" : ""}`}
              onClick={() => onChange(g.tabs[0].id)}
              title={g.label}
            >
              <Icon size={13} />
              <span>{g.label}</span>
              {typeof badge === "number" && badge > 0 && !isActive && <BadgePill count={badge} />}
            </button>
          );
        })}
      </div>
      {/* 第二级：当前组内的子面板小 Tab（仅当组内多于 1 个面板时显示） */}
      {group.tabs.length > 1 && (
        <div className="flex items-center border-b border-border-soft overflow-hidden">
          {group.tabs.map((tab) => {
            const Icon = tab.icon;
            const isActive = active === tab.id;
            return (
              <button
                key={tab.id}
                role="tab"
                aria-selected={isActive}
                data-subtab={tab.id}
                className={`flex items-center gap-1 px-2.5 py-1.5 text-[11px] bg-transparent border-0 border-b cursor-pointer transition-[color,border-color] duration-[var(--dur-base)] hover:text-fg text-fg-faint border-transparent ${isActive ? "text-accent border-accent" : ""}`}
                onClick={() => onChange(tab.id)}
                title={tab.label}
              >
                <Icon size={11} />
                <span>{tab.label}</span>
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}
