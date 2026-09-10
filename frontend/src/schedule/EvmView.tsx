import { useMemo, useState } from 'react'
import type { CpmResult, SchedProject } from './types'
import { computeEvm } from './evm'

/** EvmView — 6.6 跨域 EVM：进度×造价 挣值三数（PV/EV/AC）+ 两指数（SPI/CPI）。
 *  gaea 独有跨域：进度与造价同在本机一个库。AC（实际成本）来源=手工录入
 *  （口径注直出），缺失时 CPI/CV 诚实 n/a。纯展示：计算在 computeEvm
 *  纯函数内完成。 */
export function EvmView({ project, cpm }: { project: SchedProject; cpm: CpmResult }) {
  const [ddInput, setDdInput] = useState('')
  const [acInput, setAcInput] = useState('')

  const defaultDd = Math.max(1, Math.round(cpm.duration / 2))
  const dataDate = ddInput.trim() === '' ? defaultDd : Math.max(0, Number(ddInput) || 0)
  const ac = acInput.trim() === '' ? null : Number(acInput)

  const report = useMemo(
    () => computeEvm(project, cpm, { dataDate, actualCost: ac }),
    [project, cpm, dataDate, ac],
  )

  const tone = (v: number | null, good = 1) => {
    if (v === null) return 'text-fg-faint'
    if (v >= good) return 'text-emerald-500'
    if (v >= 0.95) return 'text-amber-500'
    return 'text-red-400'
  }
  const num = (v: number | null, digits = 0) =>
    v === null ? '—' : v.toLocaleString('zh-CN', { maximumFractionDigits: digits })

  if (!report.ok) {
    return (
      <div className="h-full overflow-auto px-4 py-3">
        <div className="text-fg text-[13px] font-medium">挣值分析（进度 × 造价 EVM）</div>
        <div className="mt-2 text-fg-faint text-[12px]">{report.error}。</div>
      </div>
    )
  }

  const numCard = (label: string, n: { value: number | null; note: string }, tone?: string) => (
    <div className="flex-1 min-w-[140px] px-3 py-2 rounded-lg border border-border-soft bg-bg-soft/40" title={n.note}>
      <div className="text-fg-faint text-[10.5px]">{label}</div>
      <div className={`text-[20px] font-semibold tabular-nums ${tone ?? 'text-fg'}`}>
        {n.value === null ? '—' : `¥${num(n.value)}`}
      </div>
    </div>
  )
  const idxCard = (label: string, n: { value: number | null; note: string }, good: number) => (
    <div className="flex-1 min-w-[120px] px-3 py-2 rounded-lg border border-border-soft bg-bg-soft/40" title={n.note}>
      <div className="text-fg-faint text-[10.5px]">{label}</div>
      <div className={`text-[20px] font-semibold tabular-nums ${tone(n.value, good)}`}>
        {n.value === null ? '—' : n.value.toFixed(3).replace(/0+$/, '').replace(/\.$/, '')}
      </div>
    </div>
  )

  return (
    <div className="h-full overflow-auto px-4 py-3">
      <div className="flex items-center gap-3 flex-wrap mb-3">
        <span className="text-fg text-[13px] font-medium">挣值分析（进度 × 造价 EVM）</span>
        <label className="inline-flex items-center gap-1.5 text-[11px] text-fg-faint">
          数据日期（第 N 工作日末）
          <input
            type="number"
            min={0}
            value={ddInput}
            placeholder={`${defaultDd}`}
            onChange={(e) => setDdInput(e.target.value)}
            className="w-20 bg-bg border border-border-soft rounded px-2 py-1 text-fg text-[11px] font-mono outline-none focus:border-accent"
          />
        </label>
        <label className="inline-flex items-center gap-1.5 text-[11px] text-fg-faint">
          实际成本 AC（元，手工录入）
          <input
            type="number"
            min={0}
            value={acInput}
            placeholder="未录入"
            onChange={(e) => setAcInput(e.target.value)}
            className="w-28 bg-bg border border-border-soft rounded px-2 py-1 text-fg text-[11px] font-mono outline-none focus:border-accent"
          />
        </label>
      </div>

      <div className="flex gap-2 flex-wrap max-w-[860px] mb-3">
        {numCard('PV 计划价值', report.pv)}
        {numCard('EV 挣值', report.ev, 'text-emerald-500')}
        {numCard('AC 实际成本', report.ac, report.ac.value === null ? 'text-fg-faint' : 'text-fg')}
      </div>
      <div className="flex gap-2 flex-wrap max-w-[860px] mb-3">
        {idxCard('SPI 进度绩效', report.spi, 1)}
        {idxCard('CPI 成本绩效', report.cpi, 1)}
        <div className="flex-1 min-w-[140px] px-3 py-2 rounded-lg border border-border-soft bg-bg-soft/40">
          <div className="text-fg-faint text-[10.5px]">SV / CV 偏差（元，正=好）</div>
          <div className="text-[14px] font-semibold tabular-nums text-fg-dim">
            SV {num(report.sv)} · CV {num(report.cv)}
          </div>
        </div>
      </div>

      <div className="max-w-[860px] space-y-1 mb-3">
        <div className="text-fg-faint text-[10.5px]">口径注（每数一条，诚实标注来源与算法）</div>
        {(['pv', 'ev', 'ac', 'spi', 'cpi'] as const).map((k) => (
          <div key={k} className="text-fg-faint text-[11px] leading-relaxed">
            · {report[k].note}
          </div>
        ))}
        <div className="text-fg-faint text-[11px] leading-relaxed">· BAC 完工预算 = 当前任务预算合计（固定成本 + 资源分配），¥{num(report.bac)}</div>
      </div>

      {report.evTop.length > 0 && (
        <div className="max-w-[860px]">
          <div className="text-fg-faint text-[10.5px] mb-1">EV 来源 Top 5（挣了哪些任务的值）</div>
          {report.evTop.map((r) => (
            <div key={r.id} className="flex items-center gap-2 text-[11.5px] py-0.5">
              <span className="truncate text-fg-dim flex-1">{r.name}</span>
              <span className="font-mono text-fg-dim tabular-nums">¥{num(r.earned)}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
