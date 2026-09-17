// novel.ts — NovelBindings（AppBindings 分域接口之一，Go NovelB 门面）：
// 小说域经 AppBindings 的方法（批次三b：章节族/叙事状态族/场景族/项目角色族
// 从 wailsjsCompat 直调转正；Go 侧 CancelCreateChapter 返回 bool、MergeCharacters
// 返回 map，签名以 bindings_novel.go 实测为准）。
// 文风指纹族（NovelFingerprintStatus/Build/Score）：CreatePage「文风指纹」面板
// 消费——参考档状态/构建 + 章节 AI 味体检（主线并行开发，wailsjs 再生前由
// CreatePage 直接 import 三个绑定，再生后 tsc 自然转绿）。

// ── 文风指纹载荷（Go 侧 omitempty 字段可能缺省，消费方需 ?. 与 ?? 防御）──
/** 风格摘要：句长/段长统计 + 词汇密度 + 口头禅（bigram/trigram/签名词）。 */
export interface FingerprintSummary {
  sentenceMean: number; sentenceSd: number; paraMean: number; ttr1000: number;
  dialogRatio: number; fourCharRatio: number; connectiveDensity: number; adjAdvDensity: number;
  topBigrams: string[]; topTrigrams: string[]; authorSignWords: string[];
}
/** 参考档状态：exists=false 表示尚未构建（summary 等字段缺省）。 */
export interface FingerprintStatusPayload {
  exists: boolean; builtAt?: string; chapters?: number; chars?: number; summary?: FingerprintSummary;
}
/** 单条体检命中：severity 分档 low/medium/high/blocker（未知档允许透传 string）。 */
export interface FingerprintIssue {
  start: number; end: number; reason: string;
  severity: 'low' | 'medium' | 'high' | 'blocker' | string;
  suggestion: string; excerpt: string;
}
/** 章节体检结果：score 0-100 越高越像 AI；delta=与参考档的函数词距离（越小越像
 *  作者，无参考档时缺省 → 消费方显示「按通用阈值打分」口径）。 */
export interface FingerprintScorePayload {
  chapterNum: number; refExists: boolean; score: number;
  delta?: number;
  issues: FingerprintIssue[];
}

// ── 伏笔一致性体检载荷（LintForeshadows；Go 侧字段可能 omitempty，消费方 ?. 与 ?? 防御）──
/** 单条体检发现：code 为发现类别（ordering/status-mismatch/dangling/stale/duplicate，
 * 未知类别允许透传 string）；chapter 为相关章节文件名（如 001.md），缺省无。 */
export interface ForeshadowLintFinding {
  code: 'ordering' | 'status-mismatch' | 'dangling' | 'stale' | 'duplicate' | 'overdue' | 'unplanned' | string;
  severity: 'high' | 'medium' | 'low' | string;
  foreshadowId: string;
  itemDesc: string;
  message: string;
  chapter?: string;
}
/** 体检报告：全书概要统计 + findings（Go 侧可能给 null，消费方 ?? [] 防御）。 */
export interface ForeshadowLintReport {
  totalChapters: number;
  items: number;
  planted: number;
  hinted: number;
  revealed: number;
  longTerm: number;
  findings: ForeshadowLintFinding[];
}

/** 清理入口统一返回：deleted=删除条数；rolledBack=回退回收态条数（Clean 专属，
 * Go omitempty 缺省）；resetManual=重置为 pending 的手工条数（Reset 专属）。 */
export interface ForeshadowCleanupResult {
  deleted: number;
  rolledBack?: number;
  resetManual?: number;
}
/** 伏笔统计（spec §5.1）：resolved 为 gaea 写入口径 revealed（resolved 别名归一
 * 并入）；overdueCount=存活条目计划回收章<当前章（currentChapter<=0 时后端自动
 * 按已写章节数）；Total=分状态计数之和（未知状态不入桶）。 */
export interface ForeshadowStatsReport {
  total: number;
  pending: number;
  planted: number;
  hinted: number;
  resolved: number;
  partiallyResolved: number;
  abandoned: number;
  longTermCount: number;
  overdueCount: number;
  currentChapter?: number;
}

