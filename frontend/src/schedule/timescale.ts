/**
 * schedule/timescale.ts — 时标刻度内核（纯函数；零 React、零 DOM、零 store 依赖）
 *
 * 权威来源（逆向报告，逐条对齐）：
 *  · `进度计划/gaea-schedule-zpert8-timescale-re-20261114.md`
 *      §2   六档时标 fov 1..6 与间隔集
 *      §3.1 单位编码（5 个选项 / 6 个编码值 / 8..10 三个派生值）
 *      §3.2 间隔索引钳位（预设回填）
 *      §4.2 `.rdata` 常量表（0.08 / 0.10 / 0.12 / 0.24 / 0.40 / 2.4 / 5 / 7 / 15 / 0.5 …）
 *      §4.3 自动联动分支（FUN_14069a4d0）与统一收尾 ×0.5
 *      §5.2 ZoomTime 字段（fov/step/min_step/width/scale/fv[6]）
 *      §7.1 双语标尺名 + 四档粒度串池（按日期/按周/按月/按年）
 *      §7.3 刻度线疏密四选一（small/big/project/none）
 *      §8   对 gaea 的落地口径（TimeScaleConfig、间隔集硬编码、自动联动、
 *           像素换算唯一出口 `pos[i] = Σ step[k] × scale × 2.4`、四档粒度资源键）
 *  · `进度计划/gaea-schedule-zpert8-uiux-port-plan-20261114.md` §W4（双标尺 +
 *    工程标尺/进度标尺）；`...uiux-re-20261114.md` §5 / §18.2（画布双标尺：月 / W 周）。
 *
 * 设计边界：
 *  - 本模块只做"刻度数学 + 分段标签"，不碰像素布局、不读 store、不引 antd；
 *  - 横道图当前的可视日宽仍是 `dayW`（px/自然日），本模块为「斑马口径时标」提供
 *    统一的换算与档位内核，供后续 5 个视图模式共用同一出口（RE §8.4 的"避免各视图漂移"）；
 *  - 双标尺的两行结构与高度常量也从这里导出，`GanttView` 表头、画布顶部偏移、
 *    `GanttTimeline` 渲染共用同一份常量（禁止各处再写字面量 40）。
 */

/* ── 0. 枚举与配置（RE §2 / §3.1 / §5.2 / §8.1）────────────────── */

/** 六档时标：1 分钟 / 2 小时 / 3 自然日族 / 4 工作周 / 5 工程月 / 6 工作日（RE §2）。 */
export type Fov = 1 | 2 | 3 | 4 | 5 | 6

/**
 * 单位编码（RE §3.1，`*` 标注的是"编码有 6 个值但下拉只有 5 项"的那一条）：
 * `1` 年-月 · `2` 月-天 · `3` 工程周 · `4` 工程月 · `6` 工程月（fov=6 分支，与 4 同族）·
 * `8` 10 日 · `9` 15 日 · `10` 月（月-天单位内的三个派生档）。
 */
export type Unit = 1 | 2 | 3 | 4 | 6 | 8 | 9 | 10

/** 刻度线疏密（RE §7.3 四选一：画小/大刻度线、画工程日历线、不画刻度线）。 */
export type LineDensity = 'small' | 'big' | 'project' | 'none'

/** `types.ts` `SchedNetworkStyle.tickLines` 的取值（斑马三档，缺省 small）。 */
export type TickLineStyle = 'none' | 'small' | 'large'

/** 局部缩放步长（RE §5.3 `0x344`，界面 1..5）。 */
export type LocalZoomStep = 1 | 2 | 3 | 4 | 5

