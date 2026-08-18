// 文件扩展名 → 展示工具（单一数据源）。
//
// Why: FileMenu 的扩展名 badge 与文件 chip（行内文件引用 / Composer 附件）此前
// 各自维护 badge 集合与图标映射（03-office-frontend.md 评审缺陷 4 同型：同一
// 视觉语义多处重复）。这里收敛为单源：扩展名 badge 集合 + 图标名 + 可读类型名。
//
// How to apply: 组件内 `import { extBadge, fileIconName } from "../../lib/fileBadge"`；
// 新增扩展名只需改本文件的集合/映射。

/** 有 badge 展示价值的常用扩展名（@ 菜单 / 附件 chip / 行内引用共用）。 */
export const BADGE_EXTS = new Set([
  "doc", "docx", "pdf", "xls", "xlsx", "ppt", "pptx", "md", "txt",
  "csv", "png", "jpg", "jpeg", "svg",
]);

/** 取文件扩展名小写（无扩展名返回 null）。 */
export function extOf(name: string): string | null {
  const m = /\.([a-z0-9]+)$/i.exec(name);
  return m ? m[1].toLowerCase() : null;
}

/** 扩展名 badge 文案（无 badge 价值的扩展名返回 "file" 兜底）。 */
export function extBadge(name: string): string {
  const ext = extOf(name);
  return ext && BADGE_EXTS.has(ext) ? ext : "file";
}

/** 扩展名 → 语义图标名（icons.ts 导出名；未知返回默认文件图标）。 */
export function fileIconName(name: string): "FileImage" | "FileSpreadsheet" | "FilePpt" | "FileText" {
  const ext = extOf(name);
  if (ext && ["png", "jpg", "jpeg", "gif", "webp", "svg", "bmp", "ico"].includes(ext)) return "FileImage";
  if (ext && ["xls", "xlsx", "csv", "tsv"].includes(ext)) return "FileSpreadsheet";
  if (ext && ["ppt", "pptx"].includes(ext)) return "FilePpt";
  return "FileText";
}

/** 可读类型名（chip title / aria-label 辅助）。 */
export function fileTypeLabel(name: string): string {
  const ext = extOf(name);
  if (!ext) return "文件";
  const labels: Record<string, string> = {
    docx: "Word 文档", doc: "Word 文档", xlsx: "Excel 表格", xls: "Excel 表格",
    pptx: "演示文稿", ppt: "演示文稿", pdf: "PDF 文档", md: "Markdown",
    txt: "文本", csv: "CSV 表格", png: "图片", jpg: "图片", jpeg: "图片",
    webp: "图片", svg: "图片", gif: "图片",
  };
  return labels[ext] ?? `${ext.toUpperCase()} 文件`;
}
