// mock/novel.ts — 小说域 dev mock（v4.171 批次一 NovelSearch legacy 直调转正；
// 批次二补 12 个 NovelB 门面方法）。
// Go 侧对应 NovelB 门面（internal/app/bindings_novel.go，除 ChatWorldview 在
// ChatB 门面同名——返回 map[string]interface{}）。
// 口径：查询类中性空态（浏览器开发无可检索小说项目，返回空/最小样例，不编造
// 全书数据）、动作类 no-op（无小说库可写）、生成类诚实样例（模拟回答/占位插图）。
import type { AppBindings } from "../bridge";

type NovelMethods = Pick<
  AppBindings,
  | "NovelSearch"
  // 批次二 legacy 直调转正（NovelB/ChatB 门面，同名前缀）。
  | "GetWorldview" | "SaveWorldview" | "ChatWorldview"
  | "GetWorldviewSections" | "SaveAllWorldviewSections"
  | "CheckConsistency" | "CheckConsistencyDeep"
  | "GetForeshadows" | "SaveForeshadows" | "SaveCharactersBatch"
  | "NovelReadingAsk" | "GenerateSceneIllustration"
>;

export function buildNovel(): NovelMethods {
  return {
    async NovelSearch(_query: string) {
      return [];
    },
    // ── 批次二 legacy 直调转正（Go NovelB/ChatB，同名前缀）────────────────
    async GetWorldview() {
      // 范例世界观文本（浏览器走查用；真实实现读项目世界观文档）。
      return "蒸汽纪元";
    },
    async SaveWorldview(_content: string) {
      // mock: no-op（浏览器开发无小说项目可持久化）。
    },
    async ChatWorldview(_userMsg: string, _content: string) {
      // 返回结构化世界观回应（对齐 Go ChatB.ChatWorldview 的 map 返回）。
      return { response: "（mock）已结合世界观上下文给出回答", sections: [] };
    },
    async GetWorldviewSections() {
      // 无分节世界观：空对象（消费方空态兜底）。
      return {};
    },
    async SaveAllWorldviewSections(_sectionsJSON: string) {
      // mock: no-op。
    },
    async CheckConsistency() {
      return { consistent: true, issues: [] };
    },
    async CheckConsistencyDeep(_maxChapters: number) {
      return { consistent: true, issues: [], checkedChapters: 0 };
    },
    async GetForeshadows() {
      return { foreshadows: [] };
    },
    async SaveForeshadows(_itemsJSON: string) {
      // mock: no-op。
    },
    async SaveCharactersBatch(namesJSON: string) {
      // 批量角色生成：诚实返回「已保存 N 个名字」（真实实现逐名生成角色）。
      let names: unknown[] = [];
      try { names = JSON.parse(namesJSON) as unknown[]; } catch { /* 坏 JSON 给空 */ }
      return { saved: Array.isArray(names) ? names.length : 0 };
    },
    async NovelReadingAsk(_kind: string, _title: string, _chapterText: string, _selection: string, _question: string, _historyJSON: string) {
      return "模拟回答";
    },
    async GenerateSceneIllustration(_chapterNum: number) {
      return { ok: true, path: ".gaea/play/exports/mock-scene.png" };
    },
  };
}