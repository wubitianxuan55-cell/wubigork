/**
 * VoiceConsoleV2.test.tsx — 语音台 V2 视图：渲染 + 轮完成写回纪律。
 */
import React from 'react'
import { render, screen, act, fireEvent } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import VoiceConsoleV2 from './VoiceConsoleV2'
import type { VoiceChatV2Api } from '../../voice/runtime/useVoiceChatV2'
import type { VoiceSessionState } from '../../voice/session/voiceSession'
import type { ChatMsg } from '../../pages/chat/types'

// 钩子整桩：测试只关心视图消费与写回纪律
const captured: {
  onTurnComplete?: (t: { user: string; assistant: string; responseId: string; itemId: string }) => void
  abortInFlightCalls?: number
} = {}
const runtimeState = {
  phase: 'idle' as VoiceSessionState['phase'],
  transcripts: [] as string[],
  partialTranscript: '',
  assistantText: '',
  pbText: '',
  tail: [] as string[],
  connected: true,
  micOn: false,
  note: '',
  ready: true,
  ensuring: false,
}
const cancelMock = vi.fn()
vi.mock('../../voice/runtime/useVoiceChatV2', () => ({
  useVoiceChatV2: (opts: {
    onTurnComplete?: (t: { user: string; assistant: string; responseId: string; itemId: string }) => void
  }) => {
    captured.onTurnComplete = opts.onTurnComplete
    return {
      phase: runtimeState.phase,
      transcripts: runtimeState.transcripts,
      partialTranscript: runtimeState.partialTranscript,
      assistantText: runtimeState.assistantText,
      pbText: runtimeState.pbText,
      tail: runtimeState.tail,
      connected: runtimeState.connected,
      micOn: runtimeState.micOn,
      micLevel: 0,
      micSilent: false,
      note: runtimeState.note,
      ensuring: runtimeState.ensuring,
      services: {
        realtime: runtimeState.ready,
        breeze: runtimeState.ready,
        ready: runtimeState.ready,
        error: '',
      },
      connect: vi.fn(async () => {}), disconnect: vi.fn(),
      startMic: vi.fn(async () => {}), stopMic: vi.fn(), cancel: cancelMock,
      updateVADThreshold: vi.fn(),
      abortInFlight: vi.fn(() => { captured.abortInFlightCalls = (captured.abortInFlightCalls ?? 0) + 1 }),
      ensureServices: vi.fn(async () => runtimeState.ready),
      refreshServices: vi.fn(async () => runtimeState.ready),
      replayEvents: vi.fn(async () => {}),
    } satisfies VoiceChatV2Api
  },
}))

// 桥：V2 权威回合收尾桩（聊天事务 + 记忆调度由后端一处完成）
const commitMock = vi.fn(async (
  _topic: string, _user: string, _assistant: string, _responseId: string, _itemId: string,
): Promise<Record<string, unknown>> => ({ committed: true, duplicate: false, memoryScheduled: true }))
const previewIntentMock = vi.fn(async (_text: string): Promise<Record<string, unknown>> => ({ handled: false }))
const executeActionMock = vi.fn(async (_previewID: string, _intentHash: string): Promise<Record<string, unknown>> => ({ executed: false }))
const speakAckMock = vi.fn(async (_text: string): Promise<void> => {})
vi.mock('../../gaea/lib/bridge', () => ({
  app: {
    VoiceV2PreviewIntent: (...args: unknown[]) => previewIntentMock(...(args as [string])),
    VoiceV2ExecuteAction: (...args: unknown[]) => executeActionMock(...(args as [string, string])),
    VoiceV2CommitTurn: (
      topic: string, user: string, assistant: string, responseId: string, itemId: string,
    ) => commitMock(topic, user, assistant, responseId, itemId),
    TTSSpeakStreaming: (...args: unknown[]) => speakAckMock(...(args as [string])),
  },
}))

type VoiceConsoleV2PropsOf = React.ComponentProps<typeof VoiceConsoleV2>

