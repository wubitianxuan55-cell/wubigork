/**
 * exportPalette.ts — 显示预设 → 导出图面配色（W6）
 *
 * 屏上预设走 CSS 变量（`displayPreset.presetColorMap`），但上报图的 SVG 构建器
 * 用的是 `ganttExport.EXP_COLORS` 里的**纸面固定色**（发布物不随屏上主题漂移）。
 * 这里在导出前把六组可映射色替换成当前预设的纸面档（`toSvgPalette`），
 * 保证「屏上看到护眼/黑白」与「导出的 PNG/PDF 同色相」。
 *
 * 映射只覆盖 `--sched-*` 领域色能对上的导出色；墨色/边框/表头底/浮动条/竣工线
 * 等版式色不动（它们承担版式层级，不是领域语义）。`normal` 原样返回。
 */
import { toSvgPalette, type DisplayPreset, type SchedVar } from './displayPreset'
import { EXP_COLORS } from './ganttExport'

/** 预设变量 → 导出色键（顺序固定，测试按它断言） */
export const EXPORT_PALETTE_MAP: readonly { key: keyof typeof EXP_COLORS; variable: SchedVar }[] = [
  { key: 'critical', variable: '--sched-critical' },
  { key: 'bar', variable: '--sched-normal' },
  { key: 'link', variable: '--sched-link' },
  { key: 'grid', variable: '--sched-grid' },
  { key: 'weekend', variable: '--sched-weekend' },
]

/**
 * 把导出 SVG 里映射到的固定色替换成该预设的纸面配色。
 * 只做**精确色值替换**（大小写不敏感的整串），不触碰其它属性；`normal` 零改动。
 */
export function applyExportPalette(svg: string, preset: DisplayPreset | string | null | undefined): string {
  if (!svg || preset == null || preset === 'normal') return svg
  const palette = toSvgPalette(preset)
  let out = svg
  for (const { key, variable } of EXPORT_PALETTE_MAP) {
    const from = EXP_COLORS[key]
    const to = palette[variable]
    if (!from || !to) continue
    // 大小写不敏感：构建器里是固定小写 hex，但用户/第三方 SVG 可能大写
    out = out.replace(new RegExp(from.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'gi'), to)
  }
  return out
}
