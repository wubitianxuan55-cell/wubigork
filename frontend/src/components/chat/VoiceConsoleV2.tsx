/**
 * voice/console/VoiceConsoleV2.tsx — 语音台 V2 实时会话视图（M4 产品接入刀）。
 *
 * voice.backend=v2 时由 ChatPage 渲染（V1 VoiceConsole 整体保留不动）。
 * UI 由协议相位直接驱动（V2 方案 §10.3）；轮完成写回当前聊天话题
 * （useChatVoice 同款纪律：会话绑开启时话题，迟到轮不串台）。
 */
import React, { useCallback, useEffect, useRef, useState } from 'react'
import { Button, message, Tooltip } from 'antd'
import {
  AudioOutlined,
  CheckCircleFilled,
  CloseOutlined,
  DisconnectOutlined,
  ExpandOutlined,
  LoadingOutlined,
  MinusCircleOutlined,
  PauseOutlined,
  SettingOutlined,
  ShrinkOutlined,
  SoundOutlined,
  StopOutlined,
  ThunderboltOutlined,
  WarningOutlined,
  WifiOutlined,
} from '@ant-design/icons'
import { useVoiceChatV2 } from '../../voice/runtime/useVoiceChatV2'
import { app } from '../../gaea/lib/bridge'
import { nextMsgKey, nowStr } from '../../pages/chat/utils'
import type { ChatMsg } from '../../pages/chat/types'
import { VoiceV2Orb } from '../../voice/components/VoiceV2Orb'
import '../../voice/voiceV2.css'

export interface VoiceConsoleV2Props {
  companionName: string
  instructions: string
  /** 人格音色（breeze 克隆 id；空=服务端缺省）——音色漂移根修 */
  voiceId?: string
  /** P1.5 内嵌形态：bar=输入区内嵌语音条（默认入口，可展开浮层）；
   * full=传统整幅控制台（默认，行为不变）。同一实例同一条会话线，不双开麦。 */
  variant?: 'full' | 'bar'
  /** 会话头恒显绑定：当前话题标题（开台时捕获）。 */
  topicTitle?: string
  onClose: () => void
  onOpenSettings: () => void
  setMessages: React.Dispatch<React.SetStateAction<ChatMsg[]>>
  getActiveId: () => string
  finalizeTopicAfterSend?: (topicId: string, firstUserText: string) => Promise<void>
}

const PHASE_META = {
  idle: {
    label: '待机',
    hint: '准备好后开始说话',
    icon: <AudioOutlined />,
  },
  listening: {
    label: '正在聆听',
    hint: '我在听，请直接说',
    icon: <AudioOutlined />,
  },
  thinking: {
    label: '正在思考',
    hint: '正在整理这一轮回答',
    icon: <LoadingOutlined spin />,
  },
  speaking: {
    label: '正在回应',
    hint: '回应中，开口即可打断',
    icon: <SoundOutlined />,
  },
  interrupted: {
    label: '已打断',
    hint: '上一轮已停止，可以继续说',
    icon: <StopOutlined />,
  },
  error: {
    label: '本轮异常',
    hint: '语音没有完成，可以重新开始',
    icon: <WarningOutlined />,
  },
}

const SERVICE_ITEMS = [
  { key: 'sidecar', label: '语音引擎' },
  { key: 'llm', label: '快速模型' },
  { key: 'breeze', label: '声线服务' },
  { key: 'gateway', label: '记忆网关' },
] as const

