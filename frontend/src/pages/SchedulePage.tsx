/**
 * SchedulePage — 「进度计划」一级板块（v4.110 刀1 / v4.111 刀2 / v4.113 刀4）
 *
 * 工程进度计划编制工作台：一套任务表数据驱动三种视图自由切换——
 * 横道图（甘特）/ 单代号网络图（PDM 六格）/ 双代号网络图（AOA 虚工作自动生成）。
 * CPM 引擎纯前端计算（FS/SS/FF/SF + 时距、正逆推、总/自由时差、关键线路、
 * 手动/自动任务模式），工作日历推算日期，MS Project XML 导入导出互通。
 * 刀4 起：计划文件化（进度计划/当前计划.gsched.json）——板块与 agent 共享
 * 同一资产（GaeaScheduleLoad/Save 水合+自动保存，agent 经 schedule_* 工具读写）。
 */
import React, { useEffect, useMemo, useRef, useState } from 'react'
import { Alert, Button, Checkbox, Input, Popconfirm, Popover, Segmented, Space, Tag, Tooltip } from 'antd'
import {
  AimOutlined, CalendarOutlined, ClearOutlined, ClusterOutlined, ExportOutlined, FundOutlined, ImportOutlined,
  MessageOutlined, NodeIndexOutlined, PlusOutlined, TableOutlined, TeamOutlined, ThunderboltOutlined, PartitionOutlined, DeleteOutlined, ToolOutlined,
} from '@ant-design/icons'
import { computeCpm } from '../schedule/cpm'
import { computeCosts } from '../schedule/cost'
import { buildAoa } from '../schedule/aoa'
import { buildProjectXml, parseProjectXml } from '../schedule/mspdi'
import { computeBaselineDrift } from '../schedule/baseline'
import { checkDeadline, planFinishOf } from '../schedule/deadline'
import type { CpmResult } from '../schedule/types'
import { useScheduleStore, isGroupRow, initScheduleSync } from '../schedule/store'
import { ResourcePanel } from '../schedule/ResourcePanel'
import { fmtCost, hasCostData } from '../schedule/costUi'
import { SCHEDULE_FILE_PATH } from '../schedule/gschedSummary'
import { usePreviewStore } from '../gaea/lib/store'
import { emitFrontendEvent, FRONTEND_EVENTS } from '../events'
import { ScheduleChatPane } from '../schedule/ChatPane'
import { clampChatWidth, loadChatPrefs, saveChatPrefs, CHAT_WIDTH_DEFAULT } from '../schedule/chatPrefs'
import { GanttView } from '../schedule/GanttView'
import { PdmView } from '../schedule/PdmView'
import { AoaView } from '../schedule/AoaView'
import '../schedule/schedule.css'
import '../gaea/styles.css'
import '../gaea/redesign.css'
import '../gaea/tailwind.css'

const WEEKDAYS: { day: number; label: string }[] = [
  { day: 1, label: '一' }, { day: 2, label: '二' }, { day: 3, label: '三' },
  { day: 4, label: '四' }, { day: 5, label: '五' }, { day: 6, label: '六' }, { day: 0, label: '日' },
]

