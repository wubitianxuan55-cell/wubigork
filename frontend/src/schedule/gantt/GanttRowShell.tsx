/**
 * schedule/gantt/GanttRowShell.tsx — 横道行统一外壳（FE7-05 单源）
 *
 * 表格窗格与画布窗格的每一任务行共用同一外壳：右键 Dropdown（行菜单）+ 行 div
 * （类名拼接/行高/点击选中）——此前两窗格逐行镜像（审计 FE7-05）。两窗格只渲染
 * 各自专有内容进 children；类名/高度/菜单接线与拆分前逐字段一致，行为零变化。
 *
 * W6 键盘可达（行为线）：外壳多接两个 **可选** 属性 `tabIndex` / `onKeyDown`——
 * 只有表格窗格（GanttTable）传值做 roving tabindex + ↑/↓/Enter/F2；画布窗格不传，
 * 行仍是不可聚焦的纯展示节点（拖拽/双击手势与拆分前逐字一致）。
 */
import React from 'react'
import { Dropdown, type MenuProps } from 'antd'
import type { SchedTask } from '../types'
import { ROW_H } from './ganttUtil'

interface GanttRowShellProps {
  t: SchedTask
  /** WBS 分组行（组行加 sched-group-row 类、菜单不含叶行专有项） */
  group: boolean
  colorCls: string
  dimCls: string
  selected: boolean
  onSelect: (id: string) => void
  rowMenu: (t: SchedTask, group: boolean) => MenuProps['items']
  onRowMenuClick: (key: string, t: SchedTask) => void
  /** roving tabindex（可选；不传=不可聚焦，保持画布窗格旧行为） */
  tabIndex?: number
  /** 行级键盘（可选；由 GanttTable 接线） */
  onKeyDown?: React.KeyboardEventHandler<HTMLDivElement>
  /** 树形拖动（可选；只有表格窗格传，画布不传保持旧行为） */
  draggable?: boolean
  onDragStart?: React.DragEventHandler<HTMLDivElement>
  onDragOver?: React.DragEventHandler<HTMLDivElement>
  onDragLeave?: React.DragEventHandler<HTMLDivElement>
  onDrop?: React.DragEventHandler<HTMLDivElement>
  /** 拖放提示类（before/after/inside 三态，样式在 sched-zebra.css） */
  dropHint?: 'before' | 'after' | 'inside' | null
  children: React.ReactNode
}

/** 任务/WBS 行统一外壳：Dropdown（右键行菜单）+ 行 div（选中/配色/淡化/高度/点击）。
 *  行 key（=任务 id）由调用方以 JSX key 传入（与拆分前 Dropdown key 同位）。 */
export const GanttRowShell: React.FC<GanttRowShellProps> = ({
  t,
  group,
  colorCls,
  dimCls,
  selected,
  onSelect,
  rowMenu,
  onRowMenuClick,
  tabIndex,
  onKeyDown,
  draggable,
  onDragStart,
  onDragOver,
  onDragLeave,
  onDrop,
  dropHint,
  children,
}) => (
  <Dropdown trigger={['contextMenu']} menu={{ items: rowMenu(t, group), onClick: (e) => onRowMenuClick(e.key, t) }}>
    <div
      className={`sched-gantt-row${group ? ' sched-group-row' : ''}${colorCls}${dimCls}${selected ? ' sched-row-selected' : ''}${dropHint ? ` sched-drop-${dropHint}` : ''}`}
      style={{ height: ROW_H }}
      onClick={() => onSelect(t.id)}
      tabIndex={tabIndex}
      onKeyDown={onKeyDown}
      draggable={draggable}
      onDragStart={onDragStart}
      onDragOver={onDragOver}
      onDragLeave={onDragLeave}
      onDrop={onDrop}
    >
      {children}
    </div>
  </Dropdown>
)
