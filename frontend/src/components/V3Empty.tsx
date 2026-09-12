// V3Empty.tsx — v3 玻璃轨道视觉语言的空态组件
// 替代散用的 antd Empty（灰插画与深色玻璃风割裂）。
// 视觉：64×64 内联 SVG「轨道环」——虚线外环抱斜向轨道椭圆 + primary 活核，
// 呼应用户 logo 的「轨道环抱活核」语言；主题色走 md-sys-color CSS 变量，明暗自动适配。
import React from 'react'

interface V3EmptyProps {
  description?: React.ReactNode
  /** 兼容旧 antd Empty 调用点：接受并忽略 */
  image?: unknown
  /** 兼容旧 antd Empty 调用点：接受并忽略 */
  imageStyle?: React.CSSProperties
  /** 紧凑档：48px 图形+缩小间距（卡片内/窄容器空态） */
  compact?: boolean
  style?: React.CSSProperties
  className?: string
  children?: React.ReactNode
}

const DEFAULT_STYLE: React.CSSProperties = {
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'center',
  justifyContent: 'center',
  textAlign: 'center',
}

export default function V3Empty({
  description,
  compact,
  style,
  className,
  children,
}: V3EmptyProps) {
  const box = compact ? 48 : 64
  return (
    <div className={className} style={{ ...DEFAULT_STYLE, ...style }}>
      <svg
        width={box}
        height={box}
        viewBox="0 0 64 64"
        fill="none"
        aria-hidden="true"
        style={{ display: 'block' }}
      >
        {/* 外描边圆环（虚线轨道） */}
        <circle
          cx={32}
          cy={32}
          r={24}
          fill="none"
          stroke="var(--md-sys-color-outline-variant)"
          strokeWidth={2}
          strokeDasharray="4 6"
        />
        {/* 斜向轨道椭圆（环抱活核） */}
        <ellipse
          cx={32}
          cy={32}
          rx={28}
          ry={11}
          fill="none"
          stroke="var(--md-sys-color-primary)"
          strokeOpacity={0.35}
          strokeWidth={1.5}
          transform="rotate(-24 32 32)"
        />
        {/* 活核 */}
        <circle cx={32} cy={32} r={5} fill="var(--md-sys-color-primary)" />
      </svg>
      {description ? (
        <div
          style={{
            marginTop: compact ? 10 : 16,
            fontSize: compact ? 12 : 13,
            lineHeight: 1.6,
            color: 'var(--md-sys-color-on-surface-variant)',
            maxWidth: 320,
          }}
        >
          {description}
        </div>
      ) : null}
      {children ? (
        <div
          style={{
            marginTop: compact ? 8 : 12,
            display: 'flex',
            justifyContent: 'center',
          }}
        >
          {children}
        </div>
      ) : null}
    </div>
  )
}
