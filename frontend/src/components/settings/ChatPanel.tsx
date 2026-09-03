import React, { useCallback, useEffect, useState } from 'react'
import { Input, Radio, Select, Switch, Typography, message } from 'antd'
import { TeamOutlined, SmileOutlined, AudioOutlined } from '@ant-design/icons'
import * as App from '../../../src/wailsjsCompat'
import { applyVoiceSettings, getVoiceSettings } from '../../api/settings'
import SettingsSection from './SettingsSection'
import { useT } from '../../gaea/lib/i18n'

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

// 微软 Edge TTS 中文音色（label 经 i18n，组件内派生；value 为服务端标识不变）
const TTS_VOICE_VALUES = [
  { value: 'zh-CN-YunxiNeural', key: 'settings.chat.ttsYunxi' },
  { value: 'zh-CN-XiaoxiaoNeural', key: 'settings.chat.ttsXiaoxiao' },
  { value: 'zh-CN-YunjianNeural', key: 'settings.chat.ttsYunjian' },
  { value: 'zh-CN-XiaoyiNeural', key: 'settings.chat.ttsXiaoyi' },
] as const

const HERDSMAN_VOICES = ['serena', 'vivian', 'sohee', 'aiden', 'dylan', 'eric', 'ono_anna', 'ryan', 'uncle_fu']

// xAI Grok TTS 音色单一来源（模型中心 modelcenter/utils.tsx，T6-6.4 单源收敛）
import { XAI_VOICES } from '../../pages/modelcenter/utils'

