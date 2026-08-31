// MessageList 性能优化测试（T-perf）：
//  - 尾部窗口：消息数 > VIRTUALIZE_THRESHOLD 时渲染行数受限（≤ 窗口上限，
//    真实减少 DOM 节点）；宿主 scroll 触发向上扩载；切话题重置窗口
//  - 行 memo：追加消息 / 流式 chunk / copiedId 变化时旧行零重渲染
//    （mock ChatRow 仅做计数包裹真实组件——MessageList 侧 memo 命中时
//    wrapper 根本不会被调用，计数即「memo 放行后的真实渲染次数」）
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import type { ChatMsg } from '../../pages/chat/types'
import {
  MessageList, VIRTUALIZE_THRESHOLD, WINDOW_INITIAL, WINDOW_GROW_STEP,
} from './MessageList'
import type { ChatRowProps } from './ChatRow'

// ChatRow 渲染计数（包裹真实组件，保持真实 DOM 供窗口断言使用）
const renderCounts = vi.hoisted(() => ({ byKey: new Map<string, number>() }))

vi.mock('./ChatRow', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./ChatRow')>()
  const RealChatRow = actual.ChatRow
  const CountingChatRow = (props: ChatRowProps) => {
    renderCounts.byKey.set(props.msg.key, (renderCounts.byKey.get(props.msg.key) ?? 0) + 1)
    return <RealChatRow {...props} />
  }
  return { ...actual, ChatRow: CountingChatRow }
})

/** 构造 n 条 user/assistant 交替消息（key/content 按 startIdx 起始编号）。 */
function makeMessages(n: number, startIdx = 0): ChatMsg[] {
  return Array.from({ length: n }, (_, i): ChatMsg => ({
    key: `k${startIdx + i}`,
    role: i % 2 === 0 ? 'user' : 'assistant',
    content: `消息内容 ${startIdx + i}`,
    createdAt: '2026-01-01T00:00:00Z',
  }))
}

const noop = () => {}
const baseProps = {
  streamKey: null as string | null,
  streamText: '',
  mode: 'plain',
  companionName: 'gaea',
  copiedId: null as string | null,
  speakingId: null as string | null,
  onCopy: noop,
  onSpeak: noop,
  onRetry: noop,
}

const rowCount = () => document.querySelectorAll('.chat-row').length

