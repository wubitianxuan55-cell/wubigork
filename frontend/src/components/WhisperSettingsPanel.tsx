// WhisperSettingsPanel.tsx — 轻语专属设置面板
// 对齐 ackem SettingsPage 伴侣设置区域 (settings-companion + settings-safety)

import React, { useState, useEffect } from 'react'
import { Button, Switch, Select, Modal, message, Card, Tag, Typography, Input, Radio, Checkbox } from 'antd'
import { ClearOutlined, SwapOutlined, ApiOutlined } from '@ant-design/icons'
import * as App from '../../wailsjs/go/app/App'

const { Text } = Typography

interface Personality {
  id: string; label: string; gender: string; tags?: string[]
  dims: { T: number; I: number; S: number; O: number; R: number }
  voiceGuide?: string; requiresAdult18?: boolean
}

interface Props {
  activePersonality: string
  personalities: Personality[]
  adultMode: boolean
  engineID: string
  onPersonalityChange: (id: string) => void
  onAdultModeChange: (v: boolean) => void
  onClearSession: () => void
}

// localStorage 持久化的额外设置
interface CompanionSettings {
  companionName: string
  companionGender: 'male' | 'female'
  companionHarassEnabled: boolean
  ageConfirmed18: boolean
}

const SETTINGS_KEY = 'wubigrok_whisper_companion_settings'

function loadSettings(): CompanionSettings {
  try {
    const r = localStorage.getItem(SETTINGS_KEY)
    if (r) return JSON.parse(r)
  } catch (_) {}
  return { companionName: '', companionGender: 'female', companionHarassEnabled: false, ageConfirmed18: false }
}

function saveSettings(s: CompanionSettings) {
  try { localStorage.setItem(SETTINGS_KEY, JSON.stringify(s)) } catch (_) {}
}

