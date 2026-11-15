// office.ts — OfficeBindings（AppBindings 分域接口之一，Go OfficeB 门面）：
// docx/pptx/xlsx 编辑 + 交付导出（图文板块：进度计划/DAG/转 PDF 面已随
// 2026-11-15 刀0 撤除，绑定面收缩）。
import type {
  ExportDeliverableInput,
  ExportDeliverableResult,
  OfficeEditResult,
  PreviewResult,
  SettingsView,
  ZipDeliverableResult,
} from "../types";

export interface OfficeBindings {
  // OfficeEditText 框选即改：按指令生成选中文本的替换；DocxApplyEdit 以修订模式
  // （w:del+w:ins）写入 docx 并返回更新后的预览。
  OfficeEditText(selectedText: string, instruction: string): Promise<OfficeEditResult>;
  DocxApplyEdit(rel: string, selectedText: string, replacement: string): Promise<PreviewResult>;
  // DocxAcceptChanges 接受/拒绝 gaea 的待处理修订，返回更新预览。
  DocxAcceptChanges(rel: string, accept: boolean): Promise<PreviewResult>;
  // ZipDeliverables 会话产物一键打包：把本次会话交付文件打成一个 zip。
  ZipDeliverables(paths: string[]): Promise<ZipDeliverableResult>;
  // WriteFile 工作区内联编辑保存（C5）：把文本原子写回工作区相对路径文本文件
  // （路径/扩展名/大小校验在后端；用户显式保存，不走 agent 审批）。
  WriteFile(rel: string, content: string): Promise<void>;
  // ExportDeliverable 统一交付出口：受控 Markdown → docx/pptx/xlsx/md/pdf。
  ExportDeliverable(input: ExportDeliverableInput): Promise<ExportDeliverableResult>;
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

  // ── 搜索主源设置（众神殿刀3 主源向导）：key 落 ~/.env 与 CLI 同源，保存热生效 ──
  WebSearchSettingsGet(): Promise<Record<string, unknown>>;
  WebSearchSettingsSet(tavilyKey: string, braveKey: string, searxngURL: string): Promise<void>;
  WebSearchTestConnection(): Promise<Record<string, unknown>>;
}
