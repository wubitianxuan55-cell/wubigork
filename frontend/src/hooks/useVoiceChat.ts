import { useState, useRef, useCallback, useEffect } from 'react'

export interface VoiceChatState {
  active: boolean
  listening: boolean
  speaking: boolean
  aiSpeaking: boolean
  transcript: string
  finalTranscript: string
  volume: number
  error: string | null
}

interface Options {
  onSpeechResult: (text: string) => Promise<string>
  onTTS: (text: string) => Promise<string | void>
}

export function useVoiceChat({ onSpeechResult, onTTS }: Options) {
  const [state, setState] = useState<VoiceChatState>({
    active: false, listening: false, speaking: false, aiSpeaking: false,
    transcript: '', finalTranscript: '', volume: 0, error: null,
  })

  const recRef = useRef<any>(null)
  const ctxRef = useRef<AudioContext | null>(null)
  const strRef = useRef<MediaStream | null>(null)
  const silenceRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const abortRef = useRef(false)
  const simRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const volRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const stopListenFn = useRef<() => void>(() => {})
  const startFn = useRef<() => void>(() => {})

  const cleanup = useCallback(() => {
    if (silenceRef.current) clearTimeout(silenceRef.current)
    if (simRef.current) clearInterval(simRef.current)
    if (volRef.current) clearInterval(volRef.current)
    if (recRef.current) { try { recRef.current.stop() } catch (_) {}; recRef.current = null }
    if (ctxRef.current) { ctxRef.current.close().catch(() => {}); ctxRef.current = null }
    if (strRef.current) { strRef.current.getTracks().forEach(t => t.stop()); strRef.current = null }
  }, [])

  // ── start ──
  const start = useCallback(() => {
    abortRef.current = false
    setState(s => ({ ...s, active: true, listening: false, speaking: false, aiSpeaking: false, transcript: '', finalTranscript: '', volume: 0, error: null }))

    // 模拟音量，确保粒子光球动起来
    let t = 0
    simRef.current = setInterval(() => {
      t += 0.1
      setState(s => ({ ...s, volume: 0.12 + Math.sin(t * 2.5) * 0.08 + Math.sin(t * 5) * 0.04, speaking: Math.sin(t * 2.5) > 0.5 }))
    }, 120)

    // 尝试接入真实麦克风
    const tryMic = async () => {
      try {
        if (!navigator.mediaDevices?.getUserMedia) return
        const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
        if (abortRef.current) { stream.getTracks().forEach(t => t.stop()); return }
        strRef.current = stream
        if (simRef.current) clearInterval(simRef.current)

        const ctx = new AudioContext()
        ctxRef.current = ctx
        const src = ctx.createMediaStreamSource(stream)
        const ana = ctx.createAnalyser()
        ana.fftSize = 256; ana.smoothingTimeConstant = 0.3
        src.connect(ana)
        const buf = new Uint8Array(ana.frequencyBinCount)
        volRef.current = setInterval(() => {
          ana.getByteFrequencyData(buf)
          const avg = buf.reduce((a, b) => a + b, 0) / buf.length
          setState(s => ({ ...s, volume: Math.min(avg / 128, 1), speaking: avg / 128 > 0.06 }))
        }, 80)
      } catch (_) { /* 无麦克风，继续模拟 */ }
    }

    // 尝试语音识别
    const tryRec = () => {
      try {
        // @ts-ignore
        const SR = window.SpeechRecognition || window.webkitSpeechRecognition
        if (!SR) { setState(s => ({ ...s, listening: true })); return }
        const r = new SR()
        r.lang = 'zh-CN'; r.interimResults = true; r.continuous = true; r.maxAlternatives = 1
        recRef.current = r

        r.onresult = (ev: any) => {
          let fi = '', im = ''
          for (let i = ev.resultIndex; i < ev.results.length; i++) {
            if (ev.results[i].isFinal) fi += ev.results[i][0].transcript
            else im += ev.results[i][0].transcript
          }
          setState(s => ({ ...s, transcript: fi + im, finalTranscript: s.finalTranscript + fi }))
          if (silenceRef.current) clearTimeout(silenceRef.current)
          silenceRef.current = setTimeout(() => stopListenFn.current(), 2000)
        }
        r.onerror = (ev: any) => {
          if (ev.error === 'no-speech' || ev.error === 'aborted') return
          const msg = ev.error === 'not-allowed'
            ? '语音识别被拒绝：请开启 Windows 设置 → 隐私 → 语音 → 联机语音识别'
            : `识别错误: ${ev.error}`
          setState(s => ({ ...s, error: msg }))
        }
        r.onend = () => { if (!abortRef.current) try { r.start() } catch (_) {} }
        r.onend = () => { if (!abortRef.current) try { r.start() } catch (_) {} }
        r.start()
        setState(s => ({ ...s, listening: true }))
      } catch (_) { setState(s => ({ ...s, listening: true })) }
    }

    setTimeout(() => { tryMic().then(tryRec) }, 200)
  }, [cleanup])

  // ── stopListening ──
  const stopListening = useCallback(async () => {
    abortRef.current = true
    if (silenceRef.current) clearTimeout(silenceRef.current)
    if (recRef.current) { try { recRef.current.stop() } catch (_) {}; recRef.current = null }
    if (ctxRef.current) { ctxRef.current.close().catch(() => {}); ctxRef.current = null }
    if (strRef.current) { strRef.current.getTracks().forEach(t => t.stop()); strRef.current = null }

    const text = await new Promise<string>(resolve => {
      setState(s => { resolve((s.finalTranscript + s.transcript).trim()); return { ...s, listening: false, speaking: false, volume: 0 } })
    })

    if (!text) return
    try {
      setState(s => ({ ...s, aiSpeaking: true }))
      const reply = await onSpeechResult(text)
      if (reply && onTTS) await onTTS(reply)
      setState(s => ({ ...s, aiSpeaking: false, transcript: '', finalTranscript: '' }))
      if (!abortRef.current) setTimeout(() => startFn.current(), 500)
    } catch (err: any) {
      const msg = err?.message || err?.toString?.() || 'AI 回复失败（请确认模型中心已启动语言模型）'
      setState(s => ({ ...s, aiSpeaking: false, error: msg }))
    }
  }, [onSpeechResult, onTTS])

  stopListenFn.current = stopListening
  startFn.current = start

  const stop = useCallback(() => {
    abortRef.current = true
    cleanup()
    setState({ active: false, listening: false, speaking: false, aiSpeaking: false, transcript: '', finalTranscript: '', volume: 0, error: null })
  }, [cleanup])

  useEffect(() => () => { abortRef.current = true; cleanup() }, [cleanup])

  return { state, start, stop, stopListening }
}
