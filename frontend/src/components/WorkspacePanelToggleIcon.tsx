import type { SVGProps } from 'react'

export interface WorkspacePanelToggleIconProps extends Omit<SVGProps<SVGSVGElement>, 'children'> {
  open: boolean
  size?: number | string
}

/**
 * 右侧工作面板开合图标。
 *
 * 右侧面板始终保持可辨认的轮廓；箭头只表达本次点击的方向：
 * - 已展开 → 向右收起；
 * - 已收起 → 向左展开。
 */
export function WorkspacePanelToggleIcon({
  open,
  size = 17,
  ...props
}: WorkspacePanelToggleIconProps) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.6}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
      data-workspace-panel-toggle={open ? 'close' : 'open'}
      {...props}
    >
      <rect x="2.75" y="4.25" width="18.5" height="15.5" rx="3.25" />
      <path d="M15 4.5v15" />
      <path
        d="M15 4.5h3a3 3 0 0 1 3 3v9a3 3 0 0 1-3 3h-3"
        fill="currentColor"
        fillOpacity=".14"
        stroke="none"
      />
      {open ? (
        <path d="M7.25 8.25 11.25 12.5 7.25 16.75" />
      ) : (
        <path d="M11.25 8.25 7.25 12.5l4 4.25" />
      )}
    </svg>
  )
}
