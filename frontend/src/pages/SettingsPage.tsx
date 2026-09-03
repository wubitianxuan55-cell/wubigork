import React, { useMemo, useState } from 'react'
import { Button, Input, Tooltip } from 'antd'
import {
  AppstoreOutlined, MessageOutlined, ReadOutlined, PictureOutlined,
  SettingOutlined, ApiOutlined, InfoCircleOutlined, SearchOutlined,
  SafetyCertificateOutlined, DatabaseOutlined, QuestionCircleOutlined, CloseOutlined,
} from '@ant-design/icons'
import './settings-page.css'
import { useT } from '../gaea/lib/i18n'
import type { DictKey } from '../gaea/locales/en'
import AppearancePanel, { DarkModePanel, FontPanel, DensityPanel, MotionPanel, AccentPanel, LanguagePanel } from '../components/settings/AppearancePanel'
import ChatPanel from '../components/settings/ChatPanel'
import WorkspacePanel from '../components/settings/WorkspacePanel'
import ImageGenPanel from '../components/settings/ImageGenPanel'
import OfficePanel from '../components/settings/OfficePanel'
import ModelPanel from '../components/settings/ModelPanel'
import SecurityPanel from '../components/settings/SecurityPanel'
import DataPanel from '../components/settings/DataPanel'
import AboutPanel from '../components/settings/AboutPanel'

interface Category {
  key: string
  icon: React.ReactNode
  /** zh 兜底文案（搜索关键词用，数据层不可 use hook） */
  label: string
  desc: string
  /** i18n 渲染 key（S2.2b 切片） */
  labelKey: DictKey
  descKey: DictKey
  keywords: string[]
  panel: React.ReactNode
}

/** 设置分组：按当前功能板块整理（通用/聊天/小说/绘梦/办公/模型/关于） */
const CATEGORIES: Category[] = [
  {
    key: 'general',
    icon: <AppstoreOutlined />,
    label: '通用',
    labelKey: 'settings.cat.general',
    desc: '外观',
    descKey: 'settings.cat.generalDesc',
    keywords: ['通用', '外观', '主题', '暗色', '亮色', '深色', '浅色', '模式', '字体', '字号', '密度', '动效', '动画', '强调色', '颜色', '显示', '语言', '界面语言', '英文', '繁体', 'language'],
    panel: (<><AppearancePanel /><DarkModePanel /><FontPanel /><DensityPanel /><MotionPanel /><AccentPanel /><LanguagePanel /></>),
  },
  {
    key: 'chat',
    icon: <MessageOutlined />,
    label: '聊天',
    labelKey: 'settings.cat.chat',
    desc: '伴侣 · 语音',
    descKey: 'settings.cat.chatDesc',
    keywords: ['聊天', '伴侣', '称呼', '性别', '人格', '角色', '语音', '朗读', '音色', '对话', '识别', 'tts', 'asr'],
    panel: <ChatPanel />,
  },
  {
    key: 'novel',
    icon: <ReadOutlined />,
    label: '小说',
    labelKey: 'settings.cat.novel',
    desc: '目录 · 风格',
    descKey: 'settings.cat.novelDesc',
    keywords: ['小说', '目录', '存储', '路径', '书库', '风格', 'skill', '剧照', '写作', '工作区'],
    panel: <WorkspacePanel />,
  },
  {
    key: 'imagegen',
    icon: <PictureOutlined />,
    label: '绘梦',
    labelKey: 'settings.cat.imagegen',
    desc: '图像后端',
    descKey: 'settings.cat.imagegenDesc',
    keywords: ['绘梦', '图像', '图片', '生成', 'comfyui', '后端', '模型', '保存', 'xai', 'grok-imagine'],
    panel: <ImageGenPanel />,
  },
  {
    key: 'office',
    icon: <SettingOutlined />,
    label: '办公',
    labelKey: 'settings.cat.office',
    desc: '引擎 · 方案',
    descKey: 'settings.cat.officeDesc',
    keywords: ['办公', '引擎', '方案', '模型', '权限', '沙箱', 'agent', '工具', '技能', '热加载', '招标', '撰写', '温度'],
    panel: <OfficePanel />,
  },
  {
    key: 'model',
    icon: <ApiOutlined />,
    label: '模型',
    labelKey: 'settings.cat.model',
    desc: '全局模型',
    descKey: 'settings.cat.modelDesc',
    keywords: ['模型', '推理', '强度', '引擎', 'grok', 'deepseek', 'ollama', 'herdsman', 'xai', 'api', 'key'],
    panel: <ModelPanel />,
  },
  {
    key: 'security',
    icon: <SafetyCertificateOutlined />,
    label: '安全',
    labelKey: 'settings.cat.security',
    desc: '隐私 · 调试',
    descKey: 'settings.cat.securityDesc',
    keywords: ['安全', '隐私', '敏感', '本地化', '本地', '局域网', '暴露', 'lan', '调试', 'webview', '远程', 'token', '报价', '成本'],
    panel: <SecurityPanel />,
  },
  {
    key: 'data',
    icon: <DatabaseOutlined />,
    label: '数据',
    labelKey: 'settings.cat.data',
    desc: '备份 · 恢复',
    descKey: 'settings.cat.dataDesc',
    keywords: ['数据', '备份', '恢复', '迁移', '导出', '导入', 'zip', '换机', '重装', '防丢', '存档', '快照'],
    panel: <DataPanel />,
  },
  {
    key: 'about',
    icon: <InfoCircleOutlined />,
    label: '关于',
    labelKey: 'settings.cat.about',
    desc: '版本 · 存储',
    descKey: 'settings.cat.aboutDesc',
    keywords: ['关于', '版本', '更新', '日志', '系统', '信息', '路径', '配置', '存储', 'token', '凭证'],
    panel: <AboutPanel />,
  },
]

