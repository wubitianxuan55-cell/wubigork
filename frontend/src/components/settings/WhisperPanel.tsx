import React, { useEffect, useState } from 'react'
import { Button, Checkbox, Input, Popconfirm, Radio, Select, Switch, Tag, Typography, message } from 'antd'
import * as App from '../../../wailsjs/go/app/App'
import SettingsSection from './SettingsSection'

const PERSONALITY_KEY = 'gaea_whisper_personality'
const LEGACY_PERSONALITY_KEY = 'wubigrok_whisper_personality'
const COMPANION_SETTINGS_KEY = 'gaea_whisper_companion_settings'
const LEGACY_COMPANION_SETTINGS_KEY = 'wubigrok_whisper_companion_settings'
const ADULT_MODE_KEY = 'gaea_whisper_adult_mode'
const TOPICS_KEY = 'gaea_whisper_topics'
const LEGACY_TOPICS_KEY = 'wubigrok_whisper_topics'

interface Personality {
  id: string; label: string; gender: string; tags?: string[]
  dims: { T: number; I: number; S: number; O: number; R: number }
  voiceGuide?: string; requiresAdult18?: boolean
}

interface CompanionSettings {
  companionName: string
  companionGender: 'male' | 'female'
  companionHarassEnabled: boolean
  ageConfirmed18: boolean
}

function loadSettings(): CompanionSettings {
  try {
    const r = localStorage.getItem(COMPANION_SETTINGS_KEY) ?? localStorage.getItem(LEGACY_COMPANION_SETTINGS_KEY)
    if (r) return JSON.parse(r)
  } catch (_) {}
  return { companionName: '', companionGender: 'female', companionHarassEnabled: false, ageConfirmed18: false }
}

function saveSettings(s: CompanionSettings) {
  try { localStorage.setItem(COMPANION_SETTINGS_KEY, JSON.stringify(s)) } catch (_) {}
}

