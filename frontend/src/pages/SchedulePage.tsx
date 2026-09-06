/**
 * SchedulePage — 「进度计划」一级板块（v4.110.0 刀1）
 *
 * 工程进度计划编制工作台：一套任务表数据驱动三种视图自由切换——
 * 横道图（甘特）/ 单代号网络图（PDM 六格）/ 双代号网络图（AOA 虚工作自动生成）。
 * CPM 引擎纯前端计算（FS/SS/FF/SF + 时距、正逆推、总/自由时差、关键线路），
 * 数据 localStorage 持久化（gaea.schedule.v1）。
 */
import React, { useMemo } from 'react'
import { Alert, Button, Input, Popconfirm, Segmented, Space, Tooltip } from 'antd'
import {
  AimOutlined, ClearOutlined, ClusterOutlined, NodeIndexOutlined, PlusOutlined,
  TableOutlined, ThunderboltOutlined, PartitionOutlined, DeleteOutlined,
} from '@ant-design/icons'
import { computeCpm } from '../schedule/cpm'
import { buildAoa } from '../schedule/aoa'
import { useScheduleStore, isGroupRow } from '../schedule/store'
import { GanttView } from '../schedule/GanttView'
import { PdmView } from '../schedule/PdmView'
import { AoaView } from '../schedule/AoaView'
import '../schedule/schedule.css'
import '../gaea/styles.css'
import '../gaea/tailwind.css'

const SchedulePage: React.FC = () => {
  const project = useScheduleStore((s) => s.project)
  const view = useScheduleStore((s) => s.view)
  const setView = useScheduleStore((s) => s.setView)
  const selectedId = useScheduleStore((s) => s.selectedId)
  const renameProject = useScheduleStore((s) => s.renameProject)
  const setStartDate = useScheduleStore((s) => s.setStartDate)
  const addTask = useScheduleStore((s) => s.addTask)
  const addGroup = useScheduleStore((s) => s.addGroup)
  const updateTask = useScheduleStore((s) => s.updateTask)
  const removeTask = useScheduleStore((s) => s.removeTask)
  const loadSample = useScheduleStore((s) => s.loadSample)
  const clearAll = useScheduleStore((s) => s.clearAll)

  const cpm = useMemo(() => computeCpm(project.tasks, project.links), [project])
  const aoa = useMemo(() => buildAoa(project.tasks, project.links), [project])

  const selected = project.tasks.find((t) => t.id === selectedId) ?? null
  const selectedIsGroup = selected ? isGroupRow(project.tasks, project.tasks.findIndex((t) => t.id === selected!.id)) : false
  const canDelete = !!selected

  return (
    <div className="sched-page" style={{ padding: '12px 16px' }}>
      <div className="sched-header">
        <h2 className="sched-header-title">进度计划</h2>
        <Input
          className="sched-name-input"
          variant="borderless"
          value={project.name}
          onChange={(e) => renameProject(e.target.value)}
          style={{ maxWidth: 260, fontSize: 15 }}
        />
        <Space size={4}>
          <span className="sched-dim">开工日期</span>
          <Input
            type="date"
            size="small"
            value={project.startDate}
            onChange={(e) => e.target.value && setStartDate(e.target.value)}
            style={{ width: 140 }}
          />
        </Space>
        <div style={{ flex: 1 }} />
        <Segmented
          value={view}
          onChange={(v) => setView(v as typeof view)}
          options={[
            { value: 'gantt', label: <span><TableOutlined /> 横道图</span> },
            { value: 'pdm', label: <span><NodeIndexOutlined /> 单代号网络图</span> },
            { value: 'aoa', label: <span><PartitionOutlined /> 双代号网络图</span> },
          ]}
        />
      </div>

      <div className="sched-toolbar">
        <Button size="small" icon={<PlusOutlined />} onClick={() => addTask(selectedId ?? undefined)}>添加任务</Button>
        <Button size="small" icon={<ClusterOutlined />} onClick={addGroup}>添加分组</Button>
        {selected && !selectedIsGroup && (
          <Button
            size="small"
            icon={<AimOutlined />}
            type={selected.isMilestone ? 'primary' : 'default'}
            onClick={() => updateTask(selected.id, { isMilestone: !selected.isMilestone })}
          >
            {selected.isMilestone ? '取消里程碑' : '设为里程碑'}
          </Button>
        )}
        {canDelete && (
          <Popconfirm
            title="删除选中项"
            description={selectedIsGroup ? '分组及其子任务一并删除' : `删除「${selected!.name}」及其搭接关系`}
            onConfirm={() => removeTask(selected!.id)}
          >
            <Button size="small" danger icon={<DeleteOutlined />}>删除</Button>
          </Popconfirm>
        )}
        <div style={{ flex: 1 }} />
        <Tooltip title="载入示例工程（办公楼施工），覆盖当前数据">
          <Button size="small" icon={<ThunderboltOutlined />} onClick={loadSample}>示例工程</Button>
        </Tooltip>
        <Popconfirm title="清空全部任务与搭接？" onConfirm={clearAll}>
          <Button size="small" icon={<ClearOutlined />}>清空</Button>
        </Popconfirm>
      </div>

      {!cpm.ok && cpm.error && (
        <Alert type="error" showIcon message={cpm.error} description="请修正搭接关系后重试；网络图视图在循环解除前不可用。" />
      )}

      {view === 'gantt' && <GanttView project={project} cpm={cpm} />}
      {view === 'pdm' && <PdmView project={project} cpm={cpm} />}
      {view === 'aoa' && <AoaView graph={aoa} tasks={project.tasks} />}
    </div>
  )
}

export default SchedulePage