/** 运行时紧急度投影（后端按计划回收章实时算，不落库；前端不自算阈值 D7）。
 *  level: 0=不紧急 1=需关注 2=急需回收 3=已超期；mustResolve=进入「本章必须回收」集合。 */
export interface ForeshadowUrgencyPayload {
  level: number;
  remainingChapters: number;
  overdueChapters?: number;
  resolveStatus: 'must_resolve_now' | 'overdue' | 'not_yet' | 'no_plan' | string;
  mustResolve: boolean;
}
/** 最近一轮章节分析的伏笔同步结果（GetLastForeshadowSync；D3 跳过原因可见不静默）。 */
export interface ForeshadowSyncSkipReason {
  kind: 'invalid_reference' | 'already_resolved' | 'not_planted' | 'no_match' | 'limit_reached' | 'empty_content' | string;
  refId?: string;
  title?: string;
  message: string;
}
export interface ForeshadowSyncResult {
  plantedCount: number;
  resolvedCount: number;
  createdCount: number;
  updatedIds: string[];
  createdIds: string[];
  matchedByContent: number;
  skippedResolveCount: number;
  skippedReasons: ForeshadowSyncSkipReason[];
  errors: string[];
}

// ── 驱动式整章重写（t4-C3 首刀：whole 模式+版本库）──────────────
/** 重写版本索引条目（列表接口不下发全文）。 */
export interface RewriteVersionIndex {
  id: string;
  chapterNum: number;
  mode: 'whole' | 'partial' | 'deslop' | string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'applied' | 'discarded' | string;
  similarity?: number;
  createdAt: string;
}
/** 重写结果（生成后不自动落章；前端对比确认后调 NovelApplyRewriteVersion）。 */
export interface NovelRewriteResult {
  versionId: string;
  status: string;
  similarity: number;
  change: number;
  changePercent: number;
  originalWordCount: number;
  newWordCount: number;
  newContent: string;
}
/** 重写请求载荷（NovelChapterRewrite 的 reqJSON）。 */
/** 单条分析标注：pos/length 为正文 rune 偏移与长度（-1=未命中不高亮）。 */
export interface ChapterAnnotation {
  type: 'hook' | 'foreshadow' | 'plot_point' | 'conflict' | 'character' | 'suggestion' | string;
  title?: string;
  content: string;
  importance?: number;
  pos: number;
  length?: number;
  tags?: string[];
}
// ── t7 分析 V2 视图（Go types.ChapterAnalysisResult 直连；顶层 snake_case
// 如实透传〔Q1 裁决：不为只读面板复制九维子类型树〕；全字段可缺省防御）──
export interface ChapterAnalysisV2View {
  chapter_num?: number;
  chapter_file?: string;
  analyzed_at?: string;
  engine?: string;
  model?: string;
  analyzer_source?: string;
  result?: AnalysisV2ResultView;
}
/** 九维分析载荷（Go AnalysisResultV2 镜像；嵌套 snake_case 同落盘契约）。 */
export interface AnalysisV2ResultView {
  hooks?: Array<{ type?: string; content?: string; strength?: number; position?: string }>;
  foreshadows?: Array<{
    title?: string; content?: string; type?: string; strength?: number; subtlety?: number;
    category?: string; is_long_term?: boolean; related_characters?: string[]; estimated_resolve_chapter?: number;
  }>;
  conflict?: {
    types?: string[]; parties?: string[]; level?: number;
    description?: string; resolution_progress?: number;
  };
  emotional_arc?: {
    primary_emotion?: string; intensity?: number; curve?: string; secondary_emotions?: string[];
  };
  character_states?: Array<{
    name?: string; old_state?: string; new_state?: string; psychological_change?: string;
    key_event?: string; survival_status?: string;
  }>;
  organization_states?: Array<{
    org_name?: string; power_value?: number; destroyed?: boolean;
    member_changes?: Array<{ character_name?: string; change_type?: string; position?: string; reason?: string }>;
  }>;
  plot_points?: Array<{ content?: string; type?: string; importance?: number; impact?: string }>;
  scenes?: Array<{ location?: string; atmosphere?: string; duration?: string }>;
  pacing?: string;
  dialogue_ratio?: number;
  description_ratio?: number;
  scores?: {
    pacing?: number; engagement?: number; coherence?: number; overall?: number; score_justification?: string;
  };
  plot_stage?: string;
  suggestions?: string[];
  summary?: string;
}
// ── 全书体检视图（GenerationGate 闭环收口；Go BookHealthReport 直连）──
/** 逐章体检行（全确定性）。aiTasteScore=-1=打分失败。 */
export interface BookHealthChapterView {
  chapterNum: number;
  words: number;
  outlineIssues: number;
  outlineErrors: number;
  qualityIssues: number;
  qualityErrors: number;
  aiTasteScore: number;
}
/** 全书体检报告：聚合 + 逐章行 + 伏笔 Lint（复用 ForeshadowLintReport 形状）。 */
export interface BookHealthReportView {
  totalChapters: number;
  chapters: BookHealthChapterView[];
  contractIssueChapters: number;
  qualityIssueChapters: number;
  worstAiTaste?: BookHealthChapterView;
  analyzedChapters: number;
  foreshadow?: {
    totalChapters: number;
    items: number;
    planted: number;
    hinted: number;
    revealed: number;
    longTerm: number;
    findings: Array<{ code?: string; message?: string; [k: string]: unknown }>;
  };
}
export interface NovelRewriteRequest {
  source?: 'custom' | 'analysis_suggestions' | 'mixed';
  suggestion_indices?: number[];
  custom_instructions?: string;
  focus_areas?: string[];
  preserve_elements?: {
    preserve_structure?: boolean;
    preserve_dialogues?: string[];
    preserve_plot_points?: string[];
    preserve_character_traits?: boolean;
  };
  target_word_count?: number;
}

