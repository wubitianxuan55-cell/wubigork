/**
 * pageLayout.test.ts — 按纸张布图几何（W6 · 纯函数用例）
 *
 * 用例口径：mm→px 按 96dpi 手算对照（210mm=794px、297mm=1123px、10mm=38px），
 * 分页用「内容盒 718×1047（A4 竖 10mm 边距）」做整数化推演；页框用 DOMParser
 * 断言结构（data-page/data-total + 图幅标记 + 页码文本）与角标不越纸边。
 * 防御面：0/负/非有限尺寸一律不抛，且不产生负尺寸页。
 */
import { describe, expect, it } from 'vitest'
import {
  DEFAULT_MIN_SCALE, FRAME_INK, PAPER_MM, TILE_MAX_PAGES, contentBoxPx, escapeXml, fitScale,
  frameSvg, mmToPx, marginPx, pageBoxPx, paperMark, svgNum, tilePages, type PaperLayout,
} from './pageLayout'

/** A4 竖 / 10mm 边距 / 96dpi —— 用例主夹具（内容盒 718×1047） */
const A4P: PaperLayout = { paper: 'A4', orientation: 'portrait', marginMm: 10 }
const A4L: PaperLayout = { paper: 'A4', orientation: 'landscape', marginMm: 10 }
const A3P: PaperLayout = { paper: 'A3', orientation: 'portrait', marginMm: 10 }
const A3L: PaperLayout = { paper: 'A3', orientation: 'landscape', marginMm: 10 }
const A2P: PaperLayout = { paper: 'A2', orientation: 'portrait', marginMm: 10 }
const A2L: PaperLayout = { paper: 'A2', orientation: 'landscape', marginMm: 10 }

/** 把页框片段包进带命名空间的根，供 DOMParser 解析 */
function parse(fragment: string): Document {
  return new DOMParser().parseFromString(`<svg xmlns="http://www.w3.org/2000/svg">${fragment}</svg>`, 'image/svg+xml')
}

describe('pageLayout · 纸张与像素盒', () => {
  it('PAPER_MM 是 A 系列真实毫米尺寸（210×297 / 297×420 / 420×594）', () => {
    expect(PAPER_MM.A4).toEqual({ w: 210, h: 297 })
    expect(PAPER_MM.A3).toEqual({ w: 297, h: 420 })
    expect(PAPER_MM.A2).toEqual({ w: 420, h: 594 })
  })

  it('mmToPx 按 96dpi 四舍五入（210/297/420/594/10mm）', () => {
    expect(mmToPx(210)).toBe(794)
    expect(mmToPx(297)).toBe(1123)
    expect(mmToPx(420)).toBe(1587)
    expect(mmToPx(594)).toBe(2245)
    expect(mmToPx(10)).toBe(38)
  })

  it('mmToPx 非法 dpi 回落 96、非法 mm 归零', () => {
    expect(mmToPx(210, 0)).toBe(794)
    expect(mmToPx(210, Number.NaN)).toBe(794)
    expect(mmToPx(Number.POSITIVE_INFINITY)).toBe(0)
  })

  it('A4 竖纸盒 794×1123，横纸盒交换为 1123×794', () => {
    expect(pageBoxPx(A4P)).toEqual({ w: 794, h: 1123 })
    expect(pageBoxPx(A4L)).toEqual({ w: 1123, h: 794 })
  })

  it('A3/A2 横竖纸盒', () => {
    expect(pageBoxPx(A3P)).toEqual({ w: 1123, h: 1587 })
    expect(pageBoxPx(A3L)).toEqual({ w: 1587, h: 1123 })
    expect(pageBoxPx(A2P)).toEqual({ w: 1587, h: 2245 })
    expect(pageBoxPx(A2L)).toEqual({ w: 2245, h: 1587 })
  })

  it('dpi 翻倍按比例放大（A4 竖 @192 = 1587×2245）', () => {
    expect(pageBoxPx({ ...A4P, dpi: 192 })).toEqual({ w: 1587, h: 2245 })
  })

  it('内容盒扣四边页边距（A4 竖 10mm：718×1047；A4 横：1047×718）', () => {
    expect(marginPx(A4P)).toBe(38)
    expect(contentBoxPx(A4P)).toEqual({ w: 718, h: 1047 })
    expect(contentBoxPx(A4L)).toEqual({ w: 1047, h: 718 })
  })

  it('边距过大钳到 ≥1px，负边距按 0（等于无边距内容盒）', () => {
    const huge = contentBoxPx({ ...A4P, marginMm: 500 })
    expect(huge.w).toBeGreaterThanOrEqual(1)
    expect(huge.h).toBeGreaterThanOrEqual(1)
    expect(contentBoxPx({ ...A4P, marginMm: -5 })).toEqual(contentBoxPx({ ...A4P, marginMm: 0 }))
  })
})

