/**
 * schedule/GanttView.tsx — 横道图（甘特图）视图（v4.112.0 斑马 UI 对齐）
 *
 * 视觉/交互对齐广联达斑马进度（官网截图口径）：
 *  - 自然日时间轴 + 周末/节假日底纹（CPM 仍按工作日计算，条形跨非工作日渲染）；
 *  - 行号列、分组行彩色底纹（五色循环 + 左色条）；
 *  - 关键红条、分组黑汇总条、里程碑黑菱形、总时差尾巴、搭接箭线、前锋线。
 * 列：行号 | 任务名称 | WBS | 工期 | 开始 | 完成 | 模式 | 前置。
 */
import React, { useMemo, useState } from 'react'
import { Button, DatePicker, Input, InputNumber, Popover, Select } from 'antd'
import { DeleteOutlined, LinkOutlined, ZoomInOutlined, ZoomOutOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import type { CpmResult, LinkType, SchedProject, SchedTask } from './types'
import { isWorkingDate, normalizeCalendar, wdToDate } from './calendar'
import { dropToWd, resizeToDuration, workdayOffsets } from './drag'
import { descendantIds, isGroupRow, useScheduleStore, type PredDraft } from './store'

const ROW_H = 30
const COLS = [
  { key: 'no', label: '', w: 34 },
  { key: 'name', label: '任务名称', w: 184 },
  { key: 'wbs', label: 'WBS', w: 46 },
  { key: 'dur', label: '工期', w: 52 },
  { key: 'start', label: '开始', w: 74 },
  { key: 'finish', label: '完成', w: 74 },
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

/** 目标竣工日期 → 开工日起自然日偏移（未设返回 -1） */
function deadlineOff(deadline: string | null | undefined, startMs: number): number {
  if (!deadline) return -1
  const t = new Date(`${deadline}T00:00:00Z`).getTime()
  if (!Number.isFinite(t)) return -1
  return Math.round((t - startMs) / 86400000)
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

/** 前锋线任务点：按完成进度取前锋位置（工作日序号，可为小数） */
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
  /** 基线条显示（有基线才出现开关；默认开，v4.116 刀7） */
  const [showBase, setShowBase] = useState(true)
  const [checkDate, setCheckDate] = useState<string>(new Date().toISOString().slice(0, 10))
  /** 拖拽态（刀9）：move=拖移（auto 转手动锁定）、resize=右缘改工期 */
  const [drag, setDrag] = useState<{
    id: string
    kind: 'move' | 'resize'
    origEs: number
    origDur: number
    preview: number
    moved: boolean
  } | null>(null)

  const cal = project.calendar
  const wbs = useMemo(() => wbsOf(project.tasks), [project.tasks])
  const startMs = useMemo(() => new Date(`${project.startDate}T00:00:00Z`).getTime(), [project.startDate])
  /** 工作日序号 → 自然日列偏移（条形 x 坐标；跨周末自然变宽，斑马口径） */
  const dayNo = useMemo(() => (wd: number) => {
    return Math.round((wdToDate(project.startDate, wd, cal).getTime() - startMs) / 86400000)
  }, [project.startDate, cal, startMs])

  const days = Math.max(dayNo(Math.max(cpm.duration, 7)) + 3, 21, deadlineOff(project.deadline, startMs) + 2)
  const chartW = days * dayW
  const totalH = project.tasks.length * ROW_H

  /** 自然日列（斑马口径：每天一列，非工作日底纹） */
  const colDates = useMemo(
    () => Array.from({ length: days }, (_, i) => {
      const d = new Date(startMs + i * 86400000)
      return { d, iso: d.toISOString().slice(0, 10), offWork: !isWorkingDate(d, normalizeCalendar(cal)) }
    }),
    [startMs, cal, days],
  )
  const todayCol = useMemo(() => {
    const off = Math.floor((Date.now() - startMs) / 86400000)
    return off >= 0 && off <= days ? off : null
  }, [startMs, days])

  /** 目标竣工线（自然日列位；超出画布为 null 不画） */
  const deadlineCol = useMemo(() => {
    const off = deadlineOff(project.deadline, startMs)
    return off >= 0 && off <= days ? off : null
  }, [project.deadline, startMs, days])

  /** 工作日→自然日偏移反查表（拖拽落点吸附用） */
  const wdOffsets = useMemo(
    () => workdayOffsets(project.startDate, Math.max(cpm.duration + 30, 45), project.calendar),
    [project.startDate, project.calendar, cpm.duration],
  )

  /** 开始拖拽：拖移（转手动锁定）或右缘缩放（改工期）。循环依赖/分组行/里程碑缩放禁用。 */
  const beginDrag = (e: React.MouseEvent, t: SchedTask, kind: 'move' | 'resize') => {
    if (!cpm.ok || t.level === 0) return
    if (kind === 'resize' && t.isMilestone) return
    const row = cpm.rows[t.id]
    if (!row) return
    e.preventDefault()
    e.stopPropagation()
    const startX = e.clientX
    const origEs = row.es
    const origDur = t.isMilestone ? 0 : Math.max(0, Math.round(t.duration))
    const st = {
      id: t.id, kind, origEs, origDur,
      preview: kind === 'move' ? origEs : origDur,
      moved: false,
    }
    const onMove = (ev: MouseEvent) => {
      const dx = ev.clientX - startX
      if (dx !== 0) st.moved = true
      st.preview = kind === 'move'
        ? dropToWd(wdOffsets, dayNo(origEs), dx, dayW)
        : resizeToDuration(wdOffsets, origEs, dayNo(origEs + origDur), dx, dayW)
      setDrag({ ...st })
    }
    const onUp = () => {
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
      window.removeEventListener('keydown', onKey)
      if (st.moved) {
        if (kind === 'move') {
          if (t.mode === 'manual') updateTask(t.id, { manualStart: st.preview })
          else updateTask(t.id, { mode: 'manual', manualStart: st.preview })
        } else {
          updateTask(t.id, { duration: st.preview })
        }
      }
      setDrag(null)
    }
    const onKey = (ev: KeyboardEvent) => {
      if (ev.key !== 'Escape') return
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
      window.removeEventListener('keydown', onKey)
      setDrag(null) // 未 moved 提交逻辑不触发=取消
    }
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
    window.addEventListener('keydown', onKey)
  }

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
    const x1 = dayNo(f.ef) * dayW
    const x2 = dayNo(t2.es) * dayW
    const mid = Math.max(x1 + 5, (x1 + x2) / 2)
    const d = x2 >= x1 + 10
      ? `M ${x1} ${y1} H ${mid} V ${y2} H ${x2 - 3}`
      : `M ${x1} ${y1} h 6 V ${y1 < y2 ? y2 - 8 : y2 + 8} H ${x2 - 3} V ${y2} H ${x2 - 3}`
    return { d, key: `${l.from}>${l.to}>${l.type}` }
  }).filter(Boolean) as { d: string; key: string }[]

  /** 前锋线折线点（行中心；x=自然日列） */
  const frontLine = useMemo(() => {
    if (!showFront || !cpm.ok) return null
    const off = Math.round((new Date(`${checkDate}T00:00:00Z`).getTime() - startMs) / 86400000)
    if (!Number.isFinite(off) || off < 0) return null
    const pts: { x: number; y: number }[] = []
    project.tasks.forEach((t, i) => {
      const w = frontierWd(t, cpm.rows[t.id], checkWdIdx(checkDate))
      if (w !== null) pts.push({ x: dayNo(w) * dayW, y: i * ROW_H + ROW_H / 2 })
    })
    return pts.length >= 2 ? { pts, checkX: Math.min(off, days) * dayW } : null
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [showFront, cpm, project.tasks, checkDate, startMs, dayNo, dayW, days, colDates])

  /** 检查日期 → 工作日序号（前锋点按进度比例需要工作日刻度） */
  function checkWdIdx(dateIso: string): number {
    let idx = 0
    for (const c of colDates) {
      if (c.iso > dateIso) break
      if (!c.offWork) idx++
    }
    return Math.max(0, idx - 1)
  }

  // 分组行配色序号（斑马口径：橙/青/粉/黄/蓝/绿 六色循环，子行随组）
  const groupColorSeq: number[] = []
  let gc = 0
  for (let i = 0; i < project.tasks.length; i++) {
    if (project.tasks[i].level === 0) gc = (gc + 1) % 6
    groupColorSeq[i] = project.tasks[i].level === 0 ? gc : (i > 0 ? groupColorSeq[i - 1] : gc)
  }

  return (
    <div className="sched-gantt" data-testid="sched-gantt">
      <div className="sched-gantt-toolbar">
        <Button
          size="small"
          type={showFront ? 'primary' : 'default'}
          onClick={() => setShowFront((v) => !v)}
          title="按任务完成进度画前锋线（斑马式进度检查）"
        >
          前锋线
        </Button>
        {project.baseline && (
          <Button
            size="small"
            type={showBase ? 'primary' : 'default'}
            onClick={() => setShowBase((v) => !v)}
            title={`基线条：保存「${project.baseline.name}」时各任务的位置（灰描边细条）`}
          >
            基线
          </Button>
        )}
        {showFront && (
          <DatePicker
            size="small"
            value={dayjs(checkDate)}
            onChange={(d) => d && setCheckDate(d.format('YYYY-MM-DD'))}
            style={{ width: 130 }}
            placeholder="检查日期"
          />
        )}
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
              {colDates.map((c, i) => (
                <div key={i} className={`sched-day-col${c.offWork ? ' sched-day-off' : ''}`} style={{ width: dayW }} title={c.offWork ? '非工作日' : `周${WEEKDAY_LABELS[c.d.getUTCDay()]}`}>
                  <div className="sched-month-cell">{c.d.getUTCDate() === 1 || i === 0 ? `${c.d.getUTCMonth() + 1}月` : ''}</div>
                  <div className="sched-day-cell">{dayW >= 14 || i % 2 === 0 ? c.d.getUTCDate() : ''}</div>
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
            const colorCls = group ? ` sched-group-c${groupColorSeq[i]}` : ''
            const baseRow = showBase && !group ? project.baseline?.rows[t.id] : undefined
            const isDrag = drag?.id === t.id
            const movePreview = isDrag && drag!.kind === 'move' ? drag!.preview : null
            const resizePreview = isDrag && drag!.kind === 'resize' ? drag!.preview : null
            return (
              <div
                key={t.id}
                className={`sched-gantt-row${group ? ' sched-group-row' : ''}${colorCls}${selectedId === t.id ? ' sched-row-selected' : ''}`}
                style={{ height: ROW_H }}
                onClick={() => select(t.id)}
              >
                <div className="sched-gantt-left sched-sticky-left" style={{ width: LEFT_W }}>
                  <div style={{ width: COLS[0].w }} className="sched-gantt-cell sched-row-no">{i + 1}</div>
                  <div style={{ width: COLS[1].w }} className="sched-gantt-cell">
                    <span style={{ paddingLeft: t.level * 14, display: 'inline-flex', alignItems: 'center', gap: 4, minWidth: 0 }}>
                      {group ? <strong className="sched-group-name">▲{t.name}</strong> : (
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
                  <div style={{ width: COLS[2].w }} className="sched-gantt-cell sched-dim">{wbs[i]}</div>
                  <div style={{ width: COLS[3].w }} className="sched-gantt-cell">
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
                  <div style={{ width: COLS[4].w }} className="sched-gantt-cell sched-dim">
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
                      (group ? span : row) ? fmtDate(project.startDate, group ? span!.es : row!.es, cal) : ''
                    )}
                  </div>
                  <div style={{ width: COLS[5].w }} className="sched-gantt-cell sched-dim">
                    {(group ? span : row) ? fmtDate(project.startDate, group ? span!.ef : row!.ef, cal) : ''}
                  </div>
                  <div style={{ width: COLS[6].w }} className="sched-gantt-cell">
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
                  <div style={{ width: COLS[7].w }} className="sched-gantt-cell">
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
                  {baseRow && (
                    <div
                      className="sched-baseline-bar"
                      style={{ left: dayNo(baseRow.es) * dayW, width: Math.max((dayNo(baseRow.ef) - dayNo(baseRow.es)) * dayW, 5) }}
                      title={row && (row.es !== baseRow.es || row.ef !== baseRow.ef)
                        ? `基线：第 ${baseRow.es}~${baseRow.ef} 工作日（当前第 ${row.es}~${row.ef}）`
                        : `基线：第 ${baseRow.es}~${baseRow.ef} 工作日（与当前一致）`}
                    />
                  )}
                  {group ? (
                    span && (
                      <div
                        className="sched-summary-bar"
                        style={{ left: dayNo(span.es) * dayW, width: Math.max((dayNo(span.ef) - dayNo(span.es)) * dayW, 8) }}
                        title={`${t.name}：${span.es} ~ ${span.ef} 工作日`}
                      />
                    )
                  ) : t.isMilestone ? (
                    row && (
                      <div
                        className={`sched-milestone${isDrag && drag!.kind === 'move' ? ' sched-bar-dragging' : ''}`}
                        style={{ left: dayNo(movePreview ?? row.es) * dayW - 7 }}
                        title={`${t.name}（里程碑，拖动可定位）`}
                        onMouseDown={(e) => beginDrag(e, t, 'move')}
                      />
                    )
                  ) : (
                    row && (
                      <>
                        <div
                          className={`sched-bar${crit ? ' sched-bar-critical' : ''}${manual ? ' sched-bar-manual' : ''}${isDrag ? ' sched-bar-dragging' : ''}`}
                          style={{
                            left: dayNo(movePreview ?? row.es) * dayW,
                            width: Math.max(
                              (dayNo((movePreview ?? row.es) + (resizePreview ?? dur)) - dayNo(movePreview ?? row.es)) * dayW,
                              6,
                            ),
                          }}
                          title={`${t.name}：第 ${row.es}~${row.ef} 工作日${manual ? '（手动锁定，拖动移位）' : crit ? '（关键，拖动=转手动锁定）' : `，总时差 ${row.tf} 天，拖动=转手动锁定`}`}
                          onMouseDown={(e) => beginDrag(e, t, 'move')}
                        >
                          {t.progress > 0 && (
                            <div className="sched-bar-progress" style={{ width: `${Math.min(100, t.progress)}%` }} />
                          )}
                          <div className="sched-resize-handle" onMouseDown={(e) => beginDrag(e, t, 'resize')} title="拖动改工期" />
                        </div>
                        {!manual && !isDrag && row.tf > 0 && (
                          <div className="sched-float" style={{ left: dayNo(row.ef) * dayW, width: row.tf * dayW }} title={`总时差 ${row.tf} 天`} />
                        )}
                        {isDrag && (
                          <div
                            className="sched-drag-tip"
                            style={{ left: dayNo(drag!.kind === 'move' ? drag!.preview : row.es + drag!.preview) * dayW }}
                          >
                            {drag!.kind === 'move'
                              ? `第 ${row.es} → ${drag!.preview} 工作日${manual ? '' : '（转手动）'}`
                              : `工期 ${dur} → ${drag!.preview} 天`}
                          </div>
                        )}
                      </>
                    )
                  )}
                </div>
              </div>
            )
          })}
          {/* 覆盖层：非工作日底纹 + 今日线 + 前锋线 + 搭接箭线 */}
          <div className="sched-overlay" style={{ left: LEFT_W, width: chartW, height: totalH + 40 }}>
            <svg width={chartW} height={totalH + 40} style={{ position: 'absolute', inset: 0, pointerEvents: 'none' }}>
              <defs>
                <marker id="sched-arrow" markerWidth="7" markerHeight="7" refX="6" refY="3.5" orient="auto">
                  <path d="M0,0 L7,3.5 L0,7 z" fill="var(--sched-link, #94a3b8)" />
                </marker>
              </defs>
              {colDates.map((c, i) => c.offWork && (
                <rect key={`off${i}`} x={i * dayW} y={0} width={dayW} height={totalH + 40} className="sched-weekend" />
              ))}
              {linkPaths.map((p) => (
                <path key={p.key} d={p.d} className="sched-link-line" markerEnd="url(#sched-arrow)" />
              ))}
              {todayCol !== null && <line x1={todayCol * dayW} y1={0} x2={todayCol * dayW} y2={totalH + 40} className="sched-today-line" />}
              {deadlineCol !== null && (
                <line x1={deadlineCol * dayW + dayW} y1={0} x2={deadlineCol * dayW + dayW} y2={totalH + 40} className="sched-deadline-line">
                  <title>{`目标竣工 ${project.deadline}（倒排校核线）`}</title>
                </line>
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

/** 工作日序号 → M/D 文本（按日历换算日期） */
function fmtDate(startISO: string, workdayIdx: number, cal: SchedProject['calendar']): string {
  const d = wdToDate(startISO, workdayIdx, cal)
  return `${d.getUTCMonth() + 1}/${d.getUTCDate()}`
}