export interface NovelBindings {
  // GenerateBookCover 生成项目书封（3:4，play exports），返回封面路径。
  GenerateBookCover(projectId: string, promptHint: string): Promise<string>;
  // NovelSearch 章节全文检索（Go NovelB.NovelSearch(query) 单参，返回扁平命中
  // 切片、每行冗余携带汇总；wailsjsCompat 直调转正，ChapterPage 搜索消费）。
  NovelSearch(query: string): Promise<Array<Record<string, unknown>>>;
  // ── 批次二 wailsjsCompat 双轨退役转正（Go NovelB 门面，同名前缀）──
  // 世界观读写（GetWorldview/SaveWorldview/SaveAllWorldviewSections）、
  // 世界观对话（ChatWorldview：Go 实为 ChatB.ChatWorldview(userMsg,
  // currentContent) (map[string]interface{}, error)——返回结构化世界观，
  // 第二参名 currentContent）、一致性检查（CheckConsistency*）、伏笔读写
  // （GetForeshadows/SaveForeshadows）、批量角色生成（SaveCharactersBatch）、
  // 阅读助手（NovelReadingAsk 六参）、场景插图（GenerateSceneIllustration）。
  GetWorldview(): Promise<string>;
  SaveWorldview(content: string): Promise<void>;
  ChatWorldview(userMsg: string, content: string): Promise<Record<string, unknown>>;
  GetWorldviewSections(): Promise<Record<string, unknown>>;
  SaveAllWorldviewSections(sectionsJSON: string): Promise<void>;
  CheckConsistency(): Promise<Record<string, unknown>>;
  CheckConsistencyDeep(maxChapters: number): Promise<Record<string, unknown>>;
  GetForeshadows(): Promise<Record<string, unknown>>;
  SaveForeshadows(itemsJSON: string): Promise<void>;
  // LintForeshadows 伏笔一致性体检（无参，主线并行开发中：wailsjs 再生前
  // wailsjs 侧缺该签名属预期，再生后 tsc 转绿；ForeshadowPanel「一致性体检」消费）。
  LintForeshadows(): Promise<ForeshadowLintReport>;
  // ── 伏笔生命周期清理与统计（t1-P3，spec §8.1/§5.1）：三个清理入口语义严格
  // 区分（章节删除/重分析前清理/项目重置），source_type 是批量清理唯一判据、
  // 手动条目（source_type 非 analysis）永不批量删除只重置。
  DeleteChapterForeshadows(chapterFile: string, onlyAnalysisSource: boolean): Promise<ForeshadowCleanupResult>;
  CleanChapterAnalysisForeshadows(chapterFile: string): Promise<ForeshadowCleanupResult>;
  ClearProjectForeshadowsForReset(): Promise<ForeshadowCleanupResult>;
  // GetForeshadowStats 伏笔统计：分状态计数 + 超期数；currentChapter<=0 时
  // 后端自动按已写章节数计算（returned currentChapter 带回实际口径）。
  GetForeshadowStats(currentChapter: number): Promise<ForeshadowStatsReport>;
  // GetLastForeshadowSync 最近一轮章节分析的伏笔同步结果（尚未分析时 reject，
  // Error.message 为中文提示；消费方 catch 后显示「尚未执行分析」）。
  GetLastForeshadowSync(): Promise<ForeshadowSyncResult>;
  // ── 驱动式整章重写（t4-C3 首刀：whole 模式）──────────────────
  // NovelChapterRewrite 按请求重写整章（温度 0.7），生成后不自动落章；
  // 应用/丢弃/恢复走版本库三绑定（恢复=写回原文快照，gaea 强制增量）。
  NovelChapterRewrite(chapterNum: number, reqJSON: string): Promise<NovelRewriteResult>;
  NovelListRewriteVersions(chapterNum: number): Promise<RewriteVersionIndex[]>;
  NovelGetRewriteVersion(chapterNum: number, versionID: string): Promise<Record<string, unknown>>;
  NovelApplyRewriteVersion(chapterNum: number, versionID: string): Promise<Record<string, unknown>>;
  NovelDiscardRewriteVersion(chapterNum: number, versionID: string): Promise<void>;
  NovelRestoreRewriteVersion(chapterNum: number, versionID: string): Promise<Record<string, unknown>>;
  // NovelChapterSuggestions 读取该章分析建议（重写建议驱动勾选数据源）；
  // 无分析结果返回空数组（正常态）。
  NovelChapterSuggestions(chapterNum: number): Promise<string[]>;
  // NovelChapterAnnotations 读取该章分析标注（缺档且有 V2 分析时后端按需
  // 重建；无分析返回空数组）。Pos=-1 表示未命中不高亮（如 suggestion 类）。
  NovelChapterAnnotations(chapterNum: number): Promise<ChapterAnnotation[]>;
  // ── t7 前端接线（规格 进度计划/gaea-analysis-v2-panel-t7-20260916.md）──
  // NovelChapterAnalysisV2 读取该章 V2 分析（analysis-v2.json 条目直连）；
  // 缺档 reject（message=「尚未分析」，面板空态引导先分析）。
  NovelChapterAnalysisV2(chapterNum: number): Promise<ChapterAnalysisV2View>;
  // AnalyzeChapter 触发该章 LLM 分析（原 Legacy 面绑定转正；V1 wire 返回
  // 面板忽略——真相源是 analysis-v2.json 落盘，完成后重拉 V2；顺带伏笔
  // 同步/记忆回填既有链路）。
  AnalyzeChapter(chapterNum: number): Promise<Record<string, unknown>>;
  // RunBookHealthCheck 全书体检（GenerationGate 闭环收口：触发式全量编译，
  // 纯确定性零 LLM；作者点按跑完出报告——无常驻无定时）。
  RunBookHealthCheck(): Promise<BookHealthReportView>;
  SaveCharactersBatch(namesJSON: string): Promise<Record<string, unknown>>;
  NovelReadingAsk(kind: string, title: string, chapterText: string, selection: string, question: string, historyJSON: string): Promise<string>;
  // 配图 v2（阶段一刀 D）：optsJSON={"characterIds":[],"style":""}（空串=旧行为）；
  // 返回增 refNote（参考使用情况：已附/降级/跳过原因）。
  GenerateSceneIllustration(chapterNum: number, optsJSON: string): Promise<Record<string, unknown>>;
  // ── 批次三b wailsjsCompat 双轨退役转正（Go NovelB 门面 bindings_novel.go，
  // 同名前缀；ChapterPage/CreatePage/api 小说角色族消费面=章节正文/分支/
  // 叙事状态/场景/项目角色——小说创作间数据面归 play）──
  // 章节族：正文读写（Go GetChapter/GetChapterBranch 返回 map 含 content/…，
  // Save* 仅 error）。
  GetChapter(num: number): Promise<Record<string, unknown>>;
  GetChapterBranch(num: number, branch: string): Promise<Record<string, unknown>>;
  SaveChapterContent(num: number, content: string): Promise<void>;
  SaveChapterBranchContent(num: number, branch: string, content: string): Promise<void>;
  // ── 批次三c wailsjsCompat 双轨退役转正（Go NovelB 门面 bindings_novel.go，
  // 同名前缀；CreatePage 创作间消费）──
  // 分支构思（Go QuickBrainstormBranches(setting, prevSummary) 返回 map 含
  // branches）、创建章节（Go CreateChapter 8 参，返回 map 含 nodeId/chapterNum/
  // branch——预创建节点信息，正文经流式事件推送）、大纲节点删除（Go
  // DeleteOutlineNode(nodeID) 仅 error）。
  QuickBrainstormBranches(setting: string, prevSummary: string): Promise<Record<string, unknown>>;
  CreateChapter(setting: string, prevSummary: string, plotReq: string, chapterNum: number, branchFromNodeID: string, skillName: string, minWords: number, temperature: number): Promise<Record<string, unknown>>;
  DeleteOutlineNode(nodeID: string): Promise<void>;
  // 叙事状态族：v4.7x 小说革命状态结算（BuildNovelStatePatch 构造 patch，
  // SettleNovelState 结算写回，均返回 map）。
  GetNovelState(): Promise<Record<string, unknown>>;
  BuildNovelStatePatch(chapterNum: number): Promise<Record<string, unknown>>;
  SettleNovelState(patchJSON: string, approved: boolean): Promise<Record<string, unknown>>;
  // AI 味净化/重写（v4.7x 反 AI 味，返回 {done,...}）。
  DeSlopChapterAiTaste(chapterNum: number): Promise<Record<string, unknown>>;
  RewriteChapterAiTaste(chapterNum: number): Promise<Record<string, unknown>>;
  // 文风指纹族（CreatePage「文风指纹」面板消费）：Status 读参考档状态；
  // Build 用全部已写章节构建/重建风格基线（样本不足时 reject，Error.message
  // 为中文提示）；Score 对指定章跑 AI 味体检（score 0-100，越高越像 AI）。
  NovelFingerprintStatus(): Promise<FingerprintStatusPayload>;
  NovelFingerprintBuild(): Promise<FingerprintStatusPayload>;
  NovelFingerprintScore(chapterNum: number): Promise<FingerprintScorePayload>;
  // 拆书反推族（v4.281）：Reconstruct 反推当前工程（立项 + 分批章节大纲），
  // 只返回预览载荷、零落库；Apply 把 items 按章号合并进大纲（幂等），返回命中节点数。
  NovelOutlineReconstruct(): Promise<OutlineReconstructPreview>;
  NovelOutlineReconstructApply(itemsJSON: string): Promise<number>;
  NovelOutlineReconstructStart(): Promise<NovelOutlineReconstructTaskState>;
  NovelOutlineReconstructTaskGet(): Promise<NovelOutlineReconstructTaskState>;
  // 平台质量评审族（v4.282，oh-story 蒸馏 T1）：Platforms 取可用档位清单（选择器数据源）；
  // ChapterReview 对指定章跑确定性评审（零 LLM：逐维 PASS/WARN/FAIL/SKIP + 原文证据 +
  // APPROVE/CONCERNS/REJECT 结论；平台档位空/未知回落 general）。
  NovelReviewPlatforms(): Promise<ReviewPlatform[]>;
  NovelChapterReview(chapterNum: number, platform: string): Promise<ChapterReviewPayload>;
  // 书源取书族（书源→拆书导入 t1，规格 docs/gaea-novel-booksource-import-2026-09.md）：
  // Search 聚合书源命中 + 泛搜索候选（HasRule=可导入；免规则候选正文不可解析）；
  // Toc 目录预览（范围选择依据）；Import 后台下载整本 → 落库书架（进度/终态走
  // novel-import-progress:<jobId> 事件）；Cancel 精确取消（未知 job 返回 false）。
  NovelBookSourceSearch(keyword: string): Promise<NovelBookSourceSearchResult>;
  NovelBookSourceToc(source: string, detailURL: string): Promise<NovelBookSourceTocPreview>;
  NovelBookSourceImport(source: string, detailURL: string, start: number, end: number, title: string, genre: string, style: string): Promise<NovelBookSourceImportStart>;
  NovelBookSourceImportCancel(jobId: string): Promise<boolean>;
  // 拆书导入出口（v4.287 tail 出口；v4.296 由 sin「送小说导入」消费转正进契约面）：
  // extractMode=full | tail:…；原 ImportNovelBook 同语义。
  ImportNovelBookEx(filePath: string, title: string, genre: string, style: string, extractMode: string, tailChapters: number): Promise<NovelImportResult>;
  NovelBookSourceImportChapters(source: string, projectPath: string, chaptersJSON: string): Promise<NovelBookSourceImportStart>;
  NovelBookSourceEnginesGet(): Promise<NovelBookSourceEnginesPayload>;
  NovelBookSourceEnginesSave(rulesJSON: string): Promise<number>;
  // 实体关系图谱（角色/组织/关系图数据）。
  GetEntityRelations(): Promise<Record<string, unknown>>;
  // 场景族：场景列表/生成/新建（Go 均返回 map；CancelCreateChapter 实测
  // (chapterNum int, branch string) bool——无 error 第二值）。
  GetChapterScenes(chapterNum: number): Promise<Array<Record<string, unknown>>>;
  GenerateScene(chapterNum: number, sceneID: string, plotReq: string, minWords: number): Promise<Record<string, unknown>>;
  CreateScene(chapterNum: number, slug: string, title: string): Promise<Record<string, unknown>>;
  // SaveSceneMeta 保存场景元数据（标题/概要/POV/地点/时间/情感/标签/状态；正文走 SaveScene）。
  SaveSceneMeta(chapterNum: number, sceneID: string, metaJSON: string): Promise<void>;
  CancelCreateChapter(chapterNum: number, branch: string): Promise<boolean>;
  // 项目角色族（novel/api/character.ts 消费；GetCharacters 同名已在 charlib.ts
  // 覆盖——同 Go NovelB.GetCharacters（map），无需重复）：AI 补全/剧照/合并/
  // 组织/关系。MergeCharacters 实测返回 map（非 void），Bridge 按 Record 暴露。
  GenerateProjectCharacterFill(chJSON: string): Promise<string>;
  GenerateCharacterPortrait(charID: string, model: string): Promise<string>;
  MergeCharacters(keepID: string, mergeID: string): Promise<Record<string, unknown>>;
  SaveOrganization(orgJSON: string): Promise<void>;
  DeleteOrganization(id: string): Promise<void>;
  ToggleOrgMember(charID: string, orgID: string): Promise<void>;
  SetCharacterCareer(charID: string, reqJSON: string): Promise<void>;
  RemoveCharacterCareer(charID: string, reqJSON: string): Promise<void>;
  SaveRelationship(relJSON: string): Promise<void>;
  DeleteRelationship(fromID: string, toID: string): Promise<void>;
  // ── t6 提示词工坊五绑定（PromptTemplate*；novel/api/prompt.ts 消费，
  // 规格 进度计划/gaea-prompt-workshop-t6-20260916.md §4.2）──
  PromptTemplateList(): Promise<PromptTemplateMetaView[]>;
  PromptTemplateGet(key: string): Promise<PromptTemplateDetailView>;
  PromptTemplateSave(key: string, reqJSON: string): Promise<PromptSaveResultView>;
  PromptTemplateReset(key: string): Promise<void>;
  PromptTemplatePreview(reqJSON: string, varsJSON: string): Promise<PromptPreviewResultView>;
  // NovelSceneBibleView 场景圣经视图（刀7续 POV 视图：sceneID 空=整章合成；
  // povView/hiddenFacts 对照为核心区）。
  NovelSceneBibleView(chapterNum: number, sceneID: string): Promise<Record<string, unknown>>;
  // ── t6-C2 模板包导入导出两绑定（PromptBundle*；规格
  // 进度计划/gaea-prompt-bundle-t6c2-20260916.md §4）──
  PromptBundleExport(): Promise<string>;
  PromptBundleImport(bundleJSON: string): Promise<PromptBundleImportResultView>;
}

