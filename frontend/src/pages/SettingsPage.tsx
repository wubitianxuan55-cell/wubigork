import React, { useEffect, useMemo, useState } from 'react'
import { Button, Input, Tooltip } from 'antd'
import {
  SearchOutlined, QuestionCircleOutlined, CloseOutlined,
} from '@ant-design/icons'
import './settings-page.css'
import { useT } from '../gaea/lib/i18n'
import { useAppStore } from '../stores/appStore'
import { SETTINGS_CATEGORIES, SETTINGS_ZONES, type Category } from '../components/settings/categories'

/** SettingsPage —「控制室」3 分区工作台：细条头部 + 左两级分类导航（全局/板块，
 *  分组定义单源在 components/settings/categories.tsx 注册表）+ 中表单区 + 右帮助
 *  inspector（可隐藏）。分类导航替代原磁贴网格，激活项 = 主色容器 + 左缘光条；
 *  搜索跨分组过滤并自动切换；openSettings(category) 锚点直达（跨板块入口共用）。 */
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
            <span className="settings-nav__count">{visible.length}</span>
          </div>
          <nav className="settings-nav__list">
            {zoneGroups.map((g) => (
              <React.Fragment key={g.key}>
                {showZoneTitles && (
                  <div className="settings-nav__zone" aria-hidden="true">{g.label}</div>
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
                      <span className="settings-nav-item__label">{it.labelKey ? t(it.labelKey) : it.label}</span>
                      <span className="settings-nav-item__desc">{it.descKey ? t(it.descKey) : it.desc}</span>
                    </span>
                    <span className="settings-nav-item__orb" aria-hidden="true" />
                  </button>
                ))}
              </React.Fragment>
            ))}
          </nav>
          <div className="settings-nav__foot">
            <SearchOutlined aria-hidden="true" />
            <span>{t('settings.searchHint')}</span>
          </div>
        </aside>

        {/* 中 = 表单区（v3-zone，分组卡片） */}
        <main className="v3-zone settings-content">
          {visible.length === 0 ? (
            <div className="v3-card settings-empty">
              {t('settings.noMatch')}
            </div>
          ) : (
            <div key={effectiveKey} className="settings-panel-enter">
              {active.panel}
            </div>
          )}
        </main>

        {/* 右 = 预留 inspector（帮助说明，可隐藏） */}
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
                  <span className="settings-help-card__label">{t('settings.currentCategory', { label: active.labelKey ? t(active.labelKey) : active.label })}</span>
                </div>
                <p className="settings-help-card__desc">{active.descKey ? t(active.descKey) : active.desc}</p>
                <div className="settings-help-card__keywords">
                  {active.keywords.slice(0, 8).map((k) => (
                    <span key={k} className="settings-keyword">{k}</span>
                  ))}
                </div>
              </section>
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
