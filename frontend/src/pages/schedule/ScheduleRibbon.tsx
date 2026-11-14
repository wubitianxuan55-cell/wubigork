/**
 * ScheduleRibbon.tsx — ZPert 式功能区骨架
 *
 * 结构直接对齐逆向复刻页：页签栏 + 当前页签的命令面板，每组命令底部有组名。
 * 组件只负责布局/键盘语义；动作仍由 SchedulePage 单一持有。
 *
 * v3 交互语义（行为/无障碍线）：命令按钮是唯一的键盘入口，语义全在这里一次性定死，
 * 不让 100+ 调用点各写一份 aria：
 *  - 可切换命令（调用点显式传 `active`，未传=普通按钮不装开关）→ `aria-pressed`；
 *  - 禁用命令 → `disabled`（浏览器层面已不可聚焦/不可点）+ `aria-disabled`（AT 可读）
 *    + `tabIndex={-1}`（明确退出 Tab 序列，语义与行为同一口径）；`title` 照旧带禁用原因；
 *  - `Enter`/`Space` 显式激活：jsdom 与部分 AT 不派发按钮默认激活事件，
 *    显式调用 `onClick` 并 `preventDefault`（挡掉真浏览器的第二次默认点击，不双触发）；
 *  - 浮层触发器（Popover 子元素）可传 `ariaExpanded`，开关态对 AT 可读。
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
  /** 可切换命令的当前态；**未传=普通按钮**（不播 aria-pressed），传 false=开关关 */
  active?: boolean
  disabled?: boolean
  big?: boolean
  title?: string
  testId?: string
  /** 作为 Popover/Dropdown 触发器时的开合态（可选；缺省不渲染该属性） */
  ariaExpanded?: boolean
}

export const RibbonCommand: React.FC<RibbonCommandProps> = ({
  icon,
  label,
  onClick,
  active,
  disabled = false,
  big = false,
  title,
  testId,
  ariaExpanded,
}) => {
  /** 只有调用点显式给 active 的命令才是「开关」——未给的不冒称可切换 */
  const isToggle = active !== undefined
  const on = active === true
  return (
    <button
      type="button"
      className={`sched-ribbon-command${big ? ' is-big' : ''}${on ? ' is-active' : ''}`}
      onClick={onClick}
      disabled={disabled}
      title={title}
      data-testid={testId}
      aria-pressed={isToggle ? on : undefined}
      aria-disabled={disabled ? true : undefined}
      aria-expanded={ariaExpanded}
      tabIndex={disabled ? -1 : undefined}
      onKeyDown={(event) => {
        if (disabled || !onClick) return
        if (event.key !== 'Enter' && event.key !== ' ') return
        event.preventDefault()
        onClick()
      }}
    >
      <span className="sched-ribbon-command-icon" aria-hidden="true">{icon}</span>
      <span className="sched-ribbon-command-label">{label}</span>
    </button>
  )
}
