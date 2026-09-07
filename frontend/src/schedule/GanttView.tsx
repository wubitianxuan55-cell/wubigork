/**
 * schedule/GanttView.tsx — 横道图（甘特图）视图（v4.112.0 斑马 UI 对齐）
 *
 * 视觉/交互对齐广联达斑马进度（官网截图口径）：
 *  - 自然日时间轴 + 周末/节假日底纹（CPM 仍按工作日计算，条形跨非工作日渲染）；
 *  - 行号列、分组行彩色底纹（五色循环 + 左色条）；
 *  - 关键红条、分组黑汇总条、里程碑黑菱形、总时差尾巴、搭接箭线、前锋线。
 * 列（v4.127 刀B 补齐 Project 口径）：行号 | 任务名称 | WBS | 工期 | 开始 | 完成 |
 * 最迟开始 | 最迟完成 | 总时差 | 自由时差 | 模式 | 前置 | 后续 | 成本。
 *
 * v4.131 刀C：工作台改「表格窗格 | 画布窗格」真双栏——
 *  - 表格窗格（左）与时间画布（右）各自独立横滚、纵向滚动同步（画布 scroll
 *    → 表格体 translateY，直改 DOM 不走 setState）；
 *  - 分隔条可拖：拖窄=收纳表格（最小=行号+名称，画布全屏）、拖宽=展开，
 *    双击复位全列；列显隐菜单（行号/名称固定，其余单列可藏）；均持久化 chatPrefs；
 *  - 时间刻度防竖排：日期格 nowrap，窄刻度（<10px）只画格不写数。
 */
