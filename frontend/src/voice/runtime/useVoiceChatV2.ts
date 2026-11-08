/**
 * voice/runtime/useVoiceChatV2.ts — Voice V2 会话运行时钩子（M4 产品接入刀）。
 *
 * 聚合：VoiceClientV2（协议/状态机/播放队列）+ AudioWorklet 采集/播放 + 依赖链
 * 监护（VoiceV2Ensure/VoiceV2Status，Go 原生管线唯一路径）。transport 可注入（实验页 mock 回放
 * /测试桩）；jsdom 无 audioWorklet 时开麦诚实降级为 note 提示。
 * 消费方：语音台 V2 视图（VoiceConsole 内 v2 分支）与 ?voicev2=1 实验页。
 */
import { useCallback, useEffect, useRef, useState } from 'react'
import { VoiceClientV2, type VoiceTransport } from '../client/VoiceClientV2'
import {
  voiceTranscriptMessages,
  type VoiceSessionState,
  type VoiceTranscriptMessage,
} from '../session/voiceSession'
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
// P1 有限次自动重连：指数退避 1/2/4/8/15s 封顶，最多 5 次后交还手动
const RECONNECT_MAX_ATTEMPTS = 5
const RECONNECT_BASE_MS = 1000
const RECONNECT_MAX_DELAY_MS = 15000
// 重连计数复位窗：连接须稳定保持此时长才算"真正恢复"。open 即清零的旧语义
// 在闪断链路（连接被服务端会话槽秒弹）下每次 open 都清零=1s 重连循环永不耗尽，
// 2026-11-04 实弹三次重启同一死局（启动连接卡死）根因之一。
const RECONNECT_STABLE_MS = 10000

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
  /** 按轮次展开的用户/助手交替时间线（含当前轮实时文本） */
  dialogue: VoiceTranscriptMessage[]
  partialTranscript: string
  assistantText: string
  pbText: string
  tail: string[]
  connected: boolean
  micOn: boolean
  /** 采集电平（0..1，worklet RMS 每 ~20ms 上报；0=静音） */
  micLevel: number
  /** 说话侧电平（0..1，声卡实际外放 RMS；speaking 相位声纹/涟漪驱动源） */
  playLevel: number
  /** 采集电平的逐帧 ref（50Hz）：星团画布直接读，避免 React 每帧重渲染 */
  micLevelRef?: { current: number }
  /** 播放电平的逐帧 ref（20Hz）：星团画布直接读，和声卡实际出声同步 */
  playLevelRef?: { current: number }
  /** 哑麦诚实检测：开麦且已连接但持续 3s 无信号（虚拟设备/选错设备/静音） */
  micSilent: boolean
  note: string
  services: {
    realtime: boolean; breeze: boolean
    ready: boolean; error: string
  }
  /** VoiceV2Ensure 正在冷启动依赖，按钮/重复调用可据此禁用 */
  ensuring: boolean
  connect: () => Promise<void>
  disconnect: () => void
  startMic: () => Promise<void>
  stopMic: () => void
  cancel: () => void
  /** 运行时改监听阈值（M5 刀一回执朗读的回声门：朗读期拉高防扬声器残余回声假插话） */
  updateVADThreshold: (threshold: number) => void
  /** 生成中关闭的精确回滚（P0a-A5f）：有未闭合 response 时发 cancel 并调
   * VoiceV2AbortTurn 回滚后端预推进；已完成轮已入提交链（A5e），此处不触碰。 */
  abortInFlight: () => void
  /** 依赖链监护（VoiceV2Ensure + 状态轮询刷新） */
  ensureServices: () => Promise<boolean>
  refreshServices: () => Promise<boolean>
  /** mock 回放：不经线缆直接灌事件（实验页/测试） */
  replayEvents: (events: ServerEvent[], paceMs?: number) => Promise<void>
  /** 语音打断开关当前值（A 拍板：界面切换钮；关=回复完整播完不被语音打断） */
  interruptEnabled: boolean
  /** 切换语音打断开关（session.update 即时下发；重连/新会话沿用） */
  setInterruptEnabled: (v: boolean) => void
}

