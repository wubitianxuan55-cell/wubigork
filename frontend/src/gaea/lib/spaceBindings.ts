// spaceBindings.ts — S2.3 bridge 分面（work/play/shared）绑定空间分类表
// （docs/gaea-space-shell-design.md §7 S2.3）
//
// 单一注册表：gaea 桥 AppBindings 全部方法 → 空间归属。导航/搜索/守卫/门面
// 代理全部从这里派生（三层防线：导航层 manifest.space → 事件层 subscribeForSpace
// → 调用层本表）。规则：
//   - work：工位工作台（会话/办公/记忆/知识库/造价/任务/文件/截图 OCR）；
//   - play：乐园数据面（轻语聊天记忆）；
//   - shared：两空间共用基础设施（元信息/更新/遥测/空间控制/模型与设置/对话/
//     统一检索——UnifiedSearch 的隔离由 scope 参数承担，搜索面板两空间都会用）；
//   - independent：编程 DSH 独立窗口（用户拍板：不并入工位、不共享工具面；
//     壳层独立入口两空间均可达）。
// 全部方法必须显式分类（satisfies Record<keyof AppBindings, BindingSpace>），
// 新方法未分类 tsc 直接报错，防止静默落入错误空间。
import type { AppBindings } from "./bridge";
import type { ShellSpace } from "../../boards/space";

export type BindingSpace = ShellSpace | "shared" | "independent";

