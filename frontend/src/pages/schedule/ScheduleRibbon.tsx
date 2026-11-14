/**
 * ScheduleRibbon.tsx — ZPert 式功能区骨架
 *
 * 结构直接对齐逆向复刻页：页签栏 + 当前页签的命令面板，每组命令底部有组名。
 * 组件只负责布局/键盘语义；动作仍由 SchedulePage 单一持有。
 */
import React from 'react'

export type ScheduleRibbonTabKey = 'file' | 'plan' | 'resource' | 'control' | 'collab' | 'view' | 'help'

export const RibbonGroup: React.FC<{ label: string; children: React.ReactNode }> = ({ label, children }) => (
  <section className="sched-ribbon-panel">
    <div className="sched-ribbon-panel-body">{children}</div>
    <div className="sched-ribbon-panel-cap">{label}</div>
  </section>
)

export const RibbonColumn: React.FC<{ children: React.ReactNode }> = ({ children }) => (
  <div className="sched-ribbon-column">{children}</div>
)

interface RibbonCommandProps {
  icon: React.ReactNode
  label: React.ReactNode
  onClick?: () => void
  active?: boolean
  disabled?: boolean
  big?: boolean
  title?: string
  testId?: string
}

export const RibbonCommand: React.FC<RibbonCommandProps> = ({
  icon,
  label,
  onClick,
  active = false,
  disabled = false,
  big = false,
  title,
  testId,
}) => (
  <button
    type="button"
    className={`sched-ribbon-command${big ? ' is-big' : ''}${active ? ' is-active' : ''}`}
    onClick={onClick}
    disabled={disabled}
    title={title}
    data-testid={testId}
  >
    <span className="sched-ribbon-command-icon" aria-hidden="true">{icon}</span>
    <span className="sched-ribbon-command-label">{label}</span>
  </button>
)
