// novel.ts — NovelBindings（AppBindings 分域接口之一，Go NovelB 门面）：
// 小说域经 AppBindings 的方法（legacy 场景生成族走 wailsjsCompat 直调）。

export interface NovelBindings {
  // GenerateBookCover 生成项目书封（3:4，play exports），返回封面路径。
  GenerateBookCover(projectId: string, promptHint: string): Promise<string>;
}
