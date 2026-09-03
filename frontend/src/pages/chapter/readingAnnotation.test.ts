// readingAnnotation.test.ts — 阅读页「划线 / 想法」回渲染工具测试。
// planAnnotationMatches 为纯函数（长短摘录占位、降序输出），renderAnnotationHighlights
// 用 jsdom 造 .novel-reading-p 段落结构，锁 mark 重建、点击回调与旧 mark 清理。
import { afterEach, describe, expect, it, vi } from 'vitest'
import { planAnnotationMatches, renderAnnotationHighlights } from './readingAnnotation'
import type { ReadingAnnotation } from '../../utils/readingAnnotations'

/** 最小划线对象（仅 id/text 可变，其余字段对匹配逻辑无意义） */
function makeAnn(id: string, text: string): ReadingAnnotation {
  return { id, nodeId: 'ch-1', title: '第一回', color: 'yellow', text, note: '', createdAt: 0 }
}

/** 造一个滚动根，每段包成 .novel-reading-p（模拟阅读列 DOM） */
function buildRoot(...paras: string[]): { root: HTMLElement; els: HTMLElement[] } {
  const root = document.createElement('div')
  root.innerHTML = paras.map((t) => `<p class="novel-reading-p">${t}</p>`).join('')
  document.body.appendChild(root)
  const els = Array.from(root.querySelectorAll<HTMLElement>('.novel-reading-p'))
  return { root, els }
}

afterEach(() => {
  document.body.innerHTML = ''
  vi.restoreAllMocks()
})

describe('planAnnotationMatches', () => {
  it('单条摘录命中：返回段内 UTF-16 区间', () => {
    const anns = [makeAnn('a1', '灯火渐暗')]
    // 夜0幕1降2临3，4灯5火6渐7暗8。9
    expect(planAnnotationMatches('夜幕降临，灯火渐暗。', anns)).toEqual([
      { start: 5, end: 9, ann: anns[0] },
    ])
  })

  it('多条摘录按起始偏移降序返回（从后往前套 mark 不串位）', () => {
    const a1 = makeAnn('a1', '夜幕')
    const a2 = makeAnn('a2', '渐暗')
    const matches = planAnnotationMatches('夜幕降临，灯火渐暗。', [a1, a2])
    expect(matches.map((m) => m.ann.id)).toEqual(['a2', 'a1'])
    expect(matches.map((m) => m.start)).toEqual([7, 0])
  })

  it('长摘录优先占位，与其重叠的短摘录无冲突位置时让位', () => {
    const long = makeAnn('long', '灯火渐暗的夜里')
    const short = makeAnn('short', '渐暗')
    const matches = planAnnotationMatches('灯火渐暗的夜里。', [short, long])
    expect(matches.length).toBe(1)
    expect(matches[0].ann.id).toBe('long')
  })

  it('同一摘录取首个未被占用的出现位置', () => {
    const a1 = makeAnn('a1', '灯火初上')
    const a2 = makeAnn('a2', '灯火')
    // 灯0火1初2上3，4再5叹6灯7火8。9：「灯火」出现于 0 与 7，0 已被 a1 占用 → 取 7
    const matches = planAnnotationMatches('灯火初上，再叹灯火。', [a1, a2])
    expect(matches.map((m) => m.ann.id)).toEqual(['a2', 'a1'])
    expect(matches[0].start).toBe(7)
  })

  it('空摘录与未命中的摘录被跳过，入参列表不被修改', () => {
    const anns = [makeAnn('empty', ''), makeAnn('miss', '不存在的摘录')]
    expect(planAnnotationMatches('夜幕降临。', anns)).toEqual([])
    expect(anns.map((a) => a.id)).toEqual(['empty', 'miss'])
  })
})

describe('renderAnnotationHighlights', () => {
  it('划线命中：摘录包进 mark[data-ann-id]，段落全文无损', () => {
    const { root } = buildRoot('夜幕降临，灯火渐暗。')
    renderAnnotationHighlights(root, [makeAnn('a1', '灯火渐暗')], () => {})
    const mark = root.querySelector('mark.novel-reading-mark')
    expect(mark).not.toBeNull()
    expect(mark!.getAttribute('data-ann-id')).toBe('a1')
    expect(mark!.getAttribute('data-ann-color')).toBe('yellow')
    expect(mark!.textContent).toBe('灯火渐暗')
    expect(root.querySelector('.novel-reading-p')!.textContent).toBe('夜幕降临，灯火渐暗。')
  })

  it('mark 点击回调 onOpen 收到对应划线对象', () => {
    const { root } = buildRoot('夜幕降临。')
    const onOpen = vi.fn()
    renderAnnotationHighlights(root, [makeAnn('a1', '夜幕')], onOpen)
    const mark = root.querySelector('mark[data-ann-id="a1"]')!
    mark.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    expect(onOpen).toHaveBeenCalledTimes(1)
    expect(onOpen).toHaveBeenCalledWith(expect.objectContaining({ id: 'a1' }))
  })

  it('重复渲染：先解旧 mark 再重建，mark 数量与段落文本保持', () => {
    const { root } = buildRoot('夜幕降临，灯火渐暗。')
    const anns = [makeAnn('a1', '灯火渐暗')]
    renderAnnotationHighlights(root, anns, () => {})
    renderAnnotationHighlights(root, anns, () => {})
    const marks = root.querySelectorAll('mark[data-ann-id]')
    expect(marks.length).toBe(1)
    expect(marks[0].textContent).toBe('灯火渐暗')
    expect(root.querySelector('.novel-reading-p')!.textContent).toBe('夜幕降临，灯火渐暗。')
  })

  it('current 为空时只清旧 mark 不新建（切换章节/删除划线场景）', () => {
    const { root } = buildRoot('夜幕降临。')
    renderAnnotationHighlights(root, [makeAnn('a1', '夜幕')], () => {})
    expect(root.querySelectorAll('mark').length).toBe(1)
    renderAnnotationHighlights(root, [], () => {})
    expect(root.querySelectorAll('mark').length).toBe(0)
    expect(root.querySelector('.novel-reading-p')!.textContent).toBe('夜幕降临。')
  })

  it('摘录已被编辑掉：不产生 mark（内容漂移自然失效）', () => {
    const { root } = buildRoot('内容已被改写。')
    renderAnnotationHighlights(root, [makeAnn('a1', '夜幕降临')], () => {})
    expect(root.querySelector('mark')).toBeNull()
  })

  it('两条摘录同段共存：各得一个 mark 且互不嵌套错位', () => {
    const { root, els } = buildRoot('夜幕降临，灯火渐暗。')
    renderAnnotationHighlights(root, [makeAnn('a1', '夜幕'), makeAnn('a2', '渐暗')], () => {})
    const marks = Array.from(root.querySelectorAll('mark[data-ann-id]'))
    expect(marks.length).toBe(2)
    expect(els[0].textContent).toBe('夜幕降临，灯火渐暗。')
    expect(marks.map((m) => m.textContent).sort()).toEqual(['夜幕', '渐暗'].sort())
  })
})
