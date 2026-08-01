import type { ReactNode } from "react";
import { Tooltip as AntdTooltip } from "antd";

// Tooltip — 基于 antd Tooltip 的统一封装（保持 label prop 兼容既有调用点）。
export function Tooltip({
  label,
  children,
  className,
}: {
  label: string;
  children: ReactNode;
  className?: string;
}) {
  return (
    <AntdTooltip title={label} className={className}>
      <span className="inline-flex">{children}</span>
    </AntdTooltip>
  );
}
