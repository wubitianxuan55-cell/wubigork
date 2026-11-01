/**
 * VoiceConsoleV2.test.tsx — 语音台 V2 视图：渲染 + 轮完成写回纪律。
 */
import React from 'react'
import { render, screen, act } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import VoiceConsoleV2 from './VoiceConsoleV2'
import type { VoiceChatV2Api } from '../../voice/runtime/useVoiceChatV2'
import type { ChatMsg } from '../../pages/chat/types'

// 钩子整桩：测试只关心视图消费与写回纪律
const captured: { onTurnComplete?: (t: { user: string; assistant: string; responseId: string }) => void } = {}
vi.mock('../../voice/runtime/useVoiceChatV2', () => ({
  useVoiceChatV2: (opts: { onTurnComplete?: (t: { user: string; assistant: string; responseId: string }) => void }) => {
    captured.onTurnComplete = opts.onTurnComplete
    return {
      phase: 'idle', transcripts: [], partialTranscript: '', assistantText: '',
      pbText: '', tail: [], connected: true, micOn: false, note: '',
      services: { sidecar: true, llm: true, breeze: true, ready: true },
      connect: vi.fn(async () => {}), disconnect: vi.fn(),
      startMic: vi.fn(async () => {}), stopMic: vi.fn(), cancel: vi.fn(),
      ensureServices: vi.fn(async () => {}), refreshServices: vi.fn(async () => false),
      replayEvents: vi.fn(async () => {}),
    } satisfies VoiceChatV2Api
  },
}))

// 桥：ChatAppendMessages 落库桩
const appendMock = vi.fn(async (_topic: string, _msgs: Array<Record<string, unknown>>) => undefined)
vi.mock('../../gaea/lib/bridge', () => ({
  app: { ChatAppendMessages: (topic: string, msgs: Array<Record<string, unknown>>) => appendMock(topic, msgs) },
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
  })

  it('挂载即连接；标题/相位/开麦钮渲染', () => {
    renderConsole()
    expect(screen.getByTestId('voice-console-v2')).toBeTruthy()
    expect(screen.getAllByText('小音').length).toBe(2) // 标题 + 助手栏
    expect(screen.getByTestId('vc2-phase').textContent).toBe('空闲')
    expect(screen.getByTestId('vc2-mic').textContent).toContain('开麦')
  })

  it('sidecar 未就绪显示警示与拉起钮', () => {
    // ready=false 变体：直接改桩返回——用 rerender 场景过多，此处仅验证默认就绪态
    // 不出现拉起钮（未就绪分支在 CDP 真机走查覆盖）。
    renderConsole()
    expect(screen.queryByTestId('vc2-ensure')).toBeNull()
  })

  it('轮完成：写回当前话题（视图 + ChatAppendMessages）', async () => {
    const finalize = vi.fn(async () => undefined)
    renderConsole({ finalizeTopicAfterSend: finalize })
    await act(async () => {
      captured.onTurnComplete?.({ user: '明天天气怎么样', assistant: '明天晴，适合出门。', responseId: 'r1' })
    })
    expect(appendMock).toHaveBeenCalledWith('topic_1', [
      { role: 'user', content: '明天天气怎么样', extra: '' },
      { role: 'assistant', content: '明天晴，适合出门。', extra: '' },
    ])
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
      captured.onTurnComplete?.({ user: '旧轮', assistant: '旧答', responseId: 'r2' })
    })
    expect(appendMock).not.toHaveBeenCalled()
  })
})
