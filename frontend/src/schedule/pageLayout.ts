/**
 * pageLayout.ts — 按纸张布图几何（W6「输出与外观」第一刀，2026-11-14）
 *
 * 把「单张超大 SVG 上报图」装订成「按 A4/A3/A2 纸张分页的可打印件」所需的
 * 全部纯几何：纸张像素盒、扣边距内容盒、等比缩放、分页网格与每页内容偏移、
 * 单页页框（外框 + 内框 + 页号 + 图幅标记 + 页脚线）。
 *
 * 零 React / 零 DOM / 零 import：mm→px 与分页算术可直接在 node 下单测；
 * XML 转义自带一份（五字符替换，与 ganttExport.esc 同口径），避免几何模块
 * 被导出门面拖进数据层依赖。
 *
 * 口径（与纸面工程图装订一致）：
 *  - mm→px 按 dpi（缺省 96 = CSS px），逐项四舍五入；
 *  - 缩放 = clamp(fitScale, minScale, 1)：不放大（>1 钳 1），也不无限缩小
 *    ——缩到 minScale 就停，余量交给分页。纯 fit 会让任何大图都收成 1 页，
 *    「按纸张布图」就退化成「按纸张压扁」，故设保可读下限；
 *  - 页序 = 先行后列、左上起（row-major），index 从 1 起；
 *  - 每页 dx/dy 是「缩放后内容坐标系」里该页窗口的左上角，w/h 是窗口尺寸
 *    （末列/末行按余量取窄，永不为 0）；
 *  - 页框配色缺省纯黑（打印件黑白语义），字号/描边可参数覆盖。
 */

/** 纸张规格（A 系列；mm 为「短边 × 长边」，纵/横向由 orientation 决定） */
export type PaperKind = 'A4' | 'A3' | 'A2'

/** 纸张方向（portrait=竖，landscape=横） */
export type PageOrientation = 'portrait' | 'landscape'

/** 纸张像素盒尺寸 */
export interface Box {
  w: number
  h: number
}

/**
 * 纸张布图参数（一张纸怎么摆）：
 *  - paper：A4 / A3 / A2；
 *  - orientation：竖 / 横；
 *  - marginMm：四边页边距（mm，负值按 0 收）；
 *  - dpi：缺省 96（CSS px 口径），非法/非正回落 96。
 */
export interface PaperLayout {
  paper: PaperKind
  orientation: PageOrientation
  marginMm: number
  dpi?: number
}

/** 单页窗口（分页结果的一项）：dx/dy 为缩放后内容坐标系的窗口左上角 */
export interface PageTile {
  /** 页号，从 1 起（= 装订顺序） */
  index: number
  /** 行号，从 0 起（自上而下） */
  row: number
  /** 列号，从 0 起（自左而右） */
  col: number
  /** 窗口左上角 x（缩放后内容坐标系） */
  dx: number
  /** 窗口左上角 y（缩放后内容坐标系） */
  dy: number
  /** 窗口宽（≤ 内容盒宽，末列取余量） */
  w: number
  /** 窗口高（≤ 内容盒高，末行取余量） */
  h: number
}

/** 分页结果 */
export interface PageTiling {
  /** 页序表（row-major）；内容尺寸非法时为空数组 */
  pages: PageTile[]
  /** 列数（内容尺寸非法时 0） */
  cols: number
  /** 行数（内容尺寸非法时 0） */
  rows: number
  /** 实际采用的内容缩放（0 < scale ≤ 1） */
  scale: number
  /** 缩放后内容宽（= 内容宽 × scale） */
  contentW: number
  /** 缩放后内容高（= 内容高 × scale） */
  contentH: number
  /** 每页可布图内容盒宽 */
  boxW: number
  /** 每页可布图内容盒高 */
  boxH: number
  /** 实际采用的缩放下限 */
  minScale: number
  /** 网格页数超出 maxPages 上限被截断 */
  truncated: boolean
}

/** 分页选项 */
export interface TileOptions {
  /** 缩放下限（缺省 DEFAULT_MIN_SCALE=0.5，钳进 [0.01, 1]） */
  minScale?: number
  /** 页数硬顶（缺省 TILE_MAX_PAGES=400，超出按页序截断并标 truncated） */
  maxPages?: number
}

