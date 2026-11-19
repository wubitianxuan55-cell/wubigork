import React, { useCallback, useEffect, useState } from 'react'
import { Select, message } from 'antd'
import { AudioOutlined } from '@ant-design/icons'
import { app } from '../../gaea/lib/bridge'
import { applyVoiceSettings, getVoiceSettings } from '../../api/settings'
import SettingsSection from './SettingsSection'
import { useT } from '../../gaea/lib/i18n'
import { XAI_VOICES } from '../../pages/modelcenter/utils'

/** ChatVoiceSection — 全局 TTS 默认音色（设置中枢批 2 从 ChatPanel 拆出归入
 *  语音组；与 VoiceSettingsPanel 互补：本段管 tts_voice 全局默认值按引擎分档，
 *  面板管朗读引擎路由/Breeze 克隆/麦克风深配——非冗余双入口，两者同键同源。 */

/** GetVoicePipelineConfig 动态载荷中 TTS 模型读取面（bridge Record<string, unknown> 收窄） */
interface VoicePipelineInfo {
  chatTts?: { model?: string }
  tts?: { model?: string }
}

// 微软 Edge TTS 中文音色（label 经 i18n，组件内派生；value 为服务端标识不变）
const TTS_VOICE_VALUES = [
  { value: 'zh-CN-YunxiNeural', key: 'settings.chat.ttsYunxi' },
  { value: 'zh-CN-XiaoxiaoNeural', key: 'settings.chat.ttsXiaoxiao' },
  { value: 'zh-CN-YunjianNeural', key: 'settings.chat.ttsYunjian' },
  { value: 'zh-CN-XiaoyiNeural', key: 'settings.chat.ttsXiaoyi' },
] as const

const HERDSMAN_VOICES = ['serena', 'vivian', 'sohee', 'aiden', 'dylan', 'eric', 'ono_anna', 'ryan', 'uncle_fu']

// CosyVoice2 内置音色（本地服务端 /v1/audio/info 返回，查询失败时兜底；value 为服务端标识保持中文）
const COSYVOICE_VOICES = ['中文女', '中文男', '英文女', '英文男']

const ChatVoiceSection: React.FC = () => {
  const t = useT()
  const [voice, setVoice] = useState<Record<string, unknown>>({})
  const [herdsmanVoices, setHerdsmanVoices] = useState<string[]>([])
  const [ttsModel, setTtsModel] = useState('')

  useEffect(() => {
    getVoiceSettings().then((v) => setVoice(v || {})).catch(() => {})
    // bridge 调用在浏览器/?mock=1 下走 dev mock（空态返回、不同步抛）；保留整体
    // try/catch 兜底——一处异常不会被兜成整个设置板块白屏（随 ChatPanel 原样迁入）。
    try {
      app.GetVoicePipelineConfig().then((p) => {
        const pipeline = p as VoicePipelineInfo | undefined
        const model = pipeline?.chatTts?.model || pipeline?.tts?.model || ''
        setTtsModel(model)
        const lower = model.toLowerCase()
        if (lower.includes('qwen3') || lower.includes('customvoice') || lower.includes('cosyvoice')) {
          app.GetTTSSpeakers(model).then((speakers: string[]) => {
            if (Array.isArray(speakers) && speakers.length > 0) setHerdsmanVoices(speakers)
          }).catch(() => {})
        }
      }).catch(() => {})
    } catch { /* 未初始化时静默 */ }
  }, [])

  const isHerdsmanVoice = ttsModel.toLowerCase().includes('qwen3') || ttsModel.toLowerCase().includes('customvoice')
  const isXaiVoice = ttsModel.toLowerCase().includes('grok-tts')
  const isCosyvoiceVoice = ttsModel.toLowerCase().includes('cosyvoice')
  // 音色 label 经 i18n 组件内派生
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

  const patchVoice = useCallback((key: string, value: unknown) => {
    setVoice((prev) => ({ ...prev, [key]: value }))
    applyVoiceSettings({ [key]: value }).catch(() => message.warning(t('settings.chat.voiceSaveFailed')))
  }, [t])

  return (
    <SettingsSection
      icon={<span style={{ fontSize: 15 }}><AudioOutlined /></span>}
      title={t('settings.chat.voiceTitle')}
      desc={t('settings.chat.voiceDesc')}
      instant
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
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
  )
}

export default ChatVoiceSection
