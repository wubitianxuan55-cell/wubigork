import { useMemo, useState } from 'react'
import type { CpmResult, SchedProject } from './types'
import { monteCarlo } from './monte'

/** MonteView — 6.5 蒙特卡洛工期带：本地模拟 → 工期 P25/中位/P75 三档 +
 *  关键路径稳定度。与造价价格带同一「工期也给你三档」哲学；种子化模拟
 *  同输入同结果（跨会话可比），全本地零云依赖（6.5 出口判据）。 */
export function MonteView({ project, cpm }: { project: SchedProject; cpm: CpmResult }) {
  const [uncertainty, setUncertainty] = useState(0.3)
  const result = useMemo(() => monteCarlo(project, { uncertainty }), [project, uncertainty])

  if (!result.ok) {
    return (
      <div className="h-full overflow-auto px-4 py-3">
        <div className="text-fg text-[13px] font-medium">蒙特卡洛工期模拟</div>
        <div className="mt-2 text-fg-faint text-[12px]">无法模拟：{result.error}。请先修复搭接关系。</div>
      </div>
    )
  }

  const maxCount = Math.max(1, ...result.histogram.map((h) => h.count))
  const band = (v: number, label: string, tone: string) => (
    <div className="flex-1 min-w-[120px] px-3 py-2 rounded-lg border border-border-soft bg-bg-soft/40">
      <div className="text-fg-faint text-[10.5px]">{label}</div>
      <div className={`text-[22px] font-semibold tabular-nums ${tone}`}>{v}</div>
      <div className="text-fg-faint text-[10px]">工作日</div>
    </div>
  )

  return (
    <div className="h-full overflow-auto px-4 py-3">
      <div className="flex items-baseline gap-3 flex-wrap mb-3">
        <span className="text-fg text-[13px] font-medium">蒙特卡洛工期模拟</span>
        <span className="text-fg-faint text-[11px]">
          {result.runs} 次本地模拟 · 计划工期 {result.plannedDuration} 天 · 关键线路稳定度{' '}
          <span
            className={result.stability >= 0.9 ? 'text-emerald-500' : result.stability >= 0.7 ? 'text-amber-500' : 'text-red-400'}
            data-testid="monte-stability"
          >
            {Math.round(result.stability * 100)}%
          </span>
        </span>
        <label className="ml-auto inline-flex items-center gap-1.5 text-[11px] text-fg-faint cursor-pointer">
          不确定度
          <input
            type="range"
            min={0}
            max={0.6}
            step={0.05}
            value={uncertainty}
            onChange={(e) => setUncertainty(Number(e.target.value))}
            className="w-32"
          />
          <span className="font-mono tabular-nums text-fg-dim w-8">{Math.round(uncertainty * 100)}%</span>
        </label>
      </div>

      <div className="flex gap-2 flex-wrap max-w-[860px] mb-3">
        {band(result.p25, 'P25（乐观）', 'text-emerald-500')}
        {band(result.p50, 'P50 中位', 'text-fg')}
        {band(result.p75, 'P75（保守）', 'text-amber-500')}
      </div>

      <div className="max-w-[860px] mb-3" data-testid="monte-histogram">
        <div className="text-fg-faint text-[10.5px] mb-1">
          工期分布：{result.min} ~ {result.max} 天（均值 {result.mean}）
        </div>
        <div className="flex items-end gap-0.5 h-16">
          {result.histogram.map((h, i) => (
            <div
              key={i}
              className="flex-1 bg-accent/50 rounded-t"
              style={{ height: `${(h.count / maxCount) * 100}%` }}
              title={`${h.count} 次 ≤ ${h.upTo} 天`}
            />
          ))}
        </div>
      </div>

      <div className="max-w-[860px]">
        <div className="text-fg-faint text-[10.5px] mb-1">
          关键线路出现频率（换线风险：当前关键但频率低 = 容易被顶掉）
        </div>
        <div className="space-y-1">
          {result.criticality.slice(0, 12).map((c) => (
            <div key={c.id} className="flex items-center gap-2 text-[11.5px]">
              <span className={`w-2 h-2 rounded-full shrink-0 ${c.criticalNow ? 'bg-red-400' : 'bg-fg-faint/40'}`} />
              <span className={`truncate ${c.criticalNow ? 'text-fg' : 'text-fg-faint'}`}>{c.name}</span>
              <div className="flex-1 h-1.5 bg-bg-soft rounded overflow-hidden min-w-[60px]">
                <div className="h-full bg-accent/60" style={{ width: `${c.rate * 100}%` }} />
              </div>
              <span className="font-mono text-fg-dim tabular-nums w-10 text-right">{Math.round(c.rate * 100)}%</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
