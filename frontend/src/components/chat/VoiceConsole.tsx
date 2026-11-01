import React, { useEffect, useMemo, useRef, useState } from 'react'
import { Select } from 'antd'
import {
  AimOutlined,
  AudioOutlined,
  CloseOutlined,
  MessageOutlined,
  NodeIndexOutlined,
  PauseCircleOutlined,
  SettingOutlined,
  SoundOutlined,
  StopOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons'
import VoiceChatOrb from '../VoiceChatOrb'
import { listMicDevices, loadMicDeviceId } from '../../hooks/useVoiceChat'
import type { VoiceChatState } from '../../hooks/useVoiceChat'
import type { ChatMsg } from '../../pages/chat/types'

export interface VoiceConsoleProps {
  voice: VoiceChatState
  voiceOn: boolean
  messages: ChatMsg[]
  companionName: string
  onToggleVoice: () => void | Promise<void | boolean>
  onInterruptVoice?: () => void
  onOpenSettings: () => void
  onClose: () => void
  /** 零信号自动换真麦：一键切换到探测到的有信号设备 */
  onSwitchMic?: (deviceId: string) => void
}

/** 专用语音工作台：大尺寸声核、实时转写、最近回答与明确的会话控制。 */
export const VoiceConsole: React.FC<VoiceConsoleProps> = ({
  voice,
  voiceOn,
  messages,
  companionName,
  onToggleVoice,
  onInterruptVoice,
  onOpenSettings,
  onClose,
  onSwitchMic,
}) => {
  const headingRef = useRef<HTMLHeadingElement>(null)
  const [busy, setBusy] = useState(false)
  const [sessionSeconds, setSessionSeconds] = useState(0)

  // 设备快切：当前录音设备一览+就地切换（switchMic 就地重启采集，会话不断）
  const [micDevices, setMicDevices] = useState<Array<{ deviceId: string; label: string }>>([])
  const [currentMic, setCurrentMic] = useState(() => loadMicDeviceId())
  useEffect(() => {
    void (async () => setMicDevices(await listMicDevices()))()
  }, [])

  useEffect(() => {
    headingRef.current?.focus()
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && !event.defaultPrevented) onClose()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [onClose])

  useEffect(() => {
    if (!voiceOn) {
      setSessionSeconds(0)
      return
    }
    const startedAt = Date.now()
    const timer = window.setInterval(() => {
      setSessionSeconds(Math.floor((Date.now() - startedAt) / 1000))
    }, 1000)
    return () => window.clearInterval(timer)
  }, [voiceOn])

  const stateLabel = voice.degraded === 'mic-unavailable'
    ? '麦克风不可用'
    : voice.aiSpeaking
      ? '正在回应'
      : voice.listening
        ? '正在聆听'
        : voiceOn
          ? '等待发言'
          : '语音已暂停'

  const lastReply = useMemo(() => {
    for (let index = messages.length - 1; index >= 0; index--) {
      if (messages[index]?.role === 'assistant') return (messages[index].content || '').trim()
    }
    return ''
  }, [messages])

  const toggle = async () => {
    if (busy) return
    setBusy(true)
    try {
      await onToggleVoice()
    } finally {
      setBusy(false)
    }
  }

  const volumePct = Math.max(0, Math.min(100, Math.round(voice.volume * 100)))
  const elapsed = `${String(Math.floor(sessionSeconds / 60)).padStart(2, '0')}:${String(sessionSeconds % 60).padStart(2, '0')}`
  const active = !voice.degraded && (voice.listening || voice.aiSpeaking)
  const systemState = voice.degraded
    ? 'DEGRADED'
    : voice.aiSpeaking
      ? 'RESPONDING'
      : voice.listening
        ? 'LISTENING'
        : voiceOn
          ? 'STANDBY'
          : 'PAUSED'
  const spectrum = Array.from({ length: 28 }, (_, index) => {
    const shape = 0.32 + Math.abs(Math.sin((index + 1) * 1.17)) * 0.68
    const energy = voice.aiSpeaking
      ? 0.34 + shape * 0.54
      : voice.listening
        ? 0.14 + Math.max(0.08, voice.volume) * shape * 0.9
        : 0.16 + shape * 0.12
    return Math.min(1, energy)
  })

  return (
    <section
      className={`chat-voice-console${active ? ' is-active' : ''}${voice.degraded ? ' is-degraded' : ''}`}
      aria-label="专门的语言交互界面"
      data-testid="chat-voice-console"
      data-voice-state={systemState.toLowerCase()}
      style={{ '--voice-energy': volumePct / 100 } as React.CSSProperties}
    >
      <div className="chat-voice-hud-frame" aria-hidden>
        <span className="chat-voice-hud-corner is-tl" />
        <span className="chat-voice-hud-corner is-tr" />
        <span className="chat-voice-hud-corner is-bl" />
        <span className="chat-voice-hud-corner is-br" />
        <span className="chat-voice-hud-scan" />
      </div>

      <div className="chat-voice-topbar">
        <div className="chat-voice-console-state" role="status" aria-live="polite">
          <span className={`chat-voice-state-dot${active ? ' is-live' : ''}`} aria-hidden />
          <span>{stateLabel}</span>
          <span className="chat-voice-state-separator" aria-hidden />
          <span>{voiceOn ? `会话 ${elapsed}` : '等待启动'}</span>
        </div>
        <div className="chat-voice-link-state" aria-hidden>
          <NodeIndexOutlined />
          <span>VOICE LINK</span>
          <strong>{voice.degraded ? 'OFFLINE' : voiceOn ? 'OPEN' : 'READY'}</strong>
        </div>
        <div className="chat-voice-signal" aria-label={`输入电平 ${volumePct}%`}>
          <SoundOutlined aria-hidden />
          <span>输入电平</span>
          <span className="chat-voice-signal-track" aria-hidden>
            <span style={{ width: `${volumePct}%` }} />
          </span>
          <strong>{voice.degraded ? '--' : volumePct}</strong>
        </div>
        <button type="button" className="chat-voice-close" onClick={onClose} aria-label="关闭语音对话">
          <CloseOutlined aria-hidden />
        </button>
      </div>

      <div className="chat-voice-stage">
        <div className={`chat-voice-visual${active ? ' is-active' : ''}`}>
          <div className="chat-voice-visual-readout is-top" aria-hidden>
            <span><AimOutlined /> SIGNAL FIELD</span>
            <strong>{systemState}</strong>
          </div>
          <span className="chat-voice-particle-field" aria-hidden />
          <span className="chat-voice-orbit-rail is-outer" aria-hidden />
          <span className="chat-voice-orbit-rail is-inner" aria-hidden />
          <span className="chat-voice-radar-sweep" aria-hidden />
          <div className="chat-voice-visual-glow" aria-hidden />
          <VoiceChatOrb
            volume={voice.volume}
            listening={voice.listening}
            speaking={voice.speaking}
            aiSpeaking={voice.aiSpeaking}
            transcript=""
            size={318}
            micUnavailable={voice.degraded === 'mic-unavailable'}
          />
          <div className="chat-voice-spectrum" aria-hidden>
            {spectrum.map((level, index) => (
              <span
                key={index}
                style={{
                  transform: `scaleY(${level})`,
                  animationDelay: `${index * 38}ms`,
                }}
              />
            ))}
          </div>
          <div className="chat-voice-core-label">
            <span>VOICE CORE</span>
            <strong>{active ? 'ACTIVE' : voice.degraded ? 'OFFLINE' : 'IDLE'}</strong>
          </div>
          <div className="chat-voice-visual-readout is-bottom" aria-hidden>
            <span><ThunderboltOutlined /> 16 kHz stream</span>
            <span>{elapsed}</span>
          </div>
        </div>

        <div className="chat-voice-copy">
          <div className="chat-voice-copy-heading">
            <span className="chat-voice-kicker">VOICE / {companionName.toUpperCase()}</span>
            <span className={`chat-voice-live-mark${active ? ' is-live' : ''}`}>
              {systemState}
            </span>
          </div>
          <div className="chat-voice-telemetry-strip" aria-hidden>
            <span><small>MODE</small><strong>CONTINUOUS</strong></span>
            <span><small>ENERGY</small><strong>{voice.degraded ? '--' : `${volumePct}%`}</strong></span>
            <span><small>SESSION</small><strong>{voiceOn ? elapsed : '00:00'}</strong></span>
          </div>
          <h2 ref={headingRef} tabIndex={-1}>与{companionName}语音对话</h2>
          <p className="chat-voice-guidance">
            {voice.degraded === 'mic-unavailable'
              ? '没有可用的麦克风。请返回文字对话，或先在语音设置中检查设备。'
              : voice.aiSpeaking
                ? '正在组织并播放回答，可以直接开口打断。'
                : voice.listening
                  ? '请自然说话，识别结果会实时显示在下方。'
                  : voiceOn
                    ? '语音通道已打开，请开始说话。'
                    : '点击开始聆听，进入连续语音对话。'}
          </p>
          {voice.degraded === 'mic-unavailable' && voice.suggestedMic && onSwitchMic && (
            <button
              type="button"
              className="chat-voice-device-recovery"
              onClick={() => onSwitchMic(voice.suggestedMic!.deviceId)}
            >
              <span>检测到可用麦克风</span>
              <strong>{voice.suggestedMic.label}</strong>
              <span>点此一键切换</span>
            </button>
          )}

          <div className="chat-voice-transcript" aria-live="polite">
            <div className="chat-voice-panel-label">
              <span>实时转写</span>
              <small>{voice.listening ? 'LIVE' : 'IDLE'}</small>
            </div>
            <p>{voice.transcript || (voice.listening ? '正在等待你的声音…' : '还没有新的语音输入')}</p>
          </div>

          {micDevices.length > 1 && onSwitchMic && (
            <label className="chat-voice-device">
              <AudioOutlined aria-hidden />
              <span>麦克风</span>
              <Select
                size="small"
                value={currentMic || 'default'}
                onChange={v => {
                  const id = v === 'default' ? '' : String(v)
                  setCurrentMic(id)
                  onSwitchMic(id) // switchMic 内含持久化+就地重启采集
                }}
                popupMatchSelectWidth={false}
                options={[
                  { value: 'default', label: '跟随系统默认' },
                  ...micDevices.map(d => ({ value: d.deviceId, label: d.label })),
                ]}
              />
            </label>
          )}

          {lastReply && (
            <div className="chat-voice-last-reply">
              <div className="chat-voice-panel-label">
                <span>最近回答</span>
                <small>AI</small>
              </div>
              <p>{lastReply.length > 240 ? `${lastReply.slice(0, 240)}…` : lastReply}</p>
            </div>
          )}

          {voice.degraded === 'mic-unavailable' && (
            <div role="alert" className="chat-voice-alert">
              麦克风不可用，未在采集声音。不会把静音状态显示成“正在聆听”。
            </div>
          )}
          {voice.error && (
            <div role="alert" className="chat-voice-alert">{voice.error}</div>
          )}
        </div>
      </div>

      <div className="chat-voice-controls">
        <span className="chat-voice-control-line" aria-hidden />
        <div className="chat-voice-control-primary">
          <button
            type="button"
            className={`chat-voice-primary${voiceOn ? ' is-stop' : ''}`}
            onClick={() => void toggle()}
            disabled={busy}
          >
            {voiceOn ? <StopOutlined aria-hidden /> : <AudioOutlined aria-hidden />}
            {busy ? '处理中…' : voiceOn ? '结束语音' : '开始聆听'}
          </button>
          {voice.aiSpeaking && onInterruptVoice && (
            <button type="button" className="chat-voice-interrupt" onClick={onInterruptVoice}>
              <PauseCircleOutlined aria-hidden /> 打断回答
            </button>
          )}
        </div>
        <div className="chat-voice-control-secondary">
          <button type="button" className="chat-voice-secondary" onClick={onOpenSettings}>
            <SettingOutlined aria-hidden /> 语音设置
          </button>
          <button type="button" className="chat-voice-secondary" onClick={onClose}>
            <MessageOutlined aria-hidden /> 返回文字对话
          </button>
        </div>
        <span className="chat-voice-control-line" aria-hidden />
      </div>

      <div className="chat-voice-footer-note">
        <span>SESSION LOG / 语音消息与文字会话共用同一记录</span>
        <kbd>Esc</kbd>
        <span>返回文字对话</span>
      </div>
    </section>
  )
}
