/**
 * voice/console/VoiceConsoleV2.tsx — 语音台 V2 实时会话视图（M4 产品接入刀）。
 *
 * voice.backend=v2 时由 ChatPage 渲染（V1 VoiceConsole 整体保留不动）。
 * UI 由协议相位直接驱动（V2 方案 §10.3）；轮完成写回当前聊天话题
 * （useChatVoice 同款纪律：会话绑开启时话题，迟到轮不串台）。
 */
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { BACKEND_EVENTS, subscribe } from '../../events'
import { Button, message, Tooltip } from 'antd'
import {
  AudioOutlined,
  ArrowLeftOutlined,
  CheckCircleFilled,
  CloseOutlined,
  DisconnectOutlined,
  DownOutlined,
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
    hint: '按下开始说话，准备好后直接开口',
    icon: <AudioOutlined />,
  },
  listening: {
    label: '正在聆听',
    hint: '正在听，说完停顿一下即可',
    icon: <AudioOutlined />,
  },
  thinking: {
    label: '正在思考',
    hint: '正在整理这一轮回答',
    icon: <LoadingOutlined spin />,
  },
  speaking: {
    label: '正在回应',
    hint: '回应中，开口或按“打断本轮”即可停止',
    icon: <SoundOutlined />,
  },
  interrupted: {
    label: '已打断',
    hint: '上一轮已停止，可以继续说',
    icon: <StopOutlined />,
  },
  error: {
    label: '本轮异常',
    hint: '语音没有完成，请重试或查看连接详情',
    icon: <WarningOutlined />,
  },
}

// 刀6 起 Go 原生管线=唯一路径：sidecar/快速模型（4B）项退役，
// realtime=进程内 Go realtime 服务器（8791）；8792 记忆网关已退役
// （LLM 腿进程内直连聊天板块功能绑定，无本地端口可探）。
const SERVICE_ITEMS = [
  { key: 'realtime', label: '实时管线' },
  { key: 'breeze', label: '声线服务' },
] as const

function userFacingVoiceNote(note: string): string {
  if (!note) return ''
  if (/Go realtime|voice native engine|语音原生引擎|启动失败/i.test(note)) {
    return '本机语音引擎未能启动。请检查 Windows 版语音组件后重试。'
  }
  return note
}

