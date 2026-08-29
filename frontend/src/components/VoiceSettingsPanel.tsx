// VoiceSettingsPanel.tsx — 聊天语音设置面板

import React, { useState, useEffect, useMemo } from 'react'
import { Card, Switch, Select, Slider, Button, Typography, Tag } from 'antd'
import { AudioOutlined, ReloadOutlined, CheckCircleOutlined, CloseCircleOutlined } from '@ant-design/icons'
import * as App from '../../src/wailsjsCompat'

const { Text } = Typography

/** 语音服务健康状态（后端 VoiceHealth 动态载荷的最小消费面） */
interface VoiceHealthInfo {
  asrReady?: boolean
  ttsReady?: boolean
  state?: string
  error?: string
}

// Edge TTS 音色（在线免费，zh-CN）
const EDGE_VOICES = [
  { value: 'zh-CN-YunxiNeural', label: '云希 (男)' },
  { value: 'zh-CN-XiaoxiaoNeural', label: '晓晓 (女)' },
  { value: 'zh-CN-YunjianNeural', label: '云健 (男)' },
  { value: 'zh-CN-XiaoyiNeural', label: '晓伊 (女)' },
]

// Herdsman qwen3-tts 音色兜底（服务端支持列表查询失败时使用）
const HERDSMAN_VOICES = ['serena', 'vivian', 'sohee', 'aiden', 'dylan', 'eric', 'ono_anna', 'ryan', 'uncle_fu']

// xAI Grok TTS 音色单一来源（模型中心 modelcenter/utils.tsx，T6-6.4 单源收敛）
import { XAI_VOICES } from '../pages/modelcenter/utils'

// CosyVoice2 内置音色（本地服务端 /v1/audio/info 返回，查询失败时兜底）
const COSYVOICE_VOICES = ['中文女', '中文男', '英文女', '英文男']


const VOICE_LABELS: Record<string, string> = {
  serena: 'Serena (女)',
  vivian: 'Vivian (女)',
  sohee: 'Sohee (女)',
  aiden: 'Aiden (男)',
  dylan: 'Dylan (男)',
  eric: 'Eric (男)',
  ryan: 'Ryan (男)',
  ono_anna: 'Anna (女)',
  uncle_fu: 'Uncle Fu (男)',
}

