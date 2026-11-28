// Find.test.tsx — 阅读窗查找条的定位函数与交互：全文查词、按摘要片段定位（忽略空白与
// Markdown 记号、可跨文本节点）、跳进来后落到命中那一段。
import { describe, expect, it } from 'vitest'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { useRef, useState } from 'react'
import ReaderFind from './Find'
import { findNeedle, findTerm, locateSnippet } from './Find.logic'

type Piece = { node: Text; start: number; end: number }

function bodyOf(html: string): HTMLElement {
  const node = document.createElement('div')
  node.innerHTML = html
  return node
}

function textOfHit(hit: { pieces: Piece[] } | null): string {
  return (hit?.pieces ?? []).map((p) => p.node.data.slice(p.start, p.end)).join('|')
}

describe('findTerm：正文里的全部出现处', () => {
  it('大小写不敏感、按文档顺序', () => {
    const node = bodyOf('<p>执行 HJ 25.1 标准；HJ 是行业标准。再提一次 hj。</p>')
    expect(findTerm(node, 'hj')).toHaveLength(3)
    expect(findTerm(node, '标准')).toHaveLength(2)
    expect(findTerm(node, '没有的词')).toHaveLength(0)
    expect(findTerm(node, '   ')).toHaveLength(0)
    // 「监测」两个字被 <b> 拆在两个文本节点里：仍算一处，片段各落在自己的节点上
    expect(textOfHit(findTerm(bodyOf('<p>坑内<b>监</b>测点</p>'), '监测')[0])).toBe('监|测')
  })

  it('跳过 script / style 与已有高亮里的文字', () => {
    expect(findTerm(bodyOf('<p>监测</p><script>监测</script><mark>监测</mark>'), '监测')).toHaveLength(1)
  })
})

describe('findNeedle / locateSnippet：按一小段原文定位', () => {
  it('忽略空白与 Markdown 记号，能跨文本节点', () => {
    // 摘要取自 Markdown 原文：「**监测**点 | 布置」在阅读窗里是拆开的加粗与表格单元
    const node = bodyOf('<p><strong>监测</strong>点</p><table><tbody><tr><td>布置</td><td>及预警</td></tr></tbody></table>')
    expect(textOfHit(findNeedle(node, '**监测**点 | 布置'))).toBe('监测|点|布置')
    expect(findNeedle(node, '完全没有这一段文字')).toBeNull()
    expect(findNeedle(node, '。')).toBeNull() // 太短的针不算
  })

  it('locateSnippet 由长到短试几档；整段都对不上才算找不到', () => {
    const node = bodyOf('<p>边坡失稳的防控措施：分层开挖、及时支护。</p>')
    expect(textOfHit(locateSnippet(node, '【边坡失稳】的防控措施：分层开挖、及时支护。'))).toBe('边坡失稳的防控措施：分层开挖、及')
    expect(locateSnippet(node, '前十六个字都对不上的另一段话，但后面有支护。')).toBeNull()
  })
})

describe('ReaderFind 组件', () => {
  function Harness({ locate }: { locate: { needle: string; fallback: string } | null }) {
    const [open, setOpen] = useState(false)
    const [req, setReq] = useState(locate)
    const bodyRef = useRef<HTMLDivElement>(null)
    return (
      <div>
        <div ref={bodyRef}>
          <p>边坡失稳的防控措施：分层开挖、及时支护。边坡要监测。</p>
        </div>
        <ReaderFind bodyRef={bodyRef} version="v1" locate={req} onLocated={() => setReq(null)} open={open} onOpenChange={setOpen} />
      </div>
    )
  }

  it('带定位请求挂载：打开查找条、落在命中那一段', async () => {
    render(<Harness locate={{ needle: '边坡失稳的防控措施：分层', fallback: '边坡' }} />)
    const find = await screen.findByTestId('proposal-find')
    expect((within(find).getByRole('textbox') as HTMLInputElement).value).toBe('边坡')
    expect(find.textContent).toContain('第 1 处，共 2 处') // 「边坡」两处，落在第一处
  })

  it('找不到摘要片段时退回全文命中词', async () => {
    render(<Harness locate={{ needle: '库里没有这一段', fallback: '监测' }} />)
    const find = await screen.findByTestId('proposal-find')
    expect((within(find).getByRole('textbox') as HTMLInputElement).value).toBe('监测')
    expect(find.textContent).toContain('第 1 处，共 1 处')
  })

  it('改查找词即重数；Enter 下一处、Shift+Enter 上一处；Esc 关闭', async () => {
    const { container } = render(<Harness locate={{ needle: '支护', fallback: '' }} />)
    const find = await screen.findByTestId('proposal-find')
    const box = within(find).getByRole('textbox') as HTMLInputElement
    expect(find.textContent).toContain('第 1 处，共 1 处')
    fireEvent.change(box, { target: { value: '' } })
    expect(find.textContent).not.toContain('处')
    fireEvent.change(box, { target: { value: '边坡' } })
    expect(find.textContent).toContain('第 1 处，共 2 处')
    fireEvent.keyDown(box, { key: 'Enter' })
    expect(find.textContent).toContain('第 2 处，共 2 处')
    fireEvent.keyDown(box, { key: 'Enter', shiftKey: true })
    expect(find.textContent).toContain('第 1 处，共 2 处')
    fireEvent.keyDown(box, { key: 'Escape' })
    await waitFor(() => expect(container.querySelector('.pl-find')).toBeNull())
  })

  it('没有定位请求、也没打开时不渲染', () => {
    const { container } = render(<Harness locate={null} />)
    expect(container.querySelector('.pl-find')).toBeNull()
  })
})
