/**
 * schedule/gantt/GanttToolbar.tsx — 横道图工具栏（自 GanttView.tsx 拆分，行为零变化）
 *
 * 前锋线/基线开关、检查日期、行筛选（全部/关键/手动 + 文本）、排序、多级分组、
 * 路径分析、列显隐菜单（自定义列改名行内）与缩放/全览按钮。
 * 排序/分组切换的 chatPrefs 持久化随逻辑一并迁入（同文件内 setState+save 顺序不变）。
 */
import React from 'react'
import { Button, Checkbox, DatePicker, Input, Popover, Segmented, Select } from 'antd'
import { ColumnWidthOutlined, EditOutlined, FullscreenOutlined, SearchOutlined, ZoomInOutlined, ZoomOutOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import type { SchedProject } from '../types'
import { customLabelOf, type CustomFieldKey } from '../customFields'
import { GANTT_COLS, GANTT_CUSTOM_KEYS, GANTT_FIXED_KEYS } from '../ganttCols'
import type { GanttFilter, GanttFilterKind } from '../ganttFilter'
import type { GanttGroupField, GanttSort, GanttSortField } from '../ganttGroup'
import { saveChatPrefs } from '../chatPrefs'

interface GanttToolbarProps {
  project: SchedProject
  showFront: boolean
  onToggleFront: () => void
  showBase: boolean
  onToggleBase: () => void
  checkDate: string
  setCheckDate: (iso: string) => void
  filter: GanttFilter
  setFilter: React.Dispatch<React.SetStateAction<GanttFilter>>
  sort: GanttSort
  setSort: React.Dispatch<React.SetStateAction<GanttSort>>
  groupBy: GanttGroupField[]
  setGroupBy: React.Dispatch<React.SetStateAction<GanttGroupField[]>>
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
  showFront,
  onToggleFront,
  showBase,
  onToggleBase,
  checkDate,
  setCheckDate,
  filter,
  setFilter,
  sort,
  setSort,
  groupBy,
  setGroupBy,
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
        onChange={(v) => setFilter((f) => ({ ...f, kind: v as GanttFilterKind }))}
        options={[
          { value: 'all', label: '全部' },
          { value: 'critical', label: '关键' },
          { value: 'manual', label: '手动' },
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
          setSort(next)
          saveChatPrefs({ ganttSort: next })
        }}
        data-testid="sched-gantt-sort"
      />
      {sort.field !== 'none' && (
        <Button
          size="small"
          data-testid="sched-gantt-sortdir"
          onClick={() => {
            const next: GanttSort = { field: sort.field, dir: sort.dir === 'asc' ? 'desc' : 'asc' }
            setSort(next)
            saveChatPrefs({ ganttSort: next })
          }}
          title="切换升/降序"
        >
          {sort.dir === 'asc' ? '升序' : '降序'}
        </Button>
      )}
      {/* 多级分组（v4.136 小刀）：至多两级合成组头行；启用后 WBS 分组行由分组接管 */}
      <Select
        size="small"
        mode="multiple"
        placeholder="分组"
        value={groupBy}
        style={{ minWidth: 100, maxWidth: 170 }}
        options={[
          { value: 'critical', label: '关键' },
          { value: 'mode', label: '模式' },
          { value: 'milestone', label: '里程碑' },
        ]}
        onChange={(v) => {
          const next = v.slice(-2) as GanttGroupField[]
          setGroupBy(next)
          setCollapsedKeys(new Set())
          saveChatPrefs({ ganttGroup: next })
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