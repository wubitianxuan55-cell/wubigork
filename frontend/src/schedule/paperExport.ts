/**
 * paperExport.ts — 按纸张布图（W6「输出与外观」第一刀，2026-11-14）
 *
 * 输入：任一上报图构建器的产物（ganttExport.buildGanttExportSvg /
 * networkExport.buildAoaExportSvg / buildPdmExportSvg —— 同形状
 * `{ svg, w, h }`，自身带 width/height/viewBox）。
 * 输出：把这张超大 SVG 按 A4/A3/A2 纸张切页装订成的**单条多页 SVG 字符串**
 * （多页纵向排列、页间装订缝缺省 12px；打印路径传 `gap:0` 让整叠高恰为
 * `pages × 页高`，配合 `@page size` 由浏览器真分页），可直接交给
 * exportArtifact.printSvg / svgToPdfBlob / svgToPngBlob，不新增 DOM 管线。
 *
 * 摆放口径（几何全部来自 pageLayout，本模块只管拼字符串）：
 *  1. 构建器产物原样作为**嵌套 `<svg>`**（保留其 viewBox，width/height 改写为
 *     「缩放后内容尺寸」），省掉对图面内部坐标的任何逐元素变换；
 *  2. 每页一个页局部坐标组 `translate(0, 页顶)`；页内再用 `translate(边距, 边距)`
 *     + `clip-path` 裁到内容盒；嵌套 svg 外层再 `translate(-dx, -dy)` 把该页窗口
 *     移到原点——即「页面 = 缩放后内容的一个取景窗」；
 *  3. 每页叠 `pageLayout.frameSvg`（外框 + 内框 + 图幅标记 + 页码），页框在
 *     内容组之前画，图面裁在内容盒内，二者不重叠；
 *  4. `opts.page=false` 只出图面取景窗（不出页框），用于屏内预览对位；
 *     `opts.title` 只写 `<title>` 元数据（图名/图签本就在构建器产物里，不重复
 *     画到纸面，避免与图面标题带打架）。
 *
 * 防御（一律不抛）：art 缺失 / svg 非字符串 / 尺寸 0 或负 / 无 `<svg>` 标签 →
 * 出一张带页框的占位页（pages=0，w/h 为单页纸盒）；页数超 MAX_PAGES=400 →
 * 按页序截断并在末页底边距标注，回填 `truncated:true`。
 */
import {
  contentBoxPx, escapeXml, frameSvg, marginPx, pageBoxPx, svgNum, tilePages,
  TILE_MAX_PAGES, type PageTiling, type PaperLayout,
} from './pageLayout'

/** 构建器产物（横道/网络图同形状） */
export interface PaperArtifact {
  svg: string
  w: number
  h: number
}

/** 布图选项 */
export interface PaperExportOptions {
  /** 是否叠页框（页号/图幅标记/内外框），缺省 true */
  page?: boolean
  /** 纸堆图名：只进 `<title>` 元数据（打印/另存窗口标题），不上纸面 */
  title?: string
  /**
   * 页间装订缝（px），缺省 PAGE_GAP=12。
   * 打印路径传 `0`：整叠纸高恰为 `pages × 页高`，浏览器按 `@page size`
   * 逐页换页时不会因累计缝偏移逐页错位（屏内预览仍用 12 看得见纸缝）。
   * 非有限/负数回落 PAGE_GAP，上限 GAP_MAX=64（防拼出天量留白）。
   */
  gap?: number
}

/** 布图产物 */
export interface PaperExportSvg {
  svg: string
  /** 实际出页数（art 非法时 0） */
  pages: number
  /** 纸堆宽（像素，= 单页纸宽） */
  w: number
  /** 纸堆高（像素，= pages×页高 + (pages-1)×装订缝） */
  h: number
  /** 页数超上限被截断 */
  truncated: boolean
  /** 实际采用的内容缩放 */
  scale: number
}

/** 页数硬顶（与 pageLayout.TILE_MAX_PAGES 同值，导出面独立常量名供 UI 引用） */
export const MAX_PAGES = TILE_MAX_PAGES

