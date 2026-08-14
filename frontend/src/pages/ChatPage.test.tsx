// T6-3「对话·流可靠」ChatPage 组件测试（frontend 侧）：
//  - T6-3.1 流订阅竞态与超时：订阅先于收帧（首帧不丢）、30s 无帧超时后
//    sending=false 且 finally 执行、done/error/启动失败终态复位
//  - T6-3.3 语音持久化与资源清理：ChatAppendMessages 落库、朗读 URL revoke、
//    模拟打字循环取消（切话题中止、卸载收尾）
//  - T6-3.4 迁移一次性：持久化标记 + 二次初始化仅执行一次 + 失败不写标记
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen } from '@testing-library/react'

// ── 运行时事件 mock（先于 import 定义，供 vi.mock 工厂引用）─────────────────
const runtimeMock = vi.hoisted(() => ({
  handlers: {} as Record<string, (payload: any) => void>,
  unsubs: [] as (() => void)[],
}))

vi.mock('../../wailsjs/runtime/runtime', () => ({
  EventsOn: vi.fn((name: string, cb: (payload: any) => void) => {
    runtimeMock.handlers[name] = cb
    const un = () => { delete runtimeMock.handlers[name] }
    runtimeMock.unsubs.push(un)
    return un
  }),
  EventsOff: vi.fn(),
}))

// ── 语音 hook mock：捕获 ChatPage 注册的回调，测试可直接触发 ───────────────
const voiceMock = vi.hoisted(() => ({
  opts: {} as { onTranscript?: (t: string) => void; onReply?: (t: string) => void },
}))

vi.mock('../hooks/useVoiceChat', () => ({
  useVoiceChat: (options?: any) => {
    voiceMock.opts = options || {}
    return {
      state: { active: false, listening: false, speaking: false, aiSpeaking: false, transcript: '', finalTranscript: '', volume: 0, error: null, mode: 'vad' },
      start: vi.fn(), stop: vi.fn(), setPTT: vi.fn(), interrupt: vi.fn(),
    }
  },
}))

// ── Wails 绑定 mock（ChatPage 经 wailsjsCompat 调用）───────────────────────
vi.mock('../../src/wailsjsCompat', () => ({
  ChatTopicsList: vi.fn(),
  ChatMessagesList: vi.fn(),
  ChatAppendMessages: vi.fn(),
  ChatStreamPlain: vi.fn(),
  ChatSend: vi.fn(),
  ChatImportTopic: vi.fn(),
  ChatTopicCreate: vi.fn(),
  ChatTopicDelete: vi.fn(),
  ChatTopicRename: vi.fn(),
  ChatTopicSetMode: vi.fn(),
  ChatTopicClear: vi.fn(),
  ChatTopicExportMarkdown: vi.fn(),
  WhisperGetPersonalities: vi.fn(),
  WhisperClearSession: vi.fn(),
  VoiceApplySettings: vi.fn(),
  TTSSpeakBase64: vi.fn(),
  GaeaLogFrontendError: vi.fn(),
}))

// ── 纯视觉/重依赖组件：测试聚焦 ChatPage 逻辑，全部替换为轻量桩 ────────────
vi.mock('../components/VoiceChatOrb', () => ({ default: () => <div data-testid="voice-orb" /> }))
vi.mock('../components/CompanionAvatar', () => ({ CompanionAvatar: () => <div data-testid="companion-avatar" /> }))
vi.mock('../components/ParticleFlow', () => ({ ParticleFlow: () => null }))
vi.mock('../components/SoundWaveOverlay', () => ({ SoundWaveOverlay: () => null }))
vi.mock('../components/FeatureModelBar', () => ({ default: () => null }))
vi.mock('../components/PersonaPicker', () => ({ default: () => null }))
vi.mock('../components/VoiceSettingsPanel', () => ({ default: () => null }))

import ChatPage, { STREAM_SILENCE_TIMEOUT_MS } from './ChatPage'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import * as AppCompat from '../../src/wailsjsCompat'
import {
  ChatTopicsList, ChatMessagesList, ChatStreamPlain,
  ChatSend, ChatImportTopic, ChatTopicCreate, TTSSpeakBase64, GaeaLogFrontendError,
} from '../../src/wailsjsCompat'

// ChatAppendMessages 为 T6-3 新绑定：wailsjs/go 生成物待 wails build 再生成，
// 测试侧经兼容层命名空间取用（与 ChatPage 内 (App as any) 逃生口一致）。
const ChatAppendMessages = (AppCompat as unknown as Record<string, unknown>).ChatAppendMessages as ReturnType<typeof vi.fn>