// ── 提示词工坊载荷（t6 首刀；Go promptstore.Issue / PromptTemplateMeta 等，
// 结构化 struct 返回直连——novel/api/prompt.ts 的本地类型与本节结构一致）──
/** 保存/预览校验问题：error 阻断不落盘；warn 落盘带回提示。 */
export interface PromptIssueView {
  code: string;
  severity: 'error' | 'warn' | string;
  message: string;
}
/** 模板清单行（列表不回传正文——MuMu 1.74MB 全量下发教训）。 */
export interface PromptTemplateMetaView {
  key: string;
  category: string;
  description: string;
  /** override=覆盖激活生效；builtin=磁盘/embed 内置。 */
  source: 'override' | 'builtin' | string;
  hasOverride: boolean;
  overrideActive: boolean;
  version: number;
  updatedAt: number;
}
/** 模板正文（Go prompt.Template 宽松视图；全字段可缺省防御）。 */
export interface PromptTemplateBody {
  name?: string;
  system?: string;
  task?: string;
  output?: { format?: string; description?: string };
  constraints?: { must?: string[]; forbidden?: string[]; style?: string[] };
  input_sections?: Record<string, unknown>;
  version?: string;
  category?: string;
  description?: string;
  parameters?: string[];
}
/** 详情：生效模板 + 内置基线（对照/恢复参照）。 */
export interface PromptTemplateDetailView {
  meta: PromptTemplateMetaView;
  template: PromptTemplateBody;
  base: PromptTemplateBody;
}
/** 保存结果：error → saved=false；warn 落盘带回 issues；version=保存次数。 */
export interface PromptSaveResultView {
  saved: boolean;
  issues: PromptIssueView[];
  version: number;
}
/** 预览结果（V1 只渲染 system 段；warnings=未解析 {{name}} 变量名）。 */
export interface PromptPreviewResultView {
  systemPrompt: string;
  warnings: string[];
}
/** 模板包导入结果（t6-C2；Go promptstore.BundleImportResult 直连）。 */
export interface PromptBundleImportResultView {
  /** 至少一行变更（前端据此刷新列表）。 */
  applied: boolean;
  statistics: {
    total: number;
    keptSystemDefault: number;
    convertedToCustom: number;
    createdOrUpdate: number;
    skippedInvalid: number;
    skippedUnknown: number;
    skippedDuplicate: number;
  };
  /** 单行结果：action=kept_system_default/converted_to_custom/created_or_updated/
   *  skipped_invalid/skipped_unknown/skipped_duplicate；reason=跳过原因。 */
  outcomes: Array<{ key: string; action: string; reason?: string }>;
}
/** 反推任务状态（v4.291 任务化：轮询返回，succeeded 时携带预览）。 */
export interface NovelOutlineReconstructTaskState {
  taskId: string;
  status: string;
  message?: string;
  error?: string;
  preview?: OutlineReconstructPreview;
}

