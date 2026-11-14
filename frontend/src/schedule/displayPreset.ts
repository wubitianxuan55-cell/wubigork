/**
 * schedule/displayPreset.ts — 显示预设（护眼 / 黑白）纯数据层（ZPert8 W6「输出与外观」）
 *
 * 口径来源：`进度计划/gaea-schedule-zpert8-uiux-port-plan-20261114.md` §W6
 * 「增加护眼与黑白两个显示预设，复用 gaea 主题令牌」，§6 风险「令牌适配，不硬编码
 * 斑马色值」——预设只给 `--sched-*` 领域色换值，**不碰** `--md-sys-color-*` 主题令牌，
 * 因此亮/暗两套基座都能叠加，主题换色相时预设也不会被一起改掉。
 *
 * 三层职责严格分开（都不碰 DOM）：
 *  1. `isDisplayPreset` / `normalizePreset`：外部脏值（旧偏好、手改 JSON）统一回落
 *     `normal`，绝不把未知字符串散进 DOM 类名；
 *  2. `presetColorMap(preset)`：**屏幕深底档**（`schedule.css` 的 `:root` 缺省档=暗色）
 *     的变量覆盖表，键就是 CSS 变量名，调用方直接 `style.setProperty(k, v)` 即可；
 *  3. `toSvgPalette(preset)`：**纸面白底档**，导出 PNG/PDF 时用；两档值同源同键集，
 *     所以屏上看到的色相层级与导出件一致（黑白=纯灰阶，护眼=低饱和）。
 *
 * 色值一律写 `rgb()` / `rgba()` 字面量而不是 `#rrggbb`：
 * 仓库自定义 ESLint 规则 `local/no-raw-hex`（frontend/eslint-rules/no-raw-hex.js）
 * 对 `**\/*.{ts,tsx}` 报错，且本文件的色值要被 `displayPreset.css.test.ts`
 * 与 `display-preset.css` 逐值比对，统一写法才可比。
 *
 * 对比度取值依据（WCAG 2.1 相对亮度、sRGB 线性化，实测值见 `display-preset.css`
 * 头部注释）：文字级 ≥4.5:1、图形级 ≥3:1；`schedule.css` 里 `--sched-critical` /
 * `--sched-normal` 同时被当文字色（`.sched-badge-*` 等）与色块底，所以预设两支都按
 * 文字级收口。
 */

/** 显示预设标识（值即偏好字段的合法取值，可原样落 localStorage） */
export type DisplayPreset = 'normal' | 'eyeCare' | 'mono'

/** 预设覆盖的 CSS 变量（与 `schedule.css` 的 `--sched-*` 领域色一一对应） */
export type SchedVar =
  | '--sched-critical'
  | '--sched-normal'
  | '--sched-link'
  | '--sched-group'
  | '--sched-weekend'
  | '--sched-grid'

/** 变量表：`--sched-*` 变量名 → CSS 色值 */
export type PresetColorMap = Record<SchedVar, string>

/** 预设入参：允许旧偏好/手改 JSON 里出现的任意脏值（由 `normalizePreset` 兜底） */
export type DisplayPresetInput = DisplayPreset | string | null | undefined

export interface DisplayPresetInfo {
  id: DisplayPreset
  /** 中文显示名（菜单项/命令面板） */
  label: string
  /** 一句话说明（菜单副标题/悬停提示） */
  desc: string
}

/**
 * 预设清单（顺序即菜单顺序）：`normal` 是缺省项，不加任何类名，走 `schedule.css`
 * 原令牌；另两个各对应 `display-preset.css` 里一个类。
 */
export const DISPLAY_PRESETS: readonly DisplayPresetInfo[] = [
  {
    id: 'normal',
    label: '常规',
    desc: 'gaea 主题令牌原色（关键红/工作蓝），亮暗两套跟随主题切换',
  },
  {
    id: 'eyeCare',
    label: '护眼',
    desc: '低饱和暖绿系，长时段盯屏不刺眼；关键/正常仍靠明度与轻暖色相区分',
  },
  {
    id: 'mono',
    label: '黑白',
    desc: '纯灰阶高对比，投影、黑白打印与色弱场景都读得清',
  },
]

