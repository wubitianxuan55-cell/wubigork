import React from 'react'
import { LoadingOutlined, ThunderboltOutlined } from '@ant-design/icons'
import { C } from '../../utils/theme'

/** 分区容器：标题 + 细分隔线 + 内容 */
export const SectionBlock: React.FC<{
  title?: string
  icon?: React.ReactNode
  children: React.ReactNode
  style?: React.CSSProperties
}> = ({ title, icon, children, style }) => (
  <div style={{ display: 'flex', flexDirection: 'column', gap: 10, minWidth: 0, ...style }}>
    {title && (
      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
        {icon && (
          <span style={{ color: 'var(--color-primary)', fontSize: 13, display: 'inline-flex' }}>{icon}</span>
        )}
        <span style={{
          fontSize: 11, fontWeight: 500, letterSpacing: '0.08em',
          color: 'var(--md-sys-color-on-surface-variant, var(--color-text-secondary))',
        }}>
          {title}
        </span>
      </div>
    )}
    {children}
  </div>
)

/** 分隔线 */
export const SectionDivider: React.FC<{ style?: React.CSSProperties }> = ({ style }) => (
  <div style={{ height: 1, background: 'var(--border-subtle)', margin: '2px 0', ...style }} />
)

/** 状态圆点（CSS，非 emoji） */
export const StatusDot: React.FC<{ tone: 'ok' | 'warn' | 'danger' | 'idle' }> = ({ tone }) => {
  const color = tone === 'ok' ? 'var(--color-success)'
    : tone === 'warn' ? 'var(--color-warning)'
    : tone === 'danger' ? '#f87171'
    : 'var(--color-text-secondary)'
  return (
    <span style={{
      width: 7, height: 7, borderRadius: '50%', background: color, display: 'inline-block', flexShrink: 0,
      boxShadow: tone === 'idle' ? 'none' : `0 0 6px ${color}`,
    }} />
  )
}

export interface PickerOption {
  label: string
  value: string | number
  icon?: React.ReactNode
}

/** 分组选择器：选中 = 主色填充 + 按压反馈 */
export const PickerGroup: React.FC<{
  options: PickerOption[]
  value: string | number
  onChange: (v: any) => void
  columns?: number
}> = ({ options, value, onChange, columns }) => (
  <div style={{
    display: 'grid',
    gridTemplateColumns: columns ? `repeat(${columns}, 1fr)` : 'repeat(auto-fill, minmax(76px, 1fr))',
    gap: 6,
  }}>
    {options.map((o) => {
      const selected = o.value === value
      return (
        <button
          key={String(o.value)}
          type="button"
          title={o.label}
          onClick={() => onChange(o.value)}
          className="img-picker-btn"
          style={{
            display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 5,
            padding: '7px 10px', borderRadius: 10, cursor: 'pointer',
            border: '1px solid',
            borderColor: selected ? 'var(--color-primary)' : 'var(--border-subtle)',
            background: selected ? 'rgba(var(--accent-rgb), 0.14)' : 'rgba(255,255,255,0.03)',
            color: selected ? 'var(--color-primary)' : C('color-text-secondary'),
            fontSize: 12, fontWeight: selected ? 600 : 400,
            whiteSpace: 'nowrap', userSelect: 'none', fontFamily: 'inherit',
          }}
        >
          {o.icon}
          <span style={{ overflow: 'hidden', textOverflow: 'ellipsis' }}>{o.label}</span>
        </button>
      )
    })}
  </div>
)

/** 大号胶囊主按钮（生成） */
export const ActionButton: React.FC<{
  loading?: boolean
  disabled?: boolean
  label: string
  hint: string
  onClick: () => void
}> = ({ loading, disabled, label, hint, onClick }) => (
  <div style={{ display: 'flex', flexDirection: 'column', gap: 5 }}>
    <button
      type="button"
      disabled={disabled || loading}
      onClick={onClick}
      className="img-picker-btn"
      style={{
        width: '100%', height: 46, borderRadius: 999, border: 'none', cursor: loading ? 'wait' : 'pointer',
        background: 'linear-gradient(135deg, var(--color-primary), rgba(var(--accent-rgb), 0.72))',
        color: 'var(--md-sys-color-on-primary, #fff)',
        fontSize: 14, fontWeight: 600, letterSpacing: '0.02em',
        display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8,
        boxShadow: 'var(--shadow-glow)', opacity: disabled ? 0.5 : 1,
        fontFamily: 'inherit',
      }}
    >
      {loading ? <LoadingOutlined /> : <ThunderboltOutlined />}
      {label}
    </button>
    <div style={{ textAlign: 'center' }}>
      <span style={{ color: C('color-text-secondary'), fontSize: 11 }}>{hint}</span>
    </div>
  </div>
)