// 拆书反推载荷（对齐 internal/app NovelOutlineReconstructPreview，camelCase 形状）：
// aiUsed=false 表示模型不可用/解析失败，本次为规则兜底（warnings 说明原因）。
export interface OutlineReconstructItem {
  chapterNumber: number;
  title: string;
  summary: string;
  scenes?: string[];
  characters?: string[];
  keyPoints?: string[];
  emotion?: string;
  goal?: string;
}

export interface OutlineReconstructPreview {
  aiUsed: boolean;
  projectTitle: string;
  description?: string;
  theme?: string;
  genre?: string;
  narrativePerspective?: string;
  targetWords?: number;
  items: OutlineReconstructItem[];
  warnings?: string[];
}

// ── 平台质量评审载荷（NovelChapterReview / NovelReviewPlatforms；v4.282，oh-story 蒸馏 T1）──
/** 档位清单条目（引擎数据资产 rubric.json 的四档：通用/番茄/起点/知乎盐言）。 */
export interface ReviewPlatform {
  id: string;
  label: string;
  /** chapter=按章评审；story=整篇口径（盐言故事）。 */
  form: string;
}

/** 单条原文证据：段落号（1-based，0=未定位）+ Go 侧已截断摘录（免前端按 rune 切字）。 */
export interface ChapterReviewEvidence {
  paragraph: number;
  excerpt: string;
}

