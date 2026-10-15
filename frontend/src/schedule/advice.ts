// advice.ts — 进度逻辑纠错建议生成器（阶段十一 11.4）
//
// 「AI 纠错」的落点=DCMA 体检发现 → **人可读修复建议**的确定性映射（纯函数、
// 零 LLM——纠错建议在样例工程上可复现是出口判据，LLM 不可复现）。检查面已由
// dcmaAudit 覆盖（缺逻辑/负搭接/悬空/断路），本文件补「给出修复建议」半边：
// 每条建议 = 问题 + 涉及任务 + 具体动作（挂接谁/删哪条/改什么）。
//
// 与 dcmaAudit 的关系：dcmaAudit 管体检单（14 点合规判定），advice 管修复建
// 议（作者下一步动手清单）；两者同源（同一 CPM 结果+同一关系表），结论必然
// 一致——建议是检查的下游投影，不另立第二套判定。
import type { SchedProject, SchedLink } from './types'
import type { CpmResult } from './types'

export type AdviceKind =
  | 'missing-link' // 缺逻辑（孤立/单侧缺失且非首末）
  | 'negative-lag' // 负搭接
  | 'dangling' // 悬空引用（关系指向不存在的任务）
  | 'redundant-link' // 冗余搭接（A→B 已直达又经 C 间接可达，传递闭包冗余）

export interface ScheduleAdvice {
  kind: AdviceKind
  /** 涉及任务名（展示用） */
  tasks: string[]
  /** 人可读问题 */
  message: string
  /** 具体修复动作（作者下一步动手清单） */
  action: string
}

const nameOf = (p: SchedProject, id: string) => p.tasks.find((t) => t.id === id)?.name ?? id

/** 叶任务（非分组；分组行不参与逻辑） */
const leaves = (p: SchedProject) => p.tasks.filter((t) => t.level !== 0 && !t.isMilestone)

/**
 * 进度逻辑纠错建议（确定性、零 LLM、可复现）。
 *
 * 覆盖四类（与 dcmaAudit 前四点同源）：
 *   1. missing-link：DCMA 第 1 点超标（或缺逻辑任务存在）时，给出孤立任务
 *      的挂接建议（挂到当前开工首任务之前 / 竣工末任务之后）；
 *   2. negative-lag：负搭接关系逐条列出（建议改 SS/FF 正搭接或删）；
 *   3. dangling：悬空引用逐条列出（建议补任务或删关系）；
 *   4. redundant-link：直达关系被间接路径覆盖的冗余（建议删直达，减少理解噪音；
 *      不影响 CPM 结果——删后关键线路不变，建议中注明）。
 */
export function scheduleAdvice(p: SchedProject, cpm: CpmResult): ScheduleAdvice[] {
  const out: ScheduleAdvice[] = []
  if (!cpm.ok) {
    // 循环依赖：CPM 已拒绝——修复建议=按环Breaking（cycle 列表点名）。
    const cycle = cpm.cycle ?? []
    if (cycle.length > 0) {
      out.push({
        kind: 'missing-link',
        tasks: cycle,
        message: `存在循环依赖：${cycle.join(' → ')}——排程无法计算`,
        action: '打断环中最不合理的一条关系（通常是最晚加上的那条），或改搭接类型',
      })
    }
    return out
  }

  const leafIds = new Set(leaves(p).map((t) => t.id))
  const nameById = (id: string) => nameOf(p, id)

  // ── 悬空引用（优先级最高：关系指向不存在的任务，CPM 静默忽略）──
  const taskIds = new Set(p.tasks.map((t) => t.id))
  const dangling = p.links.filter((l) => !taskIds.has(l.from) || !taskIds.has(l.to))
  if (dangling.length > 0) {
    out.push({
      kind: 'dangling',
      tasks: dangling.map((l) => `${nameById(l.from)} → ${nameById(l.to)}`),
      message: `${dangling.length} 条关系引用了不存在的任务（删任务时未级联清理）`,
      action: '在关系表中删除这些悬空关系（或补回被误删的任务）',
    })
  }

  // ── 负搭接 ──
  const negLags = p.links.filter((l) => l.lag < 0)
  if (negLags.length > 0) {
    out.push({
      kind: 'negative-lag',
      tasks: negLags.map((l) => `${nameById(l.from)} → ${nameById(l.to)} (${l.lag}d)`),
      message: `${negLags.length} 条负搭接（lag<0）——DCMA 第 2 点不合规，计划可被「改到过去」`,
      action: '改为 0 或正 lag；确需提前开工的用 SS+正滞后表达',
    })
  }

  // ── 缺逻辑（孤立任务：双侧皆无关系）──
  const hasPred = new Set<string>()
  const hasSucc = new Set<string>()
  for (const l of p.links) {
    if (leafIds.has(l.from)) hasSucc.add(l.from)
    if (leafIds.has(l.to)) hasPred.add(l.to)
  }
  const isolated = leaves(p).filter((t) => !hasPred.has(t.id) && !hasSucc.has(t.id))
  if (isolated.length > 0) {
    const names = isolated.map((t) => t.name)
    out.push({
      kind: 'missing-link',
      tasks: names,
      message: `${isolated.length} 个任务完全孤立（无前驱也无后继）——不在任何路径上，工期失真`,
      action: `把「${names[0]}」挂接到实际的前后任务（开工序挂项目首任务，收尾工序挂末任务）`,
    })
  }

  // ── 冗余搭接（A→B 直达 + A→C→B 间接并存；仅统计长度 2 的传递闭包）──
  const direct = new Set(p.links.map((l) => l.from + '\x00' + l.to))
  const via: Record<string, string[]> = {}
  for (const l1 of p.links) {
    for (const l2 of p.links) {
      if (l1.to !== l2.from || l1.from === l2.to) continue
      const key = l1.from + '\x00' + l2.to
      if (direct.has(key) && l1.from !== l2.from) {
        (via[key] ??= []).push(nameById(l2.from))
      }
    }
  }
  const redundant: string[] = []
  for (const [key, mids] of Object.entries(via)) {
    const [from, to] = key.split('\x00')
    redundant.push(`${nameById(from)} → ${nameById(to)}（经 ${[...new Set(mids)].join('、')} 间接可达）`)
  }
  if (redundant.length > 0) {
    out.push({
      kind: 'redundant-link',
      tasks: redundant,
      message: `${redundant.length} 条直达关系已被间接路径覆盖（冗余）——不影响工期但干扰网络图阅读`,
      action: '删除直达关系（删除后总工期与关键线路不变；不确定时保留亦可）',
    })
  }

  // 单侧缺失（有前驱无后继/反之）的占比判定**归 dcmaAudit 第 1 点**（首末
  // 豁免+5% 门槛在那边已有准确实现——多链工程每链各有合法首尾，这里再实现
  // 一套必然误报，不另立第二套判定）。本文件的孤立任务建议（双侧皆无）是
  // 确定性信号，与 DCMA 互补。

  return out
}

/** 冗余检测用的关系引用（导出供测试构造）。 */
export type AdviceLink = SchedLink
