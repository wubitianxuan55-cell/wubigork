/**
 * voice/runtime/useVoiceChatV2.test.ts — 会话运行时钩子测试（哑线缆 + 桩桥）。
 */
import { act, renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useVoiceChatV2, type VoiceV2Turn } from './useVoiceChatV2'
import type { VoiceTransport } from '../client/VoiceClientV2'
import { FIXTURE_BARGEIN, FIXTURE_TURN } from '../session/fixtures'
import { app } from '../../gaea/lib/bridge'

class FakeTransport implements VoiceTransport {
  sent: string[] = []
  onopen: (() => void) | null = null
  onclose: (() => void) | null = null
  onerror: ((err: unknown) => void) | null = null
  onmessage: ((data: string) => void) | null = null
  send(data: string): void {
    this.sent.push(data)
  }
  close(): void {
    this.onclose?.()
  }
}

// 桥：只桩钩子消费的三个方法
vi.mock('../../gaea/lib/bridge', () => ({
  app: {
    VoiceV2Status: vi.fn(async () => ({
      sidecar: true, llm: true, breeze: true, gateway: true, ready: true,
      error: '',
    })),
    VoiceV2Ensure: vi.fn(async () => ({ ready: true, starting: false })),
    ChatAppendMessages: vi.fn(async () => undefined),
  },
}))

// jsdom 无 audioWorklet：AudioContext 构造前拦截（本测不开麦不播）
class FakeAudioCtx {
  audioWorklet = { addModule: async () => undefined }
  destination = {}
  close = async () => undefined
  createMediaStreamSource() { return { connect: () => undefined } }
}
vi.stubGlobal('AudioContext', FakeAudioCtx)
const sinkMessages: unknown[] = []
vi.stubGlobal('AudioWorkletNode', class {
  port = {
    onmessage: null,
    postMessage: (message: unknown) => sinkMessages.push(message),
    close: () => undefined,
  }
  connect() {}
  disconnect() {}
})

