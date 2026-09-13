// mock/novel.ts — 小说域 dev mock（v4.171 批次一 NovelSearch legacy 直调转正；
// 批次二补 12 个 NovelB 门面方法；批次三b 补章节族/叙事状态族/场景族/项目角色族
// 22 个 NovelB 门面方法；批次三c 补 CreatePage 创作间 3 方法；文风指纹批次补
// NovelFingerprintStatus/Build/Score 3 方法——CreatePage「文风指纹」面板）。
// Go 侧对应 NovelB 门面（internal/app/bindings_novel.go，除 ChatWorldview 在
// ChatB 门面同名——返回 map[string]interface{}；GenerateCharacterPortrait 实测
// 在 ImageB 门面同名）。
// 口径：查询类中性空态（浏览器开发无可检索小说项目，返回空/最小样例，不编造
// 全书数据）、动作类 no-op（无小说库可写）、生成类诚实样例（模拟回答/占位插图）。
// 文风指纹为演示口径特例：模块级状态存参考档（初始未构建），Build 后返回诚实
// 假摘要、Score 返回固定示范分——结构必须与契约（bridge/novel.ts 载荷）一致。
import type { AppBindings } from "../bridge";
import type { FingerprintStatusPayload } from "../bridge/novel";

// 文风指纹演示态（模块级：同一会话内 Build 后保持已构建）。
const mockFingerprintStatus: FingerprintStatusPayload = { exists: false };

