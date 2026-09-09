// novel.ts — NovelBindings（AppBindings 分域接口之一，Go NovelB 门面）：
// 小说域经 AppBindings 的方法（legacy 场景生成族走 wailsjsCompat 直调）。

export interface NovelBindings {
  // GenerateBookCover 生成项目书封（3:4，play exports），返回封面路径。
  GenerateBookCover(projectId: string, promptHint: string): Promise<string>;
  // NovelSearch 章节全文检索（Go NovelB.NovelSearch(query) 单参，返回扁平命中
  // 切片、每行冗余携带汇总；wailsjsCompat 直调转正，ChapterPage 搜索消费）。
  NovelSearch(query: string): Promise<Array<Record<string, unknown>>>;
}