describe('useVoiceChatV2', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    sinkMessages.length = 0
  })

  it('connect 走注入工厂并发 session.update', async () => {
    let t: FakeTransport | null = null
    const { result } = renderHook(() => useVoiceChatV2({
      url: 'ws://test/realtime',
      voice: 'char_gaea',
      transportFactory: () => { t = new FakeTransport(); return t },
    }))
    await act(async () => { await result.current.connect() })
    expect(result.current.connected).toBe(false) // 只有 transport.onopen 才算真连上
    expect(result.current.note).toContain('连接中')
    act(() => { t!.onopen?.() })
    expect(result.current.connected).toBe(true)
    expect(t!.sent).toHaveLength(1)
    expect(JSON.parse(t!.sent[0]).type).toBe('session.update')
    expect(JSON.parse(t!.sent[0]).session.audio.input.turn_detection.type).toBe('server_vad')
    // 人格音色进 session.update.audio.output.voice（声色固定根修）
    expect(JSON.parse(t!.sent[0]).session.audio.output.voice).toBe('char_gaea')
  })

  it('回放实录：状态到 idle + 轮完成回执带用户转写与助手文本', async () => {
    const turns: VoiceV2Turn[] = []
    const { result } = renderHook(() => useVoiceChatV2({
      makeReplayTransport: () => new FakeTransport(),
      onTurnComplete: (t) => turns.push(t),
    }))
    await act(async () => { await result.current.replayEvents(FIXTURE_TURN, 0) })
    expect(result.current.phase).toBe('idle')
    expect(result.current.transcripts).toEqual(['你好请用两句话介绍一下你自己'])
    expect(result.current.assistantText).toContain('语音助手')
    expect(turns).toEqual([{
      user: '你好请用两句话介绍一下你自己',
      assistant: '你好呀，我是你的语音助手gaea。很高兴跟你聊天，有什么想聊的尽管说！',
      responseId: 'resp_turn_1',
      itemId: 'item_user_1',
    }])
  })

  it('bargein 回放：被取消轮不产生写回回执', async () => {
    const turns: VoiceV2Turn[] = []
    const { result } = renderHook(() => useVoiceChatV2({
      makeReplayTransport: () => new FakeTransport(),
      onTurnComplete: (t) => turns.push(t),
    }))
    await act(async () => { await result.current.replayEvents(FIXTURE_BARGEIN, 0) })
    // 只有第二轮 completed 产生回执（第一轮 cancelled/turn_detected）
    expect(turns.map((t) => t.responseId)).toEqual(['resp_barge_2'])
    expect(turns[0].user).toContain('天气')
    expect(sinkMessages).toContainEqual({ type: 'flush' })
  })

  it('依赖链就绪经桥上报', async () => {
    const { result } = renderHook(() => useVoiceChatV2({}))
    await waitFor(() => expect(result.current.services.ready).toBe(true))
  })

  it('后端终止错误立即结束启动等待并显示原因', async () => {
    vi.mocked(app.VoiceV2Ensure).mockResolvedValueOnce({
      ready: false,
      starting: false,
      error: 'sidecar 脚本缺失——设置 GAEA_VOICE_V2_ROOT',
    })
    const { result } = renderHook(() => useVoiceChatV2({}))
    let ready = true
    await act(async () => {
      ready = await result.current.ensureServices()
    })
    expect(ready).toBe(false)
    expect(result.current.ensuring).toBe(false)
    expect(result.current.note).toContain('GAEA_VOICE_V2_ROOT')
  })

  it('异常断线清掉旧 client，允许原地重连', async () => {
    const transports: FakeTransport[] = []
    const { result } = renderHook(() => useVoiceChatV2({
      transportFactory: () => {
        const t = new FakeTransport()
        transports.push(t)
        return t
      },
    }))
    await act(async () => { await result.current.connect() })
    act(() => { transports[0].onopen?.() })
    expect(result.current.connected).toBe(true)
    act(() => { transports[0].onclose?.() })
    expect(result.current.connected).toBe(false)
    expect(result.current.note).toContain('重连')
    expect(sinkMessages).toContainEqual({ type: 'flush' })
    await act(async () => { await result.current.connect() })
    expect(transports).toHaveLength(2)
  })

  it('P1 意外掉线按指数退避自动重连，成功后计数复位', async () => {
    vi.useFakeTimers()
    try {
      const transports: FakeTransport[] = []
      const { result } = renderHook(() => useVoiceChatV2({
        transportFactory: () => {
          const t = new FakeTransport()
          transports.push(t)
          return t
        },
      }))
      await act(async () => { await result.current.connect() })
      act(() => { transports[0].onopen?.() })
      expect(result.current.connected).toBe(true)
      act(() => { transports[0].onclose?.() })
      expect(result.current.note).toContain('自动重连')
      expect(result.current.note).toContain('1/5')
      await act(async () => { await vi.advanceTimersByTimeAsync(1000) })
      expect(transports).toHaveLength(2) // 服务探活 ready → 直接重连
      act(() => { transports[1].onopen?.() })
      expect(result.current.connected).toBe(true)
      // 再掉线：成功已把计数复位，仍从 1/5 开始
      act(() => { transports[1].onclose?.() })
      expect(result.current.note).toContain('1/5')
      await act(async () => { await vi.advanceTimersByTimeAsync(1000) })
      expect(transports).toHaveLength(3)
    } finally {
      vi.useRealTimers()
    }
  })

  it('P1 退避耗尽停链，手动重连复位并恢复', async () => {
    vi.useFakeTimers()
    try {
      const status = app.VoiceV2Status as unknown as ReturnType<typeof vi.fn>
      const ensure = app.VoiceV2Ensure as unknown as ReturnType<typeof vi.fn>
      status.mockResolvedValue({ sidecar: false, llm: false, breeze: false, gateway: false, ready: false, error: 'sidecar down' })
      ensure.mockRejectedValue(new Error('bridge down'))
      const transports: FakeTransport[] = []
      const { result } = renderHook(() => useVoiceChatV2({
        transportFactory: () => {
          const t = new FakeTransport()
          transports.push(t)
          return t
        },
      }))
      await act(async () => { await result.current.connect() })
      act(() => { transports[0].onopen?.() })
      act(() => { transports[0].onclose?.() })
      // 5 次退避总时长 1+2+4+8+15=30s；服务始终不可用 → 只耗链不重连
      await act(async () => { await vi.advanceTimersByTimeAsync(30000) })
      expect(result.current.note).toContain('自动重连未成功')
      expect(transports).toHaveLength(1)
      await act(async () => { await vi.advanceTimersByTimeAsync(60000) })
      expect(transports).toHaveLength(1) // 链已耗尽，不再尝试
      // 服务恢复 + 手动重连：计数复位、能连上
      status.mockResolvedValue({ sidecar: true, llm: true, breeze: true, gateway: true, ready: true, error: '' })
      await act(async () => { await result.current.connect() })
      act(() => { transports[1].onopen?.() })
      expect(result.current.connected).toBe(true)
      act(() => { transports[1].onclose?.() })
      expect(result.current.note).toContain('1/5') // 从头计
    } finally {
      vi.useRealTimers()
    }
  })

  it('P1 有意断开（关台/手动）不触发自动重连', async () => {
    vi.useFakeTimers()
    try {
      const transports: FakeTransport[] = []
      const { result } = renderHook(() => useVoiceChatV2({
        transportFactory: () => {
          const t = new FakeTransport()
          transports.push(t)
          return t
        },
      }))
      await act(async () => { await result.current.connect() })
      act(() => { transports[0].onopen?.() })
      act(() => { result.current.disconnect() })
      expect(result.current.connected).toBe(false)
      await act(async () => { await vi.advanceTimersByTimeAsync(30000) })
      expect(transports).toHaveLength(1)
      expect(result.current.note).toBe('')
    } finally {
      vi.useRealTimers()
    }
  })
})