/** gaea 桥方法空间归属（键名 = AppBindings 短方法名，经 gaeaToGaea 映射到 Go）。 */
export const GAEA_METHOD_FACETS = {
  // ── shared：两空间共用基础设施 ──────────────────────────────
  Meta: "shared",
  Version: "shared",
  CheckUpdate: "shared",
  ApplyUpdate: "shared",
  OpenDownloadPage: "shared",
  LogFrontendError: "shared",
  SaveWindowState: "shared",
  GaeaSpaceList: "shared",
  GaeaSpaceActive: "shared",
  GaeaSpaceActivate: "shared",
  GaeaSpaceProfiles: "shared",
  // v4.171 批次一 legacy 直调转正：应用信息/板块清单（壳层元信息，两空间共用，
  // AboutPanel/主页 launcher 消费）。
  GetAppInfo: "shared",
  GetBoardManifests: "shared",
  Models: "shared",
  SetModel: "shared",
  ModelSwitchEstimate: "shared",
  // v4.171 批次一 legacy 直调转正：功能级模型绑定读写 + 活跃模型（模型与设置域，
  // useFeatureModel / useBindState / 模型中心 / api/settings.ts 两空间共用）。
  GetFeatureModel: "shared",
  GetFeatureModelEnabled: "shared",
  SetFeatureModel: "shared",
  SetFeatureModelEnabled: "shared",
  GetActiveModel: "shared",
  Settings: "shared",
  SetDefaultModel: "shared",
  // v4.171 批次一：办公引擎设置整体写回（OfficePanel「保存」，同 Settings 归 shared）。
  SaveSettings: "shared",
  SaveProvider: "shared",
  DeleteProvider: "shared",
  LoginProvider: "shared",
  LogoutProvider: "shared",
  SetProviderKey: "shared",
  SetPermissionMode: "shared",
  AddPermissionRule: "shared",
  RemovePermissionRule: "shared",
  SetSandbox: "shared",
  SetAgentParams: "shared",
  SetSubagentTemperature: "shared",
  SetEffort: "shared",
  SetSubagentEffort: "shared",
  SetSubagentModel: "shared",
  SetSubagentModelForSkill: "shared",
  SetPermLevel: "shared",
  KeepWarmGet: "shared",
  KeepWarmSet: "shared",
  PreloadPlanGet: "shared",
  PreloadPlanSet: "shared",
  HerdsmanDigitalLife: "shared",
  HerdsmanOperations: "shared",
  ChatTopicsList: "shared",
  ChatMessagesList: "shared",
  ChatAppendMessages: "shared",
  // 统一检索：隔离由 scope 参数承担（S1.2-C），搜索面板两空间都会调用 → shared
  UnifiedSearch: "shared",
  // v4.7 S4.6 命令面板接统一意图路由：触点层内核两空间共用（路线图 §10.4a
  // 「路由内核是触点层内核（两空间共用家底）」），dry-run 预览-确认制防误触发
  RouteIntent: "shared",
  // ── 图像域 legacy 直调族转正（v4.102）：绘梦/图示消费面=play；角色数据
  // （GetCharacters/SetCharacterPortrait）跨画室参考槽（play）与青鸟/角色域
  // （work）两用 → shared。分类锁同步见 spaceBindings.test.ts。──
  GenerateFreeImage: "play",
  CancelImageGeneration: "play",
  GenerateMedia: "play",
  GenerateDiagram: "play",
  GetImageBackendInfo: "play",
  GetPortraitConfig: "play",
  SetPortraitConfig: "play",
  GetComfyUIStatus: "play",
  GetComfyUILoras: "play",
  GetComfyUITaskProgress: "play",
  StartComfyUI: "play",
  WarmComfyUI: "play",
  StopComfyUI: "play",
  GetSystemStats: "play",
  OpenImageSaveDir: "play",
  OpenNovelImagesDir: "play",
  GetCharacters: "shared",
  SetCharacterPortrait: "shared",
  // 图像域登记只读视图（T1 画室素材库）：space 由参数承担、两空间各自 ledger，
  // 消费面绘梦（play）与后续工位配图（work）都会调 → shared（转正自 Legacy 直调）。
  ImageHubAssets: "shared",
  // 项目章节配图清单（chapter-art.json）：小说域（play）专属消费。
  ChapterArtList: "play",

  // ── play：乐园数据面（轻语聊天记忆）──────────────────────────
  WhisperMemories: "play",
  WhisperEpisodes: "play",
  WhisperEpisodeReplay: "play",
  WhisperAnchors: "play",
  WhisperAnchorReplay: "play",
  WhisperMemoryRetell: "play",
  WhisperCausalExplain: "play",
  WhisperExportArchive: "play",
  // v4.3 会客厅：关系图谱/主动关心（轻语数据面）；书封生成（创作间 play）
  WhisperGraphSubgraph: "play",
  WhisperProactiveNow: "play",
  // v4.3c 后续小步：主动关心定时推送配置（频控/时窗，play 数据面）
  WhisperProactiveConfig: "play",
  WhisperProactiveSetConfig: "play",
  // v4.171 批次一 legacy 直调转正：轻语读取/写入族（角色记忆面板 CharacterMemoryModal
  // 与轻语记忆 WhisperMemoryModal 消费，同 WhisperMemories 归 play 数据面）。
  WhisperGetState: "play",
  WhisperGetFacts: "play",
  WhisperGetTraces: "play",
  WhisperDeleteFact: "play",
  WhisperUpdateFact: "play",
  GenerateBookCover: "play",
  // NovelSearch 章内全文检索（ChapterPage 小说创作间消费，同书封归 play）。
  NovelSearch: "play",
  // 批次二 legacy 直调转正（NovelB 门面）：世界观读写/一致性检查/伏笔读写/
  // 批量角色/阅读助手/场景插图——小说创作间数据面，同书封/NovelSearch 归 play。
  GetWorldview: "play",
  SaveWorldview: "play",
  ChatWorldview: "play",
  GetWorldviewSections: "play",
  SaveAllWorldviewSections: "play",
  CheckConsistency: "play",
  CheckConsistencyDeep: "play",
  GetForeshadows: "play",
  SaveForeshadows: "play",
  SaveCharactersBatch: "play",
  NovelReadingAsk: "play",
  GenerateSceneIllustration: "play",
  // 批次二：语音对话文本入口（VoiceB 门面），轻语/语音聊天 play 数据面。
  VoiceChatText: "play",
  // 批次二：绘梦后端配置（SetImageBackend），同 GetImageBackendInfo/
  // SetPortraitConfig 归 play。
  SetImageBackend: "play",
  // 批次三a legacy 直调转正（Go ImageB bindings_image.go:30/32）：TTS 音色列表/
  // 语音管线配置读取——模型配置读，模型中心/语音设置两空间共用 → shared。
  GetTTSSpeakers: "shared",
  GetVoicePipelineConfig: "shared",
  // 批次四 bridge 双轨退役终局（Go ImageB bindings_image.go:37/38/40）：语音模型
  // 设置写口（SetActiveASRModel/SetActiveTTSModel 激活识别/合成模型、
  // SetChatVoiceModel 功能绑定聊天语音）——同 GetTTSSpeakers/GetVoicePipelineConfig
  // 模型配置读写面，模型中心/语音设置两空间共用 → shared。
  SetActiveASRModel: "shared",
  SetActiveTTSModel: "shared",
  SetChatVoiceModel: "shared",
  // v4.3 情感语音：TTS 参数预览为 shared（语音朗读两空间共用）
  TTSVoiceParams: "shared",
  // 批次三a legacy 直调转正：TTS base64 合成入口同 TTSVoiceParams 归 shared
  // （语音朗读两空间共用；WhisperClearSession 是轻语会话运行时清空，
  // 同 WhisperGetState/GetFacts 等轻语数据面归 play；语音运行时启停族归 play）。
  TTSSpeakBase64: "shared",
  TTSSpeakBase64WithParams: "shared",
  WhisperClearSession: "play",
  VoiceStart: "play",
  VoiceStop: "play",
  VoicePlaybackDone: "play",
  VoiceCancelTTS: "play",
  VoicePushAudio: "play",
  VoiceSetPTTActive: "play",
  // 批次四 bridge 双轨退役终局（Go VoiceB bindings_voice.go:27）：语音服务健康
  // 状态（语音设置面板「检测」按钮消费）——语音运行时状态面，同启停族归 play。
  VoiceHealth: "play",
  // v4.171 批次一 legacy 直调转正：语音设置应用（useChatVoice/语音设置面板/
  // 模型中心 BindSection）+ 本地 TTS 服务启动（模型中心「启动」按钮）——
  // 语音朗读/模型设置两空间共用，同 TTSVoiceParams 归 shared。
  VoiceApplySettings: "shared",
  // 批次二 legacy 直调转正：语音设置读取（VoiceGetSettings）与
  // VoiceApplySettings patch 写配对，同归 shared（语音设置面板/模型中心两空间共用）。
  VoiceGetSettings: "shared",
  StartLocalTTSService: "shared",

  // ── independent：编程 DSH 独立窗口 ──────────────────────────
  GetProgrammingWebStatus: "independent",
  StartProgrammingWeb: "independent",
  StopProgrammingWeb: "independent",
  GetProgrammingWebPreflight: "independent",
  ProgrammingWebLogTail: "independent",

  // ── work：gaea 工位工作台（会话/办公/记忆/知识库/造价/任务/文件）──
  // v4.4 微信触点（书房·离线代办）：扫码绑定/通道状态/助手管理/提醒管理，全部 work。
  WhisperWeixinGetQR: "work",
  WhisperWeixinQRStatus: "work",
  WhisperWeixinQRStatusWithCode: "work",
  WhisperWeixinStatus: "work",
  WhisperAssistantList: "work",
  WhisperAssistantSave: "work",
  WhisperAssistantDelete: "work",
  // v4.48 青鸟人格选择器（WeixinPage 经 bridge 消费，转正出 legacy 名单）。
  WhisperGetPersonalities: "work",
  CharacterList: "work",
  WeixinReminderList: "work",
  WeixinReminderAdd: "work",
  WeixinReminderDelete: "work",
  WeixinReminderConfig: "work",
  WeixinReminderSetConfig: "work",
  // 批次二 wailsjsCompat 双轨退役转正（CoreB/ChatB 门面）：配置/统计/技能/
  // 导出归工位数据面；对话板块话题管理（create/delete/rename/setmode/import/
  // send/stream/clear/export-markdown）同批转正归 work（现有唯读三列
  // ChatTopicsList/MessagesList/AppendMessages 为 shared 基础设施，如需对齐随后续批次调整）。
  GetConfig: "work",
  SaveConfig: "work",
  GetStats: "work",
  ListSkills: "work",
  ExportAll: "work",
  ChatTopicCreate: "work",
  ChatTopicDelete: "work",
  ChatTopicRename: "work",
  ChatTopicSetMode: "work",
  ChatImportTopic: "work",
  ChatSend: "work",
  ChatStreamPlain: "work",
  ChatTopicClear: "work",
  ChatTopicExportMarkdown: "work",
  Submit: "work",
  SubmitDisplay: "work",
  Cancel: "work",
  Steer: "work",
  GaeaRunning: "work",
  SubagentFollowUp: "work",
  Approve: "work",
  AnswerQuestion: "work",
  Compact: "work",
  NewSession: "work",
  Reload: "work",
  History: "work",
  Checkpoints: "work",
  Rewind: "work",
  Fork: "work",
  SummarizeFrom: "work",
  SummarizeUpTo: "work",
  ListSessions: "work",
  ListProjectSessions: "work",
  ResumeSession: "work",
  SessionStats: "work",
  ArchiveSession: "work",
  UnarchiveSession: "work",
  PinSession: "work",
  DeleteSession: "work",
  RenameSession: "work",
  ListWorkspaces: "work",
  PickWorkspace: "work",
  SwitchWorkspace: "work",
  ContextUsage: "work",
  ContextView: "work",
  Trajectory: "work",
  AgentNetwork: "work",
  TCCAReport: "work",
  Balance: "work",
  Jobs: "work",
  FactBase: "work",
  FactBaseClear: "work",
  FactBasePromote: "work",
  CaptureSkill: "work",
  Commands: "work",
  Capabilities: "work",
  AddMCPServer: "work",
  RemoveMCPServer: "work",
  RetryMCPServer: "work",
  SetMCPServerEnabled: "work",
  SlashArgs: "work",
  ListDir: "work",
  FileSearch: "work",
  Materials: "work",
  WorkspaceSearch: "work",
  PinnedMaterials: "work",
  PinMaterial: "work",
  UnpinMaterial: "work",
  SummarizeFile: "work",
  TaskTemplates: "work",
  ReadFile: "work",
  Preview: "work",
  PptxOutline: "work",
  PromoteSubagent: "work",
  GaeaBrowserObserve: "work",
  OfficeEditText: "work",
  DocxApplyEdit: "work",
  PptxApplyEdit: "work",
  PptxSlideText: "work", // v4.156 pptx 真编辑刀2：每页段落全文（编辑面板取数）
  DocxAcceptChanges: "work",
  XlsxPlanEdit: "work",
  XlsxApplyEdit: "work",
  XlsxSetCell: "work",
  XlsxRecalc: "work",
  XlsxRowOps: "work",
  XlsxColOps: "work",
  XlsxChart: "work",
  ScheduleLoad: "work",
  ScheduleSave: "work",
  ScheduleExportXlsx: "work",
  ScheduleImportXlsx: "work",
  ScheduleImportMpp: "work",
  // v4.139 #15 进度计划多工程（列表/切换/新建/归档/删除）：进度计划板块数据面。
  ScheduleProjects: "work",
  ScheduleProjectOpen: "work",
  ScheduleProjectCreate: "work",
  ScheduleProjectArchive: "work",
  ScheduleProjectDelete: "work",
  ScheduleProjectCopy: "work", // v4.145 工程复制（另存为）
  ZipDeliverables: "work",
  SubagentRuns: "work",
  SubagentTranscript: "work",
  DeliverableRegistry: "work",
  ResyncEvents: "work",
  WriteFile: "work",
  ExportDeliverable: "work",
  ConvertToPdf: "work",
  CrossEmbed: "work",
  OpenWorkspacePath: "work",
  RevealWorkspacePath: "work",
  SavePastedImage: "work",
  SaveAttachmentFile: "work",
  AttachmentDataURL: "work",
  CaptureScreen: "work",
  RecognizeImage: "work",
  OCRText: "work",
  Memory: "work",
  Remember: "work",
  Forget: "work",
  SaveDoc: "work",
  UpdateFact: "work",
  ChangeFactType: "work",
  SetMemoryEnabled: "work",
  MorningPreload: "work",
  SetMorningPreload: "work",
  MemorySuggestions: "work",
  AcceptMemorySuggestion: "work",
  AcceptMergeSuggestion: "work",
  AcceptSkillSuggestion: "work",
  KnowledgeList: "work",
  KnowledgeSearch: "work",
  MemoryHubOverview: "work",
  ProfileList: "work",
  ProfileSave: "work",
  ProfileDelete: "work",
  ProfileConflicts: "work",
  PickDirectory: "work",
  MemoryGraph: "work",
  // 晨报（做梦 2.0 主动预取）：只读 work 空间记忆，play 不渲染（双空间红线）。
  MemoryMorningBrief: "work",
  CostList: "work",
  CostSearch: "work",
  CostGet: "work",
  CostSave: "work",
  CostDelete: "work",
  CostImportPreview: "work",
  CostImportAIParse: "work",
  CostImportApply: "work",
  CostImportVisionPreview: "work",
  CostCategories: "work",
  CostCategorySave: "work",
  CostCategoryDelete: "work",
  PriceSources: "work",
  PriceSourceSave: "work",
  PriceSourceDelete: "work",
  PriceFetch: "work",
  PriceFetchAll: "work",
  PriceFetches: "work",
  PriceFetchApply: "work",
  PriceFetchIgnore: "work",
  PriceHistory: "work",
  SemanticSearch: "work",
  CostCompare: "work",
  CostCompose: "work",
  CostComposeApply: "work",
  CostComposeRecords: "work", // v4.158 组价复核闭环：确认记录回看（条目详情）
  CostInquirySave: "work",
  CostInquiryList: "work",
  CostInquiryDelete: "work",
  CostInquiryExpiring: "work",
  CostInquiryAdjust: "work",
  CostStageSave: "work",
  CostStages: "work",
  CostStageCompare: "work",
  CostStageDeviations: "work",
  CostProjectSave: "work",
  CostProjectList: "work",
  CostProjectGet: "work",
  CostProjectDelete: "work",
  CostEstimateItemSave: "work",
  CostEstimateItemDelete: "work",
  CostEstimateItems: "work",
  CostEstimateVersionSave: "work",
  CostEstimateVersions: "work",
  CostEstimateSediment: "work",
  CostIndicators: "work",
  CostAttribution: "work",
  CostNoteSave: "work",
  CostNoteList: "work",
  CostNoteDelete: "work",
  CostNoteBumpRef: "work",
  CostGraph: "work",
  RetrievalEvalRun: "work",
  KnowledgeImportPreview: "work",
  KnowledgeImportAIParse: "work",
  KnowledgeImportApply: "work",
  KnowledgeHistory: "work",
  KnowledgeFindSimilar: "work",
  KnowledgeExport: "work",
  KnowledgeReview: "work",
  KnowledgeMerge: "work",
  MemoryDuplicates: "work",
  MemoryMerge: "work",
  MemoryArchivedList: "work",
  MemoryCleanupArchived: "work",
  MemoryUnarchive: "work",
  MemoryUnarchiveBatch: "work",
  MemorySetRetentionDays: "work",
  FileIndexRebuild: "work",
  FileSemanticSearch: "work",
  ProfileResolveConflict: "work",
  KnowledgeGet: "work",
  KnowledgeSave: "work",
  KnowledgeDelete: "work",
  PickFiles: "work",
  ReadFileB64: "work",
  SaveFileAs: "work",
  OpenLogsDir: "work",
  TaskList: "work",
  TaskCancel: "work",
  TaskKill: "work",
  TaskRetry: "work",
  TaskOutput: "work",
  ContextNodeDetail: "work",
  // v4.1 证据链：Journal 读取属工位（前端「证据」入口，DeliverablesPanel）。
  GaeaJournalList: "work",
  // 2b Git 面板最小集（工位工作台文件域；D3：单仓库无 push）。
  GaeaGitStatus: "work",
  GaeaGitDiff: "work",
  GaeaGitStage: "work",
  GaeaGitUnstage: "work",
  GaeaGitDiscard: "work",
  GaeaGitCommit: "work",
  GaeaGitLog: "work",
  GaeaSubagentContextView: "work",
  VerifyRecord: "work",
  RollbackRecord: "work",
  DocumentLint: "work",
  // 批次三a legacy 直调转正（Go OfficeB.GaeaDataBackup*）：数据备份/恢复属
  // 设置域（DataPanel 工位消费；同 SaveSettings 归属精神但落盘是工作台动作）→ work。
  DataBackupInfo: "work",
  DataBackupCreate: "work",
  DataBackupRestore: "work",
  DataBackupCancel: "work",
  DataBackupRollback: "work",
  DataBackupRestoreResult: "work",
  // 批次三a legacy 直调转正（Go CoreB.ListProjects）：书架工程卡片清单
  // （api/characterlib.listShelfProjects 消费面=创作间书架/角色库）→ play。
  ListProjects: "play",
  // 批次三b legacy 直调转正（Go CharlibB/NovelB 门面，同名前缀）：角色库档案/
  // 加入项目/抽卡/补全/剧照族（角色库板块 characterlib 归 play 空间，消费面=
  // CharacterLibraryPage/NewCharactersModal/小说角色面板）→ play。
  CharacterSave: "play",
  CharacterGet: "play",
  CharacterDelete: "play",
  CharacterImportProject: "play",
  CharacterListByProject: "play",
  CharacterAssociate: "play",
  CharacterAssociateTo: "play",
  CharacterSetProjectState: "play",
  CharacterDissociate: "play",
  CharacterSyncProject: "play",
  CharacterDrawRandom: "play",
  CharacterGenerateFill: "play",
  CharacterGenerateRandom: "play",
  CharacterFillAll: "play",
  CharacterGeneratePortrait: "play",
  CharacterGeneratePortraitWithRef: "play",
  // 批次三b legacy 直调转正（Go NovelB 门面，同名前缀）：章节族/叙事状态族/
  // 场景族/项目角色族——小说创作间数据面（ChapterPage/CreatePage/小说角色面板
  // 消费），同书封/NovelSearch/批次二 NovelB 族归 play。
  GetChapter: "play",
  GetChapterBranch: "play",
  SaveChapterContent: "play",
  SaveChapterBranchContent: "play",
  // 批次三c legacy 直调转正（CreatePage 收尾）：分支构思/创建章节/大纲节点删除
  // （Go NovelB.QuickBrainstormBranches/CreateChapter/DeleteOutlineNode）——
  // 创作间消费面，同章节族归 play。
  QuickBrainstormBranches: "play",
  CreateChapter: "play",
  DeleteOutlineNode: "play",
  GetNovelState: "play",
  BuildNovelStatePatch: "play",
  SettleNovelState: "play",
  DeSlopChapterAiTaste: "play",
  RewriteChapterAiTaste: "play",
  GetEntityRelations: "play",
  GetChapterScenes: "play",
  GenerateScene: "play",
  CreateScene: "play",
  CancelCreateChapter: "play",
  GenerateProjectCharacterFill: "play",
  GenerateCharacterPortrait: "play",
  MergeCharacters: "play",
  SaveOrganization: "play",
  DeleteOrganization: "play",
  ToggleOrgMember: "play",
  SaveRelationship: "play",
  DeleteRelationship: "play",
} as const satisfies Record<keyof AppBindings, BindingSpace>;

