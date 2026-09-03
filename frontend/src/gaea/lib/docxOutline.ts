// docxOutline.ts — docx 标题结构（目录/大纲）提取与渲染锚点链接（零后端依赖）。
//
// Why: DocxPreview 用 docx-preview 保真渲染 .docx，但没有 Word/WPS 那样的
// 「目录」导航——长文档只能滚动找章节。本模块从 docx 包内 word/document.xml
// + word/styles.xml 直接解析标题结构（样式继承 + 大纲级别），渲染完成后再把
// 每个标题条目链接回版式中的段落锚点，供前端目录侧栏点击定位。
//
// How: docx = zip 包。解包复用 docx-preview 自带的 jszip 传递依赖（与
// docxText.ts 同套路），解析用标准 DOMParser。标题判定优先级：
//   1) 段落自身 w:outlineLvl（0..8 → 1..9 级；9 = 正文文本，即使套了标题
//      样式也按正文处理——Word 语义如此）；
//   2) 段落样式（w:pStyle）经 styles.xml 解析出的标题级——样式自带
//      outlineLvl、或样式名/id 命中 heading/标题 N、或沿 basedOn 链继承；
//   3) 都不是 → 普通段落，不进目录。
// 目录域（TOC1/TOC2… 样式）与页眉/页脚段落不计入标题，避免「目录页重复
// 标题文本」把定位锚点引到目录页。

import JSZip from "jszip";

export interface DocxOutlineItem {
  /** 标题级（1..9，1 = 一级标题） */
  level: number;
  /** 标题文字（正文 w:t 聚合、空白收敛；不含 Word 自动编号前缀） */
  text: string;
  /** 该段落在整个 word/document.xml w:p 序列中的序号（0 起；含普通段落） */
  paraIndex: number;
}

// 样式名/id 命中标题的判定：heading 1 / Heading1 / 标题 1 / 标题1 …
const HEADING_NAME_RE = /^(heading|标题)\s*([1-9])$/i;
const MAX_MATCH_SCAN = 500;

interface DocxStyleInfo {
  id: string;
  name: string | null;
  basedOn: string | null;
  /** 样式自带 outlineLvl 原始值（0..9；undefined = 未显式声明） */
  outlineRaw: number | undefined;
}

function dataUrlToBytes(dataUrl: string): Uint8Array {
  const comma = dataUrl.indexOf(",");
  const b64 = comma >= 0 ? dataUrl.slice(comma + 1) : dataUrl;
  const bin = atob(b64);
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
  return bytes;
}

function parseXmlString(xml: string): Document {
  const doc = new DOMParser().parseFromString(xml, "application/xml");
  if (doc.getElementsByTagName("parsererror").length > 0) {
    throw new Error("docx XML 解析失败");
  }
  return doc;
}

function childElement(el: Element, tagName: string): Element | null {
  for (let i = 0; i < el.children.length; i++) {
    const c = el.children[i] as Element;
    if (c.tagName === tagName) return c;
  }
  return null;
}

function normalizeStyleToken(v: string): string {
  return v.replace(/[\s\u00a0._-]+/g, "").toLowerCase();
}

function headingLevelOfToken(v: string): number | undefined {
  const m = HEADING_NAME_RE.exec(normalizeStyleToken(v));
  return m ? Number(m[2]) : undefined;
}

function parseStylesMap(stylesDoc: Document): Map<string, DocxStyleInfo> {
  const map = new Map<string, DocxStyleInfo>();
  const styleEls = stylesDoc.getElementsByTagName("w:style");
  for (let i = 0; i < styleEls.length; i++) {
    const styleEl = styleEls[i];
    const id = styleEl.getAttribute("w:styleId") ?? "";
    if (!id) continue;
    const nameEl = childElement(styleEl, "w:name");
    const basedOnEl = childElement(styleEl, "w:basedOn");
    const pPr = childElement(styleEl, "w:pPr");
    const lvlEl = pPr ? childElement(pPr, "w:outlineLvl") : null;
    const lvlRaw = lvlEl ? parseInt(lvlEl.getAttribute("w:val") ?? "", 10) : Number.NaN;
    map.set(id, {
      id,
      name: nameEl?.getAttribute("w:val") ?? null,
      basedOn: basedOnEl?.getAttribute("w:val") ?? null,
      outlineRaw: Number.isInteger(lvlRaw) ? lvlRaw : undefined,
    });
  }
  return map;
}

