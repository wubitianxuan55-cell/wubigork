/**
 * schedule/GanttView.tsx — 横道图（甘特图）视图
 *
 * 左侧任务表（大纲缩进/工期/起止/前置编辑）+ 右侧时间轴条形图，
 * 关键工作红色、分组汇总条黑色、总时差尾巴、里程碑菱形、搭接箭线。
 * 与单代号/双代号共享同一份 store 数据与 CPM 结果（一表多图联动）。
 */
import React, { useMemo, useState } from 'react'
import { Button, Input, InputNumber, Popover, Select } from 'antd'
import { DeleteOutlined, LinkOutlined, ZoomInOutlined, ZoomOutOutlined } from '@ant-design/icons'
import type { CpmResult, LinkType, SchedProject, SchedTask } from './types'
import { descendantIds, isGroupRow, useScheduleStore, type PredDraft } from './store'

const ROW_H = 30
const COLS = [
  { key: 'name', label: '任务名称', w: 230 },
  { key: 'dur', label: '工期', w: 58 },
  { key: 'start', label: '开始', w: 82 },
  { key: 'finish', label: '完成', w: 82 },
  { key: 'links', label: '前置', w: 64 },
]
const LEFT_W = COLS.reduce((s, c) => s + c.w, 0)
const DAY_W_STEPS = [8, 14, 20, 28]

function addDays(iso: string, days: number): Date {
  const [y, m, d] = iso.split('-').map(Number)
  return new Date(Date.UTC(y || 2026, (m || 1) - 1, d || 1) + days * 86400000)
}
function fmt(d: Date): string {
  return `${d.getUTCMonth() + 1}/${d.getUTCDate()}`
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
          <InputNumber size="small" value={d.lag} style={{ width: 64 }} onChange={(v) => commit(drafts.map((x, j) => (j === i ? { ...x, lag: Number(v) || 0 } : x)))} />
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

  const days = Math.max(cpm.duration, 14) + 4
  const chartW = days * dayW
  const totalH = project.tasks.length * ROW_H
  const maxTf = Math.max(0, ...project.tasks.map((t) => cpm.rows[t.id]?.tf ?? 0))

  const weekendCols = useMemo(() => {
    const out: number[] = []
    for (let i = 0; i < days; i++) {
      const day = addDays(project.startDate, i).getUTCDay()
      if (day === 0 || day === 6) out.push(i)
    }
    return out
  }, [project.startDate, days])

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

  const todayX = useMemo(() => {
    const now = Date.now()
    const start = addDays(project.startDate, 0).getTime()
    const diff = Math.floor((now - start) / 86400000)
    return diff >= 0 && diff <= days ? diff * dayW : null
  }, [project.startDate, days, dayW])

  return (
    <div className="sched-gantt" data-testid="sched-gantt">
      <div className="sched-gantt-toolbar">
        <span className="sched-gantt-stats">
          总工期 {cpm.duration} 天 · 关键工作 {project.tasks.filter((t) => t.level > 0 && cpm.rows[t.id]?.critical).length} 项
          {maxTf > 0 && <> · 最大总时差 {maxTf} 天</>}
        </span>
        <div style={{ flex: 1 }} />
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
              {Array.from({ length: days }, (_, i) => {
                const d = addDays(project.startDate, i)
                const showMonth = d.getUTCDate() === 1 || i === 0
                return (
                  <div key={i} className="sched-day-col" style={{ width: dayW }}>
                    <div className="sched-month-cell">{showMonth ? `${d.getUTCMonth() + 1}月` : ''}</div>
                    <div className="sched-day-cell">{dayW >= 14 || i % 2 === 0 ? d.getUTCDate() : ''}</div>
                  </div>
                )
              })}
            </div>
          </div>
          {/* 数据行 */}
          {project.tasks.map((t, i) => {
            const row = cpm.rows[t.id]
            const group = isGroupRow(project.tasks, i)
            const span = group ? groupSpan(project, cpm, i) : null
            const crit = !group && row?.critical
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
                    <span style={{ paddingLeft: t.level * 16, display: 'inline-flex', alignItems: 'center', gap: 4, minWidth: 0 }}>
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
                  <div style={{ width: COLS[1].w }} className="sched-gantt-cell">
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
                  <div style={{ width: COLS[2].w }} className="sched-gantt-cell sched-dim">
                    {row && (group ? span : true) ? fmt(addDays(project.startDate, group ? span!.es : row.es)) : ''}
                  </div>
                  <div style={{ width: COLS[3].w }} className="sched-gantt-cell sched-dim">
                    {row && (group ? span : true) ? fmt(addDays(project.startDate, group ? span!.ef : row.ef)) : ''}
                  </div>
                  <div style={{ width: COLS[4].w }} className="sched-gantt-cell">
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
                        title={`${t.name}：${span.es} ~ ${span.ef} 天`}
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
                          className={`sched-bar${crit ? ' sched-bar-critical' : ''}`}
                          style={{ left: row.es * dayW, width: Math.max(dur * dayW, 6) }}
                          title={`${t.name}：第 ${row.es}~${row.ef} 天${crit ? '（关键）' : `，总时差 ${row.tf} 天`}`}
                        >
                          {t.progress > 0 && (
                            <div className="sched-bar-progress" style={{ width: `${Math.min(100, t.progress)}%` }} />
                          )}
                        </div>
                        {row.tf > 0 && (
                          <div className="sched-float" style={{ left: row.ef * dayW, width: row.tf * dayW }} title={`总时差 ${row.tf} 天`} />
                        )}
                      </>
                    )
                  )}
                </div>
                <span style={{ display: 'none' }}>{i}</span>
              </div>
            )
          })}
          {/* 搭接箭线 + 周末底纹 + 今日线（绝对定位覆盖层，条形之上、文本之下） */}
          <div className="sched-overlay" style={{ left: LEFT_W, width: chartW, height: totalH + 40 }}>
            <svg width={chartW} height={totalH + 40} style={{ position: 'absolute', inset: 0, pointerEvents: 'none' }}>
              <defs>
                <marker id="sched-arrow" markerWidth="7" markerHeight="7" refX="6" refY="3.5" orient="auto">
                  <path d="M0,0 L7,3.5 L0,7 z" fill="var(--sched-link, #94a3b8)" />
                </marker>
              </defs>
              {weekendCols.map((i) => (
                <rect key={`we${i}`} x={i * dayW} y={0} width={dayW} height={totalH + 40} className="sched-weekend" />
              ))}
              {linkPaths.map((p) => (
                <path key={p.key} d={p.d} className="sched-link-line" markerEnd="url(#sched-arrow)" />
              ))}
              {todayX !== null && <line x1={todayX} y1={0} x2={todayX} y2={totalH + 40} className="sched-today-line" />}
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
