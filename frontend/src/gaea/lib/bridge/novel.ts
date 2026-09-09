// novel.ts — NovelBindings（AppBindings 分域接口之一，Go NovelB 门面）：
// 小说域经 AppBindings 的方法（legacy 场景生成族走 wailsjsCompat 直调）。

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
  SaveCharactersBatch(namesJSON: string): Promise<Record<string, unknown>>;
  NovelReadingAsk(kind: string, title: string, chapterText: string, selection: string, question: string, historyJSON: string): Promise<string>;
  GenerateSceneIllustration(chapterNum: number): Promise<Record<string, unknown>>;
}