/** RE §8.1 的 `TimeScaleConfig` 子集：本条记录 = 一个"时标配置"。 */
export interface TimeScaleConfig {
  /** 自动档位选择器（RE §2） */
  fov: Fov
  /** 1 刻度 = step × 单位（RE §5.2 +0x04） */
  step: number
  /** 界面"（1~30）"（RE §5.2 +0x0c） */
  minStep: number
  /** 刻度比例（工作日当量/像素，RE §5.2 +0x18，钳位 0.04..0.4） */
  scale: number
  /** 六档比例副本（RE §5.2 +0x64 `fv[6]`），长度恒为 6 */
  fv: number[]
  /** 工程标尺（RE §7.2 `doc+0x58`） */
  showProjectRuler: boolean
  /** 周期行（RE §7.2 `doc+0x59`） */
  showPeriodRow: boolean
  /** 工程日历线（RE §7.2 `doc+0x5a`） */
  showCalendarLine: boolean
  /** 刻度线疏密（RE §7.3） */
  lineDensity: LineDensity
  /** 工程标尺上方刻度单位（`FUN_140698d60` +0x148 下拉：年-月 / 月-天 / 日-时） */
  topUnit: Unit
  topStep: number
  /** 进度标尺上方/下方刻度单位（同函数 +0x118 下拉：工程周 / 工程月） */
  bottomUnit: Unit
  bottomStep: number
  /** 时标刻度局部缩放（RE §9.4：不是全局比例，而是"时间段 → 档位 + 比例"的分段表） */
  localZoom: { unit: Unit; step: LocalZoomStep }[]
}

/* ── 1. 单位 / 间隔集（RE §2、§3.1；硬编码，改单位时先归零再重填）─────── */

/** 单位 → 间隔集（RE §3.1"三个单位各有一套固定集合"；8/9/10 是月-天内的派生单值档）。 */
export const UNIT_INTERVALS: Record<Unit, readonly number[]> = {
  1: [1, 2, 3, 4, 6], // 年-月
  2: [1, 2, 3, 4, 5, 6, 7, 10, 15], // 月-天
  3: [1], // 工程周
  4: [1], // 工程月
  6: [1, 2, 3, 6], // 工程月（fov=6 分支，`FUN_14069c2c0` 分支 2）
  8: [10], // 月-天口径的"10 日"
  9: [15], // 月-天口径的"15 日"
  10: [1], // 月-天口径的"月"
}

/** 六档 → 间隔集（RE §2 表格；fov=3 由月-天单位内的 2/4/5 三档给出）。 */
export const FOV_INTERVALS: Record<Fov, readonly number[]> = {
  1: [1, 2, 3, 4, 6],
  2: [1, 2, 3, 4, 5, 6, 7, 10, 15],
  3: [2, 4, 5],
  4: [1],
  5: [1],
  6: [1, 2, 3, 6],
}

/** 单位 → 自动档位（RE §2 + §3.1：自然日历族落 fov=3，工作周/工程月/工作日各自成档）。 */
export const UNIT_FOV: Record<Unit, Fov> = { 1: 3, 2: 3, 3: 4, 4: 5, 6: 6, 8: 3, 9: 3, 10: 3 }

/** 单位显示名（`FUN_140698d60` 资源池 5 串按行裁剪，RE §3.1 表）。 */
export const UNIT_LABELS: Record<Unit, string> = {
  1: '年-月',
  2: '月-天',
  3: '工程周',
  4: '工程月',
  6: '工程月',
  8: '10 日',
  9: '15 日',
  10: '月',
}

/** 四档粒度串池（RE §7.1 `0x14085de98..deb0`，`FUN_1402c56c0` 按 `doc+0x4c`/`doc+0x50` 取项）。 */
export const RULER_GRAIN_LABELS = ['按日期', '按周', '按月', '按年'] as const

/** 双标尺显示名（RE §7.1 中文分支：`doc+0x58` / `doc+0x59`）。 */
export const RULER_NAMES = { project: '工程标尺', progress: '进度标尺' } as const

/** 四档粒度标签（RE §8.5：按"日期/周/月/年"做资源键，不要按 6 档各写一套文案）。 */
export function rulerGrainLabel(grain: 0 | 1 | 2 | 3): string {
  return RULER_GRAIN_LABELS[grain]
}

/** 单位 → 间隔集（未知编码回落月-天集合：这是斑马唯一覆盖 1..15 的集合）。 */
export function intervalsForUnit(unit: Unit): readonly number[] {
  return UNIT_INTERVALS[unit] ?? UNIT_INTERVALS[2]
}

