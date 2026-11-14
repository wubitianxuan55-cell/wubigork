/**
 * schedule/gantt/GanttTable.tsx — 表格窗格内容（自 GanttView.tsx 拆分，行为零变化）
 *
 * 列头行（宽=可见列合计，收纳时多余列裁剪不压缩）+ 表体逐行渲染：
 * 合成组头行（颜色标记/折叠态/汇总跨度）与任务行（名称内联编辑/工期+单位 chip/
 * 进度/手动开始/六时参/模式/前置右键编辑/后续/成本/自定义字段单元格）。
 * 列宽 W、行高 ROW_H、格式口径 fmtDate 等来自 ganttUtil，与画布行同源；
 * 逐行派生与行外壳单源自 rowDerive.deriveRows + GanttRowShell（FE7-05，
 * 与 GanttBars 共用，本组件只留表格专有内容）。
 *
 * W5（交互一致性）增量，全部 optional、未接线时行为与拆分前逐字一致：
 *  - 单元格**双击**开统一就地编辑弹层（InplaceEditField：名称 text / 工期 number /
 *    计划开始·完成 date / 进度 number / 自定义字段按注册表类型 / 日期区间列 dateRange），
 *    提交走既有 `onUpdateTask`；**旧的单击直编控件全部保留**（零回归）；
 *  - 表头每个可筛列加筛选按钮（五类编辑器），状态经 `colFilters` + `onColFiltersChange`
 *    上抛，本组件不持状态（GanttView 接线在视图侧）。
 *
 * W6 键盘可达（行为线，交互语义与键盘可达）：
 *  - 行容器 roving tabindex：**选中行**（无选中时=首个任务行）`tabIndex=0`，其余行 `-1`，
 *    分组行一律 `-1`（组头不是选中对象，不参与 ↑/↓ 序列）；
 *  - `↑`/`↓` 移动选中并把焦点带到新行（DOM 焦点跟随，roving 才不只是个属性）；
 *  - `Enter` 打开任务检查器（与右键「任务检查器」同一条回调 `onInspect`）；
 *  - `F2` 开名称列就地编辑（受控 open：双击与 F2 落到同一个状态源，不各造一套）；
 *  - 行内输入/编辑器里的按键不抢（Enter/F2 在 input/textarea 内照旧给控件本身）；
 *  - 表头筛选按钮 `aria-expanded` 随 Popover 真实开合，`Esc` 收起并把焦点还给该按钮。
 */
