/**
 * ExportDialog — 导出图面弹窗（SchedulePage 菜单「导出图面（PNG/PDF/打印）」）
 * v4.141 导出预览（所见即所得）。瘦身 P3：自 pages/SchedulePage.tsx 分片，正文零改动。
 */
import React, { useEffect, useState } from 'react'
import { Button, Checkbox, Input, Modal, Segmented, Space } from 'antd'
import { buildGanttExportSvg } from '../../schedule/ganttExport'
import { buildAoaExportSvg, buildPdmExportSvg } from '../../schedule/networkExport'
import { printSvg, saveExportBlob, svgToPdfBlob, svgToPngBlob, svgWithViewBox } from '../../schedule/exportArtifact'
import { loadExportMeta, saveExportMeta, todayIso, type ExportMetaPrefs } from '../../schedule/exportMeta'
import type { AoaGraph } from '../../schedule/aoa'
import type { CpmResult, SchedProject } from '../../schedule/types'

type ExportKind = 'gantt' | 'aoa' | 'pdm'
const EXPORT_KIND_LABEL: Record<ExportKind, string> = { gantt: '横道图', aoa: '双代号网络图', pdm: '单代号网络图' }
const EXPORT_KIND_FILE: Record<ExportKind, string> = {
  gantt: '施工进度计划横道图', aoa: '双代号时标网络图', pdm: '单代号网络图',
}

