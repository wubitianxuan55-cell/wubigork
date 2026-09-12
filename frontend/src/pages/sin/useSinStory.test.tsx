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
      extra: JSON.stringify({
        illustrations: { '0': 'C:/tmp/art.png' },
        reasoning: 'r',
        tools: [{ id: 'call_9', name: 'sin_cast', args: '{"name":"林晚"}', output: '角色卡：林晚', error: '', elapsed_ms: 4, read_only: true }],
      }),
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
    // 重开故事：过程卡从 extra.tools 还原（历史消息没有运行态）
    expect(assistant.tools).toEqual([{
      id: 'call_9', name: 'sin_cast', args: '{"name":"林晚"}', output: '角色卡：林晚',
      error: '', elapsed_ms: 4, read_only: true, status: 'done',
    }])
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

  it('过程帧：reasoning 累加、dispatch→result 按 id 合并、notice 进提示条、done.tools 覆盖累积', async () => {
    bridgeMock.SinStream.mockResolvedValue('ss_proc')
    const { result } = renderHook(() => useSinStory())
    await waitFor(() => expect(result.current.initializing).toBe(false))
    await act(async () => { void result.current.send('写开场') })
    await waitFor(() => expect(eventsMock.handlers['sin-stream:ss_proc']).toBeTruthy())

    const frame = (p: Record<string, unknown>) => { eventsMock.handlers['sin-stream:ss_proc'](p) }
    const lastMsg = () => result.current.messages[result.current.messages.length - 1]

    act(() => {
      frame({ type: 'notice', message: '当前模型不支持工具调用，已按纯写作继续' })
      frame({ type: 'reasoning', content: '先查资料' })
      frame({ type: 'tool_dispatch', id: 'call_1', name: 'web_search', args: '{"query":"唐末长安"}', read_only: true })
    })
    await waitFor(() => {
      expect(lastMsg().reasoning).toBe('先查资料')
      expect(lastMsg().tools).toHaveLength(1)
      expect(lastMsg().tools?.[0].status).toBe('running')
    })
    expect(result.current.notice).toContain('不支持工具调用')

    act(() => {
      frame({ type: 'tool_result', id: 'call_1', name: 'web_search', output: '3 条结果', error: '', elapsed_ms: 900 })
      // 丢帧兜底：没有 dispatch 的 result 也如实补一行（不静默吞掉调用事实）
      frame({ type: 'tool_result', id: 'call_orphan', name: 'sin_notes', output: '已记录便签 #0' })
    })
    await waitFor(() => {
      expect(lastMsg().tools?.[0]).toMatchObject({ status: 'done', output: '3 条结果', elapsed_ms: 900 })
      expect(lastMsg().tools?.[1]).toMatchObject({ id: 'call_orphan', status: 'done', args: '' })
    })

    act(() => {
      frame({
        type: 'done', reply: '正文。', message_id: 11, reasoning: '先查资料',
        tools: [{
          id: 'call_1', name: 'web_search', args: '{"query":"唐末长安"}',
          output: '3 条结果', error: '', elapsed_ms: 900, read_only: true,
        }],
      })
    })
    await waitFor(() => expect(result.current.sending).toBe(false))
    // done.tools 是权威轨迹：整段覆盖（孤儿行被清掉，与落库一致）
    expect(lastMsg().tools).toEqual([{
      id: 'call_1', name: 'web_search', args: '{"query":"唐末长安"}',
      output: '3 条结果', error: '', elapsed_ms: 900, read_only: true, status: 'done',
    }])
    expect(lastMsg().key).toBe('db_11')
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

  it('reloadMessages：从后端重读当前故事消息（画廊重新生成后刷新用）', async () => {
    const { result } = renderHook(() => useSinStory())
    await waitFor(() => expect(result.current.initializing).toBe(false))
    bridgeMock.SinMessages.mockResolvedValue([
      { id: 1, topic_id: 'sin_1', role: 'user', content: '开场', extra: '', seq: 1, created_at: '2026-09-12 10:00:00' },
      {
        id: 2, topic_id: 'sin_1', role: 'assistant',
        content: '雨落在窗上。\n@@插图|灯下的女人@@',
        extra: JSON.stringify({ illustrations: { '0': 'C:/tmp/art-regen.png' } }),
        seq: 2, created_at: '2026-09-12 10:00:01',
      },
    ])
    await act(() => result.current.reloadMessages())
    expect(result.current.messages[1].illustrations).toEqual({ '0': 'C:/tmp/art-regen.png' })
  })

  it('帧到达会重置静默计时：工具循环的长回合不误判超时（v4.262）', async () => {
    vi.useFakeTimers()
    bridgeMock.SinStream.mockResolvedValue('ss_long')
    const { result } = renderHook(() => useSinStory())
    await act(async () => { await Promise.resolve() })
    await act(async () => { void result.current.send('写下一幕') })
    await act(async () => { await vi.advanceTimersByTimeAsync(SIN_STREAM_SILENCE_TIMEOUT_MS - 5_000) })
    // 半程收到一帧（工具结果）→ 计时重置
    act(() => {
      eventsMock.handlers['sin-stream:ss_long']({
        type: 'tool_result', id: 'c1', name: 'web_search', output: '3 条结果', error: '', elapsed_ms: 800,
      })
    })
    await act(async () => { await vi.advanceTimersByTimeAsync(SIN_STREAM_SILENCE_TIMEOUT_MS - 5_000) })
    // 改前：一次性计时器在 stream 开始 +90s 就判超时（真机走查：正文已落库却报超时）
    expect(result.current.messages[result.current.messages.length - 1].error).toBeFalsy()
    act(() => {
      eventsMock.handlers['sin-stream:ss_long']({ type: 'done', reply: '正文。', message_id: 21, reasoning: '' })
    })
    await act(async () => { await vi.advanceTimersByTimeAsync(10) })
    expect(result.current.messages[result.current.messages.length - 1].content).toContain('正文')
    expect(result.current.sending).toBe(false)
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
