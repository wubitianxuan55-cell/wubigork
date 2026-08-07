import React, { useMemo, useState } from 'react'
import { Input } from 'antd'
import {
  AppstoreOutlined, MessageOutlined, ReadOutlined, PictureOutlined,
  SettingOutlined, ApiOutlined, InfoCircleOutlined, SearchOutlined,
} from '@ant-design/icons'
import { C } from '../utils/theme'
import './settings-page.css'
import AppearancePanel, { DarkModePanel, FontPanel, DensityPanel, MotionPanel, AccentPanel } from '../components/settings/AppearancePanel'
import ChatPanel from '../components/settings/ChatPanel'
import WorkspacePanel from '../components/settings/WorkspacePanel'
import ImageGenPanel from '../components/settings/ImageGenPanel'
import OfficePanel from '../components/settings/OfficePanel'
import ModelPanel from '../components/settings/ModelPanel'
import AboutPanel from '../components/settings/AboutPanel'

interface Category {
  key: string
  icon: React.ReactNode
  label: string
  desc: string
  keywords: string[]
  panel: React.ReactNode
}

/** 设置分组：按当前功能板块整理（通用/聊天/小说/绘梦/办公/模型/关于） */
const CATEGORIES: Category[] = [
  {
    key: 'general',
    icon: <AppstoreOutlined />,
    label: '通用',
    desc: '外观',
    keywords: ['通用', '外观', '主题', '暗色', '亮色', '深色', '浅色', '模式', '字体', '字号', '密度', '动效', '动画', '强调色', '颜色', '显示'],
    panel: (<><AppearancePanel /><DarkModePanel /><FontPanel /><DensityPanel /><MotionPanel /><AccentPanel /></>),
  },
  {
    key: 'chat',
    icon: <MessageOutlined />,
    label: '聊天',
    desc: '伴侣 · 语音',
    keywords: ['聊天', '伴侣', '称呼', '性别', '人格', '角色', '语音', '朗读', '音色', '对话', '识别', 'tts', 'asr'],
    panel: <ChatPanel />,
  },
  {
    key: 'novel',
    icon: <ReadOutlined />,
    label: '小说',
    desc: '目录 · 风格',
    keywords: ['小说', '目录', '存储', '路径', '书库', '风格', 'skill', '剧照', '写作', '工作区'],
    panel: <WorkspacePanel />,
  },
  {
    key: 'imagegen',
    icon: <PictureOutlined />,
    label: '绘梦',
    desc: '图像后端',
    keywords: ['绘梦', '图像', '图片', '生成', 'comfyui', '后端', '模型', '保存', 'xai', 'grok-imagine'],
    panel: <ImageGenPanel />,
  },
  {
    key: 'office',
    icon: <SettingOutlined />,
    label: '办公',
    desc: '引擎 · 方案',
    keywords: ['办公', '引擎', '方案', '模型', '权限', '沙箱', 'agent', '工具', '技能', '热加载', '招标', '撰写', '温度'],
    panel: <OfficePanel />,
  },
  {
    key: 'model',
    icon: <ApiOutlined />,
    label: '模型',
    desc: '全局模型',
    keywords: ['模型', '推理', '强度', '引擎', 'grok', 'deepseek', 'ollama', 'herdsman', 'xai', 'api', 'key'],
    panel: <ModelPanel />,
  },
  {
    key: 'about',
    icon: <InfoCircleOutlined />,
    label: '关于',
    desc: '版本 · 存储',
    keywords: ['关于', '版本', '更新', '日志', '系统', '信息', '路径', '配置', '存储', 'token', '凭证'],
    panel: <AboutPanel />,
  },
]

/** SettingsPage — 设置中心：左侧按功能板块导航，支持全局搜索过滤 */
const SettingsPage: React.FC = () => {
  const [query, setQuery] = useState('')
  const [activeKey, setActiveKey] = useState('general')

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
      <header className="settings-page__header">
        <div>
          <h1 className="settings-page__title">设置</h1>
          <p className="settings-page__subtitle">按功能板块整理：通用 / 聊天 / 小说 / 绘梦 / 办公 / 模型 / 关于</p>
        </div>
        <Input
          allowClear
          prefix={<SearchOutlined style={{ color: C('color-text-secondary') }} />}
          placeholder="搜索设置项，如：主题 / 模型 / 存储"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          className="settings-page__search"
          style={{
            borderRadius: 8,
            background: 'var(--bg-glass, var(--md-sys-color-surface-container))',
            border: '1px solid var(--border-subtle, var(--md-sys-color-outline-variant))',
          }}
        />
      </header>

      <div className="settings-page__layout">
        <nav className="settings-rail" aria-label="设置分类">
          {visible.map((it) => (
            <button
              key={it.key}
              type="button"
              className={`settings-rail__item${it.key === effectiveKey ? ' is-active' : ''}`}
              onClick={() => setActiveKey(it.key)}
            >
              {it.icon}
              <span>{it.label}</span>
              <span className="settings-rail__desc">{it.desc}</span>
            </button>
          ))}
        </nav>

        <main className="settings-content">
          {visible.length === 0 ? (
            <div className="md-glass" style={{ borderRadius: 'var(--md-sys-radius-lg)', padding: '48px 20px', textAlign: 'center', color: 'var(--md-sys-color-text-secondary)' }}>
              没有匹配的设置项，试试「主题」「模型」或「存储」
            </div>
          ) : (
            active.panel
          )}
        </main>
      </div>
    </div>
  )
}

export default SettingsPage