// 解析样式表的标题级：样式自带 outlineLvl > 样式名/id 命中 > basedOn 链继承。
function styleHeadingLevel(
  styles: Map<string, DocxStyleInfo>,
  styleId: string,
  seen = new Set<string>(),
): number | undefined {
  if (!styleId || seen.has(styleId)) return undefined;
  const style = styles.get(styleId);
  if (!style) return undefined;
  seen.add(styleId);
  if (style.outlineRaw !== undefined) {
    return style.outlineRaw >= 0 && style.outlineRaw <= 8 ? style.outlineRaw + 1 : undefined;
  }
  const byName = headingLevelOfToken(style.name ?? "");
  if (byName !== undefined) return byName;
  const byId = headingLevelOfToken(style.id);
  if (byId !== undefined) return byId;
  return style.basedOn ? styleHeadingLevel(styles, style.basedOn, seen) : undefined;
}

function paragraphOutlineLevel(p: Element, styles: Map<string, DocxStyleInfo>): number | undefined {
  const pPr = childElement(p, "w:pPr");
  // 段落自身显式 outlineLvl 优先；9 及越界值 = 正文（显式排除标题样式）。
  const lvlEl = pPr ? childElement(pPr, "w:outlineLvl") : null;
  if (lvlEl) {
    const raw = parseInt(lvlEl.getAttribute("w:val") ?? "", 10);
    if (Number.isInteger(raw)) {
      return raw >= 0 && raw <= 8 ? raw + 1 : undefined;
    }
  }
  const pStyleEl = pPr ? childElement(pPr, "w:pStyle") : null;
  const styleId = pStyleEl?.getAttribute("w:val");
  return styleId ? styleHeadingLevel(styles, styleId) : undefined;
}

// 提取 docx 正文标题结构（含表格内段落，按文档顺序）。任何一步失败都抛错，
// 由调用方决定降级（目录不可用不影响版式预览本身）。
export async function extractDocxOutline(dataUrl: string): Promise<DocxOutlineItem[]> {
  const zip = await JSZip.loadAsync(dataUrlToBytes(dataUrl));
  const docEntry = zip.file("word/document.xml");
  if (!docEntry) throw new Error("docx 包内缺少 word/document.xml");
  const doc = parseXmlString(await docEntry.async("string"));

  const stylesEntry = zip.file("word/styles.xml");
  const styles = stylesEntry
    ? parseStylesMap(parseXmlString(await stylesEntry.async("string")))
    : new Map<string, DocxStyleInfo>();

  const items: DocxOutlineItem[] = [];
  const paragraphs = doc.getElementsByTagName("w:p");
  for (let i = 0; i < paragraphs.length; i++) {
    const p = paragraphs[i];
    const level = paragraphOutlineLevel(p, styles);
    if (level === undefined) continue;
    let text = "";
    const runs = p.getElementsByTagName("w:t");
    for (let j = 0; j < runs.length; j++) text += runs[j].textContent ?? "";
    text = text.replace(/\s+/g, " ").trim();
    if (!text) continue;
    items.push({ level, text, paraIndex: i });
  }
  return items;
}

export function normalizeDocxOutlineText(s: string): string {
  return s.replace(/\s+/g, " ").trim();
}

// 渲染完成后把标题条目链接回版式段落。跳过页眉/页脚、目录域（TOC1…）与
// 批注弹层内的重复文本，避免「目录页/页眉同名标题」抢占锚点。条目在版式中
// 找不到精确文本时返回 null（不推进扫描游标，后续条目仍可继续匹配）。
export function linkDocxOutlineAnchors(
  container: HTMLElement,
  items: readonly DocxOutlineItem[],
): (HTMLElement | null)[] {
  const anchors: (HTMLElement | null)[] = new Array(items.length).fill(null);
  if (items.length === 0) return anchors;
  const paragraphs = Array.from(container.querySelectorAll("p")).filter((el) => {
    if (el.closest("header, footer")) return false;
    if (el.closest(".docx-comment-popover")) return false;
    const classTokens = Array.from(el.classList);
    if (classTokens.some((c) => /(^|[_-])toc\d+$/i.test(c))) return false;
    return normalizeDocxOutlineText(el.textContent ?? "") !== "";
  });
  let cursor = 0;
  for (let i = 0; i < items.length; i++) {
    const target = normalizeDocxOutlineText(items[i].text);
    if (!target) continue;
    const end = Math.min(paragraphs.length, cursor + MAX_MATCH_SCAN);
    let found = -1;
    for (let j = cursor; j < end; j++) {
      if (normalizeDocxOutlineText(paragraphs[j].textContent ?? "") === target) {
        found = j;
        break;
      }
    }
    if (found >= 0) {
      anchors[i] = paragraphs[found];
      cursor = found + 1;
    }
  }
  return anchors;
}
