// drift.ts — 编译期绑定面漂移检查（T6-10.3）。
import { bindingNames } from "../bindingNames";
import type { AppBindings } from "./appBindings";
import { gaeaToGaea } from "./mappings";

// ── 编译期绑定面漂移检查（T6-10.3）──────────────────────────────────────
// bindingNames 由 scripts/gen_bindings -names 生成（Go 侧全部导出绑定方法名），
// CI（scripts/check-bindings-drift.ps1）保证它与 Go 侧同步。这里用类型级双向断言
// 校验前端绑定面（AppBindings + gaeaToGaea）与 bindingNames 一致：
//
//   · 方向一 _CheckAppBindingsHasNoStray：AppBindings 声明并经 gaeaToGaea 映射后
//     实际调用的每个 Go 方法名必须真实存在于 bindingNames。Go 侧改名/删除方法，
//     或 gaeaToGaea 映射目标写错（历史上有 6 处：KeepWarm*/PreloadPlan*/AgentMode/
//     SummarizeFile/Subagent*），都会在此报错。
//   · 方向二 _CheckAppBindingsCoversAll：bindingNames 里每个方法名必须被
//     AppBindings 消费（含 gaeaToGaea 映射）或显式排除。Go 侧新增绑定而前端
//     未认领时在此报错（补 AppBindings/gaeaToGaea，或列入下方两个清单并注明理由）。
//
// S2-3「App 绑定面拆分」后 Go 侧方法分为两半：经 AppBindings 消费的 gaea UI
// 绑定面，以及 wailsjsCompat / window.go.app.App.* 直接调用的 legacy 绑定面
// （小说/聊天/语音/绘图/角色库/旧 store 等），后者全部列入 LegacySurfaceNames。
// 修复提示：Go 方法新增/改名/删除 → 先重新生成 bindingNames.ts，再按报错调整。
/** 泛型工具：T 必须是 never（空联合），否则编译错误。 */
type AssertNever<T extends never> = T;

/** Go 侧全部绑定方法名（与 bindingNames.ts 同步）。 */
type BindingName = (typeof bindingNames)[number];

/** AppBindings 声明的方法经 gaeaToGaea 映射后在 Go 侧实际调用的方法名集合。 */
type AppBindingTarget = {
  [K in keyof AppBindings]: K extends keyof typeof gaeaToGaea
    ? (typeof gaeaToGaea)[K]
    : K;
}[keyof AppBindings];

/** AppBindings mock-only：Go 侧无对应绑定方法（仅 dev mock 提供）。 */
type MockOnlyNames =
  | "Compact" // 无 Go 绑定；上下文压缩由后端会话事件驱动，无独立绑定
  | "SetSubagentTemperature" // 声明但 Go 侧从未实现（仅 GaeaSetSubagentEffort 存在）
  | "SetEffort" // 同上：Go 侧无 SetEffort，推理强度实际走 GaeaSetSubagentEffort
  | "SetSubagentModel"; // 同上：Go 侧无 SetSubagentModel，实际走 GaeaSetSubagentModelForSkill

