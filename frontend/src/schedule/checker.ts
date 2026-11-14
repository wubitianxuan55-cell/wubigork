/**
 * schedule/checker.ts — 计划检查器（纯函数，逐条可定位可干预）
 *
 * 定位（对标斑马「检查」+ DCMA 的分工）：
 * - `dcma.ts` = **健康度报表**：14 点阈值、算整份计划的通过率，回答"排得健不健壮"；
 * - 本文件 = **逐条诊断**：给每条发现一个 code、严重级、任务/搭接定位与一句"下一步"，
 *   回答"这一条在哪、该改什么"。两者规则有重叠是刻意的（一个算比率，一个给出定位）。
 *
 * 口径来源：
 * - 结构闸门（悬空/自环/非法类型/分组行挂关系或分配/里程碑带工期）在 Go `Validate`
 *   里是 fail-closed 拒绝，前端在这里**提前告知**（不然用户要等保存失败才知道）；
 * - 语义发现（孤立/无前置/无收口/空分组/无里程碑/已分配未定价）与 Go
 *   `schedule_analyze` 的 `checks` 同口径同措辞，避免"界面说一套、agent 说另一套"；
 * - 冗余关系、负时距、高浮时来自斑马/行业口径（逆向报告 §8.1）。
 *
 * 纯函数、无副作用、无 store 依赖：`checkProject(project, cpm?)`，cpm 缺省时只跑
 * 结构类规则（用于保存前的快速预检）。
 */

import type { CpmResult, SchedLink, SchedProject, SchedTask } from './types'

/** 严重级：error=会写坏文件/排不出（Go 会拒绝）；warn=可疑但可保存；info=仅提示 */
export type DiagSeverity = 'error' | 'warn' | 'info'

export interface Diagnostic {
  /** 稳定代码（供 UI 分类、agent 引用、测试断言） */
  code: string
  severity: DiagSeverity
  /** 一句话说明（含任务/关系名，直接可读） */
  message: string
  /** 下一步建议（可空；与 Go schedule_analyze 文案同口径） */
  hint?: string
  /** 定位到任务（叶/分组行 id） */
  taskId?: string
  /** 定位到关系（project.links 的下标） */
  linkIndex?: number
}

/** 高浮时阈值（工作日）：超过即"排得太松"，与斑马"分组-总时差"口径一致取 20 */
export const HIGH_FLOAT_DAYS = 20
/** 近关键阈值（工作日）：0 < tf ≤ 2 */
export const NEAR_CRITICAL_DAYS = 2

const LINK_TYPES = new Set(['FS', 'SS', 'FF', 'SF'])

function tname(t: SchedTask | undefined, id: string): string {
  return t?.name ?? id
}

/**
 * 计划检查主入口。
 *
 * @param project 计划（缺 tasks/links 时按空处理，绝不抛异常——板块必须能打开）
 * @param cpm     可选 CPM 结果；缺省时跳过依赖排程的规则（高浮时/近关键/无前置判定）
 */
