import type { SVGProps } from 'react'

export interface WorkspaceSidebarToggleIconProps extends Omit<SVGProps<SVGSVGElement>, 'children'> {
  collapsed: boolean
  size?: number | string
}

/**
 * 左侧工作面板开合图标。
 * - 展开中：左向折线，提示向左收起；
 * - 已收起：右向折线，提示向右展开。
 */
export function WorkspaceSidebarToggleIcon({
  collapsed,
  size = 18,
  ...props
}: WorkspaceSidebarToggleIconProps) {
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
      data-workspace-sidebar-toggle={collapsed ? 'expand' : 'collapse'}
      {...props}
    >
      <rect x="2.75" y="4.25" width="18.5" height="15.5" rx="3.25" />
      <path d="M9 4.5v15" />
      <path
        d="M9 4.5H6a3 3 0 0 0-3 3v9a3 3 0 0 0 3 3h3"
        fill="currentColor"
        fillOpacity=".14"
        stroke="none"
      />
      {collapsed ? (
        <path d="M12.5 8.25 16.5 12.5 12.5 16.75" />
      ) : (
        <path d="M16.5 8.25 12.5 12.5l4 4.25" />
      )}
    </svg>
  )
}
