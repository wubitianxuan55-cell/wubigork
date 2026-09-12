// SinProcessCard.test.tsx — 原罪过程卡：无内容不渲染 / 历史还原折叠 / 运行态 /
// 失败态 / 工具行摘要纯函数。
import { describe, expect, it } from 'vitest'
import { fireEvent, render } from '@testing-library/react'
import { SinProcessCard } from './SinProcessCard'
import { sinToolLabel, sinToolSubject, sinToolSummary } from './sinToolMeta'
import type { SinToolTraceView } from './types'

/** 卡片头 / 工具行按钮（用类名取，避免与头部摘要文案撞名导致选择器歧义）。 */
function cardHead(c: HTMLElement): HTMLElement {
  return c.querySelector('.sin-proc-head') as HTMLElement
}

function toolHeads(c: HTMLElement): HTMLElement[] {
  return Array.from(c.querySelectorAll('.sin-proc-tool-head')) as HTMLElement[]
}

function trace(over: Partial<SinToolTraceView> = {}): SinToolTraceView {
  return {
    id: 'call_1',
    name: 'web_search',
    args: '{"query":"唐末长安坊市"}',
    output: '搜到 3 条结果',
    error: '',
    elapsed_ms: 1240,
    read_only: true,
    status: 'done',
    ...over,
  }
}

describe('SinProcessCard', () => {
  it('既无思考也无工具时不渲染（纯正文回复不该多出一条空行）', () => {
    const { container } = render(<SinProcessCard tools={[]} running={false} />)
    expect(container.firstChild).toBeNull()
    const blank = render(<SinProcessCard reasoning="   " tools={[]} running={false} />)
    expect(blank.container.firstChild).toBeNull()
  })

  it('历史还原：默认折叠，头部给思考字数与工具摘要，展开后逐行显示', () => {
    const { container } = render(
      <SinProcessCard
        reasoning="先查资料"
        tools={[trace(), trace({ id: 'call_2', name: 'sin_notes', args: '{"action":"write","content":"女主叫林晚"}' })]}
        running={false}
      />,
    )
    expect(container.textContent).toContain('思考过程 · 4 字')
    expect(container.textContent).toContain('联网搜索 · 故事便签')
    // 折叠态：工具行整体不渲染
    expect(toolHeads(container)).toHaveLength(0)

    fireEvent.click(cardHead(container))
    const rows = toolHeads(container)
    expect(rows).toHaveLength(2)
    expect(rows[0].textContent).toContain('唐末长安坊市')
    expect(rows[0].textContent).toContain('1.2s')
    expect(rows[1].textContent).toContain('写入 · 女主叫林晚')
  })

  it('运行中：头部不定态、工具行标运行中（运行态自动展开）', () => {
    const { container } = render(
      <SinProcessCard
        reasoning=""
        tools={[trace({ status: 'running', elapsed_ms: 0, output: '' })]}
        running
      />,
    )
    const rows = toolHeads(container)
    expect(rows).toHaveLength(1)
    expect(rows[0].textContent).toContain('联网搜索')
    expect(rows[0].textContent).toContain('运行中')
    expect(container.querySelector('.sin-proc')?.getAttribute('data-running')).toBe('')
  })

  it('失败：行标失败并给错误文本（展开后可见），不吞错误', () => {
    const { container } = render(
      <SinProcessCard
        tools={[trace({ name: 'web_fetch', error: '抓取失败：域名被策略拒绝', status: 'failed', output: '' })]}
        running={false}
      />,
    )
    fireEvent.click(cardHead(container)) // 展开卡片
    expect(container.textContent).toContain('失败')
    fireEvent.click(toolHeads(container)[0]) // 展开该行明细
    expect(container.textContent).toContain('域名被策略拒绝')
  })

  it('展开工具行：参数美化打印、输出超长留可见折叠标记', () => {
    const long = Array.from({ length: 90 }, (_, i) => `第 ${i} 行`).join('\n')
    const { container } = render(<SinProcessCard tools={[trace({ output: long })]} running={false} />)
    fireEvent.click(cardHead(container))
    fireEvent.click(toolHeads(container)[0])
    expect(container.textContent).toContain('"query": "唐末长安坊市"')
    expect(container.textContent).toMatch(/已折叠 \d+ 行/)
  })
})

describe('过程卡纯函数', () => {
  it('工具标签：已知工具给中文，未知工具如实显示原名', () => {
    expect(sinToolLabel('web_search')).toBe('联网搜索')
    expect(sinToolLabel('sin_outline')).toBe('故事大纲')
    expect(sinToolLabel('sin_export')).toBe('图文导出')
    expect(sinToolLabel('mystery_tool')).toBe('mystery_tool')
  })

  it('行首摘要：联网复用办公 subjectOf，便签/大纲给动作 + 内容，角色卡给名字', () => {
    expect(sinToolSubject(trace())).toBe('唐末长安坊市')
    expect(sinToolSubject(trace({ name: 'web_fetch', args: '{"url":"https://example.com/a"}' })))
      .toBe('https://example.com/a')
    expect(sinToolSubject(trace({ name: 'sin_cast', args: '{"name":"林晚"}' }))).toBe('林晚')
    expect(sinToolSubject(trace({ name: 'sin_cast', args: '{}' }))).toBe('全部角色')
    expect(sinToolSubject(trace({ name: 'sin_notes', args: '{"action":"write","content":"女主叫林晚"}' })))
      .toBe('写入 · 女主叫林晚')
    expect(sinToolSubject(trace({ name: 'sin_outline', args: '{"action":"read"}' }))).toBe('读取')
    expect(sinToolSubject(trace({ name: 'weird_tool', args: 'not json' }))).toBe('')
  })

  it('头部摘要按工具分组计数，超过 3 组收敛为「等」', () => {
    expect(sinToolSummary([])).toBe('')
    expect(sinToolSummary([
      trace(), trace({ id: 'b' }), trace({ id: 'c', name: 'sin_notes' }),
    ])).toBe('联网搜索 ×2 · 故事便签')
    expect(sinToolSummary([
      trace(), trace({ id: 'b', name: 'web_fetch' }), trace({ id: 'c', name: 'sin_notes' }),
      trace({ id: 'd', name: 'sin_cast' }),
    ])).toBe('联网搜索 · 抓取网页 · 故事便签 · 等')
  })
})
