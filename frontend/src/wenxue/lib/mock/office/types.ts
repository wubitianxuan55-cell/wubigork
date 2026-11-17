// mock/office/types.ts — office 域共享类型（P4 结构刀2 拆分自 mock/office.ts）。
// OfficeMethods = OfficeBindings 的 mock 实现契约（Pick 收窄自 AppBindings）；
// 按方法域拆出的分组文件（methods_office/methods_xlsx）各自用
// Partial<OfficeMethods> 声明自己的方法子集，build.ts 合并为完整 OfficeMethods
// （图文板块：进度计划/DAG/转 PDF 面已随 2026-11-15 刀0 撤除）。
import type { AppBindings } from "../../bridge";

export type OfficeMethods = Pick<
  AppBindings,
  | "ListDir" | "FileSearch" | "Materials" | "WorkspaceSearch"
  | "GaeaListDirBatch" | "GaeaFileTreeWatch"
  | "GaeaRenameWorkspacePath" | "GaeaCreateWorkspaceFolder" | "GaeaCreateWorkspaceFile"
  | "GaeaDeleteWorkspacePaths"
  | "GaeaWorkspaceUploadBegin" | "GaeaWorkspaceUploadAppend"
  | "GaeaWorkspaceUploadFinish" | "GaeaWorkspaceUploadAbort"
  | "GaeaPickWorkspaceUploadFiles" | "GaeaPickWorkspaceUploadFolder"
  | "GaeaReadTextFile" | "GaeaReadFilePage" | "GaeaWriteWorkspaceFile"
  | "GaeaSaveWorkspacePathAs" | "GaeaSaveWorkspacePathsAs"
  | "PinnedMaterials" | "PinMaterial" | "UnpinMaterial" | "SummarizeFile"
  | "TaskTemplates"
  | "ReadFile" | "Preview" | "OpenWorkspacePath"
  | "OfficeEditText" | "DocxApplyEdit" | "DocxAcceptChanges"
  | "ZipDeliverables" | "SubagentRuns" | "SubagentTranscript" | "PromoteSubagent" | "SubagentFollowUp" | "SubagentStop" | "DeliverableRegistry" | "WriteFile"
  // 文档 lint / PPT 大纲（批 46 NOT_MOCKED 收尾刀；Gaea 前缀经 mappings 映射）。
  // 众神殿刀3：搜索主源设置三绑定。
  | "WebSearchSettingsGet" | "WebSearchSettingsSet" | "WebSearchTestConnection"
  | "ExportDeliverable" | "RevealWorkspacePath"
  | "SavePastedImage" | "GalleryList" | "SaveAttachmentFile" | "AttachmentDataURL"
  // 分块上传三段（超大附件过桥形态；Begin 发 token → Append 吞片 → Finish 落名）。
  | "AttachmentChunkBegin" | "AttachmentChunkAppend" | "AttachmentChunkFinish" | "AttachmentChunkAbort"
  | "CaptureScreen" | "RecognizeImage" | "OCRText"
  // v4.524 CAD 图纸卡（Go OfficeB.GaeaCad*，Gaea 前缀经 mappings 映射）。
  | "CadOpen" | "CadRaster" | "CadTexts"
  | "CadCount"
  | "HerdsmanDigitalLife" | "HerdsmanOperations"
  | "PickFiles" | "PickDirectory" | "ReadFileB64" | "SaveFileAs" | "OpenLogsDir"
  | "TaskList" | "TaskCancel" | "TaskKill" | "TaskRetry" | "TaskOutput"
  | "GaeaJournalList" | "VerifyRecord" | "RollbackRecord"
  | "GaeaGitStatus" | "GaeaGitDiff" | "GaeaGitStage" | "GaeaGitUnstage" | "GaeaGitDiscard" | "GaeaGitCommit" | "GaeaGitLog" | "GaeaGitWorktrees" | "GaeaGitGraph" | "GaeaGitCommitDetail"
  // 批次三a legacy 直调转正（Go OfficeB.GaeaDataBackup*，Gaea 前缀经 mappings 映射）。
  | "DataBackupInfo" | "DataBackupCreate" | "DataBackupRestore"
  | "DataBackupCancel" | "DataBackupRollback" | "DataBackupRestoreResult"
>;
