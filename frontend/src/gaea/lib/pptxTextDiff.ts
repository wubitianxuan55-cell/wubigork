// pptxTextDiff.ts — pptx 版本结构化对比纯函数层（v4.156.0 刀3「结构化对比」，
// docs/gaea-pptx-edit-design-2026-09.md §3.3）。
//
// Why: 版本时间线的「与当前对比」此前对 .pptx 只能落 kind:"unsupported" 降级
//（双版本并排预览）。本模块把两份 pptx 的页序列做页对齐 + 页内段落 LCS diff，
// 收掉这笔记账（蓝本 docxTextDiff：段级 LCS，算法零改动直接复用）。
//
// How: pptx = zip 包。JSZip 解 ppt/slides/slideN.xml + DOMParser 抽 a:p 聚合
// a:t（docxText 解 word/document.xml 同套路）；本模块只做纯函数 diff 与结构化
// 防御，无 React 依赖、不触碰 bridge；取数与 async 编排在 versionCompare.ts。
//
// 分层（§3.3）：
//   层1 slide 级：整页文本签名做 LCS 页对齐（相等页为锚，锚间双侧余页按位置
//     配对成 changed）→ 页数/页序摘要（summary）；
//   层2 段落级：对齐后的 changed 页逐页做段落 LCS（diffDocxParagraphs 复用）→
//     {type,page,index,text} 行（equal/add/del 全量，宿主自行取舍）；
//   层3 字符级：不在本层实现——相邻 del+add 对交宿主 ChangesDiff 渲染管线，
//     自动获得改蓝配对 + 行内字符级高亮（白得，不新写）。
//
// 口径说明：
//   - 页序 = slide 文件名自然序（slide2 < slide10，按编号数值排序）。这是显示
//     层近似：后端 pptxedit.slideParts 用 presentation.xml sldIdLst 的放映顺序，
//     常规 deck 两者一致；前端 zip 直读不解析 rels，以文件名序对齐（宁可近似
//     也不伪造放映序）。
//   - 页内文本 = 递归取该页 XML 所有 a:p 聚合 a:t（正文文本框/表格/组合形状
//     自然命中，与刀1 pptxedit 的「自然命中」口径一致）；备注页/母版不在
//     slides/ 目录，天然排除。用 getElementsByTagNameNS 按 DrawingML 命名空间
//     取元素，不依赖 a: 前缀写法。
//   - 空段落保留参与 diff（空行是版面结构的一部分，同 docxTextDiff 口径）；
//     制表/换行折叠为空格后 trim。
//   - 行的 page/index：add/equal 取当前侧页码与页内序号，del 取基线侧
//     （DocxRow.index 同语义）。changed 配对页的行统一挂当前侧页码（对齐后
//     展示页 = 用户当前所见页），del/add 行再各带页内序号。
//   - 防御（宁漏勿误）：非 zip / 非 pptx（无 slides 目录）/ 页 XML 解析失败
//     → 结构化 {ok:false, reason}，宿主降级 unsupported 原文案，不伪造对比。

import JSZip from "jszip";
import { diffDocxParagraphs } from "./docxTextDiff";

/** 单页文本快照：page 为 1 起页码（提取序 = 展示序）；texts 为页内段落文本序列。 */
export interface PptxSlideTexts {
  page: number;
  texts: string[];
}

/** 层2 一行：type 语义同 DocxRow（equal = 未变 ctx），page 为展示页码，
 *  index 为页内段落序号（del 取基线侧、add/equal 取当前侧，1 起）。 */
export interface PptxRow {
  type: "equal" | "add" | "del";
  page: number;
  index: number;
  text: string;
}

/** 层1 页对齐条目（walk 序）：rowCount 为该页在 rows 中贡献的连续行数
 *  （宿主按 (pages, rows) 双游标切片，页码在基线/当前两侧各自编号，不做唯一键）。 */
export interface PptxPageDiff {
  page: number;
  state: "equal" | "add" | "del" | "changed";
  rowCount: number;
}

/** 层1 页级摘要：两侧页数 + 页级增删改计数。 */
export interface PptxPageSummary {
  pagesBase: number;
  pagesCur: number;
  added: number;
  removed: number;
  changed: number;
}

