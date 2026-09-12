// office.ts — OfficeBindings（AppBindings 分域接口之一，Go OfficeB 门面）：
// docx/pptx/xlsx 编辑 + 进度计划板块 + 交付导出/转换/跨应用联动。
import type {
  ConvertPdfResult,
  CrossEmbedInput,
  CrossEmbedResult,
  DagRunView,
  DagTemplateView,
  ExportDeliverableInput,
  ExportDeliverableResult,
  OfficeEditResult,
  PreviewResult,
  PptxOutlineView,
  ScheduleLoadResult,
  ScheduleProjectCreateResult,
  ScheduleProjectOpenResult,
  ScheduleProjectsResult,
  ScheduleSaveResult,
  SettingsView,
  XlsxChartInput,
  XlsxChartResult,
  XlsxEditResult,
  XlsxPlanResult,
  ZipDeliverableResult,
} from "../types";

// GaeaPptxSlideText 单页条目（pptx 真编辑刀2）：index=1 起页码；paragraphs=
// 该页段落**全文**列表。段落粒度 = GaeaPptxApplyEdit 的 target 粒度（apply 只在
// 单段落内精确匹配），编辑目标必须取此处全文；GaeaPptxOutline 的 texts 是
// 200 rune 截断预览，仅作大纲导航，不能当替换目标。
export interface PptxSlideTextEntry {
  index: number;
  paragraphs: string[];
}

