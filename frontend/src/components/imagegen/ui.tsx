/* eslint-disable react-refresh/only-export-components -- estimateImageTime 工具函数导出供生成流程复用 */
import React from 'react'
import { RightOutlined } from '@ant-design/icons'
import { C } from '../../utils/theme'

/** 预估单次生成耗时（秒），用于底部生成栏反馈 */
export const estimateImageTime = (
  backend: string,
  model: string,
  count: number,
  mode: string,
  frames: number,
  fps: number,
): number => {
  if (mode === 't2v') return Math.round((frames / Math.max(fps, 1)) * 4)
  if (mode === 'img2img') return count * 12
  if (backend === 'xai') return count * 5
  if (model === 'z-image-turbo') return count * 20
  if (model.startsWith('krea2')) return count * 300
  return count * 60
}

/** 可折叠分区：左栏渐进式披露的基础容器 */
export const CollapsibleSection: React.FC<{
  title: string
  icon?: React.ReactNode
  defaultOpen?: boolean
  right?: React.ReactNode
  children: React.ReactNode
  style?: React.CSSProperties
}> = ({ title, icon, defaultOpen = true, right, children, style }) => {
  const [open, setOpen] = React.useState(defaultOpen)
  return (
    <div className="ig-collapse" style={style}>
      <button
        type="button"
        className="ig-collapse-header"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
      >
        <RightOutlined className="ig-collapse-chevron" style={{ transform: open ? 'rotate(90deg)' : 'rotate(0deg)' }} />
        {icon && (
          <span className="ig-collapse-icon">{icon}</span>
        )}
        <span className="ig-collapse-title">{title}</span>
        <span className="ig-collapse-spacer" />
        {right}
      </button>
      {open && (
        <div className="ig-collapse-body">{children}</div>
      )}
    </div>
  )
}

/** 状态圆点（CSS，非 emoji） */
export const StatusDot: React.FC<{ tone: 'ok' | 'warn' | 'danger' | 'idle' }> = ({ tone }) => {
  const color = tone === 'ok' ? 'var(--color-success)'
    : tone === 'warn' ? 'var(--color-warning)'
    : tone === 'danger' ? 'var(--color-destructive)'
    : 'var(--color-text-secondary)'
  return (
    <span style={{
      width: 7, height: 7, borderRadius: '50%', background: color, display: 'inline-block', flexShrink: 0,
      boxShadow: tone === 'idle' ? 'none' : `0 0 6px ${color}`,
    }} />
  )
}

export interface PickerOption<T = string | number> {
  label: string
  value: T
  icon?: React.ReactNode
}

/** 分组选择器：选中 = 主色填充 + 按压反馈。T 由 options/value 推导，onChange 收到精确类型。 */
export const PickerGroup = <T extends string | number,>(props: {
  options: PickerOption<T>[]
  value: T
  onChange: (v: T) => void
  columns?: number
}) => {
  const { options, value, onChange, columns } = props
  return (
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
}

