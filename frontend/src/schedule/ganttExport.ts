/**
 * ganttExport.ts — 横道图上报图面构建器（纯函数 → SVG 字符串，v4.132.0 刀D1 /
 * v4.137.0 刀D4 增条尾标注与里程碑旗标）
 *
 * 对标真实上报件口径（MS Project 横道导出 / .gzp 时标计划）：
 *  - 标题带：图名 + 编制单位/编制日期；
 *  - 表格：上报 6 列（序号/任务名称/工期/开始/完成/前置），分组行汇总；
 *  - 图面：自然日双行时标（月/日）+ 非工作日底纹 + 关键红/普通蓝/手动灰条
 *    + 分组汇总条 + 里程碑菱形 + 里程碑小旗 + 条尾「任务名（N天）」标注
 *    + 总时差尾 + 依赖线 + 目标竣工线 + 今日线；
 *  - 图脚：图例 + 图签（编制人/审核人/批准）。
 * 与 GanttView 同一套几何公式（dayNo 换算/条形 left/width/依赖线锚点），但独立
 * 实现——图面是发布物，不随工作台交互态（缩放/列显隐/拖拽预览）漂移。
 * 全部纯函数零 DOM：PNG/PDF/打印管线见 exportArtifact.ts。
 */
import type { CpmResult, SchedProject } from './types'
import { isWorkingDate, normalizeCalendar, wdToDate } from './calendar'
import { ganttLinkPath } from './ganttLinks'
import { descendantIds, isGroupRow } from './store'

/** 图签/标题带信息（空字段诚实留白：图签空格=签章位） */
export interface ExportMeta {
  title?: string
  org?: string
  designer?: string
  reviewer?: string
  approver?: string
  date?: string
  /**
   * 横道条尾标注开关（v4.137.0 刀D4；仅横道图构建器消费，网络图忽略）：
   * 缺省 true=叶任务条末端右侧标「任务名（N天）」（上报件口径默认带标注）。
   */
  barLabels?: boolean
}

export interface GanttExportSvg {
  svg: string
  w: number
  h: number
}

export const EXP_MARGIN = 20 // 图面页边距（网络图导出共用）
const M = EXP_MARGIN
export const EXP_TITLE_H = 60 // 标题带高（共用）
const TITLE_H = EXP_TITLE_H
const HEAD_H = 44 // 双行时标：月 22 + 日 22
const ROW_H = 30
const FOOT_H = 76 // 图例 + 图签 + 底边距
const DAY_W = 14 // 图面固定刻度（发布物可读口径，不随工作台缩放）

/** 上报 6 列的图面列宽（发布物专用，比工作台列宽：长日期/前置引用完整可读） */
const EXP_COLS: { key: string; label: string; w: number }[] = [
  { key: 'no', label: '序号', w: 44 },
  { key: 'name', label: '任务名称', w: 200 },
  { key: 'dur', label: '工期(天)', w: 64 },
  { key: 'start', label: '开始', w: 84 },
  { key: 'finish', label: '完成', w: 84 },
  { key: 'preds', label: '前置', w: 80 },
]
const TABLE_W = EXP_COLS.reduce((s, c) => s + c.w, 0)

export const EXP_COLORS = {
  ink: '#0f172a',
  border: '#475569',
  grid: '#e2e8f0',
  headBg: '#f1f5f9',
  groupBand: '#f8fafc',
  weekend: '#eef2f7',
  bar: '#3b82f6',
  barManual: '#64748b',
  critical: '#dc2626',
  float: '#cbd5e1',
  link: '#94a3b8',
  deadline: '#a855f7',
  today: '#10b981',
  dim: '#64748b',
}
const C = EXP_COLORS
export const EXP_FONT = "system-ui, 'Microsoft YaHei', sans-serif"
const FONT = EXP_FONT

export function esc(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}

/** 估算文本像素宽（CJK 全宽、半角 0.55），超出 maxPx 截断加省略号 */
export function fitText(s: string, maxPx: number, fontPx: number): string {
  const w = (t: string): number => {
    let n = 0
    for (const ch of t) n += ch.charCodeAt(0) > 0xff ? fontPx : fontPx * 0.55
    return n
  }
  if (w(s) <= maxPx) return s
  let out = s
  while (out.length > 1 && w(out + '…') > maxPx) out = out.slice(0, -1)
  return out + '…'
}

/** 估算文本像素宽（与 fitText 同一 CJK 全宽/半角 0.55 口径，用于排版前量宽） */
function textWidth(s: string, fontPx: number): number {
  let n = 0
  for (const ch of s) n += ch.charCodeAt(0) > 0xff ? fontPx : fontPx * 0.55
  return n
}

