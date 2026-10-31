// FE6-02 回归防线：麦克风不可用必须显式降级 + 页面可见警示条。
// 此前 getUserMedia 失败只 console.warn，随后起定时器伪造正弦音量并置
// speaking=true，页面全程「聆听中」而实际零采集——用户说完整段话、AI 照常
// 回（走文本），是对用户的撒谎式降级。本用例钉三件事：
// ① degraded 置 'mic-unavailable'；② 不再有任何假音量（volume 恒 0、speaking 恒 false）；
// ③ 消费方（WelcomeScreen，语音主视觉落点）出现中文警示文案而非「聆听中」；
// ④ 反向：麦克风可用时 degraded 恒 null、警示条不出现（防「恒亮」假绿）。
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, cleanup, render, renderHook, screen } from '@testing-library/react'
import { useVoiceChat } from './useVoiceChat'
import { WelcomeScreen } from '../components/chat/WelcomeScreen'

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
  EventsOn: vi.fn(() => () => {}),
}))

// jsdom 无 canvas 2D 上下文（VoiceChatOrb 直接 ctx.setTransform），且本用例
// 只关心降级警示条——按 ChatPage.test 的既有做法整体替身语音球。
vi.mock('../components/VoiceChatOrb', () => ({ default: () => <div data-testid="voice-orb" /> }))

const DEGRADED_TEXT = '麦克风不可用，未在采集音频，本回合走文本输入'

function voiceState(degraded: 'mic-unavailable' | null) {
  return {
    active: true, listening: degraded === null, speaking: false, aiSpeaking: false,
    transcript: '', finalTranscript: '', volume: 0, error: null, mode: 'vad' as const,
    degraded, suggestedMic: null,
  }
}

function renderWelcome(degraded: 'mic-unavailable' | null) {
  return render(
    <WelcomeScreen
      mode="plain"
      personaLabel="助手"
      companionName="gaea AI"
      voice={voiceState(degraded)}
      emoColor="#34d399" // hex-exempt 测试夹具色（非产品样式）
      activePersonality="plain"
      onSwitchPersonality={vi.fn()}
      onNavigateLib={vi.fn()}
      onFillInput={vi.fn()}
      onSuggestion={vi.fn()}
    />,
  )
}

/** jsdom 无 WebAudio：给「麦克风可用」路径一个最小假 AudioContext，
 *  使 startCapture 能走完全程（只关心降级位，不关心可视化数值）。
 *  返回带事件注册表的假 track（中途失麦检测用）：trackListeners[mute/ended/
 *  unmute] 由用例手动触发，模拟设备中途移除/系统静音。 */
function installFakeAudioContext() {
  class FakeNode {
    connect() { return this }
    disconnect() { return this }
  }
  const trackListeners: Record<string, Array<() => void>> = {}
  const fakeTrack = {
    stop: vi.fn(),
    addEventListener: (type: string, fn: () => void) => {
      ;(trackListeners[type] ??= []).push(fn)
    },
    removeEventListener: vi.fn(),
  }
  const fakeMediaStream = {
    getTracks: () => [fakeTrack],
    getAudioTracks: () => [fakeTrack],
  }
  // 零信号检测开关：signal=false 时频域全零（虚拟麦克风），true 时有真实信号
  const micSource = { signal: false }
  class FakeAudioContext {
    state = 'running'
    sampleRate = 16000
    destination = new FakeNode()
    close = vi.fn(async () => {})
    resume = vi.fn(async () => {})
    decodeAudioData = vi.fn(async () => ({ duration: 0 }))
    createGain() { return Object.assign(new FakeNode(), { gain: { value: 1 } }) }
    createBufferSource() { return Object.assign(new FakeNode(), { buffer: null, start: vi.fn(), onended: null }) }
    createMediaStreamSource() { return new FakeNode() }
    createAnalyser() { return Object.assign(new FakeNode(), { fftSize: 0, smoothingTimeConstant: 0, frequencyBinCount: 8, getByteFrequencyData: (buf: Uint8Array) => { if (micSource.signal) buf.fill(10) } }) }
    createScriptProcessor() { return Object.assign(new FakeNode(), { onaudioprocess: null }) }
  }
  Object.defineProperty(window, 'AudioContext', { configurable: true, writable: true, value: FakeAudioContext })
  return { stream: fakeMediaStream, trackListeners, micSource }
}

