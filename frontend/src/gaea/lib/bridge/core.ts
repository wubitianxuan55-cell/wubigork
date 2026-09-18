// core.ts — CoreBindings（AppBindings 分域接口之一，Go CoreB 门面）：会话/工作区/
// 上下文/文件/子代理/任务/证据/Git/编程进程等核心域（含 LogFrontendError 与更新域）。
import type {
  AgentNetwork,
  BalanceInfo,
  CapabilitiesView,
  CommandInfo,
  ContextInfo,
  ContextNodeDetailView,
  ContextTimeline,
  DeliverableRegistryView,
  DirEntry,
  FactBaseView,
  FilePickResult,
  FilePreview,
  FileSearchHit,
  FileSemanticHit,
  GaeaReloadResult,
  GaeaResyncResult,
  GaeaSummaryResult,
  GitCommitInfoView,
  GitStatusView,
  HistoryMessage,
  IntentResultView,
  JobView,
  JournalChangeRecord,
  LintReportView,
  MCPServerInput,
  Meta,
  PreviewResult,
  ProgrammingWebLogTail,
  ProgrammingWebPreflight,
  ProgrammingWebStatus,
  ProjectGroup,
  QuestionAnswer,
  RetrievalEvalReport,
  SearchScope,
  SessionMeta,
  SessionStatsView,
  SkillCaptureInput,
  SkillCaptureResult,
  SkillDraft,
  SkillDistillView,
  SkillStatView,
  SkillRecordResult,
  SlashArgsResult,
  SpaceActiveView,
  SpaceOption,
  SpaceProfileView,
  SubagentRunsView,
  SubagentTranscriptView,
  TaskInboxView,
  TaskOutputView,
  TaskTemplate,
  TaskView,
  Trajectory,
  UnifiedSearchView,
  UpdateInfo,
  VerdictView,
  WorkspaceSearchHit,
  WorkspaceView,
} from "../types";
// BrowserObserveView 定义在 BrowserPanel.tsx（v4.28 A2，导出供 bridge/测试重用）。
import type { BrowserObserveView } from "../../components/BrowserPanel";

