import React, { useEffect, useMemo, useRef, useState } from 'react'
import { Button, Input, Tooltip } from 'antd'
import type { InputRef } from 'antd'
import {
  SearchOutlined, QuestionCircleOutlined, CloseOutlined, ControlOutlined,
} from '@ant-design/icons'
import './settings-page.css'
import { useT } from '../gaea/lib/i18n'
import { useAppStore } from '../stores/appStore'
import { SETTINGS_CATEGORIES, SETTINGS_ZONES, type Category } from '../components/settings/categories'

/** SettingsPage — 桌面设置控制台。
 *
 *  结构：紧凑命令栏 + 两级分类轨道 + 当前分区画布 + 可隐藏帮助 inspector。
 *  多任务分组在面板内部使用任务页签（通用/聊天/绘梦/办公/模型/安全/数据/文学/高级），
 *  避免把多个旧标签页纵向压进同一个滚动面。搜索既能直达分类，也能驱动内部任务页签，
 *  再锚定具体段。
 *  键盘：导航 ↑/↓/Home/End，`/` 聚焦搜索，Esc 清词；内部页签提供标准 tablist
 *  方向键/Home/End 语义。 */

/** 搜索无匹配时的建议分组（高频组；点击直达并清搜索） */
const SUGGEST_KEYS = ['general', 'model', 'office', 'chat'] as const

