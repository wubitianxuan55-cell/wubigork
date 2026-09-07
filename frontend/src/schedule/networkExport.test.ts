/**
 * networkExport.test.ts — 网络图上报图面构建器（刀D2）
 *
 * 用例口径：DOMParser 解析 SVG 做结构断言；AOA 用 buildAoa 真引擎输出，
 * 断言时标几何（节点 x=留白+es×日宽）、波形=自由时差（虚工作尾段含 q 波形）、
 * 工程标尺四行与总工期红刻度；PDM 断言六格盒/虚拟 S/T/绑定红链/无时标。
 * 循环依赖 fail-closed。
 */
import { describe, expect, it } from 'vitest'
import { buildAoa } from './aoa'
import { buildAoaExportSvg, buildPdmExportSvg } from './networkExport'
import { EXP_COLORS } from './ganttExport'
import { computeCpm } from './cpm'
import type { SchedProject, SchedTask } from './types'

function t(id: string, duration: number, level = 1, extra: Partial<SchedTask> = {}): SchedTask {
  return { id, name: `任务${id}`, duration, level, progress: 0, ...extra }
}

function proj(tasks: SchedTask[], links: { from: string; to: string }[]): SchedProject {
  return {
    name: '雨污分流改造工程',
    startDate: '2026-09-07',
    tasks,
    links: links.map((l) => ({ ...l, type: 'FS' as const, lag: 0 })),
  }
}

/** A(3)→B(5)→D(里程碑) 链 + 独立 C(2)：总工期 8，C 有 6 天总时差 */
const baseProj = proj([t('A', 3), t('B', 5), t('D', 0, 1, { isMilestone: true }), t('C', 2)], [
  { from: 'A', to: 'B' },
  { from: 'B', to: 'D' },
])

function parse(svg: string): Document {
  return new DOMParser().parseFromString(svg, 'image/svg+xml')
}

describe('buildAoaExportSvg 双代号时标', () => {
  it('节点圈=事件数、箭线=引擎边数、关键 4 条（链 A/B/D+虚工作）、虚工作 2 条', () => {
    const graph = buildAoa(baseProj.tasks, baseProj.links)
    expect(graph.ok).toBe(true)
    const { svg } = buildAoaExportSvg(baseProj, graph)
    const doc = parse(svg)
    expect(doc.querySelectorAll('.sched-exp-aoa-node').length, '事件圈').toBe(graph.nodes.length)
    expect(doc.querySelectorAll('path.sched-exp-aoa').length, '箭线').toBe(graph.edges.length)
    expect(doc.querySelectorAll('.sched-exp-aoa-critical').length, '关键箭线').toBe(4)
    expect(doc.querySelectorAll('path.sched-exp-aoa-dummy').length, '虚工作').toBe(2)
  })

  it('时标几何：起点事件 x=左留白、es=2 的事件 x=留白+2×日宽（自适应 110 上限）', () => {
    const graph = buildAoa(baseProj.tasks, baseProj.links)
    const { svg } = buildAoaExportSvg(baseProj, graph)
    const doc = parse(svg)
    const circles = Array.from(doc.querySelectorAll('.sched-exp-aoa-node')).map((c) => Number(c.getAttribute('cx')))
    expect(circles).toContain(40) // 起点 S：留白 40 + es 0
    expect(circles.filter((x) => x === 40 + 2 * 110).length, 'es=2 的事件（n3 与…）').toBeGreaterThanOrEqual(1)
  })

  it('波形线=自由时差：独立任务 C 的完成事件虚工作尾段画波形（d 含 q 段），关键链无波形', () => {
    const graph = buildAoa(baseProj.tasks, baseProj.links)
    const { svg } = buildAoaExportSvg(baseProj, graph)
    const doc = parse(svg)
    const waves = Array.from(doc.querySelectorAll('path.sched-exp-aoa-dummy')).filter((p) => (p.getAttribute('d') ?? '').includes(' q '))
    expect(waves.length, 'C 完成事件→T 虚工作带波形').toBe(1)
    const critWithWave = Array.from(doc.querySelectorAll('path.sched-exp-aoa-critical')).filter((p) => (p.getAttribute('d') ?? '').includes(' q '))
    expect(critWithWave.length, '关键箭线不画波形').toBe(0)
  })

  it('工程标尺四行：工程日/月/日/星期齐备，总工期 8 红刻度，缺省图名带「双代号时标网络图」', () => {
    const graph = buildAoa(baseProj.tasks, baseProj.links)
    const { svg } = buildAoaExportSvg(baseProj, graph)
    const doc = parse(svg)
    const texts = Array.from(doc.querySelectorAll('text')).map((n) => n.textContent ?? '')
    for (const s of ['工程日', '月', '日', '星期', '2026.9']) {
      expect(texts, s).toContain(s)
    }
    expect(svg, '缺省图名').toContain('双代号时标网络图')
    // 总工期刻度（文本「8」且红色 fill=#dc2626）
    const totalTick = Array.from(doc.querySelectorAll('text')).find((n) => n.textContent === '8' && n.getAttribute('fill') === EXP_COLORS.critical)
    expect(totalTick, '总工期红刻度').toBeTruthy()
    expect(doc.querySelectorAll('.sched-exp-sign rect').length, '图签三格').toBe(3)
  })

  it('循环依赖 fail-closed 抛错；长计划日宽压窄（200 天→59px/天）', () => {
    const bad = proj([t('A', 1), t('B', 1)], [{ from: 'A', to: 'B' }, { from: 'B', to: 'A' }])
    const g = buildAoa(bad.tasks, bad.links)
    expect(g.ok).toBe(false)
    expect(() => buildAoaExportSvg(bad, g)).toThrow(/循环依赖/)
    // 200 天计划：dayW = floor(12000/202) = 59
    const longTasks: SchedTask[] = [t('A', 200)]
    const longGraph = buildAoa(longTasks, [])
    const long = buildAoaExportSvg(proj(longTasks, []), longGraph)
    expect(long.w).toBe(40 + 201 * 59 + 20)
  })
})

describe('buildPdmExportSvg 单代号', () => {
  it('六格盒=叶任务数 + 虚拟 S/T（双起点双终点）+ 绑定红链 + 无时标标尺', () => {
    const cpm = computeCpm(baseProj.tasks, baseProj.links)
    const { svg } = buildPdmExportSvg(baseProj, cpm)
    const doc = parse(svg)
    // A、C 双起点；D、C 双终点 → 虚拟 S/T 各一
    expect(doc.querySelectorAll('.sched-exp-pdm-node-virtual').length, '虚拟 S/T').toBe(2)
    expect(doc.querySelectorAll('.sched-exp-pdm-node').length, '节点盒（含虚拟）').toBe(6)
    expect(doc.querySelectorAll('.sched-exp-pdm-link-critical').length, '绑定红链 A→B/B→D').toBe(2)
    const texts = Array.from(doc.querySelectorAll('text')).map((n) => n.textContent ?? '')
    for (const s of ['起点 S', '完成 T', 'FS']) expect(texts, s).toContain(s)
    expect(texts, '无工程标尺').not.toContain('工程日')
  })

  it('缺省图名带「单代号网络图」；空计划抛错', () => {
    const cpm = computeCpm(baseProj.tasks, baseProj.links)
    const { svg } = buildPdmExportSvg(baseProj, cpm)
    expect(svg).toContain('单代号网络图')
    expect(() => buildPdmExportSvg(proj([], []), computeCpm([], []))).toThrow(/暂无任务/)
  })
})
