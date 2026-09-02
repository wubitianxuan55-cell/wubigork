/**
 * Wails 运行时类型声明
 * 为 window.go.app.App.* 与 window.runtime.* 提供类型安全，
 * 消除全站的 @ts-ignore 和 any 类型（T6-10.2「any 清零」）。
 *
 * 说明：Go 侧大量绑定返回 map[string]interface{}，无法穷举字段；
 * 本文件对前端实际消费的载荷给出精确结构类型（复用 src/types），
 * 其余动态载荷一律以 unknown 表达，由调用方用类型守卫/最小接口收窄。
 */
import type {
  OutlineNode, CharacterData, OrganizationData, RelationshipData,
  WorldviewSectionData, ConsistencyReportData, TTSConfig, TTSStatus, BrainstormIdea,
} from './index'

/**
 * window.runtime 事件 API（Wails 原生与 HTTP polyfill 共有形态）。
 * 使用方法签名（方法参数双变检查）：调用方可传 (data: unknown) => void，
 * 也可传具体载荷类型（如 TTSStreamEvent），由调用方自行收窄。
 * 泛型 T（默认 unknown）让具体载荷回调在 strict 下直接通过推导，
 * 而非依赖 unknown → T 的反向赋值（strictFunctionTypes 会拒绝）。
 */
export interface RuntimeAPI {
  EventsOn<T = unknown>(event: string, handler: (data: T) => void): void
  EventsOff?<T = unknown>(event: string, callback?: (data: T) => void): void
  EventsOnce?<T = unknown>(event: string, handler: (data: T) => void): void
  EventsOnMultiple?<T = unknown>(event: string, handler: (data: T) => void, maxCallbacks: number): (() => void) | void
  EventsEmit?(event: string, ...data: unknown[]): void
  BrowserOpenURL?(url: string): void
}

declare global {
  interface Window {
    go?: {
      app?: {
        App: AppAPI
      }
    }
    runtime?: RuntimeAPI
  }
}

/** 书架项目卡片（对齐 internal/app/shelf.go ProjectCard） */
interface ProjectCard {
  title: string
  genre: string
  style: string
  path: string // 项目完整路径
  word_count: number
  chapter_count: number
  created_at: string // ISO8601
  last_opened_at: string // ISO8601
}

/** 原生文件选择结果（对齐 internal/app FilePickResult） */
interface FilePickResult {
  path: string
  name: string
  size: number
}

/** 导入成品小说结果（对齐 internal/app NovelImportResult） */
interface NovelImportResult {
  path: string
  title: string
  chapter_count: number
  total_words: number
}

/** 写作统计摘要（对齐 internal/app/stats_handler.go GetStats） */
interface StatsData {
  totalWords: number
  chapterCount: number
  avgWordsPerChapter: number
  characterCount: number
  charAlive: number
  foreshadowTotal: number
  foreshadowRevealed: number
  foreshadowRate: number
}

/** Skill 元信息（对齐 internal/app/stats_handler.go ListSkills） */
interface SkillData {
  name: string
  description: string
  appliesTo: string[]
  version: string
}

/** 全文搜索结果（对齐 internal/search.Result） */
interface SearchResultData {
  file: string
  context: string // 匹配行前后各 40 字符
}

/** 小说全文检索命中（对齐 internal/app/novel_search_handler.go NovelSearchHit；只增字段，旧字段语义不变） */
export interface NovelSearchHitData {
  node_id: string
  title: string
  chapter_num: number
  branch?: string
  snippet: string
  title_hit: boolean
  /** 本章内命中序次（1-based；标题命中为 1） */
  match_index: number
  /** 正文段落索引（0-based，按空行分段，对齐阅读渲染器 .novel-reading-p 序号；标题命中为 -1） */
  paragraph_index: number
  /** 段内命中起始 rune 偏移（标题命中为 -1） */
  char_offset: number
  /** 命中词 rune 长度 */
  match_len: number
  /** 全书总命中数（不受返回条数上限影响；绑定返回扁平切片，汇总冗余填充在每行） */
  total_hits: number
  /** 命中章节总数（每行冗余携带） */
  chapter_count: number
}

