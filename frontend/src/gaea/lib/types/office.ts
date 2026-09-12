import type { WireShape } from "./shared";
import type { app as AppModels } from "../../../../wailsjs/go/models";

// ScheduleLoadResult / ScheduleSaveResult 对齐后端 GaeaScheduleLoad/Save
// （gaea_schedule_file.go）：进度计划板块文件持久化（v4.113.0 刀4）。
// project 为计划 JSON 串（前端自行解析，schedule/types 的 SchedProject 口径）。
export interface ScheduleLoadResult {
  path: string;
  exists: boolean;
  project: string;
}

export interface ScheduleSaveResult {
  path: string;
  savedAt: string;
  duration: number;
  critical: number;
}

// ── 进度计划多工程（v4.139 #15 刀1+刀2）：每文件一工程 + Go 侧索引指针 ──
// 对齐后端 GaeaScheduleProjects/ProjectOpen/ProjectCreate/ProjectArchive/
// ProjectDelete（gaea_schedule_file.go，设计 docs/gaea-schedule-multi-project-
// design-2026-09.md §3.2/3.3）。rel=工作区相对路径（工程身份：文件名落盘后不改，
// agent 显式 path 引用稳定性优先）；current=当前指针（「对话改的=板块打开的」）。
export interface ScheduleProjectSummary {
  rel: string;
  name: string;
  archived: boolean;
  duration: number;
  taskCount: number;
  ok: boolean;
  updatedAt: string;
}

/** GaeaScheduleProjects / ProjectArchive / ProjectDelete：索引视图（当前指针 + 工程列表）。 */
export interface ScheduleProjectsResult {
  current: string;
  projects: ScheduleProjectSummary[];
}

/** GaeaScheduleProjectOpen：切指针成功后的当前 rel。 */
export interface ScheduleProjectOpenResult {
  current: string;
}

/** GaeaScheduleProjectCreate：新工程 rel（已登记并自动切为当前）。 */
export interface ScheduleProjectCreateResult {
  rel: string;
  current: string;
}
// ── xlsx 单元格级预览 ──────────────────────────────────────
export interface XlsxCellStyle {
  bold?: boolean;
  italic?: boolean;
  underline?: boolean;
  strike?: boolean;
  fontColor?: string;
  fill?: string;
  align?: "left" | "center" | "right";
  wrap?: boolean;
  numFmt?: string;
  border?: boolean;
}

export interface XlsxCell {
  ref: string; // "A1"
  value: string;
  formula?: string;
  type?: string; // number|string|bool|date|formula|error
  style?: XlsxCellStyle;
}

export interface XlsxSheet {
  name: string;
  rows: XlsxCell[][];
  merged?: string[]; // "A1:B2"
  colWidths?: Record<string, number>;
  freeze?: { row?: number; col?: number }; // 冻结窗格（表头）
  truncated?: boolean;
  // 条件格式（CellIs 静态可判定子集；后端 xlsxpreview/condfmt.go 提取）
  condRules?: XlsxCondRule[];
  condSkipped?: number; // 无法静态判定而被跳过的规则数
}

export interface XlsxCondRule {
  range: string; // "D2:D100"（sqref 单段）
  op: "greaterThan" | "lessThan" | "greaterThanOrEqual" | "lessThanOrEqual" | "equal" | "notEqual" | "between" | "notBetween";
  formulas: string[]; // 常量阈值（数字原文或带引号文本）
  priority?: number; // Excel 优先级（小者先行）
  fill?: string; // 6 位 hex
  fontColor?: string;
  bold?: boolean;
}

export interface XlsxPreview {
  sheets: XlsxSheet[];
}

// XlsxEditResult 是单元格编辑结果：更新后的预览 + 摘要。
export type XlsxEditResult = WireShape<AppModels.XlsxEditResult>;

// XlsxPlanResult 是「先规划后应用」的规划结果：ops 原样带回（应用时透传），
// 附变更清单供用户审阅批准（对标 Copilot Plan/Show Changes 范式）。
export type XlsxPlanResult = WireShape<AppModels.XlsxPlanResult>;

// ── 表格「选中区域 → 一键图表」（原生图表嵌入工作簿） ──
export interface XlsxChartInput {
  rel: string; // xlsx 工作区相对路径
  sheet?: string; // 工作表名；空 = 第一个工作表
  refs?: string; // 区域 "A1:B6" 或单单元格 "B2"；空 = 自动取前两列数据行
  chartType?: "bar" | "line" | "pie" | "scatter";
  title?: string;
}

export type XlsxChartResult = WireShape<AppModels.XlsxChartResult>;

// ── 会话产物一键打包（P0-1，对标 Kimi 工作空间 / WorkBuddy） ──
export type ZipDeliverableResult = WireShape<AppModels.ZipDeliverableResult>;

// ConvertPdfResult 是文档转 PDF 的结果（PDF 落 .gaea/exports/）。
export type ConvertPdfResult = WireShape<AppModels.ConvertPdfResult>;
// ── 统一交付出口（事实底座 → 多形态交付） ──────────────────
export interface ExportDeliverableInput {
  markdown: string;
  format: "docx" | "pptx" | "xlsx" | "md" | "pdf";
  title?: string;
  template?: "通用" | "公文" | "报告" | "合同";
  cover?: boolean;
  toc?: boolean;
  header?: string;
  footer?: string;
}

export type ExportDeliverableResult = WireShape<AppModels.ExportDeliverableResult>;

