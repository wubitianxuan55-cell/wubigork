/** 品牌主题工具 */

/** 快捷引用 CSS 变量 */
export const C = (name: string) => `var(--${name})`

/** canvas/3D 场景/WebGL 无法消费 var()，把 CSS 颜色串解析为具体色值。
 *  span 探针法：交给浏览器计算，天然支持 color-mix()/嵌套 var()/fallback 链
 *  （字符串手拆解析既不支持 color-mix 也不支持运算，v4.390 前三份手写实现
 *  各缺一块）。解析失败/非浏览器环境原样返回，由调用方 fallback 兜底。
 *  值在主题切换后变化——调用方须把 darkMode 或令牌变更纳入 memo 依赖重解析。 */
export function resolveThemeColor(spec: string): string {
  if (typeof document === 'undefined') return spec
  if (!spec.includes('var(') && !spec.includes('color-mix(')) return spec
  try {
    const el = document.createElement('span')
    el.style.display = 'none'
    el.style.color = spec
    document.body.appendChild(el)
    const resolved = getComputedStyle(el).color
    el.remove()
    return resolved && resolved !== '' ? resolved : spec
  } catch {
    return spec
  }
}

/** resolveThemeColor 的 RGB 元组版：canvas 场景常需派生半透明色（rgba()），
 *  hex 后缀拼接（`${color}22`）只接受 6 位 hex 输入——探针/令牌返回 rgb() 串时
 *  会拼出非法色被 canvas 静默忽略。这里统一解析为 {r,g,b}，不认识返回 null。 */
export function resolveThemeColorRGB(spec: string): { r: number; g: number; b: number } | null {
  const resolved = resolveThemeColor(spec).trim()
  const hex = /^#([0-9a-f]{6})$/i.exec(resolved)
  if (hex) {
    const n = parseInt(hex[1], 16)
    return { r: (n >> 16) & 255, g: (n >> 8) & 255, b: n & 255 }
  }
  const rgb = /^rgba?\(\s*([\d.]+)[,\s]+([\d.]+)[,\s]+([\d.]+)/i.exec(resolved)
  if (rgb) {
    return { r: Math.round(+rgb[1]), g: Math.round(+rgb[2]), b: Math.round(+rgb[3]) }
  }
  return null
}

/** 状态颜色常量（3.0 Wave 4：硬编码 hex → 语义令牌，随 12 主题联动） */
export const STATUS_COLORS: Record<string, string> = {
  planned: 'var(--color-text-secondary)',
  writing: 'var(--color-primary)',
  done: 'var(--color-success)',
  abandoned: 'var(--color-destructive)',
}

export const STATUS_LABELS: Record<string, string> = {
  planned: '待写', writing: '写作中', done: '已完成', abandoned: '已弃',
}

/** 角色类型颜色（语义令牌） */
export const ROLE_COLORS: Record<string, string> = {
  protagonist: 'var(--color-primary)',
  antagonist: 'var(--color-destructive)',
  supporting: 'var(--color-text-secondary)',
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
