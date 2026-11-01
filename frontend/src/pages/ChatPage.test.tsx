// T6-3「对话·流可靠」ChatPage 组件测试（frontend 侧）：
//  - T6-3.1 流订阅竞态与超时：订阅先于收帧（首帧不丢）、30s 无帧超时后
//    sending=false 且 finally 执行、done/error/启动失败终态复位
//  - T6-3.3 语音持久化与资源清理：ChatAppendMessages 落库、朗读 URL revoke、
//    模拟打字循环取消（切话题中止、卸载收尾）
//  - T6-3.4 迁移一次性：持久化标记 + 二次初始化仅执行一次 + 失败不写标记
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen, within } from '@testing-library/react'

// ── 运行时事件 mock（先于 import 定义，供 vi.mock 工厂引用）─────────────────
interface ChatTopicLike {
  id: string
  title: string
  mode: string
  created_at: string
  updated_at: string
}
interface ChatMessageLike {
  id: number
  topic_id: string
  role: string
  content: string
  extra?: string
  seq: number
  created_at: string
}
const runtimeMock = vi.hoisted(() => ({
  handlers: {} as Record<string, (payload: unknown) => void>,
  unsubs: [] as (() => void)[],
}))

vi.mock('../../wailsjs/runtime/runtime', () => ({
  EventsOn: vi.fn((name: string, cb: (payload: unknown) => void) => {
    runtimeMock.handlers[name] = cb
    const un = () => { delete runtimeMock.handlers[name] }
    runtimeMock.unsubs.push(un)
    return un
  }),
  EventsOff: vi.fn(),
}))

// ── 语音 hook mock：捕获 ChatPage 注册的回调，测试可直接触发 ───────────────
const voiceMock = vi.hoisted(() => ({
  opts: {} as {
    onTranscript?: (t: string) => void
    onReply?: (t: string) => void
    onReplyDelta?: (t: string) => void
  },
  start: vi.fn(),
  stop: vi.fn(),
  interrupt: vi.fn(),
}))

vi.mock('../hooks/useVoiceChat', () => ({
  // VoiceConsole 设备快切（角色音色线）消费的设备工具桩：jsdom 无媒体设备，
  // 列表恒空、无已选设备（Select 恒隐藏，交互断言不依赖）
  loadMicDeviceId: () => '',
  saveMicDeviceId: () => {},
  listMicDevices: async () => [],
  useVoiceChat: (options?: {
    onTranscript?: (t: string) => void
    onReply?: (t: string) => void
    onReplyDelta?: (t: string) => void
  }) => {
    voiceMock.opts = options || {}
    return {
      state: { active: false, listening: false, speaking: false, aiSpeaking: false, transcript: '', finalTranscript: '', volume: 0, error: null, mode: 'vad', suggestedMic: null },
      start: voiceMock.start, stop: voiceMock.stop, setPTT: vi.fn(), interrupt: voiceMock.interrupt,
      switchMic: vi.fn(async () => {}),
    }
  },
}))

