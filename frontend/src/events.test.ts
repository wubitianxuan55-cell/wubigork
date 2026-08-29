import { describe, expect, it, vi, afterEach } from 'vitest'
import { BACKEND_EVENTS, FRONTEND_EVENTS, chatStreamChannel, subscribe, subscribeForSpace, emitFrontendEvent } from './events'

// 3.0 01 报告 §4：22 后端事件（+v4.3c 轻语主动关心定时推送）+ 4 前端事件常量表 + subscribe 统一封装

describe('BACKEND_EVENTS（§4.1，23 个）', () => {
  it('共 23 个后端事件常量', () => {
    expect(Object.keys(BACKEND_EVENTS)).toHaveLength(23)
  })

  it('常量名与事件名字面量一一对应（防复制笔误）', () => {
    expect(BACKEND_EVENTS.MODEL_CHANGED).toBe('model-changed')
    expect(BACKEND_EVENTS.FEATURE_MODEL_CHANGED).toBe('feature-model-changed')
    expect(BACKEND_EVENTS.CHAT_STREAM).toBe('chat-stream')
    expect(BACKEND_EVENTS.CREATE_CHAPTER_STREAM).toBe('create-chapter-stream')
    expect(BACKEND_EVENTS.GHOST_STREAM).toBe('ghost-stream')
    expect(BACKEND_EVENTS.XAI_OUTPUT).toBe('xai-output')
    expect(BACKEND_EVENTS.TTS_STREAM).toBe('tts-stream')
    expect(BACKEND_EVENTS.NEW_CHARACTERS_DISCOVERED).toBe('new-characters-discovered')
    expect(BACKEND_EVENTS.CHARACTER_FILL_PROGRESS).toBe('character-fill-progress')
    expect(BACKEND_EVENTS.VOICE_STATE).toBe('voice:state')
    expect(BACKEND_EVENTS.VOICE_TRANSCRIPT).toBe('voice:transcript')
    expect(BACKEND_EVENTS.VOICE_REPLY).toBe('voice:reply')
    expect(BACKEND_EVENTS.VOICE_TTS_AUDIO).toBe('voice:tts-audio')
    expect(BACKEND_EVENTS.VOICE_TTS_SPEAK_TEXT).toBe('voice:tts-speak-text')
    expect(BACKEND_EVENTS.VOICE_TTS_SPEAK_CANCEL).toBe('voice:tts-speak-cancel')
    expect(BACKEND_EVENTS.VOICE_LISTENING).toBe('voice:listening')
    expect(BACKEND_EVENTS.VOICE_THINKING).toBe('voice:thinking')
    expect(BACKEND_EVENTS.GAEA_EVENT).toBe('gaea-event')
    expect(BACKEND_EVENTS.UPDATER_PROGRESS).toBe('updater:progress')
    expect(BACKEND_EVENTS.GAEA_TASK).toBe('gaea-task')
    expect(BACKEND_EVENTS.WHISPER_PROACTIVE).toBe('gaea-whisper-proactive')
    expect(BACKEND_EVENTS.GAEA_READY).toBe('gaea-ready')
  })

  it('chatStreamChannel 生成动态频道（chat-stream:<runID>）', () => {
    expect(chatStreamChannel('run-42')).toBe('chat-stream:run-42')
  })
})

