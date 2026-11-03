/**
 * voice/runtime/useVoiceChatV2.ts — Voice V2 会话运行时钩子（M4 产品接入刀）。
 *
 * 聚合：VoiceClientV2（协议/状态机/播放队列）+ AudioWorklet 采集/播放 + sidecar
 * 依赖链监护（VoiceV2Ensure/VoiceV2Status）。transport 可注入（实验页 mock 回放
 * /测试桩）；jsdom 无 audioWorklet 时开麦诚实降级为 note 提示。
 * 消费方：语音台 V2 视图（VoiceConsole 内 v2 分支）与 ?voicev2=1 实验页。
 */
import { useCallback, useEffect, useRef, useState } from 'react'
import { VoiceClientV2, type VoiceTransport } from '../client/VoiceClientV2'
import type { VoiceSessionState } from '../session/voiceSession'
import type { ServerEvent } from '../protocol/events'
import { app } from '../../gaea/lib/bridge'
import { subscribeWailsEvent } from '../../gaea/lib/wailsEvents'

export interface VoiceV2Turn {
  user: string
  assistant: string
  responseId: string
  itemId: string
}

export interface UseVoiceChatV2Options {
  url?: string
  instructions?: string
  /** 人格音色（breeze 克隆 id；空=服务端 YAML 缺省 char_gaea）——音色漂移根修 */
  voice?: string
  /** 真连 transport 工厂注入（测试）；缺省走 WebSocket */
  transportFactory?: (url: string) => VoiceTransport
  /** mock 回放专用哑线缆工厂（replayEvents 消费；与真连互不干扰） */
  makeReplayTransport?: (url: string) => VoiceTransport
  /** 轮完成回执（response.done=completed）：聊天写回等消费方 */
  onTurnComplete?: (turn: VoiceV2Turn) => void
}

const DEFAULT_URL = 'ws://127.0.0.1:8791/v1/realtime'

class WsTransport implements VoiceTransport {
  constructor(private ws: WebSocket) {}
  send(data: string): void {
    this.ws.send(data)
  }
  close(): void {
    this.ws.close()
  }
  onopen: (() => void) | null = null
  onclose: (() => void) | null = null
  onerror: ((err: unknown) => void) | null = null
  onmessage: ((data: string) => void) | null = null
}

function adaptWs(ws: WebSocket): WsTransport {
  const t = new WsTransport(ws)
  ws.onopen = () => t.onopen?.()
  ws.onclose = () => t.onclose?.()
  ws.onerror = () => t.onerror?.(new Error('ws error'))
  ws.onmessage = (e) => t.onmessage?.(String(e.data))
  return t
}

export interface VoiceChatV2Api {
  phase: VoiceSessionState['phase']
  transcripts: string[]
  partialTranscript: string
  assistantText: string
  pbText: string
  tail: string[]
  connected: boolean
  micOn: boolean
  note: string
  services: {
    sidecar: boolean; llm: boolean; breeze: boolean; gateway: boolean
    ready: boolean; error: string
  }
  /** VoiceV2Ensure 正在冷启动依赖，按钮/重复调用可据此禁用 */
  ensuring: boolean
  connect: () => Promise<void>
  disconnect: () => void
  startMic: () => Promise<void>
  stopMic: () => void
  cancel: () => void
  /** 依赖链监护（VoiceV2Ensure + 状态轮询刷新） */
  ensureServices: () => Promise<boolean>
  refreshServices: () => Promise<boolean>
  /** mock 回放：不经线缆直接灌事件（实验页/测试） */
  replayEvents: (events: ServerEvent[], paceMs?: number) => Promise<void>
}