/** 六档 → 间隔集。 */
export function intervalsForFov(fov: Fov): readonly number[] {
  return FOV_INTERVALS[fov] ?? FOV_INTERVALS[2]
}

/**
 * 间隔索引钳位（RE §3.2）：集合内原样返回；越界钳到边界（偏小优先，
 * 与 `sel = step-1` / `step<4 ? sel=step-1 : sel=3` 的索引语义同向）。
 */
export function clampStepToSet(set: readonly number[], step: number): number {
  if (set.length === 0) return 1
  if (set.includes(step)) return step
  if (step < set[0]) return set[0]
  for (let i = set.length - 1; i >= 0; i--) if (set[i] <= step) return set[i]
  return set[0]
}

/** 单位口径的间隔钳位（RE §3.2：三个下拉各自的固定集合）。 */
export function clampInterval(unit: Unit, step: number): number {
  return clampStepToSet(intervalsForUnit(unit), step)
}

/** 间隔在集合中的 0 基索引（RE §4.3 fov=2 分支的 `stepIdx`）。 */
export function intervalIndex(unit: Unit, step: number): number {
  const set = intervalsForUnit(unit)
  const clamped = clampStepToSet(set, step)
  return Math.max(0, set.indexOf(clamped))
}

/** 改单位第一步：`step = 0`（RE §1「单位变更 → step=0 → 重填间隔 → 置已改标志」）。 */
export function resetStep(): number {
  return 0
}

/** 改单位第二步：按新单位集合回填首项（"归零 + 重填 + 首项"）。 */
export function refillStep(unit: Unit): number {
  return intervalsForUnit(unit)[0] ?? 1
}

/**
 * 改单位（RE §1 的 `FUN_14069b8f0` 信号链）：`resetStep()` 归零 → `refillStep(unit)` 重填
 * → fov 随口径重算（RE §2/§3.1）。归零是斑马对话框的中间态，落库前必被首项覆盖，
 * 所以这里只保留"重填"结果；中间态由 {@link resetStep} 单独暴露给需要复刻该序列的调用方。
 */
export function setUnit(cfg: TimeScaleConfig, which: 'top' | 'bottom', unit: Unit): TimeScaleConfig {
  const step = refillStep(unit)
  const fov = UNIT_FOV[unit] ?? cfg.fov
  return which === 'top'
    ? { ...cfg, topUnit: unit, topStep: step, fov, step }
    : { ...cfg, bottomUnit: unit, bottomStep: step, fov, step }
}

/* ── 2. 比例常量与自动联动（RE §4.2 常量表 / §4.3 分支）────────────── */

/** 比例下限（RE §4.2 `1408bd210` = 0.08 密度下限 → 界面 scale 下界 0.04）。 */
export const SCALE_MIN = 0.04
/** 比例上限（RE §4.2 `1408bd230` = 0.40）。 */
export const SCALE_MAX = 0.4
/** 每日工作小时（RE §4.2 `1408bd238` = 2.4）——像素换算的唯一乘数。 */
export const WORK_HOURS_PER_DAY = 2.4
/** 统一收尾系数（RE §4.2 `1407f8ba0` = 0.5；漏掉则 1 刻度像素宽整体翻倍）。 */
export const SCALE_FINAL_COEF = 0.5
/** 密度下限（RE §4.2 `1408bd210` = 0.08，≈1.92 工作小时/像素）。 */
export const DENSITY_FLOOR = 0.08
/** 1 分钟档基准（RE §4.2 `1408bd218` = 0.10）。 */
export const MINUTE_BASE = 0.1
/** 小时档基准（RE §4.2 `1408bd220` = 0.12）。 */
export const HOUR_BASE = 0.12
/** 小时档第二基准（RE §4.2 `1408bd228` = 0.24）。 */
export const HOUR_BASE_2 = 0.24
/** 小时档第三基准（RE §4.2 `1408bd230` = 0.40）。 */
export const HOUR_BASE_3 = 0.4
/** 半月天数（RE §4.2 `1408bd250` = 15，"2.4/15" 是六档分支的公共因子）。 */
export const HALF_MONTH_DAYS = 15
/** 一周 7 天（RE §4.2 `1408bd248`）。 */
export const WEEK_DAYS = 7
/** 旬（RE §4.2 `140818470` = 10）。 */
export const DECADE_DAYS = 10
/** 一年 ≈ 5 个 10 日（RE §4.2 `1408bd240`）。 */
export const FIVE_PER_YEAR = 5

