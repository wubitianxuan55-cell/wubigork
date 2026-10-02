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
    degraded,
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
 *  使 startCapture 能走完全程（只关心降级位，不关心可视化数值）。 */
function installFakeAudioContext() {
  class FakeNode {
    connect() { return this }
    disconnect() { return this }
  }
  const fakeMediaStream = { getTracks: () => [{ stop: vi.fn() }] }
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
    createAnalyser() { return Object.assign(new FakeNode(), { fftSize: 0, smoothingTimeConstant: 0, frequencyBinCount: 8, getByteFrequencyData: () => {} }) }
    createScriptProcessor() { return Object.assign(new FakeNode(), { onaudioprocess: null }) }
  }
  Object.defineProperty(window, 'AudioContext', { configurable: true, writable: true, value: FakeAudioContext })
  return fakeMediaStream
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

  it('getUserMedia 成功 → degraded 保持 null（反向验证：降级位不是恒亮）', async () => {
    const stream = installFakeAudioContext()
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: { getUserMedia: vi.fn(async () => stream) },
    })
    const { result } = renderHook(() => useVoiceChat())
    await act(async () => { await result.current.start() })
    expect(result.current.state.degraded).toBeNull()
    await act(async () => { result.current.stop() })
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