/** 日历设置弹层：周工作制 + 节假日例外 */
const CalendarEditor: React.FC = () => {
  const project = useScheduleStore((s) => s.project)
  const setCalendar = useScheduleStore((s) => s.setCalendar)
  const cal = project.calendar ?? { workweek: [1, 2, 3, 4, 5], holidays: [] }
  const [holidayDraft, setHolidayDraft] = useState('')

  const toggleDay = (day: number, on: boolean) => {
    const next = on ? [...new Set([...cal.workweek, day])] : cal.workweek.filter((d) => d !== day)
    if (next.length === 0) return // 至少保留一个工作日
    setCalendar({ workweek: next, holidays: cal.holidays })
  }
  const addHoliday = () => {
    if (!/^\d{4}-\d{2}-\d{2}$/.test(holidayDraft) || cal.holidays.includes(holidayDraft)) return
    setCalendar({ workweek: cal.workweek, holidays: [...cal.holidays, holidayDraft].sort() })
    setHolidayDraft('')
  }

  return (
    <div style={{ display: 'grid', gap: 8, minWidth: 280 }}>
      <div>
        <div className="sched-dim" style={{ marginBottom: 4 }}>周工作制（至少一项）</div>
        <Space size={4} wrap>
          {WEEKDAYS.map((w) => (
            <Checkbox
              key={w.day}
              checked={cal.workweek.includes(w.day)}
              onChange={(e) => toggleDay(w.day, e.target.checked)}
            >
              周{w.label}
            </Checkbox>
          ))}
        </Space>
      </div>
      <div>
        <div className="sched-dim" style={{ marginBottom: 4 }}>节假日 / 停工日（YYYY-MM-DD）</div>
        <Space size={4} wrap style={{ marginBottom: 6 }}>
          {cal.holidays.map((h) => (
            <Tag key={h} closable onClose={() => setCalendar({ workweek: cal.workweek, holidays: cal.holidays.filter((x) => x !== h) })}>
              {h}
            </Tag>
          ))}
          {cal.holidays.length === 0 && <span className="sched-dim">无</span>}
        </Space>
        <Space size={4}>
          <Input size="small" placeholder="2026-10-01" value={holidayDraft} style={{ width: 130 }}
            onChange={(e) => setHolidayDraft(e.target.value)} onPressEnter={addHoliday} />
          <Button size="small" type="dashed" onClick={addHoliday}>添加</Button>
        </Space>
      </div>
      <span className="sched-dim">工期/时距均按工作日计算；日期轴自动跳过非工作日。</span>
    </div>
  )
}

/** 偏移量文本：+N / -N / ±0 */
function fmtDrift(n: number): string {
  return n > 0 ? `+${n}` : `${n}`
}

/** 基线漂移行类别徽标 */
const DRIFT_KIND_LABEL: Record<'shifted' | 'added' | 'removed', string> = {
  shifted: '推移',
  added: '新增',
  removed: '移除',
}

/**
 * 基线对比弹层（v4.116 刀7）：无基线时引导保存；有基线时展示漂移摘要
 * （总工期漂移/推移·新增·移除/关键链进出/偏差行清单）与更新、清除操作。
 */
