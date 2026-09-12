// sin/useSinStory.test.tsx — 原罪故事状态机（流式续写 / 终态收敛 / 插图回写）。
// 覆盖：初始化选中故事、发送乐观消息 + 流式 delta + done 终态（消息 id 对齐）、
// 启动失败与无帧超时两条失败路径、setIllustration 就地回填。
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, renderHook, waitFor } from '@testing-library/react'

const bridgeMock = vi.hoisted(() => ({
  SinTopicsList: vi.fn(),
  SinTopicCreate: vi.fn(),
  SinTopicRename: vi.fn(),
  SinTopicDelete: vi.fn(),
  SinTopicClear: vi.fn(),
  SinMessages: vi.fn(),
  SinStream: vi.fn(),
  SinCancel: vi.fn(),
}))

const eventsMock = vi.hoisted(() => ({
  handlers: {} as Record<string, (payload: unknown) => void>,
}))

vi.mock('../../gaea/lib/bridge', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../../gaea/lib/bridge')>()),
  app: new Proxy({} as Record<string, unknown>, {
    get(_t, prop: string) {
      return (bridgeMock as unknown as Record<string, unknown>)[prop]
    },
  }),
}))

vi.mock('../../events', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../../events')>()),
  subscribe: (event: string, cb: (payload: unknown) => void) => {
    eventsMock.handlers[event] = cb
    return () => { delete eventsMock.handlers[event] }
  },
}))

import { SIN_STREAM_SILENCE_TIMEOUT_MS, useSinStory } from './useSinStory'

const STORY = { id: 'sin_1', title: '夜行电车', mode: 'sin', created_at: '2026-09-12 10:00:00', updated_at: '2026-09-12 10:00:00', preview: '开场' }

beforeEach(() => {
  bridgeMock.SinTopicsList.mockReset().mockResolvedValue([STORY])
  bridgeMock.SinMessages.mockReset().mockResolvedValue([
    { id: 1, topic_id: 'sin_1', role: 'user', content: '开场', extra: '', seq: 1, created_at: '2026-09-12 10:00:00' },
    {
      id: 2, topic_id: 'sin_1', role: 'assistant',
      content: '雨落在窗上。\n@@插图|灯下的女人@@',
      extra: JSON.stringify({ illustrations: { '0': 'C:/tmp/art.png' }, reasoning: 'r' }),
      seq: 2, created_at: '2026-09-12 10:00:01',
    },
  ])
  bridgeMock.SinTopicCreate.mockReset().mockResolvedValue({ id: 'sin_2', title: '新故事', mode: 'sin' })
  bridgeMock.SinTopicRename.mockReset().mockResolvedValue(undefined)
  bridgeMock.SinStream.mockReset()
  bridgeMock.SinCancel.mockReset().mockResolvedValue(undefined)
  eventsMock.handlers = {}
})

afterEach(() => {
  vi.useRealTimers()
})