/** 六档公共因子 `2.4/15`（RE §4.3 里反复出现的 `d`）。 */
const D15 = WORK_HOURS_PER_DAY / HALF_MONTH_DAYS

/** 比例钳位到 RE §5.2 的 0.04..0.4（§4.3 的分支会算出负值/超界值，必须收口）。 */
export function clampScale(scale: number): number {
  if (!Number.isFinite(scale)) return SCALE_MIN
  return Math.min(SCALE_MAX, Math.max(SCALE_MIN, scale))
}

/**
 * 自动联动的"原始 scale"（`FUN_14069a4d0`，RE §4.3）——**含**末尾 ×0.5，**不含**钳位，
 * 便于单测看清分支原值。返回 null = 该档在 RE 里没有 scale 赋值分支（fov 3/6）。
 *
 * 逐字复刻说明：fov=2 首档与 fov=4/5 的 `1 - (2.4/15)/0.08` 在常量取值下是负数
 * （1-2=-1），按 RE §4.3 原样保留、由 {@link clampScale} 收到下限 0.04，不做"修正"。
 */
export function rawAutoScale(fov: Fov, step: number): number | null {
  switch (fov) {
    case 1: {
      const base = step < 4 ? MINUTE_BASE / WORK_HOURS_PER_DAY / (step + 1) : MINUTE_BASE / WORK_HOURS_PER_DAY / FIVE_PER_YEAR
      return (base + D15) * SCALE_FINAL_COEF
    }
    case 2: {
      const idx = intervalIndex(2, step)
      let s: number
      if (idx === 0) s = 1 - D15 / DENSITY_FLOOR
      else if (idx === 1) s = D15 / HOUR_BASE + D15 / DENSITY_FLOOR
      else if (idx === 2) s = D15 / HOUR_BASE_2 + D15 / HOUR_BASE
      else if (idx === 3) s = D15 / HOUR_BASE_2 + D15 / HOUR_BASE_3
      else if (idx < 7) s = Math.trunc(D15 / (idx + 1)) + 1
      else if (idx === 7) s = D15 / WEEK_DAYS + D15 / DECADE_DAYS
      else s = D15 / HALF_MONTH_DAYS + D15 / DECADE_DAYS
      return s * SCALE_FINAL_COEF
    }
    case 4:
    case 5:
      return (1 - D15 / DENSITY_FLOOR) * SCALE_FINAL_COEF
    case 3:
    case 6:
      return null
  }
}

/**
 * fov=6 的密度分档（RE §4.3：`d<=0.08 → 1`、`<=0.12 → 2`、`<=0.24 → 3`、`>0.24 → 6`）。
 * gaea 里它当"密度 → 首档间隔"的推荐值用，不覆盖用户 step（RE 该分支只做
 * `step = max(step, min_step)`，没有 scale 赋值）。
 */
export function fov6StepFromDensity(density: number = D15): number {
  if (density <= DENSITY_FLOOR) return 1
  if (density <= HOUR_BASE) return 2
  if (density <= HOUR_BASE_2) return 3
  return 6
}

export interface AutoLinkInput {
  fov: Fov
  step: number
  minStep: number
  scale: number
}

export interface AutoLinkResult {
  step: number
  scale: number
  /** 本档有没有重算 scale（RE §4.3 里 fov 3/6 不改 scale） */
  scaleChanged: boolean
}

/** 自动联动（RE §4.3 + §8.3）：返回钳位后的 step 与 clamp 到 0.04..0.4 的 scale。 */
export function computeAutoLink(input: AutoLinkInput): AutoLinkResult {
  const { fov, step, minStep, scale } = input
  const raw = rawAutoScale(fov, fov === 6 ? Math.max(step, minStep) : step)
  const nextStep = fov === 6
    ? clampStepToSet(FOV_INTERVALS[6], Math.max(step, minStep))
    : clampStepToSet(intervalsForFov(fov), Math.max(1, step))
  if (raw === null) return { step: nextStep, scale: clampScale(scale), scaleChanged: false }
  return { step: nextStep, scale: clampScale(raw), scaleChanged: true }
}

