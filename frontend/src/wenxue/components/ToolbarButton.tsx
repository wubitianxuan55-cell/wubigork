import { memo, type ReactNode } from "react";

/** 统一的工具栏按钮 — 5 处重复 className 合并为单个组件 */
export const ToolbarButton = memo(function ToolbarButton({
  onClick,
  disabled,
  title,
  /** 缺省回落 title：纯图标按钮必须带 aria-label（design-system MASTER 红线） */
  ariaLabel,
  children,
}: {
  onClick: () => void;
  disabled?: boolean;
  title?: string;
  ariaLabel?: string;
  children: ReactNode;
}) {
  return (
    <button
      className="toolbar-btn no-drag"
      onClick={onClick}
      disabled={disabled}
      title={title}
      aria-label={ariaLabel ?? title}
    >
      {children}
    </button>
  );
});
