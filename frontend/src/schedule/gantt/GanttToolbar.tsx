/**
 * schedule/gantt/GanttToolbar.tsx — 横道图工具栏（自 GanttView.tsx 拆分，行为零变化）
 *
 * W1 视图预设、前锋线/基线、四档筛选、排序、14 维多级分组、路径分析、
 * 列显隐菜单与缩放/全览。视图/分组/排序的持久化由 GanttView 单点提交，
 * 本组件保持受控展示，防止同一命令两处落盘。
 */
import React from 'react'
import { Button, Checkbox, DatePicker, Input, Popover, Segmented, Select } from 'antd'
import { ColumnWidthOutlined, EditOutlined, FullscreenOutlined, SearchOutlined, ZoomInOutlined, ZoomOutOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import type { SchedProject } from '../types'
import { customLabelOf, type CustomFieldKey } from '../customFields'
import { GANTT_COLS, GANTT_CUSTOM_KEYS, GANTT_FIXED_KEYS } from '../ganttCols'
import type { GanttFilter, GanttFilterKind } from '../ganttFilter'
import {
  GANTT_GROUP_OPTIONS,
  MAX_GANTT_GROUP_DEPTH,
  type GanttGroupField,
  type GanttSort,
  type GanttSortField,
} from '../ganttGroup'
import { BUILTIN_GANTT_VIEWS, CUSTOM_GANTT_VIEW_ID } from '../viewPresets'

interface GanttToolbarProps {
  project: SchedProject
  viewId: string
  onViewChange: (id: string) => void
  showFront: boolean
  onToggleFront: () => void
  showBase: boolean
  onToggleBase: () => void
  checkDate: string
  setCheckDate: (iso: string) => void
  filter: GanttFilter
  setFilter: React.Dispatch<React.SetStateAction<GanttFilter>>
  onFilterKindChange: (kind: GanttFilterKind) => void
  sort: GanttSort
  onSortChange: (sort: GanttSort) => void
  groupBy: GanttGroupField[]
  onGroupChange: (groupBy: GanttGroupField[]) => void
  setCollapsedKeys: React.Dispatch<React.SetStateAction<Set<string>>>
  pathMode: 'off' | 'pred' | 'succ'
  setPathMode: React.Dispatch<React.SetStateAction<'off' | 'pred' | 'succ'>>
  colOn: (key: string) => boolean
  onToggleCol: (key: string, on: boolean) => void
  renameKey: string | null
  renameVal: string
  setRenameKey: (k: string | null) => void
  setRenameVal: (v: string) => void
  onCommitRename: () => void
  renameCancel: React.MutableRefObject<boolean>
  onZoom: (dir: 1 | -1) => void
  onFit: () => void
}

export const GanttToolbar: React.FC<GanttToolbarProps> = ({
  project,
  viewId,
  onViewChange,
  showFront,
  onToggleFront,
  showBase,
  onToggleBase,
  checkDate,
  setCheckDate,
  filter,
  setFilter,
  onFilterKindChange,
  sort,
  onSortChange,
  groupBy,
  onGroupChange,
  setCollapsedKeys,
  pathMode,
  setPathMode,
  colOn,
  onToggleCol,
  renameKey,
  renameVal,
  setRenameKey,
  setRenameVal,
  onCommitRename,
  renameCancel,
  onZoom,
  onFit,
}) => {
  return (
    <div className="sched-gantt-toolbar">
      <Select
        size="small"
        value={viewId}
        style={{ width: 118 }}
        options={[
          ...BUILTIN_GANTT_VIEWS.map((view) => ({ value: view.id, label: view.name })),
          ...(project.views ?? []).map((v) => ({ value: v.id, label: v.name })),
          { value: CUSTOM_GANTT_VIEW_ID, label: '自定义', disabled: true },
        ]}
        onChange={onViewChange}
        title="切换工作视图：同时应用列集、分组、排序与筛选（斑马 ViewDB 口径）"
        aria-label="切换进度视图"
        data-testid="sched-gantt-view"
      />
      <Button
        size="small"
        type={showFront ? 'primary' : 'default'}
        onClick={onToggleFront}
        title="按任务完成进度画前锋线（斑马式进度检查）"
      >
        前锋线
      </Button>
      {project.baseline && (
        <Button
          size="small"
          type={showBase ? 'primary' : 'default'}
          onClick={onToggleBase}
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
      {/* 行筛选（刀C 余项）：关键/手动/文本；仅横道视图参与（网络图=逻辑图） */}
      <Segmented
        size="small"
        value={filter.kind}
        onChange={(v) => onFilterKindChange(v as GanttFilterKind)}
        options={[
          { value: 'all', label: '全部' },
          { value: 'critical', label: '关键' },
          { value: 'manual', label: '手动' },
          { value: 'milestone', label: '里程碑' },
        ]}
        data-testid="sched-gantt-filter"
      />
      <Input
        size="small"
        allowClear
        prefix={<SearchOutlined />}
        placeholder="搜任务名"
        value={filter.text}
        onChange={(e) => setFilter((f) => ({ ...f, text: e.target.value }))}
        style={{ width: 150 }}
        data-testid="sched-gantt-search"
      />
      {/* 排序（v4.136 小刀）：叶行重排、组头不动；持久化 chatPrefs */}
      <Select
        size="small"
        value={sort.field}
        style={{ width: 92 }}
        options={[
          { value: 'none', label: '不排序' },
          { value: 'name', label: '按名称' },
          { value: 'duration', label: '按工期' },
          { value: 'start', label: '按开始' },
          { value: 'progress', label: '按进度' },
          { value: 'tf', label: '按时差' },
        ]}
        onChange={(v) => {
          const next: GanttSort = { field: v as GanttSortField, dir: sort.dir }
          onSortChange(next)
        }}
        data-testid="sched-gantt-sort"
      />
      {sort.field !== 'none' && (
        <Button
          size="small"
          data-testid="sched-gantt-sortdir"
          onClick={() => {
            const next: GanttSort = { field: sort.field, dir: sort.dir === 'asc' ? 'desc' : 'asc' }
            onSortChange(next)
          }}
          title="切换升/降序"
        >
          {sort.dir === 'asc' ? '升序' : '降序'}
        </Button>
      )}
      {/* 多级分组（W1）：斑马 14 维，至多三级；启用后 WBS 分组行由分组接管。 */}
      <Select
        size="small"
        mode="multiple"
        placeholder="分组"
        value={groupBy}
        style={{ minWidth: 100, maxWidth: 190 }}
        options={[...GANTT_GROUP_OPTIONS]}
        onChange={(v) => {
          const next = v.slice(-MAX_GANTT_GROUP_DEPTH) as GanttGroupField[]
          onGroupChange(next)
          setCollapsedKeys(new Set())
        }}
        data-testid="sched-gantt-group"
      />
      {/* 路径分析（v4.136 小刀）：锚点=选中行，沿驱动边（依赖 ff=0）高亮上下游链 */}
      <Segmented
        size="small"
        value={pathMode}
        onChange={(v) => setPathMode(v as 'off' | 'pred' | 'succ')}
        options={[
          { value: 'off', label: '路径' },
          { value: 'pred', label: '前驱链' },
          { value: 'succ', label: '后继链' },
        ]}
        data-testid="sched-gantt-path"
      />
      <div style={{ flex: 1 }} />
      {/* 列显隐（刀C 双栏）：行号/名称固定，其余单列可藏让位画布；
          自定义列（v4.138 #14）菜单行附「改名」铅笔（行内 Input，回车/失焦提交、Esc 取消） */}
      <Popover
        trigger="click"
        placement="bottomRight"
        title="显示列（行号与任务名称固定）"
        content={
          <div className="sched-colmenu" data-testid="sched-gantt-colmenu">
            {GANTT_COLS.filter((c) => !GANTT_FIXED_KEYS.includes(c.key)).map((c) => {
              const isCustom = GANTT_CUSTOM_KEYS.includes(c.key)
              const label = isCustom ? customLabelOf(c.key as CustomFieldKey, project.customLabels) : c.label
              return (
                <label key={c.key} className="sched-colmenu-item">
                  <Checkbox checked={colOn(c.key)} onChange={(e) => onToggleCol(c.key, e.target.checked)} />
                  {renameKey === c.key ? (
                    <Input
                      size="small"
                      autoFocus
                      value={renameVal}
                      onChange={(e) => setRenameVal(e.target.value)}
                      onBlur={onCommitRename}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') e.currentTarget.blur()
                        else if (e.key === 'Escape') {
                          renameCancel.current = true
                          e.currentTarget.blur()
                        }
                      }}
                      style={{ width: 92 }}
                      data-testid={`sched-colmenu-rename-${c.key}`}
                    />
                  ) : (
                    <span>{label}</span>
                  )}
                  {isCustom && renameKey !== c.key && (
                    <Button
                      size="small"
                      type="text"
                      icon={<EditOutlined />}
                      title={`重命名「${label}」`}
                      onClick={(e) => {
                        e.preventDefault() // 在 label 内：阻止激活勾选框
                        e.stopPropagation()
                        setRenameKey(c.key)
                        setRenameVal(label)
                      }}
                    />
                  )}
                </label>
              )
            })}
          </div>
        }
      >
        <Button size="small" icon={<ColumnWidthOutlined />} data-testid="sched-gantt-cols-btn" title="显示/隐藏表格列（让位时间画布）">
          列
        </Button>
      </Popover>
      {/* 缩放/全览（v4.141：连续缩放 ×1.35；全览=整计划适配画布可视宽）；Ctrl+滚轮同效（v4.159） */}
      <Button size="small" icon={<ZoomOutOutlined />} data-testid="sched-gantt-zoomout" title="缩小（日宽 ÷1.35；画布上 Ctrl+滚轮同效）" onClick={() => onZoom(-1)} />
      <Button size="small" icon={<ZoomInOutlined />} data-testid="sched-gantt-zoomin" title="放大（日宽 ×1.35；画布上 Ctrl+滚轮同效）" onClick={() => onZoom(1)} />
      <Button size="small" icon={<FullscreenOutlined />} data-testid="sched-gantt-fit" title="窗口内全览：整计划适配画布宽度，不再无限横向拉长" onClick={onFit}>
        全览
      </Button>
    </div>
  )
}
