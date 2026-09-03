// readingHighlight.test.ts — 阅读页「朗读/搜索定位高亮」纯 DOM 工具测试。
// 用 jsdom 造 .novel-reading-p 段落结构，锁 textNodesOf 收集行为、
// highlightSearchHitAt 命中/未命中返回值与高亮 DOM 副作用、clearReadingHighlight 清理。
// scrollIntoView 由 src/test/setup.ts 的全局 polyfill 兜底（jsdom 未实现）。
import { afterEach, describe, expect, it } from 'vitest'
import { clearReadingHighlight, highlightSearchHitAt, textNodesOf } from './readingHighlight'

/** 造一个滚动根，每段包成 .novel-reading-p（模拟阅读列 DOM），返回根与段落列表 */
function buildRoot(...paras: string[]): { root: HTMLElement; els: HTMLElement[] } {
  const root = document.createElement('div')
  root.innerHTML = paras.map((t) => `<p class="novel-reading-p">${t}</p>`).join('')
  document.body.appendChild(root)
  const els = Array.from(root.querySelectorAll<HTMLElement>('.novel-reading-p'))
  return { root, els }
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('textNodesOf', () => {
  it('单段落收集为单个文本节点，偏移覆盖全文本', () => {
    const { els } = buildRoot('夜色沉沉，雨落在窗台上。')
    const nodes = textNodesOf(els[0])
    expect(nodes.length).toBe(1)
    expect(nodes[0].node.textContent).toBe('夜色沉沉，雨落在窗台上。')
    expect(nodes[0].start).toBe(0)
    expect(nodes[0].end).toBe('夜色沉沉，雨落在窗台上。'.length)
  })

  it('跨内联元素（<em>）按文本节点切分，偏移首尾相接且总长与 textContent 一致', () => {
    const { els } = buildRoot('他推门而入，<em>灯</em>还亮着。')
    const nodes = textNodesOf(els[0])
    expect(nodes.map((t) => t.node.textContent)).toEqual(['他推门而入，', '灯', '还亮着。'])
    expect(nodes[0].end).toBe(nodes[1].start)
    expect(nodes[1].end).toBe(nodes[2].start)
    expect(nodes[nodes.length - 1].end).toBe((els[0].textContent || '').length)
  })

  it('空文本节点被跳过且不推进偏移', () => {
    const { els } = buildRoot('abc')
    els[0].insertBefore(document.createTextNode(''), els[0].firstChild!)
    const nodes = textNodesOf(els[0])
    expect(nodes.length).toBe(1)
    expect(nodes[0].start).toBe(0)
    expect(nodes[0].end).toBe(3)
  })
})

describe('highlightSearchHitAt', () => {
  it('命中：返回 true，命中词包进 .novel-reading-search-hit 且段落全文不变', () => {
    const { root, els } = buildRoot('夜色沉沉，雨落在窗台上。')
    const ok = highlightSearchHitAt(root, els, 0, '雨落在窗台', 0)
    expect(ok).toBe(true)
    const span = root.querySelector('span.novel-reading-search-hit')
    expect(span).not.toBeNull()
    expect(span!.textContent).toBe('雨落在窗台')
    expect(els[0].textContent).toBe('夜色沉沉，雨落在窗台上。')
  })

  it('查询词不在段落内：返回 false 且不产生高亮 DOM', () => {
    const { root, els } = buildRoot('夜色沉沉。')
    expect(highlightSearchHitAt(root, els, 0, '不存在的词', 0)).toBe(false)
    expect(root.querySelector('span.novel-reading-search-hit')).toBeNull()
  })

  it('空查询与段落索引越界：均返回 false 且不动 DOM', () => {
    const { root, els } = buildRoot('第一段。', '第二段。')
    expect(highlightSearchHitAt(root, els, 0, '', 0)).toBe(false)
    expect(highlightSearchHitAt(root, els, 5, '第一段', 0)).toBe(false)
    expect(root.querySelector('span')).toBeNull()
  })

  it('两段都含查询词时只高亮指定段落那一处', () => {
    const { root, els } = buildRoot('灯火将熄。', '他望着灯火出神。')
    expect(highlightSearchHitAt(root, els, 1, '灯火', 0)).toBe(true)
    const spans = root.querySelectorAll('span.novel-reading-search-hit')
    expect(spans.length).toBe(1)
    expect(els[0].contains(spans[0])).toBe(false)
    expect(els[1].contains(spans[0])).toBe(true)
    expect(spans[0].textContent).toBe('灯火')
  })

  it('命中跨内联元素边界时仍能整体包进一个 span', () => {
    const { root, els } = buildRoot('他推门而入，<em>灯</em>还亮着。')
    expect(highlightSearchHitAt(root, els, 0, '，灯还亮', 0)).toBe(true)
    const span = root.querySelector('span.novel-reading-search-hit')!
    expect(span.textContent).toBe('，灯还亮')
  })

  it('重复定位：先清旧高亮再包新 span，全 root 只剩一处且文本无损', () => {
    const { root, els } = buildRoot('夜色沉沉，雨落在窗台上。')
    expect(highlightSearchHitAt(root, els, 0, '雨落在窗台', 0)).toBe(true)
    expect(highlightSearchHitAt(root, els, 0, '夜色沉沉', 0)).toBe(true)
    const spans = root.querySelectorAll('span.novel-reading-search-hit')
    expect(spans.length).toBe(1)
    expect(spans[0].textContent).toBe('夜色沉沉')
    expect(els[0].textContent).toBe('夜色沉沉，雨落在窗台上。')
  })
})

describe('clearReadingHighlight', () => {
  it('解开指定 class 的高亮容器并 normalize 合并回单个文本节点', () => {
    const { root, els } = buildRoot('')
    els[0].innerHTML = '夜色沉沉，<span class="novel-reading-search-hit">雨落在窗台</span>上。'
    clearReadingHighlight(root, 'novel-reading-search-hit')
    expect(root.querySelector('.novel-reading-search-hit')).toBeNull()
    expect(els[0].textContent).toBe('夜色沉沉，雨落在窗台上。')
    expect(els[0].childNodes.length).toBe(1)
  })

  it('只清理指定 class，不误伤其他高亮（如朗读 current）', () => {
    const { root, els } = buildRoot('')
    els[0].innerHTML = '前<span class="novel-reading-current">朗读</span>中<span class="novel-reading-search-hit">搜索</span>后'
    clearReadingHighlight(root, 'novel-reading-search-hit')
    expect(root.querySelector('.novel-reading-search-hit')).toBeNull()
    expect(root.querySelector('.novel-reading-current')).not.toBeNull()
    expect(els[0].textContent).toBe('前朗读中搜索后')
  })

  it('root 为 null 时安全返回（滚动根未挂载场景）', () => {
    expect(() => clearReadingHighlight(null, 'novel-reading-search-hit')).not.toThrow()
  })
})
