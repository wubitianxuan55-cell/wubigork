import React, { useEffect, useId, useRef, useState } from 'react'

export interface SettingsPanelTab {
  key: string
  label: string
  description: string
  icon: React.ReactNode
  keywords?: string[]
  content: React.ReactNode
}

interface SettingsPanelTabsProps {
  ariaLabel: string
  tabs: SettingsPanelTab[]
  query?: string
  defaultKey?: string
  onActiveChange?: (key: string) => void
}

/**
 * 设置面板内的任务分栏。
 *
 * 多任务设置过去把多个旧标签页纵向堆在同一滚动面里，这里改为一层明确的 tablist。
 * 页签一旦访问过会保持挂载，用户切换页签时未保存的草稿不会被卸载；视觉与可访问树里
 * 仍只显示当前任务。窄窗横栏会自动把当前页签带回可视区。
 */
const SettingsPanelTabs: React.FC<SettingsPanelTabsProps> = ({
  ariaLabel,
  tabs,
  query = '',
  defaultKey,
  onActiveChange,
}) => {
  const id = useId()
  const initialKey = tabs.some((tab) => tab.key === defaultKey) ? defaultKey! : tabs[0]?.key || ''
  const [activeKey, setActiveKey] = useState(initialKey)
  const [visited, setVisited] = useState<Set<string>>(() => new Set(initialKey ? [initialKey] : []))
  const rootRef = useRef<HTMLDivElement>(null)
  const tabRefs = useRef<Record<string, HTMLButtonElement | null>>({})
  const tabsRef = useRef(tabs)
  const activeChangeRef = useRef(onActiveChange)
  tabsRef.current = tabs
  activeChangeRef.current = onActiveChange

  const active = tabs.find((tab) => tab.key === activeKey) || tabs[0]
  const q = query.trim().toLowerCase()
  const mountedKeys = new Set(visited)
  if (activeKey) mountedKeys.add(activeKey)

  // 全局搜索命中任务页签时直接落到对应域。关键词只负责域选择；
  // 段标题锚定仍由 SettingsPage 在目标面板挂载后完成。
  useEffect(() => {
    if (!q) return
    const hit = tabsRef.current.find((tab) => {
      if (tab.label.toLowerCase().includes(q)) return true
      return (tab.keywords || []).some((keyword) => {
        const k = keyword.toLowerCase()
        return k.includes(q) || q.includes(k)
      })
    })
    if (hit) setActiveKey(hit.key)
  }, [q])

  useEffect(() => {
    if (defaultKey && tabsRef.current.some((tab) => tab.key === defaultKey)) {
      setActiveKey(defaultKey)
    }
  }, [defaultKey])

  useEffect(() => {
    if (!activeKey) return
    setVisited((prev) => {
      if (prev.has(activeKey)) return prev
      const next = new Set(prev)
      next.add(activeKey)
      return next
    })

    const tab = tabRefs.current[activeKey]
    const list = tab?.parentElement
    if (tab && list) {
      const listRect = list.getBoundingClientRect()
      const tabRect = tab.getBoundingClientRect()
      if (list.scrollWidth > list.clientWidth + 1) {
        if (tabRect.left < listRect.left || tabRect.right > listRect.right) {
          list.scrollLeft += tabRect.left - listRect.left - (list.clientWidth - tabRect.width) / 2
        }
      }
    }

    const scroller = rootRef.current?.closest<HTMLElement>('.settings-content')
    if (scroller && typeof scroller.scrollTo === 'function') {
      scroller.scrollTo({ top: 0, behavior: 'auto' })
    }
    activeChangeRef.current?.(activeKey)
  }, [activeKey])

  const onTabKeyDown = (event: React.KeyboardEvent) => {
    const keys = ['ArrowLeft', 'ArrowRight', 'Home', 'End']
    if (!keys.includes(event.key)) return
    event.preventDefault()
    const index = tabs.findIndex((tab) => tab.key === activeKey)
    let nextIndex = index
    if (event.key === 'Home') nextIndex = 0
    else if (event.key === 'End') nextIndex = tabs.length - 1
    else if (event.key === 'ArrowLeft') nextIndex = index <= 0 ? tabs.length - 1 : index - 1
    else nextIndex = index >= tabs.length - 1 ? 0 : index + 1
    const next = tabs[nextIndex]
    if (!next) return
    setActiveKey(next.key)
    tabRefs.current[next.key]?.focus()
  }

  if (!active) return null

  return (
    <div ref={rootRef} className="settings-panel-tabs">
      <div className="settings-panel-tabs__bar">
        <div
          className="settings-panel-tabs__list"
          role="tablist"
          aria-label={ariaLabel}
          onKeyDown={(event) => {
            if (event.target instanceof HTMLButtonElement && event.target.getAttribute('role') === 'tab') {
              onTabKeyDown(event)
            }
          }}
        >
          {tabs.map((tab) => {
            const selected = tab.key === activeKey
            return (
              <button
                key={tab.key}
                ref={(node) => { tabRefs.current[tab.key] = node }}
                type="button"
                role="tab"
                id={`${id}-tab-${tab.key}`}
                aria-selected={selected}
                aria-controls={`${id}-panel-${tab.key}`}
                tabIndex={selected ? 0 : -1}
                className={`settings-panel-tabs__tab${selected ? ' is-active' : ''}`}
                title={tab.description}
                onClick={() => setActiveKey(tab.key)}
              >
                <span className="settings-panel-tabs__icon" aria-hidden="true">{tab.icon}</span>
                <span className="settings-panel-tabs__label">{tab.label}</span>
              </button>
            )
          })}
        </div>
        <p className="settings-panel-tabs__desc" aria-live="polite">{active.description}</p>
      </div>

      {tabs.map((tab) => (
        <div
          key={tab.key}
          role="tabpanel"
          id={`${id}-panel-${tab.key}`}
          aria-labelledby={`${id}-tab-${tab.key}`}
          hidden={tab.key !== activeKey}
          tabIndex={0}
          className="settings-panel-tabs__panel"
        >
          {mountedKeys.has(tab.key) ? tab.content : null}
        </div>
      ))}
    </div>
  )
}

export default SettingsPanelTabs
