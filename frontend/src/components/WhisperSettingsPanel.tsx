// WhisperSettingsPanel.tsx — 轻语专属设置面板 v2
// 100% 对齐 ackem SettingsPage gaea设置区域
// 使用 whisper-theme.css 设计 token

import React, { useState, useEffect } from 'react'
import { Button, Switch, Select, Modal, message, Card, Tag, Typography, Input, Radio, Checkbox } from 'antd'
import { ClearOutlined, SwapOutlined, ApiOutlined, SettingOutlined } from '@ant-design/icons'
import * as App from '../../wailsjs/go/app/App'
import WhisperPersonalityModal from './WhisperPersonalityModal'

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

// ─── SettingsSection 通用设置区块 ──────────────────────────────

const SettingsSection: React.FC<{ title: string; desc?: string; children: React.ReactNode }> = ({ title, desc, children }) => (
  <div style={{ marginBottom: 20 }}>
    <div style={{ display: 'flex', alignItems: 'baseline', gap: 8, marginBottom: 8 }}>
      <Text strong style={{ fontSize: 13, color: 'var(--whisper-ink)' }}>{title}</Text>
      {desc && <Text style={{ fontSize: 11, color: 'var(--whisper-ink-muted)' }}>{desc}</Text>}
    </div>
    <Card
      size="small"
      className="whisper-glass"
      style={{ borderRadius: 12, border: '1px solid var(--whisper-glass-border)' }}
      bodyStyle={{ padding: 14 }}
    >
      {children}
    </Card>
  </div>
)

// ─── 主组件 ─────────────────────────────────────────────────────

