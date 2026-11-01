import { useState, useRef, useCallback, useEffect } from 'react'
import { app } from '../gaea/lib/bridge'
import { EventsOn } from '../../wailsjs/runtime/runtime'
// W1 b64 收口：壳内事件 []byte 以 base64 字符串传输，解码一律走中立层
import { b64ToBytes } from '../gaea/lib/bytes'

// ── 状态类型（兼容现有 VoiceChatOrb） ──

export interface VoiceChatState {
  active: boolean
  listening: boolean
  speaking: boolean
  aiSpeaking: boolean
  transcript: string
  finalTranscript: string
  volume: number
  error: string | null
  mode: 'vad' | 'ptt' | 'off'
  /** 显式降级位（FE6-02）：麦克风不可用时置 'mic-unavailable'——
   *  绝不伪造成「正在聆听」，消费方据此渲染可见警示条。 */
  degraded: 'mic-unavailable' | null
  /** 零信号自动换真麦：降级时探测到的可用设备（一键切换建议，切换后清空） */
  suggestedMic: { deviceId: string; label: string } | null
}

interface Options {
  /** 收到最终识别文本（用于添加用户消息） */
  onTranscript?: (text: string) => void
  /** 收到 AI 回复文本（用于添加 AI 消息） */
  onReply?: (text: string) => void
  /** 收到 AI 回复增量（用于首字即显示） */
  onReplyDelta?: (text: string) => void
}

// ── 事件载荷最小类型（后端 voice:* 事件动态 JSON） ──
interface VoiceStateEvent { state: string }
interface VoiceTranscriptEvent { text?: string; isFinal?: boolean }
interface VoiceReplyEvent { text?: string }
interface VoiceTTSAudioEvent { audio?: string | Uint8Array | ArrayLike<number>; mimeType?: string }
interface VoiceActiveEvent { active?: boolean }

// ── Web Speech API 最小接口（lib.dom 未内置 SpeechRecognition） ──
interface SpeechRecognitionResultItem {
  isFinal: boolean
  0: { transcript: string }
}
interface SpeechRecognitionResultEventLike {
  resultIndex: number
  results: ArrayLike<SpeechRecognitionResultItem | null>
}
interface SpeechRecognitionErrorEventLike {
  error?: string
}
interface SpeechRecognitionLike {
  lang: string
  continuous: boolean
  interimResults: boolean
  maxAlternatives: number
  start(): void
  stop(): void
  onresult: ((e: SpeechRecognitionResultEventLike) => void) | null
  onerror: ((e: SpeechRecognitionErrorEventLike) => void) | null
  onend: (() => void) | null
}

/** 取浏览器 SpeechRecognition 构造器（标准/WebKit 前缀，缺失返回 undefined） */
function speechRecognitionCtor(): (new () => SpeechRecognitionLike) | undefined {
  if (typeof window === 'undefined') return undefined
  const w = window as unknown as {
    SpeechRecognition?: new () => SpeechRecognitionLike
    webkitSpeechRecognition?: new () => SpeechRecognitionLike
  }
  return w.SpeechRecognition || w.webkitSpeechRecognition
}

// ── 音频常量（对齐后端 voice_config.go） ──

const SAMPLE_RATE = 16000

// ScriptProcessor bufferSize 只接受 2 的幂（256~16384）。目标 200ms@16k=3200
// 样本不是幂——createScriptProcessor(3200) 同步抛 IndexSizeError（2026-11-01
// 实弹真凶：麦克风就绪后 getUserMedia 秒成，此异常把 startCapture 炸回 false，
// 语音管道 100~160ms 秒停）。取 4096（≈256ms@16k）；块只是流式分片，后端
// PushAudioChunk 对块大小无契约。
const PROCESSOR_BUFFER = 4096

// 浏览器端自带语音识别（WebView2 / Edge 内核，微软云端识别），
// 免去后端 ASR 模型，识别更快。
const browserASRAvailable = !!speechRecognitionCtor()

// ── 麦克风设备选择（FE6-02 终局配套） ──
// 实弹实证：本机默认录音设备可能是零信号虚拟设备（ToDesk Virtual Audio），
// getUserMedia 默认拿到的就是它——没有设备选择，真麦插上也听不见。
// deviceId 是 WebView2 按源的持久指纹，纯前端采集关注点，localStorage 持久化
// （不进后端 config——后端不做采集，为浏览器指纹 ID 开五文件配账不值）。

const MIC_DEVICE_KEY = 'gaea.voice.micDeviceId'

/** 读取已选麦克风设备 ID（空 = 跟随系统默认） */
export function loadMicDeviceId(): string {
  try { return localStorage.getItem(MIC_DEVICE_KEY) || '' } catch { return '' }
}

/** 保存麦克风设备 ID（空串 = 清除，恢复跟随系统默认） */
export function saveMicDeviceId(id: string): void {
  try {
    if (id) localStorage.setItem(MIC_DEVICE_KEY, id)
    else localStorage.removeItem(MIC_DEVICE_KEY)
  } catch { /* localStorage 不可用（隐私模式）：仅本会话内生效 */ }
}