/** 编译期双向断言（与 bridge 绑定面漂移检查同范式）：分类表不得出现
 *  AppBindings 之外的名字，AppBindings 每个方法必须被显式分类。 */
type AssertNever<T extends never> = T;
export type _NoStrayFacet = AssertNever<
  Exclude<keyof typeof GAEA_METHOD_FACETS, keyof AppBindings>
>;
export type _NoMissingFacet = AssertNever<
  Exclude<keyof AppBindings, keyof typeof GAEA_METHOD_FACETS>
>;

/** 按空间取方法名联合（work/play/shared/independent 四类，供门面类型派生） */
type NamesOf<F extends BindingSpace> = {
  [K in keyof typeof GAEA_METHOD_FACETS]: (typeof GAEA_METHOD_FACETS)[K] extends F ? K : never
}[keyof typeof GAEA_METHOD_FACETS];

export type WorkBindingName = NamesOf<"work">;
export type PlayBindingName = NamesOf<"play">;
export type SharedBindingName = NamesOf<"shared">;
export type IndependentBindingName = NamesOf<"independent">;

/** 三空间门面两两无重叠（编译期）：一个方法只能归属一个空间门面。 */
export type _NoFacetOverlap = AssertNever<
  | Extract<WorkBindingName, PlayBindingName | SharedBindingName>
  | Extract<PlayBindingName, SharedBindingName>