/** 对整条配置跑一次自动联动（对话框"确定"时的 `FUN_14069a4d0` 位置）。 */
export function applyAutoLink(cfg: TimeScaleConfig): TimeScaleConfig {
  const next = computeAutoLink({ fov: cfg.fov, step: cfg.step, minStep: cfg.minStep, scale: cfg.scale })
  const fv = [...cfg.fv]
  if (next.scaleChanged) fv[cfg.fov - 1] = next.scale
  return { ...cfg, step: next.step, scale: next.scale, fv }
}

/** 六档缺省比例（由 §4.3 反算的 `fv[6]`，RE §5.2 +0x64 与 §5.3 0x34c..0x3e4 副本）。 */
export const DEFAULT_FV: readonly number[] = ([1, 2, 3, 4, 5, 6] as Fov[]).map((fov) =>
  computeAutoLink({ fov, step: intervalsForFov(fov)[0], minStep: 1, scale: SCALE_MIN }).scale,
)

/** 缺省配置（工程标尺=年-月、进度标尺=工程周；斑马首装口径）。 */
export function defaultTimeScale(): TimeScaleConfig {
  return {
    fov: 3,
    step: 2,
    minStep: 1,
    scale: DEFAULT_FV[2],
    fv: [...DEFAULT_FV],
    showProjectRuler: true,
    showPeriodRow: true,
    showCalendarLine: false,
    lineDensity: 'small',
    topUnit: 1,
    topStep: 1,
    bottomUnit: 3,
    bottomStep: 1,
    localZoom: [{ unit: 3, step: 1 }],
  }
}

/* ── 3. 像素换算唯一出口（RE §6 / §8.4）──────────────────────────── */

/**
 * 1 刻度像素宽 = `scale × 2.4`（RE §4.3 语义段 + §8.4）。
 * 注：RE §6 的 `FUN_1402c4b40` 循环体是 `step[i] × scale`，§8.4 的落地口径补上
 * `×2.4`（工作日当量 → 像素）。gaea 按 §8.4 执行，避免"1 刻度宽"与"1 像素代表"
 * 两个量纲在同一出口混用。
 */
export function tickWidthPx(scale: number): number {
  return scale * WORK_HOURS_PER_DAY
}

/** 单段步长的像素宽：`step × scale × 2.4`。 */
export function stepWidthPx(step: number, scale: number): number {
  return step * tickWidthPx(scale)
}

/**
 * `pos[i] = Σ step[k] × scale × 2.4`（RE §8.4）。
 * 逐元素取整口径同 `FUN_1402c4b40`：先写 `(int)acc` 再累加原始量（整数截断，非四舍五入）。
 */
export function rulerPositions(steps: readonly number[], scale: number): number[] {
  const out: number[] = []
  let acc = 0
  for (const s of steps) {
    out.push(Math.trunc(acc))
    acc += s * tickWidthPx(scale)
  }
  return out
}

/** 整张刻度表的总像素宽（未取整，供布局用）。 */
export function rulerTotalWidth(steps: readonly number[], scale: number): number {
  let acc = 0
  for (const s of steps) acc += s * tickWidthPx(scale)
  return acc
}

/* ── 4. 双标尺几何常量（W4：两行常驻，高度单一来源）───────────────── */

/** 单行标尺高度（px）——工程标尺 / 进度标尺各一行。 */
export const GANTT_RULER_ROW_H = 20
/** 标尺行数（常驻两行：上行工程标尺、下行进度标尺）。 */
export const GANTT_RULER_ROWS = 2
/** 标尺总高度 = 表格表头高 = 画布顶部偏移（**唯一来源**，禁止各处再写字面量）。 */
export const GANTT_RULER_H = GANTT_RULER_ROW_H * GANTT_RULER_ROWS

