// readingHighlight.test.ts — 阅读页「朗读/搜索定位高亮 + 划线」纯 DOM 工具测试。
// 用 jsdom 造 .novel-reading-p 段落结构，锁 textNodesOf 收集行为、
// highlightSearchHitAt 命中/未命中返回值与高亮 DOM 副作用、clearReadingHighlight 清理，
// 以及自 ChapterPage 第二批搬入的 applyTextHighlight（全文首个命中）、paraOf（段落归属）、
// textAtScrollTop（书签摘录，offsetTop 用 defineProperty 打桩——jsdom 恒为 0）、
// 第三批搬入的 readSelectionInRoot（划词选区校验，getSelection 打桩）。
// scrollIntoView 由 src/test/setup.ts 的全局 polyfill 兜底（jsdom 未实现）。
import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  applyTextHighlight, clearReadingHighlight, highlightSearchHitAt, paraOf,
  readSelectionInRoot, textAtScrollTop, textNodesOf,
} from './readingHighlight'

/** 造一个滚动根，每段包成 .novel-reading-p（模拟阅读列 DOM），返回根与段落列表 */
function buildRoot(...paras: string[]): { root: HTMLElement; els: HTMLElement[] } {
  const root = document.createElement('div')
  root.innerHTML = paras.map((t) => `<p class="novel-reading-p">${t}</p>`).join('')
  document.body.appendChild(root)
  const els = Array.from(root.querySelectorAll<HTMLElement>('.novel-reading-p'))
  return { root, els }
}

/** jsdom 未实现 offsetTop（恒 0），用实例属性打桩模拟各段真实纵向位置 */
function setOffsetTop(el: HTMLElement, value: number) {
  Object.defineProperty(el, 'offsetTop', { value, configurable: true })
}

