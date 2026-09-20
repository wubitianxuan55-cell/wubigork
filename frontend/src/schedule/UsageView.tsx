/**
 * schedule/UsageView.tsx — 资源使用视图（v4.137 #12，纯展示零业务）
 *
 * 对齐斑马/MS Project「资源使用状况」时标表：左固定资源列 + 右逐工作日负载热力格。
 * 只消费 usageTypes.ts 数据契约（数据由父级 computeUsage 算好传入），
 * 不 import 任何业务计算模块：
 *  - 顶部汇总：共 N 名工时资源 · 峰值负载 X · 超载 M 资源-日（M>0 红色）；
 *    ok=false 显示错误文案；rows 空或 days=0 显示空态引导；
 *  - 左固定列 230px：资源名 + 超载天数（>0 红 / =0 灰）+ 峰值/累计小字，行高 30px；
 *  - 右横滚区：列=工作日 wd（0..days-1，列宽 22px），表头每 5 标一个数
 *    （days>62 防挤改每 10）；负载格按 rgba(30,111,217,alpha) 深浅着色
 *    （alpha=min(workload/max(当日可用,行峰值,1),1) 映射 0.15~0.85），零负载诚实留白；
 *    超载格红底红描边 + 右上角红点（class sched-usage-over），
 *    title=「第 N 工作日：负载 X / 可用 Y（任务A×1、任务B×2）」
 *    ——当日在工任务从 row.tasks 过滤 es≤wd<ef（纯读小计算）；
 *  - 行悬浮高亮：复用 .sched-gantt-row:hover，左列 position:sticky 随行同亮
 *    （纯 CSS :hover 无 JS 状态；sticky 左列的底色规则由组件内 <style> 注入，
 *    不新建 css 文件）。
 */
import React from 'react'
import type { UsageDayCell, UsageResourceRow, UsageResult } from './usageTypes'

const LEFT_W = 230
const ROW_H = 30
const DAY_W = 22
/** 表头数字步长：≤62 天每 5 个工作日标一个数，>62 天改每 10（防挤） */
const HEAD_STEP_DENSE = 5
const HEAD_STEP_SPARSE = 10
/** 热力蓝（斑马负载色）RGB 分量，深浅由 alpha 控制 */
const HEAT_RGB = '30,111,217'
/** 超载红（斑马关键红 token） */
const OVER_COLOR = 'var(--sched-critical, #e02020)'
/** 左列与格子的分隔线 */
const SEP = '1px solid var(--md-sys-color-border, #e5e7eb)'
/** 次级小字颜色 */
const DIM_COLOR = 'var(--md-sys-color-on-surface-variant, #6b7280)'

/** sticky 左列底色 + 行悬浮同亮（注入组件内 <style>，等价一段私有样式） */
const USAGE_CSS = `
.sched-usage .sched-usage-lcol {
  position: sticky;
  left: 0;
  z-index: 2;
  background-color: var(--md-sys-color-bg-container, #fff);
}
.sched-usage .sched-gantt-row:hover .sched-usage-lcol {
  background-image: linear-gradient(rgba(30, 111, 217, 0.08), rgba(30, 111, 217, 0.08));
}
`

/** 数字展示：整数原样，小数最多 2 位并去尾零（负载/units 可能非整） */
function fmtNum(v: number): string {
  return String(Number(v.toFixed(2)))
}

/** 全项目峰值负载 = 各行峰值最大值（空行集=0） */
function peakOf(rows: UsageResourceRow[]): number {
  let peak = 0
  for (const r of rows) if (r.peakWorkload > peak) peak = r.peakWorkload
  return peak
}

/** 格子 title：当日在工任务（es≤wd<ef）从 row.tasks 纯读过滤，列出投入构成 */
function cellTitle(row: UsageResourceRow, cell: UsageDayCell): string {
  const actives = row.tasks.filter((t) => t.es <= cell.wd && cell.wd < t.ef)
  const who = actives.length > 0
    ? `（${actives.map((t) => `${t.name}×${fmtNum(t.units)}`).join('、')}）`
    : ''
  return `第 ${cell.wd} 工作日：负载 ${fmtNum(cell.workload)} / 可用 ${fmtNum(cell.availability)}${who}`
}

/** 格子样式：超载=红底+红描边；有负载=热力蓝深浅；零负载=透明留白 */
function cellStyle(row: UsageResourceRow, cell: UsageDayCell): React.CSSProperties {
  if (cell.overloaded) {
    return { background: 'rgba(224, 32, 32, 0.32)', boxShadow: `inset 0 0 0 1px ${OVER_COLOR}` }
  }
  if (cell.workload <= 0) return {}
  const denom = Math.max(cell.availability, row.peakWorkload, 1)
  const alpha = 0.15 + 0.7 * Math.min(cell.workload / denom, 1)
  return { background: `rgba(${HEAT_RGB}, ${alpha.toFixed(3)})` }
}

