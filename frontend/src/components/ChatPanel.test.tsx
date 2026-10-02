/**
 * ChatPanel.test.tsx — FE6-09：ChatPanel 模拟打字流缺取消守卫
 *
 * 口径与 canonical 实现 hooks/useChatStream.ts:216-230（T6-3.3）一致：
 * ①tick 内首行判取消 → 循环自停；②循环之后判取消 → 不再写消息/最终态；
 * 触发源：组件卸载、切话题（父层整体替换消息列表，组件不重挂载）、新回合开始复位。
 *
 * 反向证据（本仓规矩：没反向证据的修不算修，见审计报告「反向证据」节）：
 *  - 删 tick 首行守卫 或 删循环后 return → 「卸载」「切话题」两例必红；
 *  - 删新回合复位 → 「连续两回合」例必红。
 */
import { useEffect, useState } from 'react'
import type { Dispatch, SetStateAction } from 'react'
import { act, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ChatPanel from './ChatPanel'
import type { Message } from './ChatPanel'

/** rAF 手控队列：打字循环每帧自排队，flushFrames 推进到队列空 = 确定性「打到第几帧」 */
let frames: Array<(t: number) => void> = []

beforeEach(() => {
  frames = []
  vi.stubGlobal('requestAnimationFrame', (cb: (t: number) => void) => {
    frames.push(cb)
    return frames.length
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

/** 推进 limit 帧（每帧一个 act），返回实际推进帧数；打字循环未结束会一直排队 */
async function flushFrames(limit = 200): Promise<number> {
  let n = 0
  while (n < limit) {
    const due = frames.splice(0)
    if (due.length === 0) return n
    await act(async () => { for (const cb of due) cb(n) })
    n++
  }
  return n
}

interface Control {
  setMessages: Dispatch<SetStateAction<Message[]>> | null
}

/** 宿主：模拟 NovelSettingPage 的父层（持有 messages、可整体替换 = 切话题） */
function Host({ onSend, control, calls, streaming }: {
  onSend: (m: string) => Promise<string | void>
  control: Control
  calls: Message[][]
  streaming?: boolean
}) {
  const [messages, setMessages] = useState<Message[]>([])
  useEffect(() => { control.setMessages = setMessages }, [control])
  return (
    <>
      {/* 探针：外部真实持有的消息列表（不依赖 antd 文本节点，避免双节点坑） */}
      <div data-testid="msgs">
        {messages.map((m) => `${m.role}:${m.content}:${m.streaming ? 'S' : 'N'}`).join('|')}
      </div>
      <ChatPanel
        title="设定 Agent"
        messages={messages}
        onSend={onSend}
        streaming={streaming}
        onMessagesChange={(m) => { calls.push(m); setMessages(m) }}
      />
    </>
  )
}

async function send(text: string): Promise<void> {
  const ta = screen.getByPlaceholderText(/输入消息/) as HTMLTextAreaElement
  fireEvent.change(ta, { target: { value: text } })
  await act(async () => { fireEvent.keyDown(ta, { key: 'Enter', shiftKey: false }) })
}

const probe = (): string => screen.getByTestId('msgs').textContent ?? ''

/** 流式气泡文本（组件内局部 streamText）：取最后一个 .cursor-blink 的父元素
 *  （消息列表里的 streaming 占位也带光标，流式气泡渲染在其后；切话题后列表为空则唯一） */
function bubbleText(container: HTMLElement): string {
  const cursors = container.querySelectorAll('.cursor-blink')
  const last = cursors[cursors.length - 1]
  return last?.parentElement?.textContent ?? ''
}

const REPLY_A = '第一回合回复正文甲乙丙丁戊己庚辛' // 15 字 → 5 帧（每帧 +3 字）
const REPLY_B = '第二回合回复正文子丑寅卯辰巳午未'

describe('FE6-09 模拟打字流取消守卫', () => {
  it('新回合复位取消标志：连续两回合都能完整走完打字流', async () => {
    const calls: Message[][] = []
    const control: Control = { setMessages: null }
    const onSend = vi.fn(async () => REPLY_A)
    onSend.mockResolvedValueOnce(REPLY_A).mockResolvedValueOnce(REPLY_B)
    render(<Host onSend={onSend} control={control} calls={calls} />)

    await send('问题一')
    await flushFrames()
    expect(probe()).toContain(`assistant:${REPLY_A}:N`)

    await send('问题二')
    await flushFrames()
    expect(probe()).toContain(`assistant:${REPLY_B}:N`)
    expect(onSend).toHaveBeenCalledTimes(2)
  })

  it('卸载后循环不再写状态：不再 onMessagesChange（旧写法会写回旧回合快照）', async () => {
    const calls: Message[][] = []
    const control: Control = { setMessages: null }
    const onSend = vi.fn(async () => REPLY_A)
    const { unmount } = render(<Host onSend={onSend} control={control} calls={calls} />)

    await send('问题一')
    await flushFrames(2) // 打字进行中
    const duringLoop = calls.length // 用户消息 + AI 占位两条
    expect(duringLoop).toBe(2)

    unmount()
    await flushFrames() // 卸载后推进剩余全部帧
    expect(calls).toHaveLength(duringLoop) // 旧写法：+1（最终态覆盖回外部状态）
  })

  it('切话题中止打字流：不再推进 streamText、不再把旧回合快照覆盖回新会话', async () => {
    const calls: Message[][] = []
    const control: Control = { setMessages: null }
    const onSend = vi.fn(async () => REPLY_A)
    const { container } = render(<Host onSend={onSend} control={control} calls={calls} streaming />)

    await send('问题一')
    await flushFrames(2)
    const frozen = bubbleText(container)
    expect(frozen.length).toBeGreaterThan(0)

    // 切话题：父层整体替换消息列表（同 NovelSettingPage.tsx:117 setMessages([])）
    await act(async () => { control.setMessages!([]) })
    await flushFrames()

    expect(probe()).toBe('') // 旧写法被 [...withAi] 覆盖回旧对话
    expect(bubbleText(container)).toBe(frozen) // 旧写法 tick 继续推进 streamText
  })

  it('父层裁剪/分页只留本回合消息（仍含本会话 id）不误判切话题：打字走完', async () => {
    const calls: Message[][] = []
    const control: Control = { setMessages: null }
    const onSend = vi.fn(async () => REPLY_A)
    render(<Host onSend={onSend} control={control} calls={calls} />)

    await send('问题一')
    await flushFrames(2)
    // 边界：父层把列表裁到只剩最新一条（本回合的 AI 占位，id 仍属本会话）——
    // 判据是「新旧列表是否还含本会话写入过的任一消息 id」，故这里不得中止打字。
    // 口径理由：在途回合自己的消息永远是最新的，任何合法的裁剪/分页都至少留下其中一条；
    // 只有整表换掉（如 NovelSettingPage 切书 setMessages([])）才是换话题。
    await act(async () => { control.setMessages!((prev) => prev.slice(-1)) })
    await flushFrames()

    expect(probe()).toContain(`assistant:${REPLY_A}:N`) // 未被误判中止：最终态落位
  })

  it('切话题取消后，新回合仍能完整走完打字流', async () => {
    const calls: Message[][] = []
    const control: Control = { setMessages: null }
    const onSend = vi.fn(async () => REPLY_A)
    onSend.mockResolvedValueOnce(REPLY_A).mockResolvedValueOnce(REPLY_B)
    render(<Host onSend={onSend} control={control} calls={calls} />)

    await send('问题一')
    await flushFrames(2)
    await act(async () => { control.setMessages!([]) })
    await flushFrames()

    await send('问题二')
    await flushFrames()
    expect(probe()).toContain(`assistant:${REPLY_B}:N`)
  })

  it('回合取消后 onSend 失败不写错误消息（旧写法会把错误写进新会话）', async () => {
    const calls: Message[][] = []
    const control: Control = { setMessages: null }
    let failReply: (e: unknown) => void = () => {}
    const onSend = vi.fn(() => new Promise<string>((_res, rej) => { failReply = rej }))
    render(<Host onSend={onSend} control={control} calls={calls} />)

    await send('问题一')
    await act(async () => { control.setMessages!([]) }) // 切话题（onSend 仍未决）
    await act(async () => { failReply(new Error('boom')) })
    await flushFrames()

    expect(probe()).toBe('')
    expect(calls).toHaveLength(2)
  })
})
