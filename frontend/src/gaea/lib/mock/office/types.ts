// mock/office/types.ts — office 域共享类型（P4 结构刀2 拆分自 mock/office.ts）。
// OfficeMethods = OfficeBindings 的 mock 实现契约（Pick 收窄自 AppBindings）；
// 按方法域拆出的分组文件（methods_office/methods_xlsx/schedule）各自用
// Partial<OfficeMethods> 声明自己的方法子集，build.ts 合并为完整 OfficeMethods。
import type { AppBindings } from "../../bridge";

export type OfficeMethods = Pick<
  AppBindings,
  | "ListDir" | "FileSearch" | "Materials" | "WorkspaceSearch"
  | "PinnedMaterials" | "PinMaterial" | "UnpinMaterial" | "SummarizeFile"
  | "TaskTemplates"
  | "ReadFile" | "Preview" | "OpenWorkspacePath"
  | "PptxSlideText" | "PptxApplyEdit"
  | "OfficeEditText" | "DocxApplyEdit" | "DocxAcceptChanges"
  | "XlsxPlanEdit" | "XlsxApplyEdit" | "XlsxSetCell" | "XlsxRecalc" | "XlsxRowOps" | "XlsxColOps"
  | "ScheduleLoad" | "ScheduleSave" | "ScheduleExportXlsx" | "ScheduleImportXlsx" | "ScheduleImportMpp"
  | "ScheduleProjects" | "ScheduleProjectOpen" | "ScheduleProjectCreate" | "ScheduleProjectArchive" | "ScheduleProjectDelete" | "ScheduleProjectCopy"
  | "XlsxChart" | "ZipDeliverables" | "SubagentRuns" | "SubagentTranscript" | "DeliverableRegistry" | "WriteFile"
  | "ExportDeliverable" | "ConvertToPdf" | "CrossEmbed" | "RevealWorkspacePath"
  | "SavePastedImage" | "SaveAttachmentFile" | "AttachmentDataURL"
  | "CaptureScreen" | "RecognizeImage" | "OCRText"
  | "HerdsmanDigitalLife" | "HerdsmanOperations"
  | "PickFiles" | "PickDirectory" | "ReadFileB64" | "SaveFileAs" | "OpenLogsDir"
  | "TaskList" | "TaskCancel" | "TaskKill" | "TaskRetry" | "TaskOutput"
  | "GaeaJournalList" | "VerifyRecord" | "RollbackRecord"
  | "GaeaGitStatus" | "GaeaGitDiff" | "GaeaGitStage" | "GaeaGitUnstage" | "GaeaGitDiscard" | "GaeaGitCommit" | "GaeaGitLog"
  // 批次三a legacy 直调转正（Go OfficeB.GaeaDataBackup*，Gaea 前缀经 mappings 映射）。
  | "DataBackupInfo" | "DataBackupCreate" | "DataBackupRestore"
  | "DataBackupCancel" | "DataBackupRollback" | "DataBackupRestoreResult"
  // 6.3 办公多文件 DAG（Go OfficeB.GaeaDag*，Gaea 前缀经 mappings 映射）。
  | "DagList" | "DagGet" | "DagRun" | "DagNodeRun" | "DagNodeSteer" | "DagNodeAccept" | "DagCancel"
>;