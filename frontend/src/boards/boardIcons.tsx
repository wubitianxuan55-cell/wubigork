import type { ReactNode, SVGProps } from 'react'

/**
 * Gaea 一级导航图标。
 *
 * 统一视觉规则：
 * - 24×24 画布、1.6px 圆角描边、无填充，在 18–20px 下仍能辨认；
 * - 以板块的真实工作对象构图，不直接复用通用工具/数据库/皇冠图标；
 * - 图标仅作辅助图形，可访问名称始终由导航按钮提供。
 */
export interface BoardIconProps extends Omit<SVGProps<SVGSVGElement>, 'children'> {
  size?: number | string
}

type IconFrameProps = BoardIconProps & { children: ReactNode }

function IconFrame({
  size = 20,
  strokeWidth = 1.6,
  children,
  ...props
}: IconFrameProps) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={strokeWidth}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
      data-gaea-board-icon=""
      {...props}
    >
      {children}
    </svg>
  )
}

/** 聊天：带上下文线的对话气泡，和“微信助手”的双气泡区分。 */
export function GaeaChatIcon(props: BoardIconProps) {
  return (
    <IconFrame {...props}>
      <path d="M5 6.75A2.75 2.75 0 0 1 7.75 4h8.5A2.75 2.75 0 0 1 19 6.75v5.5A2.75 2.75 0 0 1 16.25 15H11l-4 3v-3.25A2.75 2.75 0 0 1 5 12.25z" />
      <path d="M8.25 8h7.5M8.25 11h4.75" />
      <circle cx="16.5" cy="11" r=".8" fill="currentColor" stroke="none" />
    </IconFrame>
  )
}

/** 文学：摊开的书稿与一页灵光，强调创作而非“读书”入口。 */
export function GaeaLiteratureIcon(props: BoardIconProps) {
  return (
    <IconFrame {...props}>
      <path d="M4 5.5c2.9-.7 5.6 0 8 2.1v11.9c-2.4-2-5.1-2.8-8-2.1z" />
      <path d="M20 5.5c-2.9-.7-5.6 0-8 2.1v11.9c2.4-2 5.1-2.8 8-2.1z" />
      <path d="M12 7.6v11.9" />
      <path d="M16 9.1c.25 1.1 1 1.85 2.1 2.1-1.1.25-1.85 1-2.1 2.1-.25-1.1-1-1.85-2.1-2.1 1.1-.25 1.85-1 2.1-2.1z" />
    </IconFrame>
  )
}

/** 绘梦：带色斑与握孔的美术调色盘，避免与文档、铅笔混淆。 */
export function GaeaDreamIcon(props: BoardIconProps) {
  return (
    <IconFrame {...props}>
      <path d="M11.9 4.15c4.7 0 8.35 2.85 8.35 6.55 0 2.15-1.5 3.65-3.45 3.65h-1.7c-1.2 0-2 1.15-1.5 2.25.5 1.1-.3 2.45-1.75 2.4-4.65-.2-8-3.55-8-7.65 0-4.05 3.45-7.2 8.05-7.2z" />
      <circle cx="7.65" cy="8.15" r=".85" fill="currentColor" stroke="none" />
      <circle cx="11.2" cy="6.6" r=".85" fill="currentColor" stroke="none" />
      <circle cx="15.05" cy="8.2" r=".85" fill="currentColor" stroke="none" />
      <circle cx="7.1" cy="12.35" r=".85" fill="currentColor" stroke="none" />
      <circle cx="16.85" cy="15.25" r="1.15" />
    </IconFrame>
  )
}

/** 办公：显示器、支架与主机塔的经典台式机组合。 */
export function GaeaOfficeIcon(props: BoardIconProps) {
  return (
    <IconFrame {...props}>
      <rect x="3.5" y="4.25" width="13" height="9.25" rx="1.5" />
      <path d="M10 13.5v3.75M7.5 17.25h5" />
      <rect x="17.5" y="5.75" width="3.25" height="12.5" rx=".75" />
      <path d="M18.55 8.25h1.15M18.55 10.25h1.15M18.55 12.25h1.15" />
    </IconFrame>
  )
}

/** 造价数据库：三层成本数据叠，和办公文档形成清晰的结构差异。 */
export function GaeaCostIcon(props: BoardIconProps) {
  return (
    <IconFrame {...props}>
      <path d="M4.5 6.5c0-1.25 3.35-2.25 7.5-2.25s7.5 1 7.5 2.25S16.15 8.75 12 8.75 4.5 7.75 4.5 6.5z" />
      <path d="M4.5 6.5v3c0 1.25 3.35 2.25 7.5 2.25s7.5-1 7.5-2.25v-3" />
      <path d="M4.5 9.5v3.25c0 1.25 3.35 2.25 7.5 2.25s7.5-1 7.5-2.25V9.5" />
      <path d="M4.5 12.75v3.25c0 1.25 3.35 2.25 7.5 2.25s7.5-1 7.5-2.25v-3.25" />
      <path d="M4.5 16v2c0 1.25 3.35 2.25 7.5 2.25s7.5-1 7.5-2.25v-2" />
    </IconFrame>
  )
}