describe('useSinStory', () => {
  it('初始化：拉故事列表、选中首个、消息（含插图映射/reasoning）解析入视图', async () => {
    const { result } = renderHook(() => useSinStory())
    await waitFor(() => expect(result.current.initializing).toBe(false))
    expect(result.current.stories.map((s) => s.id)).toEqual(['sin_1'])
    expect(result.current.activeId).toBe('sin_1')
    expect(result.current.messages).toHaveLength(2)
    const assistant = result.current.messages[1]
    expect(assistant.role).toBe('assistant')
    expect(assistant.illustrations).toEqual({ '0': 'C:/tmp/art.png' })
    expect(assistant.reasoning).toBe('r')
  })

  it('发送：乐观消息 → delta 累加 → done 落定（消息 id 对齐为 db_ 键）', async () => {
    let runID = ''
    bridgeMock.SinStream.mockImplementation((_id: string, _text: string) => {
      runID = 'ss_test_1'
      return Promise.resolve(runID)
    })
    const { result } = renderHook(() => useSinStory())
    await waitFor(() => expect(result.current.initializing).toBe(false))

    await act(async () => {
      void result.current.send('继续写')
    })
    await waitFor(() => expect(result.current.sending).toBe(true))
    await waitFor(() => expect(eventsMock.handlers['sin-stream:ss_test_1']).toBeTruthy())

    act(() => { eventsMock.handlers['sin-stream:ss_test_1']({ type: 'delta', content: '她推开门。' }) })
    await waitFor(() => {
      const last = result.current.messages[result.current.messages.length - 1]
      expect(last.content).toBe('她推开门。')
      expect(last.streaming).toBe(true)
    })

    act(() => {
      eventsMock.handlers['sin-stream:ss_test_1']({
        type: 'done', reply: '她推开门。\n@@插图|门缝的光@@', message_id: 9,
        answered_by: { engine: 'herdsman', model: 'qwen3-8b', source: 'feature' },
      })
    })
    await waitFor(() => expect(result.current.sending).toBe(false))
    const last = result.current.messages[result.current.messages.length - 1]
    expect(last.key).toBe('db_9')
    expect(last.messageId).toBe(9)
    expect(last.streaming).toBe(false)
    expect(last.content).toContain('门缝的光')
    expect(result.current.messages.map((m) => m.role)).toEqual(['user', 'assistant', 'user', 'assistant'])
  })

  it('启动失败（SinStream 拒绝）：助手位标错并复位 sending', async () => {
    bridgeMock.SinStream.mockRejectedValue(new Error('AI 客户端未初始化'))
    const { result } = renderHook(() => useSinStory())
    await waitFor(() => expect(result.current.initializing).toBe(false))
    await act(async () => { await result.current.send('写') })
    const last = result.current.messages[result.current.messages.length - 1]
    expect(last.error).toBe(true)
    expect(last.content).toContain('AI 客户端未初始化')
    expect(result.current.sending).toBe(false)
  })

  it('无帧超时：超时阈值后按失败收尾（长文阈值导出给测试共用）', async () => {
    vi.useFakeTimers()
    bridgeMock.SinStream.mockResolvedValue('ss_timeout')
    const { result } = renderHook(() => useSinStory())
    await act(async () => { await Promise.resolve() })
    await act(async () => { void result.current.send('写') })
    await act(async () => { await vi.advanceTimersByTimeAsync(SIN_STREAM_SILENCE_TIMEOUT_MS + 10) })
    const last = result.current.messages[result.current.messages.length - 1]
    expect(last.error).toBe(true)
    expect(last.content).toContain('请求超时')
    expect(result.current.sending).toBe(false)
  })

  it('setIllustration：把生成出的插图路径就地回填到对应消息', async () => {
    const { result } = renderHook(() => useSinStory())
    await waitFor(() => expect(result.current.initializing).toBe(false))
    act(() => { result.current.setIllustration('db_2', '1', 'C:/tmp/new.png') })
    expect(result.current.messages[1].illustrations).toEqual({ '0': 'C:/tmp/art.png', '1': 'C:/tmp/new.png' })
  })

  it('cancel：调用 SinCancel、收掉流式标记、sending 复位（输入框解封）', async () => {
    bridgeMock.SinStream.mockImplementation(() => new Promise<string>(() => {})) // 挂住：保持 sending=true
    const { result } = renderHook(() => useSinStory())
    await waitFor(() => expect(result.current.initializing).toBe(false))
    await act(async () => { void result.current.send('写一段') })
    await waitFor(() => expect(result.current.sending).toBe(true))

    act(() => { result.current.cancel() })
    expect(bridgeMock.SinCancel).toHaveBeenCalledWith('sin_1')
    expect(result.current.sending).toBe(false)
    const last = result.current.messages[result.current.messages.length - 1]
    expect(last.streaming).toBe(false)
    expect(last.content).toBe('（已停止）')
  })
})