describe('useVoiceChat 麦克风不可用降级（FE6-02）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    bridgeMocks.VoiceGetSettings.mockResolvedValue({ realtimeProvider: '' })
    bridgeMocks.VoiceStart.mockResolvedValue(undefined)
    bridgeMocks.VoiceStop.mockResolvedValue(undefined)
    bridgeMocks.VoicePushAudio.mockResolvedValue(undefined)
    bridgeMocks.VoiceSetPTTActive.mockResolvedValue(undefined)
    bridgeMocks.VoiceCancelTTS.mockResolvedValue(undefined)
    bridgeMocks.VoicePlaybackDone.mockResolvedValue(undefined)
    bridgeMocks.VoiceChatText.mockResolvedValue(undefined)
    // jsdom 无 speechSynthesis：hook 卸载清理会调 speechSynthesis.cancel()
    if (typeof window.speechSynthesis === 'undefined') {
      Object.defineProperty(window, 'speechSynthesis', {
        configurable: true,
        value: { speak: vi.fn(), cancel: vi.fn() },
      })
    }
  })

  afterEach(() => {
    cleanup()
    delete (navigator as unknown as { mediaDevices?: unknown }).mediaDevices
  })

  it('getUserMedia reject → degraded 置位，且无假音量（volume 0 / speaking false）', async () => {
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: { getUserMedia: vi.fn(async () => { throw new Error('NotFoundError: 未发现音频输入设备') }) },
    })
    const { result } = renderHook(() => useVoiceChat())
    expect(result.current.state.degraded).toBeNull()

    await act(async () => { await result.current.start() })

    expect(result.current.state.degraded).toBe('mic-unavailable')
    // 旧实现此处会起 simTimer 造正弦假音量并把 speaking 置真
    expect(result.current.state.volume).toBe(0)
    expect(result.current.state.speaking).toBe(false)
  })

  it('后端 VoiceStart 拒绝 → start 返回 false，状态回落到未启动并保留可见错误', async () => {
    bridgeMocks.VoiceStart.mockRejectedValueOnce(new Error('语音对话引擎未就绪'))
    const { result } = renderHook(() => useVoiceChat())

    let started = true
    await act(async () => { started = await result.current.start() })

    expect(started).toBe(false)
    expect(result.current.state.active).toBe(false)
    expect(result.current.state.listening).toBe(false)
    expect(result.current.state.error).toContain('语音对话引擎未就绪')
  })

  it('getUserMedia 成功 → degraded 保持 null（反向验证：降级位不是恒亮）', async () => {
    const { stream } = installFakeAudioContext()
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: { getUserMedia: vi.fn(async () => stream) },
    })
    const { result } = renderHook(() => useVoiceChat())
    await act(async () => { await result.current.start() })
    expect(result.current.state.degraded).toBeNull()
    await act(async () => { result.current.stop() })
  })

  // ── FE6-02 补课：中途失麦（采集成功后的静默死亡路径） ──

  it('采集中 track mute（系统静音）→ degraded 置位、音量归零（静音不冒充聆听）', async () => {
    const { stream, trackListeners } = installFakeAudioContext()
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: { getUserMedia: vi.fn(async () => stream) },
    })
    const { result } = renderHook(() => useVoiceChat())
    await act(async () => { await result.current.start() })
    expect(result.current.state.degraded).toBeNull()

    await act(async () => { (trackListeners['mute'] ?? []).forEach(fn => fn()) })

    expect(result.current.state.degraded).toBe('mic-unavailable')
    expect(result.current.state.volume).toBe(0)
    expect(result.current.state.speaking).toBe(false)
    await act(async () => { result.current.stop() })
  })

  it('mute 后 unmute（系统恢复）→ degraded 自愈清除', async () => {
    const { stream, trackListeners } = installFakeAudioContext()
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: { getUserMedia: vi.fn(async () => stream) },
    })
    const { result } = renderHook(() => useVoiceChat())
    await act(async () => { await result.current.start() })
    await act(async () => { (trackListeners['mute'] ?? []).forEach(fn => fn()) })
    expect(result.current.state.degraded).toBe('mic-unavailable')

    await act(async () => { (trackListeners['unmute'] ?? []).forEach(fn => fn()) })
    expect(result.current.state.degraded).toBeNull()
    await act(async () => { result.current.stop() })
  })

  it('track ended（设备移除/被独占）→ degraded 置位（ended 无自愈路径）', async () => {
    const { stream, trackListeners } = installFakeAudioContext()
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: { getUserMedia: vi.fn(async () => stream) },
    })
    const { result } = renderHook(() => useVoiceChat())
    await act(async () => { await result.current.start() })
    expect(result.current.state.degraded).toBeNull()

    await act(async () => { (trackListeners['ended'] ?? []).forEach(fn => fn()) })

    expect(result.current.state.degraded).toBe('mic-unavailable')
    expect(result.current.state.volume).toBe(0)
    await act(async () => { result.current.stop() })
  })

  it('stop() 主动结束会话后补发的 track ended 不误降级（abortRef 门）', async () => {
    const { stream, trackListeners } = installFakeAudioContext()
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: { getUserMedia: vi.fn(async () => stream) },
    })
    const { result } = renderHook(() => useVoiceChat())
    await act(async () => { await result.current.start() })
    await act(async () => { result.current.stop() })
    // 会话已结束：设备随后断开（ended 补发）不产生误导性的降级警示
    await act(async () => { (trackListeners['ended'] ?? []).forEach(fn => fn()) })
    expect(result.current.state.degraded).toBeNull()
    expect(result.current.state.active).toBe(false)
  })

  // ── FE6-02 终局：零信号检测（虚拟麦克风实弹场景） ──
  // 系统默认录音设备可能是虚拟音频（ToDesk Virtual Audio 实证）：
  // getUserMedia 成功、track.muted=false、却全程零样本——静音绝不冒充聆听。

  it('采集成功但连续 5s 零输入（虚拟麦克风）→ degraded 置位', async () => {
    vi.useFakeTimers()
    try {
      const { stream, micSource } = installFakeAudioContext()
      Object.defineProperty(navigator, 'mediaDevices', {
        configurable: true,
        value: { getUserMedia: vi.fn(async () => stream) },
      })
      const { result } = renderHook(() => useVoiceChat())
      await act(async () => { await result.current.start() })
      expect(result.current.state.degraded).toBeNull()

      micSource.signal = false // 虚拟设备：频域全零
      await act(async () => { vi.advanceTimersByTime(5600) })

      expect(result.current.state.degraded).toBe('mic-unavailable')
      expect(result.current.state.speaking).toBe(false)
      await act(async () => { result.current.stop() })
    } finally {
      vi.useRealTimers()
    }
  })

  it('零输入降级后出现真实信号 → 自愈恢复 degraded=null', async () => {
    vi.useFakeTimers()
    try {
      const { stream, micSource } = installFakeAudioContext()
      Object.defineProperty(navigator, 'mediaDevices', {
        configurable: true,
        value: { getUserMedia: vi.fn(async () => stream) },
      })
      const { result } = renderHook(() => useVoiceChat())
      await act(async () => { await result.current.start() })
      await act(async () => { vi.advanceTimersByTime(5600) })
      expect(result.current.state.degraded).toBe('mic-unavailable')

      micSource.signal = true // 麦克风活了（换插真设备/解除硬件静音）
      await act(async () => { vi.advanceTimersByTime(200) })
      expect(result.current.state.degraded).toBeNull()
      await act(async () => { result.current.stop() })
    } finally {
      vi.useRealTimers()
    }
  })

  it('NotReadableError 瞬态失败 → 有界重试后成功（蓝牙档位切换窗口）', async () => {
    vi.useFakeTimers()
    try {
      const { stream } = installFakeAudioContext()
      let calls = 0
      const gum = vi.fn(async () => {
        calls++
        if (calls <= 2) {
          const e = new DOMException('device busy', 'NotReadableError')
          throw e
        }
        return stream
      })
      Object.defineProperty(navigator, 'mediaDevices', {
        configurable: true,
        value: { getUserMedia: gum },
      })
      const { result } = renderHook(() => useVoiceChat())
      const startPromise = act(async () => { await result.current.start() })
      // 两次失败间隔 800ms：推进定时器放行重试
      await act(async () => { await vi.advanceTimersByTimeAsync(2000) })
      await startPromise
      expect(calls).toBe(3)
      expect(result.current.state.degraded).toBeNull()
      expect(result.current.state.active).toBe(true)
      await act(async () => { result.current.stop() })
    } finally {
      vi.useRealTimers()
    }
  })

  it('NotReadableError 连续全档失败 → 4 档阶梯后诚实降级（不无限拖）', async () => {
    vi.useFakeTimers()
    try {
      const gum = vi.fn(async () => { throw new DOMException('busy', 'NotReadableError') })
      Object.defineProperty(navigator, 'mediaDevices', {
        configurable: true,
        value: { getUserMedia: gum },
      })
      const { result } = renderHook(() => useVoiceChat())
      const startPromise = act(async () => { await result.current.start() })
      await act(async () => { await vi.advanceTimersByTimeAsync(4000) })
      await startPromise
      expect(gum).toHaveBeenCalledTimes(4) // 全约束×2 + 去 sampleRate + 裸
      expect(result.current.state.degraded).toBe('mic-unavailable')
      await act(async () => { result.current.stop() })
    } finally {
      vi.useRealTimers()
    }
  })

  it('IndexSizeError → 降档到去 sampleRate/裸约束后成功（约束雷区兜底）', async () => {
    vi.useFakeTimers()
    try {
      const { stream } = installFakeAudioContext()
      const gum = vi.fn(async (c?: MediaStreamConstraints) => {
        const audio = c?.audio
        if (typeof audio === 'object' && audio && 'sampleRate' in audio) {
          throw new DOMException('bad range', 'IndexSizeError')
        }
        return stream
      })
      Object.defineProperty(navigator, 'mediaDevices', {
        configurable: true,
        value: { getUserMedia: gum },
      })
      const { result } = renderHook(() => useVoiceChat())
      const startPromise = act(async () => { await result.current.start() })
      await act(async () => { await vi.advanceTimersByTimeAsync(2000) })
      await startPromise
      expect(result.current.state.degraded).toBeNull()
      expect(result.current.state.active).toBe(true)
      // 第 3 档（去 sampleRate）命中
      const lastCall = gum.mock.calls[gum.mock.calls.length - 1]?.[0] as { audio?: Record<string, unknown> }
      expect(lastCall?.audio && 'sampleRate' in lastCall.audio).toBe(false)
      await act(async () => { result.current.stop() })
    } finally {
      vi.useRealTimers()
    }
  })

  it('switchMic(deviceId)：持久化选择并以 exact 约束就地重启采集，降级清空', async () => {
    const { stream } = installFakeAudioContext()
    const gum = vi.fn(async (_constraints?: MediaStreamConstraints) => stream)
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: { getUserMedia: gum, enumerateDevices: vi.fn(async () => []) },
    })
    const { result } = renderHook(() => useVoiceChat())
    await act(async () => { await result.current.start() })

    await act(async () => { await result.current.switchMic('dev-real') })

    expect(result.current.state.degraded).toBeNull()
    expect(result.current.state.suggestedMic).toBeNull()
    expect(result.current.state.active).toBe(true) // 会话不中断：采集就地重启
    const lastCall = gum.mock.calls[gum.mock.calls.length - 1]?.[0] as { audio?: { deviceId?: { exact?: string } } }
    expect(lastCall?.audio?.deviceId?.exact).toBe('dev-real')
    expect(localStorage.getItem('gaea.voice.micDeviceId')).toBe('dev-real')
    await act(async () => { result.current.stop() })
    localStorage.removeItem('gaea.voice.micDeviceId')
  })
})

describe('语音降级警示条渲染落点（WelcomeScreen，chat 语音主视觉）', () => {
  it('degraded="mic-unavailable" → 出现中文警示文案（未在采集音频）', () => {
    renderWelcome('mic-unavailable')
    expect(screen.getByText(DEGRADED_TEXT)).toBeTruthy()
    expect(screen.getByRole('alert').textContent).toContain('未在采集音频')
    expect(screen.getByRole('alert').textContent).toContain('本回合走文本输入')
  })

  it('未降级（麦克风正常）→ 不出现警示条', () => {
    renderWelcome(null)
    expect(screen.queryByText(DEGRADED_TEXT)).toBeNull()
    expect(screen.queryByRole('alert')).toBeNull()
  })
})