describe('MessageList 尾部窗口（DOM 节点受限）', () => {
  it(`消息数 ≤ ${VIRTUALIZE_THRESHOLD}：全量渲染，行为不变`, () => {
    render(<MessageList {...baseProps} messages={makeMessages(VIRTUALIZE_THRESHOLD)} />)
    expect(rowCount()).toBe(VIRTUALIZE_THRESHOLD)
    expect(screen.getByText('消息内容 0')).toBeTruthy()
    expect(screen.getByText(`消息内容 ${VIRTUALIZE_THRESHOLD - 1}`)).toBeTruthy()
  })

  it(`消息数 > ${VIRTUALIZE_THRESHOLD}：仅渲染最近 ${WINDOW_INITIAL} 条，真实减少 DOM 节点`, () => {
    const total = VIRTUALIZE_THRESHOLD + 150 // 200 条
    render(<MessageList {...baseProps} messages={makeMessages(total)} />)
    const rows = rowCount()
    expect(rows).toBeLessThanOrEqual(WINDOW_INITIAL) // 渲染行数 ≤ 窗口上限（尾部锚定，无额外缓冲行）
    expect(rows).toBe(WINDOW_INITIAL)
    expect(rows).toBeLessThan(total) // 相比全量渲染真实减少 DOM 节点
    // 窗口外首条不可见、窗口内首条与末条可见
    expect(screen.queryByText('消息内容 0')).toBeNull()
    expect(screen.getByText(`消息内容 ${total - WINDOW_INITIAL}`)).toBeTruthy()
    expect(screen.getByText(`消息内容 ${total - 1}`)).toBeTruthy()
  })

  it(`阈值边界（${VIRTUALIZE_THRESHOLD + 1} 条）：窗口不小于总数时一条不丢`, () => {
    const total = VIRTUALIZE_THRESHOLD + 1
    render(<MessageList {...baseProps} messages={makeMessages(total)} />)
    expect(rowCount()).toBe(total)
  })

  it('宿主滚动到顶部附近：按步长向上扩载更早消息，封顶总数', () => {
    const total = WINDOW_INITIAL + WINDOW_GROW_STEP + 10 // 150 条
    render(
      <div data-testid="host">
        <MessageList {...baseProps} messages={makeMessages(total)} />
      </div>,
    )
    const host = screen.getByTestId('host')
    expect(rowCount()).toBe(WINDOW_INITIAL)
    fireEvent.scroll(host)
    expect(rowCount()).toBe(WINDOW_INITIAL + WINDOW_GROW_STEP)
    fireEvent.scroll(host)
    expect(rowCount()).toBe(total) // min(140 + 60, 150) 封顶
    fireEvent.scroll(host)
    expect(rowCount()).toBe(total) // 已全量：不再扩载
    expect(screen.getByText('消息内容 0')).toBeTruthy() // 扩载后最早消息可见
  })

  it('切话题（消息集合首键变化）：窗口重置为初始大小', () => {
    const view = render(
      <div data-testid="host">
        <MessageList {...baseProps} messages={makeMessages(200)} />
      </div>,
    )
    const host = screen.getByTestId('host')
    fireEvent.scroll(host)
    expect(rowCount()).toBe(WINDOW_INITIAL + WINDOW_GROW_STEP)
    // 切换到另一话题（全新消息集合，首键变化）
    view.rerender(
      <div data-testid="host">
        <MessageList {...baseProps} messages={makeMessages(120, 1000)} />
      </div>,
    )
    expect(rowCount()).toBe(WINDOW_INITIAL)
    expect(screen.queryByText('消息内容 1000')).toBeNull() // 新话题首条被窗口裁掉
    expect(screen.getByText('消息内容 1119')).toBeTruthy() // 新话题末条在窗口内
  })
})