/** 剧情分支（对齐 internal/app/plot_branch_handler.go PlotBranch） */
interface PlotBranch {
  id: string
  title: string
  summary: string
  characters_involved: string[]
  core_conflict: string
  foreshadow_impact: string
  tone: string
}

/** Lorebook 词条 */
interface LorebookEntry {
  key: string
  name?: string
  content?: string
  enabled?: boolean
}

/** gaea 后端 App 接口（legacy 单一绑定面；S2-3 后由 gaea/lib/bridge.ts 兼容代理路由） */
/**
 * legacy App 动态方法门面：已知方法保留精确签名，未知方法按 Promise 收窄。
 * 供仍以 window.go.app.App.Xxx() 直调的旧代码使用（T6-10.2 替代 (window as any)）。
 */
export type AppFacade = AppAPI & Record<string, (...args: unknown[]) => Promise<unknown>>

export interface AppAPI {
  // ── 认证 ──
  Login(): Promise<string>
  GetLoginStatus(): Promise<boolean>
  Logout(): Promise<void>

  // ── 项目 ──
  CreateProject(dir: string, title: string, genre: string, style: string): Promise<unknown>
  BootstrapProject(dir: string, title: string, genre: string, style: string, reference: string): Promise<unknown>
  OpenProject(dir: string): Promise<unknown>
  CloseProject(): Promise<void>
  GetProjectInfo(): unknown
  GetNovelsDir(): string
  ListProjects(): Promise<ProjectCard[]>
  DeleteProject(dir: string): Promise<void>
  GaeaPickFiles(): Promise<FilePickResult[]>
  ImportNovelBook(filePath: string, title: string, genre: string, style: string): Promise<NovelImportResult>

  // ── 大纲 ──
  GetOutlines(): { nodes: OutlineNode[]; story_thread?: string }
  SaveOutlineNode(nodeJSON: string): Promise<void>
  AddOutlineNode(nodeJSON: string): Promise<void>
  DeleteOutlineNode(nodeID: string): Promise<void>
  GenerateOutlineNodeDetail(nodeID: string): Promise<unknown>
  ContinueOutline(count: number): Promise<unknown>
  ExpandOutlineNode(nodeID: string, subCount: number): Promise<unknown>
  SaveStoryThread(storyThread: string): Promise<void>
  GetStoryThread(): string
  GenerateStoryThread(userHint: string): Promise<unknown>
  ChatStoryThread(userMsg: string): Promise<unknown>
  ChatOutline(userMsg: string): Promise<unknown>
  ChatOutlineNode(nodeID: string, userMsg: string): Promise<unknown>
  ImportStoryThreadFile(): Promise<unknown>
  GenerateOutlineWithDialogue(storyPrompt: string, numChapters: number, maxTurns: number): Promise<unknown>

  // ── 章节 ──
  GenerateChapter(outlineNodeID: string, skillName: string, targetWords: number): Promise<unknown>
  GetChapter(num: number): Promise<{ content?: string; summary?: unknown }>
  GetChapterBranch(num: number, branch: string): Promise<{ content?: string; summary?: unknown }>
  SaveChapterContent(num: number, content: string): Promise<void>
  SaveChapterBranchContent(num: number, branch: string, content: string): Promise<void>
  ChatChapter(chapterNum: number, userMsg: string): Promise<unknown>
  ReviewChapter(chapterNum: number): Promise<unknown>
  GenerateSceneIllustration(chapterNum: number): Promise<unknown>
  GetChapterScenes(chapterNum: number): Promise<unknown[]>
  SaveScene(chapterNum: number, sceneID: string, content: string): Promise<void>
  ReorderScenes(chapterNum: number, sceneIDs: string[]): Promise<void>
  CreateSnapshot(sceneID: string, chapterNum: number, label: string): Promise<unknown>
  ListSnapshots(sceneID: string, chapterNum: number): Promise<unknown[]>
  RestoreSnapshot(snapshotID: string, sceneID: string, chapterNum: number): Promise<void>
  MigrateProjectToV4(): Promise<void>
  IsProjectV4(): boolean

