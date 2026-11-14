/**
 * ExportDialog — 导出图面弹窗（SchedulePage 菜单「导出图面（PNG/PDF/打印）」）
 * v4.141 导出预览（所见即所得）。瘦身 P3：自 pages/SchedulePage.tsx 分片，正文零改动。
 */
import React, { useEffect, useMemo, useState } from 'react'
import { Button, Checkbox, Input, InputNumber, Modal, Segmented, Select, Space } from 'antd'
import { buildGanttExportSvg } from '../../schedule/ganttExport'
import { buildAoaExportSvg, buildPdmExportSvg } from '../../schedule/networkExport'
import { printSvg, saveExportBlob, svgToPdfBlob, svgToPngBlob, svgWithViewBox } from '../../schedule/exportArtifact'
import { buildPaperSvg } from '../../schedule/paperExport'
import { applyExportPalette } from '../../schedule/exportPalette'
import type { DisplayPreset } from '../../schedule/displayPreset'
import { PAPER_MM, type PageOrientation, type PaperKind, type PaperLayout } from '../../schedule/pageLayout'
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
  /** 显示预设（W6）：导出件跟随屏上护眼/黑白配色（normal 零改动） */
  preset: DisplayPreset
}> = ({ open, onClose, project, cpm, aoa, defaultView, preset }) => {
  const [meta, setMeta] = useState<ExportMetaPrefs>(loadExportMeta)
  const [msg, setMsg] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [kind, setKind] = useState<ExportKind>(defaultView)
  /** 横道条尾标注（v4.136 PDF 对比余项：任务名（工期）写在条上；网络图不受控） */
  const [barLabels, setBarLabels] = useState(true)
  /** 导出预览（v4.141）：构建器产物注入 viewBox 后等比缩放进弹窗，所见即所得 */
  const [previewSvg, setPreviewSvg] = useState<string | null>(null)
  /**
   * W6 按纸张布图：勾选后预览与导出都走 paperExport 的多页纸堆。
   * 会话态（不落偏好）——纸张规格是当次出图决策，不是长期设置。
   */
  const [paperOn, setPaperOn] = useState(false)
  const [paper, setPaper] = useState<PaperKind>('A4')
  const [orient, setOrient] = useState<PageOrientation>('landscape')
  const [marginMm, setMarginMm] = useState(10)
  /** 纸堆读数（页数/缩放/截断）：预览文案与 PNG/PDF 门控共用 */
  const [sheet, setSheet] = useState<{ pages: number; scale: number; truncated: boolean } | null>(null)
  const layout = useMemo<PaperLayout>(() => ({ paper, orientation: orient, marginMm }), [paper, orient, marginMm])
  const pageWmm = orient === 'landscape' ? PAPER_MM[paper].h : PAPER_MM[paper].w
  /** 纸堆页数超限时 PNG/PDF 不可用（canvas 单图编码上限），打印不受影响 */
  const tooManyPages = paperOn && !!sheet && sheet.pages > 20
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
        const tinted = { ...art, svg: applyExportPalette(art.svg, preset) }
        if (paperOn) {
          const sheetOut = buildPaperSvg(tinted, layout, { title: project.name })
          setPreviewSvg(svgWithViewBox(sheetOut.svg))
          setSheet({ pages: sheetOut.pages, scale: sheetOut.scale, truncated: sheetOut.truncated })
        } else {
          setPreviewSvg(svgWithViewBox(tinted.svg))
          setSheet(null)
        }
      } catch {
        setPreviewSvg(null)
        setSheet(null)
      }
    }, 350)
    return () => window.clearTimeout(t)
  }, [open, kind, barLabels, meta, project, cpm, aoa, paperOn, layout, preset])

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
      // 按纸张布图：纸堆页间不留缝，打印时每页恰好落在一张纸上
      const tinted = { ...art, svg: applyExportPalette(art.svg, preset) }
      const framed = paperOn ? buildPaperSvg(tinted, layout, { title: project.name, gap: 0 }) : null
      const outSvg = framed ? framed.svg : tinted.svg
      const suffix = paperOn ? `-${paper}${orient === 'landscape' ? '横' : '竖'}` : ''
      if (out === 'print') {
        printSvg(outSvg, framed ? { pageSize: `${paper} ${orient}`, margin: '0', svgWidthMm: pageWmm } : undefined)
      } else if (out === 'png') await saveExportBlob(await svgToPngBlob(outSvg), `${base}${suffix}.png`)
      else await saveExportBlob(await svgToPdfBlob(outSvg), `${base}${suffix}.pdf`)
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
        <Space size={8} wrap>
          <Checkbox
            checked={paperOn}
            onChange={(e) => setPaperOn(e.target.checked)}
            data-testid="sched-export-paper"
          >
            按纸张布图
          </Checkbox>
          <Select
            size="small"
            value={paper}
            disabled={!paperOn}
            onChange={(v) => setPaper(v as PaperKind)}
            options={[{ value: 'A4', label: 'A4' }, { value: 'A3', label: 'A3' }]}
            data-testid="sched-export-paper-kind"
          />
          <Segmented
            size="small"
            value={orient}
            disabled={!paperOn}
            onChange={(v) => setOrient(v as PageOrientation)}
            options={[{ value: 'landscape', label: '横向' }, { value: 'portrait', label: '纵向' }]}
            data-testid="sched-export-paper-orient"
          />
          <InputNumber
            size="small"
            min={5}
            max={30}
            value={marginMm}
            disabled={!paperOn}
            onChange={(v) => v != null && setMarginMm(v)}
            addonAfter="mm"
            data-testid="sched-export-paper-margin"
          />
          {paperOn && sheet && (
            <span className="sched-dim" style={{ fontSize: 12 }} data-testid="sched-export-paper-info">
              共 {sheet.pages} 页 · 缩放 {Math.round(sheet.scale * 100)}%
              {sheet.truncated ? ' · 已截断（超出 400 页上限）' : ''}
            </span>
          )}
        </Space>
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
          {/* 画布单图上限（浏览器 ~32767px）：纸堆超过 20 页时 PNG/PDF 会编码失败，
              这里提前禁用并给原因，打印仍可走浏览器分页 */}
          <Button
            type="primary"
            loading={busy}
            data-testid="sched-export-png"
            disabled={tooManyPages}
            title={tooManyPages ? '页数过多：PNG 需整图编码（浏览器画布上限 ~32767px），请改用打印或缩小纸张' : undefined}
            onClick={() => void run('png')}
          >导出 PNG</Button>
          <Button
            data-testid="sched-export-pdf"
            disabled={busy || tooManyPages}
            title={tooManyPages ? '页数过多：PDF 走整图编码，请改用打印（浏览器分页）' : undefined}
            onClick={() => void run('pdf')}
          >导出 PDF</Button>
          <Button data-testid="sched-export-print" disabled={busy} onClick={() => void run('print')}>打印</Button>
        </Space>
        {msg && <div data-testid="sched-export-msg" style={{ fontSize: 12 }}>{msg}</div>}
      </div>
    </Modal>
  )
}