export interface OfficeBindings {
  // PptxOutline 读取 pptx 结构化大纲（v4.28 B2）：python-pptx 逐页标题/正文
  // 摘要，配合 pptx 预览的逐页缩略与「针对第 N 页修改」指令。失败结构化。
  PptxOutline(rel: string): Promise<PptxOutlineView>;
  // OfficeEditText 框选即改：按指令生成选中文本的替换；DocxApplyEdit 以修订模式
  // （w:del+w:ins）写入 docx 并返回更新后的预览。
  OfficeEditText(selectedText: string, instruction: string): Promise<OfficeEditResult>;
  DocxApplyEdit(rel: string, selectedText: string, replacement: string): Promise<PreviewResult>;
  // PptxApplyEdit pptx 真编辑刀1：页码+目标文本+替换 → run 级直接替换（快照+Journal）
  PptxApplyEdit(rel: string, slideIdx: number, target: string, replacement: string): Promise<PreviewResult>;
  // PptxSlideText pptx 真编辑刀2（编辑面）：每页段落全文列表（段落粒度 =
  // PptxApplyEdit 的 target 粒度，见 PptxSlideTextEntry 注）；纯 Go 零 python。
  PptxSlideText(rel: string): Promise<PptxSlideTextEntry[]>;
  // DocxAcceptChanges 接受/拒绝 gaea 的待处理修订，返回更新预览。
  DocxAcceptChanges(rel: string, accept: boolean): Promise<PreviewResult>;
  // XlsxPlanEdit 单元格操作规划（不落盘）：上下文 → AI 规划操作 → 临时副本
  // 试运行 → 返回操作集与单元格级变更清单，供用户审阅批准。
  XlsxPlanEdit(rel: string, sheet: string, instruction: string, selection: string): Promise<XlsxPlanResult>;
  // XlsxApplyEdit 应用已批准的操作集（规划产物原样透传）：excelize 执行 +
  // LibreOffice 重算 → 返回更新预览。
  XlsxApplyEdit(rel: string, ops: string): Promise<XlsxEditResult>;
  // XlsxSetCell 直接写单元格（Excel 式双击编辑）：写值/公式 + LibreOffice 重算。
  XlsxSetCell(rel: string, sheet: string, ref: string, value: string): Promise<XlsxEditResult>;
  // ScheduleLoad/Save 进度计划板块文件持久化（v4.113.0 刀4）：默认计划文件
  // 进度计划/当前计划.gsched.json，与 agent 工具（schedule_*）共享同一资产。
  // v4.139 #15：Save 加可选 rel（方案 A 签名级扩展，防切换竞态——保存显式带
  // 装载时的工程文件，切换只改指针不动保存目标；空串=当前指针文件）。
  ScheduleLoad(): Promise<ScheduleLoadResult>;
  ScheduleSave(projectJSON: string, rel?: string): Promise<ScheduleSaveResult>;
  // ── 进度计划多工程（v4.139 #15 刀1+刀2）：每文件一工程（进度计划/*.gsched.json），
  // 当前指针收敛 Go 侧索引（.gaea/schedule/index.json）——板块 Load/Save 与 agent
  // 三工具缺省 path 解析同一指针，「对话改的=板块打开的」跨多工程保持（设计 §3.3）。
  ScheduleProjects(): Promise<ScheduleProjectsResult>;
  ScheduleProjectOpen(rel: string): Promise<ScheduleProjectOpenResult>;
  ScheduleProjectCreate(name: string): Promise<ScheduleProjectCreateResult>;
  ScheduleProjectArchive(rel: string, archived: boolean): Promise<ScheduleProjectsResult>;
  ScheduleProjectDelete(rel: string): Promise<ScheduleProjectsResult>;
  // ScheduleProjectCopy 工程复制=另存为（v4.145.0）：源文件全套校验→新名 slug
  // 去重落盘→索引登记；指针不动（文件级动作，同归档）。
  ScheduleProjectCopy(rel: string, name: string): Promise<ScheduleProjectsResult>;
  // ScheduleExportXlsx/ImportXlsx 上报 Excel 通道（v4.134.0 刀D3）：导出=计划
  // JSON → 上报口径 xlsx（base64）；导入=上报 xlsx（base64）→ 计划 JSON。
  ScheduleExportXlsx(projectJSON: string): Promise<string>;
  ScheduleImportXlsx(xlsxBase64: string): Promise<string>;
  // ScheduleImportMpp 二进制 MS Project 工程导入（v4.140.0）：mpp（base64）→ 计划 JSON。
  ScheduleImportMpp(mppBase64: string): Promise<string>;
  // XlsxRecalc 手动重算全部公式（LibreOffice）并返回更新预览。
  XlsxRecalc(rel: string): Promise<XlsxEditResult>;
  // XlsxRowOps 行级操作：insert_before / insert_after / delete（基于选中单元格所在行）。
  XlsxRowOps(rel: string, sheet: string, action: string, ref: string): Promise<XlsxEditResult>;
  // XlsxColOps 列级操作：insert_before / insert_after / delete（基于选中单元格所在列）。
  XlsxColOps(rel: string, sheet: string, action: string, ref: string): Promise<XlsxEditResult>;
  // XlsxChart 表格「选中区域 → 一键图表」：从选中区域提取数据 → excelize 在
  // 工作簿内嵌入原生图表对象（Excel/WPS 可见可编辑）→ 返回锚点与数据供迷你预览。
  XlsxChart(input: XlsxChartInput): Promise<XlsxChartResult>;
  // ZipDeliverables 会话产物一键打包：把本次会话交付文件打成一个 zip。
  ZipDeliverables(paths: string[]): Promise<ZipDeliverableResult>;
  // WriteFile 工作区内联编辑保存（C5）：把文本原子写回工作区相对路径文本文件
  // （路径/扩展名/大小校验在后端；用户显式保存，不走 agent 审批）。
  WriteFile(rel: string, content: string): Promise<void>;
  // ExportDeliverable 统一交付出口：受控 Markdown → docx/pptx/xlsx/md/pdf。
  ExportDeliverable(input: ExportDeliverableInput): Promise<ExportDeliverableResult>;
  // ConvertToPdf 文档转 PDF（LibreOffice 无头转换）：docx/xlsx/pptx/odt/html/
  // txt/csv 直接转换，md 先经 create_docx.py 出 docx 再转；PDF 落 .gaea/exports/。
  ConvertToPdf(rel: string): Promise<ConvertPdfResult>;
  // CrossEmbed 跨应用联动：xlsx 数据 → 图表 → 嵌入 docx/pptx。
  CrossEmbed(input: CrossEmbedInput): Promise<CrossEmbedResult>;
  // SaveSettings 整体写回办公引擎设置视图（Go OfficeB.GaeaSaveSettings，wailsjsCompat
  // 直调转正；OfficePanel「保存」消费；读侧见 ModelBindings.Settings → GaeaSettings）。
  SaveSettings(view: SettingsView): Promise<void>;
  // ── 批次三a legacy 直调转正（Go OfficeB.GaeaDataBackup*，Gaea 前缀经 mappings 映射；
  // DataPanel 数据备份/恢复面板消费）──
  // DataBackupInfo 备份概览（lastBackup/status 等）；DataBackupCreate 备份到目标目录；
  // DataBackupRestore 从 zip 包恢复；DataBackupCancel 取消进行中的备份/恢复；
  // DataBackupRollback 回滚最近一次恢复（bool 是否已回滚）；
  // DataBackupRestoreResult 恢复结果（DataPanel 轮询用）。
  DataBackupInfo(): Promise<Record<string, unknown>>;
  DataBackupCreate(destDir: string): Promise<Record<string, unknown>>;
  DataBackupRestore(zipPath: string): Promise<Record<string, unknown>>;
  DataBackupCancel(): Promise<void>;
  DataBackupRollback(): Promise<boolean>;
  DataBackupRestoreResult(): Promise<Record<string, unknown>>;
  // ── 6.3 办公多文件 DAG（Go OfficeB.GaeaDag*，视图类型见 types DagRunView）──
  // DagList 列出全部流水线（DagPanel 拉取/轮询）；DagGet 单条详情。
  // 起跑/单跑/续跑改向/验收/终止均返回人类可读受理文案（Promise<string>，
  // 后端受理即返回，进度靠前端轮询 DagList 自校正，不推事件流）。
  DagList(): Promise<DagRunView[]>;
  DagGet(id: string): Promise<DagRunView>;
  // DagRun 起跑/续跑整链：draft 首跑、failed/ready 续跑（未完成节点续推）。
  DagRun(id: string): Promise<string>;
  // DagNodeRun 单节点跑/重跑（不动下游，重跑只覆盖该节点）。
  DagNodeRun(id: string, nodeId: string): Promise<string>;
  // DagNodeSteer 对已完成节点续跑改向（steerCount 累加，产物刷新）。
  DagNodeSteer(id: string, nodeId: string, prompt: string): Promise<string>;
  // DagNodeAccept 验收（人拍板，产物回流记忆）→ 节点置 accepted。
  DagNodeAccept(id: string, nodeId: string): Promise<string>;
  // DagAcceptAll 一键验收（成品直出首刀）：run 内全部 done 节点一次置
  // accepted 并逐节点回流记忆（单条失败不阻断，汇总如实上报）。
  DagAcceptAll(id: string): Promise<string>;
  // DagCancel 终止级联：running 节点停推、未跑节点不再起跑。
  DagCancel(id: string): Promise<string>;
  // ── 6.3 余项：流水线模板库（「月度报告」存模板一键重建）──
  // DagTemplateList 全部模板（创建时间倒序）；DagTemplateSave 把既有 run 存为
  // 模板（name 空回退 goal 截断）；DagTemplateNew 一键重建=全新草稿 run（不自动
  // 起跑，起跑仍是人拍板）；DagTemplateDelete 删档即弃（已重建 run 不受影响）。
  DagTemplateList(): Promise<DagTemplateView[]>;
  DagTemplateSave(runId: string, name: string): Promise<string>;
  DagTemplateNew(templateId: string): Promise<string>;
  DagTemplateDelete(templateId: string): Promise<string>;
}
