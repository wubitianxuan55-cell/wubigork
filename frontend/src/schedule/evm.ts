// evm.ts — 6.6 跨域 EVM：进度×造价 挣值三数（PV/EV/AC）+ 两指数（SPI/CPI）。
// Why：进度与造价同在本机一个库是 gaea 独有跨域——基线计划×任务预算×完成率
// 可以直接算挣值，无需云协同。AC（实际成本）本地没有录入面，来源=手工录入，
// 缺失时 CPI 诚实 n/a（绝不伪造）。
// How：纯函数无副作用——
//   PV = Σ 任务预算成本 × 基线时间分摊比例（数据日期落在 [bES,bEF] 内线性，
//        里程碑在 bEF 整额兑现）
//   EV = Σ 任务预算成本 × 完成率（progress/100）
//   AC = 手工录入（元）；缺失 → CPI/CV = null
//   SPI = EV/PV；CPI = EV/AC；SV = EV−PV；CV = EV−AC
// 口径注全：每个数字都带「它从哪来、怎么摊」的一句话说明（6.6 出口判据）。

import type { CpmResult, SchedProject } from './types'
import { computeCosts } from './cost'

export interface EvmNumber {
  value: number | null
  /** 口径注：这个数从哪来、怎么算的（展示直出） */
  note: string
}

export interface EvmResult {
  /** 基线+成本数据是否齐（false 时三数缺省，note 说明缺什么） */
  ok: boolean
  error?: string
  pv: EvmNumber
  ev: EvmNumber
  ac: EvmNumber
  spi: EvmNumber
  cpi: EvmNumber
  /** 进度偏差/成本偏差（元，正=好） */
  sv: number | null
  cv: number | null
  /** EV 贡献 Top（挣值来源 transparency，上限 5） */
  evTop: { id: string; name: string; earned: number }[]
  /** 完工预算 BAC（元）= 当前任务预算合计 */
  bac: number
}

export interface EvmOptions {
  /** 数据日期（工作日序号，相对开工日） */
  dataDate: number
  /** 实际已发生成本（元，手工录入；缺省=未录入，CPI/CV n/a） */
  actualCost?: number | null
}

const clamp01 = (v: number) => Math.max(0, Math.min(1, v))

const NOTES = {
  pv: 'PV 计划价值：任务预算成本按【基线】起止线性分摊到数据日期（里程碑在基线完成点整额兑现）',
  ev: 'EV 挣值：任务预算成本 × 完成率（progress）',
  ac: 'AC 实际成本：来源=手工录入（实际已发生，如已付进度款）；本机无自动实际成本面',
  spi: 'SPI 进度绩效指数 = EV / PV（≥1 进度超前，<1 落后）',
  cpi: 'CPI 成本绩效指数 = EV / AC（需先录入实际成本）',
}

export function computeEvm(
  project: SchedProject,
  cpm: CpmResult,
  opts: EvmOptions,
): EvmResult {
  const empty = (error?: string): EvmResult => ({
    ok: false,
    error,
    pv: { value: null, note: NOTES.pv },
    ev: { value: null, note: NOTES.ev },
    ac: { value: opts.actualCost ?? null, note: NOTES.ac },
    spi: { value: null, note: NOTES.spi },
    cpi: { value: null, note: NOTES.cpi },
    sv: null,
    cv: null,
    evTop: [],
    bac: 0,
  })
  if (!project.baseline) return empty('未保存基线：PV 依赖基线计划分摊，请先在基线面板设置基线')
  if (!cpm.ok) return empty('CPM 未通过（循环依赖），成本与挣值 fail-closed')
  const costs = computeCosts(project, cpm)
  if (!costs.ok) return empty('成本核算未通过')

  const dd = opts.dataDate
  let pv = 0
  let ev = 0
  const evRows: { id: string; name: string; earned: number }[] = []
  for (const t of project.tasks) {
    if (t.level <= 0) continue
    const cost = costs.rows[t.id]?.total ?? 0
    if (cost <= 0) continue
    const base = project.baseline.rows[t.id]
    // PV：基线分摊（无基线行的任务不计入 PV——不在基线口径内）。
    if (base) {
      const dur = base.ef - base.es
      const frac = dur <= 0 ? (dd >= base.ef ? 1 : 0) : clamp01((dd - base.es) / dur)
      pv += cost * frac
    }
    // EV：完成率 × 预算。
    const earned = cost * (t.progress / 100)
    if (earned > 0) evRows.push({ id: t.id, name: t.name, earned: Math.round(earned * 100) / 100 })
    ev += earned
  }
  evRows.sort((a, b) => b.earned - a.earned)

  pv = Math.round(pv * 100) / 100
  ev = Math.round(ev * 100) / 100
  const ac = opts.actualCost ?? null
  const spi = pv > 0 ? Math.round((ev / pv) * 1000) / 1000 : null
  const cpi = ac && ac > 0 ? Math.round((ev / ac) * 1000) / 1000 : null

  return {
    ok: true,
    pv: { value: pv, note: NOTES.pv },
    ev: { value: ev, note: NOTES.ev },
    ac: { value: ac, note: NOTES.ac },
    spi: { value: spi, note: NOTES.spi },
    cpi: { value: cpi, note: NOTES.cpi },
    sv: spi === null ? null : Math.round((ev - pv) * 100) / 100,
    cv: ac && ac > 0 ? Math.round((ev - ac) * 100) / 100 : null,
    evTop: evRows.slice(0, 5),
    bac: Math.round(costs.total * 100) / 100,
  }
}
