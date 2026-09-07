// opsSim.ts — schedule_apply ops 通道的 TS 投影模拟器（diff 确认闭环刀D）。
//
// **Go internal/schedule/ops.go ApplyOps 的逐语义 TS 镜像**：审批卡在落盘前
// 投影「这批 ops 应用后会得到什么」。漂移 = 「批的是 A，落的是 B」——设计
// 文档标注的最大风险，对冲主阵地 = Go/TS 对拍单测（ops_golden.fixture.json
// 由 internal/schedule/ops_golden_test.go 生成并校验，双端同吃一份口径）。
//
// 语义要点（全部对齐 Go，含零值语义）：
// - 顺序应用、单条失败即中止（fail-closed，不做部分应用）——模拟在深拷贝上
//   跑，失败时调用方只取 error，半途状态不外泄（工具层同口径：先拷贝再应用）。
// - 指针三态：JSON 缺 key = 不动；显式值（含 0/空串）= 改。patch.mode 空串
//   在 Go 里也是 no-op（*pt.Mode != "" 额外守卫）。
// - upsert_task 缺字段按 Go 零值语义补齐（缺 level=0 分组行、缺 progress=0）。
// - set_baseline 需 savedAt（引擎纯函数不取时钟）；CPM 用无 planFinish 口径。
//
// 错误文案与 Go 尽量一致但不参与对拍（对拍只比「是否失败」与 after 状态）。

import { computeCpm } from './cpm'
import { snapshotBaseline } from './baseline'
import { normalizeCalendar } from './calendar'
import type { LinkType, SchedAssignment, SchedCalendar, SchedLink, SchedProject, SchedResource, SchedTask } from './types'

export interface SimOp {
  type: string
  afterId?: string
  task?: Partial<SchedTask> & { id?: string }
  patch?: {
    name?: string; level?: number; duration?: number; progress?: number
    isMilestone?: boolean; mode?: string; manualStart?: number; fixedCost?: number
  }
  id?: string
  toId?: string
  links?: Array<{ from: string; type?: string; lag?: number }>
  name?: string
  startDate?: string
  calendar?: SchedCalendar
  deadline?: string | null
  baselineName?: string
  savedAt?: string
  resource?: Partial<SchedResource> & { id?: string }
  resourcePatch?: { name?: string; type?: string; unit?: string; standardRate?: number; costPerUse?: number; maxUnits?: number }
  taskId?: string
  assignments?: Array<Partial<SchedAssignment> & { resourceId?: string }>
}

export type SimResult =
  | { ok: true; project: SchedProject; summary: string[] }
  | { ok: false; error: string; summary: string[] }

const FS = 'FS'

const badMoney = (v: number) => v < 0 || Number.isNaN(v) || !Number.isFinite(v)

const idxTask = (tasks: SchedTask[], id: string) => tasks.findIndex((t) => t.id === id)
const idxRes = (rs: SchedResource[], id: string) => rs.findIndex((r) => r.id === id)

/** remove_task 的子孙集：扁平数组 level 语义（第 idx 行 + 紧随其 level 更深的
 *  连续行），与 Go descendantIDs / 前端 ganttGroup 同口径。 */
function descendantIdsFlat(tasks: SchedTask[], idx: number): Set<string> {
  const out = new Set<string>()
  if (idx < 0 || idx >= tasks.length) return out
  const base = tasks[idx].level
  out.add(tasks[idx].id)
  for (let i = idx + 1; i < tasks.length && tasks[i].level > base; i++) out.add(tasks[i].id)
  return out
}

/** upsert_task 的零值语义补齐（Go JSON unmarshal：缺 duration=0、缺 level=0
 *  分组行、缺 progress=0；isMilestone/mode/manualStart/fixedCost omitempty）。 */
