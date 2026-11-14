/**
 * paperExport.test.ts — 按纸张布图（W6 · 多页装订用例）
 *
 * 用例口径：DOMParser 解析成品字符串做结构断言（页数/页框/嵌套 svg/窗口偏移），
 * 页高页宽按 pageLayout 的纸盒手算对照（A4 竖 794×1123、装订缝 12）；
 * 覆盖缩放生效、页号递增、超限截断、空输入防御，以及「真构建器产物」的集成
 * 拼装（buildGanttExportSvg → 嵌套 svg 保留 viewBox 与内部图元）。
 */
import { describe, expect, it } from 'vitest'
import { GAP_MAX, MAX_PAGES, PAGE_GAP, PAPER_CLIP_ID, PAPER_EMPTY_NOTE, buildPaperSvg, type PaperArtifact } from './paperExport'
import { pageBoxPx, type PaperLayout } from './pageLayout'
import { buildGanttExportSvg } from './ganttExport'
import { computeCpm } from './cpm'
import type { SchedProject, SchedTask } from './types'

const A4P: PaperLayout = { paper: 'A4', orientation: 'portrait', marginMm: 10 }
const A4L: PaperLayout = { paper: 'A4', orientation: 'landscape', marginMm: 10 }
const PAGE = pageBoxPx(A4P) // 794×1123

/** 合成产物：w×h 的空图面（结构断言用，几何由 tilePages 决定） */
function fakeArt(w: number, h: number): PaperArtifact {
  return {
    svg: `<svg xmlns="http://www.w3.org/2000/svg" width="${w}" height="${h}" viewBox="0 0 ${w} ${h}" class="sched-exp-fake">` +
      `<rect width="${w}" height="${h}" fill="white"/></svg>`,
    w,
    h,
  }
}

function parse(svg: string): Document {
  return new DOMParser().parseFromString(svg, 'image/svg+xml')
}

describe('paperExport · 页数与分页', () => {
  it('适配内容只出 1 页，且仍带页框与嵌套 svg', () => {
    const out = buildPaperSvg(fakeArt(1000, 2000), A4P)
    expect(out.pages).toBe(1)
    expect(out.truncated).toBe(false)
    expect(out.scale).toBeCloseTo(1047 / 2000, 10)
    const doc = parse(out.svg)
    expect(doc.querySelectorAll('.sched-paper-page')).toHaveLength(1)
    expect(doc.querySelectorAll('.sched-paper-frame')).toHaveLength(1)
    expect(doc.querySelectorAll('.sched-paper-window > svg')).toHaveLength(1)
    const texts = Array.from(doc.querySelectorAll('text')).map((n) => n.textContent ?? '')
    expect(texts).toContain('第 1/1 页')
    expect(texts).toContain('A4 竖')
  })

  it('超大内容缩到 0.5 后分 2×2=4 页，每页都有嵌套 svg 与页框', () => {
    const out = buildPaperSvg(fakeArt(2600, 2800), A4P)
    expect(out.scale).toBe(0.5)
    expect(out.pages).toBe(4)
    const doc = parse(out.svg)
    expect(doc.querySelectorAll('.sched-paper-page')).toHaveLength(4)
    expect(doc.querySelectorAll('.sched-paper-frame')).toHaveLength(4)
    expect(doc.querySelectorAll('.sched-paper-window > svg')).toHaveLength(4)
  })

  it('横向纸张页盒交换（1123×794），总高按页高 + 装订缝累加', () => {
    const out = buildPaperSvg(fakeArt(2600, 2800), A4L)
    const box = pageBoxPx(A4L)
    expect(out.w).toBe(box.w)
    expect(out.h).toBe(out.pages * box.h + (out.pages - 1) * PAGE_GAP)
    expect(PAGE_GAP).toBe(12)
    expect(parse(out.svg).documentElement.getAttribute('height')).toBe(String(out.h))
  })

  it('页号递增 1..N，各页 transform 逐页下移「页高 + 装订缝」', () => {
    const out = buildPaperSvg(fakeArt(2600, 2800), A4P)
    const pages = Array.from(parse(out.svg).querySelectorAll('.sched-paper-page'))
    expect(pages.map((p) => p.getAttribute('data-page'))).toEqual(['1', '2', '3', '4'])
    expect(pages.map((p) => p.getAttribute('transform'))).toEqual([
      'translate(0 0)',
      `translate(0 ${PAGE.h + PAGE_GAP})`,
      `translate(0 ${(PAGE.h + PAGE_GAP) * 2})`,
      `translate(0 ${(PAGE.h + PAGE_GAP) * 3})`,
    ])
    const nos = pages.map((p) => p.querySelector('.sched-paper-frame-no')!.textContent)
    expect(nos).toEqual(['第 1/4 页', '第 2/4 页', '第 3/4 页', '第 4/4 页'])
  })

  it('缩放生效：嵌套 svg 视口 = 缩放后内容尺寸，viewBox 保留原内容坐标', () => {
    const out = buildPaperSvg(fakeArt(2600, 2800), A4P)
    const nested = parse(out.svg).querySelector('.sched-paper-window > svg')!
    expect(nested.getAttribute('width')).toBe('1300')
    expect(nested.getAttribute('height')).toBe('1400')
    expect(nested.getAttribute('viewBox')).toBe('0 0 2600 2800')
    expect(nested.getAttribute('preserveAspectRatio')).toBe('xMinYMin meet')
    expect(nested.getAttribute('class')).toBe('sched-exp-fake')
  })

  it('每页窗口取景：dx/dy 落 data-*，内容组按 -dx/-dy 平移', () => {
    const out = buildPaperSvg(fakeArt(2600, 2800), A4P)
    const slots = Array.from(parse(out.svg).querySelectorAll('.sched-paper-slot'))
    expect(slots.map((s) => [s.getAttribute('data-dx'), s.getAttribute('data-dy'), s.getAttribute('data-w'), s.getAttribute('data-h')])).toEqual([
      ['0', '0', '718', '1047'],
      ['718', '0', '582', '1047'],
      ['0', '1047', '718', '353'],
      ['718', '1047', '582', '353'],
    ])
    expect(Array.from(parse(out.svg).querySelectorAll('.sched-paper-window')).map((w) => w.getAttribute('transform'))).toEqual([
      'translate(0 0)',
      'translate(-718 0)',
      'translate(0 -1047)',
      'translate(-718 -1047)',
    ])
  })

  it('裁剪到内容盒：clipPath 存在且每页引用同一 id', () => {
    const out = buildPaperSvg(fakeArt(2600, 2800), A4P)
    const doc = parse(out.svg)
    expect(doc.querySelectorAll(`clipPath#${PAPER_CLIP_ID}`)).toHaveLength(1)
    expect(doc.querySelector(`clipPath#${PAPER_CLIP_ID} rect`)!.getAttribute('width')).toBe('718')
    const slots = Array.from(doc.querySelectorAll('.sched-paper-slot'))
    expect(slots.every((s) => s.getAttribute('clip-path') === `url(#${PAPER_CLIP_ID})`)).toBe(true)
  })
})