function renderConsole(overrides: Partial<VoiceConsoleV2PropsOf> = {}) {
  const messages: ChatMsg[] = []
  const setMessages = vi.fn((u: ChatMsg[] | ((p: ChatMsg[]) => ChatMsg[])) => {
    const next = typeof u === 'function' ? u(messages) : u
    messages.splice(0, messages.length, ...next)
  })
  const utils = render(
    <VoiceConsoleV2
      companionName="小音"
      instructions="测试指令"
      onClose={() => {}}
      onOpenSettings={() => {}}
      setMessages={setMessages}
      getActiveId={() => 'topic_1'}
      {...overrides}
    />,
  )
  return { messages, setMessages, ...utils }
}

describe('VoiceConsoleV2', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    cancelMock.mockClear()
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(null)
    previewIntentMock.mockResolvedValue({ handled: false })
    executeActionMock.mockResolvedValue({ executed: false })
    captured.abortInFlightCalls = 0
    runtimeState.phase = 'idle'
    runtimeState.transcripts = []
    runtimeState.partialTranscript = ''
    runtimeState.assistantText = ''
    runtimeState.pbText = ''
    runtimeState.tail = []
    runtimeState.connected = true
    runtimeState.micOn = false
    runtimeState.note = ''
    runtimeState.ready = true
    runtimeState.ensuring = false
  })

  it('挂载即连接；标题/相位/开麦钮渲染', () => {
    renderConsole()
    expect(screen.getByTestId('voice-console-v2')).toBeTruthy()
    expect(screen.getAllByText('小音').length).toBeGreaterThanOrEqual(1)
    expect(screen.getByTestId('vc2-phase').textContent).toContain('待机')
    expect(screen.getByTestId('vc2-mic').textContent).toContain('开始聆听')
  })

  it('依赖未就绪显示警示与拉起钮', () => {
    runtimeState.ready = false
    renderConsole()
    expect(screen.getByTestId('vc2-ensure').textContent).toContain('启动本地语音')
    expect(screen.getByTestId('vc2-warn').textContent).toContain('依赖待准备')
    expect(screen.getByRole('heading', { name: '装配本地语音' })).toBeTruthy()
  })

  it('依赖拉起中按钮禁用并显示明确进度态，防止重复点满屏轮询', () => {
    runtimeState.ready = false
    runtimeState.ensuring = true
    renderConsole()
    const button = screen.getByTestId('vc2-ensure') as HTMLButtonElement
    expect(button.disabled).toBe(true)
    expect(button.textContent).toContain('正在准备本地语音')
  })

  it('聆听相位同时驱动声核、文字和主控按钮', () => {
    runtimeState.phase = 'listening'
    runtimeState.micOn = true
    const { container } = renderConsole()
    expect(container.querySelector('.vc2-orb.is-listening')).toBeTruthy()
    expect(screen.getByTestId('vc2-phase').textContent).toContain('正在聆听')
    expect(screen.getByTestId('vc2-mic').textContent).toContain('暂停聆听')
  })

  it('回应中把主操作收束为“打断回答”，不再并列第二个打断按钮', () => {
    runtimeState.phase = 'speaking'
    runtimeState.micOn = true
    renderConsole()
    const primary = screen.getByTestId('vc2-interrupt-primary')
    expect(primary.textContent).toContain('打断回答')
    expect(screen.queryByTestId('vc2-cancel')).toBeNull()
    fireEvent.click(primary)
    expect(cancelMock).toHaveBeenCalledTimes(1)
  })

  it('用户回看历史时不被新转写拉回底部，可一键回到最新', () => {
    runtimeState.transcripts = ['第一轮问题', '第二轮问题']
    runtimeState.assistantText = '这是正在增长的较长回答，用于撑开实时转写区域。'
    renderConsole()
    const thread = screen.getByTestId('vc2-transcripts')
    Object.defineProperties(thread, {
      clientHeight: { configurable: true, value: 300 },
      scrollHeight: { configurable: true, value: 900 },
      scrollTop: { configurable: true, value: 420, writable: true },
    })

    fireEvent.scroll(thread)
    expect(screen.getByTestId('vc2-jump-latest').textContent).toContain('回到最新')

    fireEvent.click(screen.getByTestId('vc2-jump-latest'))
    expect(thread.scrollTop).toBe(900)
    expect(screen.queryByTestId('vc2-jump-latest')).toBeNull()
  })

  it('实时对话流同时呈现用户转写、临时转写和助手回答', () => {
    runtimeState.transcripts = ['帮我安排今天的任务']
    runtimeState.partialTranscript = '下午优先处理'
    runtimeState.assistantText = '我先从下午的事项开始安排。'
    renderConsole()
    expect(screen.getByText('帮我安排今天的任务')).toBeTruthy()
    expect(screen.getByTestId('vc2-transcripts').textContent).toContain('下午优先处理')
    expect(screen.getByTestId('vc2-assistant').textContent).toContain('下午的事项')
  })

  it('轮完成：一声明写回聊天并由后端触发人格记忆收尾', async () => {
    const finalize = vi.fn(async () => undefined)
    const { messages } = renderConsole({ finalizeTopicAfterSend: finalize })
    await act(async () => {
      captured.onTurnComplete?.({
        user: '明天天气怎么样', assistant: '明天晴，适合出门。', responseId: 'r1', itemId: 'item-1',
      })
    })
    expect(commitMock).toHaveBeenCalledWith(
      'topic_1', '明天天气怎么样', '明天晴，适合出门。', 'r1', 'item-1',
    )
    expect(messages.map((m) => m.content)).toEqual(['明天天气怎么样', '明天晴，适合出门。'])
    expect(finalize).toHaveBeenCalledWith('topic_1', '明天天气怎么样')
  })

  it('P0a-A5e 迟到轮仍提交到捕获 topic，但不写已切走视图的 UI', async () => {
    const { rerender, messages, setMessages } = renderConsole()
    rerender(
      <VoiceConsoleV2
        companionName="小音"
        instructions="测试指令"
        onClose={() => {}}
        onOpenSettings={() => {}}
        setMessages={setMessages}
        getActiveId={() => 'topic_OTHER'}
      />,
    )
    await act(async () => {
      captured.onTurnComplete?.({ user: '旧轮', assistant: '旧答', responseId: 'r2', itemId: 'item-2' })
    })
    // 提交仍指向捕获 topic（对话发生在那里；后端幂等兜底）——旧口径「切题即丢弃」
    // 已被用户拍板「继续提交」推翻，串台防线收窄为 UI 增量不写当前视图。
    expect(commitMock).toHaveBeenCalledWith('topic_1', '旧轮', '旧答', 'r2', 'item-2')
    expect(messages).toEqual([])
  })

  it('P0a-A5e 提交链积压中关台：第二笔已完成轮仍提交（继续提交）', async () => {
    let releaseFirst!: () => void
    const firstGate = new Promise<void>((resolve) => { releaseFirst = resolve })
    commitMock
      .mockImplementationOnce(async () => {
        await firstGate
        return { committed: true, duplicate: false, memoryScheduled: true }
      })
      .mockResolvedValueOnce({ committed: true, duplicate: false, memoryScheduled: true })
    const { unmount } = renderConsole()
    await act(async () => {
      captured.onTurnComplete?.({ user: '第一问', assistant: '第一答', responseId: 'r1', itemId: 'item-1' })
      captured.onTurnComplete?.({ user: '第二问', assistant: '第二答', responseId: 'r2', itemId: 'item-2' })
    })
    expect(commitMock).toHaveBeenCalledTimes(1)
    // 第一笔提交阻塞期间用户关闭语音台：第二笔不得被丢弃
    unmount()
    await act(async () => {
      releaseFirst()
      await Promise.resolve()
    })
    expect(commitMock).toHaveBeenCalledTimes(2)
    expect(commitMock).toHaveBeenLastCalledWith('topic_1', '第二问', '第二答', 'r2', 'item-2')
  })

  it('P0a-A5f 卸载即触发生成中精确中止', () => {
    const { unmount } = renderConsole()
    expect(captured.abortInFlightCalls ?? 0).toBe(0)
    unmount()
    expect(captured.abortInFlightCalls).toBe(1)
  })

  it('记忆收尾告警可见，但聊天已提交不会改报保存失败', async () => {
    commitMock.mockResolvedValueOnce({
      committed: true, duplicate: false, memoryScheduled: false,
      warning: '聊天已保存，但人格记忆收尾失败: db busy',
    })
    renderConsole()
    await act(async () => {
      captured.onTurnComplete?.({
        user: '记住我怕冷', assistant: '好，我记住了。', responseId: 'r-warn', itemId: 'item-warn',
      })
    })
    expect((await screen.findByTestId('vc2-note')).textContent).toContain('聊天已保存')
  })

  it('连续轮次按完成顺序串行提交，快速连说不颠倒问答对', async () => {
    let releaseFirst!: () => void
    const firstGate = new Promise<void>((resolve) => { releaseFirst = resolve })
    commitMock
      .mockImplementationOnce(async () => {
        await firstGate
        return { committed: true, duplicate: false, memoryScheduled: true }
      })
      .mockResolvedValueOnce({ committed: true, duplicate: false, memoryScheduled: true })
    const { messages } = renderConsole()

    await act(async () => {
      captured.onTurnComplete?.({ user: '第一问', assistant: '第一答', responseId: 'r1', itemId: 'item-1' })
      captured.onTurnComplete?.({ user: '第二问', assistant: '第二答', responseId: 'r2', itemId: 'item-2' })
    })
    expect(commitMock).toHaveBeenCalledTimes(1)

    await act(async () => {
      releaseFirst()
      await Promise.resolve()
    })
    expect(commitMock).toHaveBeenCalledTimes(2)
    expect(messages.map((m) => m.content)).toEqual(['第一问', '第一答', '第二问', '第二答'])
  })

  it('P1.5 bar 形态：紧凑条呈现相位/绑定/服务健康，不接管工作区', () => {
    runtimeState.phase = 'listening'
    runtimeState.partialTranscript = '今晚我想听一个很短的故事'
    renderConsole({ variant: 'bar', topicTitle: 'P0a-walk-persona' })
    const root = screen.getByTestId('voice-console-v2')
    expect(root.className).toContain('voice-console-v2--bar')
    expect(root.dataset.phase).toBe('listening')
    expect(screen.getByText('正在聆听')).toBeTruthy()
    expect(screen.getByText(/小音 · P0a-walk-persona/)).toBeTruthy()
    expect(screen.getByTestId('vc2-bar-mic')).toBeTruthy()
    expect(screen.getByTestId('vc2-bar-cancel')).toBeTruthy()
    expect(screen.getByTestId('vc2-bar-expand')).toBeTruthy()
    expect(document.querySelectorAll('.vc2-bar-health i.is-online')).toHaveLength(2)
    // 整幅专有区不存在=工作区仍归聊天流
    expect(screen.queryByTestId('vc2-transcripts')).toBeNull()
  })

  it('P1.5 bar→展开→收起：浮层整幅往返，同一实例会话线不断', () => {
    renderConsole({ variant: 'bar', topicTitle: 'T' })
    act(() => { screen.getByTestId('vc2-bar-expand').click() })
    expect(screen.getByTestId('voice-console-v2').className).toContain('voice-console-v2--float')
    expect(screen.getByLabelText('收起语音台')).toBeTruthy()
    act(() => { screen.getByLabelText('收起语音台').click() })
    expect(screen.getByTestId('voice-console-v2').className).toContain('voice-console-v2--bar')
  })

  it('P1.5 浮层 Escape 收起回条（键盘可达），bar 相位为原子化 status 播报', () => {
    renderConsole({ variant: 'bar' })
    act(() => { screen.getByTestId('vc2-bar-expand').click() })
    act(() => {
      window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    })
    expect(screen.getByTestId('voice-console-v2').className).toContain('voice-console-v2--bar')
    // 回 bar 后：相位=原子 status；服务健康=汇总 aria-label（不做竞争性 live）
    const phase = document.querySelector('.vc2-bar-phase')
    expect(phase?.getAttribute('role')).toBe('status')
    expect(phase?.getAttribute('aria-atomic')).toBe('true')
    expect(screen.getByLabelText('语音服务 2/2 就绪')).toBeTruthy()
  })

  it('P1.5 bar 结束=关会话，与整幅关闭同一 onClose 路径', () => {
    const onClose = vi.fn()
    renderConsole({ variant: 'bar', onClose })
    act(() => { screen.getByTestId('vc2-bar-close').click() })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('P2 轮完成命中动作：预览卡带凭据出现（待确认），确认执行恰好一次', async () => {
    previewIntentMock.mockResolvedValue({
      handled: true, executable: true, reply: '将存为任务：写周报', action: 'save_task',
      previewId: 'vap_1', intentHash: 'hash1', ttlMs: 120000,
    })
    executeActionMock.mockResolvedValue({ executed: true, reply: '好，任务已保存。', action: 'save_task' })
    renderConsole()
    await act(async () => {
      captured.onTurnComplete?.({ user: '帮我记个任务写周报', assistant: '好', responseId: 'r9', itemId: 'i9' })
      await Promise.resolve()
    })
    expect(previewIntentMock).toHaveBeenCalledWith('帮我记个任务写周报')
    expect(screen.getByText('将存为任务：写周报')).toBeTruthy()
    expect(screen.getByText('识别到动作 · 待确认')).toBeTruthy()
    await act(async () => {
      screen.getByTestId('vc2-action-exec').click()
      await Promise.resolve()
    })
    expect(executeActionMock).toHaveBeenCalledWith('vap_1', 'hash1')
    expect(screen.queryByText('识别到动作 · 待确认')).toBeNull()
    expect(screen.getByText('好，任务已保存。')).toBeTruthy()
  })

  it('M5 刀一 T0 直执行：executed=true 出回执不出确认卡，并朗读回执', async () => {
    previewIntentMock.mockResolvedValue({
      handled: true, executed: true, reply: '好，已打开运行监测。', action: 'navigate', target: 'runtime',
    })
    renderConsole()
    await act(async () => {
      captured.onTurnComplete?.({ user: '打开运行监测', assistant: '好', responseId: 'r13', itemId: 'i13' })
      await Promise.resolve()
    })
    expect(screen.queryByText('识别到动作 · 待确认')).toBeNull()
    expect(screen.queryByTestId('vc2-action-exec')).toBeNull()
    expect(screen.getByText('好，已打开运行监测。')).toBeTruthy()
    expect(speakAckMock).toHaveBeenCalledWith('好，已打开运行监测。')
  })

  it('M5 刀一 T0 执行失败：executed=false 出失败回执，不朗读不确认', async () => {
    previewIntentMock.mockResolvedValue({
      handled: true, executed: false, reason: 'exec-failed', action: 'navigate', target: 'runtime',
    })
    renderConsole()
    await act(async () => {
      captured.onTurnComplete?.({ user: '打开不存在的板块', assistant: '好', responseId: 'r14', itemId: 'i14' })
      await Promise.resolve()
    })
    expect(screen.queryByTestId('vc2-action-exec')).toBeNull()
    expect(screen.getByText('动作未能执行，请再说一次')).toBeTruthy()
    expect(speakAckMock).not.toHaveBeenCalled()
  })

  it('P2 白名单外动作：可预览不可执行（无执行钮）', async () => {
    previewIntentMock.mockResolvedValue({
      handled: true, executable: false, reply: '将生成图片：日落', action: 'generate_image',
    })
    renderConsole()
    await act(async () => {
      captured.onTurnComplete?.({ user: '画一张日落图', assistant: '好', responseId: 'r12', itemId: 'i12' })
      await Promise.resolve()
    })
    expect(screen.getByText('将生成图片：日落')).toBeTruthy()
    expect(screen.getByText('动作暂不支持语音执行')).toBeTruthy()
    expect(screen.queryByTestId('vc2-action-exec')).toBeNull()
  })

  it('P2 未识别为动作：静默无预览', async () => {
    previewIntentMock.mockResolvedValue({ handled: false })
    renderConsole()
    await act(async () => {
      captured.onTurnComplete?.({ user: '今晚想听故事', assistant: '好', responseId: 'r10', itemId: 'i10' })
      await Promise.resolve()
    })
    expect(screen.queryByText('识别到动作 · 待确认')).toBeNull()
  })

  it('P2 预览解析失败静默，不阻塞提交链', async () => {
    previewIntentMock.mockRejectedValue(new Error('bridge down'))
    renderConsole()
    await act(async () => {
      captured.onTurnComplete?.({ user: '测试', assistant: '答', responseId: 'r11', itemId: 'i11' })
      await Promise.resolve()
    })
    expect(commitMock).toHaveBeenCalledTimes(1)
    expect(screen.queryByText('识别到动作 · 待确认')).toBeNull()
  })
})