type NovelMethods = Pick<
  AppBindings,
  | "NovelSearch"
  // 批次二 legacy 直调转正（NovelB/ChatB 门面，同名前缀）。
  | "GetWorldview" | "SaveWorldview" | "ChatWorldview"
  | "GetWorldviewSections" | "SaveAllWorldviewSections"
  | "CheckConsistency" | "CheckConsistencyDeep"
  | "GetForeshadows" | "SaveForeshadows" | "SaveCharactersBatch"
  | "NovelReadingAsk" | "GenerateSceneIllustration"
  // 伏笔一致性体检（ForeshadowPanel「一致性体检」；演示口径允许示范数据）。
  | "LintForeshadows"
  // 批次三b legacy 直调转正（NovelB 门面，同名前缀）：章节族/叙事状态族/场景族/
  // 项目角色族。
  | "GetChapter" | "GetChapterBranch" | "SaveChapterContent" | "SaveChapterBranchContent"
  | "QuickBrainstormBranches" | "CreateChapter" | "DeleteOutlineNode"
  | "GetNovelState" | "BuildNovelStatePatch" | "SettleNovelState"
  | "DeSlopChapterAiTaste" | "RewriteChapterAiTaste" | "GetEntityRelations"
  | "GetChapterScenes" | "GenerateScene" | "CreateScene" | "SaveSceneMeta" | "CancelCreateChapter"
  | "GenerateProjectCharacterFill" | "GenerateCharacterPortrait" | "MergeCharacters"
  | "SaveOrganization" | "DeleteOrganization" | "ToggleOrgMember"
  | "SaveRelationship" | "DeleteRelationship"
  // 文风指纹批次（CreatePage「文风指纹」面板；Go NovelB 门面同名前缀）。
  | "NovelFingerprintStatus" | "NovelFingerprintBuild" | "NovelFingerprintScore"
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
    async LintForeshadows() {
      // 伏笔一致性体检：诚实演示样例（示范书 12 章、登记 5 条，2 条发现）。
      // 结构与契约（bridge/novel.ts ForeshadowLintReport）一致。
      return {
        totalChapters: 12,
        items: 5,
        planted: 3,
        hinted: 1,
        revealed: 1,
        longTerm: 1,
        findings: [
          {
            code: "ordering",
            severity: "high",
            foreshadowId: "manual_demo_1",
            itemDesc: "神秘铜匣的钥匙",
            message: "回收章 001.md 早于埋设章 002.md，登记序颠倒",
            chapter: "001.md",
          },
          {
            code: "stale",
            severity: "medium",
            foreshadowId: "manual_demo_2",
            itemDesc: "主角左臂旧伤的来历",
            message: "已悬置 11 章未回收（第 1 章埋设），考虑回收或标记长线",
            chapter: "001.md",
          },
        ],
      };
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
    // ── 批次三b legacy 直调转正（Go NovelB，同名前缀）────────────────────
    // 章节族：中性空态章节对象（真实实现读项目章节正文/分支）。
    async GetChapter(_num: number) {
      return { content: "（章节内容）" };
    },
    async GetChapterBranch(_num: number, _branch: string) {
      return { content: "" };
    },
    async SaveChapterContent(_num: number, _content: string) {
      // mock: no-op。
    },
    async SaveChapterBranchContent(_num: number, _branch: string, _content: string) {
      // mock: no-op。
    },
    // 批次三c（CreatePage 创作间）：分支构思中性空态/创建章节占位/删除节点 no-op。
    async QuickBrainstormBranches(_setting: string, _prevSummary: string) {
      return { branches: [] };
    },
    async CreateChapter(_setting: string, _prevSummary: string, _plotReq: string, _chapterNum: number, _branchFromNodeID: string, _skillName: string, _minWords: number, _temperature: number) {
      return { ok: true };
    },
    async DeleteOutlineNode(_nodeID: string) {
      // mock: no-op。
    },
    // 叙事状态族：最小样例（真实实现返回叙事状态/结算结果 map）。
    async GetNovelState() {
      return { version: 1, entities: [] };
    },
    async BuildNovelStatePatch(_chapterNum: number) {
      return {};
    },
    async SettleNovelState(_patchJSON: string, _approved: boolean) {
      return { version: 2 };
    },
    async DeSlopChapterAiTaste(_chapterNum: number) {
      return { done: true };
    },
    async RewriteChapterAiTaste(_chapterNum: number) {
      return { done: false };
    },
    async GetEntityRelations() {
      return { nodes: [], edges: [] };
    },
    // 场景族：中性空态/占位场景。
    async GetChapterScenes(_chapterNum: number) {
      return [];
    },
    async GenerateScene(_chapterNum: number, _sceneID: string, _plotReq: string, _minWords: number) {
      return { content: "（场景）", aiTaste: {} };
    },
    async CreateScene(_chapterNum: number, _slug: string, _title: string) {
      return {};
    },
    async SaveSceneMeta(_chapterNum: number, _sceneID: string, _metaJSON: string) {
      // 浏览器演示态无场景落盘：no-op。
    },
    async CancelCreateChapter(_chapterNum: number, _branch: string) {
      return false;
    },
    // 项目角色族：样例 JSON/占位（真实实现读写项目 characters.json）。
    async GenerateProjectCharacterFill(_chJSON: string) {
      return '{"id":"c1"}';
    },
    async GenerateCharacterPortrait(_charID: string, _model: string) {
      return "";
    },
    async MergeCharacters(_keepID: string, _mergeID: string) {
      // Go 实测返回 map（非 void），mock 给空对象。
      return {};
    },
    async SaveOrganization(_orgJSON: string) {
      // mock: no-op。
    },
    async DeleteOrganization(_id: string) {
      // mock: no-op。
    },
    async ToggleOrgMember(_charID: string, _orgID: string) {
      // mock: no-op。
    },
    async SaveRelationship(_relJSON: string) {
      // mock: no-op。
    },
    async DeleteRelationship(_fromID: string, _toID: string) {
      // mock: no-op。
    },
    // ── 文风指纹批次（CreatePage「文风指纹」面板；演示口径允许示范数据，
    // 结构与 bridge/novel.ts 载荷契约一致）──────────────────────────────
    async NovelFingerprintStatus() {
      return { ...mockFingerprintStatus };
    },
    async NovelFingerprintBuild() {
      // 无参动作：置已构建并返回诚实假摘要（3 章 3200 字示范基线）。
      mockFingerprintStatus.exists = true;
      mockFingerprintStatus.builtAt = new Date().toISOString();
      mockFingerprintStatus.chapters = 3;
      mockFingerprintStatus.chars = 3200;
      mockFingerprintStatus.summary = {
        sentenceMean: 18.6,
        sentenceSd: 7.4,
        paraMean: 96.3,
        ttr1000: 412.5,
        dialogRatio: 0.34,
        fourCharRatio: 0.052,
        connectiveDensity: 0.018,
        adjAdvDensity: 0.041,
        topBigrams: ["的时候", "看了一眼"],
        topTrigrams: [],
        authorSignWords: ["却说"],
      };
      return { ...mockFingerprintStatus };
    },
    async NovelFingerprintScore(chapterNum: number) {
      // 固定示范分（42 分「有 AI 痕迹」档 + Δ0.31 基线距离 + 2 条样例命中）。
      return {
        chapterNum,
        refExists: mockFingerprintStatus.exists,
        score: 42,
        delta: 0.31,
        issues: [
          {
            start: 12, end: 40,
            reason: "「不是…而是…」解释腔排比",
            severity: "medium",
            suggestion: "拆成两句直述，或用角色动作收束",
            excerpt: "这不是简单的巧合，而是他早已布下的伏线。",
          },
          {
            start: 88, end: 118,
            reason: "「仿佛在诉说着什么」总结腔收尾",
            severity: "low",
            suggestion: "删去总结性收尾，让细节自己说话",
            excerpt: "夜色沉沉，仿佛在诉说着什么。",
          },
        ],
      };
    },
  };
}