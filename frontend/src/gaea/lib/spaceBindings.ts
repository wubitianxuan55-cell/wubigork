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
  Models: "shared",
  SetModel: "shared",
  ModelSwitchEstimate: "shared",
  Settings: "shared",
  SetDefaultModel: "shared",
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

  // ── play：乐园数据面（轻语聊天记忆）──────────────────────────
  WhisperMemories: "play",
  WhisperEpisodes: "play",
  WhisperExportArchive: "play",
  // v4.3 会客厅：关系图谱/主动关心（轻语数据面）；书封生成（创作间 play）
  WhisperGraphSubgraph: "play",
  WhisperProactiveNow: "play",
  GenerateBookCover: "play",
  // v4.3 情感语音：TTS 参数预览为 shared（语音朗读两空间共用）
  TTSVoiceParams: "shared",
  TTSSpeakBase64WithParams: "shared",

  // ── independent：编程 DSH 独立窗口 ──────────────────────────
  GetProgrammingWebStatus: "independent",
  StartProgrammingWeb: "independent",
  StopProgrammingWeb: "independent",
  GetProgrammingWebPreflight: "independent",
  ProgrammingWebLogTail: "independent",

  // ── work：gaea 工位工作台（会话/办公/记忆/知识库/造价/任务/文件）──
  Submit: "work",
  SubmitDisplay: "work",
  Cancel: "work",
  Steer: "work",
  GaeaRunning: "work",
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
  OfficeEditText: "work",
  DocxApplyEdit: "work",
  DocxAcceptChanges: "work",
  XlsxPlanEdit: "work",
  XlsxApplyEdit: "work",
  XlsxSetCell: "work",
  XlsxRecalc: "work",
  XlsxRowOps: "work",
  XlsxColOps: "work",
  XlsxChart: "work",
  ZipDeliverables: "work",
  SubagentRuns: "work",
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
  MemorySuggestions: "work",
  AcceptMemorySuggestion: "work",
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
  CostNoteSave: "work",
  CostNoteList: "work",
  CostNoteDelete: "work",
  CostNoteBumpRef: "work",
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
  TaskList: "work",
  TaskCancel: "work",
  TaskRetry: "work",
  TaskOutput: "work",
  // v4.1 证据链：Journal 读取属工位（前端「证据」入口，DeliverablesPanel）。
  GaeaJournalList: "work",
  VerifyRecord: "work",
  RollbackRecord: "work",
  DocumentLint: "work",
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