function materializeTask(t: Partial<SchedTask> & { id?: string }): SchedTask {
  return {
    id: t.id ?? '',
    name: t.name ?? '',
    duration: t.duration ?? 0,
    level: t.level ?? 0,
    progress: t.progress ?? 0,
    ...(t.isMilestone !== undefined ? { isMilestone: t.isMilestone } : {}),
    ...(t.mode !== undefined ? { mode: t.mode } : {}),
    ...(t.manualStart !== undefined ? { manualStart: t.manualStart } : {}),
    ...(t.fixedCost !== undefined ? { fixedCost: t.fixedCost } : {}),
    ...(t.custom !== undefined ? { custom: t.custom } : {}),
  }
}

function applyOne(p: SchedProject, op: SimOp): string {
  switch (op.type) {
    case 'upsert_task': {
      if (!op.task) throw new Error('upsert_task 缺少 task')
      const t = materializeTask(op.task)
      if (!t.id.trim()) throw new Error('upsert_task 缺少任务 id')
      if (t.level !== 0 && t.level !== 1) throw new Error(`任务 ${t.id} 层级非法（仅 0/1）：${t.level}`)
      if (t.progress < 0 || t.progress > 100) throw new Error(`任务 ${t.id} 进度超出 0-100`)
      if (t.level === 0 && t.fixedCost !== undefined && t.fixedCost !== 0) {
        throw new Error(`分组行 ${t.id} 禁止固定成本（汇总唯一口径为子孙求和）`)
      }
      if (t.fixedCost !== undefined && badMoney(t.fixedCost)) {
        throw new Error(`任务 ${t.id} 固定成本非法（须为非负有限数）：${t.fixedCost}`)
      }
      let at = p.tasks.length
      if (op.afterId) {
        const idx = idxTask(p.tasks, op.afterId)
        if (idx < 0) throw new Error(`afterId 不存在：${op.afterId}`)
        at = idx + 1
      }
      const exist = idxTask(p.tasks, t.id)
      if (exist >= 0) {
        p.tasks[exist] = t
        return `更新任务「${t.name}」`
      }
      p.tasks.splice(at, 0, t)
      return `新增任务「${t.name}」（工期 ${t.duration} 工作日）`
    }

    case 'patch_task': {
      const idx = idxTask(p.tasks, op.id ?? '')
      if (idx < 0) throw new Error(`任务不存在：${op.id}`)
      const t = p.tasks[idx]
      const changes: string[] = []
      if (op.patch) {
        const pt = op.patch
        if (pt.name !== undefined) { t.name = pt.name; changes.push(`改名「${pt.name}」`) }
        if (pt.duration !== undefined) {
          if (pt.duration < 0) throw new Error('工期为负')
          changes.push(`工期 ${t.duration}→${pt.duration} 工作日`)
          t.duration = pt.duration
        }
        if (pt.progress !== undefined) {
          if (pt.progress < 0 || pt.progress > 100) throw new Error('进度超出 0-100')
          changes.push(`进度 ${t.progress}→${pt.progress}%`)
          t.progress = pt.progress
        }
        if (pt.level !== undefined) {
          if (pt.level !== 0 && pt.level !== 1) throw new Error('层级非法（仅 0/1）')
          t.level = pt.level
          changes.push(`层级→${pt.level}`)
        }
        if (pt.isMilestone !== undefined) {
          t.isMilestone = pt.isMilestone
          if (pt.isMilestone) changes.push('设为里程碑')
        }
        if (pt.mode !== undefined && pt.mode !== '') {
          if (pt.mode !== 'auto' && pt.mode !== 'manual') throw new Error(`模式非法：${pt.mode}`)
          t.mode = pt.mode
          changes.push(`模式→${pt.mode}`)
        }
        if (pt.manualStart !== undefined) {
          t.manualStart = pt.manualStart
          changes.push(`锁定开始→第 ${pt.manualStart} 工作日`)
        }
        if (pt.fixedCost !== undefined) {
          if (badMoney(pt.fixedCost)) throw new Error(`固定成本非法（须为非负有限数）：${pt.fixedCost}`)
          if (t.level === 0) throw new Error(`分组行 ${t.id} 禁止固定成本（汇总唯一口径为子孙求和）`)
          changes.push(`固定成本 ${t.fixedCost}→${pt.fixedCost} 元`)
          t.fixedCost = pt.fixedCost
        }
      }
      if (changes.length === 0) throw new Error('patch_task 未提供任何字段')
      return `调整「${t.name}」：${changes.join('、')}`
    }

    case 'remove_task': {
      const idx = idxTask(p.tasks, op.id ?? '')
      if (idx < 0) throw new Error(`任务不存在：${op.id}`)
      const kill = descendantIdsFlat(p.tasks, idx)
      const name = p.tasks[idx].name
      p.tasks = p.tasks.filter((t) => !kill.has(t.id))
      p.links = p.links.filter((l) => !kill.has(l.from) && !kill.has(l.to))
      return `移除「${name}」及子孙共 ${kill.size} 行`
    }

    case 'set_links': {
      const idx = idxTask(p.tasks, op.toId ?? '')
      if (idx < 0) throw new Error(`任务不存在：${op.toId}`)
      const toName = p.tasks[idx].name
      const toId = op.toId ?? ''
      const kept: SchedLink[] = p.links.filter((l) => l.to !== toId)
      const byId = new Set(p.tasks.map((t) => t.id))
      for (const nl of op.links ?? []) {
        if (nl.from === toId) throw new Error('任务不能以自身为前置')
        if (!byId.has(nl.from)) throw new Error(`前置任务不存在：${nl.from}`)
        kept.push({ from: nl.from, to: toId, type: (nl.type || FS) as LinkType, lag: nl.lag ?? 0 })
      }
      p.links = kept
      return `更新「${toName}」前置关系（共 ${(op.links ?? []).length} 条）`
    }

    case 'set_meta': {
      const changes: string[] = []
      if (op.name) { p.name = op.name; changes.push(`工程名→「${op.name}」`) }
      if (op.startDate) {
        if (op.startDate.length !== 10) throw new Error(`开工日期口径应为 YYYY-MM-DD：${op.startDate}`)
        p.startDate = op.startDate
        changes.push(`开工日期→${op.startDate}`)
      }
      if (op.calendar !== undefined && op.calendar !== null) {
        p.calendar = normalizeCalendar(op.calendar)
        changes.push('更新工作日历')
      }
      if (op.deadline !== undefined && op.deadline !== null) {
        const d = op.deadline.trim()
        if (d !== '') {
          if (d.length !== 10) throw new Error(`目标竣工日期口径应为 YYYY-MM-DD：${d}`)
          p.deadline = d
          changes.push(`目标竣工→${d}`)
        } else {
          p.deadline = ''
          changes.push('清除目标竣工')
        }
      }
      if (changes.length === 0) throw new Error('set_meta 未提供任何字段')
      return changes.join('、')
    }

    case 'auto_chain': {
      // 推荐逻辑关系的确定性缺省：仅对【无前置】叶任务按 WBS 顺序补 FS 串联
      // （首个叶不补）；手动任务不补也不作链源。added==0 报错（Go 同语义）。
      const hasIn = new Set(p.links.map((l) => l.to))
      let added = 0
      let lastLeaf = ''
      for (const t of p.tasks) {
        if (t.level === 0) continue
        if (t.mode === 'manual') continue
        if (lastLeaf !== '' && !hasIn.has(t.id)) {
          p.links.push({ from: lastLeaf, to: t.id, type: FS, lag: 0 })
          added++
        }
        lastLeaf = t.id
      }
      if (added === 0) throw new Error('auto_chain 没有可补的任务（无前置的叶任务不足两个，或均已手动定位）')
      return `自动补全 ${added} 条缺省串联逻辑（仅无前置任务，FS lag=0）`
    }

    case 'set_baseline': {
      // 基线快照：固化此刻排程（ops 序列中的当前位置）。savedAt 必须由调用方
      // 标注；CPM 无 planFinish 口径（Go ComputeCpm 同款）；无叶任务/环拒绝。
      if (!op.savedAt?.trim()) throw new Error('set_baseline 缺少 savedAt（YYYY-MM-DD HH:mm，由工具层标注）')
      const cpm = computeCpm(p.tasks, p.links)
      const r = snapshotBaseline(p, cpm, op.savedAt, op.baselineName)
      if (!r.ok) throw new Error(r.error)
      p.baseline = r.baseline
      return `保存基线「${r.baseline.name}」（${Object.keys(r.baseline.rows).length} 项工作，总工期 ${r.baseline.duration} 天）`
    }

    case 'clear_baseline': {
      if (!p.baseline) throw new Error('当前没有基线可清除')
      p.baseline = undefined
      return '清除基线'
    }

    case 'upsert_resource': {
      const r = op.resource
      if (!r) throw new Error('upsert_resource 缺少 resource')
      if (!r.id?.trim()) throw new Error('upsert_resource 缺少资源 id')
      if (r.type !== 'work' && r.type !== 'material' && r.type !== 'cost') {
        throw new Error(`资源 ${r.id}：资源类型非法（work|material|cost）：${r.type}`)
      }
      for (const [label, v] of [['标准费率', r.standardRate], ['每次使用成本', r.costPerUse], ['可用上限', r.maxUnits]] as const) {
        if (v !== undefined && badMoney(v)) throw new Error(`资源 ${r.id} ${label} 非法（须为非负有限数）：${v}`)
      }
      const full: SchedResource = {
        id: r.id, name: r.name ?? '', type: r.type,
        ...(r.unit !== undefined ? { unit: r.unit } : {}),
        ...(r.standardRate !== undefined ? { standardRate: r.standardRate } : {}),
        ...(r.costPerUse !== undefined ? { costPerUse: r.costPerUse } : {}),
        ...(r.maxUnits !== undefined ? { maxUnits: r.maxUnits } : {}),
      }
      const i = idxRes(p.resources ??= [], r.id)
      if (i >= 0) { p.resources[i] = full; return `更新资源「${full.name}」（${full.type}）` }
      p.resources.push(full)
      return `新增资源「${full.name}」（${full.type}）`
    }

    case 'patch_resource': {
      p.resources ??= []
      const idx = idxRes(p.resources, op.id ?? '')
      if (idx < 0) throw new Error(`资源不存在：${op.id}`)
      const r = p.resources[idx]
      const changes: string[] = []
      if (op.resourcePatch) {
        const pr = op.resourcePatch
        if (pr.name !== undefined) { r.name = pr.name; changes.push(`改名「${pr.name}」`) }
        if (pr.type !== undefined) {
          if (pr.type !== 'work' && pr.type !== 'material' && pr.type !== 'cost') {
            throw new Error(`资源类型非法（work|material|cost）：${pr.type}`)
          }
          r.type = pr.type
          changes.push(`类型→${pr.type}`)
        }
        if (pr.unit !== undefined) {
          r.unit = pr.unit
          changes.push(pr.unit === '' ? '清除计量单位' : `计量单位→${pr.unit}`)
        }
        if (pr.standardRate !== undefined) {
          if (badMoney(pr.standardRate)) throw new Error(`标准费率非法（须为非负有限数）：${pr.standardRate}`)
          changes.push(`标准费率 ${r.standardRate}→${pr.standardRate} ${r.type === 'material' ? '元/单位' : r.type === 'cost' ? '元' : '元/工日'}`)
          r.standardRate = pr.standardRate
        }
        if (pr.costPerUse !== undefined) {
          if (badMoney(pr.costPerUse)) throw new Error(`每次使用成本非法（须为非负有限数）：${pr.costPerUse}`)
          changes.push(`每次使用成本 ${r.costPerUse}→${pr.costPerUse} 元`)
          r.costPerUse = pr.costPerUse
        }
        if (pr.maxUnits !== undefined) {
          if (badMoney(pr.maxUnits)) throw new Error(`可用上限非法（须为非负有限数）：${pr.maxUnits}`)
          changes.push(`可用上限 ${r.maxUnits}→${pr.maxUnits}`)
          r.maxUnits = pr.maxUnits
        }
      }
      if (changes.length === 0) throw new Error('patch_resource 未提供任何字段')
      return `调整资源「${r.name}」：${changes.join('、')}`
    }

    case 'remove_resource': {
      p.resources ??= []
      const idx = idxRes(p.resources, op.id ?? '')
      if (idx < 0) throw new Error(`资源不存在：${op.id}`)
      const name = p.resources[idx].name
      p.resources.splice(idx, 1)
      const before = (p.assignments ??= []).length
      p.assignments = p.assignments.filter((a) => a.resourceId !== op.id)
      const cascaded = before - p.assignments.length
      return cascaded > 0 ? `移除资源「${name}」及其 ${cascaded} 条分配` : `移除资源「${name}」（无分配）`
    }

    case 'set_assignments': {
      p.assignments ??= []
      const idx = idxTask(p.tasks, op.taskId ?? '')
      if (idx < 0) throw new Error(`任务不存在：${op.taskId}`)
      if (p.tasks[idx].level === 0) throw new Error(`分组行 ${op.taskId} 禁止挂分配（汇总唯一口径为子孙求和）`)
      const taskName = p.tasks[idx].name
      const taskId = op.taskId ?? ''
      const pairSeen = new Set<string>()
      for (const a of op.assignments ?? []) {
        if (a.taskId && a.taskId !== taskId) throw new Error(`set_assignments 只允许目标任务 ${taskId} 的分配，出现 ${a.taskId}`)
        if (idxRes(p.resources ??= [], a.resourceId ?? '') < 0) throw new Error(`资源不存在：${a.resourceId}`)
        const pair = `${taskId}\u0000${a.resourceId}`
        if (pairSeen.has(pair)) throw new Error(`分配重复（任务 ${taskId} ↔ 资源 ${a.resourceId}）`)
        pairSeen.add(pair)
      }
      // 只换该任务的分配集，其他任务原样保留；TaskID 强制归一（Go 同语义）。
      const kept = p.assignments.filter((a) => a.taskId !== taskId)
      for (const a of op.assignments ?? []) {
        kept.push({ taskId, resourceId: a.resourceId ?? '', ...(a.units !== undefined ? { units: a.units } : {}), ...(a.quantity !== undefined ? { quantity: a.quantity } : {}), ...(a.amount !== undefined ? { amount: a.amount } : {}) })
      }
      p.assignments = kept
      return `更新「${taskName}」资源分配（共 ${(op.assignments ?? []).length} 条）`
    }

    default:
      throw new Error(`不支持的操作类型：${op.type}`)
  }
}

/**
 * simulateOps：在深拷贝上顺序应用 ops（fail-closed，单条失败即中止且半途
 * 状态不外泄）。after 侧供审批卡投影；真正的落盘权威永远在 Go 侧
 * schedule_apply（模拟失败诚实降级为意图清单）。
 */
export function simulateOps(before: SchedProject, ops: SimOp[]): SimResult {
  const p: SchedProject = JSON.parse(JSON.stringify(before))
  p.resources ??= []
  p.assignments ??= []
  const summary: string[] = []
  try {
    for (let i = 0; i < ops.length; i++) {
      try {
        summary.push(applyOne(p, ops[i]))
      } catch (e) {
        throw new Error(`第 ${i + 1} 条操作失败：${e instanceof Error ? e.message : String(e)}`)
      }
    }
  } catch (e) {
    return { ok: false, error: e instanceof Error ? e.message : String(e), summary }
  }
  return { ok: true, project: p, summary }
}