/** 窄窗阈值：`dayW < 8` 时工程标尺只画月段文本、进度标尺只画周序号（W4 降级阶梯）。 */
export const NARROW_RULER_DAY_W = 8
/** 周刻度层阈值（与 `gantt/ganttUtil.WEEK_TIER_BELOW` 同值；调用方显式传 `weekTier` 保持单一判定源）。 */
export const WEEK_TIER_BELOW_RULER = 12
/** 周段标签最小像素宽（原 `GanttTimeline` 的 `w >= 12`）。 */
/** 窄窗里「段宽够就逐周标」的门槛（低于它按每 4 周标，防叠字） */
export const NARROW_WEEKLY_LABEL_PX = 24
export const MIN_WEEK_LABEL_PX = 12
/** 日层"日号+星期"并列的最小日宽（更窄只留日号，星期退到 title）。 */
export const WEEKDAY_LABEL_MIN_DAY_W = 22

/** 星期标签（与 `gantt/ganttUtil.WEEKDAY_LABELS` 同源；本模块不反向依赖 gantt/ 子目录）。 */
export const WEEKDAY_LABELS = ['日', '一', '二', '三', '四', '五', '六'] as const

/* ── 5. 刻度线疏密（RE §7.3 + `types.ts` tickLines 三档）──────────── */

/** 斑马 `tickLines`（不画/画小/画大）→ 四值密度；缺省 `small`（RE §7.3 首项）。 */
export function resolveLineDensity(
  style?: { tickLines?: TickLineStyle } | null,
  showCalendarLine = false,
): LineDensity {
  const tick = style?.tickLines ?? 'small'
  if (tick === 'none') return 'none'
  if (tick === 'large') return 'big'
  // 「画工程日历线」（RE §7.3 第三项）单独由 showCalendarLine 承载：
  // gaea 的 `SchedNetworkStyle` 目前没有该字段（types.ts 禁改），由调用方传参预留。
  return showCalendarLine ? 'project' : 'small'
}

/* ── 6. 分段标签构造（年-月跨列段 / 周·星期段 / 刻度线）────────────── */

/** 自然日列（与 `GanttView` 的 `colDates` 同形；`GanttOverlay` 复用该类型）。 */
export interface RulerColumn {
  d: Date
  iso: string
  offWork: boolean
}

export interface MonthRun {
  y: number
  m: number
  count: number
}

export interface YearRun {
  y: number
  count: number
}

export interface WeekRun {
  startIdx: number
  count: number
  /** 周序号（自首列起每 7 列一段，1 基；沿用 gaea 既有口径，非 ISO 周） */
  weekNo: number
  label: string
  title: string
}

/** 年-月跨列段（上行工程标尺主带；连续同月合成一段）。 */
export function buildMonthRuns(cols: readonly RulerColumn[]): MonthRun[] {
  const runs: MonthRun[] = []
  for (const c of cols) {
    const y = c.d.getUTCFullYear()
    const m = c.d.getUTCMonth()
    const last = runs[runs.length - 1]
    if (last && last.y === y && last.m === m) last.count++
    else runs.push({ y, m, count: 1 })
  }
  return runs
}

/** 年跨列段（大刻度线/跨年标注用）。 */
export function buildYearRuns(cols: readonly RulerColumn[]): YearRun[] {
  const runs: YearRun[] = []
  for (const c of cols) {
    const y = c.d.getUTCFullYear()
    const last = runs[runs.length - 1]
    if (last && last.y === y) last.count++
    else runs.push({ y, count: 1 })
  }
  return runs
}

/**
 * 周段（进度标尺）：连续同自然周（周一始）合成一段，跨月/跨年不断开。
 * 标签沿用既有口径——周序号步长 4（W1/W5/W9…）+ 段宽 ≥12px；`narrow` 时只打周序号
 * （无 4 周步长），因为窄窗没有别的东西可写（W4 降级阶梯）。
 */
