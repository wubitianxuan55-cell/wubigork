/**
 * pathDriver.ts — 任务路径分析引擎：驱动依赖与驱动链（纯函数，v4.136.0 刀A）
 *
 * 思路 distilled 自 ProjectLibre（E8 机制）：逐依赖算自由时差，某条依赖的
 * ff===0 即该依赖的搭接约束**绑定**了后继日期——这条依赖在「驱动」后继。
 * 视图层据此高亮某任务的前驱/后继驱动链（为什么是这个日期：沿驱动边上溯/下溯）。
 *
 * 口径：
 *  - 依赖键 from->to（同依赖表唯一）；ff 复用 cpm.freeFloatPart（四型+时距同口径，
 *    FS/FF 锚点=前置 EF、SS/SF 锚点=前置 ES）；
 *  - 驱动判定 ff===0 且**后继为 auto 模式**（mode 缺省=auto）：手动后继日期锁死
 *    （manualStart），无依赖能驱动它；
 *  - cpm.ok===false（循环依赖等）一律 fail-closed 返回空，不静默出半结果；
 *  - 驱动链只走叶任务（level!==0，分组行不进链）；里程碑正常参与；
 *  - 手动任务可为链端点：作为上游仍驱动别人（其后继以它的 EF 为约束），
 *    但它的上游不再继续上溯（manualStart 不由依赖决定）；succ 方向同理，
 *    手动后继进链即端点、不再扩展。
 */
import type { CpmResult, SchedLink, SchedTask } from './types'
import { freeFloatPart } from './cpm'

/** 依赖键：from->to（from 为前置任务，to 为后续任务） */
export function linkKey(l: SchedLink): string {
  return `${l.from}->${l.to}`
}

/**
 * 每条依赖的自由时差表（键=linkKey）。复用 freeFloatPart 四型口径；
 * from/to 任一不在 cpm.rows 或不在任务表中（悬空搭接）跳过——与 cpm 的
 * 有效搭接过滤同口径，不产出幽灵键。
 */
export function depFreeSlacks(tasks: SchedTask[], links: SchedLink[], cpm: CpmResult): Record<string, number> {
  const out: Record<string, number> = {}
  const ids = new Set(tasks.map((t) => t.id))
  for (const l of links) {
    if (l.from === l.to || !ids.has(l.from) || !ids.has(l.to)) continue
    const from = cpm.rows[l.from]
    const to = cpm.rows[l.to]
    if (!from || !to) continue
    out[linkKey(l)] = freeFloatPart(l, to, from.ef, from.es)
  }
  return out
}

/**
 * 驱动依赖键集：逐依赖 ff===0 且后继任务为 auto（mode 缺省=auto）。
 * 手动后继日期锁死，指向它的依赖再紧也不「驱动」它；cpm.ok===false 时返回空集。
 */
export function drivingLinkKeys(tasks: SchedTask[], links: SchedLink[], cpm: CpmResult): Set<string> {
  const out = new Set<string>()
  if (!cpm.ok) return out
  const byId = new Map(tasks.map((t) => [t.id, t]))
  const slacks = depFreeSlacks(tasks, links, cpm)
  for (const l of links) {
    const key = linkKey(l)
    if (slacks[key] !== 0) continue
    const to = byId.get(l.to)
    if (!to || to.mode === 'manual') continue
    out.add(key)
  }
  return out
}

/** 驱动链：链上任务 id（不含起点、去重保序）与链上依赖边（按入参顺序） */
export interface DrivingChain {
  taskIds: string[]
  links: SchedLink[]
}

/**
 * 从 taskId 出发沿**驱动依赖** BFS：
 *  - pred：逆着驱动边走上游（进入当前任务且 ff===0 的边）；succ：顺流走下游；
 *    both 双向合一遍历（visited 去重，菱形汇合不重复访问）；
 *  - 只走叶任务（level!==0）；起点是分组行（level===0）直接返回空；
 *  - 经驱动边到达的任务可继续扩展（手动节点天然无驱动入边=上游截断）；
 *    经「手动后继」边到达的手动任务只进链、不可扩展（日期锁死，链到它为止）；
 *  - cpm.ok===false 或任务不存在返回空。里程碑正常参与。
 */
export function drivingChain(
  tasks: SchedTask[],
  links: SchedLink[],
  cpm: CpmResult,
  taskId: string,
  dir: 'pred' | 'succ' | 'both',
): DrivingChain {
  const byId = new Map(tasks.map((t) => [t.id, t]))
  const start = byId.get(taskId)
  if (!cpm.ok || !start || start.level === 0) return { taskIds: [], links: [] }

  const driving = drivingLinkKeys(tasks, links, cpm)
  const outLinks = new Map<string, SchedLink[]>()
  const inLinks = new Map<string, SchedLink[]>()
  for (const l of links) {
    if (l.from === l.to || !byId.has(l.from) || !byId.has(l.to)) continue
    if (!outLinks.has(l.from)) outLinks.set(l.from, [])
    if (!inLinks.has(l.to)) inLinks.set(l.to, [])
    outLinks.get(l.from)!.push(l)
    inLinks.get(l.to)!.push(l)
  }

  const taskIds: string[] = []
  const visited = new Set<string>([taskId])
  // 可扩展集：起点 + 经驱动边到达者；手动后继端点只进链不扩展
  const expandable = new Set<string>([taskId])
  const chainLinks = new Set<SchedLink>()
  const queue: string[] = [taskId]
  while (queue.length > 0) {
    const cur = queue.shift()!
    if (!expandable.has(cur)) continue
    if (dir !== 'succ') {
      for (const l of inLinks.get(cur) ?? []) {
        if (!driving.has(linkKey(l))) continue
        const prev = byId.get(l.from)!
        if (prev.level === 0) continue
        chainLinks.add(l)
        expandable.add(l.from)
        if (!visited.has(l.from)) {
          visited.add(l.from)
          taskIds.push(l.from)
          queue.push(l.from)
        }
      }
    }
    if (dir !== 'pred') {
      for (const l of outLinks.get(cur) ?? []) {
        const next = byId.get(l.to)!
        if (next.level === 0) continue
        if (next.mode === 'manual') {
          // 手动后继：进链即端点（日期锁死，无依赖能驱动它），不再扩展
          chainLinks.add(l)
          if (!visited.has(l.to)) {
            visited.add(l.to)
            taskIds.push(l.to)
            queue.push(l.to)
          }
          continue
        }
        if (!driving.has(linkKey(l))) continue
        chainLinks.add(l)
        expandable.add(l.to)
        if (!visited.has(l.to)) {
          visited.add(l.to)
          taskIds.push(l.to)
          queue.push(l.to)
        }
      }
    }
  }
  return { taskIds, links: links.filter((l) => chainLinks.has(l)) }
}