export default function WhisperSettingsPanel({
  activePersonality, personalities, adultMode, engineID,
  onPersonalityChange, onAdultModeChange, onClearSession
}: Props) {
  const [engineList, setEngineList] = useState<string[]>([])
  const [selectedEngine, setSelectedEngine] = useState(engineID)
  const [clearConfirm, setClearConfirm] = useState(false)
  const [personalityOpen, setPersonalityOpen] = useState(false)
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
    <div style={{ padding: '16px 12px', display: 'flex', flexDirection: 'column', gap: 4 }} className="whisper-scroll">
      {/* 页头 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 12, padding: '0 4px' }}>
        <SettingOutlined style={{ color: 'var(--whisper-accent)', fontSize: 16 }} />
        <Text strong style={{ fontSize: 15, color: 'var(--whisper-ink)' }}>轻语设置</Text>
      </div>

      {/* ── gaea设置 ── */}
      <SettingsSection title="gaea设置" desc="个性化你的 AI gaea">
        <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
          {/* 称呼 */}
          <div>
            <Text style={{ fontSize: 11, color: 'var(--whisper-ink-muted)', display: 'block', marginBottom: 4 }}>gaea称呼</Text>
            <Input
              size="small"
              placeholder="给gaea起个名字…"
              value={settings.companionName}
              onChange={e => updateSettings({ companionName: e.target.value })}
              style={{
                background: 'var(--whisper-glass-bg)',
                border: '1px solid var(--whisper-glass-border)',
                color: 'var(--whisper-ink)',
                borderRadius: 8,
                maxWidth: 260,
              }}
            />
          </div>

          {/* 性别 */}
          <div>
            <Text style={{ fontSize: 11, color: 'var(--whisper-ink-muted)', display: 'block', marginBottom: 4 }}>性别</Text>
            <Radio.Group
              value={settings.companionGender}
              onChange={e => updateSettings({ companionGender: e.target.value })}
              size="small"
            >
              <Radio.Button value="female" style={{ fontSize: 12 }}>👩 女性</Radio.Button>
              <Radio.Button value="male" style={{ fontSize: 12 }}>🤵 男性</Radio.Button>
            </Radio.Group>
          </div>

          {/* 人格 */}
          <div>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 4 }}>
              <Text style={{ fontSize: 11, color: 'var(--whisper-ink-muted)' }}>gaea人格</Text>
              <Button
                size="small"
                icon={<SwapOutlined />}
                onClick={() => setPersonalityOpen(true)}
                style={{
                  background: 'var(--whisper-glass-bg)',
                  border: '1px solid var(--whisper-glass-border)',
                  color: 'var(--whisper-ink)',
                  borderRadius: 8,
                  fontSize: 12,
                }}
              >
                {currentPersonality?.label || '选择人格'}
              </Button>
            </div>
            {currentPersonality && (
              <div style={{
                fontSize: 11, color: 'var(--whisper-ink-muted)',
                background: 'var(--whisper-glass-bg)', borderRadius: 8,
                padding: '8px 12px',
              }}>
                <div style={{ fontWeight: 600, color: 'var(--whisper-ink)', marginBottom: 4 }}>
                  {currentPersonality.gender === 'male' ? '🤵' : '👩'} {currentPersonality.label}
                  {isPersonality18Plus && <Tag color="red" style={{ marginLeft: 6, fontSize: 9 }}>18+</Tag>}
                </div>
                <div style={{ display: 'flex', gap: 10 }}>
                  {['T','I','S','O','R'].map((dim, i) => {
                    const vals = [currentPersonality.dims.T, currentPersonality.dims.I, currentPersonality.dims.S, currentPersonality.dims.O, currentPersonality.dims.R]
                    return (
                      <span key={dim}>
                        <span style={{ color: 'var(--whisper-accent)' }}>{dim}</span>
                        <span style={{ marginLeft: 2 }}>{vals[i]}</span>
                      </span>
                    )
                  })}
                </div>
              </div>
            )}
          </div>
        </div>
      </SettingsSection>

      {/* ── 互动设置 ── */}
      <SettingsSection title="互动设置">
        <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
          {/* 主动搭话 */}
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <div>
              <Text style={{ fontSize: 12, color: 'var(--whisper-ink)' }}>gaea主动搭话</Text>
              <div style={{ fontSize: 10, color: 'var(--whisper-ink-muted)', marginTop: 2 }}>允许gaea在不活跃时主动发起对话</div>
            </div>
            <Switch size="small" checked={settings.companionHarassEnabled} onChange={v => updateSettings({ companionHarassEnabled: v })} />
          </div>

          {/* 成人模式 */}
          <div>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <div>
                <Text style={{ fontSize: 12, color: 'var(--whisper-ink)' }}>成人模式</Text>
                <div style={{ fontSize: 10, color: 'var(--whisper-ink-muted)', marginTop: 2 }}>启用更亲密的话题和互动</div>
              </div>
              <Switch
                size="small"
                checked={adultMode}
                disabled={!settings.ageConfirmed18}
                onChange={onAdultModeChange}
              />
            </div>
            <Checkbox
              checked={settings.ageConfirmed18}
              onChange={e => {
                const checked = e.target.checked
                updateSettings({ ageConfirmed18: checked })
                if (!checked && adultMode) onAdultModeChange(false)
              }}
              style={{ fontSize: 11, color: 'var(--whisper-ink-muted)', marginTop: 6 }}
            >
              我确认已年满 18 岁
            </Checkbox>
          </div>
        </div>
      </SettingsSection>

      {/* ── 引擎设置 ── */}
      <SettingsSection title="模型引擎">
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <ApiOutlined style={{ color: 'var(--whisper-accent)' }} />
          <Select
            size="small"
            value={selectedEngine}
            onChange={handleEngineChange}
            style={{ flex: 1, maxWidth: 200 }}
            options={engineList.map(e => ({ value: e, label: e }))}
          />
        </div>
      </SettingsSection>

      {/* ── 数据管理 ── */}
      <SettingsSection title="数据管理">
        <Button
          size="small"
          danger
          icon={<ClearOutlined />}
          block
          onClick={() => setClearConfirm(true)}
          style={{ borderRadius: 8 }}
        >
          清除当前会话
        </Button>
        <div style={{ fontSize: 10, color: 'var(--whisper-ink-muted)', marginTop: 6, textAlign: 'center' }}>
          清除后丢失当前上下文和短期记忆，长期记忆保留
        </div>
      </SettingsSection>

      {/* ── 人格选择弹窗 ── */}
      <WhisperPersonalityModal
        open={personalityOpen}
        personalities={personalities}
        activePersonality={activePersonality}
        adultMode={adultMode}
        onClose={() => setPersonalityOpen(false)}
        onSwitch={(id) => { onPersonalityChange(id); setPersonalityOpen(false) }}
      />

      {/* ── 清除确认 ── */}
      <Modal
        title="确认清除"
        open={clearConfirm}
        onOk={handleClear}
        onCancel={() => setClearConfirm(false)}
        okText="清除"
        cancelText="取消"
        okButtonProps={{ danger: true }}
      >
        <p style={{ color: 'var(--whisper-ink)' }}>清除后将丢失当前会话的所有上下文和记忆状态。确定继续？</p>
      </Modal>
    </div>
  )
}
