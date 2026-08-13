import type { ReactNode } from 'react'
import { PushpinFilled, PushpinOutlined } from '@ant-design/icons'
import { engineColor } from './utils'

export type StatusTone = 'ok' | 'warn' | 'danger' | 'neutral' | 'accent'

/** 统一分区头：图标 + 标题 + 说明 + 右侧操作区 */
export function SectionHead({
  icon,
  title,
  desc,
  extra,
}: {
  icon?: ReactNode
  title: ReactNode
  desc?: ReactNode
  extra?: ReactNode
}) {
  return (
    <div className="mc-section-head">
      <div className="mc-section-head-main">
        <div className="mc-section-title">
          {icon && <span className="mc-section-icon">{icon}</span>}
          <span>{title}</span>
        </div>
        {desc && <div className="mc-section-desc">{desc}</div>}
      </div>
      {extra && <div className="mc-section-extra">{extra}</div>}
    </div>
  )
}

/** 统一状态/类型芯片（替代 antd Tag 标签汤） */
export function StatusChip({
  tone = 'neutral',
  dot = false,
  title,
  children,
}: {
  tone?: StatusTone
  dot?: boolean
  title?: string
  children: ReactNode
}) {
  return (
    <span className={`mc-chip is-${tone}`} title={title}>
      {dot && <i className="mc-chip-dot" />}
      {children}
    </span>
  )
}

/** 引擎色点 */
export function EngineMark({ id }: { id: string }) {
  return <i className="mc-engine-mark" style={{ background: engineColor({ id }) }} />
}

/** 统一底部状态：圆点 + 文案 */
export function StatusText({ tone = 'neutral', children }: { tone?: StatusTone; children: ReactNode }) {
  return (
    <span className={`mc-status is-${tone}`}>
      <i className={`mc-status-dot is-${tone}`} />
      {children}
    </span>
  )
}

/** KPI 统计块 */
export function KpiTile({
  icon,
  label,
  value,
  hint,
}: {
  icon?: ReactNode
  label: ReactNode
  value: ReactNode
  hint?: ReactNode
}) {
  return (
    <div className="mc-kpi">
      <div className="mc-kpi-label">
        {icon}
        {label}
      </div>
      <div className="mc-kpi-value">{value}</div>
      {hint && <div className="mc-kpi-hint">{hint}</div>}
    </div>
  )
}

/** 统一空状态 */
export function EmptyState({
  icon,
  title,
  hint,
  compact,
}: {
  icon?: ReactNode
  title: ReactNode
  hint?: ReactNode
  compact?: boolean
}) {
  return (
    <div className={`mc-empty${compact ? ' mc-empty--compact' : ''}`}>
      {icon && <div className="mc-empty-icon">{icon}</div>}
      <div className="mc-empty-title">{title}</div>
      {hint && <div className="mc-empty-hint">{hint}</div>}
    </div>
  )
}

export interface ModelCardProps {
  name: string
  engineId: string
  engineName?: string
  kindChip?: ReactNode
  chips?: ReactNode[]
  /** 用途建议等说明文字，显示在芯片行下方 */
  desc?: string
  active?: boolean
  dimmed?: boolean
  pinned?: boolean
  onTogglePin?: () => void
  status?: { tone: StatusTone; text: ReactNode }
  action?: ReactNode
}

/** 统一模型卡：名称 / 芯片行 / 底部状态 + 动作（动作始终底部对齐） */
export function ModelCard({
  name,
  engineId,
  engineName,
  kindChip,
  chips = [],
  desc,
  active,
  dimmed,
  pinned,
  onTogglePin,
  status,
  action,
}: ModelCardProps) {
  const cls = [
    'mc-model-card',
    active ? 'is-active' : '',
    dimmed ? 'is-dim' : '',
  ].filter(Boolean).join(' ')
  return (
    <div className={cls}>
      <div className="mc-model-head">
        <div className="mc-model-name" title={name}>{name}</div>
        {onTogglePin && (
          <button
            type="button"
            className={`mc-pin-btn${pinned ? ' is-pinned' : ''}`}
            aria-label={pinned ? '取消置顶' : '置顶'}
            onClick={(e) => {
              e.stopPropagation()
              onTogglePin()
            }}
          >
            {pinned ? <PushpinFilled /> : <PushpinOutlined />}
          </button>
        )}
      </div>
      <div className="mc-model-chips">
        {engineName && (
          <span className="mc-chip">
            <EngineMark id={engineId} />
            {engineName}
          </span>
        )}
        {kindChip}
        {chips}
      </div>
      {desc && <div className="mc-model-hint" title={desc}>{desc}</div>}
      <div className="mc-model-foot">
        {status && <StatusText tone={status.tone}>{status.text}</StatusText>}
        {action}
      </div>
    </div>
  )
}