export default function VoiceSettingsPanel() {
  const [health, setHealth] = useState<VoiceHealthInfo | null>(null)
  const [checking, setChecking] = useState(false)

  // 语音设置
  const [ttsEnabled, setTtsEnabled] = useState(true)
  const [ttsVoice, setTtsVoice] = useState('zh-CN-YunxiNeural')
  const [ttsModel, setTtsModel] = useState('')
  const [herdsmanVoices, setHerdsmanVoices] = useState<string[]>([])
  const [voiceMode, setVoiceMode] = useState<'vad' | 'ptt'>('vad')
  const [interruptMs, setInterruptMs] = useState(500)
  const [silenceMs, setSilenceMs] = useState(1000)

  const checkHealth = async () => {
    setChecking(true)
    try {
      const h = await App.VoiceHealth?.()
      setHealth(h || { asrReady: false, ttsReady: false })
    } catch (_) {
      setHealth({ asrReady: false, ttsReady: false, error: '无法获取语音服务状态' })
    }
    setChecking(false)
  }

  const loadSettings = async () => {
    try {
      const [v, pipeline] = await Promise.all([
        App.VoiceGetSettings?.(),
        App.GetVoicePipelineConfig?.(),
      ])
      const settings = v || {}
      if (settings.ttsEnabled !== undefined) setTtsEnabled(!!settings.ttsEnabled)
      if (settings.ttsVoice) setTtsVoice(settings.ttsVoice)
      if (settings.voiceMode) setVoiceMode(settings.voiceMode === 'ptt' ? 'ptt' : 'vad')
      if (settings.interruptThresholdMs) setInterruptMs(settings.interruptThresholdMs)
      if (settings.silenceThresholdMs) setSilenceMs(settings.silenceThresholdMs)

      const model = pipeline?.chatTts?.model || pipeline?.tts?.model || ''
      setTtsModel(model)
      const lower = model.toLowerCase()
      if (lower.includes('qwen3') || lower.includes('customvoice') || lower.includes('cosyvoice')) {
        try {
          const speakers = (await App.GetTTSSpeakers?.(model)) || []
          if (Array.isArray(speakers) && speakers.length > 0) {
            setHerdsmanVoices(speakers)
            // 当前音色不在支持列表时自动对齐到默认音色
            if (settings.ttsVoice && !speakers.includes(settings.ttsVoice)) {
              const fallback = speakers.includes('serena') ? 'serena' : speakers[0]
              setTtsVoice(fallback)
              App.VoiceApplySettings?.({ ttsVoice: fallback }).catch(() => {})
            }
          }
        } catch (_) { /* 查询失败使用兜底列表 */ }
      }
    } catch (_) { /* 初始化失败忽略 */ }
  }

  useEffect(() => { checkHealth(); loadSettings() }, [])

  const applyVoiceSettings = (patch: Record<string, unknown>) => {
    App.VoiceApplySettings?.(patch).catch(() => {})
  }

  // 根据当前 TTS 模型决定音色选项：qwen3/customvoice → Herdsman 音色，其余 → Edge 音色
  const isHerdsman = useMemo(() => {
    const l = ttsModel.toLowerCase()
    return l.includes('qwen3') || l.includes('customvoice')
  }, [ttsModel])

  const isXai = useMemo(() => ttsModel.toLowerCase().includes('grok-tts'), [ttsModel])
  const isCosyvoice = useMemo(() => ttsModel.toLowerCase().includes('cosyvoice'), [ttsModel])

  const voiceOptions = useMemo(() => {
    if (isHerdsman) {
      const list = herdsmanVoices.length > 0 ? herdsmanVoices : HERDSMAN_VOICES
      return list.map(v => ({ value: v, label: VOICE_LABELS[v] || v }))
    }
    if (isXai) return XAI_VOICES
    if (isCosyvoice) {
      const list = herdsmanVoices.length > 0 ? herdsmanVoices : COSYVOICE_VOICES
      return list.map(v => ({ value: v, label: v }))
    }
    return EDGE_VOICES
  }, [isHerdsman, isXai, isCosyvoice, herdsmanVoices])

  const effectiveVoice = useMemo(() => {
    if (isHerdsman) {
      const list = herdsmanVoices.length > 0 ? herdsmanVoices : HERDSMAN_VOICES
      return list.includes(ttsVoice) ? ttsVoice : (list.includes('serena') ? 'serena' : (list[0] || 'serena'))
    }
    if (isXai) return XAI_VOICES.some(v => v.value === ttsVoice) ? ttsVoice : 'eve'
    if (isCosyvoice) {
      const list = herdsmanVoices.length > 0 ? herdsmanVoices : COSYVOICE_VOICES
      return list.includes(ttsVoice) ? ttsVoice : '中文女'
    }
    return ttsVoice || 'zh-CN-YunxiNeural'
  }, [isHerdsman, isXai, isCosyvoice, herdsmanVoices, ttsVoice])

  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 16 }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Text strong style={{ fontSize: 13, color: 'var(--color-text)' }}>语音设置</Text>
        <Button size="small" icon={<ReloadOutlined />} loading={checking} onClick={checkHealth}
          style={{ fontSize: 11 }}>检测</Button>
      </div>

      {/* 健康状态 */}
      {health && (
        <Card size="small" style={{ background: 'var(--color-surface-container)', border: '1px solid var(--color-border)', borderRadius: 8 }} bodyStyle={{ padding: 10 }}>
          <div style={{ display: 'flex', gap: 16, fontSize: 11 }}>
            <span>
              {health.asrReady ? <CheckCircleOutlined style={{ color: 'var(--color-success)' }} /> : <CloseCircleOutlined style={{ color: 'var(--color-destructive)' }} />}
              {' '}语音识别
            </span>
            <span>
              {health.ttsReady ? <CheckCircleOutlined style={{ color: 'var(--color-success)' }} /> : <CloseCircleOutlined style={{ color: 'var(--color-destructive)' }} />}
              {' '}语音合成
            </span>
            <span style={{ color: 'var(--color-text-secondary)' }}>{health.state || 'idle'}</span>
          </div>
        </Card>
      )}

      {/* TTS 开关 */}
      <Card size="small" style={{ background: 'var(--color-surface-container)', border: '1px solid var(--color-border)', borderRadius: 8 }} bodyStyle={{ padding: 12 }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div>
            <div style={{ fontSize: 12, color: 'var(--color-text-secondary)' }}>朗读回复</div>
            <div style={{ fontSize: 10, color: 'var(--color-text-secondary)', marginTop: 2, opacity: 0.75 }}>AI 回复时自动朗读</div>
          </div>
          <Switch size="small" checked={ttsEnabled} onChange={v => {
            setTtsEnabled(v)
            applyVoiceSettings({ ttsEnabled: v })
          }} />
        </div>
      </Card>

      {/* TTS 音色 */}
      <Card size="small" style={{ background: 'var(--color-surface-container)', border: '1px solid var(--color-border)', borderRadius: 8 }} bodyStyle={{ padding: 12 }}>
        <div style={{ fontSize: 12, color: 'var(--color-text-secondary)', marginBottom: 6 }}>合成音色</div>
        {isHerdsman && ttsModel && (
          <div style={{ fontSize: 10, color: '#8b5cf6', marginBottom: 6 }}> {/* hex-exempt 引擎品牌识别色（Herdsman） */}
            当前模型：{ttsModel}（Herdsman 音色，实时从服务端获取）
          </div>
        )}
        {isXai && ttsModel && (
          <div style={{ fontSize: 10, color: '#60a5fa', marginBottom: 6 }}> {/* hex-exempt 引擎品牌识别色（xAI） */}
            当前模型：{ttsModel}（xAI 云端 Grok 音色）
          </div>
        )}
        {isCosyvoice && ttsModel && (
          <div style={{ fontSize: 10, color: '#f472b6', marginBottom: 6 }}> {/* hex-exempt 引擎品牌识别色（CosyVoice） */}
            当前模型：{ttsModel}（本地 CosyVoice2，支持参考音频克隆音色）
          </div>
        )}
        <Select
          size="small"
          value={effectiveVoice}
          onChange={v => { setTtsVoice(v); applyVoiceSettings({ ttsVoice: v }) }}
          style={{ width: '100%' }}
          options={voiceOptions}
        />
      </Card>

      {/* 语音模式 */}
      <Card size="small" style={{ background: 'var(--color-surface-container)', border: '1px solid var(--color-border)', borderRadius: 8 }} bodyStyle={{ padding: 12 }}>
        <div style={{ fontSize: 12, color: 'var(--color-text-secondary)', marginBottom: 8 }}>输入模式</div>
        <div style={{ display: 'flex', gap: 8 }}>
          <Tag.CheckableTag
            checked={voiceMode === 'vad'}
            onChange={() => { setVoiceMode('vad'); applyVoiceSettings({ voiceMode: 'vad' }) }}
            style={{ fontSize: 12, padding: '4px 12px', borderRadius: 6 }}
          >
            <AudioOutlined style={{ marginRight: 4 }} /> 自动检测 (VAD)
          </Tag.CheckableTag>
          <Tag.CheckableTag
            checked={voiceMode === 'ptt'}
            onChange={() => { setVoiceMode('ptt'); applyVoiceSettings({ voiceMode: 'ptt' }) }}
            style={{ fontSize: 12, padding: '4px 12px', borderRadius: 6 }}
          >
            🎤 按住说话 (PTT)
          </Tag.CheckableTag>
        </div>
      </Card>

      {/* 打断阈值 */}
      <Card size="small" style={{ background: 'var(--color-surface-container)', border: '1px solid var(--color-border)', borderRadius: 8 }} bodyStyle={{ padding: 12 }}>
        <div style={{ fontSize: 12, color: 'var(--color-text-secondary)', marginBottom: 4 }}>
          打断阈值: {interruptMs}ms
        </div>
        <Slider
          min={100} max={2000} step={100}
          value={interruptMs}
          onChange={v => setInterruptMs(v as number)}
          onAfterChange={v => applyVoiceSettings({ interruptThresholdMs: v })}
          styles={{ track: { background: '#e85388' } }} // hex-exempt 品牌识别色（阈值轨道）
        />
      </Card>

      {/* 静默阈值 */}
      <Card size="small" style={{ background: 'var(--color-surface-container)', border: '1px solid var(--color-border)', borderRadius: 8 }} bodyStyle={{ padding: 12 }}>
        <div style={{ fontSize: 12, color: 'var(--color-text-secondary)', marginBottom: 4 }}>
          静默阈值: {silenceMs}ms
        </div>
        <Slider
          min={200} max={3000} step={100}
          value={silenceMs}
          onChange={v => setSilenceMs(v as number)}
          onAfterChange={v => applyVoiceSettings({ silenceThresholdMs: v })}
          styles={{ track: { background: '#a855f7' } }} // hex-exempt 品牌识别色（阈值轨道）
        />
      </Card>
    </div>
  )
}