const TOPIC_PLAIN = { id: 't1', title: '新对话', mode: 'plain', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' }
const LONG_REPLY = '这是一段足够长的模拟角色回复内容，专门用于触发前端模拟打字流以验证取消逻辑的正确性。'

/** 冲刷微任务链（初始化/发送链路的 promise 续延）。 */
async function flushAsync() {
  await act(async () => {
    for (let i = 0; i < 30; i++) await Promise.resolve()
  })
}

/** 完成初始化（话题列表 → 消息列表 → 状态落定）。 */
async function renderChat(opts: { topics?: any[]; messages?: any[] } = {}) {
  const topics = opts.topics ?? [TOPIC_PLAIN]
  const messages = opts.messages ?? []
  vi.mocked(ChatTopicsList).mockResolvedValue(topics)
  vi.mocked(ChatMessagesList).mockImplementation(async (id: string) => {
    if (id === topics[0]?.id) return messages
    return []
  })
  const view = render(<ChatPage />)
  await flushAsync()
  return view
}

beforeEach(() => {
  localStorage.clear()
  runtimeMock.handlers = {}
  runtimeMock.unsubs = []
  voiceMock.opts = {}
  vi.clearAllMocks()
  // 默认实现（clearAllMocks 只清调用记录，这里显式重设保证用例间隔离）
  vi.mocked(ChatTopicsList).mockResolvedValue([TOPIC_PLAIN])
  vi.mocked(ChatMessagesList).mockResolvedValue([])
  vi.mocked(ChatAppendMessages).mockResolvedValue(undefined)
  vi.mocked(ChatStreamPlain).mockResolvedValue('run-1')
  vi.mocked(ChatSend).mockResolvedValue({ reply: LONG_REPLY, reasoning: '' })
  vi.mocked(ChatImportTopic).mockResolvedValue({ id: 'imp-1', title: '', mode: 'plain', created_at: '', updated_at: '' })
  vi.mocked(ChatTopicCreate).mockResolvedValue({ id: 'new-1', title: '新对话', mode: 'plain', created_at: '', updated_at: '' })
  vi.mocked(TTSSpeakBase64).mockResolvedValue({ base64: '', mimeType: 'audio/mp3' })
  vi.mocked(GaeaLogFrontendError).mockResolvedValue(undefined)
})

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

/** 输入并回车发送一条消息。 */
async function sendMessage(text: string) {
  const ta = screen.getByPlaceholderText(/输入消息/) as HTMLTextAreaElement
  fireEvent.change(ta, { target: { value: text } })
  fireEvent.keyDown(ta, { key: 'Enter' })
  await flushAsync()
}

describe('T6-3.1 流订阅竞态与超时', () => {
  it('订阅先于收帧：runID 一到立即注册精确频道监听，随后派发的首帧不丢', async () => {
    let resolveRunID: (id: string) => void = () => {}
    vi.mocked(ChatStreamPlain).mockImplementation(() => new Promise<string>((r) => { resolveRunID = r }))
    await renderChat()
    await sendMessage('你好')

    // runID 未知：尚无任何订阅
    expect(ChatStreamPlain).toHaveBeenCalled()
    expect(EventsOn).not.toHaveBeenCalled()

    // 解析 runID：订阅必须紧跟（同一微任务），先订阅后收帧
    await act(async () => { resolveRunID('run-1') })
    await flushAsync()
    expect(EventsOn).toHaveBeenCalledWith('chat-stream:run-1', expect.any(Function))
    const handler = runtimeMock.handlers['chat-stream:run-1']
    expect(handler).toBeDefined()

    // 订阅注册后派发首帧 → 全部接收（首帧不丢）
    await act(async () => { handler({ type: 'delta', content: '首' }) })
    expect(await screen.findByText('首')).toBeTruthy()
    await act(async () => { handler({ type: 'delta', content: '帧' }) })
    expect(await screen.findByText('首帧')).toBeTruthy()

    // done 终态：回复落定 + sending 复位（正常路径）
    await act(async () => { handler({ type: 'done', reply: '完整回复内容', reasoning: '' }) })
    expect(await screen.findByText('完整回复内容')).toBeTruthy()
    const ta = screen.getByPlaceholderText(/输入消息/) as HTMLTextAreaElement
    expect(ta.disabled).toBe(false)
    // 终态后监听已清理（finally 执行）
    expect(runtimeMock.handlers['chat-stream:run-1']).toBeUndefined()
  })

  it('超时：30s 无任何帧 → sending=false、错误展示、finally 执行（fake timers）', async () => {
    vi.useFakeTimers()
    await renderChat()
    await sendMessage('你好')
    // runID 已解析、订阅已注册，但后端始终不发任何帧
    await act(async () => {})
    expect(runtimeMock.handlers['chat-stream:run-1']).toBeDefined()
    // 尚未超时：无超时错误文案
    expect(screen.queryByText(/请求超时：/)).toBeNull()

    await act(async () => { vi.advanceTimersByTime(STREAM_SILENCE_TIMEOUT_MS + 1) })

    // 超时按失败展示 + sending 复位（输入框恢复可用）
    expect(screen.getByText(/请求超时：30 秒内未收到回复/)).toBeTruthy()
    const ta = screen.getByPlaceholderText(/输入消息/) as HTMLTextAreaElement
    expect(ta.disabled).toBe(false)
    // finally 执行：订阅已清理（unsubscribe 被调用）
    expect(runtimeMock.handlers['chat-stream:run-1']).toBeUndefined()
    expect(runtimeMock.unsubs.length).toBeGreaterThanOrEqual(1)
  })

  it('error 事件：显示失败并复位 sending', async () => {
    await renderChat()
    await sendMessage('你好')
    await flushAsync()
    const handler = runtimeMock.handlers['chat-stream:run-1']
    await act(async () => { handler({ type: 'error', error: '后端生成异常' }) })
    expect(await screen.findByText(/请求失败：后端生成异常/)).toBeTruthy()
    const ta = screen.getByPlaceholderText(/输入消息/) as HTMLTextAreaElement
    expect(ta.disabled).toBe(false)
    expect(runtimeMock.handlers['chat-stream:run-1']).toBeUndefined()
  })

  it('流启动失败（ChatStreamPlain 拒绝）：显示失败并复位 sending', async () => {
    vi.mocked(ChatStreamPlain).mockRejectedValue(new Error('AI 客户端未初始化'))
    await renderChat()
    await sendMessage('你好')
    expect(await screen.findByText(/请求失败：AI 客户端未初始化/)).toBeTruthy()
    const ta = screen.getByPlaceholderText(/输入消息/) as HTMLTextAreaElement
    expect(ta.disabled).toBe(false)
  })

  it('卸载：挂起的流 Promise 被收尾（终态回调触发，无悬挂监听）', async () => {
    let resolveRunID: (id: string) => void = () => {}
    vi.mocked(ChatStreamPlain).mockImplementation(() => new Promise<string>((r) => { resolveRunID = r }))
    const { unmount } = await renderChat()
    await sendMessage('你好')
    await act(async () => { resolveRunID('run-1') })
    await flushAsync()
    expect(runtimeMock.handlers['chat-stream:run-1']).toBeDefined()
    unmount()
    // 卸载即清理监听
    expect(runtimeMock.handlers['chat-stream:run-1']).toBeUndefined()
  })
})

describe('T6-3.3 语音持久化与资源清理', () => {
  it('语音识别文本/回复经 ChatAppendMessages 落库（不静默吞错路径有日志）', async () => {
    await renderChat()
    await act(async () => { voiceMock.opts.onTranscript?.('语音你好') })
    await act(async () => { voiceMock.opts.onReply?.('语音回复') })
    expect(ChatAppendMessages).toHaveBeenCalledWith('t1', [{ Role: 'user', Content: '语音你好', Extra: '' }])
    expect(ChatAppendMessages).toHaveBeenCalledWith('t1', [{ Role: 'assistant', Content: '语音回复', Extra: '' }])
    expect(await screen.findByText('语音你好')).toBeTruthy()
    expect(await screen.findByText('语音回复')).toBeTruthy()
  })

  it('语音落库失败：记录 gaea.log，界面不受影响', async () => {
    vi.mocked(ChatAppendMessages).mockRejectedValue(new Error('db down'))
    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    await renderChat()
    await act(async () => { voiceMock.opts.onTranscript?.('落库会失败') })
    await flushAsync()
    expect(GaeaLogFrontendError).toHaveBeenCalledWith(expect.stringContaining('语音消息落库失败'))
    expect(await screen.findByText('落库会失败')).toBeTruthy()
    errSpy.mockRestore()
  })

  it('朗读：播放结束后 revokeObjectURL 被调用（finally 释放）', async () => {
    const urlCreate = vi.fn(() => 'blob:mock-audio')
    const urlRevoke = vi.fn()
    vi.stubGlobal('URL', { ...(globalThis.URL as any), createObjectURL: urlCreate, revokeObjectURL: urlRevoke })
    vi.mocked(TTSSpeakBase64).mockResolvedValue({ base64: 'AAAA', mimeType: 'audio/mp3' })
    const { container } = await renderChat({ messages: [{ id: 1, topic_id: 't1', role: 'assistant', content: '可朗读内容', extra: '', seq: 1, created_at: '2026-01-01T00:00:00Z' }] })

    const speakBtn = container.querySelector('.anticon-sound')?.closest('button')
    expect(speakBtn).toBeTruthy()
    fireEvent.click(speakBtn!)
    await flushAsync()

    expect(urlCreate).toHaveBeenCalledTimes(1)
    expect(urlRevoke).toHaveBeenCalledTimes(1)
  })
})

describe('T6-3.3 模拟打字循环取消', () => {
  it('切话题即中止模拟打字流：sending 复位、旧回复不再落进当前视图', async () => {
    vi.useFakeTimers()
    const personaTopic = { id: 'p1', title: '角色话题', mode: 'persona-1', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-03T00:00:00Z' }
    const plainTopic = { id: 't2', title: '普通话题', mode: 'plain', created_at: '2026-01-02T00:00:00Z', updated_at: '2026-01-02T00:00:00Z' }
    await renderChat({ topics: [personaTopic, plainTopic] })
    await sendMessage('陪我聊聊')
    await act(async () => {}) // ChatSend 解析 → 打字循环启动

    // 推进若干帧（模拟打字进行中）
    for (let i = 0; i < 5; i++) {
      await act(async () => { vi.advanceTimersByTime(14) })
    }

    // 中途切换话题 → 循环应中止
    fireEvent.click(screen.getByText('普通话题'))
    await act(async () => { vi.advanceTimersByTime(14 * 200) })
    await act(async () => {})

    // sending 复位（输入框恢复可用）
    const ta = screen.getByPlaceholderText(/输入消息/) as HTMLTextAreaElement
    expect(ta.disabled).toBe(false)
    // 已切到普通话题（无消息欢迎屏），角色回复未展示
    expect(screen.queryByText(/模拟角色回复/)).toBeNull()
  })

  it('卸载后继续推进计时器：无异常、无遗留更新（循环已中止）', async () => {
    vi.useFakeTimers()
    const personaTopic = { id: 'p1', title: '角色话题', mode: 'persona-1', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' }
    const { unmount } = await renderChat({ topics: [personaTopic] })
    await sendMessage('陪我聊聊')
    await act(async () => {})
    for (let i = 0; i < 3; i++) {
      await act(async () => { vi.advanceTimersByTime(14) })
    }
    unmount()
    // 卸载后推进剩余计时器：不应抛错（React 18 无卸载 setState 警告，循环已中止）
    await act(async () => { vi.advanceTimersByTime(14 * 500) })
  })
})

describe('T6-3.4 迁移一次性', () => {
  const legacy = [
    { id: 'l1', title: '旧话题一', messages: [{ role: 'user', content: '旧消息1' }] },
    { id: 'l2', title: '旧话题二', messages: [{ role: 'assistant', content: '旧消息2' }] },
  ]

  it('二次初始化迁移仅执行一次（持久化标记跳过）', async () => {
    localStorage.setItem('gaea_chat_topics', JSON.stringify(legacy))
    const { unmount } = await renderChat({ topics: [] })
    expect(ChatImportTopic).toHaveBeenCalledTimes(2)
    expect(ChatImportTopic).toHaveBeenCalledWith('旧话题一', 'plain', [{ Role: 'user', Content: '旧消息1', Extra: '' }])
    expect(ChatImportTopic).toHaveBeenCalledWith('旧话题二', 'plain', [{ Role: 'assistant', Content: '旧消息2', Extra: '' }])
    // 成功后：写标记 + 清理旧键
    expect(localStorage.getItem('gaea_chat_migration_v1')).toBe('1')
    expect(localStorage.getItem('gaea_chat_topics')).toBeNull()

    // 重新挂载（新的初始化）：标记存在 → 迁移跳过，ChatImportTopic 不再调用
    unmount()
    vi.mocked(ChatTopicsList).mockClear()
    vi.mocked(ChatImportTopic).mockClear()
    render(<ChatPage />)
    await flushAsync()
    expect(ChatImportTopic).not.toHaveBeenCalled()
    expect(ChatTopicsList).toHaveBeenCalled()
  })

  it('迁移失败：记日志、不写标记、保留旧键（下次启动可重试，本次会话内不无限重试）', async () => {
    localStorage.setItem('gaea_chat_topics', JSON.stringify(legacy))
    vi.mocked(ChatImportTopic).mockRejectedValue(new Error('db down'))
    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    await renderChat({ topics: [] })
    await flushAsync()
    // 每个旧话题尝试一次，不重试（本次会话内仅一次）
    expect(ChatImportTopic).toHaveBeenCalledTimes(2)
    expect(GaeaLogFrontendError).toHaveBeenCalledWith(expect.stringContaining('旧话题导入失败'))
    expect(errSpy).toHaveBeenCalled()
    // 失败：不写标记、不清理旧键
    expect(localStorage.getItem('gaea_chat_migration_v1')).toBeNull()
    expect(localStorage.getItem('gaea_chat_topics')).not.toBeNull()
    errSpy.mockRestore()
  })
})