describe('pageLayout · 缩放', () => {
  it('不放大：内容小于内容盒时 scale=1', () => {
    expect(fitScale({ w: 400, h: 300 }, { w: 718, h: 1047 })).toBe(1)
    expect(fitScale({ w: 718, h: 1047 }, { w: 718, h: 1047 })).toBe(1)
  })

  it('等比缩到较短边：2000×1000 进 1000×1000 = 0.5', () => {
    expect(fitScale({ w: 2000, h: 1000 }, { w: 1000, h: 1000 })).toBe(0.5)
    expect(fitScale({ w: 500, h: 4000 }, { w: 1000, h: 1000 })).toBe(0.25)
  })

  it('0/负/非有限尺寸 → 中性 1', () => {
    expect(fitScale({ w: 0, h: 100 }, { w: 718, h: 1047 })).toBe(1)
    expect(fitScale({ w: 100, h: -5 }, { w: 718, h: 1047 })).toBe(1)
    expect(fitScale({ w: Number.NaN, h: 100 }, { w: 718, h: 1047 })).toBe(1)
    expect(fitScale({ w: 100, h: 100 }, { w: 0, h: 0 })).toBe(1)
  })
})

describe('pageLayout · 分页网格', () => {
  it('适配内容只出 1 页：800×600 → scale=718/800、1 页、offset 0', () => {
    const t = tilePages({ w: 800, h: 600 }, A4P)
    expect(t.scale).toBeCloseTo(718 / 800, 10)
    expect(t.cols).toBe(1)
    expect(t.rows).toBe(1)
    expect(t.pages).toHaveLength(1)
    expect(t.pages[0]).toMatchObject({ index: 1, row: 0, col: 0, dx: 0, dy: 0 })
    expect(t.pages[0].w).toBeCloseTo(718, 6)
    expect(t.pages[0].h).toBeCloseTo(600 * (718 / 800), 6)
    expect(t.truncated).toBe(false)
  })

  it('超大内容缩到 minScale 即停、余量转分页（2000×3000 → 0.5 / 2×2 页）', () => {
    const t = tilePages({ w: 2000, h: 3000 }, A4P)
    expect(DEFAULT_MIN_SCALE).toBe(0.5)
    expect(t.scale).toBe(0.5)
    expect(t.contentW).toBe(1000)
    expect(t.contentH).toBe(1500)
    expect([t.cols, t.rows, t.pages.length]).toEqual([2, 2, 4])
    expect(t.pages[0]).toMatchObject({ index: 1, row: 0, col: 0, dx: 0, dy: 0, w: 718, h: 1047 })
    expect(t.pages[1]).toMatchObject({ index: 2, row: 0, col: 1, dx: 718, dy: 0, w: 282, h: 1047 })
    expect(t.pages[2]).toMatchObject({ index: 3, row: 1, col: 0, dx: 0, dy: 1047, w: 718, h: 453 })
    expect(t.pages[3]).toMatchObject({ index: 4, row: 1, col: 1, dx: 718, dy: 1047, w: 282, h: 453 })
  })

  it('页序为先行后列（row-major）：对角线页索引递增且同列相邻页 dx 相同', () => {
    const t = tilePages({ w: 2000, h: 3000 }, A4P)
    expect(t.pages.map((p) => p.index)).toEqual([1, 2, 3, 4])
    expect(t.pages.map((p) => `${p.row},${p.col}`)).toEqual(['0,0', '0,1', '1,0', '1,1'])
    expect(t.pages[0].dx).toBe(t.pages[2].dx)
    expect(t.pages[1].dx).toBe(t.pages[3].dx)
    expect(t.pages[0].dy).toBe(t.pages[1].dy)
  })

  it('minScale 调大即「不缩、多页」（minScale=1 → 3×3=9 页）', () => {
    const t = tilePages({ w: 2000, h: 3000 }, A4P, { minScale: 1 })
    expect(t.scale).toBe(1)
    expect([t.cols, t.rows, t.pages.length]).toEqual([3, 3, 9])
  })

  it('整倍内容不多长空白页（2154×3141 = 3×3 个整内容盒）', () => {
    const t = tilePages({ w: 2154, h: 3141 }, A4P, { minScale: 1 })
    expect([t.cols, t.rows, t.pages.length]).toEqual([3, 3, 9])
    expect(t.pages.every((p) => p.w === 718 && p.h === 1047)).toBe(true)
  })

  it('页数超 maxPages 截断并标 truncated，末页仍在网格内', () => {
    const t = tilePages({ w: 2154, h: 3141 }, A4P, { minScale: 1, maxPages: 2 })
    expect(t.cols).toBe(3)
    expect(t.rows).toBe(3)
    expect(t.pages).toHaveLength(2)
    expect(t.truncated).toBe(true)
    expect(t.pages[1]).toMatchObject({ index: 2, row: 0, col: 1 })
    expect(TILE_MAX_PAGES).toBe(400)
  })

  it('0/负/非有限内容尺寸 → 空页表、不抛、不产生负尺寸页', () => {
    for (const c of [{ w: 0, h: 0 }, { w: -100, h: 200 }, { w: Number.NaN, h: 10 }]) {
      const t = tilePages(c, A4P)
      expect(t.pages).toEqual([])
      expect([t.cols, t.rows]).toEqual([0, 0])
      expect(t.scale).toBe(1)
      expect(t.boxW).toBe(718)
    }
  })
})