const VoiceConsoleV2: React.FC<VoiceConsoleV2Props> = ({
  companionName, instructions, voiceId, variant = 'full', topicTitle,
  onClose, onOpenSettings, setMessages, getActiveId, finalizeTopicAfterSend,
}) => {
  // 会话话题绑定：开启时锁定；切话题/关闭后迟到轮不写新会话（V1 同款纪律）
  const sessionTopicRef = useRef(getActiveId())
  const mountedRef = useRef(true)
  const commitChainRef = useRef<Promise<void>>(Promise.resolve())
  const [turnWarning, setTurnWarning] = useState('')
  // P1.9 异步动作预览 + P2 确认执行：轮完成后对最终转写跑 VoiceV2PreviewIntent
  // （dry-run 零副作用、不打 LLM；命中且白名单内发 previewId/intentHash/TTL 凭据；
  // 失败=未识别为动作，静默）。执行走 VoiceV2ExecuteAction（幂等/TTL/hash 校验，
  // 确认后最多执行被预览动作一次），回执短暂显示后自动淡出。
  const [actionPreview, setActionPreview] = useState<{
    reply: string; action: string; previewId?: string; intentHash?: string; executable?: boolean
  } | null>(null)
  const [actionResult, setActionResult] = useState('')
  const actionTimerRef = useRef<number | null>(null)
  const flashActionResult = useCallback((text: string) => {
    setActionResult(text)
    if (actionTimerRef.current != null) window.clearTimeout(actionTimerRef.current)
    actionTimerRef.current = window.setTimeout(() => {
      actionTimerRef.current = null
      if (mountedRef.current) setActionResult('')
    }, 6000)
  }, [])
  // P1.5：bar 形态下的展开态（浮层整幅控制台）；full 形态恒整幅
  const [expanded, setExpanded] = useState(false)
  const mode: 'full' | 'float' | 'bar' = variant === 'bar' ? (expanded ? 'float' : 'bar') : 'full'

  const executePreview = useCallback(() => {
    setActionPreview((cur) => {
      if (!cur?.previewId || !cur.intentHash) return cur
      void app.VoiceV2ExecuteAction(cur.previewId, cur.intentHash)
        .then((r) => {
          if (!mountedRef.current) return
          if (r?.executed === true) {
            setActionPreview(null)
            flashActionResult(`${r.reply ?? '已执行'}`)
          } else {
            setActionPreview(null)
            flashActionResult(`动作未能执行（${String(r?.reason ?? 'unknown')}）`)
          }
        })
        .catch(() => {
          if (mountedRef.current) flashActionResult('动作执行调用失败')
        })
      return null // 领走即清卡（执行中不重复点击）
    })
  }, [flashActionResult])

  const onTurnComplete = useCallback((turn: {
    user: string; assistant: string; responseId: string; itemId: string
  }) => {
    const topicID = sessionTopicRef.current
    if (!topicID) return
    // P0a-A5e（众神殿定稿 + 用户拍板「继续提交」）：已完成回合的提交不依赖
    // 组件挂载与当前视图——入链后始终提交到捕获 topic（后端 response/item
    // 幂等兜底）；UI 增量仅在仍挂载且未切题时渲染，防写进当前展示的其它话题。
    // 串行提交确保多轮完成的聊天顺序与语音发生顺序一致。
    const job = commitChainRef.current.then(async () => {
      let result: Record<string, unknown>
      try {
        result = await app.VoiceV2CommitTurn(
          topicID, turn.user, turn.assistant, turn.responseId, turn.itemId,
        )
      } catch (err: unknown) {
        if (!mountedRef.current || getActiveId() !== topicID) return
        const text = `本轮保存失败：${err instanceof Error ? err.message : String(err)}`
        setTurnWarning(text)
        message.error(text)
        return
      }
      if (!mountedRef.current || getActiveId() !== topicID) return
      if (result.duplicate !== true) {
        const userKey = nextMsgKey()
        const asstKey = nextMsgKey()
        setMessages((prev) => [...prev,
          { key: userKey, role: 'user', content: turn.user, createdAt: nowStr() },
          { key: asstKey, role: 'assistant', content: turn.assistant, createdAt: nowStr() },
        ])
      }
      const warning = typeof result.warning === 'string' ? result.warning : ''
      setTurnWarning(warning)
      if (warning) message.warning(warning)
      void finalizeTopicAfterSend?.(topicID, turn.user)
    })
    commitChainRef.current = job
    void job
    // P1.9/P2：dry-run 预览不进提交链（绝不阻塞对话落库）；新轮覆盖旧预览。
    // 经微任务调用：桥桩缺方法时同步抛错也能落进 catch（静默）。
    void Promise.resolve()
      .then(() => app.VoiceV2PreviewIntent(turn.user))
      .then((r) => {
        if (!mountedRef.current || !r?.handled || !r.reply) return
        setActionResult('')
        setActionPreview({
          reply: String(r.reply),
          action: String(r.action ?? ''),
          previewId: typeof r.previewId === 'string' ? r.previewId : undefined,
          intentHash: typeof r.intentHash === 'string' ? r.intentHash : undefined,
          executable: r.executable === true,
        })
      })
      .catch(() => {})
  }, [finalizeTopicAfterSend, getActiveId, setMessages])

  const v2 = useVoiceChatV2({ instructions, voice: voiceId, onTurnComplete })

  useEffect(() => {
    mountedRef.current = true
    let alive = true
    // 依赖链未就绪时先不建立注定失败的 WebSocket；拉起后由同一按钮续连。
    void v2.refreshServices().then((ready) => {
      if (alive && ready) void v2.connect()
    })
    return () => {
      mountedRef.current = false
      alive = false
      // P0a-A5f：生成中关闭=先精确 abort 后端预推进（已完成轮已在提交链，
      // 不受影响；用户拍板「继续提交」），再断开传输。
      v2.abortInFlight()
      v2.disconnect()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // 浮层键盘可达：Escape 收起回条（会话保持；ui-ux-pro-max 键盘导航优先级 1）
  useEffect(() => {
    if (mode !== 'float') return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setExpanded(false)
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [mode])

  const ready = v2.services.ready
  const phaseMeta = PHASE_META[v2.phase]
  const feedback = turnWarning || v2.note
  const hasConversation = v2.transcripts.length > 0 || Boolean(v2.partialTranscript) || Boolean(v2.assistantText)
  const PrimaryIcon = !ready
    ? v2.ensuring ? LoadingOutlined : AudioOutlined
    : !v2.connected ? WifiOutlined : v2.micOn ? PauseOutlined : AudioOutlined
  const primaryLabel = !ready
    ? v2.ensuring ? '正在准备本地语音' : '启动本地语音'
    : !v2.connected
      ? '连接语音'
      : v2.micOn ? '暂停聆听' : '开始聆听'

  const runPrimaryAction = () => {
    if (!ready) {
      void v2.ensureServices().then((ok) => {
        if (ok) void v2.connect()
      })
      return
    }
    if (!v2.connected) {
      void v2.connect()
      return
    }
    if (v2.micOn) v2.stopMic()
    else void v2.startMic()
  }

  // ── P1.5 内嵌语音条（输入区一等入口；展开=浮层整幅，同一会话线不重挂） ──
  if (mode === 'bar') {
    const preview = v2.partialTranscript
      || v2.assistantText
      || (v2.transcripts.length > 0 ? v2.transcripts[v2.transcripts.length - 1] : '')
    const onlineCount = SERVICE_ITEMS.filter((item) => v2.services[item.key]).length
    return (
      <div
        className={`voice-console-v2 voice-console-v2--bar voice-console-v2--${v2.phase}${ready ? '' : ' is-preflight'}`}
        data-testid="voice-console-v2"
        data-phase={v2.phase}
      >
        <span className={`vc2-bar-dot is-${v2.phase}`} aria-hidden="true" />
        {/* 原子化状态播报（ui-ux-pro-max ux 规则）：只报相位一处，转写预览不做
            竞争性 live region（逐字流式会刷屏读屏） */}
        <span className="vc2-bar-phase" role="status" aria-atomic="true">{phaseMeta.label}</span>
        <span className="vc2-bar-preview" aria-hidden="true">
          {preview ? preview.slice(-80) : (ready ? '按下开始说话' : '本地语音待启动')}
        </span>
        <span className="vc2-bar-binding" title="语音会话绑定（开台时捕获）">
          {companionName} · {topicTitle || '当前会话'}
        </span>
        <span
          className="vc2-bar-health"
          role="img"
          aria-label={`语音服务 ${onlineCount}/${SERVICE_ITEMS.length} 就绪`}
        >
          {SERVICE_ITEMS.map((item) => (
            <i
              key={item.key}
              className={v2.services[item.key] ? 'is-online' : ''}
              title={`${item.label}${v2.services[item.key] ? '已就绪' : '未启动'}`}
            />
          ))}
        </span>
        <Button
          size="small"
          type="primary"
          className="vc2-bar-btn is-primary"
          icon={<PrimaryIcon />}
          loading={v2.ensuring}
          onClick={runPrimaryAction}
          data-testid={v2.connected ? 'vc2-bar-mic' : 'vc2-bar-start'}
        >
          {primaryLabel.replace('本地', '')}
        </Button>
        {v2.connected && (
          <Button size="small" className="vc2-bar-btn" icon={<StopOutlined />} onClick={v2.cancel} data-testid="vc2-bar-cancel">
            打断
          </Button>
        )}
        <Button size="small" className="vc2-bar-btn" icon={<ExpandOutlined />} onClick={() => setExpanded(true)} data-testid="vc2-bar-expand">
          展开
        </Button>
        <Button size="small" className="vc2-bar-btn is-danger" icon={<CloseOutlined />} onClick={onClose} data-testid="vc2-bar-close" aria-label="结束语音会话">
          结束
        </Button>
        {feedback && <p className="vc2-bar-note" role="status">{feedback}</p>}
        {actionPreview && (
          <div className="vc2-action-preview" role="note">
            <ThunderboltOutlined aria-hidden="true" />
            <span className="vc2-action-preview-text">{actionPreview.reply}</span>
            {actionPreview.executable && (
              <Button size="small" type="primary" className="vc2-action-preview-exec" onClick={executePreview} data-testid="vc2-action-exec">
                执行
              </Button>
            )}
            <span className="vc2-action-preview-tag">
              {actionPreview.executable ? '识别到动作 · 待确认' : '动作暂不支持语音执行'}
            </span>
            <button className="vc2-action-preview-close" aria-label="关闭动作预览" onClick={() => setActionPreview(null)}>
              <CloseOutlined />
            </button>
          </div>
        )}
        {actionResult && <p className="vc2-bar-note" role="status">{actionResult}</p>}
      </div>
    )
  }

  return (
    <div
      className={`voice-console-v2${mode === 'float' ? ' voice-console-v2--float' : ''} voice-console-v2--${v2.phase}${ready ? '' : ' is-preflight'}`}
      data-testid="voice-console-v2"
      data-phase={v2.phase}
    >
      <header className="vc2-header">
        <div className="vc2-identity">
          <span className="vc2-brand-sigil" aria-hidden="true">
            <span />
          </span>
          <div>
            <span className="vc2-eyebrow">VOICE SESSION</span>
            <div className="vc2-title-row">
              <h2>{companionName}</h2>
              <span className="vc2-badge">实时语音</span>
            </div>
            <p>本地实时对话 · 声音与记忆留在这台设备</p>
          </div>
        </div>
        <div className="vc2-header-actions">
          <span
            className={`vc2-connection-pill${v2.connected ? ' is-live' : ready ? ' is-ready' : ' is-waiting'}`}
            data-testid="vc2-warn"
          >
            {!ready ? <LoadingOutlined spin /> : v2.connected ? <WifiOutlined /> : <DisconnectOutlined />}
            {!ready ? '依赖待准备' : v2.connected ? '已连接' : '可连接'}
          </span>
          <div className="vc2-actions">
            {variant === 'bar' && (
              <Tooltip title="收起到输入区语音条">
                <Button type="text" icon={<ShrinkOutlined />} onClick={() => setExpanded(false)} aria-label="收起语音台" />
              </Tooltip>
            )}
            <Tooltip title="语音设置">
              <Button type="text" icon={<SettingOutlined />} onClick={onOpenSettings} aria-label="语音设置" />
            </Tooltip>
            <Tooltip title="关闭语音台">
              <Button type="text" icon={<CloseOutlined />} onClick={onClose} aria-label="关闭语音台" />
            </Tooltip>
          </div>
        </div>
      </header>

      {!ready ? (
        <section className="vc2-preflight" aria-labelledby="vc2-preflight-title">
          <div className="vc2-preflight-copy">
            <span className="vc2-eyebrow">FIRST RUN</span>
            <h3 id="vc2-preflight-title">准备本地语音</h3>
            <p>首次启动会加载语音识别、快速模型与声线服务，通常需要几十秒。</p>
          </div>
          <div className="vc2-service-list" aria-label="语音服务状态">
            {SERVICE_ITEMS.map((item) => {
              const online = v2.services[item.key]
              return (
                <div className={`vc2-service-item${online ? ' is-online' : ''}`} key={item.key}>
                  {online ? <CheckCircleFilled /> : <MinusCircleOutlined />}
                  <span>{item.label}</span>
                  <span className="vc2-service-state">{online ? '已就绪' : '未启动'}</span>
                </div>
              )
            })}
          </div>
          <Button
            type="primary"
            size="large"
            className="vc2-primary-button"
            icon={<PrimaryIcon />}
            loading={v2.ensuring}
            disabled={v2.ensuring}
            onClick={runPrimaryAction}
            data-testid="vc2-ensure"
          >
            {primaryLabel}
          </Button>
          {feedback && (
            <p className="vc2-feedback" role="alert" data-testid="vc2-note">{feedback}</p>
          )}
        </section>
      ) : (
        <main className="vc2-layout">
          <section className="vc2-stage" aria-label="语音状态">
            <header className="vc2-section-head">
              <div>
                <span className="vc2-eyebrow">VOICE FIELD</span>
                <h3>实时声场</h3>
              </div>
              <span className={`vc2-live-marker${v2.connected ? ' is-live' : ''}`}>
                <span aria-hidden="true" />
                {v2.connected ? 'LIVE' : 'READY'}
              </span>
            </header>

            <div className="vc2-field">
              <VoiceV2Orb phase={v2.phase} active={v2.micOn || v2.phase === 'speaking'} connected={v2.connected} />
              <div className="vc2-stage-copy" role="status" aria-live="polite">
                <span className={`vc2-phase vc2-phase-${v2.phase}`} data-testid="vc2-phase">
                  {phaseMeta.icon}
                  {phaseMeta.label}
                </span>
                <p>{v2.connected ? phaseMeta.hint : '连接后即可开始实时语音对话'}</p>
              </div>
              <div className="vc2-wave-field" aria-hidden="true">
                {Array.from({ length: 24 }, (_, index) => (
                  <i key={index} style={{ '--vc2-wave': 0.18 + ((index * 7) % 13) / 16 } as React.CSSProperties} />
                ))}
              </div>
            </div>

            <footer className="vc2-stage-facts">
              <span><AudioOutlined /> 麦克风{v2.micOn ? '开启' : '关闭'}</span>
              <span><CheckCircleFilled /> 本机处理</span>
              <span>{v2.connected ? '实时连接' : '等待连接'}</span>
            </footer>
          </section>

          <aside className="vc2-dialogue" aria-labelledby="vc2-dialogue-title">
            <header className="vc2-section-head">
              <div>
                <span className="vc2-eyebrow">LIVE TRANSCRIPT</span>
                <h3 id="vc2-dialogue-title">实时对话</h3>
              </div>
              <span className="vc2-turn-count">{v2.transcripts.length} 轮</span>
            </header>

            <div
              className="vc2-thread"
              data-testid="vc2-transcripts"
              role="log"
              aria-live="polite"
              aria-busy={v2.phase === 'thinking'}
            >
              {v2.transcripts.map((transcript, index) => (
                <article className="vc2-message is-user" key={`${transcript}-${index}`}>
                  <span className="vc2-message-label">你</span>
                  <p>{transcript}</p>
                </article>
              ))}
              {v2.partialTranscript && (
                <article className="vc2-message is-user is-partial">
                  <span className="vc2-message-label">正在听</span>
                  <p>{v2.partialTranscript}<span className="vc2-caret" aria-hidden="true" /></p>
                </article>
              )}
              {v2.assistantText && (
                <article className="vc2-message is-assistant">
                  <span className="vc2-message-label">{companionName}</span>
                  <p data-testid="vc2-assistant">{v2.assistantText}</p>
                </article>
              )}
              {!hasConversation && (
                <div className="vc2-dialogue-empty">
                  <span className="vc2-empty-mark" aria-hidden="true"><SoundOutlined /></span>
                  <strong>{v2.connected ? '从一句话开始' : '等待语音连接'}</strong>
                  <p>{v2.connected ? '安静地说，停顿后我会自然接续。' : `${companionName} 的实时转写会出现在这里。`}</p>
                </div>
              )}
              {v2.phase === 'thinking' && !v2.assistantText && (
                <div className="vc2-thinking-line" role="status">
                  <span /><span /><span />
                  <em>{companionName} 正在组织回答</em>
                </div>
              )}
            </div>
          </aside>
        </main>
      )}

      {ready && (
        <footer className="vc2-command-dock">
          {actionPreview && (
            <div className="vc2-action-preview" role="note">
              <ThunderboltOutlined aria-hidden="true" />
              <span className="vc2-action-preview-text">{actionPreview.reply}</span>
              {actionPreview.executable && (
                <Button size="small" type="primary" className="vc2-action-preview-exec" onClick={executePreview} data-testid="vc2-action-exec">
                  执行
                </Button>
              )}
              <span className="vc2-action-preview-tag">
                {actionPreview.executable ? '识别到动作 · 待确认' : '动作暂不支持语音执行'}
              </span>
              <button className="vc2-action-preview-close" aria-label="关闭动作预览" onClick={() => setActionPreview(null)}>
                <CloseOutlined />
              </button>
            </div>
          )}
          {actionResult && <p className="vc2-feedback" role="status">{actionResult}</p>}
          {feedback && (
            <p
              className={`vc2-feedback${v2.phase === 'error' || turnWarning ? ' is-alert' : ''}`}
              role={v2.phase === 'error' || turnWarning ? 'alert' : 'status'}
              data-testid="vc2-note"
            >
              {feedback}
            </p>
          )}
          <div className="vc2-command-row">
            <div className="vc2-primary-actions">
              <Button
                type="primary"
                size="large"
                className="vc2-primary-button vc2-mic-button"
                icon={<PrimaryIcon />}
                onClick={runPrimaryAction}
                data-testid={v2.connected ? 'vc2-mic' : 'vc2-connect'}
              >
                {primaryLabel}
              </Button>
              {v2.connected && (
                <Button
                  size="large"
                  className="vc2-interrupt-button"
                  icon={<StopOutlined />}
                  onClick={v2.cancel}
                  data-testid="vc2-cancel"
                >
                  打断本轮
                </Button>
              )}
            </div>
            <span className="vc2-command-hint">
              {v2.micOn ? '说话后停顿，我会自动接续' : v2.connected ? '打开麦克风开始对话' : '服务就绪后建立实时连接'}
            </span>
          </div>

          <details className="vc2-diagnostics">
            <summary>连接与播放详情</summary>
            <div className="vc2-diagnostics-body">
              <div className="vc2-diagnostic-services">
                {SERVICE_ITEMS.map((item) => (
                  <span className={v2.services[item.key] ? 'is-online' : ''} key={item.key}>
                    {v2.services[item.key] ? <CheckCircleFilled /> : <MinusCircleOutlined />}
                    {item.label}
                  </span>
                ))}
              </div>
              <span className="vc2-mono" data-testid="vc2-pb">{v2.pbText || '播放队列：—'}</span>
              <div className="vc2-tail" data-testid="vc2-tail">
                {v2.tail.length > 0
                  ? v2.tail.map((line, index) => <span key={index}>{line}</span>)
                  : <span>暂无连接事件</span>}
              </div>
            </div>
          </details>
        </footer>
      )}
    </div>
  )
}

export default VoiceConsoleV2