import React, { useMemo, useRef, useState } from 'react'
import { Button, Checkbox, DatePicker, Input, InputNumber, Popover, Select } from 'antd'
import { ColumnWidthOutlined, DeleteOutlined, TeamOutlined, ZoomInOutlined, ZoomOutOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import type { CpmResult, LinkType, SchedProject, SchedTask } from './types'
import { isWorkingDate, normalizeCalendar, wdToDate } from './calendar'
import { dropToWd, resizeToDuration, workdayOffsets } from './drag'
import { computeCosts, type CostResult } from './cost'
import { fmtLinkRefs, ganttLinkPath } from './ganttLinks'
import { descendantIds, isGroupRow, useScheduleStore, type PredDraft } from './store'
import { TaskResourceEditor } from './ResourcePanel'
import { fmtCost, hasCostData } from './costUi'
import { GANTT_COLS, GANTT_FIXED_KEYS, GANTT_LEFT_W_FULL, clampTableW, visibleCols, visibleLeftW } from './ganttCols'
import { loadChatPrefs, saveChatPrefs } from './chatPrefs'

const ROW_H = 30
const W: Record<string, number> = Object.fromEntries(GANTT_COLS.map((c) => [c.key, c.w]))
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

/** 入边（前置）/出边（后续）引用（刀B 前置/后续列数据源） */
function predsOf(id: string, project: SchedProject) {
  return project.links.filter((l) => l.to === id)
}
function succsOf(id: string, project: SchedProject) {
  return project.links.filter((l) => l.from === id)
}
function taskNameOf(project: SchedProject, id: string): string {
  return project.tasks.find((t) => t.id === id)?.name ?? id
}
/** 任务 id → 行号（Project 引用文本口径，1-based） */
function noOf(project: SchedProject): (id: string) => number {
  return (id) => project.tasks.findIndex((t) => t.id === id) + 1
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

/** 分组行成本汇总（子孙叶任务 total 求和；复用 groupSpan 的扁平 WBS 滚动口径，同甘特汇总条） */
function groupCost(project: SchedProject, costs: CostResult, idx: number): number {
  const ids = descendantIds(project.tasks, project.tasks[idx].id)
  let sum = 0
  for (const id of ids) sum += costs.rows[id]?.total ?? 0
  return sum
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

  // ── 刀C 双栏：表格窗格宽度/列显隐（持久化 chatPrefs）────────────
  const [tableW, setTableW] = useState<number>(() => loadChatPrefs().ganttTableW)
  const [hide, setHide] = useState<string[]>(() => loadChatPrefs().ganttHide)
  const splitRef = useRef<{ startX: number; startW: number; w: number } | null>(null)
  const tbodyRef = useRef<HTMLDivElement | null>(null)
  const cols = useMemo(() => visibleCols(hide), [hide])
  const leftW = useMemo(() => visibleLeftW(hide), [hide])
  const effW = Math.min(tableW, leftW)
  const colOn = (key: string) => !hide.includes(key)

  /** 分隔条拖动：右移加宽表格、左移收纳（最小=行号+名称）；拖完持久化 */
  const beginSplit = (e: React.MouseEvent) => {
    e.preventDefault()
    splitRef.current = { startX: e.clientX, startW: tableW, w: tableW }
    const onMove = (ev: MouseEvent) => {
      const s = splitRef.current!
      s.w = clampTableW(s.startW + (ev.clientX - s.startX), leftW)
      setTableW(s.w)
    }
    const onUp = () => {
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
      const s = splitRef.current
      splitRef.current = null
      if (s) saveChatPrefs({ ganttTableW: s.w }) // 提交在渲染期外（updater 禁副作用）
    }
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
  }
  const resetSplit = () => {
    const w = clampTableW(GANTT_LEFT_W_FULL, leftW)
    setTableW(w)
    saveChatPrefs({ ganttTableW: w })
  }
  /** 单列显隐（行号/名称固定不藏） */
  const toggleCol = (key: string, on: boolean) => {
    const next = on ? hide.filter((k) => k !== key) : [...hide, key]
    setHide(next)
    saveChatPrefs({ ganttHide: next })
  }
  /** 画布纵向滚动 → 表格体同步（直改 DOM，滚动帧不走 setState） */
  const onCanvasScroll = (e: React.UIEvent<HTMLDivElement>) => {
    if (tbodyRef.current) tbodyRef.current.style.transform = `translateY(${-e.currentTarget.scrollTop}px)`
  }

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

  // 资源成本（刀3）：computeCosts 每次 render 重算（与 CPM 同范式，工期变→成本自动变）。
  // 无任何资源/成本数据时成本列留空（诚实呈现，不留 ¥0 假象）；CPM 未过成本不出（fail-closed）。
  const costs = useMemo(() => computeCosts(project, cpm), [project, cpm])
  const costShown = hasCostData(project) && costs.ok

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

  /** 依赖线（刀B）：按四种搭接类型取各自锚点画正交折线 */
  const linkPaths = linkRows.map(({ l, fi, ti }) => {
    const f = cpm.rows[l.from]
    const t2 = cpm.rows[l.to]
    if (!f || !t2) return null
    return {
      d: ganttLinkPath(l.type, {
        x1s: dayNo(f.es) * dayW,
        x1f: dayNo(f.ef) * dayW,
        y1: fi! * ROW_H + ROW_H / 2,
        x2s: dayNo(t2.es) * dayW,
        x2f: dayNo(t2.ef) * dayW,
        y2: ti! * ROW_H + ROW_H / 2,
      }),
      key: `${l.from}>${l.to}>${l.type}`,
    }
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
        {/* 列显隐（刀C 双栏）：行号/名称固定，其余单列可藏让位画布 */}
        <Popover
          trigger="click"
          placement="bottomRight"
          title="显示列（行号与任务名称固定）"
          content={
            <div className="sched-colmenu" data-testid="sched-gantt-colmenu">
              {GANTT_COLS.filter((c) => !GANTT_FIXED_KEYS.includes(c.key)).map((c) => (
                <label key={c.key} className="sched-colmenu-item">
                  <Checkbox checked={colOn(c.key)} onChange={(e) => toggleCol(c.key, e.target.checked)} />
                  <span>{c.label}</span>
                </label>
              ))}
            </div>
          }
        >
          <Button size="small" icon={<ColumnWidthOutlined />} data-testid="sched-gantt-cols-btn" title="显示/隐藏表格列（让位时间画布）">
            列
          </Button>
        </Popover>
        <Button size="small" icon={<ZoomOutOutlined />} onClick={() => setDayW((w) => DAY_W_STEPS[Math.max(0, DAY_W_STEPS.indexOf(w) - 1)] ?? w)} />
        <Button size="small" icon={<ZoomInOutlined />} onClick={() => setDayW((w) => DAY_W_STEPS[Math.min(DAY_W_STEPS.length - 1, DAY_W_STEPS.indexOf(w) + 1)] ?? w)} />
      </div>
      <div className="sched-gantt-frame">
        {/* 表格窗格（左）：宽度=分隔条拖动（chatPrefs 持久化），列显隐收纳 */}
        <div className="sched-gantt-tablepane" data-testid="sched-gantt-table" style={{ width: effW }}>
          <div className="sched-gantt-row sched-gantt-head" style={{ height: 40 }}>
            {cols.map((c) => (
              <div key={c.key} style={{ width: c.w }} className="sched-gantt-cell sched-th">{c.label}</div>
            ))}
          </div>
          <div className="sched-gantt-tbody" ref={tbodyRef}>
            <div style={{ width: leftW }}>
              {project.tasks.map((t, i) => {
                const row = cpm.rows[t.id]
                const group = isGroupRow(project.tasks, i)
                const span = group ? groupSpan(project, cpm, i) : null
                const manual = t.mode === 'manual'
                const dur = t.isMilestone ? 0 : Math.max(0, Math.round(t.duration))
                const colorCls = group ? ` sched-group-c${groupColorSeq[i]}` : ''
                return (
                  <div
                    key={t.id}
                    className={`sched-gantt-row${group ? ' sched-group-row' : ''}${colorCls}${selectedId === t.id ? ' sched-row-selected' : ''}`}
                    style={{ height: ROW_H }}
                    onClick={() => select(t.id)}
                  >
                    <div style={{ width: W.no }} className="sched-gantt-cell sched-row-no">{i + 1}</div>
                    <div style={{ width: W.name }} className="sched-gantt-cell">
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
                        {!group && (
                          <Popover trigger="click" placement="left" content={<TaskResourceEditor task={t} />} title={`「${t.name}」资源与成本`}>
                            <Button
                              size="small"
                              type="text"
                              className="sched-res-entry"
                              data-testid={`sched-task-res-${t.id}`}
                              icon={<TeamOutlined />}
                              title="资源与成本（分配挂载、固定成本）"
                            >
                              {(project.assignments?.filter((a) => a.taskId === t.id).length ?? 0) || ''}
                            </Button>
                          </Popover>
                        )}
                      </span>
                    </div>
                    {colOn('wbs') && <div style={{ width: W.wbs }} className="sched-gantt-cell sched-dim">{wbs[i]}</div>}
                    {colOn('dur') && (
                      <div style={{ width: W.dur }} className="sched-gantt-cell">
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
                    )}
                    {colOn('start') && (
                      <div style={{ width: W.start }} className="sched-gantt-cell sched-dim">
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
                    )}
                    {colOn('finish') && (
                      <div style={{ width: W.finish }} className="sched-gantt-cell sched-dim">
                        {(group ? span : row) ? fmtDate(project.startDate, group ? span!.ef : row!.ef, cal) : ''}
                      </div>
                    )}
                    {/* 六时参（刀B）：LS/LF/TF/FF——手动任务逆推不回传，诚实显示 —（不放假数据） */}
                    {colOn('ls') && (
                      <div style={{ width: W.ls }} className="sched-gantt-cell sched-dim" title="最迟开始（逆推）">
                        {group || manual ? (manual && !group ? '—' : '') : fmtDate(project.startDate, row!.ls, cal)}
                      </div>
                    )}
                    {colOn('lf') && (
                      <div style={{ width: W.lf }} className="sched-gantt-cell sched-dim" title="最迟完成（逆推）">
                        {group || manual ? (manual && !group ? '—' : '') : fmtDate(project.startDate, row!.lf, cal)}
                      </div>
                    )}
                    {colOn('tf') && (
                      <div style={{ width: W.tf }} className={`sched-gantt-cell${!group && !manual && row?.tf === 0 ? ' sched-critical-text' : ' sched-dim'}`} title={manual ? '手动模式不参与时差计算' : '总时差 = LS − ES（0=关键）'}>
                        {group ? '' : manual ? '—' : row?.tf}
                      </div>
                    )}
                    {colOn('ff') && (
                      <div style={{ width: W.ff }} className="sched-gantt-cell sched-dim" title={manual ? '手动模式不参与时差计算' : '自由时差 = 紧后 ES 最小值 − EF'}>
                        {group ? '' : manual ? '—' : row?.ff}
                      </div>
                    )}
                    {colOn('mode') && (
                      <div style={{ width: W.mode }} className="sched-gantt-cell">
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
                    )}
                    {colOn('preds') && (
                      <div style={{ width: W.preds }} className="sched-gantt-cell">
                        {!group && (
                          <Popover trigger="click" placement="left" content={<PredEditor task={t} />} title={`「${t.name}」前置任务`}>
                            <Button
                              size="small"
                              type="text"
                              className="sched-link-cell"
                              data-testid={`sched-preds-${t.id}`}
                              title={predsOf(t.id, project).map((l) => taskNameOf(project, l.from)).join('、') || '无前置，点击添加'}
                            >
                              {fmtLinkRefs(predsOf(t.id, project).map((l) => ({ id: l.from, type: l.type, lag: l.lag })), noOf(project))}
                            </Button>
                          </Popover>
                        )}
                      </div>
                    )}
                    {colOn('succ') && (
                      <div style={{ width: W.succ }} className="sched-gantt-cell">
                        {!group && (
                          <span
                            className="sched-dim sched-link-cell-text"
                            data-testid={`sched-succ-${t.id}`}
                            title={succsOf(t.id, project).map((l) => taskNameOf(project, l.to)).join('、') || '无后续任务'}
                          >
                            {fmtLinkRefs(succsOf(t.id, project).map((l) => ({ id: l.to, type: l.type, lag: l.lag })), noOf(project))}
                          </span>
                        )}
                      </div>
                    )}
                    {colOn('cost') && (
                      <div style={{ width: W.cost }} className="sched-gantt-cell sched-cost-cell" data-testid={`sched-cost-${t.id}`}>
                        {costShown && (group ? (
                          <span className="sched-dim" title="分组汇总：子孙叶任务成本求和">{fmtCost(groupCost(project, costs, i))}</span>
                        ) : (
                          <span title="明细合计：固定成本 + 分配成本（随工期实时重算）">{fmtCost(costs.rows[t.id]?.total ?? 0)}</span>
                        ))}
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          </div>
        </div>
        {/* 分隔条：拖动收纳/展开表格窗格，双击复位全列 */}
        <div
          className="sched-gantt-split"
          data-testid="sched-gantt-split"
          onMouseDown={beginSplit}
          onDoubleClick={resetSplit}
          title="拖动收纳/展开表格 · 双击复位"
        />
        {/* 画布窗格（右）：时间轴+条形+覆盖层；横滚独立，纵滚同步表格体 */}
        <div className="sched-gantt-canvaspane" data-testid="sched-gantt-canvas" onScroll={onCanvasScroll}>
          <div className="sched-gantt-canvas-inner" style={{ width: chartW }}>
            <div className="sched-gantt-row sched-gantt-head" style={{ height: 40, width: chartW }}>
              <div className="sched-gantt-datehead" style={{ width: chartW }}>
                {colDates.map((c, i) => (
                  <div key={i} className={`sched-day-col${c.offWork ? ' sched-day-off' : ''}`} style={{ width: dayW }} title={c.offWork ? '非工作日' : `周${WEEKDAY_LABELS[c.d.getUTCDay()]}`}>
                    <div className="sched-month-cell">{c.d.getUTCDate() === 1 || i === 0 ? `${c.d.getUTCMonth() + 1}月` : ''}</div>
                    <div className="sched-day-cell">{dayW >= 14 || (dayW >= 10 && i % 2 === 0) ? c.d.getUTCDate() : ''}</div>
                  </div>
                ))}
              </div>
            </div>
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
            <div className="sched-overlay" style={{ left: 0, width: chartW, height: totalH + 40 }}>
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