/** 单维度评审结果：verdict 四态；severity 仅 warn/fail 有（S1 打回 / S2 顾虑 / S3 局部 / S4 建议主）。 */
export interface ChapterReviewDimension {
  id: string;
  label: string;
  verdict: 'pass' | 'warn' | 'fail' | 'skip' | string;
  severity?: 'S1' | 'S2' | 'S3' | 'S4' | string;
  detail: string;
  advice?: string;
  evidence?: ChapterReviewEvidence[];
}

/** 章节评审报告：verdict 为发布门槛结论；counts 为 S1~S4 命中计数。 */
export interface ChapterReviewPayload {
  chapterNum: number;
  platform: string;
  platformLabel: string;
  words: number;
  verdict: 'APPROVE' | 'CONCERNS' | 'REJECT' | string;
  counts: Record<string, number>;
  dimensions: ChapterReviewDimension[];
  advisories?: string[];
}

// ── 书源取书载荷（NovelBookSource*；书源→拆书导入 t1）──
/** 搜书候选：kind=rule 书源规则命中（可导入）；kind=web 泛搜索参考候选（HasRule 才可导入）。 */
export interface NovelBookSourceCandidate {
  kind: 'rule' | 'web' | string;
  source: string;
  title: string;
  author?: string;
  url: string;
  host?: string;
  hasRule: boolean;
  latestChapter?: string;
}