import React, { useEffect, useMemo, useRef, useState } from 'react'
import { Button, Checkbox, DatePicker, Input, InputNumber, Popover, Select, type MenuProps } from 'antd'
import { TeamOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import type { CpmResult, SchedProject, SchedTask } from '../types'
import type { CostResult } from '../cost'
import { isCd } from '../cpm'
import { dateToWd, isoOf, wdToDate } from '../calendar'
import { customValueOf } from '../customFields'
import { fmtLinkRefs } from '../ganttLinks'
import { fmtCost } from '../costUi'
import { GANTT_CUSTOM_KEYS, type GanttCol } from '../ganttCols'
import { GANTT_RULER_H } from '../timescale'
import type { GanttRow } from '../ganttGroup'
import {
  clearColFilter,
  COL_FILTER_ENUM_OPTIONS,
  COL_FILTER_KIND,
  isColFilterActive,
  setColFilter,
  type ColFilterKind,
  type ColFilterValue,
  type ColFilters,
} from '../ganttFilter'
import { inplaceKindOfCol, inplaceSpecOfCol } from '../inplaceEdit'
import { ROW_H, W, CUSTOM_FIELD_BY_KEY, fmtDate, groupCost, noOf, predsOf, succsOf, taskNameOf } from './ganttUtil'
import { deriveRows } from './rowDerive'
import { GanttRowShell } from './GanttRowShell'
import { InplaceEditField, type InplaceCommitValue } from './InplaceEditField'
import { PredEditor } from './PredEditor'
import { CustomFieldCell } from './CustomFieldCell'
import { TaskResourceEditor } from '../ResourcePanel'

interface GanttTableProps {
  leftW: number
  cols: GanttCol[]
  tbodyRef: React.Ref<HTMLDivElement>
  rows: GanttRow[]
  wbs: string[]
  project: SchedProject
  cpm: CpmResult
  selectedId: string | null
  costShown: boolean
  costs: CostResult
  groupColorSeq: number[]
  collapsedKeys: Set<string>
  colOn: (key: string) => boolean
  rowDim: (id: string) => boolean
  onSelect: (id: string) => void
  onUpdateTask: (id: string, patch: Partial<SchedTask>) => void
  onToggleCollapse: (key: string) => void
  rowMenu: (t: SchedTask, group: boolean) => MenuProps['items']
  onRowMenuClick: (key: string, t: SchedTask) => void
  /** 树形拖动落库（可选）：表格窗格专用，画布不传 */
  onMoveRow?: (dragId: string, targetId: string, zone: 'before' | 'after' | 'inside') => void
  /** 任务检查器（W6 键盘：行上 Enter 与右键菜单「任务检查器」同一回调；optional=未接线不报错） */
  onInspect?: (taskId: string) => void
  /** 列级筛选状态（W5，optional：未接线=不显示筛选按钮，行为与拆分前一致） */
  colFilters?: ColFilters
  /** 列筛变更上抛（受控；未接线=只读模式，不显示筛选入口） */
  onColFiltersChange?: (next: ColFilters) => void
  /** 日期区间列键（W5，optional：尾部追加；缺省空=无此类列） */
  rangeCols?: string[]
  /** 区间列当前值（optional；缺省空区间） */
  rangeValueOf?: (t: SchedTask, colKey: string) => { from: string; to: string }
  /** 区间提交（optional；未接线=区间列只读） */
  onRangeCommit?: (id: string, colKey: string, range: { from: string | null; to: string | null }) => void
}

const PARTITION_ROMAN = ['I', 'II', 'III', 'IV', 'V', 'VI', 'VII', 'VIII', 'IX', 'X', 'XI', 'XII']

/** WBS `3.2` → 分区 `III`（ZPert 计划编制首列口径）。 */
function partitionOf(wbs: string | undefined): string {
  const root = Number((wbs ?? '').split('.')[0])
  return Number.isInteger(root) && root > 0 ? (PARTITION_ROMAN[root - 1] ?? String(root)) : ''
}

/** 空列筛表（常量引用，避免每次 render 新建对象把 memo 打穿） */
const EMPTY_COL_FILTERS: ColFilters = {}

/**
 * 列筛编辑器（W5 五类）：文本包含 / 数值比较（≥ ≤ =）/ 日期区间 / 枚举集合 + 空值口径。
 * 内容由 GanttTable 表头 Popover 承载；状态全在外部（受控），本组件只发变更。
 */
const ColFilterEditor: React.FC<{
  colKey: string
  label: string
  kind: ColFilterKind
  value?: ColFilterValue
  onChange: (v: ColFilterValue) => void
  onClear: () => void
}> = ({ colKey, label, kind, value, onChange, onClear }) => {
  const cur: ColFilterValue = value
    ? { ...value }
    : kind === 'number'
      ? { kind, op: 'ge' }
      : { kind }
  const patch = (p: Partial<ColFilterValue>) => onChange({ ...cur, ...p, kind })
  return (
    <div className="sched-inplace-filter" data-testid={`sched-colfilter-editor-${colKey}`}>
      {kind === 'text' && (
        <div className="sched-inplace-filter-row">
          <span className="sched-dim">包含</span>
          <Input
            size="small"
            allowClear
            value={cur.text ?? ''}
            placeholder={`筛选${label}`}
            onChange={(e) => patch({ text: e.target.value })}
            data-testid={`sched-colfilter-text-${colKey}`}
          />
        </div>
      )}
      {kind === 'number' && (
        <div className="sched-inplace-filter-row">
          <Select
            size="small"
            style={{ width: 60 }}
            value={cur.op ?? 'ge'}
            onChange={(op) => patch({ op })}
            options={[
              { value: 'ge', label: '≥' },
              { value: 'le', label: '≤' },
              { value: 'eq', label: '=' },
            ]}
            data-testid={`sched-colfilter-op-${colKey}`}
          />
          <InputNumber
            size="small"
            value={typeof cur.num === 'number' && Number.isFinite(cur.num) ? cur.num : null}
            onChange={(v) => patch({ num: typeof v === 'number' && Number.isFinite(v) ? v : undefined })}
            data-testid={`sched-colfilter-num-${colKey}`}
          />
        </div>
      )}
      {kind === 'date' && (
        <div className="sched-inplace-filter-row">
          <DatePicker.RangePicker
            size="small"
            format="YYYY-MM-DD"
            placeholder={['起', '止']}
            value={[cur.from ? dayjs(cur.from) : null, cur.to ? dayjs(cur.to) : null]}
            onChange={(v) =>
              patch({
                from: v?.[0] ? v[0].format('YYYY-MM-DD') : '',
                to: v?.[1] ? v[1].format('YYYY-MM-DD') : '',
              })
            }
            data-testid={`sched-colfilter-date-${colKey}`}
          />
        </div>
      )}
      {kind === 'enum' && (
        <Checkbox.Group
          value={cur.set ?? []}
          onChange={(vals) => patch({ set: vals.map((v) => String(v)) })}
          options={(COL_FILTER_ENUM_OPTIONS[colKey] ?? []).map((o) => ({ label: o.label, value: o.value }))}
          data-testid={`sched-colfilter-enum-${colKey}`}
        />
      )}
      <div className="sched-inplace-filter-row">
        <span className="sched-dim">空值</span>
        <Select
          size="small"
          style={{ width: 96 }}
          value={cur.empty ?? 'any'}
          onChange={(e) => patch({ empty: e })}
          options={[
            { value: 'any', label: '不限' },
            { value: 'empty', label: '只看空' },
            { value: 'nonEmpty', label: '只看有值' },
          ]}
          data-testid={`sched-colfilter-empty-${colKey}`}
        />
        <Button size="small" type="link" onClick={onClear} data-testid={`sched-colfilter-clear-${colKey}`}>
          清除
        </Button>
      </div>
    </div>
  )
}

export const GanttTable: React.FC<GanttTableProps> = ({
  leftW,
  cols,
  tbodyRef,
  rows,
  wbs,
  project,
  cpm,
  selectedId,
  costShown,
  costs,
  groupColorSeq,
  collapsedKeys,
  colOn,
  rowDim,
  onSelect,
  onUpdateTask,
  onToggleCollapse,
  rowMenu,
  onRowMenuClick,
  onMoveRow,
  onInspect,
  colFilters,
  onColFiltersChange,
  rangeCols,
  rangeValueOf,
  onRangeCommit,
}) => {
  const cal = project.calendar
  /** 树形拖动：源行 + 当前落点行/区域（before|after|inside），仅表格窗格 */
  const [dragSrc, setDragSrc] = React.useState<string | null>(null)
  const [dropAt, setDropAt] = React.useState<{ id: string; zone: 'before' | 'after' | 'inside' } | null>(null)
  const dragEnabled = !!onMoveRow
  /** 落点：上 25% = 之前、下 25% = 之后、中段 = 成为子工作（仅分区行可接收） */
  const zoneOf = (e: React.DragEvent<HTMLDivElement>, t: SchedTask): 'before' | 'after' | 'inside' => {
    const rect = e.currentTarget.getBoundingClientRect()
    // jsdom/RTL 的 DragEvent 不携带 clientY，这里对缺失值回落到 0（真机事件始终有值）
    const y = (Number.isFinite(e.clientY) ? e.clientY : 0) - rect.top
    const h = rect.height || ROW_H
    if (t.level === 0 && y > h * 0.25 && y < h * 0.75) return 'inside'
    return y < h / 2 ? 'before' : 'after'
  }

  const cf: ColFilters = colFilters ?? EMPTY_COL_FILTERS
  /** ── W6 键盘态（全部是本组件的**视图态**，不入文件/偏好）────────────────
   *  openFilterKey：当前打开的表头筛选列（受控 Popover，aria-expanded 的事实源）；
   *  nameEditId：名称列就地编辑的受控 open（双击 / F2 同一入口）；
   *  filterBtnRefs：各列表头筛选按钮的 DOM 引用（Esc 收起后把焦点还回去）。 */
  const [openFilterKey, setOpenFilterKey] = useState<string | null>(null)
  const [nameEditId, setNameEditId] = useState<string | null>(null)
  const filterBtnRefs = useRef(new Map<string, HTMLButtonElement>())
  /** 任务行序列（roving 与 ↑/↓ 的唯一定义域）：只走叶任务行，带上渲染行下标 */
  const taskRowSeq = useMemo(
    () => rows.flatMap((r, rowIdx) => (r.kind === 'task' ? [{ id: project.tasks[r.idx].id, rowIdx }] : [])),
    [rows, project.tasks],
  )
  /** roving 锚：选中任务在序列里 → 它；否则第一个任务行（否则整张表 Tab 不进去） */
  const rovingId = taskRowSeq.some((x) => x.id === selectedId) ? selectedId : (taskRowSeq[0]?.id ?? null)
  /** 表头筛选 Popover 的 Esc 收起：关闭 + 焦点还给打开它的那个按钮（键盘用户不落回 body） */
  useEffect(() => {
    if (!openFilterKey) return
    const onKey = (ev: KeyboardEvent) => {
      if (ev.key !== 'Escape') return
      ev.preventDefault()
      const key = openFilterKey
      setOpenFilterKey(null)
      filterBtnRefs.current.get(key)?.focus()
    }
    window.addEventListener('keydown', onKey, true)
    return () => window.removeEventListener('keydown', onKey, true)
  }, [openFilterKey])
  /** 行级键盘：↑/↓ 移选中（焦点跟随）、Enter 检查器、F2 改名称。
   *  行内控件（输入框/下拉/按钮/链接/可编辑区）里的按键原样给控件——Enter 在按钮上
   *  是「激活按钮」、在输入框里是「提交」，行上抢走会把这些默认动作吃掉。 */
  const onRowKeyDown = (e: React.KeyboardEvent<HTMLDivElement>, t: SchedTask) => {
    const target = e.target as HTMLElement | null
    if (
      target &&
      typeof target.closest === 'function' &&
      target.closest('input,textarea,select,button,a,[contenteditable="true"],[role="button"]')
    ) {
      return
    }
    if (e.key === 'Enter') {
      e.preventDefault()
      e.stopPropagation()
      onInspect?.(t.id)
      return
    }
    if (e.key === 'F2') {
      e.preventDefault()
      e.stopPropagation()
      setNameEditId(t.id)
      return
    }
    if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp') return
    const at = taskRowSeq.findIndex((x) => x.id === t.id)
    if (at < 0) return
    const next = taskRowSeq[at + (e.key === 'ArrowDown' ? 1 : -1)]
    e.preventDefault()
    if (!next) return
    onSelect(next.id)
    // 行节点按任务 id 做 key，选中变化不会重建 DOM——同步把焦点搬到目标行
    const rowEl = e.currentTarget.parentElement?.children[next.rowIdx] as HTMLElement | undefined
    rowEl?.focus?.()
    rowEl?.scrollIntoView?.({ block: 'nearest', behavior: 'auto' })
  }
  /** 工作日序号 → ISO 日期（date 类就地编辑弹层的预填口径；无行=空串） */
  const isoAt = (wd: number | undefined): string =>
    wd === undefined || !Number.isFinite(wd) ? '' : isoOf(wdToDate(project.startDate, wd, cal))
  /** 表头筛选按钮：列在 COL_FILTER_KIND 内且已接线 onColFiltersChange 才出现 */
  const filterOf = (key: string, label: string): React.ReactNode => {
    const kind = COL_FILTER_KIND[key]
    if (!kind || !onColFiltersChange) return null
    const cur = cf[key]
    const active = isColFilterActive(cur)
    const opened = openFilterKey === key
    return (
      <Popover
        trigger="click"
        placement="bottomLeft"
        open={opened}
        onOpenChange={(next) => setOpenFilterKey((current) => (next ? key : current === key ? null : current))}
        title={`筛选「${label}」`}
        content={
          <ColFilterEditor
            colKey={key}
            label={label}
            kind={kind}
            value={cur}
            onChange={(v) => onColFiltersChange(setColFilter(cf, key, v))}
            onClear={() => onColFiltersChange(clearColFilter(cf, key))}
          />
        }
      >
        <button
          type="button"
          ref={(el) => {
            if (el) filterBtnRefs.current.set(key, el)
            else filterBtnRefs.current.delete(key)
          }}
          className={`sched-inplace-filter-btn${active ? ' sched-inplace-filter-on' : ''}`}
          data-testid={`sched-colfilter-${key}`}
          aria-label={active ? `已按「${label}」筛选（点击修改）` : `按「${label}」筛选`}
          aria-haspopup="dialog"
          aria-expanded={opened}
          title={active ? `已按「${label}」筛选（点击修改）` : `按「${label}」筛选`}
        />
      </Popover>
    )
  }
  return (
    <>
      {/* 列头行宽=全部可见列合计（leftW）：表格收纳时多余列被裁剪隐藏，
          绝不被 flex 压缩——压缩会造成列头与数据行永久错位（v4.142 修正） */}
      <div className="sched-gantt-row sched-gantt-head" style={{ height: GANTT_RULER_H, width: leftW }}>
        {cols.map((c) => (
          <div key={c.key} style={{ width: c.w }} className="sched-gantt-cell sched-th">
            {c.label}
            {filterOf(c.key, c.label)}
          </div>
        ))}
        {/* 日期区间列（W5 optional）：尾部追加，宽度取列宽表缺省 120 */}
        {(rangeCols ?? []).map((rk) => (
          <div key={`range-${rk}`} style={{ width: W[rk] ?? 120 }} className="sched-gantt-cell sched-th">
            {rk}
          </div>
        ))}
      </div>
      <div className="sched-gantt-tbody" ref={tbodyRef}>
        <div style={{ width: leftW }}>
          {(() => {
            // 逐行派生单源（FE7-05）：synSeq/判定/字段派生在 rowDerive.deriveRows，
            // 本组件只渲染表格窗格专有内容
            const derived = deriveRows(rows, project, cpm, groupColorSeq, rowDim)
            return derived.map((d, rowIdx) => {
            if (d.syn) {
              // 合成组头行（分组模式）：名称+计数+汇总跨度，点击折叠/展开
              const r = d.r
              const open = !collapsedKeys.has(r.key)
              return (
              <div
                key={r.key}
                data-testid={`sched-group-row-${rowIdx}`}
                className={`sched-gantt-row sched-group-row sched-group-c${d.synSeq % 6}`}
                style={{ height: ROW_H }}
                onClick={() => onToggleCollapse(r.key)}
                title="点击折叠/展开该组"
                tabIndex={-1}
              >
                <div style={{ width: W.no }} className="sched-gantt-cell sched-row-no" />
                {colOn('partition') && <div style={{ width: W.partition }} className="sched-gantt-cell" />}
                {colOn('marker') && <div style={{ width: W.marker }} className="sched-gantt-cell" />}
                <div style={{ width: W.name }} className="sched-gantt-cell">
                  <span style={{ paddingLeft: r.level * 14, display: 'inline-flex', alignItems: 'center', gap: 6, minWidth: 0 }}>
                    <strong className="sched-group-name">{open ? '▲' : '▶'}{r.label}</strong>
                    <span className="sched-dim" style={{ fontSize: 12 }}>{r.count} 项</span>
                  </span>
                </div>
                {colOn('wbs') && <div style={{ width: W.wbs }} className="sched-gantt-cell sched-dim" />}
                {colOn('dur') && <div style={{ width: W.dur }} className="sched-gantt-cell"><span className="sched-dim">汇总</span></div>}
                {colOn('preds') && <div style={{ width: W.preds }} className="sched-gantt-cell" />}
                {colOn('start') && <div style={{ width: W.start }} className="sched-gantt-cell sched-dim">{cpm.ok ? fmtDate(project.startDate, r.minEs, cal) : ''}</div>}
                {colOn('finish') && <div style={{ width: W.finish }} className="sched-gantt-cell sched-dim">{cpm.ok ? fmtDate(project.startDate, r.maxEf, cal) : ''}</div>}
                {colOn('progress') && <div style={{ width: W.progress }} className="sched-gantt-cell" />}
                {colOn('ls') && <div style={{ width: W.ls }} className="sched-gantt-cell" />}
                {colOn('lf') && <div style={{ width: W.lf }} className="sched-gantt-cell" />}
                {colOn('tf') && <div style={{ width: W.tf }} className="sched-gantt-cell" />}
                {colOn('ff') && <div style={{ width: W.ff }} className="sched-gantt-cell" />}
                {colOn('mode') && <div style={{ width: W.mode }} className="sched-gantt-cell" />}
                {colOn('succ') && <div style={{ width: W.succ }} className="sched-gantt-cell" />}
                {colOn('cost') && <div style={{ width: W.cost }} className="sched-gantt-cell sched-cost-cell" />}
                {/* 自定义字段列（v4.138 #14）：合成组头行留空（汇总只对叶任务有意义） */}
                {GANTT_CUSTOM_KEYS.filter((ck) => colOn(ck)).map((ck) => (
                  <div key={ck} style={{ width: W[ck] }} className="sched-gantt-cell" />
                ))}
                {(rangeCols ?? []).map((rk) => (
                  <div key={`range-${rk}`} style={{ width: W[rk] ?? 120 }} className="sched-gantt-cell" />
                ))}
              </div>
              )
            }
            const { i, t, row, group, span, manual, dur, colorCls, dimCls } = d
            const groupKey = `wbs:${i}`
            const groupOpen = !collapsedKeys.has(groupKey)
            /** 双击计划开始（date 弹层）：手动行改写 manualStart；自动行顺带锁定为手动
             *  （"改开始=锁定"是 ZPert 口径；自动排程下开始由 CPM 推算，不锁定则写入无意义） */
            const commitStart = (v: InplaceCommitValue) => {
              if (typeof v !== 'string' || v === '') return
              const wd = dateToWd(project.startDate, v, cal)
              if (wd === null || wd < 0) return
              onUpdateTask(t.id, manual ? { manualStart: wd } : { mode: 'manual', manualStart: wd })
            }
            /** 双击计划完成（date 弹层）：换算成工期写回（开始不动）——完成早于开始则忽略 */
            const commitFinish = (v: InplaceCommitValue) => {
              if (typeof v !== 'string' || v === '' || !row) return
              const wd = dateToWd(project.startDate, v, cal)
              if (wd === null) return
              const base = manual ? (t.manualStart ?? 0) : row.es
              const nd = Math.round(wd - base)
              if (nd <= 0) return
              onUpdateTask(t.id, { duration: nd })
            }
            /** 双击自定义字段（text/number 按注册表）：空值=删槽，不留空键 */
            const commitCustom = (ck: string, v: InplaceCommitValue) => {
              const next: Partial<Record<string, string | number>> = { ...(t.custom ?? {}) }
              if (v === null || v === '') delete next[ck]
              else if (typeof v === 'number') next[ck] = v
              else if (typeof v === 'string') next[ck] = v
              onUpdateTask(t.id, { custom: next })
            }
            return (
              <GanttRowShell
                key={t.id}
                t={t}
                group={group}
                colorCls={colorCls}
                dimCls={dimCls}
                selected={selectedId === t.id}
                onSelect={onSelect}
                rowMenu={rowMenu}
                onRowMenuClick={onRowMenuClick}
                tabIndex={rovingId === t.id ? 0 : -1}
                onKeyDown={(e) => onRowKeyDown(e, t)}
                draggable={dragEnabled ? true : undefined}
                onDragStart={dragEnabled ? (e) => {
                  setDragSrc(t.id)
                  if (e.dataTransfer) {
                    e.dataTransfer.effectAllowed = 'move'
                    e.dataTransfer.setData('text/plain', t.id)
                  }
                } : undefined}
                onDragOver={dragEnabled ? (e) => {
                  if (!dragSrc || dragSrc === t.id) return
                  e.preventDefault()
                  if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
                  const zone = zoneOf(e, t)
                  if (dropAt?.id !== t.id || dropAt.zone !== zone) setDropAt({ id: t.id, zone })
                } : undefined}
                onDragLeave={dragEnabled ? () => {
                  setDropAt((cur) => (cur?.id === t.id ? null : cur))
                } : undefined}
                onDrop={dragEnabled ? (e) => {
                  e.preventDefault()
                  const src = dragSrc ?? e.dataTransfer?.getData?.('text/plain') ?? ''
                  const zone = dropAt?.id === t.id ? dropAt.zone : zoneOf(e, t)
                  setDropAt(null)
                  setDragSrc(null)
                  if (src && src !== t.id) onMoveRow?.(src, t.id, zone)
                } : undefined}
                dropHint={dropAt?.id === t.id ? dropAt.zone : null}
              >
                <div style={{ width: W.no }} className="sched-gantt-cell sched-row-no">{i + 1}</div>
                {colOn('partition') && (
                  <div style={{ width: W.partition }} className="sched-gantt-cell sched-partition-cell" title={`分区 ${partitionOf(wbs[i])}`}>
                    {partitionOf(wbs[i])}
                  </div>
                )}
                {colOn('marker') && (
                  <div style={{ width: W.marker }} className="sched-gantt-cell sched-marker-cell">
                    {group ? <span className="sched-marker-star" title="分区汇总">★</span>
                      : t.isMilestone ? <span className="sched-marker-mile" title="里程碑">◆</span>
                        : null}
                  </div>
                )}
                <div style={{ width: W.name }} className="sched-gantt-cell">
                  <span style={{ paddingLeft: t.level * 14, display: 'inline-flex', alignItems: 'center', gap: 4, minWidth: 0 }}>
                    {group ? (
                      <strong
                        className="sched-group-name"
                        style={{ cursor: 'pointer' }}
                        onClick={(e) => { e.stopPropagation(); onToggleCollapse(groupKey) }}
                        title="点击折叠/展开"
                      >
                        {groupOpen ? '▲' : '▶'}{t.name}
                      </strong>
                    ) : (
                      <InplaceEditField
                        kind="text"
                        value={t.name}
                        spec={inplaceSpecOfCol('name')}
                        // 受控 open：双击与 F2 落在同一个状态源（不各造一套开合逻辑）
                        open={nameEditId === t.id}
                        onOpenChange={(next) =>
                          setNameEditId((current) => (next ? t.id : current === t.id ? null : current))
                        }
                        testId={`sched-inplace-name-${t.id}`}
                        onSubmit={(v: InplaceCommitValue) =>
                          onUpdateTask(t.id, { name: typeof v === 'string' ? v : '' })
                        }
                      >
                        <Input
                          size="small"
                          variant="borderless"
                          value={t.name}
                          style={{
                            color: t.style?.color,
                            fontFamily: t.style?.font,
                            fontSize: t.style?.size ? `${t.style.size}px` : undefined,
                            padding: 0,
                          }}
                          onChange={(e) => onUpdateTask(t.id, { name: e.target.value })}
                        />
                      </InplaceEditField>
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
                      <span style={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                        <InplaceEditField
                          kind="number"
                          value={String(dur)}
                          spec={inplaceSpecOfCol('dur')}
                          fullWidth
                          testId={`sched-inplace-dur-${t.id}`}
                          onSubmit={(v: InplaceCommitValue) => {
                            if (typeof v === 'number') onUpdateTask(t.id, { duration: v })
                          }}
                        >
                          <InputNumber
                            size="small"
                            variant="borderless"
                            min={0}
                            value={dur}
                            onChange={(v) => onUpdateTask(t.id, { duration: Number(v) || 0 })}
                            style={{ padding: 0, width: '100%', flex: 1, minWidth: 0 }}
                            title={isCd(t) ? '日历天（自然日定时）；双击改用日期弹层提交等效工作日；等效工作日跨度见条形悬停' : '双击开数值弹层（0-3650 工日）'}
                          />
                        </InplaceEditField>
                        {/* 单位 chip（刀3）：cd 行常显（点击切回工作日）；wd 行 hover 行时浮现。
                            切单位不换算数值（拍板项 7）；分组行/里程碑无入口。 */}
                        {!t.isMilestone && (
                          <Button
                            size="small"
                            type={isCd(t) ? 'primary' : 'text'}
                            ghost={isCd(t)}
                            className={`sched-unit-chip${isCd(t) ? ' sched-unit-cd' : ''}`}
                            data-testid={`sched-unit-${t.id}`}
                            onClick={() => onUpdateTask(t.id, isCd(t) ? { durationUnit: 'wd' } : { durationUnit: 'cd' })}
                            title={isCd(t) ? '日历天：按自然日定时（点击切回工作日；数值不变）' : '切为日历天：按自然日定时（数值不变）'}
                          >
                            日历
                          </Button>
                        )}
                      </span>
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
                {colOn('start') && (
                  <div style={{ width: W.start }} className="sched-gantt-cell sched-dim">
                    {group ? (
                      span ? fmtDate(project.startDate, span.es, cal) : ''
                    ) : row ? (
                      <InplaceEditField
                        kind="date"
                        value={isoAt(row.es)}
                        fullWidth
                        testId={`sched-inplace-start-${t.id}`}
                        onSubmit={commitStart}
                      >
                        {manual ? (
                          <InputNumber
                            size="small"
                            variant="borderless"
                            min={0}
                            value={row.es}
                            onChange={(v) => onUpdateTask(t.id, { manualStart: Number(v) || 0 })}
                            style={{ padding: 0, width: '100%' }}
                            title="手动模式：锁定开始（工作日序号）；双击改用日期弹层"
                          />
                        ) : (
                          <span className="sched-cell-text" title="双击开日期弹层（改为手动锁定开始；自动排程下改开始=锁定）">
                            {fmtDate(project.startDate, row.es, cal)}
                          </span>
                        )}
                      </InplaceEditField>
                    ) : (
                      ''
                    )}
                  </div>
                )}
                {colOn('finish') && (
                  <div style={{ width: W.finish }} className="sched-gantt-cell sched-dim">
                    {group ? (
                      span ? fmtDate(project.startDate, span.ef, cal) : ''
                    ) : row ? (
                      t.isMilestone || isCd(t) ? (
                        <span className="sched-cell-text">{fmtDate(project.startDate, row.ef, cal)}</span>
                      ) : (
                        <InplaceEditField
                          kind="date"
                          value={isoAt(row.ef)}
                          testId={`sched-inplace-finish-${t.id}`}
                          onSubmit={commitFinish}
                        >
                          <span className="sched-cell-text" title="双击开日期弹层（改完成=改工期，开始不动）">
                            {fmtDate(project.startDate, row.ef, cal)}
                          </span>
                        </InplaceEditField>
                      )
                    ) : (
                      ''
                    )}
                  </div>
                )}
                {colOn('progress') && (
                  <div style={{ width: W.progress }} className="sched-gantt-cell">
                    {!group && (
                      <InplaceEditField
                        kind="number"
                        value={String(t.progress)}
                        spec={inplaceSpecOfCol('progress')}
                        fullWidth
                        testId={`sched-inplace-progress-${t.id}`}
                        onSubmit={(v: InplaceCommitValue) => {
                          if (typeof v === 'number') onUpdateTask(t.id, { progress: v })
                        }}
                      >
                        <InputNumber
                          size="small"
                          variant="borderless"
                          min={0}
                          max={100}
                          value={t.progress}
                          onChange={(v) => onUpdateTask(t.id, { progress: Math.max(0, Math.min(100, Math.round(Number(v) || 0))) })}
                          style={{ padding: 0, width: '100%' }}
                          data-testid={`sched-progress-${t.id}`}
                          title="完成进度 0-100（条形进度覆盖与前锋线按此绘制）；双击开数值弹层"
                        />
                      </InplaceEditField>
                    )}
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
                        onClick={() => onUpdateTask(t.id, manual ? { mode: 'auto' } : { mode: 'manual', manualStart: row?.es ?? 0 })}
                        title={manual ? '手动模式（锁定开始，点击切回自动）' : '自动模式（CPM 排程，点击切手动）'}
                      >
                        {manual ? '手动' : '自动'}
                      </Button>
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
                {/* 自定义字段列（v4.138 #14）：叶行可编辑（text=Input/num=InputNumber，清空=删槽键）、
                    分组行留空；W5 双击开统一弹层（类型按注册表，提交=同样的删槽口径） */}
                {GANTT_CUSTOM_KEYS.filter((ck) => colOn(ck)).map((ck) => (
                  <div key={ck} style={{ width: W[ck] }} className="sched-gantt-cell">
                    {!group && (
                      <InplaceEditField
                        kind={inplaceKindOfCol(ck, CUSTOM_FIELD_BY_KEY[ck].type) ?? 'text'}
                        value={String(customValueOf(t, CUSTOM_FIELD_BY_KEY[ck].key) ?? '')}
                        spec={inplaceSpecOfCol(ck)}
                        fullWidth
                        testId={`sched-inplace-${ck}-${t.id}`}
                        onSubmit={(v: InplaceCommitValue) => commitCustom(ck, v)}
                      >
                        <CustomFieldCell task={t} field={CUSTOM_FIELD_BY_KEY[ck]} />
                      </InplaceEditField>
                    )}
                  </div>
                ))}
                {/* 日期区间列（W5 optional）：尾部追加；未接线 onRangeCommit 时只读文本 */}
                {(rangeCols ?? []).map((rk) => {
                  const rv = !group && rangeValueOf ? rangeValueOf(t, rk) : { from: '', to: '' }
                  const text = rv.from || rv.to ? `${rv.from || '…'} ~ ${rv.to || '…'}` : ''
                  return (
                    <div key={`range-${rk}`} style={{ width: W[rk] ?? 120 }} className="sched-gantt-cell">
                      {!group && (
                        <InplaceEditField
                          kind="dateRange"
                          value={rv}
                          fullWidth
                          testId={`sched-inplace-${rk}-${t.id}`}
                          onSubmit={(v: InplaceCommitValue) => {
                            if (onRangeCommit && v !== null && typeof v === 'object') onRangeCommit(t.id, rk, v)
                          }}
                        >
                          <span className="sched-cell-text">{text}</span>
                        </InplaceEditField>
                      )}
                    </div>
                  )
                })}
              </GanttRowShell>
            )
            }
          )
          })()}
        </div>
      </div>
    </>
  )
}
