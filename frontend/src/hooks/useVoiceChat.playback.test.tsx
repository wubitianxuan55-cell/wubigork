import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, cleanup, renderHook } from '@testing-library/react'

type EventHandler = (data: unknown) => unknown

const runtimeMocks = vi.hoisted(() => {
  const handlers = new Map<string, EventHandler>()
  const eventsOn = vi.fn((event: string, handler: EventHandler) => {
    handlers.set(event, handler)
    return () => {
      if (handlers.get(event) === handler) handlers.delete(event)
    }
  })
  return { handlers, eventsOn }
})

const bridgeMocks = vi.hoisted(() => ({
  VoiceGetSettings: vi.fn(),
  VoiceStart: vi.fn(),
  VoiceStop: vi.fn(),
  VoicePushAudio: vi.fn(),
  VoiceSetPTTActive: vi.fn(),
  VoiceCancelTTS: vi.fn(),
  VoicePlaybackDone: vi.fn(),
  VoiceChatText: vi.fn(),
}))

vi.mock('../gaea/lib/bridge', () => ({
  app: {
    VoiceGetSettings: bridgeMocks.VoiceGetSettings,
    VoiceStart: bridgeMocks.VoiceStart,
    VoiceStop: bridgeMocks.VoiceStop,
    VoicePushAudio: bridgeMocks.VoicePushAudio,
    VoiceSetPTTActive: bridgeMocks.VoiceSetPTTActive,
    VoiceCancelTTS: bridgeMocks.VoiceCancelTTS,
    VoicePlaybackDone: bridgeMocks.VoicePlaybackDone,
    VoiceChatText: bridgeMocks.VoiceChatText,
  },
}))

vi.mock('../../wailsjs/runtime/runtime', () => ({
  EventsOn: runtimeMocks.eventsOn,
}))

class FakeNode {
  connect() { return this }
  disconnect() { return undefined }
}

class FakeSource extends FakeNode {
  buffer: unknown = null
  onended: (() => void) | null = null
  start = vi.fn()
}

class FakeAudioContext {
  static instances: FakeAudioContext[] = []
  static rejectDecode = false
  state: AudioContextState = 'running'
  destination = new FakeNode()
  sources: FakeSource[] = []
  decodeAudioData = vi.fn(async (_buffer: ArrayBuffer) => {
    if (FakeAudioContext.rejectDecode) throw new Error('unsupported audio format')
    return { duration: 1 }
  })
  close = vi.fn(async () => { this.state = 'closed' })
  resume = vi.fn(async () => {})

  constructor() {
    FakeAudioContext.instances.push(this)
  }

  createGain() {
    return Object.assign(new FakeNode(), { gain: { value: 1 } })
  }

  createBufferSource() {
    const source = new FakeSource()
    this.sources.push(source)
    return source
  }

  createMediaStreamSource() {
    return new FakeNode()
  }

  createAnalyser() {
    return Object.assign(new FakeNode(), {
      fftSize: 0,
      smoothingTimeConstant: 0,
      frequencyBinCount: 8,
      getByteFrequencyData: (buf: Uint8Array) => buf.fill(10),
    })
  }

  createScriptProcessor() {
    return Object.assign(new FakeNode(), { onaudioprocess: null })
  }
}

class FakeSpeechRecognition {
  static instances: FakeSpeechRecognition[] = []
  lang = ''
  continuous = false
  interimResults = false
  maxAlternatives = 0
  onresult: ((event: unknown) => void) | null = null
  onerror: ((event: unknown) => void) | null = null
  onend: (() => void) | null = null
  start = vi.fn()
  stop = vi.fn()

  constructor() {
    FakeSpeechRecognition.instances.push(this)
  }

  emitInterim(text: string) {
    this.onresult?.({
      resultIndex: 0,
      results: [{ isFinal: false, 0: { transcript: text } }],
    })
  }
}

class FakeSpeechSynthesisUtterance {
  lang = ''
  rate = 1
  onend: (() => void) | null = null
  onerror: (() => void) | null = null

  constructor(public text: string) {}
}

class FakeAudioElement {
  static instances: FakeAudioElement[] = []
  onended: (() => void) | null = null
  onerror: (() => void) | null = null
  play = vi.fn(async () => {})
  pause = vi.fn()
  load = vi.fn()
  removeAttribute = vi.fn()

  constructor() {
    FakeAudioElement.instances.push(this)
  }
}

let useVoiceChat: typeof import('./useVoiceChat').useVoiceChat

beforeAll(async () => {
  Object.defineProperty(window, 'SpeechRecognition', {
    configurable: true,
    writable: true,
    value: FakeSpeechRecognition,
  })
  Object.defineProperty(window, 'AudioContext', {
    configurable: true,
    writable: true,
    value: FakeAudioContext,
  })
  Object.defineProperty(window, 'Audio', {
    configurable: true,
    writable: true,
    value: FakeAudioElement,
  })
  Object.defineProperty(window, 'SpeechSynthesisUtterance', {
    configurable: true,
    writable: true,
    value: FakeSpeechSynthesisUtterance,
  })
  Object.defineProperty(URL, 'createObjectURL', {
    configurable: true,
    writable: true,
    value: vi.fn(() => 'blob:voice-test'),
  })
  Object.defineProperty(URL, 'revokeObjectURL', {
    configurable: true,
    writable: true,
    value: vi.fn(),
  })
  if (typeof window.speechSynthesis === 'undefined') {
    Object.defineProperty(window, 'speechSynthesis', {
      configurable: true,
      value: { speak: vi.fn(), cancel: vi.fn() },
    })
  }
  ;({ useVoiceChat } = await import('./useVoiceChat'))
})

