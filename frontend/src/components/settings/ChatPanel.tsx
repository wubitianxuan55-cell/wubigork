import React, { useCallback, useEffect, useState } from 'react'
import { Input, Radio, Select, Switch, Typography, message } from 'antd'
import * as App from '../../../wailsjs/go/app/App'
import { applyVoiceSettings, getVoiceSettings } from '../../api/settings'
import SettingsSection from './SettingsSection'

const PERSONALITY_KEY = 'gaea_whisper_personality'
const LEGACY_PERSONALITY_KEY = 'wubigrok_whisper_personality'
const COMPANION_SETTINGS_KEY = 'gaea_whisper_companion_settings'
const LEGACY_COMPANION_SETTINGS_KEY = 'wubigrok_whisper_companion_settings'

interface Personality {
  id: string
  label: string
  gender: string
  dims: { T: number; I: number; S: number; O: number; R: number }
}

interface CompanionSettings {
  companionName: string
  companionGender: 'male' | 'female'
}

const TTS_VOICES = [
  { value: 'zh-CN-YunxiNeural', label: '云希 (男)' },
  { value: 'zh-CN-XiaoxiaoNeural', label: '晓晓 (女)' },
  { value: 'zh-CN-YunjianNeural', label: '云健 (男)' },
  { value: 'zh-CN-XiaoyiNeural', label: '晓伊 (女)' },
]

function loadSettings(): CompanionSettings {
  try {
    const r = localStorage.getItem(COMPANION_SETTINGS_KEY) ?? localStorage.getItem(LEGACY_COMPANION_SETTINGS_KEY)
    if (r) {
      const parsed = JSON.parse(r)
      return {
        companionName: parsed?.companionName || '',
        companionGender: parsed?.companionGender === 'male' ? 'male' : 'female',
      }
    }
  } catch (_) { /* 忽略损坏的本地设置 */ }
  return { companionName: '', companionGender: 'female' }
}

function saveSettings(s: CompanionSettings) {
  try { localStorage.setItem(COMPANION_SETTINGS_KEY, JSON.stringify(s)) } catch (_) { /* 忽略 */ }
}

/** ChatPanel — 聊天设置：AI 伴侣（称呼/性别）、默认人格与语音对话核心项 */
const ChatPanel: React.FC = () => {
  const [settings, setSettings] = useState<CompanionSettings>(loadSettings)
  const [personalities, setPersonalities] = useState<Personality[]>([])
  const [activePersonality, setActivePersonality] = useState<string>(() => {
    try { return (localStorage.getItem(PERSONALITY_KEY) ?? localStorage.getItem(LEGACY_PERSONALITY_KEY)) || 'gaea' } catch { return 'gaea' }
  })
  const [voice, setVoice] = useState<Record<string, any>>({})

  useEffect(() => {
    try {
      App.WhisperGetPersonalities().then((p: any) => setPersonalities(p || [])).catch(() => {})
    } catch (_) { /* 未初始化时静默 */ }
    getVoiceSettings().then((v) => setVoice(v || {})).catch(() => {})
  }, [])

  const updateSettings = (patch: Partial<CompanionSettings>) => {
    const next = { ...settings, ...patch }
    setSettings(next)
    saveSettings(next)
  }

  const handleSwitchPersonality = async (id: string) => {
    try { await (App as any).WhisperClearSession(activePersonality) } catch (_) { /* 忽略 */ }
    setActivePersonality(id)
    try { localStorage.setItem(PERSONALITY_KEY, id) } catch (_) { /* 忽略 */ }
    message.success(`已切换为「${personalities.find((p) => p.id === id)?.label || id}」人格（聊天板块生效）`)
  }

  const patchVoice = useCallback((key: string, value: any) => {
    setVoice((prev) => ({ ...prev, [key]: value }))
    applyVoiceSettings({ [key]: value }).catch(() => message.warning('语音设置保存失败'))
  }, [])

  const currentPersonality = personalities.find((p) => p.id === activePersonality)

  return (
    <>
      <SettingsSection
        icon={<span style={{ fontSize: 15 }}>🤝</span>}
        title="AI 伴侣"
        desc="聊天 AI 伴侣的基础设定（称呼/性别），在聊天板块生效。"
      >
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: 14 }}>
          <div>
            <Typography.Text style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)', display: 'block', marginBottom: 4 }}>称呼</Typography.Text>
            <Input size="small" placeholder="给 AI 起个名字…" value={settings.companionName}
              onChange={(e) => updateSettings({ companionName: e.target.value })} style={{ maxWidth: 260 }} />
          </div>
          <div>
            <Typography.Text style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)', display: 'block', marginBottom: 4 }}>性别</Typography.Text>
            <Radio.Group size="small" value={settings.companionGender}
              onChange={(e) => updateSettings({ companionGender: e.target.value })}>
              <Radio.Button value="female" style={{ fontSize: 12 }}>♀ 女性</Radio.Button>
              <Radio.Button value="male" style={{ fontSize: 12 }}>♂ 男性</Radio.Button>
            </Radio.Group>
          </div>
        </div>
      </SettingsSection>

      <SettingsSection
        icon={<span style={{ fontSize: 15 }}>🎭</span>}
        title="默认人格"
        desc="聊天默认使用的人格（默认 gaea 大地女神）。自定义人格与剧照在「角色库」管理。"
        instant
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
          <Select size="small" value={activePersonality} style={{ width: 200 }}
            onChange={handleSwitchPersonality}
            options={personalities.map((p) => ({ value: p.id, label: `${p.gender === 'male' ? '♂' : '♀'} ${p.label}` }))} />
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
      </SettingsSection>

      <SettingsSection
        icon={<span style={{ fontSize: 15 }}>🎙️</span>}
        title="语音对话"
        desc="聊天的语音输入与朗读回复核心项；完整面板（识别模式/阈值等）在聊天面板中。"
        instant
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <div>
              <div style={{ fontSize: 13, color: 'var(--md-sys-color-text)' }}>语音对话</div>
              <div style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)' }}>启用语音输入与输出</div>
            </div>
            <Switch size="small" checked={!!voice.enabled} onChange={(v) => patchVoice('enabled', v)} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <div>
              <div style={{ fontSize: 13, color: 'var(--md-sys-color-text)' }}>朗读回复</div>
              <div style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)' }}>AI 回复时自动朗读</div>
            </div>
            <Switch size="small" checked={!!voice.ttsEnabled} onChange={(v) => patchVoice('ttsEnabled', v)} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <div>
              <div style={{ fontSize: 13, color: 'var(--md-sys-color-text)' }}>合成音色</div>
              <div style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)' }}>朗读使用的 AI 音色</div>
            </div>
            <Select
              size="small"
              value={voice.ttsVoice || undefined}
              placeholder="选择音色"
              onChange={(v) => patchVoice('ttsVoice', v)}
              style={{ width: 180 }}
              options={TTS_VOICES}
            />
          </div>
        </div>
      </SettingsSection>
    </>
  )
}

export default ChatPanel
