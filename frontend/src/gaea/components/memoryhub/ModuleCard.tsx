import type { ReactNode } from "react";

/** ModuleCard 记忆中枢首页的玻璃拟态模块卡片。
 *  --hub-color 由外层设置（对应各库霓虹色），hover 发光/上浮。 */
export function ModuleCard(p: {
  label: string;
  icon: ReactNode;
  hint: string;
  count?: number;
  color: string;
  index: number;
  onClick: () => void;
}) {
  const { label, icon, hint, count, color, index, onClick } = p;
  return (
    <button
      onClick={onClick}
      className="hub-card hub-enter w-full text-left px-3.5 py-3"
      style={{ "--hub-color": color, animationDelay: `${120 + index * 70}ms` } as React.CSSProperties}
    >
      <div className="flex items-center gap-2.5">
        <span className="hub-card-icon text-[17px] leading-none" style={{ color }}>
          {icon}
        </span>
        <span className="flex-1 min-w-0">
          <span className="block text-fg text-[13px] font-semibold truncate">{label}</span>
          <span className="block text-fg-faint text-[10.5px] truncate mt-0.5">{hint}</span>
        </span>
        {typeof count === "number" && (
          <span className="hub-badge shrink-0 px-1.5 py-0.5 rounded-md text-[10.5px] font-mono">{count}</span>
        )}
      </div>
    </button>
  );
}