// ── Wails 绑定 mock（ChatPage 经 gaea/lib/bridge app 代理调用）────────────────
// P3 版3 双轨退役：批次一 useChatVoice（ChatAppendMessages/VoiceApplySettings）、
// 批次二 useChatTopics/useChatStream/pages/chat/utils（chat 组方法 +
// LogFrontendError）与批次三a 语音族（ChatPage 本体 WhisperClearSession/
// TTSSpeakBase64/TTSSpeakBase64WithParams/ChatTopicClear/ChatTopicExportMarkdown +
// useVoiceChat VoiceStart/VoiceStop）均已改走 gaea/lib/bridge 的 app 代理——它们
// 用到的绑定与 wailsjsCompat mock 共享同一 vi.fn 引用（bindingsBridge.xxx），
// 测试断言两端皆命中（wailsjsCompat mock 的这些 key 现指向 bindingsBridge 同一 fn）。
const bindingsBridge = vi.hoisted(() => ({
  ChatAppendMessages: vi.fn(),
  VoiceApplySettings: vi.fn(),
  VoiceStart: vi.fn(),
  VoiceStop: vi.fn(),
  ChatTopicsList: vi.fn(),
  ChatMessagesList: vi.fn(),
  // v4.370 分页：默认把 renderChat 注入的初始消息整页返回（无更早历史）
  ChatMessagesPage: vi.fn(),
  ChatStreamPlain: vi.fn(),
  ChatStreamPersona: vi.fn(),
  ChatStreamCancel: vi.fn(),
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
  // bridge 短名 LogFrontendError ↔ wailsjsCompat 直名 GaeaLogFrontendError
  LogFrontendError: vi.fn(),
  TTSSpeakBase64: vi.fn(),
  TTSSpeakBase64WithParams: vi.fn(),
}))
vi.mock('../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../gaea/lib/bridge')>()
  return {
    ...actual,
    app: new Proxy({} as typeof actual.app, {
      get(_t, prop: string) {
        if (prop === 'ChatAppendMessages') return bindingsBridge.ChatAppendMessages
        if (prop === 'VoiceApplySettings') return bindingsBridge.VoiceApplySettings
        if (prop === 'VoiceStart') return bindingsBridge.VoiceStart
        if (prop === 'VoiceStop') return bindingsBridge.VoiceStop
        if (prop === 'ChatTopicsList') return bindingsBridge.ChatTopicsList
        if (prop === 'ChatMessagesList') return bindingsBridge.ChatMessagesList
        if (prop === 'ChatMessagesPage') return bindingsBridge.ChatMessagesPage
        if (prop === 'ChatStreamPlain') return bindingsBridge.ChatStreamPlain
        if (prop === 'ChatStreamPersona') return bindingsBridge.ChatStreamPersona
        if (prop === 'ChatStreamCancel') return bindingsBridge.ChatStreamCancel
        if (prop === 'ChatSend') return bindingsBridge.ChatSend
        if (prop === 'ChatImportTopic') return bindingsBridge.ChatImportTopic
        if (prop === 'ChatTopicCreate') return bindingsBridge.ChatTopicCreate
        if (prop === 'ChatTopicDelete') return bindingsBridge.ChatTopicDelete
        if (prop === 'ChatTopicRename') return bindingsBridge.ChatTopicRename
        if (prop === 'ChatTopicSetMode') return bindingsBridge.ChatTopicSetMode
        if (prop === 'ChatTopicClear') return bindingsBridge.ChatTopicClear
        if (prop === 'ChatTopicExportMarkdown') return bindingsBridge.ChatTopicExportMarkdown
        if (prop === 'WhisperGetPersonalities') return bindingsBridge.WhisperGetPersonalities
        if (prop === 'WhisperClearSession') return bindingsBridge.WhisperClearSession
        if (prop === 'TTSSpeakBase64') return bindingsBridge.TTSSpeakBase64
        if (prop === 'TTSSpeakBase64WithParams') return bindingsBridge.TTSSpeakBase64WithParams
        if (prop === 'LogFrontendError') return bindingsBridge.LogFrontendError
        const fallback = (actual.app as unknown as Record<string, unknown>)[prop]
        if (typeof fallback === 'function') return fallback.bind(actual.app)
        return undefined
      },
    }),
  }
})

// ── 纯视觉/重依赖组件：测试聚焦 ChatPage 逻辑，全部替换为轻量桩 ────────────
vi.mock('../components/VoiceChatOrb', () => ({ default: () => <div data-testid="voice-orb" /> }))
vi.mock('../components/CompanionAvatar', () => ({ CompanionAvatar: () => <div data-testid="companion-avatar" /> }))
vi.mock('../components/ParticleFlow', () => ({ ParticleFlow: () => null }))
vi.mock('../components/SoundWaveOverlay', () => ({ SoundWaveOverlay: () => null }))
vi.mock('../components/PersonaPicker', () => ({ default: () => null }))
vi.mock('../components/VoiceSettingsPanel', () => ({ default: () => null }))