// ── 跨应用联动（xlsx 数据 → 图表 → 嵌入 docx/pptx） ────────
export interface CrossEmbedInput {
  xlsxRel: string;
  sheet?: string;
  range?: string; // "A1:B6"，空 = 自动
  chartType?: "bar" | "line" | "pie" | "scatter";
  title?: string;
  into: "docx" | "pptx";
  output?: string;
}

export type CrossEmbedResult = WireShape<AppModels.CrossEmbedResult>;
// ── v4.1 证据链（docs/gaea-v41-evidence-chain-design.md §3）────────────────
// JournalChangeRecord 是 GaeaJournalList 返回的证据卡视图（对齐 Go
// evidence.ChangeRecord JSON；wailsjs 生成物刷新前手写，字段漂移由
// typesGenerationCheck 同范式在后续 Step 收口）。
export interface JournalChangeRecord {
  id: string;
  sessionId: string;
  space: string;
  turn: number;
  tool: string;
  target: string;
  beforeSummary: string;
  afterSummary: string;
  model?: string;
  at: number; // unix ms
  status: string;
  // v4.8 Verifier 产品化（证据链翻成 UI）：基线快照路径——无则回滚按钮禁用
  // （「无基线快照，无法回滚」）；opsJson 为 AI 原始操作集 JSON（xlsx_apply 卡
  // 携带），可解析出「声明↔实况」diff 与操作回放时间线。
  baselinePath?: string;
  opsJson?: string;
}

// XlsxOpView 是 xlsx_apply 操作集单条 op 的前端只读视图（对齐
// internal/office/xlsxedit.Op：type 11 种——set_value/set_formula/fill_range/
// transform/replace/split_column/clean/set_style/merge_cells/unmerge_cells/
// set_col_width；字段按类型复用）。
export interface XlsxOpView {
  type: string;
  sheet?: string;
  target?: string; // "B4"
  range?: string; // "A1:B10"
  value?: string | number | boolean | null;
  formula?: string;
  find?: string;
  replace?: string;
  col?: string; // "A"
  sep?: string;
  newCols?: string[];
  headers?: string[];
  width?: number;
}

// VerifyDiffRow 是「声明↔实况」逐格比对结果（verifyDiff.ts 纯函数产出；
// ok=match 数值容差 1e-9/字符串去空白/公式归一一致，mismatch 不一致，
// skip=前端无法核对，如 replace 批量格或预览缺该格）。
export interface VerifyDiffRow {
  sheet: string;
  cell: string;
  claimed: string; // 声明值（公式行带 fx = 前缀）
  actual: string; // 实况值（来自 GaeaPreview；公式行带 fx = 前缀）
  ok: "match" | "mismatch" | "skip";
}

// VerdictView 是 Verifier 双通道复核结论（v4.1b，GaeaVerifyRecord）。
export interface VerdictView {
  id: string;
  status: "verified" | "warned" | "failed";
  channelA?: string;
  channelB?: string;
  note?: string;
  at: number;
  // v4.16 通道 B 结果产品化：像素差异率（0-1）/ 渲染页数 / 审计产物目录
  // （.gaea/work/journal/verify/<id>/，绝对路径）。旧 verdict / 无通道 B /
  // 渲染降级时省略（Go omitempty），前端不渲染「视觉复核」行。
  channelBRatio?: number;
  channelBPages?: number;
  channelBArtifacts?: string;
}

// LintReportView 是中文规范体检结果（v4.1c，GaeaDocumentLint）。
export interface LintIssueView {
  element: string;
  found: boolean;
  note: string;
  // v4.6.1 规范包机制化：Issue 归属的规范包名（GB/T 9704 红头要素 / 造价工程表式）。
  spec?: string;
}
export interface LintReportView {
  path: string;
  issues: LintIssueView[];
  passed: boolean;
  summary: string;
}

// ── 6.3 办公多文件 DAG（DagNodeView/DagRunView，Go OfficeB.GaeaDag* 契约冻结）──
// DagNodeView 是流水线单节点视图：ref=最近一次子代理运行 ref（sa_ 前缀）；
// outputs=工作区相对路径产物；runCount 已跑次数；steerCount 续跑改向次数；
// acceptedAt 验收时间。status：pending 待跑 / running / done / failed /
// skipped 级联终止或依赖失败跳过 / accepted 已验收（产物回流记忆）/
// hold 高风险待审批（v4.243 审批分级，approve 后回 pending）。risk=风险分级
// （high=起跑前置审批闸）；approved=人已批准本次执行。
export interface DagNodeView {
  id: string;
  title: string;
  prompt: string;
  dependsOn?: string[];
  status: "pending" | "running" | "done" | "failed" | "skipped" | "accepted" | "hold";
  risk?: "normal" | "high";
  approved?: boolean;
  ref?: string;
  outputs?: string[];
  error?: string;
  runCount: number;
  steerCount?: number;
  acceptedAt?: string;
}

// DagRunView 是整条流水线视图。derived=后端派生态（前端不自算，只渲染）：
// draft=还没跑过 / running / failed=有失败 / ready=全 done 待验收 / accepted=全验收。
export interface DagRunView {
  id: string;
  goal: string;
  createdAt: string;
  updatedAt: string;
  derived: "draft" | "running" | "failed" | "ready" | "accepted";
  nodes: DagNodeView[];
}

// DagTemplateView 是流水线模板（6.3 余项：存模板一键重建）。nodes 只含图形状
// （id/title/prompt/dependsOn，status 恒 pending）；sourceRunId=由哪条 run 存来。
export interface DagTemplateView {
  id: string;
  name: string;
  goal: string;
  nodes: DagNodeView[];
  createdAt: string;
  sourceRunId?: string;
}