  // ── 角色 ──
  GetCharacters(): { characters?: CharacterData[]; organizations?: OrganizationData[]; relationships?: RelationshipData[] }
  GenerateCharacters(count: number): Promise<unknown>
  SaveCharacter(chJSON: string): Promise<void>
  DeleteCharacter(id: string): Promise<void>
  GenerateSingleCharacter(chJSON: string): Promise<unknown>
  ChatCharacter(userMsg: string): Promise<unknown>
  ChatCharacterDetail(charID: string, userMsg: string): Promise<unknown>
  SaveOrganization(orgJSON: string): Promise<void>
  DeleteOrganization(id: string): Promise<void>
  ToggleOrgMember(charID: string, orgID: string): Promise<void>
  SaveRelationship(relJSON: string): Promise<void>
  DeleteRelationship(fromID: string, toID: string): Promise<void>
  SaveCharacters(cfJSON: string): Promise<void>
  GenerateCharacterPortrait(charID: string, model: string): Promise<string>
  GenerateProjectCharacterFill(chJSON: string): Promise<string>
  MergeCharacters(keepID: string, mergeID: string): Promise<void>
  SetCharacterPortrait(charID: string, imageData: string): Promise<void>

  // ── 世界观 ──
  GetWorldviewSections(): Promise<WorldviewSectionData[]>
  ChatWorldview(userMsg: string): Promise<unknown>
  ChatWorldviewSection(sectionID: string, userMsg: string): Promise<unknown>
  SaveWorldviewSection(sectionID: string, content: string): Promise<void>
  SaveAllWorldviewSections(sectionsJSON: string): Promise<void>
  CheckWorldviewConsistency(): Promise<ConsistencyReportData>
  GenerateWorldviewSections(): Promise<unknown>
  SaveWorldview(content: string): Promise<void>
  GetWorldview(): string

  // ── AI 协写 ──
  GhostComplete(currentText: string, styleProfile: string): Promise<unknown>
  CancelGhost(): void
  CmdKEdit(selectedText: string, instruction: string, styleProfile: string): Promise<{ edited?: string }>
  GenerateBeats(outlineNodeID: string): Promise<unknown[]>
  GenerateProseFromBeat(beatID: string, allBeatJSON: string, chapterNum: number): Promise<unknown>

  // ── 图片生成 ──
  GenerateFreeImage(prompt: string, negative: string, size: string, style: string, model: string, seed: number, n: number): Promise<unknown>
  CancelImageGeneration(): Promise<boolean>
  GetComfyUITaskProgress(): Promise<{ status?: string; elapsed?: number }>
  GetImageBackend(): string
  GetImageBackendInfo(): {
    backend?: string
    model?: string
    image_model?: string
    comfyui_url?: string
    image_save_dir?: string
    comfyui_path?: string
    comfyui_python_path?: string
  }
  SetImageBackend(backend: string, comfyUIURL: string, imageModel: string, imageSaveDir: string): Promise<void>

  // ── 分析/审稿 ──
  AnalyzeChapter(chapterNum: number): Promise<unknown>
  GetForeshadows(): unknown
  ReviewBook(): Promise<unknown>
  GetBookData(): unknown

  // ── 伏笔登记（闭环：登记→埋设→回收）──
  // 全量写回 foreshadows.json，手工登记/状态流转/描述编辑/删除的统一入口。
  // itemsJSON 为 ForeshadowItemData[] 的 JSON 字符串（对齐 Go types.Foreshadow）；
  // 后端校验 Status ∈ planted/hinted/revealed，空 ID 兜底 manual_ 前缀；
  // AI Analyze 侧按 ID 合并，manual_ 手工条目不会被冲掉。
  SaveForeshadows(itemsJSON: string): Promise<void>