const BaselinePanel: React.FC<{ cpm: CpmResult }> = ({ cpm }) => {
  const project = useScheduleStore((s) => s.project)
  const setBaseline = useScheduleStore((s) => s.setBaseline)
  const clearBaseline = useScheduleStore((s) => s.clearBaseline)
  const [err, setErr] = useState<string | null>(null)
  const drift = useMemo(() => computeBaselineDrift(project, cpm), [project, cpm])
  const hasLeaf = project.tasks.some((t) => t.level > 0)
  const guard = !cpm.ok ? '计划存在循环依赖，先修正搭接' : !hasLeaf ? '计划还没有任何任务' : null

  const save = () => {
    const e = setBaseline()
    setErr(e)
  }

  return (
    <div style={{ display: 'grid', gap: 8, minWidth: 300, maxWidth: 400 }}>
      {!drift ? (
        <>
          <span className="sched-dim">基线会固化当前排程结果；之后每次调整都能对比「较基线」的推移、总工期漂移与关键链变化。</span>
          {guard && <span className="sched-dim">暂不可保存：{guard}</span>}
          <Button size="small" type="primary" disabled={!!guard} onClick={save}>保存基线</Button>
          {err && <span className="sched-critical-text">{err}</span>}
        </>
      ) : (
        <>
          <div style={{ display: 'flex', alignItems: 'baseline', gap: 8 }}>
            <strong>{drift.baselineName}</strong>
            <span className="sched-dim">保存于 {drift.baselineSavedAt}</span>
          </div>
          <div style={{ fontSize: 13 }}>
            总工期 <b>{drift.baselineDuration}</b> → <b>{drift.currentDuration}</b> 天
            {drift.durationDrift !== 0 ? (
              <span className={drift.durationDrift > 0 ? 'sched-critical-text' : 'sched-drift-good'}>
                （{fmtDrift(drift.durationDrift)} 天，{drift.durationDrift > 0 ? '拖后' : '提前'}）
              </span>
            ) : (
              <span className="sched-dim">（与基线持平）</span>
            )}
          </div>
          <div className="sched-dim">
            推移 {drift.shiftedCount} · 新增 {drift.addedCount} · 移除 {drift.removedCount} · 一致 {drift.sameCount}
          </div>
          {(drift.criticalGained.length > 0 || drift.criticalLost.length > 0) && (
            <div style={{ fontSize: 12, display: 'grid', gap: 2 }}>
              {drift.criticalGained.length > 0 && <div><span className="sched-critical-text">新进关键</span>：{drift.criticalGained.join('、')}</div>}
              {drift.criticalLost.length > 0 && <div><span className="sched-drift-good">退出关键</span>：{drift.criticalLost.join('、')}</div>}
            </div>
          )}
          {drift.rows.length > 0 && (
            <div style={{ maxHeight: 180, overflow: 'auto', display: 'grid', gap: 3 }}>
              {drift.rows.map((r) => (
                <div key={r.id} style={{ display: 'flex', gap: 6, fontSize: 12, alignItems: 'center' }}>
                  <span className={`sched-base-chip sched-base-${r.kind}`}>{DRIFT_KIND_LABEL[r.kind]}</span>
                  <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{r.name}</span>
                  <span className="sched-dim">
                    {r.kind === 'removed'
                      ? `原第 ${r.base!.es}~${r.base!.ef} 工作日`
                      : r.kind === 'added'
                        ? `第 ${r.now!.es}~${r.now!.ef} 工作日`
                        : `第 ${r.base!.es}→${r.now!.es} 工作日`}
                  </span>
                </div>
              ))}
            </div>
          )}
          {guard && <span className="sched-dim">暂不可更新：{guard}</span>}
          <Space size={6}>
            <Button size="small" type="primary" ghost disabled={!!guard} onClick={save}>更新基线</Button>
            <Popconfirm title="清除基线？" description="清除后基线对比不可用，横道图不再显示基线条" onConfirm={() => { clearBaseline(); setErr(null) }}>
              <Button size="small" danger>清除基线</Button>
            </Popconfirm>
          </Space>
          {err && <span className="sched-critical-text">{err}</span>}
        </>
      )}
    </div>
  )
}

