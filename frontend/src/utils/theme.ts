/** 品牌主题工具 */

/** 快捷引用 CSS 变量 */
export const C = (name: string) => `var(--${name})`

/** 状态颜色常量 */
export const STATUS_COLORS: Record<string, string> = {
  planned: 'var(--color-text-secondary)',
  writing: '#60a5fa',
  done: 'var(--color-primary)',
  abandoned: '#f87171',
}

export const STATUS_LABELS: Record<string, string> = {
  planned: '待写', writing: '写作中', done: '已完成', abandoned: '已弃',
}

/** 角色类型颜色 */
export const ROLE_COLORS: Record<string, string> = {
  protagonist: 'var(--color-primary)',
  antagonist: '#f87171',
  supporting: '#60a5fa',
  minor: 'var(--color-text-secondary)',
}

export const ROLE_LABELS: Record<string, string> = {
  protagonist: '主角', antagonist: '反派', supporting: '配角', minor: '次要',
}

/** 关系类型标签 */
export const RELATION_LABELS: Record<string, string> = {
  friend: '朋友', enemy: '敌人', family: '家人', mentor: '导师',
  rival: '对手', lover: '恋人', member: '成员', leader: '领袖',
}