/** 页间装订缝（px）：打印裁切/打孔余量，避免相邻页框线粘连 */
export const PAGE_GAP = 12

/** 装订缝上限（px）：再大就不是装订缝而是拼版留白了 */
export const GAP_MAX = 64

/** 装订缝收敛：非有限/负数回落 PAGE_GAP，上限 GAP_MAX（0 合法=打印无缝） */
function resolveGap(v: unknown): number {
  if (typeof v !== 'number' || !Number.isFinite(v) || v < 0) return PAGE_GAP
  return Math.min(GAP_MAX, v)
}

/** 内容裁剪 clipPath 的 id（纸堆内唯一一份，各页共用） */
export const PAPER_CLIP_ID = 'sched-paper-clip'

/** 占位页提示文字（art 非法时） */
export const PAPER_EMPTY_NOTE = '（无图面内容）'

/** 正数收敛（非正/非法 → 0，供 w/h 入参防御） */
function posNum(v: unknown): number {
  return typeof v === 'number' && Number.isFinite(v) && v > 0 ? v : 0
}

/** 首标签定位（构建器产物的 `<svg ...>` 开标签） */
function firstSvgTag(svg: string): string | undefined {
  return /<svg\b[^>]*>/i.exec(svg)?.[0]
}

/**
 * 把构建器产物改写成可嵌套的 `<svg>`：
 *  - 去掉自带 width/height（视口尺寸由本模块按缩放给定）；
 *  - 保留（缺则补）viewBox —— 这是内容坐标到视口的唯一映射，缩放由它承担；
 *  - 补 xmlns（嵌套节点不缺，但独立打开/被 DOMParser 解析时要有）；
 *  - 加 x/y=0 与 preserveAspectRatio（视口与 viewBox 同比例，结果等比不裁）。
 */
function nestArtifact(artSvg: string, openTag: string, viewW: number, viewH: number, vpW: number, vpH: number): string {
  let tag = openTag
    .replace(/\s(?:width|height)\s*=\s*"[^"]*"/gi, '')
    .replace(/\s(?:width|height)\s*=\s*'[^']*'/gi, '')
  if (!/\sviewBox\s*=/i.test(tag)) tag = tag.replace(/^<svg\b/i, `<svg viewBox="0 0 ${svgNum(vpW)} ${svgNum(vpH)}"`)
  if (!/\sxmlns\s*=/i.test(tag)) tag = tag.replace(/^<svg\b/i, '<svg xmlns="http://www.w3.org/2000/svg"')
  tag = tag.replace(/^<svg\b/i, `<svg x="0" y="0" width="${svgNum(viewW)}" height="${svgNum(viewH)}" preserveAspectRatio="xMinYMin meet"`)
  return artSvg.replace(openTag, tag)
}

/** 占位页：art 非法时出一张带页框的空页（pages=0，纸盒仍是真实纸张尺寸） */
function emptySheet(layout: PaperLayout, withFrame: boolean, title: string, scale: number): PaperExportSvg {
  const page = pageBoxPx(layout)
  let body = `<rect class="sched-paper-sheet-bg" x="0" y="0" width="${svgNum(page.w)}" height="${svgNum(page.h)}" fill="white"/>`
  if (withFrame) body += frameSvg(layout, 1, 1)
  body += `<text class="sched-paper-empty" x="${svgNum(page.w / 2)}" y="${svgNum(page.h / 2)}" text-anchor="middle" font-family="system-ui, 'Microsoft YaHei', sans-serif" font-size="14" fill="gray">${escapeXml(PAPER_EMPTY_NOTE)}</text>`
  const head = `<svg xmlns="http://www.w3.org/2000/svg" width="${svgNum(page.w)}" height="${svgNum(page.h)}" viewBox="0 0 ${svgNum(page.w)} ${svgNum(page.h)}" class="sched-paper-stack">`
  const titleEl = title ? `<title>${escapeXml(title)}</title>` : ''
  return { svg: `${head}${titleEl}${body}</svg>`, pages: 0, w: page.w, h: page.h, truncated: false, scale }
}