export function checkProject(project: SchedProject, cpm?: CpmResult | null): Diagnostic[] {
  const out: Diagnostic[] = []
  const tasks = Array.isArray(project?.tasks) ? project.tasks : []
  const links = Array.isArray(project?.links) ? project.links : []
  const byId = new Map(tasks.map((t) => [t.id, t]))
  const leaves = tasks.filter((t) => t.level > 0)

  // ── 1) 排程裁决（循环依赖）─────────────────────────────────
  if (cpm && cpm.ok === false && cpm.error) {
    out.push({
      code: 'cycle',
      severity: 'error',
      message: cpm.error,
      hint: '循环的搭接无法排出日期：删掉环上一条关系，或改搭接类型/时距',
    })
  }

  // ── 2) 关系结构（对应 Go Validate 的拒绝项，前端提前报）────────
  const inDeg = new Map<string, number>()
  const outDeg = new Map<string, number>()
  const edgeKey = new Set<string>()
  links.forEach((l: SchedLink, i: number) => {
    const from = byId.get(l.from)
    const to = byId.get(l.to)
    if (!from || !to) {
      out.push({
        code: 'link-dangling',
        severity: 'error',
        message: `搭接 ${l.from}→${l.to} 指向不存在的任务`,
        hint: '删掉这条关系，或先补回被引用的任务',
        linkIndex: i,
        taskId: from ? l.from : to ? l.to : undefined,
      })
      return
    }
    if (l.from === l.to) {
      out.push({
        code: 'link-self',
        severity: 'error',
        message: `任务「${from.name}」搭接到自己`,
        hint: '删除自环关系',
        linkIndex: i,
        taskId: l.from,
      })
      return
    }
    if (!LINK_TYPES.has(l.type)) {
      out.push({
        code: 'link-bad-type',
        severity: 'error',
        message: `搭接 ${from.name}→${to.name} 类型非法：${String(l.type)}`,
        hint: '类型只能是 FS / SS / FF / SF',
        linkIndex: i,
        taskId: l.from,
      })
      return
    }
    // 分组行（level 0）不参与搭接：斑马「分区上存在关系」的 gaea 对应物
    if (from.level === 0 || to.level === 0) {
      out.push({
        code: 'group-link',
        severity: 'error',
        message: `分组行参与搭接：${from.name}→${to.name}`,
        hint: '分组行的时间由子任务汇总，不参与搭接；把关系挂到具体子任务上',
        linkIndex: i,
        taskId: from.level === 0 ? l.from : l.to,
      })
      return
    }
    inDeg.set(l.to, (inDeg.get(l.to) ?? 0) + 1)
    outDeg.set(l.from, (outDeg.get(l.from) ?? 0) + 1)
    edgeKey.add(`${l.from}\u0000${l.to}`)
    if (l.lag < 0) {
      out.push({
        code: 'negative-lag',
        severity: 'warn',
        message: `搭接 ${from.name}→${to.name} 使用负时距 ${l.lag} 天（提前搭接）`,
        hint: '负时距合法但会压缩前序工作，确认工艺上真的可以提前开始',
        linkIndex: i,
        taskId: l.to,
      })
    }
  })

  // ── 3) 传递冗余关系（斑马「多余关系」）──────────────────────
  // A→B、B→C 存在时，A→C 若同类型同时距即为冗余（删掉不影响排程结果）。
  const outEdges = new Map<string, SchedLink[]>()
  for (const l of links) {
    if (!byId.has(l.from) || !byId.has(l.to) || l.from === l.to) continue
    const arr = outEdges.get(l.from)
    if (arr) arr.push(l)
    else outEdges.set(l.from, [l])
  }
  links.forEach((l, i) => {
    if (!byId.has(l.from) || !byId.has(l.to) || l.from === l.to) return
    // 只看长度 2 的传递链：A→B→C，且 A→C
    for (const mid of outEdges.get(l.from) ?? []) {
      if (mid.to === l.to) continue
      const second = (outEdges.get(mid.to) ?? []).find((x) => x.to === l.to)
      if (!second) continue
      if (mid.type === l.type && second.type === l.type && mid.lag === l.lag && second.lag === l.lag) {
        out.push({
          code: 'redundant-link',
          severity: 'warn',
          message: `${tname(byId.get(l.from), l.from)}→${tname(byId.get(l.to), l.to)} 被 ${tname(byId.get(mid.to), mid.to)} 传递覆盖（同类同时距），属多余关系`,
          hint: '删掉直连这条即可，排程结果不变',
          linkIndex: i,
          taskId: l.from,
        })
        return
      }
    }
  })

  // ── 4) 任务结构闸门 ───────────────────────────────────────
  const assignByTask = new Map<string, number>()
  for (const a of Array.isArray(project?.assignments) ? project.assignments : []) {
    assignByTask.set(a.taskId, (assignByTask.get(a.taskId) ?? 0) + 1)
  }
  for (const t of tasks) {
    if (t.level === 0) {
      if (assignByTask.has(t.id)) {
        out.push({
          code: 'group-assignment',
          severity: 'error',
          message: `分组「${t.name}」挂了资源分配`,
          hint: '分组行不挂分配（汇总唯一口径为子孙求和），把分配下沉到子任务',
          taskId: t.id,
        })
      }
      continue
    }
    if (t.isMilestone && t.duration > 0) {
      out.push({
        code: 'milestone-duration',
        severity: 'error',
        message: `里程碑「${t.name}」工期为 ${t.duration}（应为 0）`,
        hint: '里程碑是时点：把工期改成 0，或取消里程碑标记',
        taskId: t.id,
      })
    }
    if (t.duration < 0) {
      out.push({
        code: 'duration-negative',
        severity: 'error',
        message: `任务「${t.name}」工期为负：${t.duration}`,
        hint: '工期必须 ≥ 0（工作日整数）',
        taskId: t.id,
      })
    }
    if (t.durationUnit === 'cd' && t.duration > 3650) {
      out.push({
        code: 'duration-cd-over',
        severity: 'error',
        message: `任务「${t.name}」日历天工期 ${t.duration} 超上限 3650`,
        hint: '检查单位是否误设为日历天（cd）',
        taskId: t.id,
      })
    }
  }

  // ── 5) 语义发现（与 Go schedule_analyze checks 同口径）────────
  // 首个非手动叶任务 = 计划起点，天然无前置，不算问题（与 Go 口径一致）
  const firstStart = leaves.find((t) => t.mode !== 'manual')?.id
  const lastLeaf = leaves[leaves.length - 1]?.id
  for (const t of leaves) {
    const inN = inDeg.get(t.id) ?? 0
    const outN = outDeg.get(t.id) ?? 0
    if (inN === 0 && outN === 0) {
      out.push({
        code: 'isolated-task',
        severity: 'warn',
        message: `任务「${t.name}」没有任何搭接关系，像孤立项`,
        hint: '确认是否漏了前置/后继；需要快速串起来可用 ops auto_chain',
        taskId: t.id,
      })
      continue
    }
    if (inN === 0 && t.id !== firstStart && t.mode !== 'manual') {
      out.push({
        code: 'no-predecessor',
        severity: 'warn',
        message: `任务「${t.name}」没有前置搭接`,
        hint: '可用 ops auto_chain 补缺省串联，再按工艺逻辑覆写例外',
        taskId: t.id,
      })
    }
    if (outN === 0 && t.id !== lastLeaf) {
      out.push({
        code: 'open-end',
        severity: 'warn',
        message: `任务「${t.name}」没有后继（收口缺失）`,
        hint: '确认它是否应汇入收尾工作；悬空尾巴会让总工期判断失真',
        taskId: t.id,
      })
    }
  }
  // 空分组：分组行后面（到下一个分组行之前）没有任何子任务
  {
    const childCount = new Map<string, number>()
    let cur: string | null = null
    for (const t of tasks) {
      if (t.level === 0) {
        cur = t.id
        if (!childCount.has(t.id)) childCount.set(t.id, 0)
      } else if (cur) {
        childCount.set(cur, (childCount.get(cur) ?? 0) + 1)
      }
    }
    for (const t of tasks) {
      if (t.level !== 0) continue
      if ((childCount.get(t.id) ?? 0) === 0) {
        out.push({
          code: 'empty-group',
          severity: 'warn',
          message: `分组「${t.name}」没有任何子任务`,
          hint: '补子任务，或删掉这个分组',
          taskId: t.id,
        })
      }
    }
  }
  if (tasks.length > 0 && !leaves.some((t) => t.isMilestone)) {
    out.push({
      code: 'no-milestone',
      severity: 'info',
      message: '这份计划没有里程碑',
      hint: '至少给关键节点设一个里程碑，汇报与预警都需要锚点',
    })
  }
  if (links.length === 0 && leaves.length > 1) {
    out.push({
      code: 'no-links',
      severity: 'warn',
      message: '所有任务之间没有任何搭接关系',
      hint: '没有逻辑关系时 CPM 只能并排堆着；用 ops auto_chain 起个头',
    })
  }

  // ── 6) 成本口径提示（与 Go Unpriced 同口径）──────────────────
  const resById = new Map((Array.isArray(project?.resources) ? project.resources : []).map((r) => [r.id, r]))
  for (const a of Array.isArray(project?.assignments) ? project.assignments : []) {
    const res = resById.get(a.resourceId)
    const task = byId.get(a.taskId)
    if (!res || !task || task.level === 0) continue
    const unpriced =
      (res.type === 'work' && (res.standardRate ?? 0) === 0 && (res.costPerUse ?? 0) === 0) ||
      (res.type === 'cost' && (a.amount ?? 0) === 0)
    if (unpriced) {
      out.push({
        code: 'unpriced-assignment',
        severity: 'info',
        message: `任务「${task.name}」使用了未定价资源「${res.name}」，成本按 0 计`,
        hint: res.type === 'work' ? '给工时资源设标准费率（元/工日）' : '给成本资源填金额',
        taskId: task.id,
      })
    }
  }

  // ── 7) 排程相关提示（需 cpm）────────────────────────────────
  if (cpm && cpm.ok) {
    for (const t of leaves) {
      // 关键工作不逐条报（会刷屏）：交给 DCMA / 横道红色条呈现
      const row = cpm.rows?.[t.id]
      if (!row) continue
      if (row.tf > HIGH_FLOAT_DAYS) {
        out.push({
          code: 'high-float',
          severity: 'info',
          message: `任务「${t.name}」总时差 ${row.tf} 天，排得很松`,
          hint: `总时差超过 ${HIGH_FLOAT_DAYS} 天通常意味着逻辑缺环或工期偏保守`,
          taskId: t.id,
        })
      } else if (row.tf > 0 && row.tf <= NEAR_CRITICAL_DAYS) {
        out.push({
          code: 'near-critical',
          severity: 'info',
          message: `任务「${t.name}」总时差仅 ${row.tf} 天，属近关键`,
          hint: '缓冲很小，轻微拖延就会顶到关键线路',
          taskId: t.id,
        })
      }
    }
  }

  return out
}

/** 严重级排序（error → warn → info），同级按任务表序稳定 */
export function sortDiagnostics(list: Diagnostic[], tasks: SchedTask[]): Diagnostic[] {
  const order = new Map(tasks.map((t, i) => [t.id, i]))
  const rank: Record<DiagSeverity, number> = { error: 0, warn: 1, info: 2 }
  return [...list].sort((a, b) => {
    const ra = rank[a.severity] - rank[b.severity]
    if (ra !== 0) return ra
    const ta = a.taskId !== undefined ? (order.get(a.taskId) ?? 1e9) : -1
    const tb = b.taskId !== undefined ? (order.get(b.taskId) ?? 1e9) : -1
    if (ta !== tb) return ta - tb
    return a.code < b.code ? -1 : a.code > b.code ? 1 : 0
  })
}

/** 汇总（界面徽标/agent 回执用） */
export function diagSummary(list: Diagnostic[]): { error: number; warn: number; info: number } {
  const s = { error: 0, warn: 0, info: 0 }
  for (const d of list) s[d.severity]++
  return s
}