/**
 * 里程碑小旗（v4.137.0 刀D4）：旗杆竖线 + 三角旗面，data-exp-flag 供导出
 * 图面测试断言。(x,y)=旗杆顶点，poleH=杆高，fw/fh=旗面宽高（旗面自杆顶向右）；
 * 颜色由调用方传 EXP_COLORS 现成色（横道=关键红，网络图各图面同款）。
 */
function expFlagSvg(x: number, y: number, poleH: number, fw: number, fh: number, color: string): string {
  return (
    `<g class="sched-exp-flag" data-exp-flag="1">` +
    `<line x1="${x}" y1="${y}" x2="${x}" y2="${y + poleH}" stroke="${color}" stroke-width="1.5"/>` +
    `<polygon points="${x},${y} ${x + fw},${y + fh / 2} ${x},${y + fh}" fill="${color}"/>` +
    `</g>`
  )
}

/** 分组行汇总跨度（子孙叶项 min ES / max EF，与 GanttView.groupSpan 同口径） */
function groupSpan(project: SchedProject, cpm: CpmResult, idx: number): { es: number; ef: number } | null {
  const ids = descendantIds(project.tasks, project.tasks[idx].id)
  let es = Number.POSITIVE_INFINITY
  let ef = 0
  let any = false
  for (const id of ids) {
    const row = cpm.rows[id]
    const t = project.tasks.find((x) => x.id === id)
    if (!row || !t || t.level === 0) continue
    any = true
    es = Math.min(es, row.es)
    ef = Math.max(ef, row.ef)
  }
  return any ? { es, ef } : null
}

/** 图例（左）+ 图签（右）图脚组 */
function footSvg(meta: ExportMeta, dataBottomY: number, totalW: number): string {
  const y = dataBottomY + 26
  const legend: { fill?: string; stroke?: string; dash?: boolean; diamond?: boolean; label: string }[] = [
    { fill: C.critical, label: '关键工作' },
    { fill: C.bar, label: '非关键工作' },
    { fill: C.barManual, label: '手动锁定' },
    { fill: C.ink, label: '汇总' },
    { fill: C.ink, diamond: true, label: '里程碑' },
    { stroke: C.deadline, dash: true, label: '目标竣工线' },
  ]
  let x = M
  let out = ''
  for (const it of legend) {
    if (it.diamond) {
      out += `<rect x="${x + 6}" y="${y - 4}" width="9" height="9" transform="rotate(45 ${x + 10.5} ${y + 0.5})" fill="${it.fill}"/>`
    } else {
      out += `<rect x="${x}" y="${y - 5}" width="18" height="10" fill="${it.fill ?? 'none'}"${it.stroke ? ` stroke="${it.stroke}" stroke-dasharray="4 3"` : ''}/>`
    }
    out += `<text x="${x + 24}" y="${y + 4}" font-family="${FONT}" font-size="11" fill="${C.ink}">${it.label}</text>`
    x += 24 + it.label.length * 11 + 18
  }
  out += exportSignSvg(meta, y, totalW)
  return out
}

/** 图签（右下三格）：编制人 | 审核人 | 批准，空值留白=签章位（各图面共用） */
export function exportSignSvg(meta: ExportMeta, y: number, totalW: number): string {
  const cells: [string, string][] = [['编制人', meta.designer ?? ''], ['审核人', meta.reviewer ?? ''], ['批准', meta.approver ?? '']]
  const cw = 104
  const ch = 34
  const bx = totalW - M - cells.length * cw
  let out = `<g class="sched-exp-sign">`
  for (let i = 0; i < cells.length; i++) {
    const cx = bx + i * cw
    out += `<rect x="${cx}" y="${y - 8}" width="${cw}" height="${ch}" fill="white" stroke="${C.border}"/>`
    out += `<text x="${cx + 6}" y="${y + 2}" font-family="${FONT}" font-size="9" fill="${C.dim}">${cells[i][0]}</text>`
    if (cells[i][1]) {
      out += `<text x="${cx + cw / 2}" y="${y + 20}" text-anchor="middle" font-family="${FONT}" font-size="11" fill="${C.ink}">${esc(fitText(cells[i][1], cw - 12, 11))}</text>`
    }
  }
  out += `</g>`
  return out
}