/** 页框样式（可覆盖；缺省黑白打印语义） */
export interface FrameStyle {
  /** 线框色，缺省锚定墨色（black） */
  stroke?: string
  /** 文字色，缺省同 stroke */
  ink?: string
  /** 外框描边宽，缺省 1 */
  strokeWidth?: number
  /** 字号，缺省 10 */
  fontSize?: number
  /** 字族，缺省 FRAME_FONT */
  fontFamily?: string
  /** 是否画内框（内容盒边框），缺省 true */
  inner?: boolean
  /** 是否画页脚（页脚线 + 图幅标记 + 页号），缺省 true */
  footer?: boolean
}

/** A 系列纸张 mm 尺寸（短边, 长边） */
export const PAPER_MM: Record<PaperKind, Box> = {
  A4: { w: 210, h: 297 },
  A3: { w: 297, h: 420 },
  A2: { w: 420, h: 594 },
}

/** 可选纸张（UI 下拉顺序） */
export const PAPER_KINDS: PaperKind[] = ['A4', 'A3', 'A2']

/** 1 英寸 = 25.4mm */
export const MM_PER_INCH = 25.4

/** 缺省 dpi（96 = CSS px，与 exportArtifact 的 px 口径一致） */
export const DEFAULT_DPI = 96

/** 缺省页边距（mm）：10mm ≈ 38px，够放页脚线 + 角标文字 */
export const DEFAULT_MARGIN_MM = 10

/** 缩放下限：最多缩到 1/2，再小就靠分页，保图面可读 */
export const DEFAULT_MIN_SCALE = 0.5

/** 分页页数硬顶（纸堆上限，防一次性堆出万页字符串） */
export const TILE_MAX_PAGES = 400

/** 方向文字（图幅标记用：A4 横 / A3 竖） */
export const ORIENT_TEXT: Record<PageOrientation, string> = { portrait: '竖', landscape: '横' }

/** 页框缺省字族（与导出件同一套系统字族，不引主题令牌：打印件独立于主题） */
export const FRAME_FONT = "system-ui, 'Microsoft YaHei', sans-serif"

/** 页框缺省墨色（打印件黑白语义；CSS 色名而非 hex，避免 raw-hex 门禁） */
export const FRAME_INK = 'black'

/** 有限数校验（非法/非有限回落 fallback） */
function finite(v: unknown, fallback: number): number {
  return typeof v === 'number' && Number.isFinite(v) ? v : fallback
}

/** 正数校验（非正/非法一律 0，供尺寸入参防御） */
function positive(v: unknown): number {
  const n = finite(v, 0)
  return n > 0 ? n : 0
}

/** 纸张兜底（未知 kind 按 A4） */
function paperOf(layout: PaperLayout | undefined): PaperKind {
  const p = layout?.paper
  return p === 'A4' || p === 'A3' || p === 'A2' ? p : 'A4'
}

/** 方向兜底（未知按竖） */
function orientOf(layout: PaperLayout | undefined): PageOrientation {
  return layout?.orientation === 'landscape' ? 'landscape' : 'portrait'
}

/** dpi 兜底（非正/非法回落 DEFAULT_DPI） */
function dpiOf(layout: PaperLayout | undefined): number {
  const d = finite(layout?.dpi, DEFAULT_DPI)
  return d > 0 ? d : DEFAULT_DPI
}

/** 页边距 mm 兜底（非有限回落缺省，负值按 0） */
function marginMmOf(layout: PaperLayout | undefined): number {
  return Math.max(0, finite(layout?.marginMm, DEFAULT_MARGIN_MM))
}

