import React, { useCallback, useEffect, useState } from 'react'
import { Button, Select, Slider, Switch, Tag, Typography, message } from 'antd'
import { AudioOutlined, CheckCircleOutlined, CloseCircleOutlined, ReloadOutlined, SoundOutlined, ThunderboltOutlined } from '@ant-design/icons'
import { applyVoiceSettings, getVoiceSettings, voiceHealth } from '../../api/settings'
import SettingsSection from './SettingsSection'

const TTS_VOICES = [
  { value: 'zh-CN-YunxiNeural', label: '云希 (男)' },
  { value: 'zh-CN-XiaoxiaoNeural', label: '晓晓 (女)' },
  { value: 'zh-CN-YunjianNeural', label: '云健 (男)' },
  { value: 'zh-CN-XiaoyiNeural', label: '晓伊 (女)' },
]

/** VoicePanel — 语音与朗读设置（voiceManager 实时生效） */
const VoicePanel: React.FC = () => {
  const [health, setHealth] = useState<any>(null)
  const [checking, setChecking] = useState(false)
  const [cfg, setCfg] = useState<Record<string, any>>({})

  const load = useCallback(async () => {
    try {
      const [v, h] = await Promise.all([getVoiceSettings(), voiceHealth()])
      setCfg(v || {})
      setHealth(h)
    } catch { /* 语音未就绪静默 */ }
  }, [])

  useEffect(() => { load() }, [load])

  const checkHealth = async () => {
    setChecking(true)
    try { setHealth(await voiceHealth()) } catch {}
    setChecking(false)
  }

  const patch = (key: string, value: any) => {
    setCfg((prev) => ({ ...prev, [key]: value }))
    applyVoiceSettings({ [key]: value }).catch(() => message.warning('语音设置保存失败'))
  }

  const toggle = (key: string) => (v: boolean) => patch(key, v)

  return (
    <>
      <SettingsSection
        title={<>语音服务状态</>}
        desc="本地语音识别与合成服务健康检测（轻语对话使用）。"
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
          <span style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            {health?.asrReady
              ? <CheckCircleOutlined style={{ color: '#4ade80' }} />
              : <CloseCircleOutlined style={{ color: '#f87171' }} />}
            <Typography.Text style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)' }}>语音识别</Typography.Text>
          </span>
          <span style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            {health?.ttsReady
              ? <CheckCircleOutlined style={{ color: '#4ade80' }} />
              : <CloseCircleOutlined style={{ color: '#f87171' }} />}
            <Typography.Text style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)' }}>语音合成</Typography.Text>
          </span>
          <span style={{ flex: 1 }} />
          <Button size="small" icon={<ReloadOutlined />} loading={checking} onClick={checkHealth} style={{ fontSize: 11 }}>重新检测</Button>
        </div>
      </SettingsSection>

      <SettingsSection
        title={<>语音功能</>}
        desc="控制轻语对话中的语音交互行为，修改实时生效。"
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <div>
              <div style={{ fontSize: 13, color: 'var(--md-sys-color-text)' }}>语音对话</div>
              <div style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)' }}>启用语音输入与输出</div>
            </div>
            <Switch size="small" checked={!!cfg.enabled} onChange={toggle('enabled')} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <div>
              <div style={{ fontSize: 13, color: 'var(--md-sys-color-text)' }}>朗读回复</div>
              <div style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)' }}>AI 回复时自动朗读</div>
            </div>
            <Switch size="small" checked={!!cfg.ttsEnabled} onChange={toggle('ttsEnabled')} />
          </div>
        </div>
      </SettingsSection>

      <SettingsSection
        title={<>合成音色</>}
        desc="AI 语音合成的角色音色。"
      >
        <Select
          value={cfg.ttsVoice || undefined}
          placeholder="选择音色"
          onChange={(v) => patch('ttsVoice', v)}
          style={{ width: 220 }}
          options={TTS_VOICES}
        />
      </SettingsSection>

      <SettingsSection
        title={<>输入模式</>}
        desc="语音输入方式：自动检测说话或按住说话。"
      >
        <div style={{ display: 'flex', gap: 8 }}>
          <Tag.CheckableTag
            checked={cfg.voiceMode === 'vad'}
            onChange={() => patch('voiceMode', 'vad')}
            style={{ fontSize: 12, padding: '4px 14px', borderRadius: 8, border: '1px solid var(--md-sys-color-outline-variant)' }}
          >
            <AudioOutlined style={{ marginRight: 4 }} /> 自动检测 (VAD)
          </Tag.CheckableTag>
          <Tag.CheckableTag
            checked={cfg.voiceMode === 'ptt'}
            onChange={() => patch('voiceMode', 'ptt')}
            style={{ fontSize: 12, padding: '4px 14px', borderRadius: 8, border: '1px solid var(--md-sys-color-outline-variant)' }}
          >
            🎤 按住说话 (PTT)
          </Tag.CheckableTag>
        </div>
      </SettingsSection>

      <SettingsSection
        title={<>打断与静默阈值</>}
        desc="打断阈值：语音对话中多久未说话视为打断；静默阈值：结束一轮的静默时长。"
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
              <Typography.Text style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)' }}>打断阈值</Typography.Text>
              <Typography.Text style={{ fontSize: 12, color: 'var(--gaea-glow)' }}>{cfg.interruptThresholdMs ?? 500} ms</Typography.Text>
            </div>
            <Slider
              min={100} max={3000} step={100}
              value={cfg.interruptThresholdMs ?? 500}
              onChange={(v) => patch('interruptThresholdMs', v)}
              tooltip={{ open: false }}
            />
          </div>
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
              <Typography.Text style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)' }}>静默阈值</Typography.Text>
              <Typography.Text style={{ fontSize: 12, color: 'var(--gaea-glow)' }}>{cfg.silenceThresholdMs ?? 1000} ms</Typography.Text>
            </div>
            <Slider
              min={300} max={5000} step={100}
              value={cfg.silenceThresholdMs ?? 1000}
              onChange={(v) => patch('silenceThresholdMs', v)}
              tooltip={{ open: false }}
            />
          </div>
        </div>
      </SettingsSection>

      <SettingsSection
        title={<>语音引擎</>}
        desc="底层语音引擎信息。"
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <SoundOutlined style={{ color: 'var(--gaea-glow)' }} />
          <Typography.Text style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)' }}>
            引擎：{cfg.ttsEngine || 'auto'} · 识别模型：{cfg.asrModel || 'whisper-base'}
          </Typography.Text>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 8 }}>
          <ThunderboltOutlined style={{ color: 'var(--gaea-glow)' }} />
          <Typography.Text style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)' }}>
            详细语音与轻语人格设置请前往「轻语」模块
          </Typography.Text>
        </div>
      </SettingsSection>
    </>
  )
}

export default VoicePanel