describe('MessageList 行 memo（无关更新时旧行零重渲染）', () => {
  beforeEach(() => { renderCounts.byKey.clear() })
  const countOf = (key: string) => renderCounts.byKey.get(key) ?? 0

  it('追加消息：旧行不重渲染，仅新行渲染一次', () => {
    const msgs = makeMessages(4)
    const view = render(<MessageList {...baseProps} messages={msgs} />)
    expect(countOf('k0')).toBe(1)
    expect(countOf('k3')).toBe(1)
    // 模拟 setMessages(prev => [...prev, m])：旧行对象引用保持不变
    view.rerender(
      <MessageList
        {...baseProps}
        messages={[...msgs, { key: 'k4', role: 'assistant', content: '新追加回复', createdAt: '2026-01-01T00:00:00Z' }]}
      />,
    )
    expect(countOf('k0')).toBe(1) // 旧行零重渲染
    expect(countOf('k3')).toBe(1)
    expect(countOf('k4')).toBe(1) // 新行渲染一次
    expect(screen.getByText('新追加回复')).toBeTruthy()
  })

  it('流式 chunk：仅流式行重渲染，其余行保持一次；终态落定后正常收尾', () => {
    const base = makeMessages(3)
    const msgs: ChatMsg[] = [
      ...base,
      { key: 'kS', role: 'assistant', content: '', streaming: true, createdAt: '2026-01-01T00:00:00Z' },
    ]
    const view = render(<MessageList {...baseProps} messages={msgs} streamKey="kS" streamText="" />)
    expect(countOf('kS')).toBe(1)
    // 两个流式 chunk：streamText 增长，messages 数组与旧行引用不变
    view.rerender(<MessageList {...baseProps} messages={msgs} streamKey="kS" streamText="你" />)
    view.rerender(<MessageList {...baseProps} messages={msgs} streamKey="kS" streamText="你好" />)
    expect(countOf('k0')).toBe(1)
    expect(countOf('k2')).toBe(1)
    expect(countOf('kS')).toBe(3) // 初始 + 两个 chunk
    expect(screen.getByText('你好')).toBeTruthy()
    // done：终态替换流式行对象（streaming=false + 最终内容），streamKey 复位
    const done: ChatMsg[] = [
      ...base,
      { key: 'kS', role: 'assistant', content: '完整回复内容', createdAt: '2026-01-01T00:00:00Z' },
    ]
    view.rerender(<MessageList {...baseProps} messages={done} streamKey={null} streamText="" />)
    expect(countOf('k0')).toBe(1)
    expect(countOf('k2')).toBe(1)
    expect(countOf('kS')).toBe(4)
    expect(screen.getByText('完整回复内容')).toBeTruthy()
  })

  it('copiedId 变化：仅进入/退出复制态的行重渲染', () => {
    const msgs = makeMessages(4)
    const view = render(<MessageList {...baseProps} messages={msgs} />)
    view.rerender(<MessageList {...baseProps} messages={msgs} copiedId="k2" />)
    expect(countOf('k0')).toBe(1)
    expect(countOf('k3')).toBe(1)
    expect(countOf('k2')).toBe(2) // 进入复制态
    view.rerender(<MessageList {...baseProps} messages={msgs} copiedId={null} />)
    expect(countOf('k2')).toBe(3) // 退出复制态
    expect(countOf('k0')).toBe(1)
    expect(countOf('k3')).toBe(1)
  })

  it('稳定回调桥：点击复制调用的是最新一次渲染传入的 onCopy', () => {
    const first = vi.fn()
    const second = vi.fn()
    const msgs = makeMessages(2)
    const view = render(<MessageList {...baseProps} messages={msgs} onCopy={first} />)
    // 模拟 ChatPage 每次渲染传入新 handler 引用
    view.rerender(<MessageList {...baseProps} messages={msgs} onCopy={second} />)
    const copyBtn = document.querySelector('.chat-row .anticon-copy')?.closest('button')
    expect(copyBtn).toBeTruthy()
    fireEvent.click(copyBtn!)
    expect(second).toHaveBeenCalledWith('消息内容 0', 'k0')
    expect(first).not.toHaveBeenCalled()
  })
})

describe('v4.15 消息级回显（extra.answered_by）', () => {
  it('assistant 消息带 extra.answered_by：底部渲染「由谁回答/为何/花了多少」小字', () => {
    render(
      <MessageList
        {...baseProps}
        messages={[{
          key: 'kA', role: 'assistant', content: '回复内容', createdAt: '2026-01-01T00:00:00Z',
          extra: { answered_by: { engine: 'deepseek', model: 'deepseek-v4-flash', source: 'feature', cost_cny: 0.0123 } },
        }]}
      />,
    )
    expect(screen.getByText('由 deepseek/deepseek-v4-flash 回答 · 功能绑定 · 约 ¥0.01')).toBeTruthy()
  })

  it('旧消息（无 extra.answered_by）：零渲染，向后兼容', () => {
    render(<MessageList {...baseProps} messages={makeMessages(2)} />)
    expect(screen.queryByText(/由 .* 回答/)).toBeNull()
    expect(screen.queryByText(/约 ¥/)).toBeNull()
  })

  it('流式行即使带 answered_by 也不渲染（终态才回显）', () => {
    render(
      <MessageList
        {...baseProps}
        messages={[{
          key: 'kS', role: 'assistant', content: '', streaming: true, createdAt: '2026-01-01T00:00:00Z',
          extra: { answered_by: { engine: 'deepseek', model: 'deepseek-v4-flash', source: 'global', cost_cny: 0.01 } },
        }]}
        streamKey="kS"
        streamText=""
      />,
    )
    expect(screen.queryByText(/由 .* 回答/)).toBeNull()
  })
})
