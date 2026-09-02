// foreshadowLogic.ts — 伏笔登记表纯逻辑（无 React/绑定依赖，可单测）
// 覆盖：manual_ 条目 ID 生成、章节号→文件名、状态流转合法性、登记表单→条目构造、
// GetForeshadows 载荷收窄。
import type { ForeshadowItemData, ForeshadowStatus } from '../../types'

/** 手工登记条目 ID：manual_ 前缀 + 毫秒时间戳（与 AI stable ID {category}_{chapter}_{hash} 天然不冲突） */
export function makeManualForeshadowID(now: number = Date.now()): string {
  return `manual_${now}`
}

/** 埋设章节号 → 章节文件名（对齐 AI 侧 fmt.Sprintf("%03d.md", n)） */
export function formatPlantedIn(chapterNum: number): string {
  const n = Math.floor(chapterNum)
  const safe = Number.isFinite(n) && n >= 1 ? n : 1
  return `${String(safe).padStart(3, '0')}.md`
}

/**
 * 状态流转（单按钮语义）：
 * planted → hinted（标记暗示）→ revealed（标记回收）；revealed 回退一步到 hinted（可再回收）。
 */
export function advanceForeshadowStatus(status: ForeshadowStatus): ForeshadowStatus {
  if (status === 'planted') return 'hinted'
  if (status === 'revealed') return 'hinted' // 回退
  return 'revealed'
}

/** 流转按钮文案 */
export function foreshadowFlowLabel(status: ForeshadowStatus): string {
  if (status === 'planted') return '标记暗示'
  if (status === 'hinted') return '标记回收'
  return '回退到暗示'
}

/** 登记表单输入 */
export interface ManualForeshadowInput {
  category: string // character / plot / world / relationship
  description: string
  chapterNum: number
  isLongTerm: boolean
}

/** 登记表单 → 新伏笔条目（Status 默认 planted，ID 为 manual_ 前缀） */
export function buildManualForeshadow(input: ManualForeshadowInput, now: number = Date.now()): ForeshadowItemData {
  return {
    id: makeManualForeshadowID(now),
    category: input.category,
    description: input.description,
    planted_in: formatPlantedIn(input.chapterNum),
    status: 'planted',
    is_long_term: input.isLongTerm,
  }
}

/** GetForeshadows 载荷（Go map[string]interface{}）收窄为条目数组 */
export function normalizeForeshadowItems(raw: unknown): ForeshadowItemData[] {
  const list = ((raw as { items?: ForeshadowItemData[] } | null)?.items ?? []) as ForeshadowItemData[]
  return Array.isArray(list) ? list : []
}