/** diff 成功结果：页摘要 + 页对齐条目 + 层2 全量行。 */
export interface PptxTextDiffOk {
  ok: true;
  summary: PptxPageSummary;
  pages: PptxPageDiff[];
  rows: PptxRow[];
}

/** 结构化失败：解包失败 / 非 pptx / 无 slides 目录 / 页 XML 解析失败。 */
export interface PptxTextDiffFail {
  ok: false;
  reason: string;
}

export type PptxTextDiffResult = PptxTextDiffOk | PptxTextDiffFail;

/** 解包成功结果（slides 供 diffPptxSlideTexts 消费）。 */
export interface PptxSlidesOk {
  ok: true;
  slides: PptxSlideTexts[];
}

// dataURL → 字节（docxText.dataUrlToBytes 同套路；兼容裸 base64：只取逗号后
// 负载）。AttachmentDataURL 返回字节精确 base64，本模块输入即原始字节。
export function pptxBytesFromDataUrl(dataUrl: string): Uint8Array {
  const comma = dataUrl.indexOf(",");
  const b64 = comma >= 0 ? dataUrl.slice(comma + 1) : dataUrl;
  const bin = atob(b64);
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
  return bytes;
}

// slideN.xml 的 N（自然序排序键；非 slideN.xml 命名不会进候选）。
function slideNoOf(name: string): number {
  const m = /slide(\d+)\.xml$/i.exec(name);
  return m ? Number(m[1]) : 0;
}

// DrawingML 命名空间（a:p / a:t 所在；按 NS 取元素，不依赖前缀写法）。
const DRAWINGML_NS = "http://schemas.openxmlformats.org/drawingml/2006/main";

/**
 * 解包 pptx 字节 → 逐页文本快照。结构化防御：任何一步不可信都返回
 * {ok:false, reason}（不抛错），宿主据此诚实降级。页码 = 文件名自然序
 * （显示层近似，见头部口径说明）。
 */
export async function parsePptxSlides(bytes: Uint8Array): Promise<PptxSlidesOk | PptxTextDiffFail> {
  let zip: JSZip;
  try {
    zip = await JSZip.loadAsync(bytes);
  } catch (e) {
    return { ok: false, reason: `zip 解包失败：${e instanceof Error ? e.message : String(e)}` };
  }
  const names = Object.keys(zip.files)
    .filter((n) => /^ppt\/slides\/slide\d+\.xml$/i.test(n))
    .sort((a, b) => slideNoOf(a) - slideNoOf(b));
  if (names.length === 0) {
    return { ok: false, reason: "pptx 包内无幻灯片（非 .pptx 或结构异常）" };
  }
  const slides: PptxSlideTexts[] = [];
  for (let i = 0; i < names.length; i++) {
    const xml = await zip.files[names[i]].async("string");
    const doc = new DOMParser().parseFromString(xml, "application/xml");
    if (doc.getElementsByTagName("parsererror").length > 0) {
      return { ok: false, reason: `${names[i]} 解析失败` };
    }
    const texts: string[] = [];
    const paras = doc.getElementsByTagNameNS(DRAWINGML_NS, "p");
    for (let p = 0; p < paras.length; p++) {
      let text = "";
      const runs = paras[p].getElementsByTagNameNS(DRAWINGML_NS, "t");
      for (let r = 0; r < runs.length; r++) text += runs[r].textContent ?? "";
      // 制表/换行折叠为空格后 trim（纯文本对比可读性优先，同 docxText 口径）。
      texts.push(text.replace(/[\t\r\n]+/g, " ").trim());
    }
    slides.push({ page: i + 1, texts });
  }
  return { ok: true, slides };
}

/**
 * 层1+层2：两侧逐页文本快照 → 页对齐 + 页内段落 diff。
 * 页对齐：整页文本签名（段拼接）做经典 LCS，相等页为锚；锚间 gap 里基线余页
 * 与当前余页按位置两两配对成 changed（与行级 diff 的「改 = −1 +1」同语义，
 * 位置配对保证页序错位时仍能给出可读的页级对应）；基线独有页 → del、当前
 * 独有页 → add。changed 页内再走 diffDocxParagraphs 段落 LCS。
 */
