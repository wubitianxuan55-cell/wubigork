/**
 * PdmView.test.tsx — 单代号视图冒烟 + 刀H 过桥法接线（v4.130.0）
 *
 * PdmView 此前无组件测试；刀H 把连线几何改走 aoaLayout 段序列
 * （findBridgeArcs/segsToPath 与双代号同源），补冒烟钉住：节点/搭接/标签
 * 渲染齐全，且真实交叉出现过桥半圆（d 含 A 命令）。
 * 交叉构型：三行起步（A/B/C 同列 0/1/2 行），C→D 竖段纵跨 2 行距，
 * B→E 跨两列的首段水平线（y=行1）从其中间穿过——PDM 几何中横竖段都在
 * 节点行中线上，普通两行项目只会端点相触，必须 ≥3 行 + 跨列长横线才有内交。
 */
import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { computeCpm } from './cpm'
import { PdmView, isLinkBinding } from './PdmView'
import type { SchedNetworkStyle, SchedProject } from './types'

function crossingProject(): SchedProject {
  return {
    name: '过桥样板',
    startDate: '2026-09-07',
    tasks: [
      { id: 'A', name: '甲', duration: 2, level: 1, progress: 0 },
      { id: 'B', name: '乙', duration: 1, level: 1, progress: 0 },
      { id: 'C', name: '丙', duration: 4, level: 1, progress: 0 },
      { id: 'D', name: '丁', duration: 2, level: 1, progress: 0 },
      { id: 'E', name: '戊', duration: 1, level: 1, progress: 0 },
    ],
    links: [
      { from: 'C', to: 'D', type: 'FS', lag: 0 },
      { from: 'A', to: 'E', type: 'FS', lag: 0 },
      { from: 'D', to: 'E', type: 'FS', lag: 0 },
      { from: 'B', to: 'E', type: 'FS', lag: 0 },
    ],
  }
}

