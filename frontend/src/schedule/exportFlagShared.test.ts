/**
 * exportFlagShared.test.ts — FE7-06 反向钉子：横道（ganttExport）/双代号
 * （networkExport.buildAoaExportSvg）/单代号（networkExport.buildPdmExportSvg）
 * 三个上报图面的里程碑旗标 expFlagSvg 输出必须逐字节同形。
 *
 * 钉法：从原始 SVG 字符串切出 <g class="sched-exp-flag" …>…</g> 整段：
 * ①掩码（数字→N）后三图面互等、且等于冻结字面量——任何非数字字节改动
 * （类名/属性名/引号/斜杠/属性顺序）当场红；②对每面另做字节敏感断言
 * （stroke-width="1.5"、data-exp-flag="1"、关键红）——纯数字字节改动
 * （1.5→1.6、关键红换色）也红。改共享函数一字节即红，FE7-06「副本漂移」
 * 的反向证据位。
 */
import { describe, expect, it } from 'vitest'
import { buildGanttExportSvg, EXP_COLORS } from './ganttExport'
import { buildAoaExportSvg, buildPdmExportSvg } from './networkExport'
import { buildAoa } from './aoa'
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

/** 里程碑旗标整段（原始字节，不经 DOM 序列化）：图面里恰好一面 */
function flagOf(svg: string): string {
  const hits = svg.match(/<g class="sched-exp-flag"[^>]*>.*?<\/g>/g) ?? []
  expect(hits.length, '恰好一面里程碑旗标').toBe(1)
  return hits[0] as string
}

/** 数字掩码：坐标/数值→N（颜色 hex 里的数字同样掩码，色值另用未掩码断言钉） */
function masked(svg: string): string {
  return svg.replace(/-?\d+(?:\.\d+)?/g, 'N')
}

/** 旗标掩码冻结字面量：掩码后三图面必须与之逐字节一致（属性名内数字同样被掩） */
const FROZEN_MASKED =
  '<g class="sched-exp-flag" data-exp-flag="N"><line xN="N" yN="N" xN="N" yN="N" stroke="#dcN" stroke-width="N"/><polygon points="N,N N,N N,N" fill="#dcN"/></g>'

const ganttProj = proj([t('G', 0, 0), t('D', 0, 1, { isMilestone: true })], [])
const netProj = proj([t('A', 3), t('D', 0, 1, { isMilestone: true })], [{ from: 'A', to: 'D' }])

describe('里程碑旗标 expFlagSvg 三图面字节同形（FE7-06 反向钉子）', () => {
  const ganttSvg = buildGanttExportSvg(ganttProj, computeCpm(ganttProj.tasks, ganttProj.links)).svg
  const aoaGraph = buildAoa(netProj.tasks, netProj.links)
  expect(aoaGraph.ok, 'AOA 图可构建').toBe(true)
  const aoaSvg = buildAoaExportSvg(netProj, aoaGraph).svg
  const pdmSvg = buildPdmExportSvg(netProj, computeCpm(netProj.tasks, netProj.links)).svg

  it('三面旗标掩码后与冻结字面量逐字节一致（形状字节全钉）', () => {
    expect(masked(flagOf(ganttSvg)), '横道旗掩码').toBe(FROZEN_MASKED)
    expect(masked(flagOf(aoaSvg)), '双代号旗掩码').toBe(FROZEN_MASKED)
    expect(masked(flagOf(pdmSvg)), '单代号旗掩码').toBe(FROZEN_MASKED)
  })

  it('三面旗标互相掩码等值（共享单源漂移即红）', () => {
    const g = masked(flagOf(ganttSvg))
    expect(masked(flagOf(aoaSvg)), '双代号≡横道').toBe(g)
    expect(masked(flagOf(pdmSvg)), '单代号≡横道').toBe(g)
  })

  it('字节敏感断言：stroke-width 1.5 / data-exp-flag="1" / 关键红（数字字节改动即红）', () => {
    for (const [name, svg] of [['横道', ganttSvg], ['双代号', aoaSvg], ['单代号', pdmSvg]] as const) {
      const flag = flagOf(svg)
      expect(flag, `${name}旗 stroke-width 字节`).toContain('stroke-width="1.5"')
      expect(flag, `${name}旗 data-exp-flag 字节`).toContain('data-exp-flag="1"')
      expect(flag, `${name}旗 关键红`).toContain(EXP_COLORS.critical)
      expect(flag, `${name}旗 类名字节`).toContain('class="sched-exp-flag"')
      expect(flag, `${name}旗 线段属性名字节`).toContain('<line x1="')
    }
  })
})