/** XML 文本/属性转义（与 ganttExport.esc 同口径：& < > " 四字符；单引号换 &#39;） */
export function escapeXml(s: string): string {
  return String(s)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

/** SVG 数值输出（两位小数，避免 0.30000000000000004 之类噪声） */
export function svgNum(v: number): string {
  const n = finite(v, 0)
  return String(Math.round(n * 100) / 100)
}

/** mm → px（按 dpi 四舍五入；dpi 非法回落 96） */
export function mmToPx(mm: number, dpi: number = DEFAULT_DPI): number {
  const d = finite(dpi, DEFAULT_DPI)
  const dpiSafe = d > 0 ? d : DEFAULT_DPI
  const px = (finite(mm, 0) / MM_PER_INCH) * dpiSafe
  return Number.isFinite(px) ? Math.round(px) : 0
}

/** 页面像素盒（含方向交换；结果 ≥1px） */
export function pageBoxPx(layout: PaperLayout): Box {
  const mm = PAPER_MM[paperOf(layout)]
  const dpi = dpiOf(layout)
  const short = Math.max(1, mmToPx(mm.w, dpi))
  const long = Math.max(1, mmToPx(mm.h, dpi))
  return orientOf(layout) === 'landscape' ? { w: long, h: short } : { w: short, h: long }
}

/** 页边距像素（= 内容盒四边内缩量，负边距按 0） */
export function marginPx(layout: PaperLayout): number {
  return Math.max(0, mmToPx(marginMmOf(layout), dpiOf(layout)))
}

/** 内容盒像素（页面盒扣四边页边距；边距过大时钳到 ≥1px） */
export function contentBoxPx(layout: PaperLayout): Box {
  const page = pageBoxPx(layout)
  const m = marginPx(layout)
  return { w: Math.max(1, page.w - m * 2), h: Math.max(1, page.h - m * 2) }
}

/**
 * 等比缩放系数：把 content 装进 box（不放大——>1 一律钳到 1）。
 * 任一尺寸非法（0/负/非有限）返回 1（中性，不缩不放大）。
 */
export function fitScale(content: Box, box: Box): number {
  const cw = positive(content?.w)
  const ch = positive(content?.h)
  const bw = positive(box?.w)
  const bh = positive(box?.h)
  if (cw <= 0 || ch <= 0 || bw <= 0 || bh <= 0) return 1
  const s = Math.min(bw / cw, bh / ch)
  if (!Number.isFinite(s) || s <= 0) return 1
  return Math.min(1, s)
}

/**
 * 按纸张分页：先定缩放（clamp(fit, minScale, 1)），再按缩放后内容尺寸算网格。
 * 页序 row-major（先行后列、左上起），index 1 起；末列/末行窗口取余量。
 * 内容尺寸非法（0/负/非有限）→ pages=[]、cols=rows=0、scale=1，不抛。
 */
export function tilePages(content: Box, layout: PaperLayout, opts: TileOptions = {}): PageTiling {
  const box = contentBoxPx(layout)
  const cw = positive(content?.w)
  const ch = positive(content?.h)
  const maxPages = Math.max(1, Math.floor(finite(opts.maxPages, TILE_MAX_PAGES)) || TILE_MAX_PAGES)
  const minScale = Math.min(1, Math.max(0.01, finite(opts.minScale, DEFAULT_MIN_SCALE)))
  const empty: PageTiling = {
    pages: [], cols: 0, rows: 0, scale: 1, contentW: 0, contentH: 0,
    boxW: box.w, boxH: box.h, minScale, truncated: false,
  }
  if (cw <= 0 || ch <= 0) return empty

  let scale = fitScale({ w: cw, h: ch }, box)
  if (scale < minScale) scale = minScale // 保可读下限：不再缩，余量交分页
  const contentW = cw * scale
  const contentH = ch * scale
  // -1e-9：吸收浮点误差，正好的整数倍不要多长出一页空白
  const cols = Math.max(1, Math.ceil(contentW / box.w - 1e-9))
  const rows = Math.max(1, Math.ceil(contentH / box.h - 1e-9))
  const total = cols * rows
  const limit = Math.min(total, maxPages)
  const pages: PageTile[] = []
  for (let i = 0; i < limit; i++) {
    const row = Math.floor(i / cols)
    const col = i - row * cols
    const dx = col * box.w
    const dy = row * box.h
    pages.push({
      index: i + 1,
      row,
      col,
      dx,
      dy,
      w: Math.max(1, Math.min(box.w, contentW - dx)),
      h: Math.max(1, Math.min(box.h, contentH - dy)),
    })
  }
  return {
    pages, cols, rows, scale, contentW, contentH,
    boxW: box.w, boxH: box.h, minScale, truncated: limit < total,
  }
}

/** 图幅标记文字（如「A4 横」「A3 竖」） */
export function paperMark(layout: PaperLayout): string {
  return `${paperOf(layout)} ${ORIENT_TEXT[orientOf(layout)]}`
}

/**
 * 单页页框 SVG（页局部坐标系：0,0 起，尺寸 = pageBoxPx）：
 * 外框贴纸边、内框贴内容盒边界、页脚线在底边距带内、左下 图幅标记、
 * 右下「第 n/N 页」。纯字符串；页码/总页数逐项钳（page<1→1，page>total→total）。
 * style.inner=false 不出内框，style.footer=false 不出页脚三件。
 */
export function frameSvg(layout: PaperLayout, page: number, total: number, style: FrameStyle = {}): string {
  const { w: pw, h: ph } = pageBoxPx(layout)
  const m = marginPx(layout)
  const stroke = style.stroke ?? FRAME_INK
  const ink = style.ink ?? stroke
  const sw = Math.max(0.2, finite(style.strokeWidth, 1))
  const thin = Math.max(0.2, sw * 0.6)
  const fs = Math.max(4, finite(style.fontSize, 10))
  const font = style.fontFamily ?? FRAME_FONT
  const totalSafe = Math.max(1, Math.floor(finite(total, 1)))
  const pageSafe = Math.min(totalSafe, Math.max(1, Math.floor(finite(page, 1))))
  const half = sw / 2
  const innerX = Math.max(half, m)
  const innerW = Math.max(1, pw - innerX * 2)
  const innerH = Math.max(1, ph - innerX * 2)

  let out =
    `<g class="sched-paper-frame" data-paper="${paperOf(layout)}" data-orient="${orientOf(layout)}"` +
    ` data-page="${pageSafe}" data-total="${totalSafe}">`
  // 外框：贴纸边（描边内缩半个线宽，墨线落在纸内）
  out += `<rect class="sched-paper-frame-outer" x="${svgNum(half)}" y="${svgNum(half)}" width="${svgNum(Math.max(1, pw - sw))}" height="${svgNum(Math.max(1, ph - sw))}" fill="none" stroke="${stroke}" stroke-width="${svgNum(sw)}"/>`
  // 内框：内容盒边界（布图区与页脚带的分界）
  if (style.inner !== false) {
    out += `<rect class="sched-paper-frame-inner" x="${svgNum(innerX)}" y="${svgNum(innerX)}" width="${svgNum(innerW)}" height="${svgNum(innerH)}" fill="none" stroke="${stroke}" stroke-width="${svgNum(thin)}"/>`
  }
  if (style.footer !== false) {
    const baseY = ph - 4 // 角标基线：离纸边 4px，落在底边距带内
    const lineY = baseY - fs - 4 // 页脚线：角标上方一行
    const pad = Math.max(3, m / 2)
    // 底边距放得下「页脚线 + 一行字」才画线，否则只留角标（窄边距不压内容）
    if (m >= fs + 8) {
      out += `<line class="sched-paper-frame-rule" x1="${svgNum(pad)}" y1="${svgNum(lineY)}" x2="${svgNum(pw - pad)}" y2="${svgNum(lineY)}" stroke="${stroke}" stroke-width="${svgNum(thin)}"/>`
    }
    out += `<text class="sched-paper-frame-mark" x="${svgNum(pad)}" y="${svgNum(baseY)}" font-family="${font}" font-size="${svgNum(fs)}" fill="${ink}">${escapeXml(paperMark(layout))}</text>`
    out += `<text class="sched-paper-frame-no" x="${svgNum(pw - pad)}" y="${svgNum(baseY)}" text-anchor="end" font-family="${font}" font-size="${svgNum(fs)}" fill="${ink}">${escapeXml(`第 ${pageSafe}/${totalSafe} 页`)}</text>`
  }
  out += `</g>`
  return out
}
