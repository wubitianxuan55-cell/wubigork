// dcma.ts — 6.4 计划质量体检：DCMA 14 点纯函数（前端镜像口径，对标 Deltek Acumen Fuse）。
// Why：进度计划的质量此前只有「能否排程」（CPM ok）与基线漂移展示，没有
// 「排得健不健壮」的体检——缺逻辑、负搭接、高浮时、长工期、关键线路断点等
// 结构性问题无人提示。
// How：纯函数无副作用——dcmaAudit(project, cpm, dataDate) 逐点计算
// 实测值 vs DCMA 阈值；数据不够的点诚实标 n/a（缺基线/缺数据日期/未启用
// 资源维度），绝不伪造可评估假象。口径与标准 14 点的差异逐点写进 note。

import type { CpmResult, SchedProject, SchedTask } from './types'
import defaultThresholds from './dcma-thresholds.json'

// v4.226 阈值表数据资产（规范知识出内核红线）：DCMA 阈值是行业标准值，可被
// 工作区覆盖文件 .gaea/skills/schedule-dcma/thresholds.json 部分覆盖（DcmaView
// 经 ReadFile 懒加载传入）——改阈值不发版；计算逻辑是机制，留码。
export interface DcmaThresholds {
  /** 高浮时判定（工作日） */
  highFloatDays: number
  /** 长工期判定（任务自身单位） */
  highDurationDays: number
  /** 缺逻辑/正搭接/高浮/长工期/错过基线的超标占比上限（%） */
  ratioLimitPct: number
  /** FS 关系占比下限（%） */
  fsRatioMinPct: number
  /** CPLI 通过下限 */
  cpliMin: number
  /** BEI 通过下限 */
  beiMin: number
}

const DEFAULT_THRESHOLDS: DcmaThresholds = { ...defaultThresholds }

/** 兼容导出：默认阈值常量（历史消费方；判定一律走 dcmaAudit 的 th 参数）。 */
export const DCMA_HIGH_FLOAT_DAYS = DEFAULT_THRESHOLDS.highFloatDays
export const DCMA_HIGH_DURATION_DAYS = DEFAULT_THRESHOLDS.highDurationDays
export const DCMA_RATIO_LIMIT = DEFAULT_THRESHOLDS.ratioLimitPct
export const DCMA_FS_RATIO_MIN = DEFAULT_THRESHOLDS.fsRatioMinPct
export const DCMA_CPLI_MIN = DEFAULT_THRESHOLDS.cpliMin
export const DCMA_BEI_MIN = DEFAULT_THRESHOLDS.beiMin

/** 体检单点结论。na 非空 = 不可评估（数据不足，诚实口径）。 */
export interface DcmaCheck {
  id: number
  name: string
  na?: string
  /** 实测值：占比类为百分数（0-100），计数类为个数，比率为小数 */
  value: number | null
  /** DCMA 阈值口径（展示用，如 "≤5%"） */
  threshold: string
  pass?: boolean
  /** 超标样本（任务名/关系描述，上限 5 条） */
  offenders: string[]
  /** 本刀实现口径与标准 14 点的差异说明 */
  note?: string
}

export interface DcmaReport {
  checks: DcmaCheck[]
  /** 可评估项数 */
  evaluable: number
  passed: number
  failed: number
  /** 总分 = 可评估项通过率（0-100）；无可评估项为 null */
  score: number | null
}

const OFFENDER_LIMIT = 5

function check(id: number, name: string, threshold: string): DcmaCheck {
  return { id, name, threshold, value: null, offenders: [] }
}

function pct(part: number, whole: number): number {
  if (whole <= 0) return 0
  return Math.round((part / whole) * 10000) / 100
}

/** 体检范围：叶任务（level>0，非分组行）。多数检查排除里程碑。 */
function leafTasks(project: SchedProject): SchedTask[] {
  return project.tasks.filter((t) => t.level > 0)
}
function leafNoMilestone(project: SchedProject): SchedTask[] {
  return leafTasks(project).filter((t) => !t.isMilestone)
}