afterEach(() => {
  document.body.innerHTML = ''
  vi.restoreAllMocks()
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

describe('applyTextHighlight', () => {
  it('命中：返回 true，首个含目标文本的段落被包进指定 class 的 span', () => {
    const { root, els } = buildRoot('夜色沉沉，雨落在窗台上。', '他推门而入，灯还亮着。')
    expect(applyTextHighlight(root, true, '夜色沉沉', 'novel-reading-current')).toBe(true)
    const span = root.querySelector('span.novel-reading-current')
    expect(span).not.toBeNull()
    expect(span!.textContent).toBe('夜色沉沉')
    expect(els[0].contains(span!)).toBe(true)
    // 高亮后段落全文无损
    expect(els[0].textContent).toBe('夜色沉沉，雨落在窗台上。')
  })

  it('两段都含目标文本时只高亮第一处（全文首个命中语义）', () => {
    const { root, els } = buildRoot('灯火将熄。', '他望着灯火出神。')
    expect(applyTextHighlight(root, true, '灯火', 'novel-reading-current')).toBe(true)
    const spans = root.querySelectorAll('span.novel-reading-current')
    expect(spans.length).toBe(1)
    expect(els[0].contains(spans[0])).toBe(true)
    expect(els[1].contains(spans[0])).toBe(false)
  })

  it('readMode 为 false 或 root 为 null：直接返回 false 且不动 DOM', () => {
    const { root } = buildRoot('夜色沉沉。')
    expect(applyTextHighlight(root, false, '夜色', 'novel-reading-current')).toBe(false)
    expect(applyTextHighlight(null, true, '夜色', 'novel-reading-current')).toBe(false)
    expect(root.querySelector('span.novel-reading-current')).toBeNull()
  })

  it('空白文本与未命中：返回 false 且不产生高亮 DOM', () => {
    const { root } = buildRoot('夜色沉沉。')
    expect(applyTextHighlight(root, true, '   ', 'novel-reading-current')).toBe(false)
    expect(applyTextHighlight(root, true, '不存在的词', 'novel-reading-current')).toBe(false)
    expect(root.querySelector('span.novel-reading-current')).toBeNull()
  })

  it('重复调用：先清同类旧高亮再标新位置，全 root 只剩一处且文本无损', () => {
    const { root, els } = buildRoot('夜色沉沉，雨落在窗台上。', '他推门而入，灯还亮着。')
    expect(applyTextHighlight(root, true, '夜色沉沉', 'novel-reading-current')).toBe(true)
    expect(applyTextHighlight(root, true, '灯还亮着', 'novel-reading-current')).toBe(true)
    const spans = root.querySelectorAll('span.novel-reading-current')
    expect(spans.length).toBe(1)
    expect(spans[0].textContent).toBe('灯还亮着')
    expect(els[0].textContent).toBe('夜色沉沉，雨落在窗台上。')
    expect(els[1].textContent).toBe('他推门而入，灯还亮着。')
  })

  it('命中片段跨内联元素边界时仍整体包进一个 span', () => {
    const { root } = buildRoot('他推门而入，<em>灯</em>还亮着。')
    expect(applyTextHighlight(root, true, '，灯还亮', 'novel-reading-current')).toBe(true)
    const span = root.querySelector('span.novel-reading-current')!
    expect(span.textContent).toBe('，灯还亮')
  })
})

describe('paraOf', () => {
  it('文本节点向上找到所属阅读段落', () => {
    const { els } = buildRoot('夜色沉沉。')
    expect(paraOf(els[0].firstChild!)).toBe(els[0])
  })

  it('段落内的内联元素子节点也归属该段落', () => {
    const { els } = buildRoot('他推门而入，<em>灯</em>还亮着。')
    expect(paraOf(els[0].querySelector('em')!)).toBe(els[0])
  })

  it('不在阅读段落内（如滚动根自身）返回 null', () => {
    const { root } = buildRoot('夜色沉沉。')
    expect(paraOf(root)).toBeNull()
  })
})

describe('textAtScrollTop', () => {
  it('取滚动位置 +48px 容差内最后到达段落的文本', () => {
    const { root, els } = buildRoot('第一段落的开头文字。', '第二段落的开头文字。', '第三段落的开头文字。')
    setOffsetTop(els[0], 0)
    setOffsetTop(els[1], 200)
    setOffsetTop(els[2], 400)
    // scrollTop=260：第三段 400 > 308 未到达，第二段 200 <= 308 命中
    expect(textAtScrollTop(root, 260)).toBe('第二段落的开头文字。')
  })

  it('滚动位置落在段落间隙时沿用上一个已到达段落', () => {
    const { root, els } = buildRoot('第一段落。', '第二段落。')
    setOffsetTop(els[0], 0)
    setOffsetTop(els[1], 300)
    // scrollTop=200：第二段 300 > 248 未到达 → 仍取第一段
    expect(textAtScrollTop(root, 200)).toBe('第一段落。')
  })

  it('摘录去首尾空白并截断到 48 字', () => {
    const { root } = buildRoot(`  ${'夜'.repeat(60)}  `)
    expect(textAtScrollTop(root, 0)).toBe('夜'.repeat(48))
  })

  it('无任何阅读段落时返回空串', () => {
    const root = document.createElement('div')
    document.body.appendChild(root)
    expect(textAtScrollTop(root, 100)).toBe('')
  })
})

describe('readSelectionInRoot', () => {
  /** 打桩 window.getSelection（jsdom 选区能力有限，直接给最小消费面对象） */
  function stubSelection(range: Range | null, text: string, collapsed = false) {
    vi.spyOn(window, 'getSelection').mockReturnValue({
      isCollapsed: collapsed,
      getRangeAt: () => range,
      toString: () => text,
    } as unknown as Selection)
  }

  it('根内单段落选择：返回折叠空白后的文本与选区几何', () => {
    const { root, els } = buildRoot('他推门而入，灯还亮着。')
    const range = document.createRange()
    // jsdom 的 Range 未实现 getBoundingClientRect（浏览器原生有），测试侧补桩
    ;(range as Range & { getBoundingClientRect?: () => DOMRect }).getBoundingClientRect =
      () => ({ top: 12, left: 34, width: 5, height: 6 } as DOMRect)
    range.setStart(els[0].firstChild!, 0)
    range.setEnd(els[0].firstChild!, 9)
    stubSelection(range, '他推门而入，   灯还亮着')
    const sel = readSelectionInRoot(root)
    expect(sel).not.toBeNull()
    expect(sel!.text).toBe('他推门而入， 灯还亮着')
    expect(sel!.rect.top).toBe(12)
  })

  it('折叠选区 / 纯空白选择 / root 为 null：均返回 null', () => {
    const { root, els } = buildRoot('夜色沉沉。')
    const range = document.createRange()
    range.setStart(els[0].firstChild!, 0)
    range.setEnd(els[0].firstChild!, 2)
    stubSelection(range, '夜色', true)
    expect(readSelectionInRoot(root)).toBeNull()
    stubSelection(range, '   ')
    expect(readSelectionInRoot(root)).toBeNull()
    stubSelection(range, '夜色')
    expect(readSelectionInRoot(null)).toBeNull()
  })

  it('跨段落选择：paraOf 归属不一致 → null（划线只在单段落内）', () => {
    const { root, els } = buildRoot('第一段落。', '第二段落。')
    const range = document.createRange()
    range.setStart(els[0].firstChild!, 0)
    range.setEnd(els[1].firstChild!, 2)
    stubSelection(range, '第一段落。第二')
    expect(readSelectionInRoot(root)).toBeNull()
  })

  it('选区落在滚动根之外：null', () => {
    const { root, els } = buildRoot('根内段落。')
    const outside = document.createElement('div')
    outside.innerHTML = '<p>根外的文字。</p>'
    document.body.appendChild(outside)
    const range = document.createRange()
    range.setStart(outside.querySelector('p')!.firstChild!, 0)
    range.setEnd(outside.querySelector('p')!.firstChild!, 2)
    stubSelection(range, '根外')
    expect(readSelectionInRoot(root)).toBeNull()
    expect(readSelectionInRoot(els[0])).toBeNull()
  })
})
