import React, { useMemo, useState } from 'react'
import { Empty, Input, Tabs, Typography } from 'antd'
import { SearchOutlined } from '@ant-design/icons'
import {
  AppstoreOutlined, FolderOpenOutlined, BookOutlined, FileTextOutlined, HeartOutlined,
  SoundOutlined, PictureOutlined, SettingOutlined, InfoCircleOutlined,
} from '@ant-design/icons'
import { C } from '../utils/theme'
import AppearancePanel, { DarkModePanel, FontPanel, DensityPanel, MotionPanel, AccentPanel } from '../components/settings/AppearancePanel'
import WorkspacePanel from '../components/settings/WorkspacePanel'
import ProposalPanel from '../components/settings/ProposalPanel'
import VoicePanel from '../components/settings/VoicePanel'
import WhisperPanel from '../components/settings/WhisperPanel'
import ImageGenPanel from '../components/settings/ImageGenPanel'
import OfficePanel from '../components/settings/OfficePanel'
import SystemPanel from '../components/settings/SystemPanel'

/** SettingsPage — 一站式设置中心（整合全部 gaea 设置参数，支持全局搜索） */
const SettingsPage: React.FC = () => {
  const [query, setQuery] = useState('')
  const [activeKey, setActiveKey] = useState('appearance')

  const tabItems = [
    {
      key: 'appearance',
      label: (<span><AppstoreOutlined style={{ marginRight: 6 }} />外观</span>),
      keywords: ['外观', '主题', '暗色', '亮色', '深色', '模式', '昼夜', '字体', '字号', '密度', '紧凑', '动效', '动画', '强调色', '颜色', '跟随系统'],
      children: (<><AppearancePanel /><DarkModePanel /><FontPanel /><DensityPanel /><MotionPanel /><AccentPanel /></>),
    },
    {
      key: 'workspace',
      label: (<span><BookOutlined style={{ marginRight: 6 }} />小说</span>),
      keywords: ['小说', '目录', '存储', '保存', '章节', '工作区', '书库'],
      children: (<WorkspacePanel />),
    },
    {
      key: 'voice',
      label: (<span><SoundOutlined style={{ marginRight: 6 }} />语音</span>),
      keywords: ['语音', '声音', '识别', '合成', 'stt', 'tts', '说话', '唤醒'],
      children: (<VoicePanel />),
    },
    {
      key: 'whisper',
      label: (<span><HeartOutlined style={{ marginRight: 6 }} />聊天</span>),
      keywords: ['聊天', '人格', '记忆', '对话', 'hermes', '角色'],
      children: (<WhisperPanel />),
    },
    {
      key: 'imagegen',
      label: (<span><PictureOutlined style={{ marginRight: 6 }} />绘梦</span>),
      keywords: ['绘梦', '图片', '生成', 'comfyui', '模型', '画面', '插图'],
      children: (<ImageGenPanel />),
    },
    {
      key: 'proposal',
      label: (<span><FileTextOutlined style={{ marginRight: 6 }} />方案</span>),
      keywords: ['方案', '投标', '编写', '文档'],
      children: (<ProposalPanel />),
    },
    {
      key: 'office',
      label: (<span><SettingOutlined style={{ marginRight: 6 }} />办公</span>),
      keywords: ['办公', '方案文档', '排版', '导出'],
      children: (<OfficePanel />),
    },
    {
      key: 'system',
      label: (<span><InfoCircleOutlined style={{ marginRight: 6 }} />系统</span>),
      keywords: ['系统', '更新', '日志', '数据', '存储', '关于'],
      children: (<SystemPanel />),
    },
  ]

  const q = query.trim().toLowerCase()

  // 全局搜索：匹配 tab 名 + 该 tab 的设置项关键词
  const filtered = useMemo(() => {
    if (!q) return tabItems
    return tabItems.filter((it) => {
      if (String(it.key).toLowerCase().includes(q)) return true
      const label = String(it.label).toLowerCase()
      if (label.includes(q)) return true
      return (it.keywords || []).some((k) => k.toLowerCase().includes(q) || q.includes(k.toLowerCase()))
    })
  }, [q, tabItems])

  // 当前激活 tab 被过滤掉时，自动切换到第一个匹配 tab
  const visibleKeys = filtered.map((it) => it.key)
  const effectiveKey = visibleKeys.includes(activeKey) ? activeKey : (visibleKeys[0] || activeKey)

  return (
    <div>
      {/* 标题区 */}
      <div style={{ marginBottom: 16 }}>
        <Typography.Title level={4} style={{ color: C('color-text'), margin: '0 0 4px', display: 'flex', alignItems: 'center', gap: 8 }}>
          <span style={{
            width: 6, height: 22, borderRadius: 3, display: 'inline-block',
            background: 'var(--gaea-glow)', boxShadow: '0 0 10px var(--gaea-glow)',
          }} />
          设置中心
        </Typography.Title>
        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12 }}>
          整合 gaea 全部设置参数 —— 外观 / 小说 / 语音 / 聊天 / 绘梦 / 方案 / 办公 / 系统
        </Typography.Text>
      </div>

      {/* 全局搜索 */}
      <div style={{ marginBottom: 14, maxWidth: 420 }}>
        <Input
          allowClear
          prefix={<SearchOutlined style={{ color: C('color-text-secondary') }} />}
          placeholder="搜索设置项，如：主题 / 模型 / 存储 / 更新…"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          style={{ borderRadius: 8, background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)' }}
        />
        {q && (
          <div style={{ marginTop: 6, fontSize: 12, color: C('color-text-secondary') }}>
            {filtered.length > 0
              ? <>匹配 {filtered.length} 个分组：{filtered.map((it) => it.key).join(' / ')}</>
              : <span style={{ color: C('color-error') }}>未找到匹配的设置分组</span>}
          </div>
        )}
      </div>

      {filtered.length === 0 ? (
        <Empty description="没有匹配的设置项" style={{ padding: 40 }} />
      ) : (
        <Tabs
          activeKey={effectiveKey}
          onChange={setActiveKey}
          items={filtered}
          tabPosition="top"
          size="middle"
          style={{ height: '100%' }}
        />
      )}
    </div>
  )
}

export default SettingsPage