describe('paperExport · 防御与选项', () => {
  it('页数超上限按页序截断：pages=MAX_PAGES、truncated=true、末页标注', () => {
    const out = buildPaperSvg(fakeArt(30000, 60000), A4P)
    expect(MAX_PAGES).toBe(400)
    expect(out.pages).toBe(MAX_PAGES)
    expect(out.truncated).toBe(true)
    const doc = parse(out.svg)
    expect(doc.querySelectorAll('.sched-paper-page')).toHaveLength(MAX_PAGES)
    const note = doc.querySelector('.sched-paper-cut-note')
    expect(note).toBeTruthy()
    expect(note!.textContent).toContain('已截断')
    expect(note!.textContent).toContain('400')
  })

  it('空输入防御：art 缺 svg / 尺寸 0 出占位页，pages=0 且纸盒仍是真实纸张', () => {
    for (const art of [
      { svg: '', w: 0, h: 0 },
      { svg: '<div/>', w: 100, h: 100 },
      { svg: '<svg xmlns="http://www.w3.org/2000/svg"/>', w: 0, h: 100 },
      { svg: '<svg xmlns="http://www.w3.org/2000/svg"/>', w: Number.NaN, h: 200 },
    ] as PaperArtifact[]) {
      const out = buildPaperSvg(art, A4P)
      expect(out.pages).toBe(0)
      expect(out.truncated).toBe(false)
      expect(out.w).toBe(PAGE.w)
      expect(out.h).toBe(PAGE.h)
      expect(out.svg).toContain(PAPER_EMPTY_NOTE)
      expect(parse(out.svg).querySelectorAll('.sched-paper-frame')).toHaveLength(1)
    }
  })

  it('opts.page=false 只出取景窗，不出页框与页号', () => {
    const out = buildPaperSvg(fakeArt(2600, 2800), A4P, { page: false })
    const doc = parse(out.svg)
    expect(out.pages).toBe(4)
    expect(doc.querySelectorAll('.sched-paper-frame')).toHaveLength(0)
    expect(doc.querySelectorAll('.sched-paper-window > svg')).toHaveLength(4)
    expect(Array.from(doc.querySelectorAll('text')).map((n) => n.textContent)).not.toContain('第 1/4 页')
  })

  it('opts.title 落到 <title> 元数据且转义；不上纸面正文', () => {
    const out = buildPaperSvg(fakeArt(1000, 2000), A4P, { title: '雨污分流 <上报件>' })
    const doc = parse(out.svg)
    expect(doc.querySelector('title')!.textContent).toBe('雨污分流 <上报件>')
    expect(out.svg).toContain('&lt;上报件&gt;')
    expect(Array.from(doc.querySelectorAll('text')).map((n) => n.textContent)).not.toContain('雨污分流 <上报件>')
  })

  it('输出 w/h 与页数自洽（w=纸宽，h=页数×页高+(页数-1)×缝）', () => {
    for (const art of [fakeArt(1000, 2000), fakeArt(2600, 2800), fakeArt(30000, 60000)]) {
      const out = buildPaperSvg(art, A4P)
      expect(out.w).toBe(PAGE.w)
      expect(out.h).toBe(out.pages * PAGE.h + (out.pages - 1) * PAGE_GAP)
      expect(parse(out.svg).documentElement.getAttribute('viewBox')).toBe(`0 0 ${out.w} ${out.h}`)
    }
  })

  it('gap:0（打印口径）：页数/页框不变、页距恰为页高、总高 = 页数×页高', () => {
    const base = buildPaperSvg(fakeArt(2600, 2800), A4P)
    const out = buildPaperSvg(fakeArt(2600, 2800), A4P, { gap: 0 })
    expect(out.pages).toBe(base.pages)
    expect(out.w).toBe(base.w)
    expect(out.h).toBe(out.pages * PAGE.h)
    const doc = parse(out.svg)
    const pages = Array.from(doc.querySelectorAll('.sched-paper-page'))
    expect(pages).toHaveLength(out.pages)
    expect(doc.querySelectorAll('.sched-paper-frame')).toHaveLength(out.pages)
    expect(doc.documentElement.getAttribute('height')).toBe(String(out.pages * PAGE.h))
    expect(pages.map((p) => p.getAttribute('transform'))).toEqual([
      'translate(0 0)',
      'translate(0 1123)',
      'translate(0 2246)',
      'translate(0 3369)',
    ])
    // 相邻页顶差恰为页高（无缝累计，@page 分页才对得齐）
    const tops = pages.map((p) => Number(/translate\(0 ([\d.]+)\)/.exec(p.getAttribute('transform')!)![1]))
    expect(tops.map((t, i) => (i === 0 ? 0 : t - tops[i - 1]))).toEqual([0, PAGE.h, PAGE.h, PAGE.h])
  })

  it('gap 脏值收敛：负数/NaN/±Infinity/缺省回落 12，超上限钳 64', () => {
    const base = buildPaperSvg(fakeArt(2600, 2800), A4P)
    expect(base.h).toBe(base.pages * PAGE.h + (base.pages - 1) * PAGE_GAP)
    for (const gap of [-5, Number.NaN, Number.POSITIVE_INFINITY, Number.NEGATIVE_INFINITY, undefined]) {
      const out = buildPaperSvg(fakeArt(2600, 2800), A4P, { gap })
      expect(out.h).toBe(base.h)
      expect(out.pages).toBe(base.pages)
    }
    const capped = buildPaperSvg(fakeArt(2600, 2800), A4P, { gap: 200 })
    expect(GAP_MAX).toBe(64)
    expect(capped.h).toBe(capped.pages * PAGE.h + (capped.pages - 1) * GAP_MAX)
  })
})