export interface NovelBookSourceSearchResult {
  candidates: NovelBookSourceCandidate[];
  warnings?: string[];
}

/** 目录预览：total 全量章数；sample 首 8 + 末 4（truncated=true 时），供范围选择。 */
export interface NovelBookSourceTocPreview {
  total: number;
  sample: Array<{ title: string; url: string; order: number }>;
  truncated: boolean;
}

/** 在线导入起跑回执：进度/终态订阅 novel-import-progress:<jobId>（progress/done/error）。 */
export interface NovelBookSourceImportStart {
  jobId: string;
}

/** 下载失败章（引擎重试穷尽后如实上报，不占位）。 */
export interface NovelBookSourceFailedChapter {
  title: string;
  url: string;
  error: string;
}

/** 引擎规则条目（对齐 booksource.SearchEngineRule；字段可 omitempty，整体往返保存）。 */
export interface NovelBookSourceEngineRule {
  name: string;
  url: string;
  queryFormat?: string;
  result: string;
  title: string;
  link?: string;
  disabled?: boolean;
  comment?: string;
  linkParam?: string;
  redirectHosts?: string[];
  crawl?: Record<string, unknown>;
}

/** 引擎规则清单 + 落盘路径（编辑器数据面）。 */
export interface NovelBookSourceEnginesPayload {
  path: string;
  rules: NovelBookSourceEngineRule[];
}

/** 失败章补下结果（t3 重试；追加语义词——appended/addedWords 为本次增量）。 */
export interface NovelBookSourceAppendResult {
  path: string;
  title: string;
  appended: number;
  totalChapters: number;
  addedWords: number;
  failed?: NovelBookSourceFailedChapter[];
}

/** 文件导入结果（对齐 internal/app NovelImportResult；书源成书「送小说导入」同款返回）。 */
export interface NovelImportResult {
  path: string;
  title: string;
  chapter_count: number;
  total_words: number;
  encoding?: string;
  /** strong | weak | window | single | epub | booksource */
  split_strategy?: string;
  warnings?: Array<{ code: string; message: string; level: string }>;
}