export function buildWeekRuns(cols: readonly RulerColumn[], dayW: number, narrow = false): WeekRun[] {
  const runs: WeekRun[] = []
  for (let i = 0; i < cols.length; i++) {
    const c = cols[i]
    const last = runs[runs.length - 1]
    if (last && c.d.getUTCDay() !== 1) {
      last.count++
      continue
    }
    // 序号按“第几个周段”递增，不用 floor(i/7)：首周若不是周一起始，
    // 按列号算会把首段与下一个周一段都算成 W1（相邻重复标签，实页走查实锤）
    runs.push({ startIdx: i, count: 1, weekNo: runs.length + 1, label: '', title: '' })
  }
  const labeled = new Set<number>()
  for (const r of runs) {
    const start = cols[r.startIdx].d
    const end = cols[Math.min(r.startIdx + r.count - 1, cols.length - 1)].d
    const w = r.count * dayW
    // 窄窗分层：段宽够（≥24px）逐周标；只有 12~24px 时退回「每 4 周标一次」，
    // 否则 W1..W20 会在 18px 宽里互相叠字（实页走查实锤）
    const shouldLabel = narrow
      ? (w >= NARROW_WEEKLY_LABEL_PX || r.weekNo === 1 || r.weekNo % 4 === 1) && w >= MIN_WEEK_LABEL_PX
      : (r.weekNo === 1 || r.weekNo % 4 === 1) && !labeled.has(r.weekNo) && w >= MIN_WEEK_LABEL_PX
    r.label = shouldLabel ? `W${r.weekNo}` : ''
    if (shouldLabel) labeled.add(r.weekNo)
    r.title = `第 ${r.weekNo} 周（${start.getUTCMonth() + 1}/${start.getUTCDate()} ~ ${end.getUTCMonth() + 1}/${end.getUTCDate()}）`
  }
  return runs
}

/** 月段文本降级阶梯：≥76 年月 → ≥44 月 → (非窄窗) ≥14 数字 → 空。 */
export function monthLabelForWidth(y: number, m: number, widthPx: number, narrow = false): string {
  if (widthPx >= 76) return `${y}年${m + 1}月`
  if (widthPx >= 44) return `${m + 1}月`
  if (narrow) return ''
  return widthPx >= 14 ? String(m + 1) : ''
}

export interface RulerSegment {
  key: string
  startIdx: number
  count: number
  widthPx: number
  label: string
  title: string
}

export interface RulerDayCell {
  key: string
  widthPx: number
  /** 日号文本（窄日宽留空，沿用既有 2 列一标的节奏） */
  label: string
  /** 星期字母（日宽 ≥22 才出，否则退到 title） */
  weekday: string
  title: string
  offWork: boolean
}

export interface RulerTickMark {
  key: string
  xPx: number
  /** `year`/`month` = 年-月边界；`week` = 周边界；`day` = 日边界 */
  tier: 'year' | 'month' | 'week' | 'day'
}

export interface RulerModelInput {
  colDates: readonly RulerColumn[]
  dayW: number
  /** 刻度线疏密（`resolveLineDensity(project.networkStyle)`） */
  lineDensity?: LineDensity
  /** 下行是否周层（缺省按 `WEEK_TIER_BELOW_RULER` 判定；`GanttView` 显式传入保持单一判定源） */
  weekTier?: boolean
  /** 今日列（自然日列偏移；null=不画） */
  todayCol?: number | null
  /** 目标竣工列（自然日列偏移；线画在该列右缘） */
  deadlineCol?: number | null
  showProjectRuler?: boolean
  showPeriodRow?: boolean
}

export interface RulerModel {
  /** 总高（常驻两行，= {@link GANTT_RULER_H}） */
  heightPx: number
  rowHPx: number
  /** 窄窗（dayW < {@link NARROW_RULER_DAY_W}） */
  narrow: boolean
  lineDensity: LineDensity
  /** 上行：工程标尺 = 年-月跨列段 + 日/周刻度线 */
  engineering: {
    show: boolean
    segments: RulerSegment[]
    ticks: RulerTickMark[]
    /** 窄窗只画月段文本 → ticks 为空、段标签不做单数字降级 */
    narrow: boolean
  }
  /** 下行：进度标尺 = 周/星期 */
  progress: {
    show: boolean
    tier: 'day' | 'week'
    weeks: RulerSegment[]
    days: RulerDayCell[]
  }
  /** 今日线 x（px；null=不画）——穿透两行标尺 */
  todayX: number | null
  /** 目标竣工线 x（px；null=不画） */
  deadlineX: number | null
}

