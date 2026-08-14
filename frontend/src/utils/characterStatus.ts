/**
 * 角色状态枚举（CharacterData.status / LibraryCharacter.status）前端校验（T6-7.5 状态收敛）。
 *
 * 审查发现：状态为英文枚举（"Alive" 等），各页面各自手写选项/标签映射，且对
 * 后端/角色库返回的非法值（历史数据、AI 生成异常值）无校验，UI 会直接展示原始
 * 英文串。这里收敛为单一事实来源：非法值统一回退默认 'Alive'。
 */

export const CHARACTER_STATUSES = ['Alive', 'Dead', 'Missing', 'Transformed'] as const

export type CharacterStatus = (typeof CHARACTER_STATUSES)[number]

/** 非法/缺失状态的回退默认值 */
export const DEFAULT_CHARACTER_STATUS: CharacterStatus = 'Alive'

const CHARACTER_STATUS_SET: ReadonlySet<string> = new Set<string>(CHARACTER_STATUSES)

/** 是否为合法的角色状态枚举值 */
export function isCharacterStatus(value: unknown): value is CharacterStatus {
  return typeof value === 'string' && CHARACTER_STATUS_SET.has(value)
}

/** 校验并收敛角色状态：非法值（含空/缺失）回退默认 'Alive' */
export function normalizeCharacterStatus(value: unknown): CharacterStatus {
  return isCharacterStatus(value) ? value : DEFAULT_CHARACTER_STATUS
}

/** 角色状态 → 中文标签（非法值按默认状态展示，不泄露原始英文串） */
export const CHARACTER_STATUS_LABELS: Record<CharacterStatus, string> = {
  Alive: '存活',
  Dead: '已故',
  Missing: '失踪',
  Transformed: '变身',
}

/** 角色状态中文标签（带非法值回退） */
export function characterStatusLabel(value: unknown): string {
  return CHARACTER_STATUS_LABELS[normalizeCharacterStatus(value)]
}

/** Select 选项（状态枚举唯一来源，页面不再各自手写） */
export const CHARACTER_STATUS_OPTIONS = CHARACTER_STATUSES.map((s) => ({
  value: s,
  label: CHARACTER_STATUS_LABELS[s],
}))
