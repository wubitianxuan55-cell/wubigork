// mock/novel.ts — 小说域 dev mock（v4.171 批次一 NovelSearch legacy 直调转正）。
// Go 侧对应 NovelB 门面（internal/app/bindings_novel.go NovelSearch(query) 单参，
// 返回 []NovelSearchHit 扁平切片、每行冗余携带全书汇总）。
// 口径：查询类中性空态——浏览器开发无可检索小说项目，返回空命中切片即可
//（ChapterPage 搜索面板对空结果渲染「无命中」空态，不编造样例命中）。
import type { AppBindings } from "../bridge";

type NovelMethods = Pick<AppBindings, "NovelSearch">;

export function buildNovel(): NovelMethods {
  return {
    async NovelSearch(_query: string) {
      return [];
    },
  };
}