const SettingsPage: React.FC = () => {
  const t = useT()
  const [query, setQuery] = useState('')
  const [activeKey, setActiveKey] = useState('general')
  const [directTab, setDirectTab] = useState<string | undefined>()
  const [inspectorOpen, setInspectorOpen] = useState(false)
  const searchRef = useRef<InputRef>(null)
  const panelRootRef = useRef<HTMLDivElement>(null)
  const contentRef = useRef<HTMLElement>(null)
  const navRef = useRef<HTMLElement>(null)
  const tabMemoryRef = useRef<Record<string, string>>({})
  const pendingCategory = useAppStore((s) => s.pendingSettingsCategory)
  const pendingTab = useAppStore((s) => s.pendingSettingsTab)
  const clearPendingCategory = useAppStore((s) => s.clearPendingSettingsCategory)

  // 锚点直达（批 1）：openSettings(category) 经 store pending 传信——挂载消费 +
  // 已挂载订阅共用。命中即切组并清搜索（过滤态下 effectiveKey 会回退首个可见组，
  // 与直达语义相悖）。
  useEffect(() => {
    if (!pendingCategory) return
    if (SETTINGS_CATEGORIES.some((c) => c.key === pendingCategory)) {
      setActiveKey(pendingCategory)
      setDirectTab(pendingTab || undefined)
      setQuery('')
    } else {
      setDirectTab(undefined)
    }
    clearPendingCategory()
  }, [pendingCategory, pendingTab, clearPendingCategory])

  const q = query.trim().toLowerCase()
  const visible = useMemo(() => {
    if (!q) return SETTINGS_CATEGORIES
    return SETTINGS_CATEGORIES.filter((it) => {
      if (it.label.toLowerCase().includes(q)) return true
      if (it.desc.toLowerCase().includes(q)) return true
      return it.keywords.some((k) => k.toLowerCase().includes(q) || q.includes(k.toLowerCase()))
    })
  }, [q])

  // 两级导航：按 zone 分段（全局/板块）。单 zone 可见时不渲染分区标题——批 1
  // 全部现有组驻 global 区（board 区批 5 挂入），UI 保持零变化。
  const zoneGroups = useMemo(() => SETTINGS_ZONES
    .map((z) => ({ ...z, cats: visible.filter((c: Category) => c.zone === z.key) }))
    .filter((g) => g.cats.length > 0), [visible])
  const showZoneTitles = zoneGroups.length > 1

  // 当前激活分组被过滤掉时，自动切到第一个匹配分组
  const effectiveKey = visible.some((it) => it.key === activeKey) ? activeKey : (visible[0]?.key || 'general')
  const active = SETTINGS_CATEGORIES.find((it) => it.key === effectiveKey)!
  const catLabel = (c: Category) => (c.labelKey ? t(c.labelKey) : c.label)
  const catDesc = (c: Category) => (c.descKey ? t(c.descKey) : c.desc)

  // 分类切换是新的阅读上下文：回到内容顶部。搜索驱动的切组随后仍可把命中段
  // 滚到视口内；纯导航切换不再继承上一分类的滚动位置。
  useEffect(() => {
    const content = contentRef.current
    if (!content) return
    if (typeof content.scrollTo === 'function') {
      content.scrollTo({ top: 0, behavior: 'auto' })
    } else {
      content.scrollTop = 0
    }
  }, [effectiveKey])

  // 竖栏与窄窗横栏共用：让当前分类保持在分类轨道可视区内，不牵动页面滚动。
  useEffect(() => {
    const nav = navRef.current
    const item = nav?.querySelector<HTMLElement>('.settings-nav-item.is-active')
    if (!nav || !item) return
    const navRect = nav.getBoundingClientRect()
    const itemRect = item.getBoundingClientRect()
    if (nav.scrollHeight > nav.clientHeight + 1) {
      if (itemRect.top < navRect.top || itemRect.bottom > navRect.bottom) {
        nav.scrollTop += itemRect.top - navRect.top - (nav.clientHeight - itemRect.height) / 2
      }
    }
    if (nav.scrollWidth > nav.clientWidth + 1) {
      if (itemRect.left < navRect.left || itemRect.right > navRect.right) {
        nav.scrollLeft += itemRect.left - navRect.left - (nav.clientWidth - itemRect.width) / 2
      }
    }
  }, [effectiveKey, q, visible.length])

  const rememberTab = (category: string, tab: string) => {
    tabMemoryRef.current[category] = tab
  }

  // 内容头 zone 徽标：激活组所属分区（全局/板块）
  const activeZone = SETTINGS_ZONES.find((z) => z.key === active.zone)

  // 相关分组（inspector 快跳）：与激活组共享关键词的其他组（纯注册表数据计算）
  const related = useMemo(() => {
    const kw = new Set(active.keywords.map((k) => k.toLowerCase()))
    return SETTINGS_CATEGORIES
      .filter((c) => c.key !== active.key && c.keywords.some((k) => kw.has(k.toLowerCase())))
      .slice(0, 6)
  }, [active])

  // 搜索无匹配建议组：SUGGEST_KEYS 按注册表现序取实
  const suggestions = useMemo(() => SUGGEST_KEYS
    .map((key) => SETTINGS_CATEGORIES.find((c) => c.key === key))
    .filter((c): c is Category => !!c), [])

  // 直达 + 清搜索（建议胶囊/相关分组共用；与锚点直达同语义）
  const jumpTo = (key: string) => {
    setQuery('')
    setActiveKey(key)
    setDirectTab(undefined)
  }

  // 键盘可达：`/` 聚焦搜索（输入框/可编辑区不劫持，避免打字冲突）
  const onPageKeyDown = (e: React.KeyboardEvent) => {
    if (e.key !== '/') return
    const el = e.target as HTMLElement
    if (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.isContentEditable) return
    e.preventDefault()
    searchRef.current?.focus()
  }

  // 导航 ↑/↓/Home/End 移焦（roving focus：焦点移动、Enter 原生激活按钮）。
  // 过滤态下列表变化，每次按键现场查询可见项，不缓存句柄。
  const onNavKeyDown = (e: React.KeyboardEvent) => {
    if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp' && e.key !== 'Home' && e.key !== 'End') return
    const items = Array.from(e.currentTarget.querySelectorAll<HTMLButtonElement>('.settings-nav-item'))
    if (items.length === 0) return
    const idx = items.indexOf(document.activeElement as HTMLButtonElement)
    let next: HTMLButtonElement
    if (e.key === 'Home') next = items[0]
    else if (e.key === 'End') next = items[items.length - 1]
    else if (e.key === 'ArrowDown') next = idx < 0 || idx >= items.length - 1 ? items[0] : items[idx + 1]
    else next = idx <= 0 ? items[items.length - 1] : items[idx - 1]
    e.preventDefault()
    next.focus()
  }

  // 搜索命中段锚定：过滤态切组后扫描已挂面板的段标题（strong/h2~h4/mc-section-title，
  // 各面板头部异构故走 DOM 扫描零面板耦合），首个命中查询词的段卡 scrollIntoView +
  // 一闪 accent 环——搜索从「直达组」升到「直达项」；无命中静默回落组级语义。
  // 仅过滤态触发（纯切组不滚屏）。段标题多随 bridge 数据异步挂载（OfficeMemorySection
  // 等面板数据后置），单次扫描会扑空 → 短窗轮询（150ms×13≈2s）命中即停；
  // jsdom 无 scrollIntoView 实现，try/catch 兜底。
  useEffect(() => {
    if (!q) return
    let flashTimer = 0
    const scan = () => {
      const root = panelRootRef.current
      if (!root) return false
      const activePanel = root.querySelector<HTMLElement>('[role="tabpanel"]:not([hidden])') || root
      const heads = activePanel.querySelectorAll<HTMLElement>('strong, h2, h3, h4, .mc-section-title span, .mc-bind-title')
      const hit = Array.from(heads).find((h) => (h.textContent || '').trim().toLowerCase().includes(q))
      if (!hit) return false
      const card = (hit.closest('.v3-card, .mc-bind-card, .mc-section') as HTMLElement) || hit
      try {
        card.scrollIntoView({ block: 'start' })
      } catch { /* jsdom 无实现 */ }
      card.classList.add('settings-sec-flash')
      flashTimer = window.setTimeout(() => card.classList.remove('settings-sec-flash'), 1700)
      return true
    }
    let attempts = 0
    const pollTimer = window.setInterval(() => {
      attempts += 1
      if (scan() || attempts >= 13) window.clearInterval(pollTimer)
    }, 150)
    return () => {
      window.clearInterval(pollTimer)
      if (flashTimer) window.clearTimeout(flashTimer)
    }
  }, [q, effectiveKey])

  return (
    <div className="settings-page" onKeyDown={onPageKeyDown}>
      <a className="settings-skip" href="#settings-content">{t('settings.skipToContent')}</a>

      {/* ── 命令栏：控制台标识 + 全局搜索 + 帮助开关 ── */}
      <header className="settings-bar">
        <div className="settings-bar__context">
          <span className="settings-bar__mark" aria-hidden="true"><ControlOutlined /></span>
          <span className="settings-bar__identity">
            <strong className="settings-bar__title">{t('settings.controlRoom')}</strong>
            <span className="settings-bar__hint">{t('settings.groupsHint', { count: SETTINGS_CATEGORIES.length })}</span>
          </span>
        </div>
        <div className="settings-bar__spacer" />
        <Input
          allowClear
          prefix={<SearchOutlined />}
          placeholder={t('settings.searchPlaceholder')}
          value={query}
          onChange={(e) => {
            setQuery(e.target.value)
            if (e.target.value) setDirectTab(undefined)
          }}
          onKeyDown={(e) => {
            // Esc：有词清词（与 allowClear 同义），无词退焦回导航
            if (e.key !== 'Escape') return
            if (query) setQuery('')
            else (e.target as HTMLInputElement).blur()
          }}
          ref={searchRef}
          className="settings-page__search"
          aria-label={t('settings.searchAria')}
        />
        <Tooltip title={inspectorOpen ? t('settings.inspectorHide') : t('settings.inspectorShow')}>
          <Button
            type="text"
            size="small"
            aria-label={inspectorOpen ? t('settings.inspectorHide') : t('settings.inspectorShow')}
            className={`settings-bar__toggle${inspectorOpen ? ' is-active' : ''}`}
            icon={<QuestionCircleOutlined />}
            onClick={() => setInspectorOpen((v) => !v)}
          />
        </Tooltip>
      </header>

      {/* ── 工作台：分类轨道 | 当前分区 | 帮助 inspector ── */}
      <div className="settings-workbench">
        {/* 左 = 分类导航（竖向，两级分区） */}
        <aside className="v3-panel settings-nav" aria-label={t('settings.navAria')}>
          <div className="v3-panel-head">
            <span className="v3-panel-title">{t('settings.groups')}</span>
            <span className="v3-panel-spacer" />
            <span className="settings-nav__count">
              {q ? `${visible.length}/${SETTINGS_CATEGORIES.length}` : visible.length}
            </span>
          </div>
          <nav ref={navRef} className="settings-nav__list" onKeyDown={onNavKeyDown}>
            {zoneGroups.map((g) => (
              <React.Fragment key={g.key}>
                {showZoneTitles && (
                  <div className="settings-nav__zone" aria-hidden="true">{t(g.labelKey)}</div>
                )}
                {g.cats.map((it) => (
                  <button
                    key={it.key}
                    type="button"
                    className={`settings-nav-item${it.key === effectiveKey ? ' is-active' : ''}`}
                    onClick={() => {
                      setActiveKey(it.key)
                      setDirectTab(undefined)
                    }}
                    title={catDesc(it)}
                    aria-current={it.key === effectiveKey ? 'page' : undefined}
                  >
                    <span className="settings-nav-item__icon" aria-hidden="true">{it.icon}</span>
                    <span className="settings-nav-item__text">
                      <span className="settings-nav-item__label">{catLabel(it)}</span>
                      <span className="settings-nav-item__desc">{catDesc(it)}</span>
                    </span>
                  </button>
                ))}
              </React.Fragment>
            ))}
          </nav>
        </aside>

        {/* 中 = 当前分区画布 */}
        <main ref={contentRef} id="settings-content" className="v3-zone settings-content" tabIndex={-1}>
          {visible.length === 0 ? (
            <div className="v3-card settings-empty">
              <span className="settings-empty__icon" aria-hidden="true"><SearchOutlined /></span>
              <p className="settings-empty__title">{t('settings.noMatch')}</p>
              <div className="settings-empty__suggest">
                {suggestions.map((c) => (
                  <button
                    key={c.key}
                    type="button"
                    className="settings-empty__chip"
                    onClick={() => jumpTo(c.key)}
                  >
                    <span className="settings-empty__chip-icon" aria-hidden="true">{c.icon}</span>
                    {catLabel(c)}
                  </button>
                ))}
              </div>
            </div>
          ) : (
            <div ref={panelRootRef} key={effectiveKey} className="settings-panel-enter">
              <div className="settings-head">
                <div className="settings-head__intro">
                  <div className="settings-head__eyebrow">
                    <span>{t('settings.controlRoom')}</span>
                    <span aria-hidden="true">/</span>
                    <span>{catLabel(active)}</span>
                  </div>
                  <div className="settings-head__title-row">
                    <span className="settings-head__icon" aria-hidden="true">{active.icon}</span>
                    <h2 className="settings-head__title">{catLabel(active)}</h2>
                  </div>
                  <p className="settings-head__desc">{catDesc(active)}</p>
                </div>
                {activeZone && (
                  <span className="settings-head__zone">{t(activeZone.labelKey)}</span>
                )}
              </div>
              {typeof active.panel === 'function'
                ? active.panel(
                  q,
                  directTab ?? tabMemoryRef.current[effectiveKey],
                  (tab) => rememberTab(effectiveKey, tab),
                )
                : active.panel}
            </div>
          )}
        </main>

        {/* 右 = 帮助 inspector：当前组档案 + 相关分组快跳 + 操作提示，可隐藏 */}
        {inspectorOpen && (
          <aside className="v3-panel settings-inspector" aria-label={t('settings.inspectorAria')}>
            <div className="v3-panel-head">
              <span className="v3-panel-title">{t('settings.help')}</span>
              <span className="v3-panel-spacer" />
              <button
                type="button"
                className="settings-inspector__close"
                onClick={() => setInspectorOpen(false)}
                aria-label={t('settings.closeInspector')}
              >
                <CloseOutlined />
              </button>
            </div>
            <div className="settings-inspector__body">
              <section className="v3-card settings-help-card">
                <div className="settings-help-card__row">
                  <span className="settings-help-card__icon" aria-hidden="true">{active.icon}</span>
                  <span className="settings-help-card__label">{t('settings.currentCategory', { label: catLabel(active) })}</span>
                </div>
                <p className="settings-help-card__desc">{catDesc(active)}</p>
                <div className="settings-help-card__keywords">
                  {active.keywords.slice(0, 8).map((k) => (
                    <span key={k} className="settings-keyword">{k}</span>
                  ))}
                </div>
              </section>
              {related.length > 0 && (
                <section className="v3-card settings-help-card">
                  <h4 className="settings-help-card__title">{t('settings.related')}</h4>
                  <div className="settings-help-card__keywords">
                    {related.map((c) => (
                      <button
                        key={c.key}
                        type="button"
                        className="settings-related-item"
                        onClick={() => jumpTo(c.key)}
                      >
                        <span className="settings-related-item__icon" aria-hidden="true">{c.icon}</span>
                        {catLabel(c)}
                      </button>
                    ))}
                  </div>
                </section>
              )}
              <section className="v3-card settings-help-card">
                <h4 className="settings-help-card__title">{t('settings.tips')}</h4>
                <ul className="settings-help-card__list">
                  <li>{t('settings.tip1')}</li>
                  <li>{t('settings.tip2')}</li>
                  <li>{t('settings.tip3')}</li>
                </ul>
              </section>
            </div>
          </aside>
        )}
      </div>
    </div>
  )
}

export default SettingsPage