export interface MicDeviceInfo {
  deviceId: string
  label: string
  isDefault: boolean
}

/** 列出录音输入设备。权限未授予时 label 为空——借一次瞬时采集换取可读标签
 *  （失败静默，设备仍可列只是名字叫「麦克风」）。 */
export async function listMicDevices(): Promise<MicDeviceInfo[]> {
  if (!navigator.mediaDevices?.enumerateDevices) return []
  try {
    let devs = await navigator.mediaDevices.enumerateDevices()
    if (devs.filter(d => d.kind === 'audioinput').some(d => !d.label)) {
      try {
        const s = await navigator.mediaDevices.getUserMedia({ audio: true })
        s.getTracks().forEach(t => t.stop())
        devs = await navigator.mediaDevices.enumerateDevices()
      } catch { /* 权限拒绝：按匿名设备列出 */ }
    }
    return devs
      .filter(d => d.kind === 'audioinput')
      .map(d => ({
        deviceId: d.deviceId,
        label: d.label || '麦克风',
        isDefault: d.deviceId === 'default',
      }))
  } catch {
    return []
  }
}

/** 探测有真实信号的录音设备（零信号自动换真麦）：逐台开 500ms 测频域能量，
 *  返回按信号强度降序的候选。跳过 default/communications 别名口（指向同一
 *  物理设备）与当前设备；单台探测失败静默跳过。 */
export async function probeMicDevicesForSignal(
  excludeDeviceId?: string,
): Promise<Array<MicDeviceInfo & { strength: number }>> {
  if (!navigator.mediaDevices?.enumerateDevices) return []
  try {
    const devs = (await navigator.mediaDevices.enumerateDevices()).filter(d => d.kind === 'audioinput')
    const out: Array<MicDeviceInfo & { strength: number }> = []
    for (const d of devs) {
      if (d.deviceId === 'default' || d.deviceId === 'communications') continue
      if (excludeDeviceId && d.deviceId === excludeDeviceId) continue
      let strength = 0
      try {
        const s = await navigator.mediaDevices.getUserMedia({ audio: { deviceId: { exact: d.deviceId } } })
        const ctx = new AudioContext()
        const src = ctx.createMediaStreamSource(s)
        const analyser = ctx.createAnalyser()
        analyser.fftSize = 256
        src.connect(analyser)
        const buf = new Uint8Array(analyser.frequencyBinCount)
        const deadline = Date.now() + 500
        while (Date.now() < deadline) {
          analyser.getByteFrequencyData(buf)
          for (let i = 0; i < buf.length; i++) {
            if (buf[i] > strength) strength = buf[i]
          }
          await new Promise(r => setTimeout(r, 50))
        }
        src.disconnect()
        await ctx.close()
        s.getTracks().forEach(t => t.stop())
      } catch { /* 单台失败跳过（占用/权限） */ }
      if (strength > 0) {
        out.push({ deviceId: d.deviceId, label: d.label || '麦克风', isDefault: false, strength })
      }
    }
    return out.sort((a, b) => b.strength - a.strength)
  } catch {
    return []
  }
}