/**
 * 构建「按纸张布图」的多页 SVG 字符串。
 * @param art    构建器产物（{svg,w,h}）
 * @param layout 纸张/方向/边距/dpi
 * @param opts   page=是否出页框（缺省 true）、title=纸堆图名（元数据）、
 *               gap=页间装订缝（缺省 12；打印传 0 让浏览器按纸张真分页）
 */
export function buildPaperSvg(art: PaperArtifact, layout: PaperLayout, opts: PaperExportOptions = {}): PaperExportSvg {
  const title = typeof opts.title === 'string' ? opts.title.trim() : ''
  const withFrame = opts.page !== false
  const gap = resolveGap(opts.gap)
  const artSvg = typeof art?.svg === 'string' ? art.svg : ''
  const openTag = artSvg ? firstSvgTag(artSvg) : undefined
  const artW = posNum(art?.w)
  const artH = posNum(art?.h)
  if (!openTag || artW <= 0 || artH <= 0) return emptySheet(layout, withFrame, title, 1)

  const tiling: PageTiling = tilePages({ w: artW, h: artH }, layout, { maxPages: MAX_PAGES })
  if (tiling.pages.length === 0) return emptySheet(layout, withFrame, title, tiling.scale)

  const page = pageBoxPx(layout)
  const box = contentBoxPx(layout)
  const m = marginPx(layout)
  const total = tiling.pages.length
  const totalW = page.w
  const totalH = total * page.h + (total - 1) * gap
  const nested = nestArtifact(artSvg, openTag, tiling.contentW, tiling.contentH, artW, artH)
  const noteShown = tiling.truncated && m >= 14

  let body = `<rect class="sched-paper-stack-bg" x="0" y="0" width="${svgNum(totalW)}" height="${svgNum(totalH)}" fill="white"/>`
  body += `<defs><clipPath id="${PAPER_CLIP_ID}" clipPathUnits="userSpaceOnUse">` +
    `<rect x="0" y="0" width="${svgNum(box.w)}" height="${svgNum(box.h)}"/>` +
    `</clipPath></defs>`
  for (let i = 0; i < total; i++) {
    const tile = tiling.pages[i]
    const top = i * (page.h + gap)
    body += `<g class="sched-paper-page" data-page="${tile.index}" data-total="${total}" transform="translate(0 ${svgNum(top)})">`
    body += `<rect class="sched-paper-sheet-bg" x="0" y="0" width="${svgNum(page.w)}" height="${svgNum(page.h)}" fill="white"/>`
    if (withFrame) body += frameSvg(layout, tile.index, total)
    body += `<g class="sched-paper-slot" data-dx="${svgNum(tile.dx)}" data-dy="${svgNum(tile.dy)}"` +
      ` data-w="${svgNum(tile.w)}" data-h="${svgNum(tile.h)}" clip-path="url(#${PAPER_CLIP_ID})"` +
      ` transform="translate(${svgNum(m)} ${svgNum(m)})">`
    body += `<g class="sched-paper-window" transform="translate(${svgNum(-tile.dx)} ${svgNum(-tile.dy)})">${nested}</g>`
    body += `</g>`
    // 截断注记：落在末页底边距带中部，与左右角标不重叠
    if (noteShown && i === total - 1) {
      body += `<text class="sched-paper-cut-note" x="${svgNum(page.w / 2)}" y="${svgNum(page.h - 4)}" text-anchor="middle"` +
        ` font-family="system-ui, 'Microsoft YaHei', sans-serif" font-size="9" fill="gray">` +
        `${escapeXml(`图面超出 ${MAX_PAGES} 页上限，已截断（共需 ${tiling.cols * tiling.rows} 页）`)}</text>`
    }
    body += `</g>`
  }

  const head = `<svg xmlns="http://www.w3.org/2000/svg" width="${svgNum(totalW)}" height="${svgNum(totalH)}"` +
    ` viewBox="0 0 ${svgNum(totalW)} ${svgNum(totalH)}" class="sched-paper-stack">`
  const titleEl = title ? `<title>${escapeXml(title)}</title>` : ''
  return {
    svg: `${head}${titleEl}${body}</svg>`,
    pages: total,
    w: totalW,
    h: totalH,
    truncated: tiling.truncated,
    scale: tiling.scale,
  }
}