describe('paperExport · 成品良构性', () => {
  it('成品是良构 XML（DOMParser 无 parsererror），根唯一 svg + 每页一份嵌套 svg', () => {
    const out = buildPaperSvg(fakeArt(2600, 2800), A4P, { title: '布图' })
    const doc = parse(out.svg)
    expect(doc.querySelector('parsererror')).toBeNull()
    expect(doc.documentElement.tagName.toLowerCase()).toBe('svg')
    expect(doc.querySelectorAll('svg').length).toBe(1 + out.pages)
    // 根上只该有一条 viewBox（页盒口径），嵌套 svg 各自保留内容坐标系
    expect(doc.documentElement.getAttribute('viewBox')).toBe(`0 0 ${out.w} ${out.h}`)
    expect(doc.querySelector('.sched-paper-window > svg')!.getAttribute('viewBox')).toBe('0 0 2600 2800')
  })
})

describe('paperExport · 与真构建器集成', () => {
  function t(id: string, duration: number, level = 1): SchedTask {
    return { id, name: `任务${id}`, duration, level, progress: 0 }
  }
  const project: SchedProject = {
    name: '雨污分流改造工程',
    startDate: '2026-09-07',
    tasks: [t('G', 0, 0), t('A', 3), t('B', 5)],
    links: [{ from: 'A', to: 'B', type: 'FS', lag: 0 }],
  }

  it('横道上报件 → 按 A4 竖布图：嵌套 svg 保留 class/viewBox，图元完整', () => {
    const cpm = computeCpm(project.tasks, project.links)
    const art = buildGanttExportSvg(project, cpm, { title: '施工进度计划横道图' })
    const out = buildPaperSvg(art, A4P)
    expect(out.pages).toBeGreaterThanOrEqual(1)
    const doc = parse(out.svg)
    const nested = doc.querySelector('.sched-paper-window > svg')!
    expect(nested.getAttribute('class')).toBe('sched-exp-gantt')
    expect(nested.getAttribute('viewBox')).toBe(`0 0 ${art.w} ${art.h}`)
    // 图元随嵌套 svg 原样带入：条形与图签都在
    expect(doc.querySelectorAll('.sched-exp-bar').length).toBeGreaterThan(0)
    expect(doc.querySelectorAll('.sched-exp-sign rect')).toHaveLength(3)
    // 缩放不放大：内容盒装得下时 1:1
    expect(out.scale).toBeLessThanOrEqual(1)
    expect(Number(nested.getAttribute('width'))).toBeLessThanOrEqual(718)
  })
})
