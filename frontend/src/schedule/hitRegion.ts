/**
 * schedule/hitRegion.ts — 命中区枚举与优先级（ZPert8 W4 剩余项「命中区顺序」纯函数层）
 *
 * 逆向依据（`进度计划/gaea-schedule-zpert8-uiux-re-20261114.md`）：
 *  - §16.2 `FUN_1403c16d0(doc, hitType, pt)` 统一命中测试 —— 表格/画布/表头/分隔条
 *    共用**一套**查询，不是按控件各写一套；命中对象走对象菜单，未命中走空白菜单；
 *  - §17.1 `hitType` 取值（0..6）是**硬编码 if 链**，先判先得，优先级写死在代码里。
 * 移植口径（`...uiux-port-plan-20261114.md` §W4）：「命中区顺序固定：列头 → 画布 →
 * 单元格 → 状态栏 → 分隔条 → 行头」。
 *
 * 本模块零 DOM 依赖（除 `hitRegionOfElement` 只读 `data-hit` 属性），可独立单测：
 *  - `hitRegionOfElement` 自限深度向上找 `data-hit`，**绝不读 DOM 文本**当判据
 *    （文本随时被就地编辑改掉，属性才是稳定契约）；
 *  - 口径只有一条：**优先级高者胜**（`pickHitRegion`）。向上查找时把沿途声明都收进
 *    候选，再由优先级裁决 —— 因此调用方要不要给外层容器加 `data-hit` 必须先想清楚：
 *    外层若声明了更高优先级的区域，会把内层更具体的区域盖掉（例如给表格行声明
 *    `cell` 会让行头的 `row-header` 被裁成 `cell`，所以表格行/表体故意不声明）。
 */

/** 命中区（值同时是 `data-hit` 属性的合法取值，DOM 上可读可断言） */
export type HitRegion = 'column-header' | 'canvas' | 'cell' | 'status-bar' | 'splitter' | 'row-header'

/**
 * 命中区优先级：**下标越小越优先**（严格按移植计划 §W4 的固定顺序）。
 * 同一坐标被多处声明时，取本表里最靠前的那个。
 */
export const HIT_REGION_ORDER: readonly HitRegion[] = [
  'column-header',
  'canvas',
  'cell',
  'status-bar',
  'splitter',
  'row-header',
]

/** 元素声明命中区的唯一属性名（写入与读取都只认它） */
export const HIT_ATTR = 'data-hit'

/**
 * 向上查找的最大层数：单元格 → 行 → 表体 → 窗格 → 框 → 根，正常链路 ≤5 层；
 * 给到 8 是留覆盖层/浮层余量，超限即视为“未命中”（防止一路走到 document 误命中）。
 */
export const HIT_LOOKUP_DEPTH = 8

const REGION_SET: ReadonlySet<string> = new Set<string>(HIT_REGION_ORDER)
const RANK: ReadonlyMap<string, number> = new Map(HIT_REGION_ORDER.map((r, i) => [r, i]))

/** `data-hit` 读到的东西是否是我们认识的命中区（外部脏值一律不认）。 */
export function isHitRegion(value: string | null | undefined): value is HitRegion {
  return !!value && REGION_SET.has(value)
}

/** 命中区优先级序号（0=最高）；未知区域返回 -1（便于“无效值排在最后”的比较）。 */
export function hitRegionRank(region: string | null | undefined): number {
  if (!isHitRegion(region)) return -1
  return RANK.get(region) ?? -1
}

/**
 * 候选命中区 → 最高优先级那一个（同区重复不影响结果）；空/全未知 → null。
 * 这就是 §17.1 那条 if 链的等价物：先判先得，只是把“顺序”提成了数据。
 */
export function pickHitRegion(hits: readonly (string | null | undefined)[]): HitRegion | null {
  let best: HitRegion | null = null
  let bestRank = Number.POSITIVE_INFINITY
  for (const hit of hits) {
    if (!isHitRegion(hit)) continue
    const rank = hitRegionRank(hit)
    if (rank < bestRank) {
      best = hit
      bestRank = rank
    }
  }
  return best
}

/**
 * 从元素向上收集命中区声明（**最近优先**的顺序无关紧要 —— 结果交给 `pickHitRegion`
 * 按优先级裁决）。自限深度 `maxDepth` 个元素（含起点），未知 `data-hit` 值跳过继续
 * 向上（脏属性不该阻断真实命中）。
 */
export function hitRegionChain(
  el: Element | null,
  maxDepth: number = HIT_LOOKUP_DEPTH,
  attr: string = HIT_ATTR,
): HitRegion[] {
  const out: HitRegion[] = []
  if (!el || maxDepth <= 0) return out
  let cur: Element | null = el
  let depth = 0
  while (cur && depth < maxDepth) {
    const raw = cur.getAttribute(attr)
    if (isHitRegion(raw)) out.push(raw)
    cur = cur.parentElement
    depth++
  }
  return out
}

/**
 * 元素 → 命中区（单入口）。向上找 `data-hit` 并做优先级裁决，找不到/超深/脏值 → null。
 * 只读属性、不读文本；`maxDepth` 可调（测试深度上限用）。
 */
export function hitRegionOfElement(
  el: Element | null,
  maxDepth: number = HIT_LOOKUP_DEPTH,
  attr: string = HIT_ATTR,
): HitRegion | null {
  return pickHitRegion(hitRegionChain(el, maxDepth, attr))
}