const VoiceConsoleV2: React.FC<VoiceConsoleV2Props> = ({
  companionName, instructions, voiceId, variant = 'full', topicTitle,
  onClose, onOpenSettings, setMessages, getActiveId, finalizeTopicAfterSend,
}) => {
  // 会话话题绑定：开启时锁定；切话题/关闭后迟到轮不写新会话（V1 同款纪律）
  const sessionTopicRef = useRef(getActiveId())
  const mountedRef = useRef(true)
  const commitChainRef = useRef<Promise<void>>(Promise.resolve())
  const rootRef = useRef<HTMLDivElement>(null)
  const threadRef = useRef<HTMLDivElement>(null)
  const threadFollowingRef = useRef(true)
  const [threadFollowing, setThreadFollowing] = useState(true)
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

  // M5 刀一：T1 执行成功也朗读回执（与 T0 直执行同款反馈闭环）
  // 回声门：朗读走后端 TTSSpeakStreaming（fire-and-forget，播完时机不可知）
  // → 前端拉 session.update 阈值 0.9（≈禁听，防扬声器残余回声假插话——11-01
  // 教训 0.6 被回声触发）+ 估时恢复 0.5（2s 底+220ms/字，上限 8s，宁长勿短）。
  // v2 在 onTurnComplete 之后创建（循环依赖），阈值控制经 ref 注入。
  const vadCtlRef = useRef<(v: number) => void>(() => {})
  const ackGateTimerRef = useRef<number | null>(null)
  const speakActionAck = useCallback((ack: string) => {
    if (!mountedRef.current) return
    vadCtlRef.current(0.9)
    void Promise.resolve()
      .then(() => app.TTSSpeakStreaming(ack))
      .catch(() => {}) // 朗读失败静默（回执已上屏，不打扰）
    if (ackGateTimerRef.current != null) window.clearTimeout(ackGateTimerRef.current)
    const chars = [...ack].length
    const holdMs = Math.min(8000, 2000 + chars * 220)
    ackGateTimerRef.current = window.setTimeout(() => {
      ackGateTimerRef.current = null
      if (mountedRef.current) vadCtlRef.current(0.5)
    }, holdMs)
  }, [])

  // M5 刀三：语音确认词通道（默认关，面板开关）——T1 确认卡在屏时说完整
  // 短语「确认执行」替代点按。防线：每预览一次（执行即清卡）+非播放期（卡在
  // 屏=无响应播放）+精确匹配（去标点后恒等，别的话不触发）。本轮对话照常
  // 走 LLM+落库（模型自然应答；垃圾轮代价接受，管线面拦截挂后续刀）。
  const [confirmWordEnabled, setConfirmWordEnabled] = useState(false)
  useEffect(() => {
    let alive = true
    void Promise.resolve()
      .then(() => app.VoiceGetSettings())
      .then((s) => {
        if (alive && s && typeof s.voiceConfirmWord === 'boolean') setConfirmWordEnabled(s.voiceConfirmWord)
      })
      .catch(() => {})
    return () => { alive = false }
  }, [])
  const maybeExecuteByConfirmWord = useCallback((userText: string) => {
    if (!confirmWordEnabled) return
    const normalized = userText.replace(/[，。！？,.!?、\s]/g, '')
    if (normalized !== '确认执行') return
    // 复用执行钮通道（内部无凭据=幂等早退；执行即清卡=每预览仅一次）
    executePreviewRef.current?.()
  }, [confirmWordEnabled])
  const executePreviewRef = useRef<() => void>(() => {})

  const executePreview = useCallback(() => {
    setActionPreview((cur) => {
      if (!cur?.previewId || !cur.intentHash) return cur
      void app.VoiceV2ExecuteAction(cur.previewId, cur.intentHash)
        .then((r) => {
          if (!mountedRef.current) return
          if (r?.executed === true) {
            setActionPreview(null)
            const ack = String(r.reply ?? '已执行')
            flashActionResult(ack)
            speakActionAck(ack)
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
  }, [flashActionResult, speakActionAck])
  executePreviewRef.current = executePreview

  const onTurnComplete = useCallback((turn: {
    user: string; assistant: string; responseId: string; itemId: string
  }) => {
    // M5 刀三：确认词轮先于一切落库逻辑（本轮照常提交——模型自然应答；
    // 拦截仅触发预览执行）
    maybeExecuteByConfirmWord(turn.user)
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
    // M5 刀一：T0（navigate/status）后端已直执行（executed=true/false）——
    // 前端出回执不出确认卡，并 TTS 朗读回执（设计稿 §3 反馈闭环）；朗读期
    // 拉监听阈值 0.9 防扬声器残余回声假插话（11-01 教训），估时恢复 0.5
    // （宁长勿短，上限 8s）。T1 维持预览台账+确认卡。
    // 经微任务调用：桥桩缺方法时同步抛错也能落进 catch（静默）。
    void Promise.resolve()
      .then(() => app.VoiceV2PreviewIntent(turn.user))
      .then((r) => {
        if (!mountedRef.current || !r?.handled) return
        if (typeof r.executed === 'boolean') {
          setActionResult('')
          if (r.executed) {
            const ack = String(r.reply ?? '已执行')
            flashActionResult(ack)
            speakActionAck(ack)
          } else {
            flashActionResult('动作未能执行，请再说一次')
          }
          return
        }
        if (!r.reply) return
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
  }, [finalizeTopicAfterSend, flashActionResult, getActiveId, maybeExecuteByConfirmWord, setMessages, speakActionAck])

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
      // M5 刀一：卸载清回声门计时器（残留会在无会话态调阈值——无害但脏）
      if (ackGateTimerRef.current != null) window.clearTimeout(ackGateTimerRef.current)
      // P0a-A5f：生成中关闭=先精确 abort 后端预推进（已完成轮已在提交链，
      // 不受影响；用户拍板「继续提交」），再断开传输。
      v2.abortInFlight()
      v2.disconnect()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // 回声门阈值控制注入（speakActionAck 定义于 v2 之前，经 ref 解循环）
  useEffect(() => {
    vadCtlRef.current = (v: number) => v2.updateVADThreshold(v)
  }, [v2.updateVADThreshold])

  // M5 刀二：模型工具 T1 预览凭据经后端事件下发（save_task 等不经规则解析，
  // PreviewIntent 摸不到）——收凭据直接出确认卡，执行复用 P2 通道。
  useEffect(() => {
    const unsub = subscribe(BACKEND_EVENTS.VOICE_TOOL_PREVIEW, (data: unknown) => {
      if (!mountedRef.current || !data || typeof data !== 'object') return
      const p = data as Record<string, unknown>
      const previewId = typeof p.previewId === 'string' ? p.previewId : ''
      const intentHash = typeof p.intentHash === 'string' ? p.intentHash : ''
      if (!previewId || !intentHash) return
      setActionResult('')
      setActionPreview({
        reply: String(p.reply ?? ''),
        action: String(p.action ?? ''),
        previewId,
        intentHash,
        executable: true,
      })
    })
    return unsub
  }, [])

  // Dedicated room: move focus into the workspace on entry. Float keeps the
  // compact bar and collapses back into it. Escape follows the same hierarchy.
  useEffect(() => {
    if (mode === 'full') rootRef.current?.focus()
    if (mode === 'bar') return
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return
      if (mode === 'float') setExpanded(false)
      else onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [mode, onClose])

  // Keep the live transcript anchored to the newest turn while the temporary
  // transcript and assistant response stream in. If the reader has moved up to
  // inspect an earlier turn, new chunks must not pull the viewport away.
  useEffect(() => {
    const el = threadRef.current
    if (!el || !threadFollowingRef.current) return
    el.scrollTop = el.scrollHeight
  }, [v2.dialogue, v2.partialTranscript])

  // ── 交互升级（ui-ux-pro-max 检索落地 2026-11-07）──
  // 打断涟漪：相位进入 interrupted 即触发一次性扩散环（key 重挂重放；
  // 无 animationend 依赖——瞬时可打断规则；reduced-motion 静态化在 CSS）
  const [interruptSeq, setInterruptSeq] = useState(0)
  const prevPhaseRef = useRef(v2.phase)
  useEffect(() => {
    if (v2.phase === 'interrupted' && prevPhaseRef.current !== 'interrupted') {
      setInterruptSeq((s) => s + 1)
    }
    prevPhaseRef.current = v2.phase
  }, [v2.phase])

  // 真声纹：电平历史环形采样去相关——五杆各取历史错位样本+当前电平混合，
  // 声纹呈波状展开（Siri/ChatGPT Voice 声纹条观感；说话侧=下行播放 RMS）
  const liveLevel = v2.micOn
    ? v2.micLevel
    : v2.phase === 'speaking' ? v2.playLevel : 0
  const levelHistoryRef = useRef<number[]>([])
  useEffect(() => {
    levelHistoryRef.current.push(liveLevel)
    if (levelHistoryRef.current.length > 8) levelHistoryRef.current.shift()
  }, [liveLevel])
  const barMs = useMemo(() => {
    // 波形包络：五杆固定形状权重（低-高-峰-落-谷），电平历史做时序去相关——
    // 单样本/静音时仍呈自然波形，有声时随电平起伏
    const barShape = [0.62, 0.95, 1.0, 0.78, 0.5]
    const h = levelHistoryRef.current
    const n = h.length || 1
    return barShape.map((shape, i) => {
      const sample = h[Math.max(n - 1 - ((i * 2 + 1) % n), 0)] ?? 0
      const base = 0.18 + 0.82 * (sample * 0.55 + liveLevel * 0.45)
      return Math.max(0.06, base * shape + 0.08 * (1 - shape))
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [liveLevel])

  const handleThreadScroll = useCallback((e: React.UIEvent<HTMLDivElement>) => {
    const el = e.currentTarget
    const following = el.scrollHeight - el.scrollTop - el.clientHeight <= 56
    if (threadFollowingRef.current === following) return
    threadFollowingRef.current = following
    setThreadFollowing(following)
  }, [])

  const scrollThreadToLatest = useCallback(() => {
    const el = threadRef.current
    if (!el) return
    threadFollowingRef.current = true
    setThreadFollowing(true)
    el.scrollTop = el.scrollHeight
  }, [])

  const ready = v2.services.ready
  const phaseMeta = PHASE_META[v2.phase]
  const feedback = turnWarning || userFacingVoiceNote(v2.note)
  const hasConversation = v2.dialogue.length > 0 || Boolean(v2.partialTranscript)
  const liveLine = v2.partialTranscript
    || (v2.phase === 'speaking' ? v2.assistantText : '')
    || (v2.connected ? phaseMeta.hint : '连接后即可开始实时语音对话')
  const PrimaryIcon = !ready
    ? v2.ensuring ? LoadingOutlined : AudioOutlined
    : !v2.connected ? WifiOutlined : v2.micOn ? PauseOutlined : AudioOutlined
  const primaryLabel = !ready
    ? v2.ensuring
      ? '正在准备本地语音'
      : v2.services.error ? '重试语音启动' : '启动本地语音'
    : !v2.connected
      ? '连接语音'
      : v2.micOn ? '暂停聆听' : '开始聆听'
  const responseActive = v2.phase === 'thinking' || v2.phase === 'speaking'
  const DockPrimaryIcon = responseActive ? StopOutlined : PrimaryIcon
  const dockPrimaryLabel = responseActive ? '打断回答' : primaryLabel

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

  const runDockPrimaryAction = () => {
    if (responseActive) {
      v2.cancel()
      return
    }
    runPrimaryAction()
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
        <span
          className={`vc2-bar-meter is-${v2.phase}`}
          aria-hidden="true"
          style={{ '--vc2-live': liveLevel.toFixed(3) } as React.CSSProperties}
        >
          {interruptSeq > 0 && (
            <span key={interruptSeq} className="vc2-ripple" data-testid="vc2-ripple" aria-hidden="true" />
          )}
          {Array.from({ length: 5 }, (_, i) => (
            <i key={i} style={{ '--vc2-m': barMs[i].toFixed(3) } as React.CSSProperties} />
          ))}
        </span>
        {/* 原子化状态播报（ui-ux-pro-max ux 规则）：只报相位一处，转写预览不做
            竞争性 live region（逐字流式会刷屏读屏） */}
        <span className="vc2-bar-phase" role="status" aria-atomic="true">{phaseMeta.label}</span>
        <span className="vc2-bar-preview" aria-hidden="true">
          {preview ? (
            <>
              <span className="vc2-bar-preview-src">{v2.phase === 'speaking' ? companionName : '你'}</span>
              {preview.slice(-72)}
            </>
          ) : (ready ? '按下开始说话' : '本地语音待启动')}
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
          aria-label={primaryLabel}
          aria-pressed={v2.connected ? v2.micOn : undefined}
          data-testid={v2.connected ? 'vc2-bar-mic' : 'vc2-bar-start'}
        >
          <span className="vc2-bar-label">{primaryLabel.replace('本地', '')}</span>
        </Button>
        {v2.connected && (
          <Button size="small" className="vc2-bar-btn" icon={<StopOutlined />} onClick={v2.cancel} data-testid="vc2-bar-cancel" aria-label="打断本轮回应">
            <span className="vc2-bar-label">打断</span>
          </Button>
        )}
        <Button size="small" className="vc2-bar-btn" icon={<ExpandOutlined />} onClick={() => setExpanded(true)} data-testid="vc2-bar-expand" aria-label="展开语音台">
          <span className="vc2-bar-label">展开</span>
        </Button>
        <Button size="small" className="vc2-bar-btn is-danger" icon={<CloseOutlined />} onClick={onClose} data-testid="vc2-bar-close" aria-label="结束语音会话">
          <span className="vc2-bar-label">结束</span>
        </Button>
        {v2.micSilent && (
          <p className="vc2-bar-note is-alert" role="alert">
            麦克风无信号。可能是虚拟音频设备、选错设备或静音；可在语音设置的「麦克风设备」里检测切换。
          </p>
        )}
        {feedback && <p className={`vc2-bar-note${v2.phase === 'error' ? ' is-alert' : ''}`} role="status">{feedback}</p>}
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
      ref={rootRef}
      id="voice-workspace"
      tabIndex={-1}
      aria-label="实时语音对话"
      className={`voice-console-v2${mode === 'float' ? ' voice-console-v2--float' : ''} voice-console-v2--${v2.phase}${ready ? '' : ' is-preflight'}`}
      data-testid="voice-console-v2"
      data-phase={v2.phase}
    >
      <header className="vc2-header">
        <div className="vc2-header-leading">
          {mode === 'full' && (
            <Tooltip title="返回聊天">
              <Button
                type="text"
                className="vc2-back-button"
                icon={<ArrowLeftOutlined />}
                onClick={onClose}
                aria-label="返回聊天"
              />
            </Tooltip>
          )}
          <div className="vc2-identity">
            <span className="vc2-brand-sigil" aria-hidden="true">
              <span />
            </span>
            <div>
              <span className="vc2-eyebrow">实时语音</span>
              <div className="vc2-title-row">
                <h2>{companionName}</h2>
                <span className="vc2-badge">语音空间</span>
              </div>
              <p>本地实时对话，声音与会话记录留在这台设备</p>
            </div>
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
            {mode === 'float' && (
              <Tooltip title="收起到输入区语音条">
                <Button type="text" icon={<ShrinkOutlined />} onClick={() => setExpanded(false)} aria-label="收起语音台" />
              </Tooltip>
            )}
            <Tooltip title="语音设置">
              <Button type="text" icon={<SettingOutlined />} onClick={onOpenSettings} aria-label="语音设置" />
            </Tooltip>
            {mode !== 'full' && <Tooltip title="关闭语音台">
              <Button type="text" icon={<CloseOutlined />} onClick={onClose} aria-label="关闭语音台" />
            </Tooltip>}
          </div>
        </div>
      </header>

      {!ready ? (
        <section className="vc2-preflight" aria-labelledby="vc2-preflight-title">
          <div className="vc2-preflight-copy">
            <span className="vc2-eyebrow">本机语音</span>
            <h3 id="vc2-preflight-title">装配本地语音</h3>
            <p>首次使用会准备本机语音引擎与声线服务，通常数秒完成。准备期间仍可继续输入文字。</p>
          </div>
          <div className="vc2-preflight-orb" aria-hidden="true">
            <VoiceV2Orb
              phase="idle"
              active={false}
              connected={false}
            />
            <span>{v2.services.error ? '语音场等待修复' : '语音场待启动'}</span>
          </div>
          <div className="vc2-service-list" aria-label="语音服务状态">
            {SERVICE_ITEMS.map((item) => {
              const online = v2.services[item.key]
              const starting = v2.ensuring && !online
              return (
                <div className={`vc2-service-item${online ? ' is-online' : starting ? ' is-starting' : ''}`} key={item.key}>
                  {online ? <CheckCircleFilled /> : starting ? <LoadingOutlined spin /> : <MinusCircleOutlined />}
                  <span>{item.label}</span>
                  <span className="vc2-service-state">{online ? '已就绪' : starting ? '装配中' : '未启动'}</span>
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
                <span className="vc2-eyebrow">声音状态</span>
                <h3>现在正在发生什么</h3>
              </div>
              <span className={`vc2-live-marker${v2.connected ? ' is-live' : ''}`}>
                <span aria-hidden="true" />
                {v2.connected ? '实时' : '就绪'}
              </span>
            </header>

            <div className="vc2-field">
              <VoiceV2Orb
                phase={v2.phase}
                active={v2.micOn || v2.phase === 'speaking'}
                connected={v2.connected}
                level={liveLevel}
              />
              {interruptSeq > 0 && (
                <span key={interruptSeq} className="vc2-ripple vc2-ripple--stage" data-testid="vc2-ripple-stage" aria-hidden="true" />
              )}
              <div className="vc2-stage-copy">
                <span
                  className={`vc2-phase vc2-phase-${v2.phase}`}
                  data-testid="vc2-phase"
                  role="status"
                  aria-atomic="true"
                >
                  {phaseMeta.icon}
                  {phaseMeta.label}
                </span>
                <p className="vc2-live-line" aria-hidden="true">{liveLine}</p>
                <span className="vc2-stage-hint">{phaseMeta.hint}</span>
              </div>
              <div className="vc2-wave-field" aria-hidden="true">
                {Array.from({ length: 24 }, (_, index) => (
                  <i key={index} style={{ '--vc2-wave': 0.18 + ((index * 7) % 13) / 16 } as React.CSSProperties} />
                ))}
              </div>
            </div>

            <footer className="vc2-stage-facts">
              <span>
                <AudioOutlined /> 麦克风{v2.micOn ? (v2.micSilent ? '无信号' : '开启') : '关闭'}
                {v2.micOn && !v2.micSilent && (
                  <i className="vc2-mic-level" aria-hidden="true" style={{ '--vc2-live': v2.micLevel.toFixed(3) } as React.CSSProperties} />
                )}
              </span>
              <span><CheckCircleFilled /> 本机处理</span>
              <span>{v2.connected ? '实时连接' : '等待连接'}</span>
            </footer>
          </section>

          <aside className="vc2-dialogue" aria-labelledby="vc2-dialogue-title">
            <header className="vc2-section-head">
              <div>
                <span className="vc2-eyebrow">对话记录</span>
                <h3 id="vc2-dialogue-title">这一轮聊了什么</h3>
              </div>
              <div className="vc2-dialogue-tools">
                <span className="vc2-turn-count">{v2.transcripts.length} 轮对话</span>
                {hasConversation && !threadFollowing && (
                  <button
                    type="button"
                    className="vc2-thread-jump"
                    onClick={scrollThreadToLatest}
                    aria-label="回到最新一条对话"
                    data-testid="vc2-jump-latest"
                  >
                    <DownOutlined aria-hidden="true" />
                    <span>回到最新</span>
                  </button>
                )}
              </div>
            </header>

            <div
              ref={threadRef}
              className="vc2-thread"
              data-testid="vc2-transcripts"
              role="log"
              aria-label="实时语音转写"
              aria-live="polite"
              aria-busy={v2.phase === 'thinking'}
              tabIndex={0}
              onScroll={handleThreadScroll}
            >
              {v2.dialogue.map((entry) => (
                <article className={`vc2-message is-${entry.role}`} key={entry.id}>
                  <span className="vc2-message-label">
                    {entry.role === 'user' ? '你' : companionName}
                  </span>
                  <p data-testid={entry.role === 'assistant' ? 'vc2-assistant' : undefined}>
                    {entry.text}
                  </p>
                </article>
              ))}
              {v2.partialTranscript && (
                <article className="vc2-message is-user is-partial">
                  <span className="vc2-message-label">正在听</span>
                  <p>{v2.partialTranscript}<span className="vc2-caret" aria-hidden="true" /></p>
                </article>
              )}
              {!hasConversation && (
                <div className="vc2-dialogue-empty">
                  <span className="vc2-empty-mark" aria-hidden="true"><SoundOutlined /></span>
                  <strong>{v2.connected ? '从一句话开始' : '连接后开始对话'}</strong>
                  <p>{v2.connected ? '自然地说，停顿后我会接续。转写会同步写回当前会话。' : `${companionName} 的转写与回答会出现在这里。`}</p>
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
                className={`vc2-primary-button vc2-mic-button${responseActive ? ' is-interrupting' : ''}`}
                icon={<DockPrimaryIcon />}
                onClick={runDockPrimaryAction}
                aria-label={dockPrimaryLabel}
                aria-pressed={v2.connected && !responseActive ? v2.micOn : undefined}
                data-testid={responseActive ? 'vc2-interrupt-primary' : v2.connected ? 'vc2-mic' : 'vc2-connect'}
              >
                {dockPrimaryLabel}
              </Button>
            </div>
            <Button
              size="large"
              className="vc2-end-session"
              icon={<CloseOutlined />}
              onClick={onClose}
              aria-label="结束语音会话"
              title="结束语音会话"
              data-testid="vc2-end-session"
            >
              <span className="vc2-end-session-label">结束语音</span>
            </Button>
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
              <span className="vc2-mono" data-testid="vc2-pb">{v2.pbText || '播放队列：暂无'}</span>
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
