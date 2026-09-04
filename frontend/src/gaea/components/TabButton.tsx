import { memo } from "react";

export const TabButton = memo(function TabButton(p: {
  active: boolean;
  onClick: () => void;
  badge: number;
  children: string;
}) {
  return (
    <button
      className={`flex items-center justify-center gap-1.5 rounded-lg px-3 py-1.5 text-[12.5px] font-medium border-0 cursor-pointer transition-colors ${
        p.active
          ? "bg-accent/15 text-accent"
          : "text-fg-faint hover:bg-bg-soft/60 hover:text-fg-dim"
      }`}
      onClick={p.onClick}
      type="button"
    >
      {p.children}
      {p.badge > 0 && (
        <span className={`rounded-full px-1.5 py-px text-[10px] tabular-nums ${p.active ? "bg-accent/15 text-accent" : "bg-bg-soft text-fg-faint"}`}>
          {p.badge}
        </span>
      )}
    </button>
  );
});