export function diffPptxSlideTexts(base: PptxSlideTexts[], cur: PptxSlideTexts[]): PptxTextDiffOk {
  const n = base.length;
  const m = cur.length;
  // 整页签名：段文本以 NUL 拼接（提取层已把段落内换行折叠为空格，NUL 不会
  // 出现在段落文本里，拼接无歧义）。
  const baseSig = base.map((s) => s.texts.join("\u0000"));
  const curSig = cur.map((s) => s.texts.join("\u0000"));
  // dp[i][j] = baseSig[i..] 与 curSig[j..] 的 LCS 长度（与 docxTextDiff 同一套全量矩阵）。
  const dp: number[][] = Array.from({ length: n + 1 }, () => new Array<number>(m + 1).fill(0));
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      dp[i][j] =
        baseSig[i] === curSig[j] ? dp[i + 1][j + 1] + 1 : Math.max(dp[i + 1][j], dp[i][j + 1]);
    }
  }
  // 回溯出相等页锚点（与 docxTextDiff 同一回溯取向：先耗基线侧）。
  const anchors: Array<[number, number]> = [];
  {
    let i = 0;
    let j = 0;
    while (i < n && j < m) {
      if (baseSig[i] === curSig[j]) {
        anchors.push([i, j]);
        i++;
        j++;
      } else if (dp[i + 1][j] >= dp[i][j + 1]) {
        i++;
      } else {
        j++;
      }
    }
  }
  const pages: PptxPageDiff[] = [];
  const rows: PptxRow[] = [];
  const summary: PptxPageSummary = { pagesBase: n, pagesCur: m, added: 0, removed: 0, changed: 0 };
  // 段行统一入列并记账：equal 行也入列（层2 全量口径，宿主自行取舍）。
  const pushRows = (page: number, state: PptxPageDiff["state"], made: PptxRow[]) => {
    for (const r of made) rows.push(r);
    pages.push({ page, state, rowCount: made.length });
  };
  let bi = 0;
  let cj = 0;
  for (let a = 0; a <= anchors.length; a++) {
    // 锚点（或尾部哨兵 [n,m]）之前的 gap：基线余页与当前余页按位置配对。
    const anchorEnd = a < anchors.length ? anchors[a] : ([n, m] as [number, number]);
    let gi = bi;
    let gj = cj;
    while (gi < anchorEnd[0] && gj < anchorEnd[1]) {
      const page = cur[gj].page; // changed 页挂当前侧页码（用户当前所见页）
      const made: PptxRow[] = diffDocxParagraphs(base[gi].texts, cur[gj].texts).map((r) => ({
        type: r.type === "ctx" ? ("equal" as const) : r.type,
        page,
        index: r.index,
        text: r.text,
      }));
      pushRows(page, "changed", made);
      summary.changed++;
      gi++;
      gj++;
    }
    while (gi < anchorEnd[0]) {
      // 基线独有页 → 整页 del（页码/序号取基线侧）。
      const made: PptxRow[] = base[gi].texts.map((t, k) => ({
        type: "del" as const,
        page: base[gi].page,
        index: k + 1,
        text: t,
      }));
      pushRows(base[gi].page, "del", made);
      summary.removed++;
      gi++;
    }
    while (gj < anchorEnd[1]) {
      // 当前独有页 → 整页 add。
      const made: PptxRow[] = cur[gj].texts.map((t, k) => ({
        type: "add" as const,
        page: cur[gj].page,
        index: k + 1,
        text: t,
      }));
      pushRows(cur[gj].page, "add", made);
      summary.added++;
      gj++;
    }
    if (a < anchors.length) {
      const [ai, aj] = anchors[a];
      pushRows(
        cur[aj].page,
        "equal",
        cur[aj].texts.map((t, k) => ({ type: "equal" as const, page: cur[aj].page, index: k + 1, text: t })),
      );
      bi = ai + 1;
      cj = aj + 1;
    }
  }
  return { ok: true, summary, pages, rows };
}

/**
 * 顶层入口：两侧 pptx 原始字节 → 结构化 diff。任一侧解包/解析不可信时返回
 * {ok:false, reason}（宿主降级 unsupported 原文案，诚实降级不伪造对比）。
 */
export async function diffPptxBytes(baseBytes: Uint8Array, curBytes: Uint8Array): Promise<PptxTextDiffResult> {
  const [base, cur] = await Promise.all([parsePptxSlides(baseBytes), parsePptxSlides(curBytes)]);
  if (!base.ok) return base;
  if (!cur.ok) return cur;
  return diffPptxSlideTexts(base.slides, cur.slides);
}
