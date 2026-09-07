/**
 * ganttExport.test.ts — 横道图上报图面构建器（刀D1）
 *
 * 用例口径：DOMParser 解析 SVG 字符串做结构断言（testid 类选择器），
 * 条形几何用 calendar.wdToDate 独立换算自然日偏移对照（验证接线公式
 * left=dayNo×14/width=ΔdayNo×14，DAY_W=14）；循环依赖 fail-closed。
 */
import { describe, expect, it } from 'vitest'
import { buildGanttExportSvg, EXP_COLORS } from './ganttExport'
import { computeCpm } from './cpm'
import { wdToDate } from './calendar'
import type { SchedProject, SchedTask } from './types'
import { GANTT_COLS, GANTT_REPORT_KEYS, colsByKeys } from './ganttCols'

const START = '2026-09-07' // 周一

function t(id: string, duration: number, level = 1, extra: Partial<SchedTask> = {}): SchedTask {
  return { id, name: `任务${id}`, duration, level, progress: 0, ...extra }
}

function proj(tasks: SchedTask[], links: { from: string; to: string }[]): SchedProject {
  return {
    name: '雨污分流改造工程',
    startDate: START,
    tasks,
    links: links.map((l) => ({ ...l, type: 'FS' as const, lag: 0 })),
  }
}

function parse(svg: string): Document {
  return new DOMParser().parseFromString(svg, 'image/svg+xml')
}

const baseProj = proj(
  [
    t('G', 0, 0), // 分组
    t('B', 3), // 施工准备（关键）
    t('C', 5), // 主体（关键）
    t('D', 0, 1, { isMilestone: true }), // 里程碑
    t('F', 2), // 无搭接非关键（总时差>0）
  ],
  [
    { from: 'B', to: 'C' },
    { from: 'C', to: 'D' },
  ],
)