export function useVoiceChatV2(opts: UseVoiceChatV2Options = {}): VoiceChatV2Api {
  const [phase, setPhase] = useState<VoiceSessionState['phase']>('idle')
  const [transcripts, setTranscripts] = useState<string[]>([])
  const [dialogue, setDialogue] = useState<VoiceTranscriptMessage[]>([])
  const [partial, setPartial] = useState('')
  const [assistantText, setAssistantText] = useState('')
  const [pbText, setPbText] = useState('')
  const [tail, setTail] = useState<string[]>([])
  const [connected, setConnected] = useState(false)
  const [micOn, setMicOn] = useState(false)
  const [micLevel, setMicLevel] = useState(0)
  // 说话相位声纹源：声卡实际外放 RMS（sink worklet __gaeaPlayLevel；delta 回退）
  const [playLevel, setPlayLevel] = useState(0)
  // 星团画布用的近实时电平：worklet 50Hz / 声卡 20Hz 写 ref，React 只按 60ms
  // 节流取一次做 CSS 电平（否则整棵语音台 50Hz 重渲染，长会话更卡）。
  const micLevelRef = useRef(0)
  const playLevelRef = useRef(0)
  const micStateTsRef = useRef(0)
  const [micSilent, setMicSilent] = useState(false)
  const [note, setNote] = useState('')
  const [services, setServices] = useState({
    realtime: false, breeze: false, ready: false, error: '',
  })
  const [ensuring, setEnsuring] = useState(false)

  const clientRef = useRef<VoiceClientV2 | null>(null)
  // 语音打断开关（A 拍板，2026-11-07：连续说话场景的断续根除旋钮；上游 M13
  // interrupt_response_enabled 语义——关=回复完整播完，真打断走 UI 打断键）
  const [interruptEnabled, setInterruptEnabledState] = useState(true)
  const interruptEnabledRef = useRef(true)
  const transportRef = useRef<VoiceTransport | null>(null)
  const connectedRef = useRef(false)
  const mountedRef = useRef(true)
  const stopMicRef = useRef<() => void>(() => {})
  // 哑麦检测：最近一次有信号时刻（level 消息热路径走 ref，1s 兜底轮询只碰 state）
  const lastSoundTsRef = useRef(0)
  const micSilentTimerRef = useRef<number | null>(null)
  const speechStartRef = useRef(0)
  const responseIdRef = useRef('')
  // playLevel setState 节流（2026-11-07 修复 D）：VoiceClientV2 每个 delta 调
  // 一次 onPlayLevel，长文本流式期每秒数十次全树渲染——泵 fill 报告被渲染
  // 挤到迟到=双灌根因之一，且渲染成本随会话变长而涨=「越到后面越严重」。
  // 60ms 粒度对声纹 UI 足够；归零（done/取消）立即放行不节流。
  const playLevelLastTsRef = useRef(0)
  const onPlayLevelThrottled = useCallback((lv: number) => {
    // 画布 ref 每一条电平消息都写（20Hz/50Hz），state 仍按 60ms 节流。
    const prev = playLevelRef.current
    playLevelRef.current = lv
    if (lv === 0 && prev === 0) return
    const now = Date.now()
    if (lv === 0 || now - playLevelLastTsRef.current >= 60) {
      playLevelLastTsRef.current = now
      setPlayLevel(lv)
    }
  }, [])
  // sink ring 水位（worklet ~10 次/秒上报；泵按水位灌注，审计 B1）
  const sinkFillRef = useRef(0)
  // 泵自上次 fill 报告以来已灌入的样本数（2026-11-07 修复 A：报告经主线程
  // 消息队列有迟到——长文本期 React 渲染密集时迟到数个泵 tick，旧 fill 参考
  // 会让泵重复灌注→ring 超限。本地记账=报告迟到期间泵自知已灌量）
  const postedSinceReportRef = useRef(0)
  // 播放面诊断 tap（重叠线复开 2026-11-07）：worklet 实际外放样本攒批落盘。
  // 队列在 ref 里流转（零 setState），POST 泵 500ms 一趟。
  const tapQueueRef = useRef<ArrayBuffer[]>([])
  const tapFileRef = useRef('')
  const tapSeqRef = useRef(0)
  const tapTruncateRef = useRef(true)
  const tapRateRef = useRef(16000)
  const tapNoteOnceRef = useRef(false)
  const optsRef = useRef(opts)
  optsRef.current = opts
  const capCtxRef = useRef<AudioContext | null>(null)
  const capStreamRef = useRef<MediaStream | null>(null)
  const capNodeRef = useRef<AudioWorkletNode | null>(null)
  const sinkCtxRef = useRef<AudioContext | null>(null)
  const sinkNodeRef = useRef<AudioWorkletNode | null>(null)
  const pumpRef = useRef<number | null>(null)
  // 诊断 tap POST 泵（ensureSink 内启动，与 sink 同生命周期）
  const tapPumpRef = useRef<number | null>(null)
  const tailRef = useRef<string[]>([])
  const urlRef = useRef(opts.url ?? DEFAULT_URL)
  const ensuringRef = useRef(false)
  const serviceSnapshotRef = useRef({
    ready: false, error: '', stage: '',
  })
  // P1 有限次自动重连：服务就绪后的 WS 掉线是既有自愈盲区（3s 轮询只在
  // services.ready=false 期间跑）——服务无痕退场实测用户必须手点重连。
  // 非用户主动断开才按指数退避重试；服务不在了顺路走 Ensure 监护自愈。
  const reconnectTimerRef = useRef<number | null>(null)
  const reconnectAttemptRef = useRef(0)
  const deliberateCloseRef = useRef(false)
  const autoReconnectingRef = useRef(false)
  const scheduleReconnectRef = useRef<() => void>(() => {})
  urlRef.current = opts.url ?? DEFAULT_URL
  // 并发 connect 去重（2026-11-04 卡死根因①）：就绪事件/低频轮询/按钮回调三条
  // 自动连接路径会在同一 tick 各自调用 connect；connect 的引用 guard 要等
  // ensureSink 之后的同步段才置位，不先挡住=三个 WebSocket 同时建连，第一个
  // 抢到服务端会话槽（会话池=1）后被引用覆盖成孤儿——无人 close、浏览器 pong
  // 保活，服务端永远清不掉，后续所有连接被 session_limit_reached 弹回。
  const connectingRef = useRef(false)
  // 连接稳定窗定时器：稳定 RECONNECT_STABLE_MS 才清重连计数（见常量注）
  const reconnectStableTimerRef = useRef<number | null>(null)

  const pushTail = useCallback((line: string) => {
    const next = [...tailRef.current, line].slice(-14)
    tailRef.current = next
    if (mountedRef.current) setTail(next)
  }, [])

  const flushSink = useCallback(() => {
    // PlaybackQueue 只是调度面；worklet ring 里可能已有提前泵入的音频。
    sinkNodeRef.current?.port.postMessage({ type: 'flush' })
    playLevelRef.current = 0
  }, [])

  const onState = useCallback((s: VoiceSessionState) => {
    if (s.speechStartCount > speechStartRef.current) {
      speechStartRef.current = s.speechStartCount
      // barge-in 清 ring 仅在打断开启时执行（审计 P1，audio.py:361-367 同位——
      // interrupt 关时服务端刻意不取消响应，客户端掐播=断续；勿打断语义由
      // 用户显式点「打断」键行使）。speechStartCount 在 replayEvents 场景也增，
      // 与服务端事件一一对应。
      if (interruptEnabledRef.current) flushSink()
    }
    setPhase(s.phase)
    setTranscripts(s.committedTranscripts)
    setDialogue(voiceTranscriptMessages(s))
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
    // cache-bust：WebView2 对 public 静态 worklet 会启发式缓存，文件更新后仍加载
    // 旧版（2026-11-07 tap 链路排查一并破防）；每次建会话重新取，几 KB 开销可忽略
    await ctx.audioWorklet.addModule(`/voice-worklets/sinkProcessor.js?v=${Date.now()}`)
    const node = new AudioWorkletNode(ctx, 'gaea-voice-sink', { outputChannelCount: [2] })
    node.connect(ctx.destination)
    node.port.onmessage = (e) => {
      if (e.data && typeof e.data.overflow === 'number' && e.data.overflow > 0) {
        pushTail(`sink overflow ×${e.data.overflow}（上游泄漏信号）`)
      }
      // 整块拒绝上报（2026-11-07 修复 B）：泵记账下正常路径恒 0，出现=灌注
      // 面仍有过量（观测信号；worklet 侧已保证不逐样本顶替）
      if (e.data && e.data.type === 'rejected' && typeof e.data.samples === 'number') {
        pushTail(`sink 整块拒绝 ${e.data.samples} 样本（灌注过量信号）`)
      }
      // 水位报告（worklet ~10 次/秒）：泵灌注的权威 fill 校准；报告含了记账期
      // 已灌内容，本地记账随之清零（2026-11-07 修复 A）
      if (e.data && typeof e.data.buffered === 'number') {
        sinkFillRef.current = e.data.buffered
        postedSinceReportRef.current = 0
      }
      // 声卡实际外放电平（20Hz）：说话相位的星团/声纹以此为准，而不是
      // 「收到 delta 的时刻」——流式 TTS 常先收完整块再播，旧口径会提前
      // 起跳、播完又归零，看起来「不跟声音」。
      if (e.data && typeof e.data.__gaeaPlayLevel === 'number') {
        onPlayLevelThrottled(Math.max(0, Math.min(1, e.data.__gaeaPlayLevel)))
      }
      // 诊断 tap 批（≈1s/条）：入队待 POST 泵落盘，不碰 React
      if (e.data && e.data.type === 'tap' && e.data.buffer instanceof ArrayBuffer) {
        tapRateRef.current = typeof e.data.rate === 'number' ? e.data.rate : 16000
        tapQueueRef.current.push(e.data.buffer)
      }
    }
    sinkCtxRef.current = ctx
    sinkNodeRef.current = node
    if (ctx.sampleRate !== 16000) pushTail(`音频输出 ${ctx.sampleRate}Hz（16k source 自动重采样）`)
    // 高水位 8s（2026-11-07 修复 C，原 3s）：worklet ring 已扩 30s，水位线给
    // 报告迟到留足余量——报告迟到 1s 也不双灌（此前 3s 水位+报告迟到=重复灌
    // 3s→ring 超 4s cap→逐样本顶替=「语速加快+杂音」）。上游语义=无上限
    // 队列+声卡按需拉取，水位线只是搬运节流，延迟上界由它定。
    const highWater = Math.round(ctx.sampleRate * 8)
    let lastPulledId = ''
    let audioWasFlowing = false
    let drainedSent = true
    pumpRef.current = window.setInterval(() => {
      const client = clientRef.current
      const sink = sinkNodeRef.current
      if (!client || !sink) return
      // fill 估计=最近报告 + 报告迟到期间已灌量（修复 A：报告迟到时旧参考
      // 会让泵把同一水位「再灌一遍」——2026-11-07 tap 解析定罪的中段顶替源）
      let fill = sinkFillRef.current + postedSinceReportRef.current
      if (fill >= highWater) return // 已达水位：本 tick 整体跳过（报告迟到也不渗漏）
      for (let chunk = client.playback.pull(); chunk; chunk = client.playback.pull()) {
        // 交错探测器（2026-11-07「几段语音叠加」）：同一响应未闭合就拉出另一
        // 响应的块=双流并存，当即上屏点名（配合服务端交错告警归因）
        const st = client.playback.stats
        if (lastPulledId && st.currentResponseId !== lastPulledId && !client.isResponseClosed(lastPulledId)) {
          pushTail(`⚠ 音频流交错：${lastPulledId} 未闭合即切 ${st.currentResponseId}`)
        }
        lastPulledId = st.currentResponseId
        sink.port.postMessage({ type: 'pcm', buffer: chunk.buffer }, [chunk.buffer])
        const posted = chunk.length * (ctx.sampleRate / 16000)
        fill += posted
        postedSinceReportRef.current += posted
        if (fill >= highWater) break
      }
      const st = client.playback.stats
      // 播放排空回执：queue+ring 双空且刚播过 → 通知服务端关回声门
      // （真实播放结束锚点；服务端合成终态不再直接关门）。
      // 「静默」还要求服务端无未闭合响应（2026-11-07 实弹：句间合成间隙
      // queue/ring 瞬时空→误报 drained→回声门在响应中途开门 3s=回声入 VAD）
      const anyOpenResponse = Object.values(client.snapshot.responses).some((r) => !r.closed)
      const quiet = st.pendingChunkCount === 0 && sinkFillRef.current === 0 && !anyOpenResponse
      if (!quiet) {
        audioWasFlowing = true
        drainedSent = false
      } else if (audioWasFlowing && !drainedSent) {
        audioWasFlowing = false
        drainedSent = true
        client.sendPlaybackDrained()
      }
      setPbText(`${st.currentResponseId || '—'} 播${st.playedBytes}B 弃${st.droppedBytes}B 存${st.pendingChunkCount}块${st.playbackDone ? ' ✓回执' : ''}`)
    }, 100)
    // 诊断 tap POST 泵（500ms 一趟）：攒批实际外放样本落盘 8791 /v1/diag/playback。
    // 失败静默（诊断面不干扰主链路），只首错上屏一次。
    tapPumpRef.current = window.setInterval(() => {
      const batch = tapQueueRef.current
      if (batch.length === 0 || !tapFileRef.current) return
      // origin 提取（2026-11-07 实弹 404 教训）：urlRef 是完整 realtime URL 带
      // /v1/realtime 路径，replace(/^ws/,'http') 后直接拼路径=404 静默丢批
      const httpOrigin = new URL(urlRef.current ?? DEFAULT_URL).origin.replace(/^ws/, 'http')
      let data: ArrayBuffer
      if (batch.length === 1) {
        data = batch[0]
      } else {
        const total = batch.reduce((n, b) => n + b.byteLength, 0)
        const merged = new Uint8Array(total)
        let off = 0
        for (const b of batch) {
          merged.set(new Uint8Array(b), off)
          off += b.byteLength
        }
        data = merged.buffer
      }
      tapQueueRef.current = []
      const headers: Record<string, string> = {
        'Content-Type': 'application/octet-stream',
        'X-Gaea-Diag-File': tapFileRef.current,
        'X-Gaea-Diag-Rate': String(tapRateRef.current),
        'X-Gaea-Diag-Seq': String(tapSeqRef.current++),
      }
      if (tapTruncateRef.current) {
        headers['X-Gaea-Diag-Truncate'] = '1'
        tapTruncateRef.current = false
      }
      fetch(`${httpOrigin}/v1/diag/playback`, { method: 'POST', headers, body: data })
        .then((r) => {
          if (!r.ok && !tapNoteOnceRef.current) {
            tapNoteOnceRef.current = true
            pushTail(`播放诊断落盘失败（HTTP ${r.status}）——后续静默`)
          }
        })
        .catch(() => {
          if (!tapNoteOnceRef.current) {
            tapNoteOnceRef.current = true
            pushTail('播放诊断落盘不可达——后续静默')
          }
        })
    }, 500)
  }, [onPlayLevelThrottled, pushTail])

  const buildClient = useCallback((transport: VoiceTransport) => {
    return new VoiceClientV2({
      transport,
      instructions: optsRef.current.instructions,
      voice: optsRef.current.voice,
      // 监听阈值=silero 官方默认 0.5（2026-11-05 实弹：恒 0.75 把弱电平真麦全
      // discard——segment 820ms 里 active 仅 64~192ms，过不了 384ms 门）。回声
      // 防线（2026-11-01：0.6 被扬声器回声假插话）收敛进播放期回声门=后端
      // ttsStage 首音频 EchoGuardOn(0.75)/finishResponse EchoGuardOff。
      vadThreshold: 0.5,
      // 语音打断开关（A 拍板：界面切换钮）：初值取当前状态，重连/新会话沿用
      interruptResponse: interruptEnabledRef.current,
      onState,
      onPlaybackDone: (id) => {
        setPlayLevel(0)
        pushTail(`播放完成回执 ${id}`)
      },
      onPlayLevel: onPlayLevelThrottled,
      onOpen: () => {
        if (!mountedRef.current) return
        connectedRef.current = true
        setConnected(true)
        setNote('')
        pushTail('已连接')
        // 诊断 tap：新会话开新文件（truncate 首块）；文件标识本会话共用
        tapFileRef.current = Array.from(crypto.getRandomValues(new Uint8Array(8)))
          .map((b) => b.toString(16).padStart(2, '0'))
          .join('')
        tapTruncateRef.current = true
        tapSeqRef.current = 0
        // 重连成功：有意断开标记作废
        deliberateCloseRef.current = false
        if (reconnectTimerRef.current != null) {
          window.clearTimeout(reconnectTimerRef.current)
          reconnectTimerRef.current = null
        }
        // 重连计数不随 open 立即清零：闪断链路 open→秒断每次清零=退避链永不
        // 耗尽（2026-11-04 卡死根因③）。稳定保持 RECONNECT_STABLE_MS 才算恢复。
        if (reconnectStableTimerRef.current != null) {
          window.clearTimeout(reconnectStableTimerRef.current)
        }
        reconnectStableTimerRef.current = window.setTimeout(() => {
          reconnectStableTimerRef.current = null
          if (mountedRef.current && connectedRef.current) reconnectAttemptRef.current = 0
        }, RECONNECT_STABLE_MS)
      },
      onClose: () => {
        if (transportRef.current !== transport) return
        const client = clientRef.current
        transportRef.current = null
        clientRef.current = null
        connectedRef.current = false
        if (reconnectStableTimerRef.current != null) {
          window.clearTimeout(reconnectStableTimerRef.current)
          reconnectStableTimerRef.current = null
        }
        stopMicRef.current()
        client?.playback.reset()
        flushSink()
        if (!mountedRef.current) return
        setConnected(false)
        // P1 自动重连：有意断开（关台/手动断开）不重连；意外掉线走退避链
        if (deliberateCloseRef.current) {
          setNote('')
          pushTail('连接关闭')
          return
        }
        pushTail('连接关闭')
        scheduleReconnectRef.current()
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
  }, [flushSink, onPlayLevelThrottled, onState, pushTail])

  const connect = useCallback(async () => {
    if (clientRef.current || transportRef.current) return
    if (connectingRef.current) return
    connectingRef.current = true
    try {
      // 手动接管：清掉待触发的退避链；自动重连路径不重置计数（autoReconnecting
      // 由定时器回调置位），否则每次 connect 都把 attempt 清零=退避链永不耗尽。
      if (reconnectTimerRef.current != null) {
        window.clearTimeout(reconnectTimerRef.current)
        reconnectTimerRef.current = null
      }
      if (!autoReconnectingRef.current) reconnectAttemptRef.current = 0
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
      setDialogue([])
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
    } finally {
      connectingRef.current = false
    }
  }, [buildClient, ensureSink])

  const disconnect = useCallback(() => {
    // 有意断开：豁免自动重连（关台卸载与手动断开都走这里）
    deliberateCloseRef.current = true
    if (reconnectTimerRef.current != null) {
      window.clearTimeout(reconnectTimerRef.current)
      reconnectTimerRef.current = null
    }
    if (reconnectStableTimerRef.current != null) {
      window.clearTimeout(reconnectStableTimerRef.current)
      reconnectStableTimerRef.current = null
    }
    reconnectAttemptRef.current = 0
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
    if (micSilentTimerRef.current != null) {
      window.clearInterval(micSilentTimerRef.current)
      micSilentTimerRef.current = null
    }
    setMicLevel(0)
    micLevelRef.current = 0
    setMicSilent(false)
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
      await ctx.audioWorklet.addModule(`/voice-worklets/captureProcessor.js?v=${Date.now()}`)
      at('graph')
      const src = ctx.createMediaStreamSource(stream)
      const node = new AudioWorkletNode(ctx, 'gaea-voice-capture')
      node.port.onmessage = (e) => {
        const data: unknown = e.data
        if (data && typeof data === 'object' && '__gaeaLevel' in (data as Record<string, unknown>)) {
          const lv = Number((data as Record<string, unknown>).__gaeaLevel) || 0
          const prev = micLevelRef.current
          micLevelRef.current = lv
          if (lv > 0.02) lastSoundTsRef.current = Date.now()
          // 画布用 ref 50Hz 拿包络；React state 60ms 节流，避免整棵语音台
          // 每秒 50 次重渲染（长对话时渲染成本会随转写长度上升）。
          if (lv === 0 && prev === 0) return
          const now = Date.now()
          if (lv === 0 || now - micStateTsRef.current >= 60) {
            micStateTsRef.current = now
            setMicLevel(lv)
          }
          return
        }
        clientRef.current?.sendAudioChunk(new Int16Array(data as ArrayBuffer))
      }
      src.connect(node)
      at('state')
      capCtxRef.current = ctx
      capStreamRef.current = stream
      capNodeRef.current = node
      setMicOn(true)
      lastSoundTsRef.current = Date.now()
      setMicSilent(false)
      if (micSilentTimerRef.current != null) window.clearInterval(micSilentTimerRef.current)
      micSilentTimerRef.current = window.setInterval(() => {
        // 开麦 + 已连接 + 持续 3s 无信号 = 哑麦诚实提示（虚拟设备/选错设备/静音）
        setMicSilent(
          connectedRef.current && Date.now() - lastSoundTsRef.current > 3000,
        )
      }, 1000)
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

  // P0a-A5f（众神殿定稿 + 用户拍板「继续提交」）：关闭语音台时若存在未闭合
  // response（生成中），发 response.cancel 并调 VoiceV2AbortTurn 精确回滚
  // 后端预推进；已完成轮的 ledger 已 closed、提交任务已在链上，这里不触碰。
  const abortInFlight = useCallback(() => {
    const client = clientRef.current
    if (!client) return
    const snap = client.snapshot
    const led = snap.responses[snap.activeResponseId]
    if (!led || led.closed) return
    const itemId = led.userItemId
    client.cancel()
    void app.VoiceV2AbortTurn(itemId)
      .then((r) => pushTail(`关闭回滚 ${JSON.stringify(r)}`))
      .catch((err) => pushTail(`关闭回滚失败: ${err instanceof Error ? err.message : String(err)}`))
  }, [pushTail])

  const refreshServices = useCallback(async (): Promise<boolean> => {
    try {
      const st = (await app.VoiceV2Status()) as Record<string, unknown>
      const ready = st.ready === true
      const error = typeof st.error === 'string' ? st.error.trim() : ''
      const stage = typeof st.stage === 'string' ? st.stage.trim() : ''
      serviceSnapshotRef.current = { ready, error, stage }
      if (mountedRef.current) {
        setServices({
          realtime: st.realtime === true,
          breeze: st.breeze === true,
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
      // Go 原生引擎首次装配（模型加载）与 breeze 冷启实测可超过 90s；与后端
      // 硬超时同口径，保持进度可见；一旦后端回报终止错误立即停轮询，不能让
      // 依赖缺失的失败仍把按钮锁满三分钟。
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

  // P1 有限次自动重连（指数退避 1s→15s 封顶，最多 5 次）。经 ref 间接调用
  // 解循环依赖：buildClient.onClose → scheduleReconnect → connect → buildClient。
  // 服务探活失败顺路走 ensureServices（Ensure 幂等，与开台按钮同一监护自愈路径）。
  const scheduleReconnect = useCallback(() => {
    if (reconnectTimerRef.current != null) return
    if (!mountedRef.current || deliberateCloseRef.current) return
    const attempt = reconnectAttemptRef.current + 1
    if (attempt > RECONNECT_MAX_ATTEMPTS) {
      setNote('连接已断开，自动重连未成功，可点击重连')
      pushTail(`自动重连 ${RECONNECT_MAX_ATTEMPTS} 次未成功，可手动重连`)
      return
    }
    reconnectAttemptRef.current = attempt
    const delay = Math.min(RECONNECT_BASE_MS * 2 ** (attempt - 1), RECONNECT_MAX_DELAY_MS)
    setNote(`连接已断开，正在自动重连（${attempt}/${RECONNECT_MAX_ATTEMPTS}）…`)
    pushTail(`连接断开，${Math.round(delay / 1000)}s 后自动重连（${attempt}/${RECONNECT_MAX_ATTEMPTS}）`)
    reconnectTimerRef.current = window.setTimeout(() => {
      reconnectTimerRef.current = null
      if (!mountedRef.current || deliberateCloseRef.current) return
      void (async () => {
        autoReconnectingRef.current = true
        try {
          let ready = await refreshServices()
          if (!ready) ready = await ensureServices()
          if (!ready) {
            scheduleReconnect()
            return
          }
          await connect()
          // transport 没建出来（异常早退、桥不可用）且无人接管退避链——续上
          if (!transportRef.current && !clientRef.current && reconnectTimerRef.current == null) {
            scheduleReconnect()
          }
        } catch {
          scheduleReconnect()
        } finally {
          autoReconnectingRef.current = false
        }
      })()
    }, delay)
  }, [connect, ensureServices, pushTail, refreshServices])
  scheduleReconnectRef.current = scheduleReconnect

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
      if (tapPumpRef.current) window.clearInterval(tapPumpRef.current)
      if (reconnectTimerRef.current != null) {
        window.clearTimeout(reconnectTimerRef.current)
        reconnectTimerRef.current = null
      }
      if (reconnectStableTimerRef.current != null) {
        window.clearTimeout(reconnectStableTimerRef.current)
        reconnectStableTimerRef.current = null
      }
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

  const updateVADThreshold = useCallback((threshold: number) => {
    clientRef.current?.updateVADThreshold(threshold)
  }, [])

  /** 语音打断开关：界面切换钮下发 session.update（连接中即时生效；断线存值待重连沿用） */
  const setInterruptEnabled = useCallback((v: boolean) => {
    interruptEnabledRef.current = v
    setInterruptEnabledState(v)
    clientRef.current?.setInterruptResponse(v)
  }, [])

  return {
    phase, transcripts, dialogue, partialTranscript: partial, assistantText,
    pbText, tail, connected, micOn, micLevel, playLevel, micLevelRef, playLevelRef,
    micSilent, note, services, ensuring,
    connect, disconnect, startMic, stopMic, cancel, abortInFlight, updateVADThreshold,
    ensureServices, refreshServices, replayEvents,
    interruptEnabled, setInterruptEnabled,
  }
}