/** WhisperPanel — 轻语设置（原轻语界面设置面板，合并到设置中心） */
const WhisperPanel: React.FC = () => {
  const [settings, setSettings] = useState<CompanionSettings>(loadSettings)
  const [personalities, setPersonalities] = useState<Personality[]>([])
  const [activePersonality, setActivePersonality] = useState<string>(() => {
    try { return (localStorage.getItem(PERSONALITY_KEY) ?? localStorage.getItem(LEGACY_PERSONALITY_KEY)) || 'gaea' } catch { return 'gaea' }
  })
  const [adultMode, setAdultMode] = useState<boolean>(() => {
    try { return localStorage.getItem(ADULT_MODE_KEY) === '1' } catch { return false }
  })

  useEffect(() => {
    try { App.WhisperGetPersonalities().then((p: any) => setPersonalities(p || [])).catch(() => {}) } catch (_) {}
  }, [])

  const updateSettings = (patch: Partial<CompanionSettings>) => {
    const next = { ...settings, ...patch }
    setSettings(next)
    saveSettings(next)
  }

  const handleSwitchPersonality = async (id: string) => {
    try { await (App as any).WhisperClearSession(activePersonality) } catch (_) {}
    setActivePersonality(id)
    try { localStorage.setItem(PERSONALITY_KEY, id) } catch (_) {}
    message.success(`已切换为「${personalities.find(p => p.id === id)?.label || id}」人格（轻语界面生效）`)
  }

  const handleAdultMode = async (v: boolean) => {
    setAdultMode(v)
    try { localStorage.setItem(ADULT_MODE_KEY, v ? '1' : '0') } catch (_) {}
    try { await (App as any).WhisperSetAdultMode(activePersonality, v) } catch (_) {}
    message.success(v ? '已开启成人模式' : '已关闭成人模式')
  }

  const handleClearAll = async () => {
    try { localStorage.removeItem(TOPICS_KEY); localStorage.removeItem(LEGACY_TOPICS_KEY) } catch (_) {}
    try { await (App as any).WhisperClearSession(activePersonality) } catch (_) {}
    message.success('轻语全部会话已清除')
  }

  const currentPersonality = personalities.find(p => p.id === activePersonality)

  return (
    <>
      <SettingsSection
        title={<>AI 伴侣</>}
        desc="轻语 AI 伴侣的基础设定（称呼 / 性别）。"
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <div>
            <Typography.Text style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)', display: 'block', marginBottom: 4 }}>称呼</Typography.Text>
            <Input size="small" placeholder="给 AI 起个名字…" value={settings.companionName} style={{ maxWidth: 260 }}
              onChange={(e) => updateSettings({ companionName: e.target.value })} />
          </div>
          <div>
            <Typography.Text style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)', display: 'block', marginBottom: 4 }}>性别</Typography.Text>
            <Radio.Group size="small" value={settings.companionGender}
              onChange={(e) => updateSettings({ companionGender: e.target.value })}>
              <Radio.Button value="female" style={{ fontSize: 12 }}>👩 女性</Radio.Button>
              <Radio.Button value="male" style={{ fontSize: 12 }}>🤵 男性</Radio.Button>
            </Radio.Group>
          </div>
        </div>
      </SettingsSection>

      <SettingsSection
        title={<>人格</>}
        desc="轻语对话使用的人格（默认 gaea 大地女神）。切换即时生效。"
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <Select size="small" value={activePersonality} style={{ width: 200 }}
            onChange={handleSwitchPersonality}
            options={personalities.map(p => ({ value: p.id, label: `${p.gender === 'male' ? '🤵' : '👩'} ${p.label}${p.requiresAdult18 ? ' (18+)' : ''}` }))} />
          {currentPersonality && (
            <span style={{ display: 'flex', gap: 10, fontSize: 11, color: 'var(--md-sys-color-text-secondary)' }}>
              {['T', 'I', 'S', 'O', 'R'].map((dim, i) => {
                const vals = [currentPersonality.dims.T, currentPersonality.dims.I, currentPersonality.dims.S, currentPersonality.dims.O, currentPersonality.dims.R]
                return (
                  <span key={dim}>
                    <span style={{ color: 'var(--md-sys-color-primary)' }}>{dim}</span>
                    <span style={{ marginLeft: 2 }}>{vals[i]}</span>
                  </span>
                )
              })}
            </span>
          )}
        </div>
        <Typography.Paragraph type="secondary" style={{ fontSize: 11, margin: '8px 0 0' }}>
          角色管理（自定义人格 / 角色剧照 / 小说互传）在轻语界面「虚拟助手管理中心」操作。
        </Typography.Paragraph>
      </SettingsSection>

      <SettingsSection
        title={<>互动</>}
        desc="轻语互动行为设置。"
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <div>
              <Typography.Text style={{ fontSize: 12, color: 'var(--md-sys-color-text)' }}>主动搭话</Typography.Text>
              <div style={{ fontSize: 10, color: 'var(--md-sys-color-text-secondary)', marginTop: 2 }}>允许 AI 在不活跃时主动发起对话</div>
            </div>
            <Switch size="small" checked={settings.companionHarassEnabled}
              onChange={(v) => updateSettings({ companionHarassEnabled: v })} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <div>
              <Typography.Text style={{ fontSize: 12, color: 'var(--md-sys-color-text)' }}>成人模式</Typography.Text>
              <div style={{ fontSize: 10, color: 'var(--md-sys-color-text-secondary)', marginTop: 2 }}>启用更亲密的话题和互动</div>
            </div>
            <Switch size="small" checked={adultMode} disabled={!settings.ageConfirmed18} onChange={handleAdultMode} />
          </div>
          <Checkbox
            checked={settings.ageConfirmed18}
            onChange={(e) => {
              const checked = e.target.checked
              updateSettings({ ageConfirmed18: checked })
              if (!checked && adultMode) handleAdultMode(false)
            }}
            style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)' }}
          >
            我确认已年满 18 岁
          </Checkbox>
        </div>
      </SettingsSection>

      <SettingsSection
        title={<>数据管理</>}
        desc="清除轻语全部会话记录与后端上下文。"
      >
        <Popconfirm title="确认清除全部轻语会话？" okText="清除" cancelText="取消" onConfirm={handleClearAll}>
          <Button size="small" danger>清除全部会话</Button>
        </Popconfirm>
      </SettingsSection>
    </>
  )
}

export default WhisperPanel