export function useVoiceChatV2(opts: UseVoiceChatV2Options = {}): VoiceChatV2Api {
  const [phase, setPhase] = useState<VoiceSessionState['phase']>('idle')
  const [transcripts, setTranscripts] = useState<string[]>([])
  const [partial, setPartial] = useState('')
  const [assistantText, setAssistantText] = useState('')
  const [pbText, setPbText] = useState('')
  const [tail, setTail] = useState<string[]>([])
  const [connected, setConnected] = useState(false)
  const [micOn, setMicOn] = useState(false)
  const [note, setNote] = useState('')
  const [services, setServices] = useState({
    sidecar: false, llm: false, breeze: false, gateway: false, ready: false, error: '',
  })
  const [ensuring, setEnsuring] = useState(false)

  const clientRef = useRef<VoiceClientV2 | null>(null)
  const transportRef = useRef<VoiceTransport | null>(null)
  const connectedRef = useRef(false)
  const mountedRef = useRef(true)
  const stopMicRef = useRef<() => void>(() => {})
  const speechStartRef = useRef(0)
  const responseIdRef = useRef('')
  const optsRef = useRef(opts)
  optsRef.current = opts
  const capCtxRef = useRef<AudioContext | null>(null)
  const capStreamRef = useRef<MediaStream | null>(null)
  const capNodeRef = useRef<AudioWorkletNode | null>(null)
  const sinkCtxRef = useRef<AudioContext | null>(null)
  const sinkNodeRef = useRef<AudioWorkletNode | null>(null)
  const pumpRef = useRef<number | null>(null)
  const tailRef = useRef<string[]>([])
  const urlRef = useRef(opts.url ?? DEFAULT_URL)
  const ensuringRef = useRef(false)
  const serviceSnapshotRef = useRef({
    ready: false, error: '', stage: '',
  })
  urlRef.current = opts.url ?? DEFAULT_URL

  const pushTail = useCallback((line: string) => {
    const next = [...tailRef.current, line].slice(-14)
    tailRef.current = next
    if (mountedRef.current) setTail(next)
  }, [])

  const flushSink = useCallback(() => {
    // PlaybackQueue 只是调度面；worklet ring 里可能已有提前泵入的音频。
    sinkNodeRef.current?.port.postMessage({ type: 'flush' })
  }, [])

  const onState = useCallback((s: VoiceSessionState) => {
    if (s.speechStartCount > speechStartRef.current) {
      speechStartRef.current = s.speechStartCount
      flushSink()
    }
    // 新 response 开启前清旧 ring，防旧代际音频跨轮泄漏。
    if (s.activeResponseId && s.activeResponseId !== responseIdRef.current) {
      responseIdRef.current = s.activeResponseId
      flushSink()
    }
    setPhase(s.phase)
    setTranscripts(s.committedTranscripts)
    setPartial(s.partialTranscript)
    setAssistantText(s.assistantText)
  }, [flushSink])

  const ensureSink = useCallback(async () => {
    if (sinkCtxRef.current) return
    let ctx: AudioContext
    try {
      ctx = new AudioContext({ sampleRate: 16000 })
    } catch {
      // 设备拒绝 16k 时 worklet 按实际 sampleRate 重采样，链路仍可用。
      ctx = new AudioContext()
    }
    if (ctx.state === 'suspended') {
      // 自动挂载不是用户手势，失败留给开麦点击再恢复，不能因此拒绝建立会话。
      void ctx.resume().catch(() => {})
    }
    await ctx.audioWorklet.addModule('/voice-worklets/sinkProcessor.js')
    const node = new AudioWorkletNode(ctx, 'gaea-voice-sink', { outputChannelCount: [2] })
    node.connect(ctx.destination)
    node.port.onmessage = (e) => {
      if (e.data && typeof e.data.overflow === 'number' && e.data.overflow > 0) {
        pushTail(`sink overflow ×${e.data.overflow}（上游泄漏信号）`)
      }
    }
    sinkCtxRef.current = ctx
    sinkNodeRef.current = node
    if (ctx.sampleRate !== 16000) pushTail(`音频输出 ${ctx.sampleRate}Hz（16k source 自动重采样）`)
    pumpRef.current = window.setInterval(() => {
      const client = clientRef.current
      const sink = sinkNodeRef.current
      if (!client || !sink) return
      for (let chunk = client.playback.pull(); chunk; chunk = client.playback.pull()) {
        sink.port.postMessage({ type: 'pcm', buffer: chunk.buffer }, [chunk.buffer])
      }
      const st = client.playback.stats
      setPbText(`${st.currentResponseId || '—'} 播${st.playedBytes}B 弃${st.droppedBytes}B 存${st.pendingChunkCount}块${st.playbackDone ? ' ✓回执' : ''}`)
    }, 100)
  }, [pushTail])

  const buildClient = useCallback((transport: VoiceTransport) => {
    return new VoiceClientV2({
      transport,
      instructions: optsRef.current.instructions,
      voice: optsRef.current.voice,
      vadThreshold: 0.75, // 2026-11-01 实弹：0.6 易被扬声器回声触发假插话（截断回复）
      interruptResponse: true,
      onState,
      onPlaybackDone: (id) => pushTail(`播放完成回执 ${id}`),
      onOpen: () => {
        if (!mountedRef.current) return
        connectedRef.current = true
        setConnected(true)
        setNote('')
        pushTail('已连接')
      },
      onClose: () => {
        if (transportRef.current !== transport) return
        const client = clientRef.current
        transportRef.current = null
        clientRef.current = null
        connectedRef.current = false
        stopMicRef.current()
        client?.playback.reset()
        flushSink()
        if (!mountedRef.current) return
        setConnected(false)
        setNote('连接已断开，可点击重连')
        pushTail('连接关闭')
      },
      onError: (error) => {
        if (!mountedRef.current) return
        setNote(`连接异常：${error instanceof Error ? error.message : String(error)}`)
      },
      onResponseDone: (info) => {
        pushTail(`response.done ${info.status}${info.reason ? '/' + info.reason : ''} ${info.responseId}`)
        if (info.status !== 'completed') return
        // 用户与助手文本由 response ledger 精确配对；不能用全局最后一句猜，
        // 否则旧 response.done 迟到时会把旧答写进新用户轮。
        const user = info.user.trim()
        const assistant = info.assistant.trim()
        if (!user || !assistant) return
        optsRef.current.onTurnComplete?.({
          user, assistant, responseId: info.responseId, itemId: info.itemId,
        })
      },
    })
  }, [flushSink, onState, pushTail])

  const connect = useCallback(async () => {
    if (clientRef.current || transportRef.current) return
    try {
      await ensureSink()
    } catch (err) {
      // 播放引擎起不来必须可见（开麦/播报都依赖 sink；此前 void 吞异常=按钮无反应）
      setNote(`音频引擎初始化失败: ${err instanceof Error ? err.message : String(err)}`)
      return
    }
    // url 实时读 optsRef（实验页改 URL 后免重渲染即连）
    const targetUrl = optsRef.current.url ?? DEFAULT_URL
    let transport: VoiceTransport
    try {
      transport = optsRef.current.transportFactory
        ? optsRef.current.transportFactory(targetUrl)
        : adaptWs(new WebSocket(targetUrl))
    } catch (err) {
      setNote(`连接失败: ${err instanceof Error ? err.message : String(err)}`)
      return
    }
    if (!mountedRef.current) {
      transport.close()
      return
    }
    // 新会话清掉上一连接的可视账；聊天持久化不在这里删除。
    setPhase('idle')
    setTranscripts([])
    setPartial('')
    setAssistantText('')
    setPbText('')
    setConnected(false)
    connectedRef.current = false
    speechStartRef.current = 0
    responseIdRef.current = ''
    setNote('连接中…')
    transportRef.current = transport
    clientRef.current = buildClient(transport)
  }, [buildClient, ensureSink])

  const disconnect = useCallback(() => {
    const client = clientRef.current
    const transport = transportRef.current
    transportRef.current = null
    clientRef.current = null
    connectedRef.current = false
    speechStartRef.current = 0
    responseIdRef.current = ''
    stopMicRef.current()
    client?.playback.reset()
    flushSink()
    if (client) client.dispose()
    else transport?.close()
    if (!mountedRef.current) return
    setConnected(false)
    setNote('')
    pushTail('已断开')
  }, [flushSink, pushTail])

  const stopMic = useCallback(() => {
    capNodeRef.current?.port.close()
    capNodeRef.current?.disconnect()
    capNodeRef.current = null
    capStreamRef.current?.getTracks().forEach((t) => t.stop())
    capStreamRef.current = null
    void capCtxRef.current?.close()
    capCtxRef.current = null
    if (mountedRef.current) setMicOn(false)
    pushTail('采集停止')
  }, [pushTail])
  stopMicRef.current = stopMic

  const startMicInner = useCallback(async () => {
    const client = clientRef.current
    if (!client || !connectedRef.current) {
      setNote('先连接服务再开麦')
      return
    }
    // 逐步标签：异常带阶段名上屏（无栈错如 Illegal invocation 的定位手段）
    let stage = 'env'
    const at = (n: string) => { stage = n }
    try {
      at('sink-resume')
      const sinkCtx = sinkCtxRef.current
      if (sinkCtx?.state === 'suspended') await sinkCtx.resume()
      at('gum')
      // 采集约束阶梯（884e9eb3 同族）：回声消除/降噪在每一档都保住（扬声器外放
      // 场景丢了 EC = AI 自打断截断回复，2026-11-01 实弹教训）；仅逐档削设备级
      // 约束。AudioWorklet 无 bufferSize 参数，ScriptProcessor 雷不存在。
      let stream: MediaStream
      const ec = { echoCancellation: true, noiseSuppression: true, autoGainControl: true }
      try {
        stream = await navigator.mediaDevices.getUserMedia({ audio: { ...ec, channelCount: 1 } })
      } catch {
        try {
          stream = await navigator.mediaDevices.getUserMedia({ audio: ec })
        } catch {
          stream = await navigator.mediaDevices.getUserMedia({ audio: true })
        }
      }
      at('ctx')
      let ctx: AudioContext
      try {
        ctx = new AudioContext({ sampleRate: 16000 })
      } catch {
        ctx = new AudioContext() // 个别设备拒绝 16k 强制档；回落默认率（服务端重采样）
      }
      at('addModule')
      await ctx.audioWorklet.addModule('/voice-worklets/captureProcessor.js')
      at('graph')
      const src = ctx.createMediaStreamSource(stream)
      const node = new AudioWorkletNode(ctx, 'gaea-voice-capture')
      node.port.onmessage = (e) => {
        clientRef.current?.sendAudioChunk(new Int16Array(e.data))
      }
      src.connect(node)
      at('state')
      capCtxRef.current = ctx
      capStreamRef.current = stream
      capNodeRef.current = node
      setMicOn(true)
      pushTail('采集开始（20ms/320 Int16@16k）')
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err)
      throw new Error(`${stage}: ${msg}`)
    }
  }, [pushTail])

  const startMic = useCallback(async () => {
    // 特性检测必须用 `in`：audioWorklet 是 prototype 上的 accessor getter，
    // 直接取值（!!AudioContext.prototype.audioWorklet）在真浏览器抛
    // Illegal invocation（jsdom 桩测不出，2026-11-01 实弹开麦无反应根因）。
    if (typeof AudioContext === 'undefined' || !('audioWorklet' in AudioContext.prototype)) {
      setNote('当前环境不支持 AudioWorklet 采集')
      return
    }
    try {
      await startMicInner()
    } catch (err) {
      // 采集链任何一步失败必须可见（此前 void 吞异常=按钮无反应，2026-11-01 实弹）
      setNote(`开麦失败 ${err instanceof Error ? err.message : String(err)}`)
    }
  }, [startMicInner])

  const cancel = useCallback(() => {
    clientRef.current?.cancel()
    flushSink()
  }, [flushSink])

  const refreshServices = useCallback(async (): Promise<boolean> => {
    try {
      const st = (await app.VoiceV2Status()) as Record<string, unknown>
      const ready = st.ready === true
      const error = typeof st.error === 'string' ? st.error.trim() : ''
      const stage = typeof st.stage === 'string' ? st.stage.trim() : ''
      serviceSnapshotRef.current = { ready, error, stage }
      if (mountedRef.current) {
        setServices({
          sidecar: st.sidecar === true,
          llm: st.llm === true,
          breeze: st.breeze === true,
          gateway: st.gateway === true,
          ready,
          error,
        })
        if (!ready && (error || stage)) setNote(error || stage)
      }
      return ready
    } catch {
      /* 桥不可用（纯浏览器）：保持 false */
      serviceSnapshotRef.current = { ready: false, error: '', stage: '' }
      return false
    }
  }, [])

  const ensureServices = useCallback(async (): Promise<boolean> => {
    if (ensuringRef.current) return false
    ensuringRef.current = true
    setEnsuring(true)
    setNote('正在检测本地语音服务…')
    try {
      const st = (await app.VoiceV2Ensure()) as Record<string, unknown>
      const immediateError = typeof st.error === 'string' ? st.error.trim() : ''
      if (immediateError) {
        setNote(immediateError)
        return false
      }
      if (st.ready === true) {
        const ready = await refreshServices()
        if (ready) setNote('')
        return ready
      }
      // sidecar 首次加载 Paraformer/Smart Turn 实测可超过 90s；与后端 180s
      // 硬超时同口径，保持进度可见；一旦后端回报终止错误立即停轮询，不能让
      // 缺脚本/缺 venv 的失败仍把按钮锁满三分钟。
      for (let i = 0; i < 60; i++) {
        await new Promise((r) => setTimeout(r, 3000))
        if (!mountedRef.current) return false
        if (await refreshServices()) {
          setNote('')
          return true
        }
        const { error, stage } = serviceSnapshotRef.current
        if (error) {
          setNote(error)
          return false
        }
        const elapsed = (i + 1) * 3
        setNote(`${stage || '正在准备本地语音服务'}（已等待 ${elapsed}s）`)
      }
      setNote('依赖链未在 180s 内就绪，可重试或检查语音引擎日志')
      return false
    } catch (err) {
      setNote(`拉起失败: ${err instanceof Error ? err.message : String(err)}`)
      return false
    } finally {
      ensuringRef.current = false
      if (mountedRef.current) setEnsuring(false)
    }
  }, [refreshServices])

  const replayEvents = useCallback(async (events: ServerEvent[], paceMs = 90) => {
    if (clientRef.current) {
      setNote('mock 回放需先断开服务连接')
      return
    }
    const makeTransport = optsRef.current.makeReplayTransport
    if (!makeTransport) {
      setNote('mock 回放需要 makeReplayTransport 注入（哑线缆）')
      return
    }
    await ensureSink()
    clientRef.current = buildClient(makeTransport(urlRef.current))
    setConnected(false)
    setNote(`mock 回放中（${events.length} 事件）`)
    for (const ev of events) {
      pushTail(`⇐ ${ev.type}`)
      clientRef.current.handleServerEvent(ev)
      await new Promise((r) => setTimeout(r, paceMs))
    }
    setNote('mock 回放完成')
  }, [buildClient, ensureSink, pushTail])

  useEffect(() => {
    if (!window.runtime?.EventsOn) return
    return subscribeWailsEvent(window.runtime, 'voice-v2-status', (payload) => {
      if (!mountedRef.current) return
      const ev = payload as { ready?: boolean; error?: string; stage?: string }
      if (ev.ready === true) {
        void refreshServices().then((ready) => {
          if (!ready || !mountedRef.current) return
          setNote('')
          if (!clientRef.current && !transportRef.current) void connect()
        })
        return
      }
      if (typeof ev.error === 'string' && ev.error.trim()) setNote(ev.error.trim())
      else if (typeof ev.stage === 'string' && ev.stage.trim()) setNote(ev.stage.trim())
    })
  }, [connect, refreshServices])

  // 事件与按钮 Promise 之外的最后一道自愈：冷启动可能持续数十秒，控制台只要
  // 未就绪就持续低频轮询；四项齐备后自动连上，不要求用户再点一次。
  useEffect(() => {
    if (services.ready) return
    let cancelled = false
    const timer = window.setInterval(() => {
      if (cancelled || !mountedRef.current) return
      void refreshServices().then((ready) => {
        if (!ready || cancelled || !mountedRef.current) return
        setNote('')
        if (!clientRef.current && !transportRef.current) void connect()
      })
    }, 3000)
    return () => {
      cancelled = true
      window.clearInterval(timer)
    }
  }, [connect, refreshServices, services.ready])

  useEffect(() => {
    mountedRef.current = true
    void refreshServices()
    return () => {
      mountedRef.current = false
      if (pumpRef.current) window.clearInterval(pumpRef.current)
      void capCtxRef.current?.close()
      void sinkCtxRef.current?.close()
      capStreamRef.current?.getTracks().forEach((t) => t.stop())
      flushSink()
      transportRef.current = null
      clientRef.current?.dispose()
      clientRef.current = null
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return {
    phase, transcripts, partialTranscript: partial, assistantText,
    pbText, tail, connected, micOn, note, services, ensuring,
    connect, disconnect, startMic, stopMic, cancel, ensureServices, refreshServices, replayEvents,
  }
}