  // ── 统计 ──
  GetStats(): StatsData
  GetConfig(): Record<string, string>
  SaveConfig(key: string, value: string): Promise<void>
  ListSkills(): SkillData[]
  GetDashboard(dailyGoal: number): Promise<unknown>

  // ── TTS ──
  StartTTSServer(modelPath: string, port: number, backend: string): Promise<void>
  StopTTSServer(): Promise<void>
  TTSSpeakStreaming(text: string): Promise<void>
  GetTTSStatus(): TTSStatus
  GetTTSConfig(): TTSConfig
  SaveTTSConfig(modelPath: string, serverPath: string, port: number, backend: string, speed: number): Promise<void>

  // ── 导出 ──
  // onlyMainline: 传 true 仅导出主线章节（跳过分支）；不传/false = 含分支（默认，历史行为）。
  ExportAll(...onlyMainline: boolean[]): Promise<Record<string, string>>

  // ── 知识图谱 ──
  BuildBacklinkIndex(): Promise<unknown>
  GetBacklinks(entityName: string): Promise<unknown[]>
  GetAllEntityNames(): Promise<unknown[]>
  BuildContextBudget(systemPrompt: string, currentScene: string, previousScene: string, characterInfo: string, memoryInfo: string, modelCapacity: number): Promise<unknown>
  CheckConsistency(): Promise<unknown>
  GetEntityRelations(): Promise<unknown>

  // ── 一致性深检 ──
  /** AI 逐章提取状态卡 + 本地跨章比对，合并规则层结果（载荷见 ConsistencyDeepResult） */
  CheckConsistencyDeep(maxChapters: number): Promise<unknown>

  // ── 可视化 ──
  ExtractTimeline(): Promise<unknown>
  ExtractEmotionCurve(): Promise<unknown[]>
  ExtractCharacterHeatmap(): Promise<unknown>
  GenerateDefaultCanvas(): Promise<unknown>

  // ── 搜索 ──
  Search(query: string): Promise<Record<string, SearchResultData[]>>

  // ── 小说全文搜索（对齐 NovelB.NovelSearch）──
  NovelSearch(query: string): Promise<NovelSearchHitData[]>

  // ── AI 伴读 ──
  // historyJSON：问书历史 [{"q":"...","a":"..."}] 数组 JSON（元素见 NovelReadingAskTurn）；
  // 空串 = 单轮（兼容旧签名），解析失败后端忽略历史；summary 忽略历史。
  NovelReadingAsk(kind: string, title: string, chapterText: string, selection: string, question: string, historyJSON: string): Promise<string>

  // ── 脑暴 ──
  BrainstormIdeas(genre: string): Promise<BrainstormIdea[]>
  BrainstormBranches(nodeID: string): Promise<{ branches?: PlotBranch[] }>
  ApplyBranch(nodeID: string, branchIndex: number, userInput: string): Promise<void>

  // ── Lorebook ──
  GetLorebookEntries(): LorebookEntry[]
  SaveLorebookEntry(entryJSON: string): Promise<void>

  // ── Herdsman 安全检测（S2-1）──
  // 解析 herdsman config.yaml 的 api 段，返回 LAN 暴露检测结果（H0-4）。
  HerdsmanSecurityCheck(): Promise<LanExposure>

  // ── 模型（legacy 兼容路由，S2-3 后归属 ModelB 门面）──
  GetActiveModel(): Promise<string>
}

/** AI 伴读问书历史一轮（historyJSON 数组元素，对齐 internal/app readingTurn） */
export interface NovelReadingAskTurn {
  q: string // 用户问题
  a: string // 助手回答
}

/** Herdsman LAN 暴露检测结果（对齐 internal/herdsman.LanExposure） */
interface LanExposure {
  config_path: string
  exposed: boolean
  port: number
  config_missing: boolean
  parse_error?: string
  guidance?: string
}
