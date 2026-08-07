// VoiceSettingsPanel.tsx — 聊天语音设置面板
// 对齐 ackem VoiceSettings.tsx

import React, { useState, useEffect } from 'react'
import { Card, Switch, Select, Slider, Button, Typography, message, Tag } from 'antd'
import { SoundOutlined, AudioOutlined, ReloadOutlined, CheckCircleOutlined, CloseCircleOutlined } from '@ant-design/icons'
import * as App from '../../wailsjs/go/app/App'

const { Text } = Typography

const TTS_VOICES = [
  { value: 'zh-CN-YunxiNeural', label: '云希 (男)' },
  { value: 'zh-CN-XiaoxiaoNeural', label: '晓晓 (女)' },
  { value: 'zh-CN-YunjianNeural', label: '云健 (男)' },
  { value: 'zh-CN-XiaoyiNeural', label: '晓伊 (女)' },
]

export default function VoiceSettingsPanel() {
  const [health, setHealth] = useState<any>(null)
  const [checking, setChecking] = useState(false)

  // 语音设置
  const [ttsEnabled, setTtsEnabled] = useState(true)
  const [ttsVoice, setTtsVoice] = useState('zh-CN-YunxiNeural')
  const [voiceMode, setVoiceMode] = useState<'vad' | 'ptt'>('vad')
  const [interruptMs, setInterruptMs] = useState(500)
  const [silenceMs, setSilenceMs] = useState(1000)

  const checkHealth = async () => {
    setChecking(true)
    try {
      const h = await (App as any).VoiceHealth?.()
      setHealth(h || { asrReady: false, ttsReady: false })
    } catch (_) {
      setHealth({ asrReady: false, ttsReady: false, error: '无法获取语音服务状态' })
    }
    setChecking(false)
  }

  useEffect(() => { checkHealth() }, [])

  const applyVoiceSettings = (patch: Record<string, any>) => {
    (App as any).VoiceApplySettings?.(patch).catch(() => {})
  }

  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 16 }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Text strong style={{ fontSize: 13, color: '#d4d4d8' }}>语音设置</Text>
        <Button size="small" icon={<ReloadOutlined />} loading={checking} onClick={checkHealth}
          style={{ fontSize: 11 }}>检测</Button>
      </div>

      {/* 健康状态 */}
      {health && (
        <Card size="small" style={{ background: '#18181b', border: '1px solid #27272a', borderRadius: 8 }} bodyStyle={{ padding: 10 }}>
          <div style={{ display: 'flex', gap: 16, fontSize: 11 }}>
            <span>
              {health.asrReady ? <CheckCircleOutlined style={{ color: '#4ade80' }} /> : <CloseCircleOutlined style={{ color: '#f87171' }} />}
              {' '}语音识别
            </span>
            <span>
              {health.ttsReady ? <CheckCircleOutlined style={{ color: '#4ade80' }} /> : <CloseCircleOutlined style={{ color: '#f87171' }} />}
              {' '}语音合成
            </span>
            <span style={{ color: '#71717a' }}>{health.state || 'idle'}</span>
          </div>
        </Card>
      )}

      {/* TTS 开关 */}
      <Card size="small" style={{ background: '#18181b', border: '1px solid #27272a', borderRadius: 8 }} bodyStyle={{ padding: 12 }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div>
            <div style={{ fontSize: 12, color: '#a1a1aa' }}>朗读回复</div>
            <div style={{ fontSize: 10, color: '#52525b', marginTop: 2 }}>AI 回复时自动朗读</div>
          </div>
          <Switch size="small" checked={ttsEnabled} onChange={v => {
            setTtsEnabled(v)
            applyVoiceSettings({ ttsEnabled: v })
          }} />
        </div>
      </Card>

      {/* TTS 音色 */}
      <Card size="small" style={{ background: '#18181b', border: '1px solid #27272a', borderRadius: 8 }} bodyStyle={{ padding: 12 }}>
        <div style={{ fontSize: 12, color: '#a1a1aa', marginBottom: 6 }}>合成音色</div>
        <Select
          size="small"
          value={ttsVoice}
          onChange={v => { setTtsVoice(v); applyVoiceSettings({ ttsVoice: v }) }}
          style={{ width: '100%' }}
          options={TTS_VOICES}
        />
      </Card>

      {/* 语音模式 */}
      <Card size="small" style={{ background: '#18181b', border: '1px solid #27272a', borderRadius: 8 }} bodyStyle={{ padding: 12 }}>
        <div style={{ fontSize: 12, color: '#a1a1aa', marginBottom: 8 }}>输入模式</div>
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
      <Card size="small" style={{ background: '#18181b', border: '1px solid #27272a', borderRadius: 8 }} bodyStyle={{ padding: 12 }}>
        <div style={{ fontSize: 12, color: '#a1a1aa', marginBottom: 4 }}>
          打断阈值: {interruptMs}ms
        </div>
        <Slider
          min={100} max={2000} step={100}
          value={interruptMs}
          onChange={v => setInterruptMs(v as number)}
          onAfterChange={v => applyVoiceSettings({ interruptThresholdMs: v })}
          styles={{ track: { background: '#e85388' } }}
        />
      </Card>

      {/* 静默阈值 */}
      <Card size="small" style={{ background: '#18181b', border: '1px solid #27272a', borderRadius: 8 }} bodyStyle={{ padding: 12 }}>
        <div style={{ fontSize: 12, color: '#a1a1aa', marginBottom: 4 }}>
          静默阈值: {silenceMs}ms
        </div>
        <Slider
          min={200} max={3000} step={100}
          value={silenceMs}
          onChange={v => setSilenceMs(v as number)}
          onAfterChange={v => applyVoiceSettings({ silenceThresholdMs: v })}
          styles={{ track: { background: '#a855f7' } }}
        />
      </Card>
    </div>
  )
}
