import { useEffect, useMemo, useState } from 'react'
import type { CpmResult, SchedProject } from './types'
import { app } from '../gaea/lib/bridge'
import { dcmaAudit, type DcmaThresholds } from './dcma'

/** DcmaView — 6.4 计划质量体检：DCMA 14 点逐项结论（对标 Acumen Fuse 的
 *  体检单形态）。展示 + 阈值覆盖加载：审计在 dcmaAudit 纯函数内完成（同一输入
 *  同一结论）；阈值表是数据资产（v4.226 规范知识出内核）——挂载时尝试读工作区
 *  覆盖文件 .gaea/skills/schedule-dcma/thresholds.json（部分字段可覆盖），缺失
 *  或解析失败静默用内置默认（增强面不挡主功能），数据不足的点诚实显示「不可
 *  评估」+原因，绝不伪造。 */
const THRESHOLDS_OVERRIDE_REL = '.gaea/skills/schedule-dcma/thresholds.json'

export function DcmaView({ project, cpm, dataDate }: { project: SchedProject; cpm: CpmResult; dataDate?: number | null }) {
  const [thresholds, setThresholds] = useState<Partial<DcmaThresholds> | null>(null)

  useEffect(() => {
    let alive = true
    app
      .ReadFile(THRESHOLDS_OVERRIDE_REL)
      .then((f) => {
        if (!alive) return
        try {
          const t = JSON.parse(f.markdown) as Partial<DcmaThresholds>
          if (t && typeof t === 'object') setThresholds(t)
        } catch {
          // 覆盖文件解析失败：保内置默认，不打断体检
        }
      })
      .catch(() => {
        // 无覆盖文件：内置默认
      })
    return () => {
      alive = false
    }
  }, [])

  const report = useMemo(
    () => dcmaAudit(project, cpm, dataDate ?? null, thresholds ?? undefined),
    [project, cpm, dataDate, thresholds],
  )

  const scoreTone =
    report.score === null
      ? 'text-fg-faint'
      : report.score >= 90
        ? 'text-emerald-500'
        : report.score >= 70
          ? 'text-amber-500'
          : 'text-red-400'

  return (
    <div className="h-full overflow-auto px-4 py-3">
      <div className="flex items-baseline gap-3 flex-wrap mb-3">
        <span className="text-fg text-[13px] font-medium">DCMA 14 点计划质量体检</span>
        <span className={`text-[20px] font-semibold tabular-nums ${scoreTone}`} data-testid="dcma-score">
          {report.score === null ? '—' : `${report.score} 分`}
        </span>
        <span className="text-fg-faint text-[11px]">
          可评估 {report.evaluable} · 通过 {report.passed} · 未过 {report.failed}
          {report.checks.length - report.evaluable > 0 && ` · 不可评估 ${report.checks.length - report.evaluable}`}
        </span>
      </div>
      <div className="space-y-1.5 max-w-[860px]">
        {report.checks.map((c) => (
          <div
            key={c.id}
            data-testid={`dcma-check-${c.id}`}
            className="flex items-start gap-2.5 px-3 py-2 rounded-lg border border-border-soft bg-bg-soft/40"
          >
            <span className="shrink-0 w-6 text-right font-mono text-[11px] text-fg-faint">{c.id}</span>
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 flex-wrap">
                <span className="text-fg text-[12.5px] font-medium">{c.name}</span>
                <span className="text-fg-faint text-[10.5px]">{c.threshold}</span>
              </div>
              {c.na ? (
                <div className="text-fg-faint text-[11px] mt-0.5">不可评估：{c.na}</div>
              ) : (
                <>
                  {c.offenders.length > 0 && (
                    <div className="text-fg-dim text-[11px] mt-0.5 leading-relaxed">
                      {c.offenders.join('；')}
                      {c.note && <span className="text-fg-faint">（{c.note}）</span>}
                    </div>
                  )}
                  {!c.offenders.length && c.note && (
                    <div className="text-fg-faint text-[11px] mt-0.5">{c.note}</div>
                  )}
                </>
              )}
            </div>
            <div className="shrink-0 flex items-center gap-2">
              {c.value !== null && (
                <span className="font-mono text-[11px] text-fg-dim tabular-nums">{c.value}</span>
              )}
              {c.na ? (
                <span className="shrink-0 rounded px-1.5 py-0.5 text-[10px] border border-border text-fg-faint">n/a</span>
              ) : (
                <span
                  className={`shrink-0 rounded px-1.5 py-0.5 text-[10px] ${
                    c.pass ? 'text-emerald-500 bg-emerald-500/10' : 'text-red-400 bg-red-400/10'
                  }`}
                >
                  {c.pass ? '通过' : '未过'}
                </span>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