describe('pageLayout · 页框', () => {
  it('页框含外框/内框/图幅标记/页码，data-* 带页序', () => {
    const doc = parse(frameSvg(A4L, 2, 5))
    const frame = doc.querySelector('.sched-paper-frame')!
    expect(frame.getAttribute('data-paper')).toBe('A4')
    expect(frame.getAttribute('data-orient')).toBe('landscape')
    expect(frame.getAttribute('data-page')).toBe('2')
    expect(frame.getAttribute('data-total')).toBe('5')
    expect(frame.getAttribute('class')).toContain('sched-paper-frame')
    expect(doc.querySelectorAll('.sched-paper-frame-outer')).toHaveLength(1)
    expect(doc.querySelectorAll('.sched-paper-frame-inner')).toHaveLength(1)
    const texts = Array.from(doc.querySelectorAll('text')).map((n) => n.textContent ?? '')
    expect(texts).toContain('第 2/5 页')
    expect(texts).toContain('A4 横')
    expect(texts).not.toContain('第 2/0 页')
  })

  it('页框外框贴纸边（x=半个线宽），内框贴内容盒边界（38px）', () => {
    const doc = parse(frameSvg(A4P, 1, 1))
    const outer = doc.querySelector('.sched-paper-frame-outer')!
    expect(outer.getAttribute('x')).toBe('0.5')
    expect(outer.getAttribute('width')).toBe(String(794 - 1))
    const inner = doc.querySelector('.sched-paper-frame-inner')!
    expect(inner.getAttribute('x')).toBe('38')
    expect(inner.getAttribute('width')).toBe(String(794 - 76))
  })

  it('页码钳位：page<1 取 1、page>total 取 total、total 非法取 1', () => {
    expect(parse(frameSvg(A4P, 0, 3)).querySelector('.sched-paper-frame')!.getAttribute('data-page')).toBe('1')
    expect(parse(frameSvg(A4P, 9, 3)).querySelector('.sched-paper-frame')!.getAttribute('data-page')).toBe('3')
    const texts = Array.from(parse(frameSvg(A4P, 5, 0)).querySelectorAll('text')).map((n) => n.textContent ?? '')
    expect(texts).toContain('第 1/1 页')
  })

  it('页脚线只在底边距放得下时画；窄边距只留角标', () => {
    expect(parse(frameSvg(A4P, 1, 1)).querySelectorAll('.sched-paper-frame-rule')).toHaveLength(1)
    const narrow = parse(frameSvg({ ...A4P, marginMm: 2 }, 1, 1))
    expect(narrow.querySelectorAll('.sched-paper-frame-rule')).toHaveLength(0)
    expect(Array.from(narrow.querySelectorAll('text')).map((n) => n.textContent ?? '')).toContain('第 1/1 页')
  })

  it('可关内框/页脚，可覆盖墨色与字号（打印件黑白语义缺省纯黑）', () => {
    expect(FRAME_INK).toBe('black')
    const bare = parse(frameSvg(A4P, 1, 2, { inner: false, footer: false }))
    expect(bare.querySelectorAll('.sched-paper-frame-inner')).toHaveLength(0)
    expect(bare.querySelectorAll('text')).toHaveLength(0)
    const styled = parse(frameSvg(A4P, 1, 2, { stroke: 'maroon', fontSize: 12 }))
    expect(styled.querySelector('.sched-paper-frame-outer')!.getAttribute('stroke')).toBe('maroon')
    const no = styled.querySelector('.sched-paper-frame-no')!
    expect(no.getAttribute('fill')).toBe('maroon')
    expect(no.getAttribute('font-size')).toBe('12')
  })

  it('图幅标记与转义：A4 横 / A3 竖，escapeXml 覆盖五字符', () => {
    expect(paperMark(A4L)).toBe('A4 横')
    expect(paperMark(A3P)).toBe('A3 竖')
    expect(escapeXml(`<a href="x">'&'</a>`)).toBe('&lt;a href=&quot;x&quot;&gt;&#39;&amp;&#39;&lt;/a&gt;')
    expect(svgNum(0.30000000000000004)).toBe('0.3')
    expect(svgNum(Number.NaN)).toBe('0')
  })
})
