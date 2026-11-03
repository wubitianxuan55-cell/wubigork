/**
 * VoiceConsoleV2.test.tsx — 语音台 V2 视图：渲染 + 轮完成写回纪律。
 */
import React from 'react'
import { render, screen, act } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import VoiceConsoleV2 from './VoiceConsoleV2'
import type { VoiceChatV2Api } from '../../voice/runtime/useVoiceChatV2'
import type { VoiceSessionState } from '../../voice/session/voiceSession'
import type { ChatMsg } from '../../pages/chat/types'

// 钩子整桩：测试只关心视图消费与写回纪律
const captured: {
  onTurnComplete?: (t: { user: string; assistant: string; responseId: string; itemId: string }) => void
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
      note: runtimeState.note,
      ensuring: runtimeState.ensuring,
      services: {
        sidecar: runtimeState.ready,
        llm: runtimeState.ready,
        breeze: runtimeState.ready,
        gateway: runtimeState.ready,
        ready: runtimeState.ready,
        error: '',
      },
      connect: vi.fn(async () => {}), disconnect: vi.fn(),
      startMic: vi.fn(async () => {}), stopMic: vi.fn(), cancel: vi.fn(),
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
vi.mock('../../gaea/lib/bridge', () => ({
  app: {
    VoiceV2CommitTurn: (
      topic: string, user: string, assistant: string, responseId: string, itemId: string,
    ) => commitMock(topic, user, assistant, responseId, itemId),
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

  it('sidecar 未就绪显示警示与拉起钮', () => {
    runtimeState.ready = false
    renderConsole()
    expect(screen.getByTestId('vc2-ensure').textContent).toContain('启动本地语音')
    expect(screen.getByTestId('vc2-warn').textContent).toContain('依赖待准备')
    expect(screen.getByRole('heading', { name: '准备本地语音' })).toBeTruthy()
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

  it('实时对话流同时呈现用户转写、临时转写和助手回答', () => {
    runtimeState.transcripts = ['帮我安排今天的任务']
    runtimeState.partialTranscript = '下午优先处理'
    runtimeState.assistantText = '我先从下午的事项开始安排。'
    renderConsole()
    expect(screen.getByText('帮我安排今天的任务')).toBeTruthy()
    expect(screen.getByText(/下午优先处理/)).toBeTruthy()
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

  it('迟到轮不串台：活动话题已切走时不写回', async () => {
    const { rerender } = renderConsole()
    rerender(
      <VoiceConsoleV2
        companionName="小音"
        instructions="测试指令"
        onClose={() => {}}
        onOpenSettings={() => {}}
        setMessages={() => {}}
        getActiveId={() => 'topic_OTHER'}
      />,
    )
    await act(async () => {
      captured.onTurnComplete?.({ user: '旧轮', assistant: '旧答', responseId: 'r2', itemId: 'item-2' })
    })
    expect(commitMock).not.toHaveBeenCalled()
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
})
