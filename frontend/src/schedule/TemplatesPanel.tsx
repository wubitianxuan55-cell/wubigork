/**
 * schedule/TemplatesPanel.tsx — 项目模板面板（v4.138 #10）
 *
 * 将被主线挂进 SchedulePage 工具栏 Popover（与 BaselinesPanel 同一挂载形态，
 * 参考其自取 store 范式——工程数据与动作一律取自 useScheduleStore，不依赖
 * 页面局部状态，可整体替换进模板 Popover）：
 *  - 上半「另存为模板」：名输入缺省=当前工程名（空名回落工程名），保存走
 *    saveTemplate（同名覆盖 / 上限 10 FIFO 淘汰最旧，存满给提示）；
 *  - 下半模板列表：每行 名称 + 保存时间 + 「载入」（Popconfirm「载入将覆盖
 *    当前计划（可撤销），确定？」，确认后 importProject——store 内
 *    pushHistory + normalizeProject，天然可撤销）+ 删除（Popconfirm 确认）；
 *    空态给起步引导文案。
 * props 只收 cpm（与 BaselinesPanel 挂载签名一致，供工具栏统一挂载；模板
 * 逻辑本身用不到 CPM 结果）。列表为 storage 的内存镜像，写动作后以纯函数
 * 返回值刷新。
 */
import React, { useState } from 'react'
import { Button, Input, Popconfirm, Space } from 'antd'
import { useScheduleStore } from './store'
import { TEMPLATES_CAP, deleteTemplate, listTemplates, saveTemplate } from './templates'
import type { TemplateItem } from './templates'
import type { CpmResult } from './types'

/** 项目模板面板（数据自取 store；挂载签名与 BaselinesPanel 对齐） */
export const TemplatesPanel: React.FC<{ cpm: CpmResult }> = () => {
  const project = useScheduleStore((s) => s.project)
  const importProject = useScheduleStore((s) => s.importProject)

  /** 模板列表内存镜像（首次渲染读 storage；写动作后用返回值刷新） */
  const [templates, setTemplates] = useState<TemplateItem[]>(() => listTemplates())
  /** 另存表单名草稿：缺省=当前工程名 */
  const [nameDraft, setNameDraft] = useState(() => project.name)

  /** 另存为模板：空草稿回落工程名（仍空则 templates.ts 内 no-op） */
  const handleSave = () => {
    setTemplates(saveTemplate(nameDraft.trim() || project.name, project))
  }

  return (
    <div data-testid="sched-templates-panel" style={{ display: 'grid', gap: 8, minWidth: 300, maxWidth: 420 }}>
      <span className="sched-dim">
        模板保存整份计划（任务、搭接、日历、资源），新工程可从同类工程模板一键起步。
      </span>

      {/* 上半：另存为模板（名缺省=工程名） */}
      <Space size={6} wrap>
        <Input
          size="small" style={{ width: 170 }} value={nameDraft}
          data-testid="sched-tpl-name-input" placeholder="模板名"
          onChange={(e) => setNameDraft(e.target.value)}
          onPressEnter={handleSave}
        />
        <Button size="small" type="primary" data-testid="sched-tpl-save" onClick={handleSave}>
          存为模板
        </Button>
      </Space>
      {templates.length >= TEMPLATES_CAP && (
        <span className="sched-dim" style={{ fontSize: 12 }}>
          已存满 {TEMPLATES_CAP} 个模板，再存将淘汰最旧模板
        </span>
      )}

      {/* 下半：模板列表 / 空态 */}
      {templates.length === 0 ? (
        <span className="sched-dim">暂无模板——把当前计划另存为模板，新工程从这里起步</span>
      ) : (
        <div style={{ display: 'grid', gap: 4 }}>
          {templates.map((t) => (
            <div
              key={t.name}
              data-testid="sched-template-item"
              data-tpl-name={t.name}
              style={{ display: 'flex', alignItems: 'center', gap: 6, padding: '3px 6px', borderRadius: 6 }}
            >
              <span
                style={{ flex: 1, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
              >
                {t.name}
              </span>
              <span className="sched-dim" style={{ fontSize: 12 }}>{t.savedAt}</span>
              <Popconfirm title="载入将覆盖当前计划（可撤销），确定？" onConfirm={() => importProject(t.project)}>
                <Button size="small" type="text" data-testid="sched-tpl-load">载入</Button>
              </Popconfirm>
              <Popconfirm title={`删除模板「${t.name}」？`} onConfirm={() => setTemplates(deleteTemplate(t.name))}>
                <Button size="small" type="text" danger data-testid="sched-tpl-del" aria-label={`删除模板${t.name}`}>删除</Button>
              </Popconfirm>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
