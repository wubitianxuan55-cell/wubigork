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
  code: 'ordering' | 'status-mismatch' | 'dangling' | 'stale' | 'duplicate' | string;
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
  SaveCharactersBatch(namesJSON: string): Promise<Record<string, unknown>>;
  NovelReadingAsk(kind: string, title: string, chapterText: string, selection: string, question: string, historyJSON: string): Promise<string>;
  GenerateSceneIllustration(chapterNum: number): Promise<Record<string, unknown>>;
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
  SaveRelationship(relJSON: string): Promise<void>;
  DeleteRelationship(fromID: string, toID: string): Promise<void>;
}