/** 预设覆盖的变量全序（也是色表键序，测试按它断言三套表键集一致） */
export const SCHED_VARS: readonly SchedVar[] = [
  '--sched-critical',
  '--sched-normal',
  '--sched-link',
  '--sched-group',
  '--sched-weekend',
  '--sched-grid',
]

const PRESET_IDS: readonly string[] = DISPLAY_PRESETS.map((p) => p.id)

/** 预设类名（`normal` 无类名；类名只在本模块与 `display-preset.css` 两处出现） */
const PRESET_CLASS: Record<Exclude<DisplayPreset, 'normal'>, string> = {
  eyeCare: 'sched-preset-eye',
  mono: 'sched-preset-mono',
}

/** 外部值是否是我们认识的预设（未知值一律不认，不做模糊匹配/大小写兼容） */
export function isDisplayPreset(value: unknown): value is DisplayPreset {
  return typeof value === 'string' && PRESET_IDS.includes(value)
}

/** 归一化：脏值/缺省回落 `normal`（旧偏好没有该字段时就是这条路） */
export function normalizePreset(value: DisplayPresetInput): DisplayPreset {
  return isDisplayPreset(value) ? value : 'normal'
}

/**
 * 根元素要叠加的类名：`normal` 返回 `''`（不加类=零影响，`className` 拼串时空串无害）。
 * 调用方挂在进度板块根节点/横道根节点上都行——变量只覆盖 `--sched-*`，不会漏到别处。
 */
export function presetClass(preset: DisplayPresetInput): string {
  const id = normalizePreset(preset)
  return id === 'normal' ? '' : PRESET_CLASS[id]
}

/**
 * 常规档 · 屏幕深底值 = `schedule.css` 的 `:root` 缺省档**原值**
 * （本模块只做映射，不重新配色；`displayPreset.test.ts` 用 schedule.css 逐值反查，
 * 防有人改基座色而预设表悄悄漂移）。
 */
const NORMAL_DARK: PresetColorMap = {
  '--sched-critical': 'rgb(240, 72, 72)',
  '--sched-normal': 'rgb(77, 141, 240)',
  '--sched-link': 'rgb(139, 152, 171)',
  '--sched-group': 'rgb(213, 218, 226)',
  '--sched-weekend': 'rgba(148, 163, 184, 0.14)',
  '--sched-grid': 'rgba(148, 163, 184, 0.16)',
}

/** 常规档 · 纸面白底值 = `schedule.css` 的 `:root[data-hl="light"]` 档原值 */
const NORMAL_LIGHT: PresetColorMap = {
  '--sched-critical': 'rgb(224, 32, 32)',
  '--sched-normal': 'rgb(30, 111, 217)',
  '--sched-link': 'rgb(148, 163, 184)',
  '--sched-group': 'rgb(31, 41, 55)',
  '--sched-weekend': 'rgba(148, 163, 184, 0.14)',
  '--sched-grid': 'rgba(148, 163, 184, 0.16)',
}

/**
 * 护眼档 · 深底（屏上缺省档）：低饱和暖绿系，HSV 饱和度逐键低于常规档
 * （关键 0.405<0.700、正常 0.276<0.679、连线 0.136<0.187、汇总 0.041<0.058）。
 * 对暗容器 rgb(31, 32, 52) 实测 6.60 / 6.27 / 6.97 / 9.08 : 1，全部 ≥4.5:1。
 */
const EYE_DARK: PresetColorMap = {
  '--sched-critical': 'rgb(222, 148, 132)',
  '--sched-normal': 'rgb(126, 174, 126)',
  '--sched-link': 'rgb(160, 176, 152)',
  '--sched-group': 'rgb(191, 197, 189)',
  '--sched-weekend': 'rgba(150, 170, 138, 0.14)',
  '--sched-grid': 'rgba(150, 170, 138, 0.18)',
}