/** 刻度线（疏密：small=月+周密线；big=年+月疏粗线；project=月界工程日历线；none=无）。 */
function buildTickMarks(
  months: readonly MonthRun[],
  years: readonly YearRun[],
  weeks: readonly WeekRun[],
  dayW: number,
  density: LineDensity,
  narrow: boolean,
): RulerTickMark[] {
  if (density === 'none' || narrow) return []
  const marks: RulerTickMark[] = []
  const seen = new Set<string>()
  const push = (xPx: number, tier: RulerTickMark['tier']) => {
    const key = `${tier}:${xPx}`
    if (xPx <= 0 || seen.has(key)) return
    seen.add(key)
    marks.push({ key, xPx, tier })
  }
  let mStart = 0
  for (const m of months) {
    push(mStart * dayW, 'month')
    mStart += m.count
  }
  if (density === 'big' || density === 'project') {
    let yStart = 0
    for (const y of years) {
      push(yStart * dayW, 'year')
      yStart += y.count
    }
    return marks
  }
  // small：周边界也画（"画小刻度线"= 最密）
  for (const w of weeks) push(w.startIdx * dayW, 'week')
  return marks
}

/**
 * 双标尺模型（组件零计算：`GanttTimeline` 只按此结构渲染）。
 * 行归属：上行=年-月跨列段 + 日/周刻度线；下行=周段（周层）或逐日列（日层）。
 */
export function buildRulerModel(input: RulerModelInput): RulerModel {
  const { colDates, dayW } = input
  const density = input.lineDensity ?? 'small'
  const narrow = dayW < NARROW_RULER_DAY_W
  const months = buildMonthRuns(colDates)
  const years = buildYearRuns(colDates)
  const tier: 'day' | 'week' = (input.weekTier ?? dayW < WEEK_TIER_BELOW_RULER) ? 'week' : 'day'
  const weeks = buildWeekRuns(colDates, dayW, narrow && tier === 'week')

  const segments: RulerSegment[] = []
  let start = 0
  for (const m of months) {
    const widthPx = m.count * dayW
    segments.push({
      key: `m:${m.y}-${m.m}`,
      startIdx: start,
      count: m.count,
      widthPx,
      label: monthLabelForWidth(m.y, m.m, widthPx, narrow),
      title: `${m.y}年${m.m + 1}月`,
    })
    start += m.count
  }

  const days: RulerDayCell[] = colDates.map((c, i) => {
    const weekday = WEEKDAY_LABELS[c.d.getUTCDay()]
    return {
      key: `d:${i}:${c.iso}`,
      widthPx: dayW,
      label: dayW >= 14 || (dayW >= 10 && i % 2 === 0) ? String(c.d.getUTCDate()) : '',
      weekday: dayW >= WEEKDAY_LABEL_MIN_DAY_W ? weekday : '',
      title: c.offWork ? '非工作日' : `周${weekday}`,
      offWork: c.offWork,
    }
  })

  const weekSegments: RulerSegment[] = weeks.map((w) => ({
    key: `w:${w.startIdx}`,
    startIdx: w.startIdx,
    count: w.count,
    widthPx: w.count * dayW,
    label: w.label,
    title: w.title,
  }))

  return {
    heightPx: GANTT_RULER_H,
    rowHPx: GANTT_RULER_ROW_H,
    narrow,
    lineDensity: density,
    engineering: {
      show: input.showProjectRuler ?? true,
      segments,
      ticks: buildTickMarks(months, years, weeks, dayW, density, narrow),
      narrow,
    },
    progress: {
      show: input.showPeriodRow ?? true,
      tier,
      weeks: weekSegments,
      days: tier === 'day' ? days : [],
    },
    todayX: input.todayCol != null && input.todayCol >= 0 ? input.todayCol * dayW : null,
    deadlineX: input.deadlineCol != null && input.deadlineCol >= 0 ? input.deadlineCol * dayW + dayW : null,
  }
}