beforeEach(() => {
  vi.clearAllMocks()
  runtimeMocks.handlers.clear()
  FakeAudioContext.instances = []
  FakeAudioContext.rejectDecode = false
  FakeSpeechRecognition.instances = []
  FakeAudioElement.instances = []
  bridgeMocks.VoiceGetSettings.mockResolvedValue({ realtimeProvider: '' })
  bridgeMocks.VoiceStart.mockResolvedValue(undefined)
  bridgeMocks.VoiceStop.mockResolvedValue(undefined)
  bridgeMocks.VoicePushAudio.mockResolvedValue(undefined)
  bridgeMocks.VoiceSetPTTActive.mockResolvedValue(undefined)
  bridgeMocks.VoiceCancelTTS.mockResolvedValue(undefined)
  bridgeMocks.VoicePlaybackDone.mockResolvedValue(undefined)
  bridgeMocks.VoiceChatText.mockResolvedValue(undefined)

  const track = {
    stop: vi.fn(),
    addEventListener: vi.fn(),
    getSettings: () => ({ deviceId: 'fake-mic' }),
  }
  const stream = {
    getTracks: () => [track],
    getAudioTracks: () => [track],
  }
  Object.defineProperty(navigator, 'mediaDevices', {
    configurable: true,
    value: {
      getUserMedia: vi.fn(async () => stream),
      enumerateDevices: vi.fn(async () => []),
    },
  })
})

afterEach(() => {
  cleanup()
})

function handler(name: string): EventHandler {
  const fn = runtimeMocks.handlers.get(name)
  if (!fn) throw new Error(`missing event handler: ${name}`)
  return fn
}

describe('useVoiceChat 播报与自动打断', () => {
  it('正常播放结束后只回执一次 PlaybackDone', async () => {
    const { result } = renderHook(() => useVoiceChat())
    await act(async () => {})

    await act(async () => {
      await handler('voice:tts-audio')({ audio: 'AQID', mimeType: 'audio/wav' })
    })

    expect(result.current.state.aiSpeaking).toBe(true)
    const playback = FakeAudioContext.instances.at(-1)
    expect(playback?.sources).toHaveLength(1)
    expect(playback?.sources[0].start).toHaveBeenCalledTimes(1)

    await act(async () => {
      playback?.sources[0].onended?.()
    })

    expect(result.current.state.aiSpeaking).toBe(false)
    expect(bridgeMocks.VoicePlaybackDone).toHaveBeenCalledTimes(1)
  })

  it('解码期间打断 → 迟到的旧音频不得启动或污染下一轮状态', async () => {
    const { result } = renderHook(() => useVoiceChat())
    await act(async () => {})

    let releaseDecode: ((buffer: { duration: number }) => void) | undefined
    const audioPromise = handler('voice:tts-audio')({ audio: 'AQID', mimeType: 'audio/wav' })
    const playback = FakeAudioContext.instances.at(-1)
    if (!playback) throw new Error('playback context was not created')
    playback.decodeAudioData.mockImplementation(
      () => new Promise<{ duration: number }>(resolve => {
        releaseDecode = resolve
      }),
    )

    await act(async () => { result.current.interrupt() })
    expect(bridgeMocks.VoiceCancelTTS).toHaveBeenCalledTimes(1)

    await act(async () => {
      releaseDecode?.({ duration: 1 })
      await audioPromise
    })

    expect(playback.sources).toHaveLength(0)
    expect(bridgeMocks.VoicePlaybackDone).not.toHaveBeenCalled()
    expect(result.current.state.aiSpeaking).toBe(false)
  })

  it('HTML Audio 兜底播放时打断 → 旧元素立即暂停且不回执', async () => {
    FakeAudioContext.rejectDecode = true
    const { result } = renderHook(() => useVoiceChat())
    await act(async () => {})

    await act(async () => {
      await handler('voice:tts-audio')({ audio: 'AQID', mimeType: 'audio/mp3' })
    })

    const audio = FakeAudioElement.instances.at(-1)
    expect(audio?.play).toHaveBeenCalledTimes(1)

    await act(async () => { result.current.interrupt() })

    expect(audio?.pause).toHaveBeenCalledTimes(1)
    expect(bridgeMocks.VoiceCancelTTS).toHaveBeenCalledTimes(1)
    expect(bridgeMocks.VoicePlaybackDone).not.toHaveBeenCalled()
    expect(result.current.state.aiSpeaking).toBe(false)
  })

  it('多句浏览器 TTS fallback 进入队列，不互相 cancel', async () => {
    renderHook(() => useVoiceChat())
    await act(async () => {})

    handler('voice:tts-speak-text')({ text: '第一句。' })
    handler('voice:tts-speak-text')({ text: '第二句。' })

    expect(window.speechSynthesis.speak).toHaveBeenCalledTimes(2)
    expect(window.speechSynthesis.cancel).not.toHaveBeenCalled()
  })

  it('AI 说话时浏览器识别到 interim → 立即停播，最终文本仍进入对话', async () => {
    const { result } = renderHook(() => useVoiceChat())
    await act(async () => {
      expect(await result.current.start()).toBe(true)
      handler('voice:thinking')({ active: true })
    })
    expect(result.current.state.aiSpeaking).toBe(true)

    const recognition = FakeSpeechRecognition.instances.at(-1)
    if (!recognition) throw new Error('speech recognition was not started')
    await act(async () => {
      recognition.emitInterim('等一下')
    })

    expect(bridgeMocks.VoiceCancelTTS).toHaveBeenCalledTimes(1)
    expect(result.current.state.aiSpeaking).toBe(false)
    expect(result.current.state.transcript).toBe('等一下')
  })
})