/**
 * 护眼档 · 白底（导出/亮色主题）：同色相降明度，对亮容器 rgb(241, 245, 249)
 * 实测 5.69 / 7.01 / 4.91 / 6.93 : 1；对纯白 6.23 / 7.68 / 5.38 / 7.59 : 1。
 */
const EYE_LIGHT: PresetColorMap = {
  '--sched-critical': 'rgb(138, 82, 68)',
  '--sched-normal': 'rgb(52, 92, 56)',
  '--sched-link': 'rgb(98, 110, 92)',
  '--sched-group': 'rgb(74, 88, 66)',
  '--sched-weekend': 'rgba(88, 108, 80, 0.12)',
  '--sched-grid': 'rgba(88, 108, 80, 0.16)',
}

/**
 * 黑白档 · 深底（屏上缺省档）：纯灰阶（R=G=B），明度阶梯承载层级而非色相。
 * 对暗容器 rgb(31, 32, 52) 实测 12.33 / 9.16 / 5.27 / 7.53 : 1，关键最亮。
 */
const MONO_DARK: PresetColorMap = {
  '--sched-critical': 'rgb(226, 226, 226)',
  '--sched-normal': 'rgb(196, 196, 196)',
  '--sched-link': 'rgb(148, 148, 148)',
  '--sched-group': 'rgb(178, 178, 178)',
  '--sched-weekend': 'rgba(200, 200, 200, 0.10)',
  '--sched-grid': 'rgba(200, 200, 200, 0.14)',
}

/**
 * 黑白档 · 白底（导出/亮色主题）：纯灰阶，关键最深（16.21/10.07/5.24/8.62 : 1
 * 对亮容器；对纯白 17.76/11.03/5.74/9.44 : 1），黑白打印机上灰阶可辨。
 */
const MONO_LIGHT: PresetColorMap = {
  '--sched-critical': 'rgb(24, 24, 24)',
  '--sched-normal': 'rgb(60, 60, 60)',
  '--sched-link': 'rgb(102, 102, 102)',
  '--sched-group': 'rgb(70, 70, 70)',
  '--sched-weekend': 'rgba(0, 0, 0, 0.06)',
  '--sched-grid': 'rgba(0, 0, 0, 0.10)',
}

/** 屏幕深底档色表（每次返回新对象，调用方改了也不污染模块内常量） */
const DARK_MAP: Record<DisplayPreset, PresetColorMap> = {
  normal: NORMAL_DARK,
  eyeCare: EYE_DARK,
  mono: MONO_DARK,
}

/** 纸面白底档色表（导出 PNG/PDF、亮色主题） */
const LIGHT_MAP: Record<DisplayPreset, PresetColorMap> = {
  normal: NORMAL_LIGHT,
  eyeCare: EYE_LIGHT,
  mono: MONO_LIGHT,
}

/**
 * 预设对 `--sched-*` 的覆盖值（屏幕深底档）。
 * 只输出变量表、不生成 DOM/class：接线由页面根元素负责（见 `presetClass`）。
 */
export function presetColorMap(preset: DisplayPresetInput): PresetColorMap {
  return { ...DARK_MAP[normalizePreset(preset)] }
}

/**
 * 导出/图面用打印安全色（纸面白底档）。
 * 关键点：黑白=纯灰阶、护眼=低饱和，且键集与 `presetColorMap` 完全一致，
 * PNG/PDF 与屏上预设不会出现「导出变彩色」的割裂。
 * 常规档沿用 `schedule.css` 亮档原值（与屏上亮色一致，零漂移）；其中连线色
 * rgb(148, 163, 184) 是既有弱化线基线（2.56:1，只作虚线不做前景文字）。
 */
export function toSvgPalette(preset: DisplayPresetInput): PresetColorMap {
  return { ...LIGHT_MAP[normalizePreset(preset)] }
}