/** 编程：清晰的终端输入/输出符号。 */
export function GaeaCodeIcon(props: BoardIconProps) {
  return (
    <IconFrame {...props}>
      <path d="M8.25 8 4.5 12l3.75 4M15.75 8l3.75 4-3.75 4" />
      <path d="m13.5 6.5-3 11" />
    </IconFrame>
  )
}

/** 众神殿：带三角檐、柱廊与台基的殿堂。 */
export function GaeaPantheonIcon(props: BoardIconProps) {
  return (
    <IconFrame {...props}>
      <path d="M4 9.25 12 4l8 5.25z" />
      <path d="M5.5 9.25h13M6.5 17.5V10.5M10.25 17.5V10.5M13.75 17.5V10.5M17.5 17.5V10.5M4 17.5h16M3.5 20h17" />
    </IconFrame>
  )
}

/** 记忆中枢：双脑叶轮廓与内部分区线，直接表达“大脑”。 */
export function GaeaMemoryIcon(props: BoardIconProps) {
  return (
    <IconFrame {...props}>
      <path d="M12 5a3 3 0 1 0-5.997.125 4 4 0 0 0-2.526 5.77 4 4 0 0 0 .556 6.588A4 4 0 1 0 12 18Z" />
      <path d="M12 5a3 3 0 1 1 5.997.125 4 4 0 0 1 2.526 5.77 4 4 0 0 1-.556 6.588A4 4 0 1 1 12 18Z" />
      <path d="M15 13a4.5 4.5 0 0 1-3-4 4.5 4.5 0 0 1-3 4" />
      <path d="M17.6 6.5a3 3 0 0 0 .4-1.375M6 5.125a3 3 0 0 0 .4 1.375M3.475 10.9a4 4 0 0 1 .585-.4M19.94 10.5a4 4 0 0 1 .585.4M6 18a4 4 0 0 1-1.967-.516M19.967 17.484A4 4 0 0 1 18 18" />
    </IconFrame>
  )
}

/** 模型中心：带算力触点的模型芯片。 */
export function GaeaModelIcon(props: BoardIconProps) {
  return (
    <IconFrame {...props}>
      <rect x="6" y="6" width="12" height="12" rx="3" />
      <rect x="9" y="9" width="6" height="6" rx="1.75" />
      <path d="M12 3.25V6M12 18v2.75M3.25 12H6M18 12h2.75" />
    </IconFrame>
  )
}

/** 角色库：档案卡中的人物剪影。 */
export function GaeaCharacterIcon(props: BoardIconProps) {
  return (
    <IconFrame {...props}>
      <rect x="4" y="3.75" width="16" height="16.5" rx="3" />
      <circle cx="12" cy="9.25" r="2.35" />
      <path d="M8.1 17c.7-2.15 2.05-3.25 3.9-3.25s3.2 1.1 3.9 3.25" />
      <path d="M17.25 6.2h1.1" />
    </IconFrame>
  )
}

/** 进度计划：时间轴与错落的横道条。 */
export function GaeaScheduleIcon(props: BoardIconProps) {
  return (
    <IconFrame {...props}>
      <path d="M5 4.5v15" />
      <circle cx="5" cy="6" r=".75" fill="currentColor" stroke="none" />
      <circle cx="5" cy="12" r=".75" fill="currentColor" stroke="none" />
      <circle cx="5" cy="18" r=".75" fill="currentColor" stroke="none" />
      <path d="M8 6h8.75M8 9h4.75M8 12h10M8 15h5.5M8 18h7" />
    </IconFrame>
  )
}

/** 运行监测：实时信号与环形仪表。 */
export function GaeaRuntimeIcon(props: BoardIconProps) {
  return (
    <IconFrame {...props}>
      <circle cx="12" cy="12" r="8.25" />
      <path d="M5.5 12h2.9l1.45-3.75 2.85 7.5 1.55-3.75h4.25" />
      <circle cx="19.25" cy="7.25" r="1.1" fill="currentColor" stroke="none" />
    </IconFrame>
  )
}

/** 方案库：分层档案抽屉与检索镜，表达“归档成库 + 按需查找”。
 *  刻意不与办公共用“单页文档 + 横线”轮廓，主栏 20px 下仍先读到柜体。 */
export function GaeaProposalIcon(props: BoardIconProps) {
  return (
    <IconFrame {...props}>
      <rect x="3.75" y="4.25" width="12.25" height="15.5" rx="2.1" />
      <path d="M3.75 8.5h12.25M3.75 12.75h12.25" />
      <path d="M7.5 6.5h4.75M7.5 10.75h4.75M7.5 15h4.75" />
      <circle cx="16.7" cy="15.5" r="3.2" />
      <path d="m18.9 17.75 2.15 2.15" />
    </IconFrame>
  )
}