/** legacy 绑定面：Go 侧存在但不经 AppBindings 消费（wailsjsCompat 直接调用）。 */
type LegacySurfaceNames =
  | "AddCustomEngine"
  | "AddOutlineNode"
  | "AnalyzeChapter"
  | "AnalyzeStyle"
  | "ApplyBranch"
  | "BrainCrossRefs"
  | "BrainSearch"
  | "BrainWrite"
  | "BrainstormBranches"
  | "BuildBacklinkIndex"
  | "BuildContextBudget"
  | "BuildRichContext"
  | "CancelCreateChapter"
  | "CharacterAssociate"
  | "CharacterAssociateTo"
  | "CharacterDelete"
  | "CharacterDissociate"
  | "CharacterDrawRandom"
  | "CharacterFillAll"
  | "CharacterGenerateFill"
  | "CharacterGeneratePortrait"
  | "CharacterGeneratePortraitWithRef"
  | "CharacterGenerateRandom"
  | "CharacterGet"
  | "CharacterImportProject"
  | "CharacterListByProject"
  | "CharacterSave"
  | "CharacterSetProjectState"
  | "CharacterSyncProject"
  | "Chat"
  | "ChatCharacter"
  | "ChatCharacterDetail"
  | "ChatGeneral"
  | "ChatOutline"
  | "ChatOutlineNode"
  | "CloseProject"
  | "CmdKEdit"
  | "ContinueOutline"
  | "CreateChapter"
  | "CreateProject"
  | "CreateSnapshot"
  | "DeleteCharacter"
  | "DeleteLorebookEntry"
  | "DeleteOrganization"
  | "DeleteOutlineNode"
  | "DeleteProject"
  | "DeleteRelationship"
  | "ExpandOutlineNode"
  | "ExportHTML"
  | "ExtractCharacterHeatmap"
  | "ExtractEmotionCurve"
  | "ExtractTimeline"
  | "FindLorebookTriggers"
  | "FindUnlinkedMentions"
  | "GaeaBenchmarkDetail"
  | "GaeaBenchmarkExport"
  | "GaeaBenchmarkList"
  | "GaeaBenchmarkStart"
  | "GaeaBenchmarkStreamProbe"
  | "GaeaCallTool"
  | "GaeaDataBackupCancel"
  | "GaeaDataBackupCreate"
  | "GaeaDataBackupInfo"
  | "GaeaDataBackupPending"
  | "GaeaDataBackupRestore"
  | "GaeaDataBackupRestoreResult"
  | "GaeaDataBackupRollback"
  | "GaeaEngines"
  | "GaeaGetUsdCnyRate"
  | "GaeaInit"
  | "GaeaModel"
  | "GaeaPermLevel"
  | "GaeaSemanticIndexStatus"
  | "GaeaSetEngine"
  | "GaeaSetUsdCnyRate"
  | "GaeaSkills"
  | "GaeaTools"
  | "GaeaUsageOverview"
  | "GenerateCharacterPortrait"
  | "GenerateCharacters"
  | "GenerateDefaultCanvas"
  | "GenerateOutlineWithDialogue"
  | "GenerateProjectCharacterFill"
  | "GenerateSingleCharacter"
  | "GetActiveASRModel"
  | "GetActiveEngine"
  | "GetActiveOCRModel"
  | "GetActiveTTSModel"
  | "GetAllEntityNames"
  | "GetBacklinks"
  | "GetBookData"
  | "CheckModuleIntegrity" // 3.0 Step 2：板块装配启动自检（Startup 内部调用，前端不经 AppBindings 消费）
  | "GetChapter"
  | "GetChapterBranch"
  | "GetChapterScenes"
  | "GetChatVoiceModel"
  | "GetCompileTemplates"
  | "GetDashboard"
  | "GetDeepseekKeyStatus"
  | "GetGlmKeyStatus"
  | "GetEngineList"
  | "GetEngines"
  | "GetEngineFailover"
  | "GetEntityRelations"
  | "GetImageBackend"
  | "GetImageBackendConfig"
  | "GetLoginStatus"
  | "GetLorebookEntries"
  | "GetModelCallStats"
  | "GetModelHubKeyStatus" // Model Hub（Unsloth 本地引擎）Key 状态（引擎管理经 App() 直调）
  | "GetModelMonitor"
  | "GetModelRoute"
  | "GetNovelsDir"
  | "GetOfflineMode"
  | "GetOfficeLocal"
  | "GetOpencodeGoKeyStatus"
  | "GetOpencodeZenKeyStatus"
  | "GetOutlines"
  | "GetProjectInfo"
  | "GetSensitiveLocal"
  | "GetStyleProfile"
  | "GetTTSConfig"
  | "GetTTSSpeakers"
  | "GetTTSStatus"
  | "GetVoicePipelineConfig"
  | "GetWorldMapImage"
  | "HerdsmanHealth"
  | "HerdsmanLaunchPresets"
  | "HerdsmanModelCatalog"
  | "HerdsmanModelDownload"
  | "HerdsmanModelStart"
  | "HerdsmanModelStats"
  | "HerdsmanModelStop"
  | "HerdsmanModelUninstall"
  | "HerdsmanProbe"
  | "HerdsmanSecurityCheck"
  | "ImportNovelBook"
  | "ImportStyleProfile"
  | "InjectMemories"
  | "IsProjectV4"
  | "ListProjects"
  | "ListSnapshots"
  | "LocalTranslate"
  | "Login"
  | "Logout"
  | "MainBrainChat"
  | "MergeCharacters"
  | "MigrateProjectToV4"
  | "OfficeCancelJob"
  | "OfficeExecute"
  | "OfficeGetJobState"
  | "OfficeGetMode"
  | "OfficeIsTask"
  | "OfficeListFolder"
  | "OfficeReadFile"
  | "OfficeSetMode"
  | "OpenProject"
  | "ParseLinks"
  | "QueryEntities"
  | "QuickBrainstormBranches"
  | "RefreshEngineModels"
  | "RemoveCustomEngine"
  | "ReorderScenes"
  | "ResetModelCallStats"
  | "RestoreSnapshot"
  | "ReviewBook"
  | "RunModule"
  | "SaveChapterBranchContent"
  | "SaveChapterContent"
  | "SaveCharacter"
  | "SaveCharacters"
  | "SaveEngine"
  | "SaveLorebookEntry"
  | "SaveOrganization"
  | "SaveOutlineNode"
  | "SaveRelationship"
  | "SaveScene"
  | "SaveTTSConfig"
  | "SaveToken"
  | "SaveWorldMapImage"
  | "SaveWorldviewSection"
  | "Search"
  | "SearchMemories"
  | "SetActiveASRModel"
  | "SetActiveEngine"
  | "SetActiveOCRModel"
  | "SetActiveTTSModel"
  | "SetChatVoiceModel"
  | "SetDeepseekKey"
  | "SetGlmEndpoint"
  | "SetGlmKey"
  | "SetModelHubKey" // Model Hub（Unsloth 本地引擎）Key（Unsloth 设置 → API 创建）
  | "StartModelHubModel" // Model Hub：让 Unsloth Studio 加载/切换模型（ollama-manifest 引用）
  | "SetDistFS"
  | "SetEngineDefaultModel"
  | "SetEngineFailover"
  | "SetOfflineMode"
  | "SetOfficeLocal"
  | "SetOpencodeGoKey"
  | "SetOpencodeZenKey"
  | "SetPromptFS"
  | "SetSensitiveLocal"
  | "Shutdown"
  | "StartTTSServer"
  | "Startup"
  | "StopTTSServer"
  | "SyncEntityDB"
  | "TTSSpeak"
  | "TTSSpeakBase64"
  | "TTSSpeakBase64WithParams" // v4.3d：带风格/情绪参数合成（wailsjsCompat 直调）
  | "TTSSpeakStreaming"
  | "TestEngineConnection"
  | "ToggleOrgMember"
  | "VoiceCancelTTS"
  | "VoiceGetState"
  | "VoiceHealth"
  | "VoicePlaybackDone"
  | "VoicePushAudio"
  | "VoiceRestartService"
  | "VoiceSetInputChannel"
  | "VoiceSetMode"
  | "VoiceSetPTTActive"
  | "VoiceStart"
  | "VoiceStop"
  | "WhisperChat"
  | "WhisperChatWithSearch"
  | "WhisperClearSession"
  | "WhisperGetConfig"
  | "WhisperGetEngine"
  | "WhisperGetEngines"
  | "WhisperGetImageModel"
  | "UpdateCustomEngine"
  | "WhisperGetModel"
  | "WhisperSetEngine"
  | "WhisperSetImageModel"
  | "WhisperSetModel"
  | "WhisperTaskPlanResume"
  | "WhisperTaskPlanStatus"
  | "WhisperWebSearch"
  // ── v4.7x 小说革命：场景级生成 + 叙事状态结算（wailsjsCompat 直调，未经 AppBindings）──
  | "CreateScene"
  | "GenerateScene"
  | "BuildNovelStatePatch"
  | "GetNovelState"
  | "SettleNovelState"
  | "DeSlopChapterAiTaste"
  | "RewriteChapterAiTaste"
  | "RunChapterGate";

/** 显式排除 = mock-only + legacy 绑定面。 */
type ExcludeNames = MockOnlyNames | LegacySurfaceNames;

// 方向一：AppBindings 声明的每个绑定（映射后）必须真实存在于 Go 绑定清单。
// 报错 → Go 侧方法被改名/删除，或 gaeaToGaea 映射目标写错。
export type _CheckAppBindingsHasNoStray = AssertNever<
  Exclude<AppBindingTarget, BindingName | ExcludeNames>
>;

// 方向二：Go 绑定清单的每个方法名必须被 AppBindings 消费或显式排除。
// 报错 → Go 侧新增绑定无人认领（补 AppBindings/gaeaToGaea 或 ExcludeNames）。
export type _CheckAppBindingsCoversAll = AssertNever<
  Exclude<BindingName, AppBindingTarget | ExcludeNames>
>;