/** 关键线路连通段数：对 critical 任务子图做连通分量计数（经由搭接边双向）。 */
function criticalSegments(project: SchedProject, cpm: CpmResult): { segments: number; breaks: string[] } {
  const crit = project.tasks.filter((t) => t.level > 0 && cpm.rows[t.id]?.critical)
  if (crit.length === 0) return { segments: 0, breaks: [] }
  const critIds = new Set(crit.map((t) => t.id))
  const adj = new Map<string, string[]>()
  for (const l of project.links) {
    if (!critIds.has(l.from) || !critIds.has(l.to)) continue
    adj.set(l.from, [...(adj.get(l.from) ?? []), l.to])
    adj.set(l.to, [...(adj.get(l.to) ?? []), l.from])
  }
  const seen = new Set<string>()
  let segments = 0
  const roots: string[] = []
  for (const t of crit) {
    if (seen.has(t.id)) continue
    segments++
    roots.push(t.name)
    const stack = [t.id]
    while (stack.length) {
      const cur = stack.pop()!
      if (seen.has(cur)) continue
      seen.add(cur)
      for (const nb of adj.get(cur) ?? []) if (!seen.has(nb)) stack.push(nb)
    }
  }
  return { segments, breaks: segments > 1 ? roots.slice(1) : [] }
}

/**
 * DCMA 14 点体检。dataDate = 数据日期（工作日序号，相对开工日；缺省/null =
 * 未提供实际执行口径 → 11/13/14 按未开工或 n/a 诚实处理）。
 * thresholds = 阈值覆盖（v4.226 数据资产：部分字段可覆盖，缺省用内置默认表）。
 */
