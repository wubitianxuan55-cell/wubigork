/**
 * ganttExport.test.ts — 横道图上报图面构建器（刀D1 / v4.137.0 刀D4 增
 * 条尾标注与里程碑旗标用例）
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

describe('条尾标注与里程碑旗标（v4.137.0 刀D4）', () => {
  it('条尾标注：叶任务条末端 +6px 标「任务名（N天）」，摘要/里程碑不标注', () => {
    const cpm = computeCpm(baseProj.tasks, baseProj.links)
    const { svg } = buildGanttExportSvg(baseProj, cpm)
    const doc = parse(svg)
    const labels = Array.from(doc.querySelectorAll('.sched-exp-bar-label')).map((n) => n.textContent ?? '')
    expect(labels, '实体条 B/C/F 各一条').toHaveLength(3)
    for (const s of ['任务B（3天）', '任务C（5天）', '任务F（2天）']) expect(labels, s).toContain(s)
    expect(labels.some((s) => s.includes('任务D')), '里程碑走旗标不标注').toBe(false)
    expect(labels.some((s) => s.includes('任务G')), '分组摘要条不标注').toBe(false)
    // 位置口径：标注 x = 条末端（dayNo(ef)×14）+ 6
    const xOf = (wd: number): number =>
      Math.round((wdToDate(START, wd, baseProj.calendar).getTime() - new Date(`${START}T00:00:00Z`).getTime()) / 86400000) * 14
    const cLabel = Array.from(doc.querySelectorAll('.sched-exp-bar-label')).find((n) => n.textContent === '任务C（5天）')
    expect(cLabel, 'C 标注存在').toBeTruthy()
    expect(cLabel!.getAttribute('x'), '条末端 +6px').toBe(String(xOf(cpm.rows.C!.ef) + 6))
  })

  it('条尾标注开关：meta.barLabels=false 不出标注；缺省（不传）=开启（向后兼容）', () => {
    const cpm = computeCpm(baseProj.tasks, baseProj.links)
    const off = buildGanttExportSvg(baseProj, cpm, { barLabels: false })
    expect(off.svg, '关闭后无标注元素').not.toContain('sched-exp-bar-label')
    expect(off.svg, '关闭后无「（N天）」文本').not.toContain('（3天）')
    const on = buildGanttExportSvg(baseProj, cpm)
    expect(on.svg, '不传 meta 仍带标注').toContain('任务B（3天）')
  })

  it('右界回退：条尾空间不足改画条内右端白字、超长名 fitText 截断', () => {
    // 单任务 14 工作日（跨两个周末=18 自然日）→ 画布 days=max(21,21)=21 天、
    // 右界 294px：条末端 252+6 放不下标注 → 回退条内右端（白字右对齐）
    const p = proj([t('A', 14)], [])
    p.tasks[0].name = '超'.repeat(60)
    const cpm = computeCpm(p.tasks, p.links)
    const { svg } = buildGanttExportSvg(p, cpm)
    const doc = parse(svg)
    const label = doc.querySelector('.sched-exp-bar-label')
    expect(label, '回退条内右端').toBeTruthy()
    expect(label!.getAttribute('text-anchor'), '右端对齐').toBe('end')
    expect(label!.getAttribute('fill'), '条内白字').toBe('white')
    expect(label!.textContent, '超长名截断加省略号').toContain('…')
  })

  it('里程碑旗标：菱形右侧一面小旗（data-exp-flag、关键红），非里程碑无', () => {
    const cpm = computeCpm(baseProj.tasks, baseProj.links)
    const { svg } = buildGanttExportSvg(baseProj, cpm)
    const doc = parse(svg)
    const flags = doc.querySelectorAll('[data-exp-flag]')
    expect(flags.length, '仅里程碑 D 一面').toBe(1)
    expect(flags[0].querySelector('line'), '旗杆竖线').toBeTruthy()
    const flag = flags[0].querySelector('polygon')
    expect(flag, '三角旗面').toBeTruthy()
    expect(flag!.getAttribute('fill'), '关键红').toBe(EXP_COLORS.critical)
    // 旗在菱形右侧 +12px（菱形外接半径 6.5，points 首点横坐标 = cx-6.5）
    const mile = doc.querySelector('.sched-exp-mile')!
    const cx = Number(mile.getAttribute('points')!.split(',')[0]) + 6.5
    expect(Number(flags[0].querySelector('line')!.getAttribute('x1')), '旗杆贴菱形右侧').toBe(cx + 12)
  })
})
