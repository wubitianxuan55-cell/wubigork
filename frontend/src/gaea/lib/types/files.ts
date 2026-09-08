import type { WireShape } from "./shared";
import type { app as AppModels } from "../../../../wailsjs/go/models";

export type DirEntry = WireShape<AppModels.DirEntry>;

export interface FileSearchHit {
  path: string; // 工作区相对路径（/ 分隔）
  name: string;
  isDir: boolean;
  size?: number;
  modTime: number; // S2.3 types 漂移修复：后端返回修改时间（unix ms）
}

/** AtEntry 是 @ 菜单的统一条目（目录浏览 / 工作区搜索 / 最近使用文件）。 */
export interface AtEntry {
  path: string; // 工作区相对路径；目录以 / 结尾
  name: string;
  isDir: boolean;
  size?: number;
}

// WorkspaceSearchHit 是工作区全文搜索的一条命中（轻量 RAG）。
export interface WorkspaceSearchHit {
  path: string;
  name: string;
  size: number;
  modTime: number;
  score: number;
  snippet: string;
  truncated?: boolean;
  skipped?: string;
}

// GaeaSummaryResult 是资料「摘要」操作的结果（map-reduce 分块摘要）。
export type GaeaSummaryResult = WireShape<AppModels.GaeaSummaryResult>;

// TaskTemplate 是预置办公任务模板（欢迎页「任务模板」区 + slash 命令）。
export type TaskTemplate = WireShape<AppModels.TaskTemplate>;

// 后端 GaeaReadFile 真实契约（gaea_ui_extra.go FilePreview = path/markdown/size）；
// 旧手写体含 body/truncated/binary 为历史残留，已收敛为生成模型别名。
export type FilePreview = WireShape<AppModels.FilePreview>;
// 文件预览负载：kind 决定渲染方式（image/docx/xlsx/pdf/markdown/text/unsupported/error）。
// docx 时 dataUrl 为原始文件（前端 docx-preview 保真渲染）。
// xlsx 时 body 为结构化单元格 JSON（值/公式/样式，前端表格渲染）。
// pdf 时为 .pptx 转换产物（gaea_pptx.go previewPptx）：pages 有值走逐页缩略
// （页锚点供大纲卡滚动），否则回退 dataUrl 整本内嵌；hint="outline" 提示可
// 另行拉取 PptxOutline 大纲卡。
export interface PreviewResult {
  path: string;
  name: string;
  ext: string;
  size: number;
  kind: "image" | "docx" | "xlsx" | "pdf" | "markdown" | "text" | "html" | "unsupported" | "error";
  body: string;
  dataUrl: string;
  error: string;
  truncated?: boolean;
  totalPages?: number;
  /** .pptx 预览的逐页缩略（page 1-based，dataUrl 为该页 PNG）。 */
  pages?: PreviewPageThumb[];
  /** 扩展能力提示："outline" = 可调 GaeaPptxOutline 拉取大纲卡。 */
  hint?: string;
}

// PreviewPageThumb 是 pptx→PDF 预览的逐页缩略（对齐 gaea_pptx.go PreviewPage）。
export interface PreviewPageThumb {
  page: number;
  dataUrl: string;
}

// PptxSlideOutline / PptxOutlineView 对齐后端 GaeaPptxOutline（gaea_pptx.go）
// 契约：B2 pptx 结构化大纲卡；index 为 1-based 页码（与逐页预览页锚点一致，
// 「针对第 N 页修改」指令即引用此页码）。
export interface PptxSlideOutline {
  index: number;
  title: string;
  /** 正文文本框文本（标题除外，单条后端已截断 ~200 字）。 */
  texts: string[];
  shapeCount: number;
}

export interface PptxOutlineView {
  available: boolean;
  error?: string;
  slides: PptxSlideOutline[];
}

// OfficeEditResult 是框选即改的 AI 编辑结果（替换文本）。
export interface OfficeEditResult {
  edited: string;
}
/** FilePickResult 描述从原生对话框选取的一个文件 */
export interface FilePickResult {
  path: string;
  /** 仅图片文件有 previewUrl（data: URL） */
  previewUrl?: string;
  type: "image" | "file";
  name: string;
  /** 文件字节数（P2-4 附件上下文占用展示用；后端 GaeaPickFiles 已返回）。 */
  size?: number;
}
// 工作区文件语义索引状态 / 命中。
export interface FileIndexStatus {
  total: number;
  skipped: number;
  error: string;
}
export type FileSemanticHit = WireShape<AppModels.FileSemanticHit>;
