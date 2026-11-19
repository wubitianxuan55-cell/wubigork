import React, { useEffect, useMemo, useState } from 'react'
import { Button, Input, Tooltip } from 'antd'
import {
  SearchOutlined, QuestionCircleOutlined, CloseOutlined,
} from '@ant-design/icons'
import './settings-page.css'
import { useT } from '../gaea/lib/i18n'
import { useAppStore } from '../stores/appStore'
import { SETTINGS_CATEGORIES, SETTINGS_ZONES, type Category } from '../components/settings/categories'

/** SettingsPage —「控制室」3 分区工作台（2026-11-20 UI 重设计）：细条头部 +
 *  左两级分类导航（全局/板块，分组定义单源在 components/settings/categories.tsx
 *  注册表）+ 中表单区 + 右帮助 inspector（可隐藏）。重设计四点：①中栏激活组
 *  头部（图标章+标题+描述+zone 徽标）给方位感；②搜索无匹配出建议分组胶囊
 *  （零死端）；③导航精修（zone tick/激活图标着色/过滤计数 n/total/去底部提示
 *  噪音）；④inspector 增「相关分组」快跳（注册表关键词交集，纯前端数据）。
 *  搜索跨分组过滤并自动切换；openSettings(category) 锚点直达（跨板块入口共用）。 */

/** 搜索无匹配时的建议分组（高频组；点击直达并清搜索） */
const SUGGEST_KEYS = ['general', 'model', 'office', 'chat'] as const

const SettingsPage: React.FC = () => {
  const t = useT()
  const [query, setQuery] = useState('')
  const [activeKey, setActiveKey] = useState('general')
  const [inspectorOpen, setInspectorOpen] = useState(false)
  const pendingCategory = useAppStore((s) => s.pendingSettingsCategory)
  const clearPendingCategory = useAppStore((s) => s.clearPendingSettingsCategory)

  // 锚点直达（批 1）：openSettings(category) 经 store pending 传信——挂载消费 +
  // 已挂载订阅共用。命中即切组并清搜索（过滤态下 effectiveKey 会回退首个可见组，
  // 与直达语义相悖）。
  useEffect(() => {
    if (!pendingCategory) return
    if (SETTINGS_CATEGORIES.some((c) => c.key === pendingCategory)) {
      setActiveKey(pendingCategory)
      setQuery('')
    }
    clearPendingCategory()
  }, [pendingCategory, clearPendingCategory])

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
  }

  return (
    <div className="settings-page">
      {/* ── 细条头部：控制室标识 + 全局搜索 + 帮助开关 ── */}
      <header className="settings-bar">
        <div className="settings-bar__context">
          <span className="settings-bar__glow" aria-hidden="true" />
          <strong className="settings-bar__title">{t('settings.controlRoom')}</strong>
          <span className="settings-bar__hint">{t('settings.groupsHint', { count: SETTINGS_CATEGORIES.length })}</span>
        </div>
        <div className="settings-bar__spacer" />
        <Input
          allowClear
          prefix={<SearchOutlined />}
          placeholder={t('settings.searchPlaceholder')}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
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

      {/* ── 3 分区工作台：两级分类导航 | 表单区 | 帮助 inspector ── */}
      <div className="settings-workbench">
        {/* 左 = 分类导航（竖向，两级分区） */}
        <aside className="v3-panel settings-nav" aria-label={t('settings.navAria')}>
          <div className="v3-panel-head">
            <span className="v3-panel-title">{t('settings.groups')}</span>
            <span className="v3-panel-spacer" />
            {/* 过滤态显示 n/total（命中数有语义），常态只显总数 */}
            <span className="settings-nav__count">
              {q ? `${visible.length}/${SETTINGS_CATEGORIES.length}` : visible.length}
            </span>
          </div>
          <nav className="settings-nav__list">
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
                    onClick={() => setActiveKey(it.key)}
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

        {/* 中 = 表单区（v3-zone）：激活组头部给方位感，其下分组卡片 */}
        <main className="v3-zone settings-content">
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
            <div key={effectiveKey} className="settings-panel-enter">
              <div className="settings-head">
                <span className="settings-head__icon" aria-hidden="true">{active.icon}</span>
                <div className="settings-head__text">
                  <h2 className="settings-head__title">{catLabel(active)}</h2>
                  <p className="settings-head__desc">{catDesc(active)}</p>
                </div>
                {activeZone && (
                  <span className="settings-head__zone">{t(activeZone.labelKey)}</span>
                )}
              </div>
              {active.panel}
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