/** SettingsPage —「控制室」3 分区工作台：细条头部 + 左分类导航 + 中表单区 + 右帮助 inspector（可隐藏）
 *  分类导航替代原磁贴网格，激活项 = 主色容器 + 左缘光条；搜索跨分组过滤并自动切换。 */
const SettingsPage: React.FC = () => {
  const t = useT()
  const [query, setQuery] = useState('')
  const [activeKey, setActiveKey] = useState('general')
  const [inspectorOpen, setInspectorOpen] = useState(false)

  const q = query.trim().toLowerCase()
  const visible = useMemo(() => {
    if (!q) return CATEGORIES
    return CATEGORIES.filter((it) => {
      if (it.label.toLowerCase().includes(q)) return true
      if (it.desc.toLowerCase().includes(q)) return true
      return it.keywords.some((k) => k.toLowerCase().includes(q) || q.includes(k.toLowerCase()))
    })
  }, [q])

  // 当前激活分组被过滤掉时，自动切到第一个匹配分组
  const effectiveKey = visible.some((it) => it.key === activeKey) ? activeKey : (visible[0]?.key || 'general')
  const active = CATEGORIES.find((it) => it.key === effectiveKey)!

  return (
    <div className="settings-page">
      {/* ── 细条头部：控制室标识 + 全局搜索 + 帮助开关 ── */}
      <header className="settings-bar">
        <div className="settings-bar__context">
          <span className="settings-bar__glow" aria-hidden="true" />
          <strong className="settings-bar__title">{t('settings.controlRoom')}</strong>
          <span className="settings-bar__hint">{t('settings.groupsHint', { count: CATEGORIES.length })}</span>
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

      {/* ── 3 分区工作台：分类导航 | 表单区 | 帮助 inspector ── */}
      <div className="settings-workbench">
        {/* 左 = 分类导航（竖向，替代磁贴） */}
        <aside className="v3-panel settings-nav" aria-label={t('settings.navAria')}>
          <div className="v3-panel-head">
            <span className="v3-panel-title">{t('settings.groups')}</span>
            <span className="v3-panel-spacer" />
            <span className="settings-nav__count">{visible.length}</span>
          </div>
          <nav className="settings-nav__list">
            {visible.map((it) => (
              <button
                key={it.key}
                type="button"
                className={`settings-nav-item${it.key === effectiveKey ? ' is-active' : ''}`}
                onClick={() => setActiveKey(it.key)}
                aria-current={it.key === effectiveKey ? 'page' : undefined}
              >
                <span className="settings-nav-item__icon" aria-hidden="true">{it.icon}</span>
                <span className="settings-nav-item__text">
                  <span className="settings-nav-item__label">{t(it.labelKey)}</span>
                  <span className="settings-nav-item__desc">{t(it.descKey)}</span>
                </span>
                <span className="settings-nav-item__orb" aria-hidden="true" />
              </button>
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
                  <span className="settings-help-card__label">{t('settings.currentCategory', { label: t(active.labelKey) })}</span>
                </div>
                <p className="settings-help-card__desc">{t(active.descKey)}</p>
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