export default function WhisperSettingsPanel({
  activePersonality, personalities, adultMode, engineID,
  onPersonalityChange, onAdultModeChange, onClearSession
}: Props) {
  const [engineList, setEngineList] = useState<string[]>([])
  const [selectedEngine, setSelectedEngine] = useState(engineID)
  const [clearConfirm, setClearConfirm] = useState(false)
  const [personalityOpen, setPersonalityOpen] = useState(false)

  // ── 对齐 ackem: 伴侣设置 ──
  const [settings, setSettings] = useState<CompanionSettings>(loadSettings)

  useEffect(() => {
    (async () => {
      try { const list = await (App as any).GetEngineList?.() as string[]; if (list) setEngineList(list) }
      catch (_) { setEngineList(['default']) }
    })()
  }, [])

  const updateSettings = (patch: Partial<CompanionSettings>) => {
    const next = { ...settings, ...patch }
    setSettings(next)
    saveSettings(next)
  }

  const handleEngineChange = async (id: string) => {
    setSelectedEngine(id)
    try { await (App as any).WhisperSetEngine?.(id) } catch (_) {}
  }

  const handleClear = async () => {
    await onClearSession()
    setClearConfirm(false)
    message.success('会话已清除')
  }

  const currentPersonality = personalities.find(p => p.id === activePersonality)
  const isPersonality18Plus = currentPersonality?.requiresAdult18 ?? false

  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 16 }}>
      <Text strong style={{ fontSize: 13, color: '#d4d4d8' }}>轻语设置</Text>

      {/* ── 对齐 ackem settings-companion: 伴侣称呼 ── */}
      <Card size="small" style={{ background: '#18181b', border: '1px solid #27272a', borderRadius: 8 }} bodyStyle={{ padding: 12 }}>
        <div style={{ fontSize: 12, color: '#a1a1aa', marginBottom: 6 }}>伴侣称呼</div>
        <Input
          size="small"
          placeholder="给伴侣起个名字..."
          value={settings.companionName}
          onChange={e => updateSettings({ companionName: e.target.value })}
          style={{ background: '#27272a', border: '1px solid #3f3f46', color: '#d4d4d8', borderRadius: 6 }}
        />
      </Card>

      {/* ── 对齐 ackem: 性别选择 ── */}
      <Card size="small" style={{ background: '#18181b', border: '1px solid #27272a', borderRadius: 8 }} bodyStyle={{ padding: 12 }}>
        <div style={{ fontSize: 12, color: '#a1a1aa', marginBottom: 6 }}>性别</div>
        <Radio.Group
          value={settings.companionGender}
          onChange={e => updateSettings({ companionGender: e.target.value })}
          size="small"
        >
          <Radio.Button value="female" style={{ fontSize: 12 }}>👩 女性</Radio.Button>
          <Radio.Button value="male" style={{ fontSize: 12 }}>🤵 男性</Radio.Button>
        </Radio.Group>
      </Card>

      {/* ── 人格选择 ── */}
      <Card size="small" style={{ background: '#18181b', border: '1px solid #27272a', borderRadius: 8 }} bodyStyle={{ padding: 12 }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 8 }}>
          <span style={{ fontSize: 12, color: '#a1a1aa' }}>伴侣人格</span>
          <Button size="small" icon={<SwapOutlined />} onClick={() => setPersonalityOpen(true)}>
            {currentPersonality?.label || '选择'}
          </Button>
        </div>
        {currentPersonality && (
          <div style={{ fontSize: 11, color: '#71717a' }}>
            {currentPersonality.gender === 'male' ? '🤵' : '👩'} {currentPersonality.label}
            {isPersonality18Plus && <Tag color="red" style={{ marginLeft: 6, fontSize: 10 }}>18+</Tag>}
            <div style={{ marginTop: 4 }}>T:{currentPersonality.dims.T} I:{currentPersonality.dims.I} S:{currentPersonality.dims.S} O:{currentPersonality.dims.O} R:{currentPersonality.dims.R}</div>
          </div>
        )}
      </Card>

      {/* ── 对齐 ackem: 伴侣主动搭话 ── */}
      <Card size="small" style={{ background: '#18181b', border: '1px solid #27272a', borderRadius: 8 }} bodyStyle={{ padding: 12 }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div>
            <div style={{ fontSize: 12, color: '#a1a1aa' }}>伴侣主动搭话</div>
            <div style={{ fontSize: 10, color: '#52525b', marginTop: 2 }}>允许伴侣在不活跃时主动发起对话</div>
          </div>
          <Switch
            size="small"
            checked={settings.companionHarassEnabled}
            onChange={v => updateSettings({ companionHarassEnabled: v })}
          />
        </div>
      </Card>

      {/* ── 对齐 ackem settings-safety: 年龄确认 ── */}
      <Card size="small" style={{ background: '#18181b', border: '1px solid #27272a', borderRadius: 8 }} bodyStyle={{ padding: 12 }}>
        <div style={{ fontSize: 12, color: '#a1a1aa', marginBottom: 8 }}>安全设置</div>
        <Checkbox
          checked={settings.ageConfirmed18}
          onChange={e => {
            const checked = e.target.checked
            updateSettings({ ageConfirmed18: checked })
            if (!checked && adultMode) onAdultModeChange(false)
            if (checked) message.success('已确认年满 18 岁')
            else message.info('已取消年龄确认，成人模式已关闭')
          }}
          style={{ fontSize: 12, color: '#a1a1aa' }}
        >
          我确认已年满 18 岁
        </Checkbox>
      </Card>

      {/* ── 成人模式（对齐 ackem adultContentMode）── */}
      <Card size="small" style={{ background: '#18181b', border: '1px solid #27272a', borderRadius: 8 }} bodyStyle={{ padding: 12 }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div>
            <div style={{ fontSize: 12, color: '#a1a1aa' }}>成人模式 (18+)</div>
            {!settings.ageConfirmed18 && (
              <div style={{ fontSize: 10, color: '#f87171', marginTop: 2 }}>需要先确认年龄</div>
            )}
          </div>
          <Switch
            size="small"
            checked={adultMode}
            disabled={!settings.ageConfirmed18}
            onChange={onAdultModeChange}
          />
        </div>
      </Card>

      {/* ── 引擎选择 ── */}
      <Card size="small" style={{ background: '#18181b', border: '1px solid #27272a', borderRadius: 8 }} bodyStyle={{ padding: 12 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <ApiOutlined style={{ color: '#a1a1aa' }} />
          <Select size="small" value={selectedEngine} onChange={handleEngineChange}
            style={{ flex: 1 }} options={engineList.map(e => ({ value: e, label: e }))} />
        </div>
      </Card>

      {/* ── 清除会话 ── */}
      <Card size="small" style={{ background: '#18181b', border: '1px solid #27272a', borderRadius: 8 }} bodyStyle={{ padding: 12 }}>
        <Button size="small" danger icon={<ClearOutlined />} block onClick={() => setClearConfirm(true)}>
          清除当前会话
        </Button>
      </Card>

      {/* 人格选择弹窗 */}
      <Modal title="选择伴侣人格" open={personalityOpen} onCancel={() => setPersonalityOpen(false)}
        footer={null} width={680} centered>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(160px, 1fr))', gap: 8, maxHeight: 400, overflow: 'auto' }}>
          {personalities.map(p => {
            const locked = Boolean(p.requiresAdult18 && !settings.ageConfirmed18)
            return (
              <Card key={p.id} hoverable={!locked} size="small"
                onClick={() => {
                  if (locked) { message.warning('此人格需要先确认年满 18 岁'); return }
                  onPersonalityChange(p.id); setPersonalityOpen(false)
                }}
                style={{
                  border: activePersonality === p.id ? '2px solid #e85388' : '1px solid #27272a',
                  background: locked ? '#1a1a1a' : activePersonality === p.id ? '#e8538808' : '#18181b',
                  borderRadius: 8, cursor: locked ? 'not-allowed' : 'pointer',
                  opacity: locked ? 0.5 : 1,
                }}
                bodyStyle={{ padding: '8px 10px' }}>
                <div style={{ fontSize: 12, fontWeight: 600, color: '#d4d4d8' }}>
                  {p.gender === 'male' ? '🤵' : '👩'} {p.label}
                  {p.requiresAdult18 && <Tag color="red" style={{ marginLeft: 4, fontSize: 9 }}>18+</Tag>}
                </div>
                {locked && <div style={{ fontSize: 9, color: '#f87171', marginTop: 2 }}>需年龄确认</div>}
              </Card>
            )
          })}
        </div>
      </Modal>

      {/* 清除确认 */}
      <Modal title="确认清除" open={clearConfirm} onOk={handleClear} onCancel={() => setClearConfirm(false)}
        okText="清除" cancelText="取消" okButtonProps={{ danger: true }}>
        <p>清除后将丢失当前会话的所有上下文和记忆状态。确定继续？</p>
      </Modal>
    </div>
  )
}
