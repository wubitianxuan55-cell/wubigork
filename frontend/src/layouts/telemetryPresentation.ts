export type TelemetryTone = 'idle' | 'normal' | 'watch' | 'hot'

const TONE_RANK: Record<TelemetryTone, number> = {
  idle: 0,
  normal: 1,
  watch: 2,
  hot: 3,
}

/** 将资源占用映射为稳定的展示状态；未知采样保持 idle。 */
export function telemetryTone(
  value: number | null | undefined,
  warnAt = 70,
  hotAt = 85,
): TelemetryTone {
  if (value == null || !Number.isFinite(value)) return 'idle'
  if (value >= hotAt) return 'hot'
  if (value >= warnAt) return 'watch'
  return 'normal'
}

/** 汇总所有已采样指标，取最高负载作为系统状态。 */
export function telemetrySystemTone(
  values: Array<number | null | undefined>,
  warnAt = 70,
  hotAt = 85,
): TelemetryTone {
  return telemetryWorstTone(values.map((value) => telemetryTone(value, warnAt, hotAt)))
}

export function telemetryWorstTone(tones: TelemetryTone[]): TelemetryTone {
  return tones.reduce<TelemetryTone>(
    (worst, tone) => (TONE_RANK[tone] > TONE_RANK[worst] ? tone : worst),
    'idle',
  )
}

export function telemetryMeterWidth(value: number | null | undefined): number {
  if (value == null || !Number.isFinite(value)) return 0
  return Math.max(0, Math.min(100, value))
}
