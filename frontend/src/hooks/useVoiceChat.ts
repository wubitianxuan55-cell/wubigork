import { useState, useRef, useCallback, useEffect } from 'react'
import * as App from '../../wailsjs/go/app/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

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
}

interface Options {
  /** 收到最终识别文本（用于添加用户消息） */
  onTranscript?: (text: string) => void
  /** 收到 AI 回复文本（用于添加 AI 消息） */
  onReply?: (text: string) => void
}

// ── 音频常量（对齐后端 voice_config.go） ──

const SAMPLE_RATE = 16000
const CHUNK_MS = 200
const CHUNK_SIZE = SAMPLE_RATE * 2 * CHUNK_MS / 1000 // 6400 bytes

// 浏览器端自带语音识别（WebView2 / Edge 内核，微软云端识别），
// 免去后端 ASR 模型，识别更快。
const browserASRAvailable = typeof window !== 'undefined' && !!((window as any).SpeechRecognition || (window as any).webkitSpeechRecognition)

export function useVoiceChat({ onTranscript, onReply }: Options = {}) {
  const [state, setState] = useState<VoiceChatState>({
    active: false, listening: false, speaking: false, aiSpeaking: false,
    transcript: '', finalTranscript: '', volume: 0, error: null, mode: 'vad',
  })

  // ── Refs ──
  const captureCtxRef = useRef<AudioContext | null>(null)
  const playbackCtxRef = useRef<AudioContext | null>(null)
  const streamRef = useRef<MediaStream | null>(null)
  const processorRef = useRef<ScriptProcessorNode | null>(null)
  const gainRef = useRef<GainNode | null>(null)
  const abortRef = useRef(false)
  const pttRef = useRef(false)
  const recognitionRef = useRef<any>(null)
  const recognitionActiveRef = useRef(false)
  const pendingSpeechRef = useRef(0)
  const volSmoothRef = useRef(0)
  const volTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const simTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const stateRef = useRef<VoiceChatState>(state)

  // 保持 stateRef 同步
  stateRef.current = state

  const setState2 = useCallback((patch: Partial<VoiceChatState> | ((s: VoiceChatState) => VoiceChatState)) => {
    setState(prev => {
      const next = typeof patch === 'function' ? patch(prev) : { ...prev, ...patch }
      stateRef.current = next
      return next
    })
  }, [])

  // ── 事件监听 ──

  useEffect(() => {
    const unsubs: (() => void)[] = []

    // 状态变更
    unsubs.push(EventsOn('voice:state', (data: any) => {
      if (abortRef.current) return
      const s = data.state as string
      setState2({
        listening: s === 'listening',
        aiSpeaking: s === 'thinking' || s === 'speaking',
        speaking: s === 'speaking',
      })
    }))

    // 识别结果
    unsubs.push(EventsOn('voice:transcript', (data: any) => {
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
    unsubs.push(EventsOn('voice:reply', (data: any) => {
      if (abortRef.current) return
      const text = data.text || ''
      if (text) onReply?.(text)
    }))

    // TTS 音频播放
    unsubs.push(EventsOn('voice:tts-audio', async (data: any) => {
      if (abortRef.current) return
      pendingSpeechRef.current += 1
      setState2({ aiSpeaking: true })
      try {
        await playAudio(data.audio, data.mimeType)
      } catch (err) {
        console.error('[Voice] TTS 播放失败:', err)
        pendingSpeechRef.current = Math.max(0, pendingSpeechRef.current - 1)
        if (pendingSpeechRef.current === 0) setState2({ aiSpeaking: false })
      }
    }))

    // 浏览器 TTS fallback
    unsubs.push(EventsOn('voice:tts-speak-text', (data: any) => {
      if (abortRef.current) return
      const text = data.text || ''
      if (text && 'speechSynthesis' in window) {
        const u = new SpeechSynthesisUtterance(text)
        u.lang = 'zh-CN'
        u.rate = 1.0
        pendingSpeechRef.current += 1
        setState2({ aiSpeaking: true })
        u.onend = () => {
          pendingSpeechRef.current = Math.max(0, pendingSpeechRef.current - 1)
          if (pendingSpeechRef.current === 0) setState2({ aiSpeaking: false })
          App.VoicePlaybackDone().catch(() => {})
        }
        u.onerror = () => {
          pendingSpeechRef.current = Math.max(0, pendingSpeechRef.current - 1)
          if (pendingSpeechRef.current === 0) setState2({ aiSpeaking: false })
          App.VoicePlaybackDone().catch(() => {})
        }
        speechSynthesis.cancel()
        speechSynthesis.speak(u)
      }
    }))

    // TTS 取消
    unsubs.push(EventsOn('voice:tts-speak-cancel', () => {
      stopPlayback()
    }))

    // 监听指示
    unsubs.push(EventsOn('voice:listening', (data: any) => {
      if (abortRef.current) return
      setState2({ listening: data.active ?? false })
    }))

    // 思考指示
    unsubs.push(EventsOn('voice:thinking', (data: any) => {
      if (abortRef.current) return
      setState2({ aiSpeaking: data.active ?? false })
    }))

    return () => {
      unsubs.forEach(fn => { try { fn() } catch (_) {} })
    }
  }, [onTranscript, onReply, setState2])

  // ── 音频播放 ──

  const playAudio = useCallback(async (audioData: any, mimeType: string) => {
    if (!audioData) {
      pendingSpeechRef.current = Math.max(0, pendingSpeechRef.current - 1)
      if (pendingSpeechRef.current === 0) setState2({ aiSpeaking: false })
      App.VoicePlaybackDone().catch(() => {})
      return
    }

    // 确保 playback context 存在
    if (!playbackCtxRef.current || playbackCtxRef.current.state === 'closed') {
      playbackCtxRef.current = new AudioContext()
      gainRef.current = playbackCtxRef.current.createGain()
      gainRef.current.connect(playbackCtxRef.current.destination)
      gainRef.current.gain.value = 1.0
    }

    const ctx = playbackCtxRef.current
    if (ctx.state === 'suspended') await ctx.resume()

    // 转换数据：Wails 事件中 []byte 以 base64 字符串传输（对齐 TTSPlayer atob 解码）
    let bytes: Uint8Array
    if (typeof audioData === 'string') {
      const bin = atob(audioData)
      bytes = Uint8Array.from(bin, c => c.charCodeAt(0))
    } else if (audioData instanceof Uint8Array) {
      bytes = audioData
    } else {
      bytes = new Uint8Array(audioData)
    }
    if (bytes.length === 0) {
      pendingSpeechRef.current = Math.max(0, pendingSpeechRef.current - 1)
      if (pendingSpeechRef.current === 0) setState2({ aiSpeaking: false })
      App.VoicePlaybackDone().catch(() => {})
      return
    }
    const buffer = bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.length) as ArrayBuffer

    try {
      const audioBuffer = await ctx.decodeAudioData(buffer)
      const source = ctx.createBufferSource()
      source.buffer = audioBuffer
      source.connect(gainRef.current!)
      source.onended = () => {
        pendingSpeechRef.current = Math.max(0, pendingSpeechRef.current - 1)
        if (pendingSpeechRef.current === 0) setState2({ aiSpeaking: false })
        App.VoicePlaybackDone().catch(() => {})
      }
      source.start(0)
    } catch (err) {
      // decodeAudioData 可能失败（非浏览器原生格式），尝试用 Audio 元素
      const blob = new Blob([bytes as BlobPart], { type: mimeType || 'audio/mp3' })
      const url = URL.createObjectURL(blob)
      const audio = new Audio(url)
      audio.onended = () => {
        URL.revokeObjectURL(url)
        pendingSpeechRef.current = Math.max(0, pendingSpeechRef.current - 1)
        if (pendingSpeechRef.current === 0) setState2({ aiSpeaking: false })
        App.VoicePlaybackDone().catch(() => {})
      }
      audio.onerror = () => {
        URL.revokeObjectURL(url)
        pendingSpeechRef.current = Math.max(0, pendingSpeechRef.current - 1)
        if (pendingSpeechRef.current === 0) setState2({ aiSpeaking: false })
        App.VoicePlaybackDone().catch(() => {})
      }
      try {
        await audio.play()
      } catch (playErr) {
        // 播放被阻止等失败：兜底释放状态机，避免后端一直等待播放完成
        URL.revokeObjectURL(url)
        pendingSpeechRef.current = Math.max(0, pendingSpeechRef.current - 1)
        if (pendingSpeechRef.current === 0) setState2({ aiSpeaking: false })
        App.VoicePlaybackDone().catch(() => {})
      }
    }
  }, [setState2])

  const stopPlayback = useCallback(() => {
    pendingSpeechRef.current = 0
    setState2({ aiSpeaking: false })
    if (playbackCtxRef.current && playbackCtxRef.current.state !== 'closed') {
      playbackCtxRef.current.close().catch(() => {})
    }
    playbackCtxRef.current = null
    gainRef.current = null
    speechSynthesis.cancel()
    App.VoiceCancelTTS().catch(() => {})
    App.VoicePlaybackDone().catch(() => {})
  }, [])

  // ── 音频采集 ──

  const startCapture = useCallback(async () => {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: {
          sampleRate: SAMPLE_RATE,
          channelCount: 1,
          echoCancellation: true,
          noiseSuppression: true,
        },
      })
      if (abortRef.current) { stream.getTracks().forEach(t => t.stop()); return }

      streamRef.current = stream

      // AudioContext 用于采集（对齐 Ackem captureContextRef）
      captureCtxRef.current = new AudioContext({ sampleRate: SAMPLE_RATE })
      const source = captureCtxRef.current.createMediaStreamSource(stream)

      // AnalyserNode 用于音量可视化
      const analyser = captureCtxRef.current.createAnalyser()
      analyser.fftSize = 256
      analyser.smoothingTimeConstant = 0.3
      source.connect(analyser)

      const freqBuf = new Uint8Array(analyser.frequencyBinCount)
      volTimerRef.current = setInterval(() => {
        analyser.getByteFrequencyData(freqBuf)
        const avg = freqBuf.reduce((a, b) => a + b, 0) / freqBuf.length
        // 平滑 + 较高阈值：过滤环境杂音导致的"正在聆听"误亮
        const raw = avg / 128
        volSmoothRef.current = volSmoothRef.current * 0.7 + raw * 0.3
        const vol = Math.min(volSmoothRef.current, 1)
        setState2({ volume: vol, speaking: vol > 0.12 })
      }, 80)

      // ScriptProcessorNode 用于 PCM 采集（对齐 Ackem）
      const processor = captureCtxRef.current.createScriptProcessor(CHUNK_SIZE / 2, 1, 1)
      processorRef.current = processor
      source.connect(processor)
      processor.connect(captureCtxRef.current.destination) // 必须连接才能触发

      processor.onaudioprocess = (event) => {
        if (abortRef.current) return
        // 仅在 PTT 激活或 VAD 模式下发送；浏览器识别模式下不上送后端 ASR
        if (!browserASRAvailable && (pttRef.current || stateRef.current.mode === 'vad')) {
          const input = event.inputBuffer.getChannelData(0)
          const int16 = float32ToInt16(input)
          App.VoicePushAudio(Array.from(new Uint8Array(int16.buffer))).catch(() => {})
        }
      }

      // 消除模拟音量
      if (simTimerRef.current) { clearInterval(simTimerRef.current); simTimerRef.current = null }
    } catch (err) {
      console.warn('[Voice] 麦克风不可用，使用模拟模式', err)
      // 模拟音量
      let t = 0
      simTimerRef.current = setInterval(() => {
        t += 0.1
        setState2({ volume: 0.12 + Math.sin(t * 2.5) * 0.08 + Math.sin(t * 5) * 0.04, speaking: Math.sin(t * 2.5) > 0.5 })
      }, 120)
    }
  }, [setState2])

  // ── start / stop ──

  // ── 浏览器端语音识别（Web Speech API，WebView2 自带） ──

  const startBrowserRecognition = useCallback(() => {
    if (!browserASRAvailable) return
    const SR = (window as any).SpeechRecognition || (window as any).webkitSpeechRecognition
    if (!SR || recognitionActiveRef.current) return
    recognitionActiveRef.current = true

    const rec = new SR()
    recognitionRef.current = rec
    rec.lang = 'zh-CN'
    rec.continuous = true
    rec.interimResults = true
    rec.maxAlternatives = 1

    rec.onresult = (e: any) => {
      if (abortRef.current) return
      let interim = ''
      for (let i = e.resultIndex; i < e.results.length; i++) {
        const res = e.results[i]
        if (!res) continue
        const text = (res[0]?.transcript || '').trim()
        if (!text) continue
        if (res.isFinal) {
          setState2(s => ({ ...s, transcript: '', finalTranscript: s.finalTranscript + text }))
          // 直接进入后端对话管道（跳过 ASR 模型，识别更快）
          App.VoiceChatText(text).catch(() => {})
        } else {
          interim += text
        }
      }
      setState2(s => ({ ...s, transcript: interim }))
    }

    rec.onerror = (e: any) => {
      console.warn('[Voice] 浏览器识别错误:', e?.error)
      if (e?.error === 'not-allowed' || e?.error === 'service-not-allowed') {
        setState2({ error: '语音识别被拒绝（麦克风权限或服务不可用）' })
      }
    }

    rec.onend = () => {
      // continuous 模式在静音后可能自行结束，自动重连保持聆听
      if (!abortRef.current && recognitionActiveRef.current) {
        setTimeout(() => {
          if (!abortRef.current && recognitionActiveRef.current && recognitionRef.current) {
            try { recognitionRef.current.start() } catch (_) {}
          }
        }, 150)
      }
    }

    try { rec.start() } catch (_) {}
  }, [setState2])

  const stopBrowserRecognition = useCallback(() => {
    recognitionActiveRef.current = false
    if (recognitionRef.current) {
      try { recognitionRef.current.stop() } catch (_) {}
      recognitionRef.current = null
    }
  }, [])

  const start = useCallback(async () => {
    abortRef.current = false
    volSmoothRef.current = 0
    setState2({
      active: true, listening: false, speaking: false, aiSpeaking: false,
      transcript: '', finalTranscript: '', volume: 0, error: null,
    })

    // 启动后端语音管道（浏览器识别模式下后端仅负责对话与 TTS）
    try {
      await App.VoiceStart(browserASRAvailable)
    } catch (err: any) {
      setState2({ error: `语音启动失败: ${err?.message || err}` })
      return
    }

    // 启动本地音频采集（音量可视化）
    await startCapture()

    // 浏览器端自带识别：直接出文本，不需要后端 ASR 模型
    if (browserASRAvailable) {
      startBrowserRecognition()
    }
  }, [startCapture, setState2])

  const stop = useCallback(() => {
    abortRef.current = true

    // 停止浏览器端识别
    stopBrowserRecognition()

    // 停止后端
    App.VoiceStop().catch(() => {})

    // 停止采集
    volSmoothRef.current = 0
    if (volTimerRef.current) { clearInterval(volTimerRef.current); volTimerRef.current = null }
    if (simTimerRef.current) { clearInterval(simTimerRef.current); simTimerRef.current = null }
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

    // 停止播放
    stopPlayback()
    pendingSpeechRef.current = 0

    setState2({
      active: false, listening: false, speaking: false, aiSpeaking: false,
      transcript: '', finalTranscript: '', volume: 0, error: null,
    })
  }, [stopPlayback, setState2])

  // ── PTT 控制 ──

  const setPTT = useCallback((active: boolean) => {
    pttRef.current = active
    App.VoiceSetPTTActive(active).catch(() => {})
  }, [])

  // ── 打断 ──

  const interrupt = useCallback(() => {
    App.VoiceCancelTTS().catch(() => {})
    stopPlayback()
    setState2({ aiSpeaking: false })
  }, [stopPlayback, setState2])

  // ── 清理 ──

  useEffect(() => {
    return () => {
      abortRef.current = true
      recognitionActiveRef.current = false
      if (recognitionRef.current) {
        try { recognitionRef.current.stop() } catch (_) {}
        recognitionRef.current = null
      }
      if (volTimerRef.current) clearInterval(volTimerRef.current)
      if (simTimerRef.current) clearInterval(simTimerRef.current)
      if (processorRef.current) { try { processorRef.current.disconnect() } catch (_) {} }
      if (captureCtxRef.current) { captureCtxRef.current.close().catch(() => {}) }
      if (playbackCtxRef.current) { playbackCtxRef.current.close().catch(() => {}) }
      if (streamRef.current) { streamRef.current.getTracks().forEach(t => t.stop()) }
      speechSynthesis.cancel()
      App.VoiceStop().catch(() => {})
    }
  }, [])

  return { state, start, stop, setPTT, interrupt }
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