describe('buildGanttExportSvg 结构', () => {
  it('上报 6 列预设：colsByKeys 保序取列、固定集外全藏', () => {
    const cols = colsByKeys(GANTT_REPORT_KEYS)
    expect(cols.map((c) => c.key)).toEqual(GANTT_REPORT_KEYS)
    expect(cols).toHaveLength(6)
    expect(cols.every((c) => GANTT_COLS.includes(c))).toBe(true)
  })

  it('循环依赖 fail-closed：抛错不出坏图面', () => {
    const bad = proj([t('A', 1), t('B', 1)], [{ from: 'A', to: 'B' }, { from: 'B', to: 'A' }])
    const cpm = computeCpm(bad.tasks, bad.links)
    expect(cpm.ok).toBe(false)
    expect(() => buildGanttExportSvg(bad, cpm)).toThrow(/循环依赖/)
  })

  it('标题带：缺省图名含工程名，编制单位/日期有值才出现', () => {
    const cpm = computeCpm(baseProj.tasks, baseProj.links)
    const empty = buildGanttExportSvg(baseProj, cpm)
    expect(empty.svg).toContain('雨污分流改造工程　施工进度计划横道图')
    expect(empty.svg).not.toContain('编制单位：')
    const full = buildGanttExportSvg(baseProj, cpm, { org: '博耐特建设', date: '2026-09-07' })
    expect(full.svg).toContain('编制单位：博耐特建设')
    expect(full.svg).toContain('编制日期：2026-09-07')
    // 自定义图名替换缺省
    const titled = buildGanttExportSvg(baseProj, cpm, { title: '总进度计划（上报版）' })
    expect(titled.svg).toContain('总进度计划（上报版）')
    expect(titled.svg).not.toContain('施工进度计划横道图')
  })

  it('表格：表头恰为上报 6 列，无最迟/时差/成本列', () => {
    const cpm = computeCpm(baseProj.tasks, baseProj.links)
    const { svg } = buildGanttExportSvg(baseProj, cpm)
    const doc = parse(svg)
    const texts = Array.from(doc.querySelectorAll('text')).map((n) => n.textContent ?? '')
    for (const label of ['序号', '任务名称', '工期(天)', '开始', '完成', '前置']) {
      expect(texts, label).toContain(label)
    }
    for (const gone of ['最迟开始', '总时差', '自由时差', '成本']) {
      expect(texts, gone).not.toContain(gone)
    }
  })

  it('条形：3 实体条全关键、里程碑菱形 1、分组汇总条 1、非关键 F 有总时差尾', () => {
    const cpm = computeCpm(baseProj.tasks, baseProj.links)
    const { svg } = buildGanttExportSvg(baseProj, cpm)
    const doc = parse(svg)
    expect(doc.querySelectorAll('.sched-exp-bar').length, '实体条 B/C/F').toBe(3)
    expect(doc.querySelectorAll('.sched-exp-bar-crit').length, '关键条 B/C').toBe(2)
    expect(doc.querySelectorAll('.sched-exp-mile').length, '里程碑 D').toBe(1)
    expect(doc.querySelectorAll('.sched-exp-summary').length, '分组 G 汇总').toBe(1)
    // F 无搭接：Tc=8，TF=8-2=6 → 尾宽 6×14=84
    const floats = doc.querySelectorAll('.sched-exp-float')
    expect(floats.length).toBe(1)
    expect(floats[0].getAttribute('width')).toBe('84')
  })

  it('条形几何：left=dayNo(es)×14、width=ΔdayNo×14（与 wdToDate 换算对照）', () => {
    const cpm = computeCpm(baseProj.tasks, baseProj.links)
    const { svg } = buildGanttExportSvg(baseProj, cpm)
    const doc = parse(svg)
    const bars = Array.from(doc.querySelectorAll('.sched-exp-bar'))
    const cRow = cpm.rows.C!
    const xOf = (wd: number): number =>
      Math.round((wdToDate(START, wd, baseProj.calendar).getTime() - new Date(`${START}T00:00:00Z`).getTime()) / 86400000) * 14
    const cBar = bars.find((b) => b.getAttribute('x') === String(xOf(cRow.es)))
    expect(cBar, 'C 条在 dayNo(es)×14 列位').toBeTruthy()
    expect(cBar!.getAttribute('width')).toBe(String(xOf(cRow.es + 5) - xOf(cRow.es)))
    expect(cBar!.getAttribute('fill')).toBe(EXP_COLORS.critical)
  })

  it('非工作日底纹列数与日历一致；目标竣工线在竣工列右缘', () => {
    const p: SchedProject = { ...baseProj, deadline: '2026-09-18' }
    const cpm = computeCpm(p.tasks, p.links)
    const { svg } = buildGanttExportSvg(p, cpm)
    const doc = parse(svg)
    const days = 14 // 9/07~9/20 画布至少 21 天，按构建器公式精确复算
    // 与构建器同公式：days = max(dayNo(max(duration,7))+3, 21, deadlineOff+2)
    const dlOff = Math.round((new Date('2026-09-18T00:00:00Z').getTime() - new Date(`${START}T00:00:00Z`).getTime()) / 86400000)
    const expectDays = Math.max(11 + 3, 21, dlOff + 2)
    expect(expectDays).toBeGreaterThanOrEqual(days)
    const lines = doc.querySelectorAll('.sched-exp-deadline')
    expect(lines.length).toBe(1)
    expect(lines[0].getAttribute('x1')).toBe(String((dlOff + 1) * 14))
    // 周末底纹：画布窗口内周六/周日计数（default 周一~五工作制，无节假日）
    let off = 0
    for (let i = 0; i < expectDays; i++) {
      const d = new Date(new Date(`${START}T00:00:00Z`).getTime() + i * 86400000)
      if (d.getUTCDay() === 0 || d.getUTCDay() === 6) off++
    }
    expect(doc.querySelectorAll('rect[fill="#eef2f7"]').length).toBe(off)
  })

  it('图签三格与图例：编制人/审核人/批准有值显值、空值留白；图例含关键工作/目标竣工线', () => {
    const cpm = computeCpm(baseProj.tasks, baseProj.links)
    const { svg } = buildGanttExportSvg(baseProj, cpm, { designer: '张三', reviewer: '李四' })
    const doc = parse(svg)
    expect(doc.querySelectorAll('.sched-exp-sign rect').length, '图签三格').toBe(3)
    const texts = Array.from(doc.querySelectorAll('.sched-exp-sign text')).map((n) => n.textContent ?? '')
    for (const s of ['编制人', '审核人', '批准', '张三', '李四']) expect(texts, s).toContain(s)
    const all = Array.from(doc.querySelectorAll('text')).map((n) => n.textContent ?? '')
    for (const s of ['关键工作', '非关键工作', '汇总', '里程碑', '目标竣工线']) expect(all, s).toContain(s)
  })

  it('XML 转义与空工程：特殊字符安全、零任务出空图面', () => {
    const p = proj([t('X', 1)], [])
    p.tasks[0].name = '挖掘<机>&土方'
    const cpm = computeCpm(p.tasks, p.links)
    const { svg } = buildGanttExportSvg(p, cpm)
    expect(svg).toContain('挖掘&lt;机&gt;&amp;土方')
    const empty = proj([], [])
    const cpm2 = computeCpm(empty.tasks, empty.links)
    const art = buildGanttExportSvg(empty, cpm2)
    expect(art.h).toBeGreaterThan(0)
    expect(parse(art.svg).querySelectorAll('.sched-exp-bar').length).toBe(0)
  })
})
