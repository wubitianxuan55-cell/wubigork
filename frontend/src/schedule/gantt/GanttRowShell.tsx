/**
 * schedule/gantt/GanttRowShell.tsx — 横道行统一外壳（FE7-05 单源）
 *
 * 表格窗格与画布窗格的每一任务行共用同一外壳：右键 Dropdown（行菜单）+ 行 div
 * （类名拼接/行高/点击选中）——此前两窗格逐行镜像（审计 FE7-05）。两窗格只渲染
 * 各自专有内容进 children；类名/高度/菜单接线与拆分前逐字段一致，行为零变化。
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
  children,
}) => (
  <Dropdown trigger={['contextMenu']} menu={{ items: rowMenu(t, group), onClick: (e) => onRowMenuClick(e.key, t) }}>
    <div
      className={`sched-gantt-row${group ? ' sched-group-row' : ''}${colorCls}${dimCls}${selected ? ' sched-row-selected' : ''}`}
      style={{ height: ROW_H }}
      onClick={() => onSelect(t.id)}
    >
      {children}
    </div>
  </Dropdown>
)
