import { useState } from "react";
import { Moon, Sun } from "../icons";

// ThemeSwitcher — 暗/亮主题切换。
// 主题已统一：跟随主应用（老栈）darkMode，本组件仅作为切换入口。
// theme: "slate"（暗色）| "paper"（亮色）| 其他值视为暗色
export function ThemeSwitcher({
  theme,
  onSet,
}: {
  theme: string;
  onSet: () => void;
}) {
  const [open, setOpen] = useState(false);
  const isDark = theme === "slate" || theme === "earth" || theme === "noir";

  const handleToggle = () => {
    setOpen(false);
    onSet();
  };

  return (
    <div className="relative inline-flex no-drag">
      <button
        className="toolbar-btn no-drag"
        onClick={handleToggle}
        title={isDark ? "切换到亮色" : "切换到暗色"}
      >
        {isDark ? <Sun size={13} /> : <Moon size={13} />}
        <span>{isDark ? "亮色" : "暗色"}</span>
      </button>
      {open && (
        <div className="absolute top-full right-0 mt-1 z-50 min-w-[120px] py-1 bg-bg-elev-2 border border-border rounded-lg" style={{boxShadow: "var(--ds-shadow-dropdown)"}}>
          <button
            className="w-full text-left px-3 py-1.5 border-0 bg-transparent text-fg-dim text-[12px] cursor-pointer hover:bg-bg-soft hover:text-fg"
            onClick={handleToggle}
          >
            {isDark ? "切换到亮色" : "切换到暗色"}
          </button>
        </div>
      )}
    </div>
  );
}
