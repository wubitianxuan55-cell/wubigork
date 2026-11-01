/**
 * voice/console/VoiceConsoleV2.tsx — 语音台 V2 实时会话视图（M4 产品接入刀）。
 *
 * voice.backend=v2 时由 ChatPage 渲染（V1 VoiceConsole 整体保留不动）。
 * UI 由协议相位直接驱动（V2 方案 §10.3）；轮完成写回当前聊天话题
 * （useChatVoice 同款纪律：会话绑开启时话题，迟到轮不串台）。
 */
import React, { useCallback, useEffect, useRef } from 'react'
import { Button, Tooltip } from 'antd'
import { CloseOutlined, SettingOutlined, AudioOutlined, PauseOutlined, StopOutlined } from '@ant-design/icons'
import { useVoiceChatV2 } from '../../voice/runtime/useVoiceChatV2'
import { app } from '../../gaea/lib/bridge'
import { nextMsgKey, nowStr } from '../../pages/chat/utils'
import type { ChatMsg } from '../../pages/chat/types'
import '../../voice/dev/voiceV2Lab.css'

export interface VoiceConsoleV2Props {
  companionName: string
  instructions: string
  /** 人格音色（breeze 克隆 id；空=服务端缺省）——音色漂移根修 */
  voiceId?: string
  onClose: () => void
  onOpenSettings: () => void
  setMessages: React.Dispatch<React.SetStateAction<ChatMsg[]>>
  getActiveId: () => string
  finalizeTopicAfterSend?: (topicId: string, firstUserText: string) => Promise<void>
}

const PHASE_LABEL: Record<string, string> = {
  idle: '空闲',
  listening: '聆听中',
  thinking: '思考中',
  speaking: '播报中',
  interrupted: '被打断',
  error: '错误',
}

const VoiceConsoleV2: React.FC<VoiceConsoleV2Props> = ({
  companionName, instructions, voiceId, onClose, onOpenSettings, setMessages, getActiveId, finalizeTopicAfterSend,
}) => {
  // 会话话题绑定：开启时锁定；切话题/关闭后迟到轮不写新会话（V1 同款纪律）
  const sessionTopicRef = useRef(getActiveId())

  const onTurnComplete = useCallback((turn: { user: string; assistant: string }) => {
    const topicID = sessionTopicRef.current
    if (!topicID || getActiveId() !== topicID) return
    const userKey = nextMsgKey()
    const asstKey = nextMsgKey()
    setMessages((prev) => [...prev,
      { key: userKey, role: 'user', content: turn.user, createdAt: nowStr() },
      { key: asstKey, role: 'assistant', content: turn.assistant, createdAt: nowStr() },
    ])
    void app.ChatAppendMessages(topicID, [
      { role: 'user', content: turn.user, extra: '' },
      { role: 'assistant', content: turn.assistant, extra: '' },
    ]).then(() => {
      void finalizeTopicAfterSend?.(topicID, turn.user)
    }).catch(() => {
      /* 落库失败不静默但不打断会话：V1 侧 logFrontendError；此处 UI 内可见于日志尾 */
    })
  }, [finalizeTopicAfterSend, getActiveId, setMessages])

  const v2 = useVoiceChatV2({ instructions, voice: voiceId, onTurnComplete })

  useEffect(() => {
    void v2.connect()
    return () => v2.disconnect()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const ready = v2.services.ready

  return (
    <div className="voice-console-v2" data-testid="voice-console-v2">
      <header className="v2l-head">
        <div className="vc2-title">
          <span className="vc2-companion">{companionName}</span>
          <span className="vc2-badge">V2 实时</span>
          {!ready && <span className="vc2-warn" data-testid="vc2-warn">sidecar 未就绪</span>}
        </div>
        <div className="vc2-actions">
          <Tooltip title="语音设置">
            <Button type="text" icon={<SettingOutlined />} onClick={onOpenSettings} aria-label="语音设置" />
          </Tooltip>
          <Tooltip title="关闭语音台">
            <Button type="text" icon={<CloseOutlined />} onClick={onClose} aria-label="关闭语音台" />
          </Tooltip>
        </div>
      </header>

      <div className="v2l-row">
        {!ready && (
          <Button className="v2l-btn" onClick={() => void v2.ensureServices()} data-testid="vc2-ensure">
            拉起依赖链
          </Button>
        )}
        {v2.connected ? (
          <>
            <Button
              className="v2l-btn"
              icon={v2.micOn ? <PauseOutlined /> : <AudioOutlined />}
              onClick={() => (v2.micOn ? v2.stopMic() : void v2.startMic())}
              data-testid="vc2-mic"
            >
              {v2.micOn ? '关麦' : '开麦'}
            </Button>
            <Button className="v2l-btn" icon={<StopOutlined />} onClick={v2.cancel} data-testid="vc2-cancel">
              打断
            </Button>
          </>
        ) : (
          <Button className="v2l-btn" onClick={() => void v2.connect()} data-testid="vc2-connect">连接</Button>
        )}
        {v2.note && <span className="v2l-note">{v2.note}</span>}
      </div>

      <div className="v2l-cols">
        <section className="v2l-sec">
          <h4>用户转写</h4>
          <div className="v2l-scroll" data-testid="vc2-transcripts">
            {v2.transcripts.map((t, i) => (
              <p key={i} className="v2l-user">{t}</p>
            ))}
            {v2.partialTranscript && <p className="v2l-user v2l-partial">{v2.partialTranscript}…</p>}
            {v2.transcripts.length === 0 && !v2.partialTranscript && (
              <p className="v2l-partial">{v2.micOn ? '说话吧…' : '开麦开始对话'}</p>
            )}
          </div>
        </section>
        <section className="v2l-sec">
          <h4>{companionName}</h4>
          <div className="v2l-scroll">
            <p className="v2l-assistant" data-testid="vc2-assistant">{v2.assistantText || '—'}</p>
          </div>
        </section>
      </div>

      <footer className="v2l-foot">
        <span className={`v2l-phase v2l-phase-${v2.phase}`} data-testid="vc2-phase">
          {PHASE_LABEL[v2.phase] ?? v2.phase}
        </span>
        <span className="v2l-pb">{v2.pbText || '播放队列：—'}</span>
        <div className="v2l-tail" data-testid="vc2-tail">
          {v2.tail.map((l, i) => (
            <span key={i}>{l}</span>
          ))}
        </div>
      </footer>
    </div>
  )
}

export default VoiceConsoleV2