export const UsageView: React.FC<{ usage: UsageResult }> = ({ usage }) => {
  // ok=false：只显示错误文案（数据不可信，不渲染热力表）
  if (!usage.ok) {
    return (
      <div className="sched-usage" data-testid="sched-usage">
        <div className="sched-gantt-stats" data-testid="sched-usage-summary" style={{ color: OVER_COLOR }}>
          {`使用视图计算失败：${usage.error ?? '未知错误'}`}
        </div>
      </div>
    )
  }

  const { rows, days } = usage
  const headStep = days > 62 ? HEAD_STEP_SPARSE : HEAD_STEP_DENSE
  const peak = peakOf(rows)
  const empty = rows.length === 0 || days === 0

  return (
    <div
      className="sched-usage"
      data-testid="sched-usage"
      style={{ display: 'flex', flexDirection: 'column', gap: 6, minHeight: 0, flex: 1 }}
    >
      <style>{USAGE_CSS}</style>
      {/* 顶部汇总：超载资源-日 >0 时红色 */}
      <div className="sched-gantt-stats" data-testid="sched-usage-summary">
        {`共 ${rows.length} 名工时资源 · 峰值负载 ${fmtNum(peak)} · `}
        <span
          data-testid="sched-usage-over-total"
          style={usage.totalOverloadDays > 0 ? { color: OVER_COLOR, fontWeight: 600 } : undefined}
        >
          {`超载 ${fmtNum(usage.totalOverloadDays)} 资源-日`}
        </span>
      </div>
      {empty ? (
        <div className="sched-empty" data-testid="sched-usage-empty">
          暂无工时资源或计划为空——在「资源」中添加工时资源并分配到任务
        </div>
      ) : (
        <div style={{ overflowX: 'auto', minHeight: 0 }}>
          <div style={{ width: LEFT_W + days * DAY_W, minWidth: '100%' }}>
            {/* 表头行：左列标题 + 每 headStep 个工作日标一个序号 */}
            <div className="sched-gantt-row sched-gantt-head" style={{ height: 28 }}>
              <div
                className="sched-usage-lcol"
                style={{
                  width: LEFT_W, height: 28, flexShrink: 0, display: 'flex', alignItems: 'center',
                  padding: '0 6px', fontSize: 12, fontWeight: 600, borderRight: SEP,
                }}
              >
                工时资源
              </div>
              {Array.from({ length: days }, (_, wd) => (
                <div
                  key={wd}
                  data-testid={`sched-usage-head-${wd}`}
                  style={{
                    width: DAY_W, flexShrink: 0, display: 'flex', alignItems: 'center', justifyContent: 'center',
                    fontSize: 10, color: DIM_COLOR,
                  }}
                >
                  {wd % headStep === 0 ? wd : ''}
                </div>
              ))}
            </div>
            {/* 资源行：左固定列（sticky）+ 逐工作日热力格（行悬浮同亮靠 .sched-gantt-row:hover） */}
            {rows.map((row) => (
              <div key={row.resourceId} className="sched-gantt-row" style={{ height: ROW_H }}>
                <div
                  className="sched-usage-lcol"
                  style={{
                    width: LEFT_W, height: ROW_H, flexShrink: 0, display: 'flex', flexDirection: 'column',
                    justifyContent: 'center', overflow: 'hidden', padding: '0 6px', borderRight: SEP,
                  }}
                >
                  <div style={{ display: 'flex', alignItems: 'baseline', gap: 6, fontSize: 12, lineHeight: '15px', whiteSpace: 'nowrap' }}>
                    <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', minWidth: 0 }}>{row.name}</span>
                    <span
                      style={{
                        color: row.overloadDays > 0 ? OVER_COLOR : DIM_COLOR,
                        fontSize: 11, flexShrink: 0,
                      }}
                    >
                      {`超载 ${fmtNum(row.overloadDays)} 天`}
                    </span>
                  </div>
                  <div style={{ fontSize: 10, lineHeight: '12px', color: DIM_COLOR, whiteSpace: 'nowrap' }}>
                    {`峰值 ${fmtNum(row.peakWorkload)} · 累计 ${fmtNum(row.totalWorkload)} 工日`}
                  </div>
                </div>
                {row.cells.map((cell) => (
                  <div
                    key={cell.wd}
                    className={`sched-usage-cell${cell.overloaded ? ' sched-usage-over' : ''}`}
                    style={{
                      width: DAY_W, height: ROW_H, flexShrink: 0, position: 'relative',
                      display: 'flex', alignItems: 'center', justifyContent: 'center',
                      ...cellStyle(row, cell),
                    }}
                    title={cellTitle(row, cell)}
                  >
                    {cell.overloaded && (
                      <span
                        style={{
                          position: 'absolute', top: 1, right: 1, width: 4, height: 4,
                          borderRadius: '50%', background: OVER_COLOR,
                        }}
                      />
                    )}
                  </div>
                ))}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