describe('FRONTEND_EVENTS（§4.2，4 个）', () => {
  it('共 4 个前端自定义事件常量', () => {
    expect(Object.keys(FRONTEND_EVENTS)).toHaveLength(4)
  })

  it('常量名与事件名一致', () => {
    expect(FRONTEND_EVENTS.NAVIGATE).toBe('navigate')
    expect(FRONTEND_EVENTS.GAEA_PERSONA_CHANGED).toBe('gaea-persona-changed')
    expect(FRONTEND_EVENTS.GAEA_PROJECT_CHARS_CHANGED).toBe('gaea-project-chars-changed')
    expect(FRONTEND_EVENTS.AI_ASSIST_SEND).toBe('ai-assist-send')
  })

  it('emitFrontendEvent 分发 CustomEvent 并可被监听', () => {
    const handler = vi.fn()
    window.addEventListener(FRONTEND_EVENTS.NAVIGATE, handler)
    emitFrontendEvent(FRONTEND_EVENTS.NAVIGATE, { page: 'novel' })
    expect(handler).toHaveBeenCalledTimes(1)
    const ev = handler.mock.calls[0][0] as CustomEvent
    expect(ev.detail).toEqual({ page: 'novel' })
    window.removeEventListener(FRONTEND_EVENTS.NAVIGATE, handler)
  })
})

describe('subscribe（§4.3 两套并存收敛）', () => {
  afterEach(() => {
    window.runtime = undefined
  })

  it('window.runtime 缺失时返回 noop（事件通道缺失不抛错）', () => {
    expect(() => subscribe('x', () => {})).not.toThrow()
  })

  it('window.runtime.EventsOn 返回卸载函数时直接复用', () => {
    const unsub = vi.fn()
    window.runtime = { EventsOn: () => unsub } as unknown as Window['runtime']
    const got = subscribe('evt', () => {})
    expect(typeof got).toBe('function')
    got()
    expect(unsub).toHaveBeenCalledTimes(1)
  })

  it('window.runtime.EventsOn 返回 void 时用 EventsOff 卸载', () => {
    const off = vi.fn()
    window.runtime = {
      EventsOn: () => {},
      EventsOff: off,
    } as unknown as Window['runtime']
    const got = subscribe('evt', () => {})
    got()
    expect(off).toHaveBeenCalledWith('evt', expect.any(Function))
  })
})

describe('subscribeForSpace（S2.1 事件空间过滤）', () => {
  afterEach(() => {
    window.runtime = undefined
  })

  it('payload.spaceId 与当前空间不匹配的事件被丢弃，全局事件放行', () => {
    const unsub = vi.fn()
    window.runtime = { EventsOn: vi.fn(() => unsub) } as unknown as Window['runtime']
    const handler = vi.fn()
    const off = subscribeForSpace('gaea-task', handler, 'work')
    const cb = (window.runtime!.EventsOn as ReturnType<typeof vi.fn>).mock.calls[0][1] as (data: unknown) => void
    cb({ id: 't1', spaceId: 'work' })
    cb({ id: 't2', spaceId: 'play' })
    cb({ id: 't3' })
    expect(handler).toHaveBeenCalledTimes(2)
    expect(handler.mock.calls[0][0]).toMatchObject({ id: 't1' })
    expect(handler.mock.calls[1][0]).toMatchObject({ id: 't3' })
    off()
    expect(unsub).toHaveBeenCalledTimes(1)
  })

  it('payload.space 别名同样过滤', () => {
    window.runtime = { EventsOn: vi.fn() } as unknown as Window['runtime']
    const handler = vi.fn()
    subscribeForSpace('x', handler, 'play')
    const cb = (window.runtime!.EventsOn as ReturnType<typeof vi.fn>).mock.calls[0][1] as (data: unknown) => void
    cb({ space: 'work' })
    cb({ space: 'play' })
    expect(handler).toHaveBeenCalledTimes(1)
    expect(handler.mock.calls[0][0]).toMatchObject({ space: 'play' })
  })

  it('无 space 参数 = 普通 subscribe（全部放行）', () => {
    window.runtime = { EventsOn: vi.fn() } as unknown as Window['runtime']
    const handler = vi.fn()
    subscribeForSpace('model-changed', handler)
    const cb = (window.runtime!.EventsOn as ReturnType<typeof vi.fn>).mock.calls[0][1] as (data: unknown) => void
    cb({ model: 'x' })
    cb({ spaceId: 'play' })
    expect(handler).toHaveBeenCalledTimes(2)
  })
})
