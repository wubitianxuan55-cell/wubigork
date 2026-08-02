import type { ReactNode } from "react";

/** ComingSoon 通用占位：成本库（预留）/ 记忆图谱（下阶段）等未上线库。 */
export function ComingSoon(p: { icon: ReactNode; title: string; desc: string }) {
  const { icon, title, desc } = p;
  return (
    <div className="h-full flex flex-col items-center justify-center gap-3 px-6 text-center">
      <div className="w-14 h-14 rounded-2xl bg-bg-elev border border-border flex items-center justify-center text-fg-faint">
        {icon}
      </div>
      <div className="text-fg text-[15px] font-semibold">{title}</div>
      <div className="text-fg-faint text-[12.5px] max-w-xs leading-relaxed">{desc}</div>
    </div>
  );
}