describe('PdmView（刀H G1 过桥接线）', () => {
  it('节点/搭接/标签渲染齐全', () => {
    const p = crossingProject()
    render(<PdmView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const root = screen.getByTestId('sched-pdm')
    expect(root.textContent).toContain('丙')
    expect(root.textContent).toContain('FS')
    // 实搭接 4 条（虚拟 S 桩同挂 sched-net-link 类，排除后计数）
    expect(root.querySelectorAll('path.sched-net-link:not(.sched-net-link-virtual)').length).toBe(4)
  })

  it('连线交叉处出现过桥半圆（d 含 A 命令）', () => {
    const p = crossingProject()
    render(<PdmView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    // 图例里有 antd 按钮图标 svg，必须按类名取网络图连线（桥画在连线 path 上）
    const links = Array.from(screen.getByTestId('sched-pdm').querySelectorAll('path.sched-net-link, path.sched-net-link-virtual'))
    const bridged = links.filter((el) => (el.getAttribute('d') ?? '').includes('A 5 5'))
    expect(bridged.length).toBe(1) // C→D 的竖段被 B→E 的长横线内交，恰一处
  })
})

describe('PdmView 滚轮缩放（v4.159）', () => {
  it('普通滚轮：以光标为锚缩放；Shift+滚轮放行原生横移不缩放', () => {
    const p = crossingProject()
    render(<PdmView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const el = screen.getByTestId('sched-pdm')
    expect(screen.getByTestId('sched-pdm-zoom').textContent).toBe('100%')
    fireEvent.wheel(el, { deltaY: -100, clientX: 60, clientY: 40 })
    expect(screen.getByTestId('sched-pdm-zoom').textContent).toBe('120%')
    fireEvent.wheel(el, { deltaY: -100, shiftKey: true })
    expect(screen.getByTestId('sched-pdm-zoom').textContent).toBe('120%')
  })
})

// ── v4.366 W4「图面专业密度」：四开关 + 节点几何参数 ─────────────────
//
// 与双代号同一份解析入口（networkDisplay.ts）。零回归口径：四开关全缺省时
// 节点仍是六格标注法（ES/D/EF · LS/TF/LF）、无附加行、搭接标签照画。

describe('PdmView 图面四开关与几何参数（W4）', () => {
  const styled = (style?: SchedNetworkStyle): SchedProject => {
    const p = crossingProject()
    if (style) p.networkStyle = style
    return p
  }

  it('缺省：六格标注法齐备、无附加行、搭接标签照画', () => {
    const p = styled(undefined)
    render(<PdmView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const root = screen.getByTestId('sched-pdm')
    const node = root.querySelector('.sched-pdm-node:not(.sched-pdm-virtual)')!
    expect(node.querySelectorAll('.sched-pdm-grid span')).toHaveLength(6)
    expect(node.querySelector('.sched-net-pdm-extra-body')).toBeNull()
    expect(root.querySelectorAll('text.sched-net-label').length).toBeGreaterThan(0)
  })

  it('showSixPara=true：节点框下出六参数（两列三行，ES/EF/LS/LF/TF/FF 顺序）', () => {
    const p = styled({ showSixPara: true })
    render(<PdmView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const block = screen.getByTestId('sched-pdm').querySelector('[data-testid^="sched-net-pdm-sixpara-"]')
    expect(block).not.toBeNull()
    const lines = Array.from(block!.querySelectorAll('.sched-net-sixpara-line')).map((el) => el.textContent ?? '')
    expect(lines).toHaveLength(6)
    expect(lines.map((l) => l.split(' ')[0])).toEqual(['ES', 'EF', 'LS', 'LF', 'TF', 'FF'])
  })

  it('showStartStopTime=true：节点框下出「计划开始 → 计划完成」（M/D 口径）', () => {
    const p = styled({ showStartStopTime: true })
    render(<PdmView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const line = screen.getByTestId('sched-pdm').querySelector('[data-testid^="sched-net-pdm-startstop-"]')
    expect(line).not.toBeNull()
    expect(line!.textContent).toMatch(/\d+\/\d+ → \d+\/\d+/)
  })

  it('showDuration=false：搭接标签与工期格一起收掉（节点时间开关不动）', () => {
    const p = styled({ showDuration: false })
    render(<PdmView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const root = screen.getByTestId('sched-pdm')
    expect(root.querySelectorAll('text.sched-net-label')).toHaveLength(0)
    const node = root.querySelector('.sched-pdm-node:not(.sched-pdm-virtual)')!
    expect(node.querySelectorAll('.sched-pdm-grid span')).toHaveLength(5) // 缺的正是工期格
  })

  it('dotStyle=rect 出方角类；lineWidth/headHeight 显式值落到连线与箭头', () => {
    const plain = styled(undefined)
    const { unmount } = render(<PdmView project={plain} cpm={computeCpm(plain.tasks, plain.links)} />)
    expect(screen.getByTestId('sched-pdm').querySelector('.sched-net-pdm-rect')).toBeNull()
    unmount()

    const p = styled({ dotStyle: 'rect', lineWidth: 2.5, headHeight: 12 })
    render(<PdmView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const root = screen.getByTestId('sched-pdm')
    expect(root.querySelector('.sched-net-pdm-rect')).not.toBeNull()
    const link = root.querySelector<SVGPathElement>('path.sched-net-link:not(.sched-net-link-virtual)')!
    expect(link.style.strokeWidth).toBe('2.5')
    const marker = root.querySelector('marker#pdm-arrow')!
    expect(marker.getAttribute('markerWidth')).toBe('12')
    expect(marker.getAttribute('refX')).toBe('11')
  })
})

// ── W4 收口：布图参数（density/namePos/hideUnimportantLink）在单代号视图的消费 ──
//
// 单代号没有 buildAoa 的层行网格，density 直接调 COL_GAP/ROW_GAP 比例；
// 缺省（系数 1 / namePos auto / hide false）必须与「不落 networkStyle」逐字一致。

describe('PdmView 布图参数消费（W4 收口）', () => {
  const styled = (style?: SchedNetworkStyle): SchedProject => {
    const p = crossingProject()
    if (style) p.networkStyle = style
    return p
  }
  /** 渲染一次并交出 root + 卸载器 */
  const renderPdm = (p: SchedProject) => {
    const r = render(<PdmView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    return { root: screen.getByTestId('sched-pdm'), unmount: r.unmount }
  }
  /** 节点 [left,top] 序列（DOM 顺序稳定=leaves 顺序） */
  const nodePos = (root: HTMLElement): string[][] =>
    Array.from(root.querySelectorAll<HTMLElement>('.sched-pdm-node:not(.sched-pdm-virtual)')).map((el) => [el.style.left, el.style.top])

  it('缺省零回归锚点：不落 networkStyle vs 落全缺省布图参数 —— 渲染输出逐字一致', () => {
    const a = renderPdm(styled(undefined))
    const plain = a.root.innerHTML
    const plainPos = nodePos(a.root)
    a.unmount()
    const b = renderPdm(styled({ density: 'normal', namePos: 'auto', hideUnimportantLink: false }))
    expect(b.root.innerHTML).toBe(plain)
    expect(nodePos(b.root)).toEqual(plainPos)
    // 缺省不出「隐藏提示」、名称不加排布类
    expect(screen.queryByTestId('sched-pdm-hide-hint')).toBeNull()
    expect(b.root.querySelector('.sched-net-name-vertical, .sched-net-name-horizontal')).toBeNull()
    b.unmount()
  })

  it('density=compact/loose：列距/行距只调比例（compact 收紧、loose 放宽），节点数不变', () => {
    const normal = renderPdm(styled(undefined))
    const basePos = nodePos(normal.root)
    normal.unmount()
    const compact = renderPdm(styled({ density: 'compact' }))
    const compactPos = nodePos(compact.root)
    compact.unmount()
    const loose = renderPdm(styled({ density: 'loose' }))
    const loosePos = nodePos(loose.root)
    loose.unmount()
    expect(compactPos.length).toBe(basePos.length) // 拓扑/节点数不变
    const maxLeft = (rows: string[][]) => Math.max(...rows.map((r) => parseFloat(r[0])))
    const maxTop = (rows: string[][]) => Math.max(...rows.map((r) => parseFloat(r[1])))
    expect(maxLeft(compactPos)).toBeLessThan(maxLeft(basePos))
    expect(maxLeft(loosePos)).toBeGreaterThan(maxLeft(basePos))
    expect(maxTop(compactPos)).toBeLessThanOrEqual(maxTop(basePos))
    expect(maxTop(loosePos)).toBeGreaterThanOrEqual(maxTop(basePos))
  })

  it('namePos=vertical/horizontal：节点名称加排布类（auto 两者都不加）', () => {
    const vert = renderPdm(styled({ namePos: 'vertical' }))
    expect(vert.root.querySelectorAll('.sched-pdm-name.sched-net-name-vertical').length).toBeGreaterThan(0)
    expect(vert.root.querySelector('.sched-net-name-horizontal')).toBeNull()
    vert.unmount()
    const horiz = renderPdm(styled({ namePos: 'horizontal' }))
    expect(horiz.root.querySelectorAll('.sched-pdm-name.sched-net-name-horizontal').length).toBeGreaterThan(0)
    expect(horiz.root.querySelector('.sched-net-name-vertical')).toBeNull()
    horiz.unmount()
  })

  it('hideUnimportantLink=true：非关键搭接整条不画，虚拟起点/终点桩保留并给图例提示', () => {
    const p = styled(undefined)
    const cpm = computeCpm(p.tasks, p.links)
    const hidden = p.links.filter((l) => {
      const f = cpm.rows[l.from]
      const t = cpm.rows[l.to]
      return !(f && t && f.critical && t.critical && isLinkBinding(l.type, l.lag, f, t))
    }).length
    const auto = renderPdm(p)
    const before = {
      real: auto.root.querySelectorAll('path.sched-net-link:not(.sched-net-link-virtual)').length,
      virt: auto.root.querySelectorAll('path.sched-net-link-virtual').length,
    }
    expect(before.virt).toBeGreaterThan(0) // 多起点 ⇒ 有虚拟桩
    auto.unmount()

    const hide = renderPdm(styled({ hideUnimportantLink: true }))
    const after = {
      real: hide.root.querySelectorAll('path.sched-net-link:not(.sched-net-link-virtual)').length,
      virt: hide.root.querySelectorAll('path.sched-net-link-virtual').length,
    }
    expect(hidden).toBeGreaterThan(0) // 本图确有非关键搭接
    expect(after.real).toBe(before.real - hidden) // 只少非关键搭接
    expect(after.virt).toBe(before.virt) // 虚拟桩是拓扑兜底，不受开关影响
    expect(screen.getByTestId('sched-pdm-hide-hint').textContent).toContain(`已隐藏 ${hidden} 条`)
    hide.unmount()
  })
})

// ── W6 收尾：`nodeRadius` 在单代号视图的消费（尺度系数口径）───────────────
//
// 单代号节点是矩形盒，所以 `nodeRadius` 不是绝对 px 而是尺度系数 k = radius/16：
// 盒宽高 / 列距 / 行距 / 连线端点全部同源缩放（networkLayout.resolvePdmNodeMetrics）。
// 缺省 16 ⇒ k=1 ⇒ 必须与「不落 networkStyle」逐字一致；非缺省按 k 等比，
// 且任何半径下净间距恒为正（不重叠）。
describe('PdmView 节点半径消费（W6 收尾）', () => {
  const styled = (style?: SchedNetworkStyle): SchedProject => {
    const p = crossingProject()
    if (style) p.networkStyle = style
    return p
  }
  const renderPdm = (p: SchedProject) => {
    const r = render(<PdmView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    return { root: screen.getByTestId('sched-pdm'), unmount: r.unmount }
  }
  const leafNodes = (root: HTMLElement): HTMLElement[] =>
    Array.from(root.querySelectorAll<HTMLElement>('.sched-pdm-node:not(.sched-pdm-virtual)'))
  const virtualStart = (root: HTMLElement): HTMLElement =>
    root.querySelector<HTMLElement>('[data-testid="sched-pdm-virtual-start"]')!
  const canvas = (root: HTMLElement): HTMLElement => root.querySelector<HTMLElement>('.sched-network-canvas')!
  const px = (v: string): number => Number.parseFloat(v)
  /** 首条实搭接（C→D）的起点 x —— 就是源节点右缘 = left + 盒宽（连线端点锚点） */
  const firstLinkX = (root: HTMLElement): number => {
    const path = root.querySelector<SVGPathElement>('path.sched-net-link:not(.sched-net-link-virtual)')!
    const m = (path.getAttribute('d') ?? '').match(/^M\s*(-?[\d.]+)/)!
    return px(m[1])
  }
  /** 递增后的相邻差值（列 x / 行 y），用于「净间距为正」判定 */
  const uniqDiffs = (values: number[]): number[] => {
    const xs = [...new Set(values)].sort((a, b) => a - b)
    return xs.slice(1).map((v, i) => v - xs[i])
  }

  it('缺省零回归锚点：不落 networkStyle vs 落 nodeRadius:16 —— 渲染输出逐字一致', () => {
    const a = renderPdm(styled(undefined))
    const plain = a.root.innerHTML
    a.unmount()
    const b = renderPdm(styled({ nodeRadius: 16 }))
    expect(b.root.innerHTML).toBe(plain)
    // k=1：盒宽/虚拟桩尺寸=现行基准；叶子**不内联高度**（沿用 CSS 内容撑高）
    const leaf = leafNodes(b.root)[0]
    expect(px(leaf.style.width)).toBe(150)
    expect(leaf.style.height).toBe('')
    expect(px(virtualStart(b.root).style.width)).toBe(150)
    expect(px(virtualStart(b.root).style.height)).toBe(92)
    b.unmount()
  })

  it('nodeRadius=20（k=1.25）：盒宽高与虚拟桩同比放大，连线端点跟着盒边同系数', () => {
    const base = renderPdm(styled(undefined))
    const baseLeaf = leafNodes(base.root)
    const baseLinkX = firstLinkX(base.root)
    const baseCanvasW = px(canvas(base.root).style.width)
    base.unmount()

    const p = renderPdm(styled({ nodeRadius: 20 }))
    const leaf = leafNodes(p.root)
    expect(px(leaf[0].style.width)).toBeCloseTo(187.5) // 150 × 1.25
    expect(px(leaf[0].style.height)).toBeCloseTo(115) // 92 × 1.25
    expect(px(virtualStart(p.root).style.height)).toBeCloseTo(115)
    // 端点 = 源节点（C，leaves 第 3 个）右缘：left + 187.5
    const c = leaf[2]
    expect(firstLinkX(p.root)).toBeCloseTo(px(c.style.left) + 187.5)
    // 与缺省同比：端点相对左缘的位移 187.5/150=1.25
    expect(firstLinkX(p.root) - px(c.style.left)).toBeCloseTo((baseLinkX - px(baseLeaf[2].style.left)) * 1.25)
    // 画布宽 = 边距(不缩放, 56) + 4 段 × (盒宽 187.5 + 列距 92.5)：盒与间距同系数，边距是常量
    expect(px(canvas(p.root).style.width)).toBeCloseTo(56 + 4 * (187.5 + 92.5))
    expect(px(canvas(p.root).style.width)).toBeGreaterThan(baseCanvasW)
    p.unmount()
  })

  it('nodeRadius=12（k=0.75）：盒宽高收缩，列/行净间距仍为正（不重叠）', () => {
    const p = renderPdm(styled({ nodeRadius: 12 }))
    const leaf = leafNodes(p.root)
    expect(px(leaf[0].style.width)).toBeCloseTo(112.5) // 150 × 0.75
    expect(px(leaf[0].style.height)).toBeCloseTo(69) // 92 × 0.75
    const colPitch = uniqDiffs(leaf.map((el) => px(el.style.left)))
    const rowPitch = uniqDiffs(leaf.map((el) => px(el.style.top)))
    expect(colPitch.length).toBeGreaterThan(0)
    expect(rowPitch.length).toBeGreaterThan(0)
    for (const d of colPitch) expect(d).toBeGreaterThan(112.5) // 净列距 = 差 − 盒宽 > 0
    for (const d of rowPitch) expect(d).toBeGreaterThan(69) // 净行距 = 差 − 盒高 > 0
    p.unmount()
  })

  it('半径下限 4（k=0.25）：尺寸与间距都仍为正，不出现负间距', () => {
    const p = renderPdm(styled({ nodeRadius: 4 }))
    const leaf = leafNodes(p.root)
    expect(px(leaf[0].style.width)).toBeCloseTo(37.5)
    expect(px(leaf[0].style.height)).toBeCloseTo(23)
    for (const d of uniqDiffs(leaf.map((el) => px(el.style.left)))) expect(d).toBeGreaterThan(37.5)
    for (const d of uniqDiffs(leaf.map((el) => px(el.style.top)))) expect(d).toBeGreaterThan(23)
    p.unmount()
  })

  it('尺度线性：20 与 12 的盒宽比等于半径比（20/12），不是各写一套缺省', () => {
    const big = renderPdm(styled({ nodeRadius: 20 }))
    const bigW = px(leafNodes(big.root)[0].style.width)
    big.unmount()
    const small = renderPdm(styled({ nodeRadius: 12 }))
    const smallW = px(leafNodes(small.root)[0].style.width)
    small.unmount()
    expect(bigW / smallW).toBeCloseTo(20 / 12)
  })
})
