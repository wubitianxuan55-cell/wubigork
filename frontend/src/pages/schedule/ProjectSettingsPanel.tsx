/**
 * ProjectSettingsPanel — 工程属性弹出面板。
 *
 * ZPert 主工作区不常驻工程属性横条；gaea 将必要项收进“计划 → 工程设置”，
 * 保留工程名、开工/目标竣工、日历、检视、基线与资源入口。
 */
import React from 'react'
import { Button, Input, Popover, Space } from 'antd'
import { CalendarOutlined, FieldTimeOutlined, FundOutlined, TeamOutlined } from '@ant-design/icons'
import type { CpmResult } from '../../schedule/types'
import { useScheduleStore } from '../../schedule/store'
import { hasCostData } from '../../schedule/costUi'
import { ReviewPanel } from '../../schedule/ReviewPanel'
import { BaselinesPanel } from '../../schedule/BaselinesPanel'
import { ResourcePanel } from '../../schedule/ResourcePanel'
import { CalendarEditor } from './CalendarEditor'

export const ProjectSettingsPanel: React.FC<{ cpm: CpmResult }> = ({ cpm }) => {
  const project = useScheduleStore((s) => s.project)
  const renameProject = useScheduleStore((s) => s.renameProject)
  const setStartDate = useScheduleStore((s) => s.setStartDate)
  const setDeadline = useScheduleStore((s) => s.setDeadline)
  const hasResources = (project.resources?.length ?? 0) > 0

  return (
    <div className="sched-project-settings" data-testid="sched-project-settings">
      <label>
        <span>工程名称</span>
        <Input
          value={project.name}
          onChange={(event) => renameProject(event.target.value)}
          placeholder="工程名称"
          aria-label="工程名称"
        />
      </label>
      <Space size={8} wrap>
        <label>
          <span>开工日期</span>
          <Input
            type="date"
            value={project.startDate}
            onChange={(event) => event.target.value && setStartDate(event.target.value)}
          />
        </label>
        <label>
          <span>目标竣工</span>
          <Input
            type="date"
            value={project.deadline ?? ''}
            onChange={(event) => setDeadline(event.target.value || null)}
          />
        </label>
      </Space>
      <div className="sched-project-settings-actions">
        <Popover trigger="click" placement="left" content={<CalendarEditor />} title="工作日历">
          <Button size="small" icon={<CalendarOutlined />}>日历</Button>
        </Popover>
        <Popover trigger="click" placement="left" content={<ReviewPanel cpm={cpm} />} title="检视（多期进度快照）">
          <Button size="small" icon={<FieldTimeOutlined />}>检视</Button>
        </Popover>
        <Popover trigger="click" placement="left" content={<BaselinesPanel cpm={cpm} />} title="基线对比">
          <Button size="small" icon={<FundOutlined />}>基线</Button>
        </Popover>
        <Popover trigger="click" placement="left" content={<ResourcePanel cpm={cpm} />} title="资源与成本">
          <Button size="small" icon={<TeamOutlined />} type={hasResources ? 'primary' : 'default'} ghost={hasResources}>
            资源
          </Button>
        </Popover>
      </div>
      {hasCostData(project) && <div className="sched-project-settings-note">资源与成本已有数据</div>}
    </div>
  )
}