/** 标题带（图名居中 + 编制单位/编制日期有值才显示；各图面共用） */
export function exportTitleSvg(meta: ExportMeta, totalW: number, fallbackTitle: string): string {
  const title = meta.title?.trim() || fallbackTitle
  let out = `<text x="${totalW / 2}" y="30" text-anchor="middle" font-family="${FONT}" font-size="18" font-weight="700" fill="${C.ink}">${esc(fitText(title, totalW - M * 2, 18))}</text>`
  const metaParts = [meta.org?.trim() ? `编制单位：${meta.org.trim()}` : '', meta.date?.trim() ? `编制日期：${meta.date.trim()}` : ''].filter(Boolean)
  if (metaParts.length > 0) {
    out += `<text x="${totalW / 2}" y="50" text-anchor="middle" font-family="${FONT}" font-size="12" fill="${C.dim}">${esc(metaParts.join('\u3000\u3000'))}</text>`
  }
  return out
}

/**
 * 构建横道图上报件 SVG。cpm 未过（循环依赖）时抛错——发布物不得出自坏计划。
 */
export function buildGanttExportSvg(project: SchedProject, cpm: CpmResult, meta: ExportMeta = {}): GanttExportSvg {
  if (!cpm.ok) throw new Error('计划存在循环依赖，导出前先修正搭接')
  const barLabels = meta.barLabels !== false // 条尾标注缺省开启（上报件口径）
  const cal = project.calendar
  const norm = normalizeCalendar(cal)
  const startMs = new Date(`${project.startDate}T00:00:00Z`).getTime()
  const dayNo = (wd: number): number =>
    Math.round((wdToDate(project.startDate, wd, cal).getTime() - startMs) / 86400000)
  const deadlineOff = project.deadline
    ? Math.round((new Date(`${project.deadline}T00:00:00Z`).getTime() - startMs) / 86400000)
    : -1
  const days = Math.max(dayNo(Math.max(cpm.duration, 7)) + 3, 21, deadlineOff + 2)
  const chartW = days * DAY_W
  const gridW = TABLE_W + chartW
  const totalW = gridW + M * 2
  const dataH = project.tasks.length * ROW_H
  const gridH = HEAD_H + dataH
  const totalH = TITLE_H + gridH + FOOT_H

  // 双行时标（自然日列）：底纹先画、月/日文本后画不被盖；月行 + 日行
  const colDates = Array.from({ length: days }, (_, i) => {
    const d = new Date(startMs + i * 86400000)
    return { d, offWork: !isWorkingDate(d, norm) }
  })
  let scale = ''
  for (let i = 0; i < days; i++) {
    if (colDates[i].offWork) {
      scale += `<rect x="${i * DAY_W}" y="0" width="${DAY_W}" height="${gridH}" fill="${C.weekend}"/>`
    }
  }
  for (let i = 0; i < days; i++) {
    const c = colDates[i]
    const month = c.d.getUTCMonth() + 1
    if (c.d.getUTCDate() === 1 || i === 0) {
      scale += `<text x="${i * DAY_W + 3}" y="15" font-family="${FONT}" font-size="11" fill="${C.ink}">${month}月</text>`
    }
    scale += `<text x="${i * DAY_W + DAY_W / 2}" y="36" text-anchor="middle" font-family="${FONT}" font-size="10" fill="${c.offWork ? C.dim : C.ink}">${c.d.getUTCDate()}</text>`
  }
  // 日列分隔（浅）
  for (let i = 1; i < days; i++) {
    scale += `<line x1="${i * DAY_W}" y1="0" x2="${i * DAY_W}" y2="${gridH}" stroke="${C.grid}" stroke-width="0.5"/>`
  }

  // 表格：表头 + 行文本 + 网格
  const colX: number[] = []
  {
    let x = 0
    for (const c of EXP_COLS) { colX.push(x); x += c.w }
  }
  const fmtDate = (wd: number): string => {
    const d = wdToDate(project.startDate, wd, cal)
    return `${d.getUTCFullYear()}/${d.getUTCMonth() + 1}/${d.getUTCDate()}`
  }
  const noOf = (id: string): number => project.tasks.findIndex((t) => t.id === id) + 1
  let table = `<rect x="0" y="0" width="${TABLE_W}" height="${gridH}" fill="${C.headBg}"/>`
  for (let i = 0; i < EXP_COLS.length; i++) {
    const c = EXP_COLS[i]
    table += `<text x="${colX[i] + c.w / 2}" y="27" text-anchor="middle" font-family="${FONT}" font-size="12" font-weight="600" fill="${C.ink}">${c.label}</text>`
  }
  project.tasks.forEach((t, i) => {
    const row = cpm.rows[t.id]
    const group = isGroupRow(project.tasks, i)
    const span = group ? groupSpan(project, cpm, i) : null
    const yTop = HEAD_H + i * ROW_H
    const yMid = yTop + ROW_H / 2 + 4
    if (group) table += `<rect x="0" y="${yTop}" width="${TABLE_W}" height="${ROW_H}" fill="${C.groupBand}"/>`
    const indent = t.level * 12
    const cells: { key: string; text: string; anchor: 'start' | 'middle'; bold?: boolean; dim?: boolean }[] = [
      { key: 'no', text: `${i + 1}`, anchor: 'middle' },
      { key: 'name', text: fitText(group ? `▲ ${t.name}` : t.name, EXP_COLS[1].w - 8 - indent, 11), anchor: 'start', bold: group },
      { key: 'dur', text: group ? '汇总' : t.isMilestone ? '0' : `${Math.max(0, Math.round(t.duration))}`, anchor: 'middle', dim: group },
      {
        key: 'start', anchor: 'middle', dim: true,
        text: (group ? span : row) ? fmtDate(group ? span!.es : row!.es) : '',
      },
      {
        key: 'finish', anchor: 'middle', dim: true,
        text: (group ? span : row) ? fmtDate(group ? span!.ef : row!.ef) : '',
      },
      {
        key: 'preds', anchor: 'middle', dim: true,
        text: group ? '' : project.links.filter((l) => l.to === t.id)
          .map((l) => `${noOf(l.from)}${l.type}${l.lag !== 0 ? (l.lag > 0 ? '+' : '') + l.lag : ''}`)
          .join(','),
      },
    ]
    for (const cell of cells) {
      const ci = EXP_COLS.findIndex((c) => c.key === cell.key)
      if (!cell.text || ci < 0) continue
      const c = EXP_COLS[ci]
      const tx = cell.anchor === 'middle' ? colX[ci] + c.w / 2 : colX[ci] + 6 + indent
      table += `<text x="${tx}" y="${yMid}" text-anchor="${cell.anchor}" font-family="${FONT}" font-size="11"${cell.bold ? ' font-weight="600"' : ''} fill="${cell.dim ? C.dim : C.ink}">${esc(cell.text)}</text>`
    }
  })
  // 表格网格线：列分界 + 行分界
  for (let i = 1; i < EXP_COLS.length; i++) {
    table += `<line x1="${colX[i]}" y1="0" x2="${colX[i]}" y2="${gridH}" stroke="${C.grid}" stroke-width="0.5"/>`
  }
  for (let i = 1; i <= project.tasks.length; i++) {
    table += `<line x1="0" y1="${HEAD_H + i * ROW_H}" x2="${TABLE_W}" y2="${HEAD_H + i * ROW_H}" stroke="${C.grid}" stroke-width="0.5"/>`
  }
  table += `<line x1="0" y1="${HEAD_H}" x2="${TABLE_W}" y2="${HEAD_H}" stroke="${C.border}"/>`

  // 图面组：条形/汇总/里程碑/时差尾/依赖线/竣工线/今日线
  const bar = (t: (typeof project.tasks)[number], i: number): string => {
    const row = cpm.rows[t.id]
    if (!row) return ''
    const group = isGroupRow(project.tasks, i)
    const manual = t.mode === 'manual'
    const dur = t.isMilestone ? 0 : Math.max(0, Math.round(t.duration))
    const yTop = HEAD_H + i * ROW_H
    let out = ''
    if (group) {
      const span = groupSpan(project, cpm, i)
      if (span) {
        out += `<rect class="sched-exp-summary" x="${dayNo(span.es) * DAY_W}" y="${yTop + ROW_H / 2 - 2.5}" width="${Math.max((dayNo(span.ef) - dayNo(span.es)) * DAY_W, 8)}" height="5" fill="${C.ink}"/>`
      }
      return out
    }
    if (t.isMilestone) {
      const cx = dayNo(row.es) * DAY_W
      const cy = yTop + ROW_H / 2
      out += `<polygon class="sched-exp-mile" points="${cx - 6.5},${cy} ${cx},${cy - 6.5} ${cx + 6.5},${cy} ${cx},${cy + 6.5}" fill="${C.ink}"/>`
      // 里程碑旗标（上报口径恒带）：菱形右侧一面小旗（旗杆高 14，关键红）
      out += expFlagSvg(cx + 12, cy - 7, 14, 8, 7, C.critical)
      return out
    }
    const crit = !manual && row.critical
    const bx = dayNo(row.es) * DAY_W
    const bw = Math.max((dayNo(row.es + dur) - dayNo(row.es)) * DAY_W, 6)
    out += `<rect class="sched-exp-bar${crit ? ' sched-exp-bar-crit' : ''}${manual ? ' sched-exp-bar-manual' : ''}" x="${bx}" y="${yTop + (ROW_H - 12) / 2}" width="${bw}" height="12" fill="${crit ? C.critical : manual ? C.barManual : C.bar}" rx="2"/>`
    if (!manual && row.tf > 0) {
      out += `<rect class="sched-exp-float" x="${dayNo(row.ef) * DAY_W}" y="${yTop + (ROW_H - 6) / 2}" width="${row.tf * DAY_W}" height="6" fill="${C.float}"/>`
    }
    // 条尾标注（缺省开启，meta.barLabels=false 关闭）：叶任务条末端右侧 +6px
    // 标「任务名（N天）」，字号同表格小字口径（11px）、墨色深字与条面对比可读；
    // 超出图面右界时改画条内右端白字——保持简单：只判右边界，宁可 fitText 截断。
    if (barLabels) {
      const label = `${t.name}（${dur}天）`
      const outX = bx + bw + 6
      const labelY = yTop + ROW_H / 2 + 4
      if (outX + textWidth(label, 11) <= days * DAY_W) {
        out += `<text class="sched-exp-bar-label" x="${outX}" y="${labelY}" font-family="${FONT}" font-size="11" fill="${C.ink}">${esc(label)}</text>`
      } else {
        out += `<text class="sched-exp-bar-label" x="${bx + bw - 6}" y="${labelY}" text-anchor="end" font-family="${FONT}" font-size="11" fill="white">${esc(fitText(label, Math.max(bw - 12, 1), 11))}</text>`
      }
    }
    return out
  }
  const rowIdx = new Map(project.tasks.map((t, i) => [t.id, i]))
  let links = ''
  for (const l of project.links) {
    const fi = rowIdx.get(l.from)
    const ti = rowIdx.get(l.to)
    const f = cpm.rows[l.from]
    const t2 = cpm.rows[l.to]
    if (fi === undefined || ti === undefined || !f || !t2) continue
    const d = ganttLinkPath(l.type, {
      x1s: dayNo(f.es) * DAY_W,
      x1f: dayNo(f.ef) * DAY_W,
      y1: HEAD_H + fi * ROW_H + ROW_H / 2,
      x2s: dayNo(t2.es) * DAY_W,
      x2f: dayNo(t2.ef) * DAY_W,
      y2: HEAD_H + ti * ROW_H + ROW_H / 2,
    })
    links += `<path class="sched-exp-link" d="${d}" fill="none" stroke="${C.link}" stroke-width="1" marker-end="url(#exp-arrow)"/>`
  }
  let overlay = ''
  const todayOff = Math.floor((Date.now() - startMs) / 86400000)
  if (todayOff >= 0 && todayOff <= days) {
    overlay += `<line class="sched-exp-today" x1="${todayOff * DAY_W}" y1="0" x2="${todayOff * DAY_W}" y2="${gridH}" stroke="${C.today}" stroke-width="1"/>`
  }
  if (deadlineOff >= 0 && deadlineOff <= days) {
    overlay += `<line class="sched-exp-deadline" x1="${deadlineOff * DAY_W + DAY_W}" y1="0" x2="${deadlineOff * DAY_W + DAY_W}" y2="${gridH}" stroke="${C.deadline}" stroke-width="1.2" stroke-dasharray="5 4"/>`
  }

  // 标题带（共用 helper，缺省图名按图种给）
  const head = exportTitleSvg(meta, totalW, `${project.name || '进度计划'}\u3000施工进度计划横道图`)

  const svg =
    `<svg xmlns="http://www.w3.org/2000/svg" width="${totalW}" height="${totalH}" viewBox="0 0 ${totalW} ${totalH}" class="sched-exp-gantt">` +
    `<rect width="${totalW}" height="${totalH}" fill="white"/>` +
    head +
    `<g transform="translate(${M + TABLE_W},${TITLE_H})">` +
    `<defs><marker id="exp-arrow" markerWidth="7" markerHeight="7" refX="6" refY="3.5" orient="auto"><path d="M0,0 L7,3.5 L0,7 z" fill="${C.link}"/></marker></defs>` +
    scale + links +
    project.tasks.map((t, i) => bar(t, i)).join('') +
    overlay +
    `</g>` +
    `<g transform="translate(${M},${TITLE_H})">` + table + `</g>` +
    `<rect x="${M}" y="${TITLE_H}" width="${gridW}" height="${gridH}" fill="none" stroke="${C.border}"/>` +
    footSvg(meta, TITLE_H + gridH, totalW) +
    `</svg>`
  return { svg, w: totalW, h: totalH }
}