export const ExportDialog: React.FC<{
  open: boolean
  onClose: () => void
  project: SchedProject
  cpm: CpmResult
  aoa: AoaGraph
  defaultView: 'gantt' | 'pdm' | 'aoa'
}> = ({ open, onClose, project, cpm, aoa, defaultView }) => {
  const [meta, setMeta] = useState<ExportMetaPrefs>(loadExportMeta)
  const [msg, setMsg] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [kind, setKind] = useState<ExportKind>(defaultView)
  /** 横道条尾标注（v4.136 PDF 对比余项：任务名（工期）写在条上；网络图不受控） */
  const [barLabels, setBarLabels] = useState(true)
  /** 导出预览（v4.141）：构建器产物注入 viewBox 后等比缩放进弹窗，所见即所得 */
  const [previewSvg, setPreviewSvg] = useState<string | null>(null)
  useEffect(() => {
    if (open) setKind(defaultView) // 每次打开跟随当前视图
  }, [open, defaultView])
  useEffect(() => {
    if (!open || !cpm.ok) {
      setPreviewSvg(null)
      return
    }
    const t = window.setTimeout(() => {
      try {
        const m = { ...meta, date: meta.date || todayIso(), barLabels }
        const art = kind === 'gantt'
          ? buildGanttExportSvg(project, cpm, m)
          : kind === 'aoa'
            ? buildAoaExportSvg(project, aoa, m)
            : buildPdmExportSvg(project, cpm, m)
        setPreviewSvg(svgWithViewBox(art.svg))
      } catch {
        setPreviewSvg(null)
      }
    }, 350)
    return () => window.clearTimeout(t)
  }, [open, kind, barLabels, meta, project, cpm, aoa])

  const setField = (k: keyof ExportMetaPrefs) => (e: React.ChangeEvent<HTMLInputElement>) =>
    setMeta((p) => ({ ...p, [k]: e.target.value }))

  const run = async (out: 'png' | 'pdf' | 'print') => {
    if (!cpm.ok) { setMsg('计划存在循环依赖，导出前先修正搭接'); return }
    setBusy(true)
    setMsg(null)
    try {
      const m = { ...meta, date: meta.date || todayIso(), barLabels }
      const art = kind === 'gantt'
        ? buildGanttExportSvg(project, cpm, m)
        : kind === 'aoa'
          ? buildAoaExportSvg(project, aoa, m)
          : buildPdmExportSvg(project, cpm, m)
      const base = `${project.name || '进度计划'}-${EXPORT_KIND_FILE[kind]}`
      if (out === 'print') printSvg(art.svg)
      else if (out === 'png') await saveExportBlob(await svgToPngBlob(art.svg), `${base}.png`)
      else await saveExportBlob(await svgToPdfBlob(art.svg), `${base}.pdf`)
      saveExportMeta(meta)
      setMsg(out === 'print' ? '已唤起系统打印（目标选「另存为 PDF」可得 PDF 文件）' : '已导出')
    } catch (e) {
      setMsg(`导出失败：${e instanceof Error ? e.message : String(e)}`)
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal open={open} onCancel={onClose} title="导出图面（上报件）" footer={null} width={720} destroyOnClose>
      <div className="sched-export-modal" data-testid="sched-export-modal" style={{ display: 'grid', gap: 10 }}>
        <Segmented
          value={kind}
          onChange={(v) => setKind(v as ExportKind)}
          options={(Object.keys(EXPORT_KIND_LABEL) as ExportKind[]).map((k) => ({ value: k, label: EXPORT_KIND_LABEL[k] }))}
        />
        {/* 预览（v4.141）：构建器 SVG 注入 viewBox 等比缩放，导出前所见即所得 */}
        <div className="sched-export-preview" data-testid="sched-export-preview">
          {previewSvg ? (
            <div className="sched-export-preview-body" dangerouslySetInnerHTML={{ __html: previewSvg }} />
          ) : (
            <span className="sched-dim">{cpm.ok ? '正在生成预览…' : '计划存在循环依赖，无法生成图面，请先修正搭接'}</span>
          )}
        </div>
        <span className="sched-dim" style={{ fontSize: 12 }}>
          {kind === 'gantt'
            ? '横道图：标题带 + 上报 6 列（序号/任务名称/工期/开始/完成/前置）+ 双行时标 + 图例 + 图签。'
            : kind === 'aoa'
              ? '双代号：时标网络（1 格=1 天，波形线=自由时差）+ 工程标尺（工程日/月/日/星期）+ 图例 + 图签。'
              : '单代号：拓扑分层 + 六格节点（ES/工期/EF·LS/总时差/LF）+ 绑定红链 + 图例 + 图签。'}
        </span>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
          <label style={{ display: 'grid', gap: 2, fontSize: 12 }}><span className="sched-dim">编制单位</span>
            <Input size="small" value={meta.org} onChange={setField('org')} placeholder="（可空）" />
          </label>
          <label style={{ display: 'grid', gap: 2, fontSize: 12 }}><span className="sched-dim">编制日期</span>
            <Input size="small" type="date" value={meta.date || todayIso()} onChange={setField('date')} />
          </label>
          <label style={{ display: 'grid', gap: 2, fontSize: 12 }}><span className="sched-dim">编制人</span>
            <Input size="small" value={meta.designer} onChange={setField('designer')} placeholder="（可空，图签留白签章位）" />
          </label>
          <label style={{ display: 'grid', gap: 2, fontSize: 12 }}><span className="sched-dim">审核人</span>
            <Input size="small" value={meta.reviewer} onChange={setField('reviewer')} placeholder="（可空）" />
          </label>
          <label style={{ display: 'grid', gap: 2, fontSize: 12 }}><span className="sched-dim">批准</span>
            <Input size="small" value={meta.approver} onChange={setField('approver')} placeholder="（可空）" />
          </label>
        </div>
        {kind === 'gantt' && (
          <Checkbox
            checked={barLabels}
            onChange={(e) => setBarLabels(e.target.checked)}
            data-testid="sched-export-barlabels"
          >
            条尾标注（任务名与工期写在条上）
          </Checkbox>
        )}
        <label style={{ display: 'grid', gap: 2, fontSize: 12 }}>
          <span className="sched-dim">编制说明（可空，随图面出注：工程概况/控制性节点/关键工序安排）</span>
          <Input.TextArea
            rows={2}
            placeholder="1. 本计划开工日期以开工令为准。 2. 关键线路为……"
            maxLength={600}
            value={meta.notes ?? ''}
            onChange={(e) => setMeta((p2) => ({ ...p2, notes: e.target.value }))}
            data-testid="sched-export-notes"
          />
        </label>
        <Space size={8}>
          <Button type="primary" loading={busy} data-testid="sched-export-png" onClick={() => void run('png')}>导出 PNG</Button>
          <Button data-testid="sched-export-pdf" disabled={busy} onClick={() => void run('pdf')}>导出 PDF</Button>
          <Button data-testid="sched-export-print" disabled={busy} onClick={() => void run('print')}>打印</Button>
        </Space>
        {msg && <div data-testid="sched-export-msg" style={{ fontSize: 12 }}>{msg}</div>}
      </div>
    </Modal>
  )
}