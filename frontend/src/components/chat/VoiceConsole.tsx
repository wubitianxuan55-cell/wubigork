import React, { useEffect, useMemo, useRef, useState } from 'react'
import {
  AudioOutlined,
  CloseOutlined,
  MessageOutlined,
  PauseCircleOutlined,
  SettingOutlined,
  StopOutlined,
} from '@ant-design/icons'
import VoiceChatOrb from '../VoiceChatOrb'
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
}) => {
  const headingRef = useRef<HTMLHeadingElement>(null)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    headingRef.current?.focus()
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && !event.defaultPrevented) onClose()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [onClose])

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

  return (
    <section className="chat-voice-console" aria-label="专门的语言交互界面" data-testid="chat-voice-console">
      <div className="chat-voice-console-state" role="status" aria-live="polite">
        <span className={`chat-voice-state-dot${!voice.degraded && (voice.listening || voice.aiSpeaking) ? ' is-live' : ''}`} aria-hidden />
        <span>{stateLabel}</span>
        <span className="chat-voice-state-separator" aria-hidden />
        <span>{voiceOn ? '语音会话进行中' : '等待启动'}</span>
      </div>

      <div className="chat-voice-stage">
        <div className={`chat-voice-visual${!voice.degraded && (voice.listening || voice.aiSpeaking) ? ' is-active' : ''}`}>
          <div className="chat-voice-visual-glow" aria-hidden />
          <VoiceChatOrb
            volume={voice.volume}
            listening={voice.listening}
            speaking={voice.speaking}
            aiSpeaking={voice.aiSpeaking}
            transcript=""
            size={248}
            micUnavailable={voice.degraded === 'mic-unavailable'}
          />
          <div className="chat-voice-wave" aria-hidden>
            {Array.from({ length: 9 }).map((_, index) => (
              <span key={index} style={{ animationDelay: `${index * 70}ms` }} />
            ))}
          </div>
        </div>

        <div className="chat-voice-copy">
          <span className="chat-voice-kicker">VOICE SESSION</span>
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

          <div className="chat-voice-transcript" aria-live="polite">
            <span>实时转写</span>
            <p>{voice.transcript || (voice.listening ? '正在等待你的声音…' : '还没有新的语音输入')}</p>
          </div>

          {lastReply && (
            <div className="chat-voice-last-reply">
              <span>最近回答</span>
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
          <button type="button" className="chat-voice-secondary" onClick={onInterruptVoice}>
            <PauseCircleOutlined aria-hidden /> 打断回答
          </button>
        )}
        <button type="button" className="chat-voice-secondary" onClick={onOpenSettings}>
          <SettingOutlined aria-hidden /> 语音设置
        </button>
        <button type="button" className="chat-voice-secondary" onClick={onClose}>
          <MessageOutlined aria-hidden /> 返回文字对话
        </button>
        <button type="button" className="chat-voice-close" onClick={onClose} aria-label="关闭语音对话">
          <CloseOutlined aria-hidden />
        </button>
      </div>

      <p className="chat-voice-shortcuts">Esc 返回文字对话 · 语音消息与文字会话共用同一记录</p>
    </section>
  )
}