import ChatPage, { STREAM_SILENCE_TIMEOUT_MS } from './ChatPage'
import { EventsOn } from '../../wailsjs/runtime/runtime'
// P3 版3 终局：wailsjsCompat shim 已退役——测试经 bindingsBridge 共享 vi.fn 直取。
const {
  ChatTopicsList, ChatMessagesList, ChatMessagesPage, ChatStreamPlain,
  ChatStreamPersona, ChatStreamCancel, ChatImportTopic, ChatTopicCreate, TTSSpeakBase64,
  TTSSpeakBase64WithParams,
} = bindingsBridge
// bridge 短名 LogFrontendError ↔ 旧直名 GaeaLogFrontendError（断言用直名）
const GaeaLogFrontendError = bindingsBridge.LogFrontendError

// ChatAppendMessages 为 T6-3 新绑定：测试侧经 bindingsBridge 共享 vi.fn 取用。
const ChatAppendMessages = bindingsBridge.ChatAppendMessages as ReturnType<typeof vi.fn>

const TOPIC_PLAIN = { id: 't1', title: '新对话', mode: 'plain', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' }

/** 冲刷微任务链（初始化/发送链路的 promise 续延）。 */
async function flushAsync() {
  await act(async () => {
    for (let i = 0; i < 30; i++) await Promise.resolve()
  })
}

/** 完成初始化（话题列表 → 消息列表 → 状态落定）。 */
async function renderChat(opts: { topics?: ChatTopicLike[]; messages?: ChatMessageLike[] } = {}) {
  const topics = opts.topics ?? [TOPIC_PLAIN]
  const messages = opts.messages ?? []
  vi.mocked(ChatTopicsList).mockResolvedValue(topics)
  vi.mocked(ChatMessagesList).mockImplementation(async (id: string) => {
    if (id === topics[0]?.id) return messages
    return []
  })
  // v4.370 分页：ChatMessagesPage 返回同一份消息（page 形状，无更早历史）
  vi.mocked(ChatMessagesPage).mockImplementation(async (id: string) => {
    if (id === topics[0]?.id) return { messages, hasMore: false, oldestSeq: messages[0]?.seq ?? 0 }
    return { messages: [], hasMore: false, oldestSeq: 0 }
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
  vi.mocked(voiceMock.start).mockResolvedValue(true)
  // 默认实现（clearAllMocks 只清调用记录，这里显式重设保证用例间隔离）
  vi.mocked(ChatTopicsList).mockResolvedValue([TOPIC_PLAIN])
  vi.mocked(ChatMessagesList).mockResolvedValue([])
  vi.mocked(ChatMessagesPage).mockResolvedValue({ messages: [], hasMore: false, oldestSeq: 0 })
  vi.mocked(ChatAppendMessages).mockResolvedValue(undefined)
  vi.mocked(ChatStreamPlain).mockResolvedValue('run-1')
  vi.mocked(ChatStreamPersona).mockResolvedValue('run-persona')
  vi.mocked(ChatStreamCancel).mockResolvedValue(true)
  vi.mocked(ChatImportTopic).mockResolvedValue({ id: 'imp-1', title: '', mode: 'plain', created_at: '', updated_at: '' })
  vi.mocked(ChatTopicCreate).mockResolvedValue({ id: 'new-1', title: '新对话', mode: 'plain', created_at: '', updated_at: '' })
  vi.mocked(TTSSpeakBase64).mockResolvedValue({ base64: '', mimeType: 'audio/mp3' })
  vi.mocked(TTSSpeakBase64WithParams).mockResolvedValue({ base64: '', mimeType: 'audio/mp3' })
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

    // runID 未知：尚无任何流订阅
    // （v3「星枢」ChatInspector 会订阅 feature-model-changed 事件——功能模型状态，
    //   与流订阅无关；此处只断言 chat-stream 频道订阅尚未发生）
    expect(ChatStreamPlain).toHaveBeenCalled()
    expect(EventsOn).not.toHaveBeenCalledWith('chat-stream:run-1', expect.any(Function))

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

describe('专用语音交互界面', () => {
  it('从输入区进入语音台，返回后恢复文字对话', async () => {
    await renderChat()
    fireEvent.click(screen.getByLabelText('语音输入'))
    const voiceConsole = await screen.findByTestId('chat-voice-console')
    expect(within(voiceConsole).getByRole('heading', { name: /语音对话/ })).toBeTruthy()
    expect(screen.getByRole('button', { name: '结束语音' })).toBeTruthy()

    fireEvent.click(screen.getByRole('button', { name: '返回文字对话' }))
    await flushAsync()
    expect(screen.queryByTestId('chat-voice-console')).toBeNull()
    expect(screen.getByRole('textbox', { name: '聊天消息' })).toBeTruthy()
  })

  it('语音启动失败时保留可重试界面，不伪装成进行中的语音会话', async () => {
    vi.mocked(voiceMock.start).mockResolvedValue(false)
    await renderChat()
    fireEvent.click(screen.getByLabelText('语音输入'))
    expect(await screen.findByTestId('chat-voice-console')).toBeTruthy()
    await flushAsync()
    expect(screen.getByRole('button', { name: '开始聆听' })).toBeTruthy()
    expect(screen.queryByRole('button', { name: '结束语音' })).toBeNull()
  })

  it('切换话题会结束当前语音会话，避免后续识别串入新话题', async () => {
    const second = { id: 't2', title: '第二会话', mode: 'plain', created_at: '2026-01-02T00:00:00Z', updated_at: '2026-01-02T00:00:00Z' }
    await renderChat({ topics: [TOPIC_PLAIN, second] })
    fireEvent.click(screen.getByLabelText('语音输入'))
    expect(await screen.findByTestId('chat-voice-console')).toBeTruthy()
    fireEvent.click(screen.getByText('第二会话'))
    await flushAsync()
    await act(async () => { voiceMock.opts.onTranscript?.('迟到语音') })
    expect(voiceMock.stop).toHaveBeenCalled()
    expect(ChatAppendMessages).not.toHaveBeenCalled()
    expect(screen.queryByTestId('chat-voice-console')).toBeNull()
  })

  it('语音启动尚未完成时切话题，迟到的成功结果也会被回收', async () => {
    const second = { id: 't2', title: '第二会话', mode: 'plain', created_at: '2026-01-02T00:00:00Z', updated_at: '2026-01-02T00:00:00Z' }
    let finishStart: (started: boolean) => void = () => {}
    vi.mocked(voiceMock.start).mockImplementationOnce(() => new Promise<boolean>((resolve) => {
      finishStart = resolve
    }))
    await renderChat({ topics: [TOPIC_PLAIN, second] })
    fireEvent.click(screen.getByLabelText('语音输入'))
    expect(await screen.findByTestId('chat-voice-console')).toBeTruthy()

    fireEvent.click(screen.getByText('第二会话'))
    await flushAsync()
    await act(async () => { finishStart(true) })
    await flushAsync()

    expect(voiceMock.stop.mock.calls.length).toBeGreaterThanOrEqual(2)
    expect(screen.queryByTestId('chat-voice-console')).toBeNull()
  })
})

describe('T6-3.3 语音持久化与资源清理', () => {
  it('语音识别文本/回复经 ChatAppendMessages 落库（不静默吞错路径有日志）', async () => {
    await renderChat()
    fireEvent.click(screen.getByLabelText('语音输入'))
    await screen.findByTestId('chat-voice-console')
    await flushAsync()
    await act(async () => { voiceMock.opts.onTranscript?.('语音你好') })
    await act(async () => { voiceMock.opts.onReply?.('语音回复') })
    await flushAsync()
    // v4.276 形状普查：useChatVoice 发送载荷已改 camelCase（真源=Go json 标签），断言对齐。
    expect(ChatAppendMessages).toHaveBeenCalledWith('t1', [{ role: 'user', content: '语音你好', extra: '' }])
    expect(ChatAppendMessages).toHaveBeenCalledWith('t1', [{ role: 'assistant', content: '语音回复', extra: '' }])
    expect(screen.getAllByText('语音你好').length).toBeGreaterThan(0)
    expect(await screen.findByText('语音回复')).toBeTruthy()
    // 语音首条消息与文字发送共用同一会话元数据收尾：标题/预览/最近活跃同步。
    expect(bindingsBridge.ChatTopicRename).toHaveBeenCalledWith('t1', '语音你好')
  })

  it('流式语音回复先显示增量，最终终态复用同一条消息', async () => {
    await renderChat()
    fireEvent.click(screen.getByLabelText('语音输入'))
    await screen.findByTestId('chat-voice-console')
    await flushAsync()

    await act(async () => { voiceMock.opts.onTranscript?.('讲个短故事') })
    await act(async () => { voiceMock.opts.onReplyDelta?.('从前有一束光') })
    await act(async () => {
      await new Promise<void>((resolve) => requestAnimationFrame(() => resolve()))
    })
    expect((await screen.findAllByText('从前有一束光')).length).toBeGreaterThan(0)

    await act(async () => { voiceMock.opts.onReply?.('从前有一束光，照亮了夜色。') })
    await flushAsync()
    expect((await screen.findAllByText('从前有一束光，照亮了夜色。')).length).toBeGreaterThan(0)
    expect(screen.queryByText('从前有一束光')).toBeNull()
    expect(ChatAppendMessages).toHaveBeenCalledWith(
      't1',
      [{ role: 'assistant', content: '从前有一束光，照亮了夜色。', extra: '' }],
    )
  })

  it('语音落库失败：记录 gaea.log，界面不受影响', async () => {
    vi.mocked(ChatAppendMessages).mockRejectedValue(new Error('db down'))
    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    await renderChat()
    fireEvent.click(screen.getByLabelText('语音输入'))
    await screen.findByTestId('chat-voice-console')
    await flushAsync()
    await act(async () => { voiceMock.opts.onTranscript?.('落库会失败') })
    await flushAsync()
    expect(GaeaLogFrontendError).toHaveBeenCalledWith(expect.stringContaining('语音消息落库失败'))
    fireEvent.click(screen.getByRole('button', { name: '返回文字对话' }))
    await flushAsync()
    expect(await screen.findByText('落库会失败')).toBeTruthy()
    errSpy.mockRestore()
  })

  it('朗读：播放结束后 revokeObjectURL 被调用（finally 释放）', async () => {
    const urlCreate = vi.fn(() => 'blob:mock-audio')
    const urlRevoke = vi.fn()
    vi.stubGlobal('URL', { ...globalThis.URL, createObjectURL: urlCreate, revokeObjectURL: urlRevoke })
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

describe('T6-3.3 朗读情绪（v4.3d）', () => {
  const personaTopic = { id: 'p1', title: '角色话题', mode: 'persona-1', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' }
  const assistantMsg = (content: string, extra = '') => ({
    id: 1, topic_id: 'p1', role: 'assistant', content, extra, seq: 1, created_at: '2026-01-01T00:00:00Z',
  })
  /** 顶栏模式条宿主（生产 MainLayout 恒提供 #v3-chatmode-host；测试手动挂载以渲染模式条/情绪选择器）。 */
  const mountStripHost = () => {
    const host = document.createElement('div')
    host.id = 'v3-chatmode-host'
    document.body.appendChild(host)
    return host
  }

  it('朗读携带会话最近一轮情绪（extra.emotion → TTSSpeakBase64WithParams）', async () => {
    const messages = [assistantMsg('带着甜蜜情绪的话', JSON.stringify({ emotion: 'SWEET_ATTACHMENT' }))]
    const { container } = await renderChat({ topics: [personaTopic], messages })
    const speakBtn = container.querySelector('.chat-row-assistant .anticon-sound')?.closest('button')
    expect(speakBtn).toBeTruthy()
    fireEvent.click(speakBtn!)
    await flushAsync()

    expect(TTSSpeakBase64WithParams).toHaveBeenCalledWith(
      '带着甜蜜情绪的话',
      expect.objectContaining({ Emotion: 'SWEET_ATTACHMENT' }),
    )
    expect(TTSSpeakBase64).not.toHaveBeenCalled()
  })

  it('无情绪：回退无参数版 TTSSpeakBase64（不携带情绪参数）', async () => {
    const { container } = await renderChat({ topics: [personaTopic], messages: [assistantMsg('普通内容')] })
    const speakBtn = container.querySelector('.chat-row-assistant .anticon-sound')?.closest('button')
    expect(speakBtn).toBeTruthy()
    fireEvent.click(speakBtn!)
    await flushAsync()

    expect(TTSSpeakBase64).toHaveBeenCalledWith('普通内容')
    expect(TTSSpeakBase64WithParams).not.toHaveBeenCalled()
  })

  it('情绪选择 UI：选中后朗读携带所选情绪，且生效情绪徽标更新', async () => {
    const host = mountStripHost()
    try {
      const { container } = await renderChat({ topics: [personaTopic], messages: [assistantMsg('需要愤怒地读的内容')] })
      await flushAsync()

      // 顶栏模式条 portal 已渲染，默认「自动」（跟随会话）
      const trigger = screen.getByLabelText('朗读情绪')
      expect(trigger.textContent).toContain('自动')

      // 打开 Popover 选择「愤怒反击」
      fireEvent.click(trigger)
      fireEvent.click(await screen.findByText('愤怒反击'))
      await flushAsync()

      // 生效情绪徽标更新为所选情绪
      expect(screen.getByLabelText('朗读情绪').textContent).toContain('愤怒反击')

      // 朗读携带所选情绪参数
      const speakBtn = container.querySelector('.chat-row-assistant .anticon-sound')?.closest('button')
      expect(speakBtn).toBeTruthy()
      fireEvent.click(speakBtn!)
      await flushAsync()
      expect(TTSSpeakBase64WithParams).toHaveBeenCalledWith(
        '需要愤怒地读的内容',
        expect.objectContaining({ Emotion: 'ANGRY_ATTACK' }),
      )
    } finally {
      host.remove()
    }
  })
})

describe('T6-3.3 角色真实流取消', () => {
  it('角色流首帧直接上屏，切话题时取消后端流且旧回复不落进当前视图', async () => {
    const personaTopic = { id: 'p1', title: '角色话题', mode: 'persona-1', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-03T00:00:00Z' }
    const plainTopic = { id: 't2', title: '普通话题', mode: 'plain', created_at: '2026-01-02T00:00:00Z', updated_at: '2026-01-02T00:00:00Z' }
    await renderChat({ topics: [personaTopic, plainTopic] })
    await sendMessage('陪我聊聊')
    await flushAsync()
    expect(ChatStreamPersona).toHaveBeenCalled()
    const handler = runtimeMock.handlers['chat-stream:run-persona']
    await act(async () => { handler({ type: 'delta', content: '角色流首帧' }) })
    expect(await screen.findByText('角色流首帧')).toBeTruthy()

    // 中途切换话题 → 后端流应收到取消请求
    fireEvent.click(screen.getByText('普通话题'))
    await flushAsync()
    expect(ChatStreamCancel).toHaveBeenCalledWith('run-persona')
    await act(async () => { handler({ type: 'cancelled', reply: '角色流首帧' }) })
    await flushAsync()

    // sending 复位（输入框恢复可用）
    const ta = screen.getByPlaceholderText(/输入消息/) as HTMLTextAreaElement
    expect(ta.disabled).toBe(false)
    // 已切到普通话题（无消息欢迎屏），角色回复未展示
    expect(screen.queryByText(/模拟角色回复/)).toBeNull()
  })

  it('卸载时主动取消角色流，不留下后台生成', async () => {
    const personaTopic = { id: 'p1', title: '角色话题', mode: 'persona-1', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' }
    const { unmount } = await renderChat({ topics: [personaTopic] })
    await sendMessage('陪我聊聊')
    await flushAsync()
    expect(runtimeMock.handlers['chat-stream:run-persona']).toBeDefined()
    unmount()
    expect(ChatStreamCancel).toHaveBeenCalledWith('run-persona')
    expect(runtimeMock.handlers['chat-stream:run-persona']).toBeUndefined()
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
