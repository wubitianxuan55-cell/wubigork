/**
 * schedule/GanttView.tsx — 横道图（甘特图）视图
 *
 * 左侧任务表（WBS 编码/大纲缩进/工期/起止/模式/前置编辑）+ 右侧时间轴条形图，
 * 关键工作红色、分组汇总条黑色、总时差尾巴、里程碑菱形、搭接箭线。
 * v4.111.0 对齐 Project/斑马：工作日历日期轴（列=工作日序号，跨周末/节假日）、
 * 手动/自动任务模式、WBS 编码列、前锋线（进度检查线）。
 * 与单代号/双代号共享同一份 store 数据与 CPM 结果（一表多图联动）。
 */
import React, { useMemo, useState } from 'react'
import { Button, DatePicker, Input, InputNumber, Popover, Select } from 'antd'
import { DeleteOutlined, LinkOutlined, ZoomInOutlined, ZoomOutOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import type { CpmResult, LinkType, SchedProject, SchedTask } from './types'
import { dateToWd, wdToDate } from './calendar'
import { descendantIds, isGroupRow, useScheduleStore, type PredDraft } from './store'

const ROW_H = 30
const COLS = [
  { key: 'name', label: '任务名称', w: 196 },
  { key: 'wbs', label: 'WBS', w: 46 },
  { key: 'dur', label: '工期', w: 52 },
  { key: 'start', label: '开始', w: 78 },
  { key: 'finish', label: '完成', w: 78 },
  { key: 'mode', label: '模式', w: 54 },
  { key: 'links', label: '前置', w: 56 },
]
const LEFT_W = COLS.reduce((s, c) => s + c.w, 0)
const DAY_W_STEPS = [8, 14, 20, 28]
const WEEKDAY_LABELS = ['日', '一', '二', '三', '四', '五', '六']

/** WBS 编码：分组 n、叶 n.k（与 mspdi 导出同构） */
function wbsOf(tasks: SchedTask[]): string[] {
  const out: string[] = []
  let g = 0
  let k = 0
  for (const t of tasks) {
    if (t.level === 0) { g++; k = 0; out.push(`${g}`) } else { k++; out.push(`${g}.${k}`) }
  }
  return out
}

/** 分组行汇总跨度（子孙叶项的 min ES / max EF） */
function groupSpan(project: SchedProject, cpm: CpmResult, idx: number): { es: number; ef: number } | null {
  const ids = descendantIds(project.tasks, project.tasks[idx].id)
  let es = Number.POSITIVE_INFINITY
  let ef = 0
  let any = false
  for (const id of ids) {
    const row = cpm.rows[id]
    const t = project.tasks.find((x) => x.id === id)
    if (!row || !t || t.level === 0) continue
    any = true
    es = Math.min(es, row.es)
    ef = Math.max(ef, row.ef)
  }
  return any ? { es, ef } : null
}

/** 前锋线任务点：按完成进度取横道条上的前锋位置（工作日序号，可为小数） */
function frontierWd(t: SchedTask, row: { es: number; ef: number } | undefined, checkIdx: number): number | null {
  if (!row || t.level === 0) return null
  const dur = t.isMilestone ? 0 : Math.max(0, Math.round(t.duration))
  if (row.es >= checkIdx) return row.es // 未开始
  const pct = Math.min(100, Math.max(0, t.progress)) / 100
  return Math.min(row.ef, row.es + pct * dur)
}

/** 前置关系编辑 Popover（整体替换该任务入边） */
const PredEditor: React.FC<{ task: SchedTask }> = ({ task }) => {
  const project = useScheduleStore((s) => s.project)
  const setPreds = useScheduleStore((s) => s.setPreds)
  const drafts: PredDraft[] = useMemo(
    () => project.links.filter((l) => l.to === task.id).map((l) => ({ from: l.from, type: l.type, lag: l.lag })),
    [project.links, task.id],
  )
  const banned = descendantIds(project.tasks, task.id)
  const options = project.tasks.filter((t) => !banned.has(t.id) && t.level > 0)

  const commit = (next: PredDraft[]) => setPreds(task.id, next)

  return (
    <div style={{ display: 'grid', gap: 6, minWidth: 280 }}>
      {drafts.length === 0 && <span style={{ color: 'var(--md-sys-color-outline, #999)', fontSize: 12 }}>暂无前置任务</span>}
      {drafts.map((d, i) => (
        <div key={i} style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
          <Select
            size="small"
            value={d.from}
            style={{ flex: 1 }}
            options={options.map((o) => ({ value: o.id, label: o.name }))}
            onChange={(v) => commit(drafts.map((x, j) => (j === i ? { ...x, from: v } : x)))}
          />
          <Select
            size="small"
            value={d.type}
            style={{ width: 72 }}
            options={[
              { value: 'FS', label: 'FS' }, { value: 'SS', label: 'SS' },
              { value: 'FF', label: 'FF' }, { value: 'SF', label: 'SF' },
            ]}
            onChange={(v) => commit(drafts.map((x, j) => (j === i ? { ...x, type: v as LinkType } : x)))}
          />
          <InputNumber size="small" value={d.lag} style={{ width: 64 }} addonAfter="日" onChange={(v) => commit(drafts.map((x, j) => (j === i ? { ...x, lag: Number(v) || 0 } : x)))} />
          <Button size="small" type="text" danger icon={<DeleteOutlined />} onClick={() => commit(drafts.filter((_, j) => j !== i))} />
        </div>
      ))}
      <Button
        size="small"
        type="dashed"
        disabled={options.length === 0}
        onClick={() => commit([...drafts, { from: options[0]?.id ?? '', type: 'FS', lag: 0 }])}
      >
        添加前置
      </Button>
    </div>
  )
}

export const GanttView: React.FC<{ project: SchedProject; cpm: CpmResult }> = ({ project, cpm }) => {
  const selectedId = useScheduleStore((s) => s.selectedId)
  const select = useScheduleStore((s) => s.select)
  const updateTask = useScheduleStore((s) => s.updateTask)
  const [dayW, setDayW] = useState(20)
  const [showFront, setShowFront] = useState(false)
  const [checkDate, setCheckDate] = useState<string>(new Date().toISOString().slice(0, 10))

  const cal = project.calendar
  const wbs = useMemo(() => wbsOf(project.tasks), [project.tasks])
  const days = Math.max(cpm.duration, 14) + 4
  const chartW = days * dayW
  const totalH = project.tasks.length * ROW_H
  const maxTf = Math.max(0, ...project.tasks.map((t) => cpm.rows[t.id]?.tf ?? 0))

  /** 列序号 → 日期（工作日历口径）+ 月份变化标记 */
  const colDates = useMemo(
    () => Array.from({ length: days }, (_, i) => {
      const d = wdToDate(project.startDate, i, cal)
      return { d, iso: d.toISOString().slice(0, 10), monthChange: i === 0 || d.getUTCMonth() !== wdToDate(project.startDate, i - 1, cal).getUTCMonth() }
    }),
    [project.startDate, cal, days],
  )
  const todayIdx = useMemo(() => dateToWd(project.startDate, new Date().toISOString().slice(0, 10), cal), [project.startDate, cal])

  const linkRows = useMemo(() => {
    const rowIdx = new Map(project.tasks.map((t, i) => [t.id, i]))
    return project.links
      .map((l) => ({ l, fi: rowIdx.get(l.from), ti: rowIdx.get(l.to) }))
      .filter((x) => x.fi !== undefined && x.ti !== undefined && cpm.ok)
  }, [project.links, project.tasks, cpm.ok])

  const linkPaths = linkRows.map(({ l, fi, ti }) => {
    const f = cpm.rows[l.from]
    const t2 = cpm.rows[l.to]
    if (!f || !t2) return null
    const y1 = fi! * ROW_H + ROW_H / 2
    const y2 = ti! * ROW_H + ROW_H / 2
    const x1 = f.ef * dayW
    const x2 = t2.es * dayW
    const mid = Math.max(x1 + 5, (x1 + x2) / 2)
    const d = x2 >= x1 + 10
      ? `M ${x1} ${y1} H ${mid} V ${y2} H ${x2 - 3}`
      : `M ${x1} ${y1} h 6 V ${y1 < y2 ? y2 - 8 : y2 + 8} H ${x2 - 3} V ${y2} H ${x2 - 3}`
    return { d, key: `${l.from}>${l.to}>${l.type}` }
  }).filter(Boolean) as { d: string; key: string }[]

  /** 前锋线折线点（行中心） */
  const frontLine = useMemo(() => {
    if (!showFront || !cpm.ok) return null
    const checkIdx = dateToWd(project.startDate, checkDate, cal)
    if (checkIdx === null) return null
    const pts: { x: number; y: number }[] = []
    project.tasks.forEach((t, i) => {
      const w = frontierWd(t, cpm.rows[t.id], checkIdx)
      if (w !== null) pts.push({ x: w * dayW, y: i * ROW_H + ROW_H / 2 })
    })
    return pts.length >= 2 ? { pts, checkX: checkIdx * dayW } : null
  }, [showFront, cpm, project.tasks, project.startDate, checkDate, cal, dayW])

  return (
    <div className="sched-gantt" data-testid="sched-gantt">
      <div className="sched-gantt-toolbar">
        <span className="sched-gantt-stats">
          总工期 {cpm.duration} 工作日 · 关键工作 {project.tasks.filter((t) => t.level > 0 && cpm.rows[t.id]?.critical).length} 项
          {maxTf > 0 && <> · 最大总时差 {maxTf} 天</>}
        </span>
        <div style={{ flex: 1 }} />
        <Button
          size="small"
          type={showFront ? 'primary' : 'default'}
          onClick={() => setShowFront((v) => !v)}
          title="按任务完成进度画前锋线（斑马式进度检查）"
        >
          前锋线
        </Button>
        {showFront && (
          <DatePicker
            size="small"
            value={dayjs(checkDate)}
            onChange={(d) => d && setCheckDate(d.format('YYYY-MM-DD'))}
            style={{ width: 130 }}
            placeholder="检查日期"
          />
        )}
        <Button size="small" icon={<ZoomOutOutlined />} onClick={() => setDayW((w) => DAY_W_STEPS[Math.max(0, DAY_W_STEPS.indexOf(w) - 1)] ?? w)} />
        <Button size="small" icon={<ZoomInOutlined />} onClick={() => setDayW((w) => DAY_W_STEPS[Math.min(DAY_W_STEPS.length - 1, DAY_W_STEPS.indexOf(w) + 1)] ?? w)} />
      </div>
      <div className="sched-gantt-scroll">
        <div className="sched-gantt-inner" style={{ width: LEFT_W + chartW }}>
          {/* 表头 */}
          <div className="sched-gantt-row sched-gantt-head" style={{ height: 40 }}>
            <div className="sched-gantt-left sched-sticky-left" style={{ width: LEFT_W }}>
              {COLS.map((c) => (
                <div key={c.key} style={{ width: c.w }} className="sched-gantt-cell sched-th">{c.label}</div>
              ))}
            </div>
            <div className="sched-gantt-datehead" style={{ width: chartW }}>
              {colDates.map((c, i) => (
                <div key={i} className="sched-day-col" style={{ width: dayW }}>
                  <div className="sched-month-cell">{c.monthChange ? `${c.d.getUTCMonth() + 1}月` : ''}</div>
                  <div className="sched-day-cell" title={`周${WEEKDAY_LABELS[c.d.getUTCDay()]}`}>
                    {c.d.getUTCDate()}
                  </div>
                </div>
              ))}
            </div>
          </div>
          {/* 数据行 */}
          {project.tasks.map((t, i) => {
            const row = cpm.rows[t.id]
            const group = isGroupRow(project.tasks, i)
            const span = group ? groupSpan(project, cpm, i) : null
            const crit = !group && t.mode !== 'manual' && row?.critical
            const manual = t.mode === 'manual'
            const dur = t.isMilestone ? 0 : Math.max(0, Math.round(t.duration))
            return (
              <div
                key={t.id}
                className={`sched-gantt-row${selectedId === t.id ? ' sched-row-selected' : ''}`}
                style={{ height: ROW_H }}
                onClick={() => select(t.id)}
              >
                <div className="sched-gantt-left sched-sticky-left" style={{ width: LEFT_W }}>
                  <div style={{ width: COLS[0].w }} className="sched-gantt-cell">
                    <span style={{ paddingLeft: t.level * 14, display: 'inline-flex', alignItems: 'center', gap: 4, minWidth: 0 }}>
                      {group ? <strong className="sched-group-name">{t.name}</strong> : (
                        <Input
                          size="small"
                          variant="borderless"
                          value={t.name}
                          onChange={(e) => updateTask(t.id, { name: e.target.value })}
                          style={{ padding: 0 }}
                        />
                      )}
                      {t.isMilestone && <span className="sched-milestone-tag">里程碑</span>}
                    </span>
                  </div>
                  <div style={{ width: COLS[1].w }} className="sched-gantt-cell sched-dim">{wbs[i]}</div>
                  <div style={{ width: COLS[2].w }} className="sched-gantt-cell">
                    {group ? <span className="sched-dim">汇总</span> : (
                      <InputNumber
                        size="small"
                        variant="borderless"
                        min={0}
                        value={dur}
                        onChange={(v) => updateTask(t.id, { duration: Number(v) || 0 })}
                        style={{ padding: 0, width: '100%' }}
                      />
                    )}
                  </div>
                  <div style={{ width: COLS[3].w }} className="sched-gantt-cell sched-dim">
                    {manual && !group ? (
                      <InputNumber
                        size="small"
                        variant="borderless"
                        min={0}
                        value={row?.es ?? 0}
                        onChange={(v) => updateTask(t.id, { manualStart: Number(v) || 0 })}
                        style={{ padding: 0, width: '100%' }}
                        title="手动模式：锁定开始（工作日序号）"
                      />
                    ) : (
                      colDates.length > 0 && (group ? span : row) ? fmtDate(colDates, group ? span!.es : row!.es) : ''
                    )}
                  </div>
                  <div style={{ width: COLS[4].w }} className="sched-gantt-cell sched-dim">
                    {colDates.length > 0 && (group ? span : row) ? fmtDate(colDates, group ? span!.ef : row!.ef) : ''}
                  </div>
                  <div style={{ width: COLS[5].w }} className="sched-gantt-cell">
                    {!group && (
                      <Button
                        size="small"
                        type={manual ? 'primary' : 'text'}
                        ghost={manual}
                        className={`sched-mode-chip${manual ? ' sched-mode-manual' : ''}`}
                        onClick={() => updateTask(t.id, manual ? { mode: 'auto' } : { mode: 'manual', manualStart: row?.es ?? 0 })}
                        title={manual ? '手动模式（锁定开始，点击切回自动）' : '自动模式（CPM 排程，点击切手动）'}
                      >
                        {manual ? '手动' : '自动'}
                      </Button>
                    )}
                  </div>
                  <div style={{ width: COLS[6].w }} className="sched-gantt-cell">
                    {!group && (
                      <Popover trigger="click" placement="left" content={<PredEditor task={t} />} title={`「${t.name}」前置任务`}>
                        <Button size="small" type="text" icon={<LinkOutlined />}>
                          {project.links.filter((l) => l.to === t.id).length || ''}
                        </Button>
                      </Popover>
                    )}
                  </div>
                </div>
                {/* 条形区 */}
                <div className="sched-bar-lane" style={{ width: chartW }}>
                  {group ? (
                    span && (
                      <div
                        className="sched-summary-bar"
                        style={{ left: span.es * dayW, width: Math.max((span.ef - span.es) * dayW, 8) }}
                        title={`${t.name}：${span.es} ~ ${span.ef} 工作日`}
                      />
                    )
                  ) : t.isMilestone ? (
                    row && (
                      <div className="sched-milestone" style={{ left: row.es * dayW - 7 }} title={`${t.name}（里程碑）`} />
                    )
                  ) : (
                    row && (
                      <>
                        <div
                          className={`sched-bar${crit ? ' sched-bar-critical' : ''}${manual ? ' sched-bar-manual' : ''}`}
                          style={{ left: row.es * dayW, width: Math.max(dur * dayW, 6) }}
                          title={`${t.name}：第 ${row.es}~${row.ef} 工作日${manual ? '（手动锁定）' : crit ? '（关键）' : `，总时差 ${row.tf} 天`}`}
                        >
                          {t.progress > 0 && (
                            <div className="sched-bar-progress" style={{ width: `${Math.min(100, t.progress)}%` }} />
                          )}
                        </div>
                        {!manual && row.tf > 0 && (
                          <div className="sched-float" style={{ left: row.ef * dayW, width: row.tf * dayW }} title={`总时差 ${row.tf} 天`} />
                        )}
                      </>
                    )
                  )}
                </div>
              </div>
            )
          })}
          {/* 覆盖层：今日线 + 前锋线（搭接箭线在条形下方） */}
          <div className="sched-overlay" style={{ left: LEFT_W, width: chartW, height: totalH + 40 }}>
            <svg width={chartW} height={totalH + 40} style={{ position: 'absolute', inset: 0, pointerEvents: 'none' }}>
              <defs>
                <marker id="sched-arrow" markerWidth="7" markerHeight="7" refX="6" refY="3.5" orient="auto">
                  <path d="M0,0 L7,3.5 L0,7 z" fill="var(--sched-link, #94a3b8)" />
                </marker>
              </defs>
              {linkPaths.map((p) => (
                <path key={p.key} d={p.d} className="sched-link-line" markerEnd="url(#sched-arrow)" />
              ))}
              {todayIdx !== null && todayIdx <= days && (
                <line x1={todayIdx * dayW} y1={0} x2={todayIdx * dayW} y2={totalH + 40} className="sched-today-line" />
              )}
              {frontLine && (
                <>
                  <line x1={frontLine.checkX} y1={0} x2={frontLine.checkX} y2={totalH + 40} className="sched-front-check" />
                  <polyline points={frontLine.pts.map((p) => `${p.x},${p.y}`).join(' ')} className="sched-front-line" />
                  {frontLine.pts.map((p, i) => <circle key={i} cx={p.x} cy={p.y} r={2.4} className="sched-front-dot" />)}
                </>
              )}
            </svg>
          </div>
        </div>
      </div>
      {project.tasks.length === 0 && (
        <div className="sched-empty">暂无任务，点击上方「添加任务」或载入示例工程开始编制</div>
      )}
    </div>
  )
}

/** 日期列查找：工作日序号 → M/D 文本 */
function fmtDate(colDates: { iso: string }[], workdayIdx: number): string {
  const c = colDates[Math.max(0, Math.min(colDates.length - 1, Math.round(workdayIdx)))]
  if (!c) return ''
  const [, m, d] = c.iso.split('-')
  return `${Number(m)}/${Number(d)}`
}
