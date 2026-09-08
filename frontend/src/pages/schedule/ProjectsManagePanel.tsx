/**
 * 工程管理面板（v4.139 #15 刀2 §3.4）：列表（名称/工期/任务数）+ 归档/反归档 +
 * 删除（Popconfirm 二次确认，文案注明不可恢复）+ 新建工程。v1 简化：仅当前
 * 工程可行内改名（走既有 renameProject=改文件内 project.name 而非文件名——
 * 文件名机械化，agent 显式 path 引用稳定性优先），其余工程显示只读名+载入。
 * 瘦身 P3：自 pages/SchedulePage.tsx 分片，正文零改动。
 */
import React, { useState } from 'react'
import { Button, Input, Modal, Popconfirm, Space } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { useScheduleStore } from '../../schedule/store'
import type { ScheduleProjectSummary } from '../../schedule/api'

export const ProjectsManagePanel: React.FC = () => {
  const projects = useScheduleStore((s) => s.projects)
  const currentPath = useScheduleStore((s) => s.currentPath)
  const projectName = useScheduleStore((s) => s.project.name)
  const renameProject = useScheduleStore((s) => s.renameProject)
  const openProject = useScheduleStore((s) => s.openProject)
  const createProject = useScheduleStore((s) => s.createProject)
  const archiveProject = useScheduleStore((s) => s.archiveProject)
  const deleteProject = useScheduleStore((s) => s.deleteProject)
  const copyProject = useScheduleStore((s) => s.copyProject)
  const [draft, setDraft] = useState('')
  // v4.145 工程复制（另存为）：行内「复制」→ 弹窗取名（默认「-副本」）→ 落独立文件
  const [copyTarget, setCopyTarget] = useState<ScheduleProjectSummary | null>(null)
  const [copyName, setCopyName] = useState('')

  /** 管理动作统一入口：store 动作未就绪（旧形态）时无害跳过 */
  const submit = (fn: () => unknown) => { void Promise.resolve(fn()) }

  const doCreate = () => {
    const name = draft.trim()
    if (!name) return
    submit(() => createProject(name))
    setDraft('')
  }

  return (
    <div data-testid="sched-projects-manage-panel" style={{ display: 'grid', gap: 6, minWidth: 420 }}>
      {(projects ?? []).map((p) => (
        <div key={p.rel} style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          {p.rel === currentPath ? (
            <Input
              size="small"
              variant="borderless"
              value={projectName}
              onChange={(e) => renameProject(e.target.value)}
              style={{ flex: 1, minWidth: 0, fontSize: 13 }}
              title="行内改名=改文件内工程名（文件名不变）"
            />
          ) : (
            <span style={{ flex: 1, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', fontSize: 13 }} title={p.rel}>
              {p.name || p.rel}{p.archived ? '（已归档）' : ''}
            </span>
          )}
          <span className="sched-dim" style={{ fontSize: 12, whiteSpace: 'nowrap' }}>{p.duration} 天 / {p.taskCount} 项</span>
          {p.rel !== currentPath && (
            <Button size="small" onClick={() => submit(() => openProject(p.rel))} title="切换为当前工程（板块重水合）">载入</Button>
          )}
          <Button
            size="small"
            data-testid="sched-project-copy"
            onClick={() => { setCopyTarget(p); setCopyName(`${p.name}-副本`) }}
            title="复制为独立工程文件（含基线/布局/日历），当前打开的工程不变"
          >复制</Button>
          <Button
            size="small"
            data-testid="sched-project-archive"
            onClick={() => submit(() => archiveProject(p.rel, !p.archived))}
            title={p.archived ? '取消归档（切换器恢复显示）' : '归档（文件保留原位，切换器不再显示）'}
          >
            {p.archived ? '反归档' : '归档'}
          </Button>
          <Popconfirm title="删除工程" description="删除后不可恢复" onConfirm={() => submit(() => deleteProject(p.rel))}>
            <Button size="small" danger data-testid="sched-project-delete" title="物理删除计划文件并从注册表摘除">删除</Button>
          </Popconfirm>
        </div>
      ))}
      <Space size={4} style={{ marginTop: 2 }}>
        <Input
          size="small"
          placeholder="新工程名称"
          value={draft}
          data-testid="sched-project-create-input"
          style={{ width: 220 }}
          onChange={(e) => setDraft(e.target.value)}
          onPressEnter={doCreate}
        />
        <Button size="small" type="dashed" icon={<PlusOutlined />} data-testid="sched-project-create" disabled={!draft.trim()} onClick={doCreate}>
          新建工程
        </Button>
      </Space>
      <span className="sched-dim" style={{ fontSize: 12 }}>删除当前工程时自动切到剩余第一个工程；归档不移动文件；复制不切换指针。</span>
      <Modal
        title={`复制工程「${copyTarget?.name ?? ''}」`}
        open={!!copyTarget}
        okText="复制"
        okButtonProps={{ disabled: !copyName.trim(), 'data-testid': 'sched-project-copy-ok' }}
        onOk={async () => { if (copyTarget && (await copyProject(copyTarget.rel, copyName))) setCopyTarget(null) }}
        onCancel={() => setCopyTarget(null)}
        destroyOnHidden
      >
        <Input
          size="small"
          value={copyName}
          data-testid="sched-project-copy-input"
          placeholder="新工程名称"
          onChange={(e) => setCopyName(e.target.value)}
          onPressEnter={async () => { if (copyTarget && copyName.trim() && (await copyProject(copyTarget.rel, copyName))) setCopyTarget(null) }}
        />
        <div className="sched-dim" style={{ marginTop: 8, fontSize: 12 }}>
          复制为独立工程文件入注册表（含基线/AOA 布点/日历），副本打开后改动互不影响——签证「调整1→调整2」迭代可先复制再改，基线漂移直接对比。
        </div>
      </Modal>
    </div>
  )
}