// CosyVoice2 内置音色（本地服务端 /v1/audio/info 返回，查询失败时兜底；value 为服务端标识保持中文）
const COSYVOICE_VOICES = ['中文女', '中文男', '英文女', '英文男']

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
  const t = useT()
  const [settings, setSettings] = useState<CompanionSettings>(loadSettings)
  const [personalities, setPersonalities] = useState<Personality[]>([])
  const [activePersonality, setActivePersonality] = useState<string>(() => {
    try { return (localStorage.getItem(PERSONALITY_KEY) ?? localStorage.getItem(LEGACY_PERSONALITY_KEY)) || 'gaea' } catch { return 'gaea' }
  })
  const [voice, setVoice] = useState<Record<string, unknown>>({})
  const [herdsmanVoices, setHerdsmanVoices] = useState<string[]>([])
  const [ttsModel, setTtsModel] = useState('')

  useEffect(() => {
    try {
      App.WhisperGetPersonalities().then((p) => setPersonalities(p || [])).catch(() => {})
    } catch (_) { /* 未初始化时静默 */ }
    getVoiceSettings().then((v) => setVoice(v || {})).catch(() => {})
    // wailsjsCompat 直调在浏览器/?mock=1 下（window.go 为空）会同步抛——
    // `?.` 只防「导出缺失」，生成函数存在即会执行，必须整体 try 兜底，
    // 否则一处直调失败会被 ErrorBoundary 兜成整个设置板块白屏。
    try {
      App.GetVoicePipelineConfig?.().then((p) => {
        const model = p?.chatTts?.model || p?.tts?.model || ''
        setTtsModel(model)
        const lower = model.toLowerCase()
        if (lower.includes('qwen3') || lower.includes('customvoice') || lower.includes('cosyvoice')) {
          App.GetTTSSpeakers?.(model).then((speakers: string[]) => {
            if (Array.isArray(speakers) && speakers.length > 0) setHerdsmanVoices(speakers)
          }).catch(() => {})
        }
      }).catch(() => {})
    } catch (_) { /* 未初始化时静默 */ }
  }, [])

  const isHerdsmanVoice = ttsModel.toLowerCase().includes('qwen3') || ttsModel.toLowerCase().includes('customvoice')
  const isXaiVoice = ttsModel.toLowerCase().includes('grok-tts')
  const isCosyvoiceVoice = ttsModel.toLowerCase().includes('cosyvoice')
  // 音色 label 经 i18n 组件内派生（原为模块级 TTS_VOICES / VOICE_LABELS 常量）
  const ttsVoices = TTS_VOICE_VALUES.map((v) => ({ value: v.value, label: t(v.key) }))
  const herdsmanLabels: Record<string, string> = {
    serena: `Serena (${t('settings.chat.tagFemale')})`,
    vivian: `Vivian (${t('settings.chat.tagFemale')})`,
    sohee: `Sohee (${t('settings.chat.tagFemale')})`,
    aiden: `Aiden (${t('settings.chat.tagMale')})`,
    dylan: `Dylan (${t('settings.chat.tagMale')})`,
    eric: `Eric (${t('settings.chat.tagMale')})`,
    ryan: `Ryan (${t('settings.chat.tagMale')})`,
    ono_anna: `Anna (${t('settings.chat.tagFemale')})`,
    uncle_fu: `Uncle Fu (${t('settings.chat.tagMale')})`,
  }
  const cosyLabels: Record<string, string> = {
    中文女: t('settings.chat.cosyZhF'),
    中文男: t('settings.chat.cosyZhM'),
    英文女: t('settings.chat.cosyEnF'),
    英文男: t('settings.chat.cosyEnM'),
  }
  const voiceOptions = isHerdsmanVoice
    ? (herdsmanVoices.length > 0 ? herdsmanVoices : HERDSMAN_VOICES).map(v => ({ value: v, label: herdsmanLabels[v] || v }))
    : isXaiVoice
      ? XAI_VOICES
      : isCosyvoiceVoice
        ? (herdsmanVoices.length > 0 ? herdsmanVoices : COSYVOICE_VOICES).map(v => ({ value: v, label: cosyLabels[v] || v }))
        : ttsVoices
  const ttsVoice = (voice.ttsVoice as string) || ''
  const effectiveVoice = isHerdsmanVoice
    ? (herdsmanVoices.length > 0 ? herdsmanVoices : HERDSMAN_VOICES).includes(ttsVoice)
      ? ttsVoice
      : 'serena'
    : isXaiVoice
      ? XAI_VOICES.some(v => v.value === ttsVoice) ? ttsVoice : 'eve'
      : isCosyvoiceVoice
        ? (herdsmanVoices.length > 0 ? herdsmanVoices : COSYVOICE_VOICES).includes(ttsVoice) ? ttsVoice : '中文女'
        : ttsVoice

  const updateSettings = (patch: Partial<CompanionSettings>) => {
    const next = { ...settings, ...patch }
    setSettings(next)
    saveSettings(next)
  }

  const handleSwitchPersonality = async (id: string) => {
    try { await App.WhisperClearSession(activePersonality) } catch (_) { /* 忽略 */ }
    setActivePersonality(id)
    try { localStorage.setItem(PERSONALITY_KEY, id) } catch (_) { /* 忽略 */ }
    message.success(t('settings.chat.personaSwitched', { name: personalities.find((p) => p.id === id)?.label || id }))
  }

  const patchVoice = useCallback((key: string, value: unknown) => {
    setVoice((prev) => ({ ...prev, [key]: value }))
    applyVoiceSettings({ [key]: value }).catch(() => message.warning(t('settings.chat.voiceSaveFailed')))
  }, [t])

  const currentPersonality = personalities.find((p) => p.id === activePersonality)

  return (
    <>
      <SettingsSection
        icon={<span style={{ fontSize: 15 }}><TeamOutlined /></span>}
        title={t('settings.chat.companionTitle')}
        desc={t('settings.chat.companionDesc')}
      >
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: 14 }}>
          <div>
            <Typography.Text style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)', display: 'block', marginBottom: 4 }}>{t('settings.chat.nameLabel')}</Typography.Text>
            <Input size="small" placeholder={t('settings.chat.namePlaceholder')} value={settings.companionName}
              onChange={(e) => updateSettings({ companionName: e.target.value })} style={{ maxWidth: 260 }} />
          </div>
          <div>
            <Typography.Text style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)', display: 'block', marginBottom: 4 }}>{t('settings.chat.genderLabel')}</Typography.Text>
            <Radio.Group size="small" value={settings.companionGender}
              onChange={(e) => updateSettings({ companionGender: e.target.value })}>
              <Radio.Button value="female" style={{ fontSize: 12 }}>{t('settings.chat.genderFemale')}</Radio.Button>
              <Radio.Button value="male" style={{ fontSize: 12 }}>{t('settings.chat.genderMale')}</Radio.Button>
            </Radio.Group>
          </div>
        </div>
      </SettingsSection>

      <SettingsSection
        icon={<span style={{ fontSize: 15 }}><SmileOutlined /></span>}
        title={t('settings.chat.personaTitle')}
        desc={t('settings.chat.personaDesc')}
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
        icon={<span style={{ fontSize: 15 }}><AudioOutlined /></span>}
        title={t('settings.chat.voiceTitle')}
        desc={t('settings.chat.voiceDesc')}
        instant
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <div>
              <div style={{ fontSize: 13, color: 'var(--md-sys-color-text)' }}>{t('settings.chat.voiceTitle')}</div>
              <div style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)' }}>{t('settings.chat.voiceEnableDesc')}</div>
            </div>
            <Switch size="small" checked={!!voice.enabled} onChange={(v) => patchVoice('enabled', v)} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <div>
              <div style={{ fontSize: 13, color: 'var(--md-sys-color-text)' }}>{t('settings.chat.ttsRow')}</div>
              <div style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)' }}>{t('settings.chat.ttsDesc')}</div>
            </div>
            <Switch size="small" checked={!!voice.ttsEnabled} onChange={(v) => patchVoice('ttsEnabled', v)} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <div>
              <div style={{ fontSize: 13, color: 'var(--md-sys-color-text)' }}>{t('settings.chat.voiceSelRow')}</div>
              <div style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)' }}>{t('settings.chat.voiceSelDesc')}</div>
            </div>
            <Select
              size="small"
              value={effectiveVoice || undefined}
              placeholder={t('settings.chat.voicePlaceholder')}
              onChange={(v) => patchVoice('ttsVoice', v)}
              style={{ width: 180 }}
              options={voiceOptions}
            />
          </div>
        </div>
      </SettingsSection>
    </>
  )
}

export default ChatPanel