export function useVoiceChat({ onTranscript, onReply, onReplyDelta }: Options = {}) {
  const [state, setState] = useState<VoiceChatState>({
    active: false, listening: false, speaking: false, aiSpeaking: false,
    transcript: '', finalTranscript: '', volume: 0, error: null, mode: 'vad',
    degraded: null, suggestedMic: null,
  })

  // ── Refs ──
  const captureCtxRef = useRef<AudioContext | null>(null)
  const playbackCtxRef = useRef<AudioContext | null>(null)
  const streamRef = useRef<MediaStream | null>(null)
  const processorRef = useRef<ScriptProcessorNode | null>(null)
  const gainRef = useRef<GainNode | null>(null)
  const abortRef = useRef(false)
  const pttRef = useRef(false)
  const recognitionRef = useRef<SpeechRecognitionLike | null>(null)
  const recognitionActiveRef = useRef(false)
  const pendingSpeechRef = useRef(0)
  const volSmoothRef = useRef(0)
  const volTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const stateRef = useRef<VoiceChatState>(state)
  // S2 realtime 模式（后端注入了 Realtime 会话 = realtimeProvider 非空）：
  // 强制推送 PCM + 抑制本地识别路径（防双输入）。start 时经 VoiceGetSettings 刷新。
  const realtimeModeRef = useRef(false)
  // 零信号自动换真麦：一次会话只探测一轮（防降级↔探测循环）
  const micProbeRef = useRef(false)
  // 每次取消播报递增：decode/play 的在途 Promise 与旧 onended 只能收尾
  // 自己所属的一代，不能污染下一轮播放状态。
  const playbackEpochRef = useRef(0)
  const fallbackAudioRef = useRef(new Map<HTMLAudioElement, string>())
  const recognitionRestartTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const recognitionRestartDelayRef = useRef(150)

  // 保持 stateRef 同步
  stateRef.current = state

  const setState2 = useCallback((patch: Partial<VoiceChatState> | ((s: VoiceChatState) => VoiceChatState)) => {
    setState(prev => {
      const next = typeof patch === 'function' ? patch(prev) : { ...prev, ...patch }
      stateRef.current = next
      return next
    })
  }, [])

  // ── 音频播放 ──

  const playAudio = useCallback(async (
    audioData: string | Uint8Array | ArrayLike<number>,
    mimeType: string,
    epoch: number,
  ) => {
    const isCurrent = () => !abortRef.current && playbackEpochRef.current === epoch
    let completed = false
    const complete = () => {
      if (completed) return
      completed = true
      if (!isCurrent()) return
      pendingSpeechRef.current = Math.max(0, pendingSpeechRef.current - 1)
      if (pendingSpeechRef.current === 0) setState2({ aiSpeaking: false })
      app.VoicePlaybackDone().catch(() => {})
    }

    if (!audioData) {
      complete()
      return
    }

    if (!playbackCtxRef.current || playbackCtxRef.current.state === 'closed') {
      playbackCtxRef.current = new AudioContext()
      gainRef.current = playbackCtxRef.current.createGain()
      gainRef.current.connect(playbackCtxRef.current.destination)
      gainRef.current.gain.value = 1.0
    }

    const ctx = playbackCtxRef.current
    if (ctx.state === 'suspended') await ctx.resume()
    if (!isCurrent()) return

    // 转换数据：Wails 事件中 []byte 以 base64 字符串传输（对齐 TTSPlayer b64ToBytes 解码）
    let bytes: Uint8Array
    if (typeof audioData === 'string') {
      bytes = b64ToBytes(audioData)
    } else if (audioData instanceof Uint8Array) {
      bytes = audioData
    } else {
      bytes = new Uint8Array(audioData)
    }
    if (bytes.length === 0) {
      complete()
      return
    }
    const buffer = bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.length) as ArrayBuffer

    try {
      const audioBuffer = await ctx.decodeAudioData(buffer)
      if (!isCurrent()) return
      const source = ctx.createBufferSource()
      source.buffer = audioBuffer
      source.connect(gainRef.current!)
      source.onended = complete
      source.start(0)
    } catch {
      // decodeAudioData 可能失败（非浏览器原生格式），尝试用 Audio 元素。
      // 取消可能发生在 decode 等待期间：此时不得启动一个迟到的旧音频。
      if (!isCurrent()) return
      const blob = new Blob([bytes as BlobPart], { type: mimeType || 'audio/mp3' })
      const url = URL.createObjectURL(blob)
      const audio = new Audio(url)
      fallbackAudioRef.current.set(audio, url)
      const release = () => {
        audio.onended = null
        audio.onerror = null
        try { audio.pause() } catch (_) {}
        try { audio.removeAttribute('src') } catch (_) {}
        try { audio.load() } catch (_) {}
        fallbackAudioRef.current.delete(audio)
        URL.revokeObjectURL(url)
      }
      const finishFallback = () => {
        release()
        complete()
      }
      audio.onended = finishFallback
      audio.onerror = finishFallback
      try {
        await audio.play()
      } catch {
        // 播放被阻止等失败：兜底释放状态机，避免后端一直等待播放完成。
        finishFallback()
      }
    }
  }, [setState2])

  const stopPlayback = useCallback(() => {
    // 先换代再关闭资源：所有旧 Promise/onended 都会因 epoch 不符而静默退出。
    playbackEpochRef.current += 1
    pendingSpeechRef.current = 0
    setState2({ aiSpeaking: false })
    if (playbackCtxRef.current && playbackCtxRef.current.state !== 'closed') {
      playbackCtxRef.current.close().catch(() => {})
    }
    playbackCtxRef.current = null
    gainRef.current = null
    for (const [audio, url] of fallbackAudioRef.current) {
      audio.onended = null
      audio.onerror = null
      try { audio.pause() } catch (_) {}
      try { audio.removeAttribute('src') } catch (_) {}
      try { audio.load() } catch (_) {}
      URL.revokeObjectURL(url)
    }
    fallbackAudioRef.current.clear()
    if (typeof speechSynthesis !== 'undefined') speechSynthesis.cancel()
  }, [setState2])

  const cancelPlayback = useCallback(() => {
    stopPlayback()
    app.VoiceCancelTTS().catch(() => {})
  }, [stopPlayback])

  // ── 事件监听 ──

  useEffect(() => {
    const unsubs: (() => void)[] = []

    // 状态变更
    unsubs.push(EventsOn('voice:state', (data: VoiceStateEvent) => {
      if (abortRef.current) return
      const s = data.state
      setState2({
        listening: s === 'listening',
        aiSpeaking: s === 'thinking' || s === 'speaking',
        speaking: s === 'speaking',
      })
    }))

    // 识别结果
    unsubs.push(EventsOn('voice:transcript', (data: VoiceTranscriptEvent) => {
      if (abortRef.current) return
      const text = data.text || ''
      const isFinal = data.isFinal ?? false
      setState2(s => ({
        ...s,
        transcript: text,
        finalTranscript: isFinal ? s.finalTranscript + text : s.finalTranscript,
      }))
      if (isFinal && text) {
        // 只发当前识别片段，避免累积拼接导致旧句子重复入聊天
        onTranscript?.(text)
      }
    }))

    // AI 回复（文本）
    unsubs.push(EventsOn('voice:reply', (data: VoiceReplyEvent) => {
      if (abortRef.current) return
      const text = data.text || ''
      if (text) onReply?.(text)
    }))

    unsubs.push(EventsOn('voice:reply-delta', (data: VoiceReplyEvent) => {
      if (abortRef.current) return
      const text = data.text || ''
      if (text) onReplyDelta?.(text)
    }))

    // TTS 音频播放
    unsubs.push(EventsOn('voice:tts-audio', async (data: VoiceTTSAudioEvent) => {
      if (abortRef.current) return
      const epoch = playbackEpochRef.current
      pendingSpeechRef.current += 1
      setState2({ aiSpeaking: true })
      try {
        await playAudio(
          data.audio as string | Uint8Array | ArrayLike<number>,
          data.mimeType || 'audio/mp3',
          epoch,
        )
      } catch (err) {
        console.error('[Voice] TTS 播放失败:', err)
        if (abortRef.current || playbackEpochRef.current !== epoch) return
        pendingSpeechRef.current = Math.max(0, pendingSpeechRef.current - 1)
        if (pendingSpeechRef.current === 0) setState2({ aiSpeaking: false })
        app.VoicePlaybackDone().catch(() => {})
      }
    }))

    // 浏览器 TTS fallback
    unsubs.push(EventsOn('voice:tts-speak-text', (data: VoiceReplyEvent) => {
      if (abortRef.current) return
      const text = data.text || ''
      if (text && 'speechSynthesis' in window && typeof SpeechSynthesisUtterance !== 'undefined') {
        const u = new SpeechSynthesisUtterance(text)
        u.lang = 'zh-CN'
        u.rate = 1.0
        const epoch = playbackEpochRef.current
        pendingSpeechRef.current += 1
        setState2({ aiSpeaking: true })
        u.onend = () => {
          if (abortRef.current || playbackEpochRef.current !== epoch) return
          pendingSpeechRef.current = Math.max(0, pendingSpeechRef.current - 1)
          if (pendingSpeechRef.current === 0) setState2({ aiSpeaking: false })
          app.VoicePlaybackDone().catch(() => {})
        }
        u.onerror = () => {
          if (abortRef.current || playbackEpochRef.current !== epoch) return
          pendingSpeechRef.current = Math.max(0, pendingSpeechRef.current - 1)
          if (pendingSpeechRef.current === 0) setState2({ aiSpeaking: false })
          app.VoicePlaybackDone().catch(() => {})
        }
        // 浏览器 TTS 自身按队列播放；逐句 fallback 时不能在每次 speak 前
        // cancel，否则后一句会掐掉前一句。只有打断/结束才统一 cancel。
        speechSynthesis.speak(u)
      }
    }))

    // TTS 取消
    unsubs.push(EventsOn('voice:tts-speak-cancel', () => {
      stopPlayback()
    }))

    // 监听指示
    unsubs.push(EventsOn('voice:listening', (data: VoiceActiveEvent) => {
      if (abortRef.current) return
      setState2({ listening: data.active ?? false })
    }))

    // 思考指示
    unsubs.push(EventsOn('voice:thinking', (data: VoiceActiveEvent) => {
      if (abortRef.current) return
      setState2({ aiSpeaking: data.active ?? false })
    }))

    return () => {
      unsubs.forEach(fn => { try { fn() } catch (_) {} })
    }
  }, [onTranscript, onReply, onReplyDelta, setState2, playAudio, stopPlayback])

  // ── S2 realtime 模式判定 ──

  // realtimeProvider 非空 = 后端已注入 Realtime 会话（端到端模式）。
  // 读取失败（后端未就绪/mock 环境）按非 realtime 处理，行为与现状一致。
  const refreshRealtimeMode = useCallback(async () => {
    try {
      const settings = (await app.VoiceGetSettings()) as { realtimeProvider?: string } | undefined
      realtimeModeRef.current = !!settings?.realtimeProvider
    } catch {
      realtimeModeRef.current = false
    }
  }, [])

  // ── 浏览器端语音识别（Web Speech API，WebView2 自带） ──
  // （定义在 startCapture 之前：中途失麦检测需在 track ended 时停转识别器）

  const startBrowserRecognition = useCallback((): boolean => {
    if (!browserASRAvailable) return false
    const SR = speechRecognitionCtor()
    if (!SR) return false
    if (recognitionActiveRef.current) return true

    let rec: SpeechRecognitionLike
    try {
      rec = new SR()
    } catch (err: unknown) {
      const detail = err instanceof Error ? err.message : String(err)
      setState2({ error: `语音识别初始化失败：${detail}` })
      return false
    }
    recognitionActiveRef.current = true
    recognitionRef.current = rec
    if (recognitionRestartTimerRef.current) {
      clearTimeout(recognitionRestartTimerRef.current)
      recognitionRestartTimerRef.current = null
    }
    rec.lang = 'zh-CN'
    rec.continuous = true
    rec.interimResults = true
    rec.maxAlternatives = 1

    let scheduleReconnect: () => void
    scheduleReconnect = () => {
      if (abortRef.current || !recognitionActiveRef.current || recognitionRestartTimerRef.current) return
      recognitionRestartTimerRef.current = setTimeout(() => {
        recognitionRestartTimerRef.current = null
        if (abortRef.current || !recognitionActiveRef.current || recognitionRef.current !== rec) return
        try {
          rec.start()
        } catch (err: unknown) {
          const detail = err instanceof Error ? err.message : String(err)
          console.warn('[Voice] 浏览器识别重连失败:', detail)
          recognitionRestartDelayRef.current = Math.min(
            3000,
            recognitionRestartDelayRef.current * 2,
          )
          scheduleReconnect()
        }
      }, recognitionRestartDelayRef.current)
    }

    rec.onresult = (e: SpeechRecognitionResultEventLike) => {
      if (abortRef.current) return
      let interim = ''
      let finalText = ''
      for (let i = e.resultIndex; i < e.results.length; i++) {
        const res = e.results[i]
        if (!res) continue
        const text = (res[0]?.transcript || '').trim()
        if (!text) continue
        if (res.isFinal) {
          finalText += text
        } else {
          interim += text
        }
      }
      recognitionRestartDelayRef.current = 150
      setState2(s => ({
        ...s,
        transcript: interim,
        finalTranscript: finalText ? s.finalTranscript + finalText : s.finalTranscript,
        error: null,
      }))

      // Web Speech 只把最终文本送后端；若等 final 才打断，AI 可能已多念
      // 好几秒。检测到用户在 AI 思考/说话时开口，就用 interim 先行停播；
      // 最终文本随后照常进入同一条对话管道。
      if ((interim || finalText) && stateRef.current.aiSpeaking) {
        cancelPlayback()
      }
      if (finalText) {
        // 直接进入后端对话管道（跳过 ASR 模型，识别更快）
        app.VoiceChatText(finalText).catch((err: unknown) => {
          const detail = err instanceof Error ? err.message : String(err)
          setState2({ error: `语音消息发送失败：${detail}` })
        })
      }
    }

    rec.onerror = (e: SpeechRecognitionErrorEventLike) => {
      console.warn('[Voice] 浏览器识别错误:', e?.error)
      if (e?.error === 'not-allowed' || e?.error === 'service-not-allowed') {
        // 识别器侧权限被拒（同采集路径的撒谎式降级）：显式降级，警示条可见
        recognitionActiveRef.current = false
        if (recognitionRestartTimerRef.current) {
          clearTimeout(recognitionRestartTimerRef.current)
          recognitionRestartTimerRef.current = null
        }
        setState2({ degraded: 'mic-unavailable', error: '语音识别被拒绝（麦克风权限或服务不可用）' })
      } else if (e?.error === 'audio-capture') {
        // 识别器侧采集失败（设备中途消失/被独占）：阻断 onend 自动重连，
        // 与采集路径同口径降级——静音/断麦绝不冒充「聆听」。
        recognitionActiveRef.current = false
        if (recognitionRestartTimerRef.current) {
          clearTimeout(recognitionRestartTimerRef.current)
          recognitionRestartTimerRef.current = null
        }
        setState2({ degraded: 'mic-unavailable' })
      } else if (e?.error === 'network') {
        // 云端识别短暂断网不等于麦克风坏；指数退避重连，并明确告知用户。
        recognitionRestartDelayRef.current = Math.min(
          3000,
          Math.max(500, recognitionRestartDelayRef.current * 2),
        )
        setState2({ error: '语音识别网络中断，正在重连…' })
        scheduleReconnect()
      } else if (e?.error !== 'no-speech' && e?.error !== 'aborted') {
        setState2({ error: `语音识别异常：${e?.error || '未知错误'}` })
      }
    }

    rec.onend = () => {
      // continuous 模式在静音后可能自行结束，自动重连保持聆听
      scheduleReconnect()
    }

    try {
      rec.start()
      return true
    } catch (err: unknown) {
      recognitionActiveRef.current = false
      recognitionRef.current = null
      const detail = err instanceof Error ? err.message : String(err)
      setState2({ error: `语音识别启动失败：${detail}` })
      return false
    }
  }, [cancelPlayback, setState2])

  const stopBrowserRecognition = useCallback(() => {
    recognitionActiveRef.current = false
    if (recognitionRestartTimerRef.current) {
      clearTimeout(recognitionRestartTimerRef.current)
      recognitionRestartTimerRef.current = null
    }
    if (recognitionRef.current) {
      try { recognitionRef.current.stop() } catch (_) {}
      recognitionRef.current = null
    }
  }, [])

  const stopCapture = useCallback(() => {
    stopBrowserRecognition()
    volSmoothRef.current = 0
    if (volTimerRef.current) { clearInterval(volTimerRef.current); volTimerRef.current = null }
    if (processorRef.current) {
      try { processorRef.current.disconnect() } catch (_) {}
      processorRef.current = null
    }
    if (captureCtxRef.current) {
      captureCtxRef.current.close().catch(() => {})
      captureCtxRef.current = null
    }
    if (streamRef.current) {
      streamRef.current.getTracks().forEach(t => t.stop())
      streamRef.current = null
    }
  }, [stopBrowserRecognition])

  // ── 音频采集 ──

  /** 启动本地采集。返回是否真正采到音频（FE6-02 复审）：调用方据此决定要不要
   *  起本地识别，不依赖 React eager state 的时序（stateRef 在 updater 内赋值）。 */
  const startCapture = useCallback(async (): Promise<boolean> => {
    // 采集约束阶梯（实弹取证）：① 全约束（已选麦克风 exact + EC/NS）——
    // 默认设备可能是零信号虚拟设备，用户选过真麦就必须用真麦；② 同款重试
    // 一次（蓝牙 hands-free 档位切换窗口的瞬态 NotReadableError/AbortError）；
    // ③ 去 sampleRate——Chromium 在 BT HFP/虚拟设备混路上对 sampleRate 约束
    // 会抛非确定性 IndexSizeError，而 16k 重采样由 AudioContext 强制（约束
    // 本身冗余）；④ 裸 {audio:true}——实测百发百中的最后底线。
    const buildAudioConstraints = (withSampleRate: boolean): MediaTrackConstraints => {
      const savedMicId = loadMicDeviceId()
      return {
        ...(savedMicId ? { deviceId: { exact: savedMicId } } : {}),
        ...(withSampleRate ? { sampleRate: SAMPLE_RATE } : {}),
        channelCount: 1,
        echoCancellation: true,
        noiseSuppression: true,
      }
    }
    const ladder: Array<MediaTrackConstraints | true> = [
      buildAudioConstraints(true),
      buildAudioConstraints(true),
      buildAudioConstraints(false),
      true,
    ]
    // 全档全败才降级：可恢复失败不当永久不可用。
    const openStream = async (): Promise<MediaStream> => {
      let lastErr: unknown
      for (let i = 0; i < ladder.length; i++) {
        if (abortRef.current) throw lastErr ?? new DOMException('aborted', 'AbortError')
        try {
          const constraints = ladder[i]
          return await navigator.mediaDevices.getUserMedia({
            audio: constraints === true ? true : constraints,
          })
        } catch (err) {
          lastErr = err
          const name = (err as DOMException)?.name || ''
          console.warn(`[Voice] 采集失败（${name}，档位 ${i + 1}/${ladder.length}），降档重试`)
          if (i < ladder.length - 1) {
            await new Promise(r => setTimeout(r, 500))
          }
        }
      }
      throw lastErr
    }
    try {
      const stream = await openStream()
      if (abortRef.current) { stream.getTracks().forEach(t => t.stop()); return false }

      streamRef.current = stream

      // FE6-02 补课：中途失麦检测。采集成功 ≠ 一直成功——设备拔出/被其他
      // 应用独占（track ended）或被系统隐私静音（track mute）后，采集静默
      // 死亡而 UI 仍按旧状态显示「聆听」。显式降级 + 音量归零（unmute 自愈
      // 恢复），断麦/静音绝不冒充「正在聆听/准备聆听」。
      const track = stream.getAudioTracks()[0] ?? null
      if (track) {
        track.addEventListener('ended', () => {
          if (abortRef.current) return
          console.warn('[Voice] 麦克风已断开（设备移除或被独占），采集停止')
          stopBrowserRecognition()
          setState2({ degraded: 'mic-unavailable', volume: 0, speaking: false })
        })
        track.addEventListener('mute', () => {
          if (abortRef.current) return
          console.warn('[Voice] 麦克风被系统静音，采集暂停')
          setState2({ degraded: 'mic-unavailable', volume: 0, speaking: false })
        })
        track.addEventListener('unmute', () => {
          if (abortRef.current || streamRef.current?.getAudioTracks()[0] !== track) return
          setState2({ degraded: null })
          if (browserASRAvailable && !realtimeModeRef.current && !recognitionActiveRef.current) {
            startBrowserRecognition()
          }
        })
      }

      // AudioContext 用于采集（对齐 Ackem captureContextRef）
      captureCtxRef.current = new AudioContext({ sampleRate: SAMPLE_RATE })
      const source = captureCtxRef.current.createMediaStreamSource(stream)

      // AnalyserNode 用于音量可视化
      const analyser = captureCtxRef.current.createAnalyser()
      analyser.fftSize = 256
      analyser.smoothingTimeConstant = 0.3
      source.connect(analyser)

      const freqBuf = new Uint8Array(analyser.frequencyBinCount)
      // FE6-02 终局（实弹实证）：系统默认录音设备可能是虚拟音频（如 ToDesk
      // Virtual Audio）——getUserMedia 成功、track.muted=false，却全程零样本。
      // 旧 UI 此时永远「准备聆听/正在聆听」而实际一个字都没采到。连续 5s 频域
      // 全零 = 未在采集真实声音 → 显式降级警示；一旦出现真实信号即自愈恢复。
      let lastSignalAt = Date.now()
      volTimerRef.current = setInterval(() => {
        analyser.getByteFrequencyData(freqBuf)
        const avg = freqBuf.reduce((a, b) => a + b, 0) / freqBuf.length
        // 平滑 + 较高阈值：过滤环境杂音导致的"正在聆听"误亮
        const raw = avg / 128
        volSmoothRef.current = volSmoothRef.current * 0.7 + raw * 0.3
        const vol = Math.min(volSmoothRef.current, 1)
        let hasSignal = false
        for (let i = 0; i < freqBuf.length; i++) {
          if (freqBuf[i] > 0) { hasSignal = true; break }
        }
        const degradedNow = stateRef.current.degraded === 'mic-unavailable'
        if (hasSignal) {
          lastSignalAt = Date.now()
          if (degradedNow) setState2({ degraded: null }) // 麦克风活了：自愈恢复
        } else if (!degradedNow && Date.now() - lastSignalAt > 5000) {
          console.warn('[Voice] 麦克风连续 5 秒零输入（虚拟设备/硬件静音），降级为不可用')
          setState2({ degraded: 'mic-unavailable', speaking: false })
          // 零信号自动换真麦：一次性探测其余设备，发现信号源给出一键切换建议
          if (!micProbeRef.current) {
            micProbeRef.current = true
            const currentId = streamRef.current?.getAudioTracks()[0]?.getSettings?.().deviceId || ''
            void probeMicDevicesForSignal(currentId || undefined).then(cands => {
              if (abortRef.current || cands.length === 0) return
              const best = cands[0]
              console.info('[Voice] 探测到有信号的麦克风，建议切换', best.label, 'strength=', best.strength)
              setState2({ suggestedMic: { deviceId: best.deviceId, label: best.label } })
            })
          }
        }
        setState2({ volume: vol, speaking: vol > 0.12 })
      }, 80)

      // ScriptProcessorNode 用于 PCM 采集（对齐 Ackem；bufferSize 必须 2 的幂，见常量注释）
      const processor = captureCtxRef.current.createScriptProcessor(PROCESSOR_BUFFER, 1, 1)
      processorRef.current = processor
      source.connect(processor)
      processor.connect(captureCtxRef.current.destination) // 必须连接才能触发

      processor.onaudioprocess = (event) => {
        if (abortRef.current) return
        // 仅在 PTT 激活或 VAD 模式下发送；浏览器识别模式下不上送后端 ASR。
        // S2 realtime 模式强制推送（旁路 browserASRAvailable 门）：服务端 VAD
        // 与输入转写依赖 16k PCM 流，本地识别此时必须抑制（防双输入）。
        const realtimeForce = realtimeModeRef.current
        if ((realtimeForce || !browserASRAvailable) && (pttRef.current || stateRef.current.mode === 'vad')) {
          const input = event.inputBuffer.getChannelData(0)
          const int16 = float32ToInt16(input)
          // VoicePushAudio 契约：Go []byte 经 bridge 以 base64 字符串传输（对齐
          // b64ToBytes 解码规范的反向编码）；PCM 块内联编码，块小无大数组展开问题。
          app.VoicePushAudio(btoa(String.fromCharCode(...new Uint8Array(int16.buffer)))).catch(() => {})
        }
      }

      // 采集成功：清降级位（用户重新插稳麦克风/授予权限后重启即恢复）
      setState2({ degraded: null })
      return true
    } catch (err) {
      // FE6-02：getUserMedia 失败绝不伪造「正在聆听」。此前在此起定时器
      // 造正弦假音量 + speaking 置真，用户戴着没插稳的麦克风说完整段话，
      // 页面全程显示聆听中、却一个字节都没采到——对用户撒谎式降级。
      // 现改为显式降级位 + 音量归零，由消费方渲染可见警示条。
      // 错误名进日志（DOMException 序列化为 {}，不带 name 无法诊断）。
      const name = (err as DOMException)?.name || ''
      console.warn(`[Voice] 麦克风不可用（${name || '未知错误'}），未采集音频，本回合走文本输入`, err)
      setState2({ degraded: 'mic-unavailable', volume: 0, speaking: false })
      return false
    }
  }, [setState2, startBrowserRecognition, stopBrowserRecognition])

  // ── start / stop ──

  const start = useCallback(async (): Promise<boolean> => {
    abortRef.current = false
    volSmoothRef.current = 0
    micProbeRef.current = false
    setState2({
      active: true, listening: false, speaking: false, aiSpeaking: false,
      transcript: '', finalTranscript: '', volume: 0, error: null, degraded: null, suggestedMic: null,
    })

    // S2：启动前刷新 realtime 模式判定（决定 PCM 强制推送与本地识别抑制）
    await refreshRealtimeMode()

    // 启动后端语音管道（浏览器识别模式下后端仅负责对话与 TTS）
    try {
      await app.VoiceStart(browserASRAvailable)
    } catch (err: unknown) {
      abortRef.current = true
      setState2({
        active: false,
        listening: false,
        speaking: false,
        aiSpeaking: false,
        error: `语音启动失败: ${err instanceof Error ? err.message : String(err)}`,
      })
      return false
    }

    // 启动本地音频采集（音量可视化）
    const captured = await startCapture()
    if (!captured) {
      abortRef.current = true
      stopCapture()
      try { await app.VoiceStop() } catch (_) {}
      setState2({ active: false, listening: false, speaking: false, aiSpeaking: false })
      return false
    }

    // 浏览器端自带识别：直接出文本，不需要后端 ASR 模型。
    // realtime 模式抑制（服务端转写经 voice:transcript 事件回传，防双输入）。
    // FE6-02：麦克风不可用（采集失败）时不起本地识别——否则识别器只报
    // not-allowed 空转，UI 仍显「聆听中」，同一撒谎式降级换条路径复现。
    // 用 startCapture 的返回值判定，不读 stateRef（其赋值发生在 setState
    // updater 内，靠 eager state 才同步，队列里有别的更新时会读到旧值）。
    if (browserASRAvailable && !realtimeModeRef.current) {
      if (!startBrowserRecognition()) {
        abortRef.current = true
        stopCapture()
        try { await app.VoiceStop() } catch (_) {}
        setState2({ active: false, listening: false, speaking: false, aiSpeaking: false })
        return false
      }
    }
    return true
  }, [startCapture, setState2, startBrowserRecognition, refreshRealtimeMode, stopCapture])

  const stop = useCallback(() => {
    abortRef.current = true

    // 停止浏览器端识别与采集
    stopCapture()

    // 停止后端（wailsjsCompat 直调在 mock/未就绪时同步 throw，需 try/catch 防 cleanup 崩溃）
    try { app.VoiceStop().catch(() => {}) } catch { /* 后端未就绪 */ }

    // 停止播放
    stopPlayback()
    pendingSpeechRef.current = 0

    setState2({
      active: false, listening: false, speaking: false, aiSpeaking: false,
      transcript: '', finalTranscript: '', volume: 0, error: null, degraded: null, suggestedMic: null,
    })
  }, [stopCapture, stopPlayback, setState2])

  // ── PTT 控制 ──

  const setPTT = useCallback((active: boolean) => {
    pttRef.current = active
    app.VoiceSetPTTActive(active).catch(() => {})
  }, [])

  // ── 打断 ──

  const interrupt = useCallback(() => {
    cancelPlayback()
    setState2({ aiSpeaking: false })
  }, [cancelPlayback, setState2])

  // ── 零信号自动换真麦：一键切换 ──

  /** 切换录音设备并就地重启采集（会话不中断）：持久化 → 停旧采集 → 用新
   *  设备重开 → 视需要重启浏览器识别。降级位与切换建议一并清空。 */
  const switchMic = useCallback(async (deviceId: string) => {
    saveMicDeviceId(deviceId)
    micProbeRef.current = false // 新设备若再零信号，允许重新探测
    stopCapture()
    setState2({ degraded: null, suggestedMic: null, volume: 0, speaking: false })
    const captured = await startCapture()
    if (captured && browserASRAvailable && !realtimeModeRef.current) {
      startBrowserRecognition()
    }
  }, [stopCapture, startCapture, startBrowserRecognition, setState2])

  // ── 清理 ──

  useEffect(() => {
    return () => {
      abortRef.current = true
      playbackEpochRef.current += 1
      recognitionActiveRef.current = false
      if (recognitionRestartTimerRef.current) {
        clearTimeout(recognitionRestartTimerRef.current)
        recognitionRestartTimerRef.current = null
      }
      if (recognitionRef.current) {
        try { recognitionRef.current.stop() } catch (_) {}
        recognitionRef.current = null
      }
      if (volTimerRef.current) clearInterval(volTimerRef.current)
      if (processorRef.current) { try { processorRef.current.disconnect() } catch (_) {} }
      if (captureCtxRef.current) { captureCtxRef.current.close().catch(() => {}) }
      if (playbackCtxRef.current) { playbackCtxRef.current.close().catch(() => {}) }
      for (const [audio, url] of fallbackAudioRef.current) {
        audio.onended = null
        audio.onerror = null
        try { audio.pause() } catch (_) {}
        try { audio.removeAttribute('src') } catch (_) {}
        try { audio.load() } catch (_) {}
        URL.revokeObjectURL(url)
      }
      fallbackAudioRef.current.clear()
      if (streamRef.current) { streamRef.current.getTracks().forEach(t => t.stop()) }
      if (typeof speechSynthesis !== 'undefined') speechSynthesis.cancel()
      // wailsjsCompat 直调在 mock/未就绪时同步 throw，需 try/catch 防 cleanup 崩溃
      try { app.VoiceStop().catch(() => {}) } catch { /* 后端未就绪 */ }
    }
  }, [])

  return { state, start, stop, setPTT, interrupt, switchMic }
}

// ── 工具函数 ──

/** Float32Array → Int16Array（对齐 Ackem float32ToInt16） */
function float32ToInt16(float32: Float32Array): Int16Array {
  const int16 = new Int16Array(float32.length)
  for (let i = 0; i < float32.length; i++) {
    const s = Math.max(-1, Math.min(1, float32[i]))
    int16[i] = s < 0 ? s * 0x8000 : s * 0x7fff
  }
  return int16
}
