// docxText.ts — docx 纯文本提取（渲染失败降级的兜底视图用，零后端依赖）。
//
// Why: DocxPreview 的 docx-preview 保真渲染在个别文档（损坏包/未支持的特性）
// 上会抛异常，此前直接落死错误页。降级策略：尽力从 docx 包内
// word/document.xml 提取段落文本，给用户一个可读的纯文本视图 + 诚实的
// 降级说明（不含图片/文本框/版式），而不是一屏报错。
//
// How: docx = zip 包。解包复用 docx-preview 自带的 jszip 传递依赖（DocxPreview
// 渲染时 jszip 必然已在模块图内），解析用标准 DOMParser（浏览器/jsdom 均内建），
// 只取 w:p 聚合 w:t 文本，不引第三方解析器。

import JSZip from "jszip";

const DOCX_MIME = "application/vnd.openxmlformats-officedocument.wordprocessingml.document";

// dataURL → 二进制（兼容裸 base64：只取逗号后的负载）。
function dataUrlToBytes(dataUrl: string): Uint8Array {
  const comma = dataUrl.indexOf(",");
  const b64 = comma >= 0 ? dataUrl.slice(comma + 1) : dataUrl;
  const bin = atob(b64);
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
  return bytes;
}

// 提取 docx 正文段落文本（含表格内段落，按文档顺序）。任何一步失败都抛错，
// 由调用方决定降级层级（提取失败则回死错误页并如实展示两个错误）。
export async function extractDocxParagraphs(dataUrl: string): Promise<string[]> {
  const zip = await JSZip.loadAsync(dataUrlToBytes(dataUrl));
  const entry = zip.file("word/document.xml");
  if (!entry) throw new Error("docx 包内缺少 word/document.xml");
  const xml = await entry.async("string");
  const doc = new DOMParser().parseFromString(xml, "application/xml");
  if (doc.getElementsByTagName("parsererror").length > 0) {
    throw new Error("word/document.xml 解析失败");
  }
  const out: string[] = [];
  const paragraphs = doc.getElementsByTagName("w:p");
  for (let i = 0; i < paragraphs.length; i++) {
    const p = paragraphs[i];
    let text = "";
    const runs = p.getElementsByTagName("w:t");
    for (let j = 0; j < runs.length; j++) text += runs[j].textContent ?? "";
    // 制表/换行标记折叠为空格，行内多余空白收敛（纯文本视图可读性优先）。
    out.push(text.replace(/[\t\r\n]+/g, " ").trim());
  }
  return out;
}

// 降级视图用 dataURL 构建（测试样例 docx 生成用）。
export function bytesToDocxDataUrl(bytes: Uint8Array): string {
  let bin = "";
  for (let i = 0; i < bytes.length; i++) bin += String.fromCharCode(bytes[i]);
  return `data:${DOCX_MIME};base64,${btoa(bin)}`;
}