export function dcmaAudit(
  project: SchedProject,
  cpm: CpmResult,
  dataDate?: number | null,
  thresholds?: Partial<DcmaThresholds>,
): DcmaReport {
  const th: DcmaThresholds = { ...DEFAULT_THRESHOLDS, ...(thresholds ?? {}) }
  const checks: DcmaCheck[] = []
  const leaves = leafTasks(project)
  const leavesNm = leafNoMilestone(project)
  const leafIds = new Set(leaves.map((t) => t.id))
  const nameOf = (id: string) => project.tasks.find((t) => t.id === id)?.name ?? id
  const rowOf = (id: string) => cpm.rows[id]

  // ── 1 缺逻辑 Logic（≤5%）：叶任务（非里程碑）无前驱且无后继；
  // 开工首任务与竣工末任务豁免（项目级起止，标准口径允许）。
  {
    const c = check(1, '缺逻辑任务', `≤${th.ratioLimitPct}%`)
    if (!cpm.ok) c.na = 'CPM 未通过（循环依赖），先修复排程'
    else {
      const hasPred = new Set<string>()
      const hasSucc = new Set<string>()
      for (const l of project.links) {
        if (leafIds.has(l.from)) hasSucc.add(l.from)
        if (leafIds.has(l.to)) hasPred.add(l.to)
      }
      let esMin = Infinity
      let efMax = -1
      for (const t of leavesNm) {
        const r = rowOf(t.id)
        if (!r) continue
        if (r.es < esMin) esMin = r.es
        if (r.ef > efMax) efMax = r.ef
      }
      const offenders: string[] = []
      let bad = 0
      for (const t of leavesNm) {
        const r = rowOf(t.id)
        if (!r) continue
        const noPred = !hasPred.has(t.id)
        const noSucc = !hasSucc.has(t.id)
        if (!noPred && !noSucc) continue
        // 首末豁免只对「单侧缺失」生效：项目起点（无前驱且最早）与终点
        // （无后继且最晚）；孤立任务（双侧皆无）不得因 es 并列被豁免。
        const isFirst = noPred && !noSucc && r.es === esMin
        const isLast = noSucc && !noPred && r.ef === efMax
        if (!(isFirst || isLast)) {
          bad++
          if (offenders.length < OFFENDER_LIMIT) offenders.push(t.name)
        }
      }
      c.value = pct(bad, leavesNm.length)
      c.offenders = offenders
      c.pass = c.value <= th.ratioLimitPct
    }
    checks.push(c)
  }

  // ── 2 Leads 负搭接（=0%）
  {
    const c = check(2, '负搭接（Leads）', '=0%')
    const bad = project.links.filter((l) => l.lag < 0)
    c.value = pct(bad.length, project.links.length)
    c.offenders = bad.slice(0, OFFENDER_LIMIT).map((l) => `${nameOf(l.from)} → ${nameOf(l.to)}（${l.type} lag ${l.lag}）`)
    c.pass = bad.length === 0
    checks.push(c)
  }

  // ── 3 Lags 正搭接（<5%）
  {
    const c = check(3, '正搭接（Lags）', `<${th.ratioLimitPct}%`)
    const bad = project.links.filter((l) => l.lag > 0)
    c.value = pct(bad.length, project.links.length)
    c.offenders = bad.slice(0, OFFENDER_LIMIT).map((l) => `${nameOf(l.from)} → ${nameOf(l.to)}（${l.type} lag +${l.lag}）`)
    c.pass = c.value < th.ratioLimitPct
    checks.push(c)
  }

  // ── 4 FS 关系占比（≥90%）
  {
    const c = check(4, 'FS 关系占比', `≥${th.fsRatioMinPct}%`)
    const fs = project.links.filter((l) => l.type === 'FS')
    c.value = project.links.length === 0 ? 100 : pct(fs.length, project.links.length)
    c.offenders = project.links
      .filter((l) => l.type !== 'FS')
      .slice(0, OFFENDER_LIMIT)
      .map((l) => `${nameOf(l.from)} → ${nameOf(l.to)}（${l.type}）`)
    c.pass = c.value >= th.fsRatioMinPct
    c.note = '无搭接关系时按 100%（不误伤小计划）'
    checks.push(c)
  }

  // ── 5 硬约束（<5%）：当前模型无约束字段，恒 0（manual 锁定开始是排程模式
  // 而非日期约束，不计）。
  {
    const c = check(5, '硬约束', `<${th.ratioLimitPct}%`)
    c.value = 0
    c.pass = true
    c.note = '当前计划模型无日期约束字段，恒通过'
    checks.push(c)
  }

  // ── 6 高浮时 High Float（<5%）：叶任务 TF>默认 44（th.highFloatDays）工作日。
  {
    const c = check(6, '高浮时任务', `<${th.ratioLimitPct}%（>${th.highFloatDays} 工作日）`)
    if (!cpm.ok) c.na = 'CPM 未通过'
    else {
      const bad = leavesNm.filter((t) => (rowOf(t.id)?.tf ?? 0) > th.highFloatDays)
      c.value = pct(bad.length, leavesNm.length)
      c.offenders = bad
        .slice(0, OFFENDER_LIMIT)
        .map((t) => `${t.name}（TF ${rowOf(t.id)?.tf}）`)
      c.pass = c.value < th.ratioLimitPct
    }
    checks.push(c)
  }

  // ── 7 负浮时 Negative Float（=0）
  {
    const c = check(7, '负浮时任务', '=0')
    if (!cpm.ok) c.na = 'CPM 未通过'
    else {
      const bad = leavesNm.filter((t) => (rowOf(t.id)?.tf ?? 0) < 0)
      c.value = bad.length
      c.offenders = bad
        .slice(0, OFFENDER_LIMIT)
        .map((t) => `${t.name}（TF ${rowOf(t.id)?.tf}）`)
      c.pass = bad.length === 0
    }
    checks.push(c)
  }

  // ── 8 长工期 High Duration（<5%）：>默认 44（th.highDurationDays，按任务自身单位口径）。
  {
    const c = check(8, '长工期任务', `<${th.ratioLimitPct}%（>${th.highDurationDays}）`)
    const bad = leavesNm.filter((t) => t.duration > th.highDurationDays)
    c.value = pct(bad.length, leavesNm.length)
    c.offenders = bad.slice(0, OFFENDER_LIMIT).map((t) => `${t.name}（${t.duration}${t.durationUnit === 'cd' ? 'cd' : 'wd'}）`)
    c.pass = c.value < th.ratioLimitPct
    checks.push(c)
  }

  // ── 9 无效日期（=0）：工作日序号口径下 ES<0 或 EF<ES。
  {
    const c = check(9, '无效日期', '=0')
    if (!cpm.ok) c.na = 'CPM 未通过'
    else {
      const bad = leavesNm.filter((t) => {
        const r = rowOf(t.id)
        return !!r && (r.es < 0 || r.ef < r.es)
      })
      c.value = bad.length
      c.offenders = bad.slice(0, OFFENDER_LIMIT).map((t) => t.name)
      c.pass = bad.length === 0
      c.note = '工作日序号口径：开工前开始（ES<0）或完成早于开始'
    }
    checks.push(c)
  }

  // ── 10 资源缺失（目标 0 缺失）：启用资源维度才有意义。
  {
    const c = check(10, '资源缺失任务', '=0')
    if (!project.resources || project.resources.length === 0) {
      c.na = '未启用资源维度（无资源表）'
    } else {
      const assigned = new Set((project.assignments ?? []).map((a) => a.taskId))
      const bad = leavesNm.filter((t) => !assigned.has(t.id))
      c.value = pct(bad.length, leavesNm.length)
      c.offenders = bad.slice(0, OFFENDER_LIMIT).map((t) => t.name)
      c.pass = c.value === 0
    }
    checks.push(c)
  }

  // ── 11 错过基线 Missed Tasks（≤5%）：当前 EF 晚于基线 EF 且未完成。
  {
    const c = check(11, '错过基线任务', `≤${th.ratioLimitPct}%`)
    if (!project.baseline) c.na = '未保存基线'
    else if (!cpm.ok) c.na = 'CPM 未通过'
    else {
      const bad = leavesNm.filter((t) => {
        const base = project.baseline?.rows[t.id]
        const r = rowOf(t.id)
        if (!base || !r) return false
        return r.ef > base.ef && t.progress < 100
      })
      c.value = pct(bad.length, leavesNm.length)
      c.offenders = bad.slice(0, OFFENDER_LIMIT).map((t) => t.name)
      c.pass = c.value <= th.ratioLimitPct
    }
    checks.push(c)
  }

  // ── 12 关键路径测试（连续性）：关键任务子图应为一个连通段。
  {
    const c = check(12, '关键路径连续性', '=1 段')
    if (!cpm.ok) c.na = 'CPM 未通过'
    else {
      const { segments, breaks } = criticalSegments(project, cpm)
      c.value = segments
      c.offenders = breaks.slice(0, OFFENDER_LIMIT).map((n) => `独立段起点：${n}`)
      c.pass = segments === 1
      c.note = '手动任务不参与关键线路（排程模式语义），可能造成分段'
    }
    checks.push(c)
  }

  // ── 13 CPLI（≥0.95）：CPLI = (PD − 执行至) / (PF − 执行至)，需数据日期+基线。
  {
    const c = check(13, 'CPLI 关键路径长度指数', `≥${th.cpliMin}`)
    if (!project.baseline) c.na = '未保存基线'
    else if (dataDate == null) c.na = '未提供数据日期'
    else {
      const pd = project.baseline.duration
      const pf = cpm.duration
      const asOf = dataDate
      if (pf - asOf <= 0) c.na = '数据日期已达当前竣工，无法计算'
      else {
        const v = (pd - asOf) / (pf - asOf)
        c.value = Math.round(v * 1000) / 1000
        c.pass = c.value >= th.cpliMin
      }
      c.note = 'CPLI=(基线工期−执行至)/(当前预测工期−执行至)，工作日口径'
    }
    checks.push(c)
  }

  // ── 14 BEI（≥0.95）：完成任务数 / 数据日期应完成（基线 EF≤数据日期）数。
  {
    const c = check(14, 'BEI 基线执行指数', `≥${th.beiMin}`)
    if (!project.baseline) c.na = '未保存基线'
    else if (dataDate == null) c.na = '未提供数据日期'
    else {
      const scheduled = leavesNm.filter((t) => (project.baseline?.rows[t.id]?.ef ?? Infinity) <= dataDate)
      if (scheduled.length === 0) c.na = '数据日期前无应完成任务（未开工口径）'
      else {
        const done = scheduled.filter((t) => t.progress >= 100).length
        c.value = Math.round((done / scheduled.length) * 1000) / 1000
        c.pass = c.value >= th.beiMin
      }
      c.note = '完成口径=progress≥100（本模型无实际日期字段），诚实近似'
    }
    checks.push(c)
  }

  const evaluable = checks.filter((c) => !c.na)
  const passed = evaluable.filter((c) => c.pass).length
  // CPM 未通过时整体分不可信（排程本身断裂），score 置 null——逐点结论仍在。
  const score = !cpm.ok || evaluable.length === 0 ? null : Math.round((passed / evaluable.length) * 100)
  return { checks, evaluable: evaluable.length, passed, failed: evaluable.length - passed, score }
}