const SchedulePage: React.FC = () => {
  const project = useScheduleStore((s) => s.project)
  const view = useScheduleStore((s) => s.view)
  const setView = useScheduleStore((s) => s.setView)
  const selectedId = useScheduleStore((s) => s.selectedId)
  const renameProject = useScheduleStore((s) => s.renameProject)
  const setStartDate = useScheduleStore((s) => s.setStartDate)
  const setDeadline = useScheduleStore((s) => s.setDeadline)
  const importProject = useScheduleStore((s) => s.importProject)
  const addTask = useScheduleStore((s) => s.addTask)
  const addGroup = useScheduleStore((s) => s.addGroup)
  const updateTask = useScheduleStore((s) => s.updateTask)
  const removeTask = useScheduleStore((s) => s.removeTask)
  const loadSample = useScheduleStore((s) => s.loadSample)
  const clearAll = useScheduleStore((s) => s.clearAll)
  const fileRef = useRef<HTMLInputElement>(null)
  const [importMsg, setImportMsg] = useState<{ type: 'success' | 'error'; text: string } | null>(null)
  const hydrated = useScheduleStore((s) => s.hydrated)
  const sync = useScheduleStore((s) => s.sync)
  const savedAt = useScheduleStore((s) => s.savedAt)
  const syncError = useScheduleStore((s) => s.syncError)
  /** 左栏 AI 对话偏好（刀11：折叠态+宽度持久化 localStorage） */
  const [chat, setChat] = useState(loadChatPrefs)
  const chatDragRef = useRef<{ startX: number; startW: number } | null>(null)

  const toggleChat = () => setChat(saveChatPrefs({ collapsed: !chat.collapsed }))

  /** 拖宽分隔条：左移加宽；拖完持久化；双击复位 420 */
  const onDividerMouseDown = (e: React.MouseEvent) => {
    e.preventDefault()
    chatDragRef.current = { startX: e.clientX, startW: chat.width }
    const onMove = (ev: MouseEvent) => {
      const w = clampChatWidth(chatDragRef.current!.startW - (ev.clientX - chatDragRef.current!.startX))
      setChat((p) => ({ ...p, width: w }))
    }
    const onUp = () => {
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
      setChat((p) => saveChatPrefs({ width: p.width }))
    }
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
  }
  const resetChatWidth = () => setChat(saveChatPrefs({ width: CHAT_WIDTH_DEFAULT }))

  /** 反向入口（刀12 办公联动）：在办公板块中预览计划文件（store 预置，未挂载也能落地） */
  const openInOffice = () => {
    usePreviewStore.getState().openFilePreview(SCHEDULE_FILE_PATH)
    emitFrontendEvent(FRONTEND_EVENTS.NAVIGATE, { page: 'gaea' })
  }

  // 文件同步：水合（文件为准，localStorage 迁移）+ 自动保存 + agent 写入回读
  useEffect(() => { void initScheduleSync() }, [])

  const cpm = useMemo(() => computeCpm(project.tasks, project.links, { planFinish: planFinishOf(project) }), [project])
  const aoa = useMemo(() => buildAoa(project.tasks, project.links, { planFinish: planFinishOf(project) }), [project])
  const drift = useMemo(() => computeBaselineDrift(project, cpm), [project, cpm])
  const dl = useMemo(() => checkDeadline(project, cpm), [project, cpm])
  // 资源成本（刀3）：总成本随每次 render 重算（与 CPM 同范式）；无资源/成本数据不显示（诚实呈现）
  const costs = useMemo(() => computeCosts(project, cpm), [project, cpm])
  const showCost = hasCostData(project) && costs.ok
  const hasResources = (project.resources?.length ?? 0) > 0

  const selected = project.tasks.find((t) => t.id === selectedId) ?? null
  const selectedIsGroup = selected ? isGroupRow(project.tasks, project.tasks.findIndex((t) => t.id === selected!.id)) : false

  const exportXml = () => {
    const xml = buildProjectXml(project, cpm.rows)
    const blob = new Blob([xml], { type: 'application/xml;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${project.name || '进度计划'}.xml`
    a.click()
    URL.revokeObjectURL(url)
  }

  const onImportFile = async (file: File) => {
    const text = await file.text()
    const r = parseProjectXml(text)
    if (r.ok && r.project) {
      importProject(r.project)
      setImportMsg({ type: 'success', text: `已导入「${r.project.name}」：${r.project.tasks.length} 行 / ${r.project.links.length} 条搭接` })
    } else {
      setImportMsg({ type: 'error', text: r.error ?? '导入失败' })
    }
  }

  return (
    <div className="sched-shell">
      {!chat.collapsed && (
        <div className="sched-chat-pane" style={{ width: chat.width }}>
          <ScheduleChatPane />
        </div>
      )}
      {!chat.collapsed && (
        <div
          className="sched-chat-divider"
          title="拖动调宽 · 双击复位"
          onMouseDown={onDividerMouseDown}
          onDoubleClick={resetChatWidth}
        />
      )}
      <div className="sched-page" style={{ flex: 1, minWidth: 0, padding: '12px 16px' }}>
      <div className="sched-header">
        <h2 className="sched-header-title">进度计划</h2>
        <Button
          size="small"
          type={chat.collapsed ? 'primary' : 'default'}
          ghost={chat.collapsed}
          icon={<MessageOutlined />}
          onClick={toggleChat}
          title={chat.collapsed ? '展开左栏 AI 对话（对话即排程）' : '收起左栏 AI 对话'}
          aria-label="切换 AI 对话栏"
        />
        <Tooltip title="在办公板块中预览计划文件（进度计划/当前计划.gsched.json）">
          <Button size="small" icon={<ToolOutlined />} onClick={openInOffice} aria-label="在办公板块中查看计划" />
        </Tooltip>
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
        <Tooltip title="目标竣工日期（合同/定额工期约束）：引擎按工作日换算并裁决可行性，横道图画竣工线">
          <Space size={4}>
            <span className="sched-dim">目标竣工</span>
            <Input
              type="date"
              size="small"
              value={project.deadline ?? ''}
              onChange={(e) => setDeadline(e.target.value || null)}
              style={{ width: 140 }}
            />
          </Space>
        </Tooltip>
        <Popover trigger="click" placement="bottom" content={<CalendarEditor />} title="工作日历">
          <Button size="small" icon={<CalendarOutlined />}>日历</Button>
        </Popover>
        <Popover trigger="click" placement="bottom" content={<BaselinePanel cpm={cpm} />} title="基线对比">
          <Button
            size="small"
            icon={<FundOutlined />}
            type={project.baseline ? 'primary' : 'default'}
            ghost={!!project.baseline}
            title="保存基线快照，对比每次调整的推移与关键链变化"
          >
            基线{project.baseline ? `·${project.baseline.name}` : ''}
          </Button>
        </Popover>
        <Popover trigger="click" placement="bottom" content={<ResourcePanel cpm={cpm} />} title="资源与成本">
          <Button
            size="small"
            icon={<TeamOutlined />}
            type={hasResources ? 'primary' : 'default'}
            ghost={hasResources}
            data-testid="sched-resource-btn"
            title="资源工作表：工时/材料/成本三类资源与费率；任务行「资源」入口挂载分配"
          >
            资源
          </Button>
        </Popover>
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
        {selected && (
          <Popconfirm
            title="删除选中项"
            description={selectedIsGroup ? '分组及其子任务一并删除' : `删除「${selected!.name}」及其搭接关系`}
            onConfirm={() => removeTask(selected!.id)}
          >
            <Button size="small" danger icon={<DeleteOutlined />}>删除</Button>
          </Popconfirm>
        )}
        <div className="sched-tool-divider" />
        <input
          ref={fileRef}
          type="file"
          accept=".xml,application/xml,text/xml"
          style={{ display: 'none' }}
          onChange={(e) => {
            const f = e.target.files?.[0]
            e.target.value = ''
            if (f) void onImportFile(f)
          }}
        />
        <Tooltip title="导入 MS Project XML（mspdi 口径），覆盖当前工程">
          <Button size="small" icon={<ImportOutlined />} onClick={() => fileRef.current?.click()}>导入 XML</Button>
        </Tooltip>
        <Tooltip title="导出为 MS Project XML（可被 Project / 斑马进度打开）">
          <Button size="small" icon={<ExportOutlined />} onClick={exportXml}>导出 XML</Button>
        </Tooltip>
        <div className="sched-tool-divider" />
        <Tooltip title="载入示例工程（办公楼施工），覆盖当前数据">
          <Button size="small" icon={<ThunderboltOutlined />} onClick={() => { loadSample(); setImportMsg(null) }}>示例工程</Button>
        </Tooltip>
        <Popconfirm title="清空全部任务与搭接？" onConfirm={() => { clearAll(); setImportMsg(null) }}>
          <Button size="small" icon={<ClearOutlined />}>清空</Button>
        </Popconfirm>
      </div>

      {importMsg && (
        <Alert
          type={importMsg.type}
          showIcon
          closable
          message={importMsg.text}
          onClose={() => setImportMsg(null)}
        />
      )}
      {!cpm.ok && cpm.error && (
        <Alert type="error" showIcon message={cpm.error} description="请修正搭接关系后重试；网络图视图在循环解除前不可用。" />
      )}

      {view === 'gantt' && <GanttView project={project} cpm={cpm} />}
      {view === 'pdm' && <PdmView project={project} cpm={cpm} />}
      {view === 'aoa' && <AoaView graph={aoa} tasks={project.tasks} />}

      {/* 底部状态栏（斑马口径：共 N 项工作总工期 N 天 + 关键/工作制常驻） */}
      <div className="sched-statusbar">
        <span>视图：<span className="sched-sb-strong">{view === 'gantt' ? '横道图' : view === 'pdm' ? '单代号网络图' : '双代号网络图'}</span></span>
        <span>共 <span className="sched-sb-strong">{project.tasks.filter((t) => t.level > 0).length}</span> 项工作，总工期 <span className="sched-sb-strong">{cpm.duration}</span> 天</span>
        <span>关键工作 <span className="sched-sb-crit">{project.tasks.filter((t) => t.level > 0 && cpm.rows[t.id]?.critical).length}</span> 项</span>
        {showCost && (
          <span data-testid="sched-statusbar-cost" title="总成本 = Σ任务（固定成本 + 分配成本），随工期实时重算（元）">
            总成本 <span className="sched-sb-strong">¥{fmtCost(costs.total)}</span>
          </span>
        )}
        {drift && (
          <span title={`基线「${drift.baselineName}」保存于 ${drift.baselineSavedAt}`}>
            基线 <span className="sched-sb-strong">{drift.baselineDuration}</span> → 当前 {drift.currentDuration} 天
            {drift.durationDrift !== 0 && (
              <span className={drift.durationDrift > 0 ? 'sched-sb-crit' : 'sched-sb-good'}>（{fmtDrift(drift.durationDrift)}）</span>
            )}
          </span>
        )}
        {dl && (
          <span title="倒排校核：目标竣工按工作日换算（第 N 个工作日竣工）">
            目标竣工 <span className="sched-sb-strong">{dl.deadline}</span>（第 {dl.targetWorkdays} 工作日）
            {!dl.feasible && <span className="sched-sb-crit">超 {dl.overrun} 天</span>}
            {dl.feasible && dl.overrun < 0 && <span className="sched-sb-good">富余 {-dl.overrun} 天</span>}
            {dl.feasible && dl.overrun === 0 && <span className="sched-sb-good">压线达成</span>}
          </span>
        )}
        <span>工作制：<span className="sched-sb-strong">{workweekLabel(project.calendar)}</span>{(project.calendar?.holidays.length ?? 0) > 0 && <> · 节假日 {project.calendar!.holidays.length} 天</>}</span>
        <span>时间单位：工作日（日期轴为自然日）</span>
        <span style={{ marginLeft: 'auto' }} title={syncError ?? '进度计划/当前计划.gsched.json'}>
          {!hydrated ? '读取计划…' : sync === 'dirty' ? '改动待保存…'
            : sync === 'saving' ? '保存中…'
            : sync === 'error' ? <span className="sched-sb-crit">保存失败</span>
            : savedAt ? <>已保存 {savedAt} · <span className="sched-dim">agent 可读写</span></> : '已同步 · agent 可读写'}
        </span>
      </div>
      </div>
    </div>
  )
}

/** 工作制描述（周一~周五 简写） */
function workweekLabel(cal?: { workweek: number[] }): string {
  const names = ['日', '一', '二', '三', '四', '五', '六']
  const wk = (cal?.workweek?.length ? [...cal.workweek] : [1, 2, 3, 4, 5]).sort((a, b) => ((a + 6) % 7) - ((b + 6) % 7))
  return '周' + wk.map((d) => names[d]).join('、')
}

export default SchedulePage
