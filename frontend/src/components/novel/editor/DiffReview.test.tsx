// DiffReview（W3 diff 收口刀）：本地贪心 diff 已删，统一走 gaea/lib/diff.ts
// LCS。用例锁住贪心会漂移、LCS 不会的构造（前向 3 行窗口的边界用例）：
// · 公共行前插入 4 行 → 窗口失配，贪心把公共行同时标删+标增；
// · 删除首行 → 前向搜索不到旧行，贪心把保留行同时标删+标增。
import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import DiffReview from './DiffReview'

function renderReview(original: string, revised: string) {
  return render(
    <DiffReview original={original} revised={revised} onAccept={vi.fn()} onReject={vi.fn()} />,
  )
}

/** 取 diff 行（行容器带 white-space:pre-wrap 内联样式），去前缀空白后按序返回 */
function rowsOf(container: HTMLElement): string[] {
  return Array.from(container.querySelectorAll('div'))
    .filter((d) => (d as HTMLElement).style.whiteSpace === 'pre-wrap')
    .map((d) => (d.textContent ?? '').trim())
}

describe('DiffReview 统一 LCS（W3）', () => {
  it('公共行前插入 4 行：只标新增，公共行保持上下文（贪心会漂移成删+增）', () => {
    const { container } = renderReview('共同句', '新1\n新2\n新3\n新4\n共同句')
    expect(rowsOf(container)).toEqual(['+ 新1', '+ 新2', '+ 新3', '+ 新4', '共同句'])
    expect(screen.getByText('+4 新增')).toBeTruthy()
    expect(screen.getByText('-0 删除')).toBeTruthy()
    expect(screen.getByText('5 行')).toBeTruthy()
  })

  it('删除首行：保留行只出现一次且为上下文（贪心会同时标删+标增）', () => {
    const { container } = renderReview('第一章\n旧句子', '旧句子')
    expect(rowsOf(container)).toEqual(['- 第一章', '旧句子'])
    expect(screen.getByText('+0 新增')).toBeTruthy()
    expect(screen.getByText('-1 删除')).toBeTruthy()
  })

  it('空原文 → 新文：仅标新增，不产出幽灵删除行（空串=0 行，显示不劣化）', () => {
    const { container } = renderReview('', '新句')
    expect(rowsOf(container)).toEqual(['+ 新句'])
    expect(screen.getByText('+1 新增')).toBeTruthy()
  })

  it('完全相同：全部为上下文行，零增零删', () => {
    const { container } = renderReview('第一行\n第二行', '第一行\n第二行')
    expect(rowsOf(container)).toEqual(['第一行', '第二行'])
    expect(screen.getByText('+0 新增')).toBeTruthy()
    expect(screen.getByText('-0 删除')).toBeTruthy()
    expect(screen.getByText('2 行')).toBeTruthy()
  })
})