>;

/** 运行时方法名清单（测试/门面代理过滤用，由单一注册表派生）。 */
export const WORK_BINDING_NAMES = Object.keys(GAEA_METHOD_FACETS).filter(
  (k) => GAEA_METHOD_FACETS[k as keyof typeof GAEA_METHOD_FACETS] === "work",
) as WorkBindingName[];
export const PLAY_BINDING_NAMES = Object.keys(GAEA_METHOD_FACETS).filter(
  (k) => GAEA_METHOD_FACETS[k as keyof typeof GAEA_METHOD_FACETS] === "play",
) as PlayBindingName[];
export const SHARED_BINDING_NAMES = Object.keys(GAEA_METHOD_FACETS).filter(
  (k) => GAEA_METHOD_FACETS[k as keyof typeof GAEA_METHOD_FACETS] === "shared",
) as SharedBindingName[];

/** 方法空间归属解析（未知名兜底 work——合法调用面均已被编译期覆盖，运行期仅防呆）。 */
export function bindingSpaceOf(method: string): BindingSpace {
  return (GAEA_METHOD_FACETS as Record<string, BindingSpace>)[method] ?? "work";
}

/** gaea 方法在指定壳层空间是否可调用：shared/independent 两空间可达，其余仅所属空间。 */
export function isBindingAllowedInSpace(method: string, space: ShellSpace): boolean {
  const facet = bindingSpaceOf(method);
  return facet === "shared" || facet === "independent" || facet === space;
}

/** 是否共用绑定（sharedApp 门面只暴露 shared 方法）。 */
export function isSharedBinding(method: string): boolean {
  return bindingSpaceOf(method) === "shared";
}

/**
 * 类型级门面：按壳层空间窄化 gaea 桥方法集（work/play 各自只看到
 * 所属空间 + shared + independent），编译期防呆——play 页面代码引用
 * workApp 上的 work 专属方法会直接 tsc 报错。
 */
type FacetOf<K extends keyof AppBindings> = (typeof GAEA_METHOD_FACETS)[K];
export type GaeaFacetBySpace = {
  [S in ShellSpace]: {
    [K in keyof AppBindings as FacetOf<K> extends S | "shared" | "independent"
      ? K
      : never]: AppBindings[K];
  };
};