export interface CoreBindings {
  Submit(input: string): Promise<void>;
  SubmitDisplay(display: string, input: string): Promise<void>;
  Cancel(): Promise<void>;
  // Steer 任务运行中插话调整：消息注入当前回合作为补充指引（不打断执行、
  // 不开新回合）；未运行时走 Submit 排队兜底。
  Steer(input: string): Promise<void>;
  // GaeaRunning 返回办公引擎当前是否真的在跑（看门狗校准用）。
  GaeaRunning(): Promise<boolean>;
  Approve(id: string, decision: "allow_once" | "allow_session" | "persist_allow" | "deny" | "abort"): Promise<void>;
  AnswerQuestion(id: string, answers: QuestionAnswer[]): Promise<void>;
  Compact(): Promise<void>;
  NewSession(): Promise<void>;
  // Reload 热加载办公引擎：重新读取磁盘上的持久化配置并重建 controller，
  // 使技能/工具/插件/参数变更无需重启即生效；返回重建后的工具/技能数量。
  Reload(): Promise<GaeaReloadResult>;
  History(): Promise<HistoryMessage[]>;
  Rewind(turn: number, scope: string): Promise<void>;
  Fork(turn: number): Promise<void>;
  SummarizeFrom(turn: number): Promise<void>;
  SummarizeUpTo(turn: number): Promise<void>;
  // Session history: list saved sessions, resume one (returns its transcript),
  // delete one, or give one a custom display name ("" clears it).
  ListSessions(): Promise<SessionMeta[]>;
  // ListProjectSessions returns saved sessions grouped by workspace
  // (current first, then recently opened workspaces with sessions).
  ListProjectSessions(): Promise<ProjectGroup[]>;
  ResumeSession(path: string): Promise<HistoryMessage[]>;
  // SessionStats 返回会话级 token/成本派生统计（事件日志重放；legacy 会话
  // available=false）。恢复会话后调用，回填「全会话成本」缺失的历史部分。
  SessionStats(path: string): Promise<SessionStatsView>;
  // ArchiveSession moves a saved session to <sessions>/archive/ (restorable);
  // UnarchiveSession moves it back and returns the active path;
  // PinSession toggles the pinned flag (pinned sessions sort first).
  ArchiveSession(path: string): Promise<void>;
  UnarchiveSession(path: string): Promise<string>;
  PinSession(path: string, pinned: boolean): Promise<void>;
  DeleteSession(path: string): Promise<void>;
  RenameSession(path: string, title: string): Promise<void>;
  // Workspace: open a folder chooser and switch to that project (fresh session);
  // returns the chosen path, or "" if cancelled.
  ListWorkspaces(): Promise<WorkspaceView[]>;
  PickWorkspace(): Promise<string>;
  SwitchWorkspace(path: string): Promise<string>;
  // 双空间（S4，设计 §6）：work/play 静态枚举、当前生效空间、切换持久化。
  // Activate 只写配置（非法 space 拒绝）；生效时机为下次引擎重建/重启。
  GaeaSpaceList(): Promise<SpaceOption[]>;
  GaeaSpaceActive(): Promise<SpaceActiveView>;
  // 双空间装配 profile 视图（模型中心「总闸/空间策略」分区）。
  GaeaSpaceProfiles(): Promise<SpaceProfileView[]>;
  // 写一处空间 profile（ref 空=清除），返回刷新后视图；生效=下次引擎重建/重启。
  GaeaSpaceProfileSet(space: string, key: string, ref: string): Promise<SpaceProfileView[]>;
  GaeaSpaceActivate(space: string): Promise<SpaceActiveView>;
  ContextUsage(): Promise<ContextInfo>;
  // ContextView 返回会话的上下文构成快照（dsh-context Go 移植 Phase A）：
  // 六分类当前组成、逐请求趋势、上下文事件、模型可见节点与归档。sessionPath
  // 单元素数组可选（Go 变参 → JS 数组，与 Trajectory 同形态）：[path]=按该
  // 会话读取（UI 会话切换语义）；缺省/[]=内核当前会话。
  ContextView(sessionPath: string): Promise<ContextTimeline>;
  // Trajectory 返回会话的轨迹时间线（轮次→步骤→工具调用）。sessionPath
  // 单元素数组可选（Go 变参 → JS 数组，与 TaskList 同形态）：[path]=按该
  // 会话读取（UI 会话切换语义）；缺省/[]=内核当前会话。
  Trajectory(sessionPath: string): Promise<Trajectory>;
  // AgentNetwork 返回会话的 Agent 网络（主 agent 根 + 子代理树）。sessionPath
  // 语义同 Trajectory。
  AgentNetwork(sessionPath: string): Promise<AgentNetwork>;
  // Balance queries the active provider's wallet balance (a network call);
  // returns an unavailable readout when no balance_url is configured or it fails.
  Balance(): Promise<BalanceInfo>;
  // Jobs lists the running background jobs (bash/task started in the background)
  // for the status-bar indicator.
  Jobs(): Promise<JobView[]>;
  // FactBase reads the current conversation's fact base (settled facts before
  // deliverables); FactBaseClear resets it from the sidebar panel.
  FactBase(): Promise<FactBaseView>;
  FactBaseClear(): Promise<void>;
  // FactBasePromote writes the fact base into permanent memory (dedup by key),
  // returning how many facts were saved/updated.
  FactBasePromote(): Promise<number>;
  // CaptureSkill 把一次成功对话沉淀为可复用技能（写入 .gaea/skills + 全局镜像，
  // 成功后热加载引擎）。同名技能允许覆盖。
  CaptureSkill(input: SkillCaptureInput): Promise<SkillCaptureResult>;
  // SkillDraftFromSession 把当前会话回放蒸馏为技能草稿（LLM 结构化，只回不落盘；
  // 落盘走 SkillDraftSave，用户审阅后）。
  SkillDraftFromSession(): Promise<SkillRecordResult>;
  // SkillDraftSave 保存（通常已审阅修订的）技能草稿：渲染→落盘→镜像→热加载。
  SkillDraftSave(draft: SkillDraft): Promise<SkillCaptureResult>;
  // SkillDistill* journal 历史蒸馏（7.2-2）：Candidates 现算重复模式候选（零 LLM
  // 只读）；Draft 按 patternID 从多次执行回放蒸馏草稿（复用 7.2-1 审阅管道，只回
  // 不落盘）；Decide 落用户决定（ignore=静默不再提示 / crystallized=已结晶审计）。
  SkillDistillCandidates(): Promise<SkillDistillView>;
  SkillDistillDraft(patternID: string): Promise<SkillRecordResult>;
  SkillDistillDecide(patternID: string, decision: "ignore" | "crystallized", skillName: string): Promise<void>;
  // SkillStats 技能调用计数（7.2-2 判据②）：read_skill/run_skill 工具级累计，
  // calls 降序只读视图。
  SkillStats(): Promise<SkillStatView[]>;
  Meta(): Promise<Meta>;
  Commands(): Promise<CommandInfo[]>;
  // Capabilities feeds the MCP & Skills drawer: connected/failed servers + skills.
  // Add connects + persists a server; Remove disconnects + drops it from config;
  // Retry reconnects a configured server that failed (config untouched).
  Capabilities(): Promise<CapabilitiesView>;
  AddMCPServer(input: MCPServerInput): Promise<number>;
  RemoveMCPServer(name: string): Promise<void>;
  RetryMCPServer(name: string): Promise<void>;
  // SetMCPServerEnabled is the per-session connector toggle (on reconnects, off
  // disconnects; config untouched).
  SetMCPServerEnabled(name: string, enabled: boolean): Promise<void>;
  SlashArgs(input: string): Promise<SlashArgsResult>;
  ListDir(rel: string): Promise<DirEntry[]>;
  // FileSearch 工作区文件名搜索（@ 引用增强：跨目录定位资料）。
  FileSearch(query: string, limit?: number): Promise<FileSearchHit[]>;
  // Materials 工作区资料概览：office/文本文件按修改时间倒序。
  Materials(limit?: number): Promise<FileSearchHit[]>;
  // WorkspaceSearch 工作区全文搜索（轻量 RAG）：正文关键词 + 命中片段。
  WorkspaceSearch(query: string, limit?: number): Promise<WorkspaceSearchHit[]>;
  // 常用资料固定（P1-②）：钉住的文件在新会话启动时自动带入上下文。
  PinnedMaterials(): Promise<FileSearchHit[]>;
  PinMaterial(rel: string): Promise<FileSearchHit[]>;
  UnpinMaterial(rel: string): Promise<FileSearchHit[]>;
  // SummarizeFile 工作区资料分块摘要（map-reduce），供资料面板「摘要后引用」。
  SummarizeFile(rel: string, focus?: string): Promise<GaeaSummaryResult>;
  // TaskTemplates 返回预置办公任务模板库（欢迎页 + slash 共用）。
  TaskTemplates(): Promise<TaskTemplate[]>;
  ReadFile(rel: string): Promise<FilePreview>;
  Preview(rel: string): Promise<PreviewResult>;
  // GaeaBrowserObserve 受控浏览器观察帧（v4.28 A2）：当前页 jpeg 截图+URL/
  // 标题；浏览器未运行 Available=false（被动动作，绝不拉起）。同名直调。
  GaeaBrowserObserve(): Promise<BrowserObserveView>;
  // SubagentRuns 读取当前会话派发的全部子代理分工（状态/任务摘要/回答/工具数）。
  SubagentRuns(sessionPath: string): Promise<SubagentRunsView>;
  // Side Chat 式追问（v4.64）：对已完结的 sa_ 运行追加用户提问，后台运行
  // 即刻返回；文本增量走 gaea-subagent-text 专用通道，完成态经轮询自校正。
  SubagentFollowUp(sessionPath: string, ref: string, prompt: string): Promise<string>;
  // PromoteSubagent 把子代理会话提升为独立顶层会话（dsh Side Chat promote 语义，
  // v4.66.0）：忠实投影 transcript 为新会话日志，返回新 sessionPath；不动原运行。
  PromoteSubagent(sessionPath: string, ref: string): Promise<string>;
  // SubagentTranscript 读取某个子代理的完整 transcript（Agent 网络节点查看用）。
  SubagentTranscript(sessionPath: string, ref: string): Promise<SubagentTranscriptView>;
  // DeliverableRegistry 读取会话的权威产物登记表（v4.24 C1）：后端从事件日志
  // 折叠出写类工具落盘登记（路径/工具/轮次/时间/次数），替代正文扩展名白名单。
  DeliverableRegistry(sessionPath: string): Promise<DeliverableRegistryView>;
  // ResyncEvents 事件序号防线补拉（v4.26 对话流式重造）：Wails 事件流吞件时
  // 前端按 seq 跳号调用，后端从当前会话磁盘日志折叠出对话项全量快照整体替换。
  ResyncEvents(afterSeq: number): Promise<GaeaResyncResult>;
  OpenWorkspacePath(rel: string): Promise<void>;
  RevealWorkspacePath(rel: string): Promise<void>;
  SavePastedImage(dataUrl: string): Promise<string>;
  SaveAttachmentFile(fileName: string, base64Data: string): Promise<string>;
  AttachmentDataURL(path: string): Promise<string>;
  // Auto-updater (desktop/updater_app.go): the injected build version, a manifest
  // check, applying an update (win/linux self-update; macOS opens the download
  // page), and opening that page directly. Progress streams on "updater:progress".
  Version(): Promise<string>;
  CheckUpdate(): Promise<UpdateInfo | null>;
  ApplyUpdate(): Promise<void>;
  OpenDownloadPage(): Promise<void>;
  // GetAppInfo 应用信息（name/version/tagline/releases），设置中心「更新信息」
  // 展示；wailsjsCompat 直调转正（Go CoreB.GetAppInfo，同名前缀无需映射）。
  GetAppInfo(): Promise<Record<string, unknown>>;
  // GetBoardManifests 板块清单（壳层首页 launcher manifest 驱动）；空/失败由
  // 调用方 loadBoardManifests fail-closed 回退静态 canonicalBoards。同批转正。
  GetBoardManifests(): Promise<Array<Record<string, unknown>>>;
  // Feature Model 功能级模型绑定（读侧，Go CoreB.GetFeatureModel/GetFeatureModelEnabled，
  // wailsjsCompat 直调转正；useFeatureModel / useBindState 共用；feature 取值
  // chat/whisper/novel/office/gaea/characterlib）。
  GetFeatureModel(feature: string): Promise<Record<string, string>>;
  GetFeatureModelEnabled(feature: string): Promise<boolean>;
  // Window state persistence.
  SaveWindowState(state: {width:number;height:number;x:number;y:number;maximised:boolean}): Promise<void>;
  // UnifiedSearch 跨库统一检索一次调用：工作区关键词命中（topN 条）+ 语义跨库命中。
  // S1.2-B/C（docs/gaea-memory-isolation-design.md）：签名加 scope 参数——
  // ""=全部（旧行为，仅显式选择时使用），"work"/"play"=只搜对应空间；前端默认
  // 传当前生效空间（GaeaSpaceActive，双空间红线：默认不跨空间混搜）。
  // 注意：后端 B 步合入前，旧绑定为 (query, topN)，此签名按约定先行对齐。
  UnifiedSearch(query: string, scope: SearchScope, topN?: number): Promise<UnifiedSearchView>;
  // RouteIntent 统一意图路由（v4.5 指令中枢）命令面板入口（S4.6）：
  // dryRun=true 只解析与校验、零副作用（返回「将发生什么」预览语 + action/target），
  // 面板据此渲染指令预览卡；用户显式确认后以 dryRun=false 真执行并返回回执。
  // 预览-确认制 = 「宁漏勿误」纪律在搜索框面的落地（搜索词不是整句指令入口）。
  RouteIntent(text: string, dryRun: boolean): Promise<IntentResultView>;
  // RetrievalEvalRun 运行检索质量测评：内置查询集统计平均 recall@10，
  // 达标门槛后端固定 0.8，返回指标 + 逐查询命中明细。
  RetrievalEvalRun(): Promise<RetrievalEvalReport>;
  // ── 工作区文件语义索引 ──
  // FileIndexRebuild 重建索引（异步任务，T5-1）：进度经 gaea-task 事件推送，
  // 结果（total/skipped）在任务 result 里。
  FileIndexRebuild(): Promise<TaskView>;
  FileSemanticSearch(query: string, topN: number): Promise<FileSemanticHit[]>;
  // Cost database panel.
  // PickFiles opens a native file dialog and imports selected files.
  // filters 可选（审计刀D）：逗号分隔小写扩展名（如 "png,jpg"），空串=不过滤；
  // Wails 系统对话框前置过滤，壳内仍以后置校验兜底。
  PickFiles(filters?: string): Promise<FilePickResult[]>;
  // ReadFileB64 读取本地文件为 base64（配合 PickFiles：进度计划导入链路，v4.162）。
  ReadFileB64(path: string): Promise<string>;
  // SaveFileAs 系统「另存为」对话框 + 写入 base64 内容（进度计划导出链路，v4.162）。
  // 返回实际写入路径；用户取消返回空串。
  SaveFileAs(defaultName: string, base64Data: string): Promise<string>;
  // OpenLogsDir 在文件管理器中打开长期日志目录（<DataRoot>/logs/，v4.163）。
  OpenLogsDir(): Promise<void>;
  // ── 阶段 5 T5-1 任务中心 ──
  // TaskList 返回最近任务（新→旧）；TaskCancel 取消（running 中断/queued 取消）；
  // TaskKill 强制终止（v4.78：协作取消 + 击杀任务自有 OS 进程树；纯函数任务
  // 等价 TaskCancel）；TaskRetry 重试失败/已取消的任务；TaskOutput 读取任务
  // 实时输出尾部（C1）。任务实时进度经 onTaskEvent 推送。
  TaskList(space: string): Promise<TaskView[]>;
  TaskCancel(id: string): Promise<void>;
  TaskKill(id: string): Promise<void>;
  TaskRetry(id: string): Promise<void>;
  TaskOutput(id: string): Promise<TaskOutputView>;
  // ── 阶段七 7.3-1 任务收件箱（多入口统一；Go OfficeB.GaeaTaskInbox*，同名
  // 前缀无需映射——GetProgrammingWebStatus 先例）──
  // GaeaTaskInboxList 按空间过滤收件箱（''=全部；状态组序+最近变化优先）；
  // Save 新建（id 空，pending）或只改 title/note（来源字段不可变——审计链）；
  // SetStatus 走状态机（pending→doing→done；pending|doing→abandoned）；
  // Delete 删除。纯同步按需：无轮询无事件，前端在每次变更后重拉。
  GaeaTaskInboxList(space: string): Promise<TaskInboxView[]>;
  GaeaTaskInboxSave(reqJSON: string): Promise<TaskInboxView>;
  GaeaTaskInboxSetStatus(id: string, status: string): Promise<TaskInboxView>;
  GaeaTaskInboxDelete(id: string): Promise<void>;
  // v4.80 上下文浏览器：懒加载节点「完整调用」详情（按 seq 回读会话日志；
  // tool_result 配对 dispatch 取参数，user/assistant 取全文）。sessionPath
  // 语义同 ContextView。
  ContextNodeDetail(seq: number, sessionPath: string): Promise<ContextNodeDetailView>;
  // v4.1 证据链：Journal 最近证据卡（跨会话聚合，时间倒序；前端「证据」入口）。
  GaeaJournalList(limit: number): Promise<JournalChangeRecord[]>;
  // 2b Git 面板最小集（D3：单仓库 status/diff/stage/unstage/discard/commit/log，无 push）。
  GaeaGitStatus(): Promise<GitStatusView>;
  GaeaGitDiff(path: string, staged: boolean): Promise<string>;
  GaeaGitStage(paths: string[]): Promise<void>;
  GaeaGitUnstage(paths: string[]): Promise<void>;
  GaeaGitDiscard(path: string): Promise<void>;
  GaeaGitCommit(message: string): Promise<string>;
  GaeaGitLog(limit: number): Promise<GitCommitInfoView[]>;
  // 2.5e 后半：指定子代理会话的上下文构成快照（Agent 网络节点跳转）。
  GaeaSubagentContextView(sessionPath: string, ref: string): Promise<ContextTimeline>;
  // v4.1b：双通道复核一张证据卡（A 结构/引用完整性 + B 视觉健全性）。
  VerifyRecord(id: string): Promise<VerdictView>;
  // v4.1b：基线快照回滚（目标被手工修改时拒绝，零覆盖）。
  RollbackRecord(id: string): Promise<void>;
  // v4.1c：中文规范体检（GB/T 9704 红头要素 lint，md/txt/docx）。
  DocumentLint(rel: string): Promise<LintReportView>;
  // ── 编程板块：DeepSeek Harness Web 进程管理 ──────────────
  // GetProgrammingWebStatus 返回运行状态（running/owned/pid/url/root/log/uptime_s）；
  // StartProgrammingWeb 启动 dsh web（已运行幂等返回）；StopProgrammingWeb
  // 仅停止 gaea 自启实例（外部实例返回提示，不误杀）。
  // GetProgrammingWebPreflight 返回启动前置条件逐项检查（harness 有效 / pnpm 可用 /
  // 依赖已装 / 构建就绪 / 端口空闲 + all_ready）；ProgrammingWebLogTail(n) 读自启
  // 日志尾部（n 钳制 [1,200]，默认 50），启动引导视图渲染真实清单与日志。
  GetProgrammingWebStatus(): Promise<ProgrammingWebStatus>;
  StartProgrammingWeb(): Promise<void>;
  StopProgrammingWeb(): Promise<void>;
  GetProgrammingWebPreflight(): Promise<ProgrammingWebPreflight>;
  ProgrammingWebLogTail(n?: number): Promise<ProgrammingWebLogTail>;
  // LogFrontendError 记录前端错误/主线程卡死诊断到 gaea.log。
  LogFrontendError(message: string): Promise<void>;
  // ── 批次二 wailsjsCompat 双轨退役转正（Go CoreB 门面，同名前缀无需映射）──
  // GetConfig/SaveConfig 会话/工作区全局配置读写（Record<string,string>）；
  // GetStats 会话运行统计（NovelPage 消费）；ListSkills 技能清单
  // （CreateInspector 消费）；ExportAll 全量导出（ExportPanel 消费，
  // 只导出主线时传 onlyMainline=true，返回文件名 → 内容映射）。
  GetConfig(): Promise<Record<string, string>>;
  SaveConfig(key: string, value: string): Promise<void>;
  GetStats(): Promise<Record<string, unknown>>;
  ListSkills(): Promise<Array<Record<string, unknown>>>;
  ExportAll(onlyMainline: boolean): Promise<Record<string, string>>;
  // ── 批次三a legacy 直调转正（Go CoreB.ListProjects，同名前缀无需映射）──
  // ListProjects 书架工程卡片列表（api/characterlib.listShelfProjects 消费；
  // Go 返回 []ProjectCard，前端投影为 Record）。
  ListProjects(): Promise<Array<Record<string, unknown>>>;
}
