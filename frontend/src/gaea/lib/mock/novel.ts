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
import type { ChapterReviewPayload, FingerprintStatusPayload, ReviewPlatform } from "../bridge/novel";

// 文风指纹演示态（模块级：同一会话内 Build 后保持已构建）。
const mockFingerprintStatus: FingerprintStatusPayload = { exists: false };

type NovelMethods = Pick<
  AppBindings,
  | "NovelSearch"
  // sin 书源 t5「送小说导入」消费（浏览器无桌面文件系统，如实拒绝）。
  | "ImportNovelBookEx"
  // 批次二 legacy 直调转正（NovelB/ChatB 门面，同名前缀）。
  | "GetWorldview" | "SaveWorldview" | "ChatWorldview"
  | "GetWorldviewSections" | "SaveAllWorldviewSections"
  | "CheckConsistency" | "CheckConsistencyDeep"
  | "GetForeshadows" | "SaveForeshadows" | "SaveCharactersBatch"
  | "NovelReadingAsk" | "GenerateSceneIllustration"
  // 伏笔一致性体检（ForeshadowPanel「一致性体检」；演示口径允许示范数据）。
  | "LintForeshadows"
  // 伏笔清理/统计/同步结果批次（t1-P3 后端 + t1-P4 面板消费；浏览器演示口径）。
  | "GetForeshadowStats" | "GetLastForeshadowSync"
  | "DeleteChapterForeshadows" | "CleanChapterAnalysisForeshadows"
  | "ClearProjectForeshadowsForReset"
  // 批次三b legacy 直调转正（NovelB 门面，同名前缀）：章节族/叙事状态族/场景族/
  // 项目角色族。
  | "GetChapter" | "GetChapterBranch" | "SaveChapterContent" | "SaveChapterBranchContent"
  | "QuickBrainstormBranches" | "CreateChapter" | "DeleteOutlineNode"
  | "GetNovelState" | "BuildNovelStatePatch" | "SettleNovelState"
  | "DeSlopChapterAiTaste" | "RewriteChapterAiTaste" | "GetEntityRelations"
  | "GetChapterScenes" | "GenerateScene" | "CreateScene" | "SaveSceneMeta" | "CancelCreateChapter"
  | "GenerateProjectCharacterFill" | "GenerateCharacterPortrait" | "MergeCharacters"
  | "SaveOrganization" | "DeleteOrganization" | "ToggleOrgMember"
  | "SetCharacterCareer" | "RemoveCharacterCareer"
  | "SaveRelationship" | "DeleteRelationship"
  // 文风指纹批次（CreatePage「文风指纹」面板；Go NovelB 门面同名前缀）。
  | "NovelFingerprintStatus" | "NovelFingerprintBuild" | "NovelFingerprintScore"
  // 平台评审批次（v4.282，oh-story 蒸馏 T1；CreatePage「平台评审」面板）。
  | "NovelReviewPlatforms" | "NovelChapterReview"
  // 驱动式整章重写批次（t4-C3；浏览器演示口径）。
  | "NovelChapterRewrite" | "NovelChapterSuggestions"
  | "NovelListRewriteVersions" | "NovelGetRewriteVersion"
  | "NovelApplyRewriteVersion" | "NovelDiscardRewriteVersion"
  | "NovelRestoreRewriteVersion"
  // 书源在线搜书批次（v4.283，书源取书→拆书导入 t2；HomePage「在线搜书」）。
  | "NovelBookSourceSearch" | "NovelBookSourceToc"
  | "NovelBookSourceImport" | "NovelBookSourceImportCancel"
  | "NovelBookSourceImportChapters"
  | "NovelBookSourceEnginesGet" | "NovelBookSourceEnginesSave"
>;

export function buildNovel(): NovelMethods {
  return {
    async ImportNovelBookEx() {
      throw new Error("dev mock：导入需桌面端文件系统");
    },
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
    async NovelChapterSuggestions(_chapterNum: number) {
      // 浏览器演示无分析产物：空数组（UI 提示先分析，锁自定义指令）。
      return [];
    },
    async NovelChapterRewrite(_chapterNum: number, _reqJSON: string) {
      throw new Error("dev mock：整章重写需本地模型调用");
    },
    async NovelListRewriteVersions(_chapterNum: number) {
      return [];
    },
    async NovelGetRewriteVersion(_chapterNum: number, _versionID: string) {
      throw new Error("dev mock：无重写版本");
    },
    async NovelApplyRewriteVersion(_chapterNum: number, _versionID: string) {
      throw new Error("dev mock：无重写版本可应用");
    },
    async NovelDiscardRewriteVersion(_chapterNum: number, _versionID: string) {
      throw new Error("dev mock：无重写版本可丢弃");
    },
    async NovelRestoreRewriteVersion(_chapterNum: number, _versionID: string) {
      throw new Error("dev mock：无重写版本可恢复");
    },
    async GetForeshadows() {
      return { items: [], currentChapter: 0 };
    },
    async SaveForeshadows(_itemsJSON: string) {
      // mock: no-op。
    },
    async GetForeshadowStats(_currentChapter: number) {
      // 伏笔统计：浏览器演示无真实登记表，全 0（诚实空态）。
      return {
        total: 0, pending: 0, planted: 0, hinted: 0, resolved: 0,
        partiallyResolved: 0, abandoned: 0, longTermCount: 0,
        overdueCount: 0, currentChapter: 0,
      };
    },
    async GetLastForeshadowSync() {
      // 尚未分析：如实 reject（消费方 catch 后显示「尚未执行分析」）。
      throw new Error("dev mock：尚未执行过章节分析");
    },
    async DeleteChapterForeshadows(_chapterFile: string, _onlyAnalysisSource: boolean) {
      return { deleted: 0 };
    },
    async CleanChapterAnalysisForeshadows(_chapterFile: string) {
      return { deleted: 0, rolledBack: 0 };
    },
    async ClearProjectForeshadowsForReset() {
      return { deleted: 0, resetManual: 0 };
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
    async SetCharacterCareer(_charID: string, _reqJSON: string) {
      // mock: no-op（浏览器走查用演示数据，职业设置落真项目需桌面端）。
    },
    async RemoveCharacterCareer(_charID: string, _reqJSON: string) {
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
    // ── 平台评审批次（v4.282；演示口径允许示范数据，结构与 bridge/novel.ts 契约一致）──
    async NovelReviewPlatforms(): Promise<ReviewPlatform[]> {
      return [
        { id: "general", label: "通用", form: "chapter" },
        { id: "fanqie", label: "番茄小说", form: "chapter" },
        { id: "qidian", label: "起点中文网", form: "chapter" },
        { id: "zhihu", label: "知乎盐言故事", form: "story" },
      ];
    },
    async NovelChapterReview(chapterNum: number, platform: string): Promise<ChapterReviewPayload> {
      // 示范报告：一条 S1（预告式收尾）+ 一条 S2（开篇钩子）+ 一条 S3 + 一条 pass + 一条 skip，
      // 覆盖面板的全部渲染分支（结论 CONCERNS/REJECT、证据摘录、skip 说明）。
      const label = platform === "fanqie" ? "番茄小说" : platform === "qidian" ? "起点中文网" : platform === "zhihu" ? "知乎盐言故事" : "通用";
      return {
        chapterNum,
        platform,
        platformLabel: label,
        words: 2380,
        verdict: "REJECT",
        counts: { S1: 1, S2: 1, S3: 1, S4: 0 },
        dimensions: [
          {
            id: "opening_freshness", label: "开篇钩子", verdict: "warn", severity: "S2",
            detail: "前 3 段只有悬念铺垫，没有冲突或台词落地",
            advice: "开篇把悬念变成当场发生的事：一句对话、一次阻拦、一个具体麻烦。",
            evidence: [{ paragraph: 1, excerpt: "夜色很深。风从窗缝里钻进来。" }],
          },
          {
            id: "trailer_ending", label: "预告式收尾", verdict: "fail", severity: "S1",
            detail: "章尾出现预告/总结腔：才刚刚开始",
            advice: "删掉「才刚刚开始 / 没人知道 / 命运的齿轮」式收束，把悬念落到具体动作或物件上。",
            evidence: [{ paragraph: 42, excerpt: "属于他的反击，才刚刚开始。" }],
          },
          {
            id: "format_readability", label: "段落节奏", verdict: "warn", severity: "S3",
            detail: "段落节奏：3 段超过 150 字（平均 78.2 字，最长 246 字）",
            advice: "长短交错：冲突处短段推进，沉淀处可长段，别通篇同长。",
            evidence: [{ paragraph: 17, excerpt: "他把三张单据摊在桌上，一张一张码齐……" }],
          },
          {
            id: "punctuation_rhythm", label: "标点节奏", verdict: "pass",
            detail: "标点节奏正常（省略号 1.2 次/千字、问号 6、感叹号 2）",
          },
          {
            id: "protagonist_presence", label: "主角存在感", verdict: "skip",
            detail: "角色库未标注主角，跳过（在角色库把主要角色标为「主角」后本维度生效）",
          },
        ],
        advisories: [
          "读者为什么翻下一页？答不出至少记 S2。",
          "本章改变了什么？情节、关系、信息、情绪至少改变一项。",
          "哪个原文证据支持你的判断？没有证据的结论不采纳。",
        ],
      };
    },

    // ── 书源在线搜书批次（v4.283）────────────────────────────────────
    // 浏览器 dev 无真实网络与书源规则目录：查询类诚实空态（带说明，不编造
    // 候选），动作类如实拒绝——在线导入只在 Wails 壳内可用。
    async NovelBookSourceSearch(_keyword: string) {
      return {
        candidates: [],
        warnings: ["浏览器 mock 模式：书源搜索需要真实网络与规则目录，请在应用内使用"],
      };
    },
    async NovelBookSourceToc(_source: string, _detailURL: string) {
      return { total: 0, sample: [], truncated: false };
    },
    async NovelBookSourceImport(): Promise<{ jobId: string }> {
      throw new Error("浏览器 mock 模式不支持在线导入（无网络与书源规则），请在应用内使用");
    },
    async NovelBookSourceImportCancel(_jobId: string): Promise<boolean> {
      return false;
    },
    async NovelBookSourceImportChapters(): Promise<{ jobId: string }> {
      throw new Error("浏览器 mock 模式不支持失败章补下（无网络与书源规则），请在应用内使用");
    },
    async NovelBookSourceEnginesGet() {
      return {
        path: "C:/mock/gaea/booksource/rules/websearch-engines.json",
        rules: [
          { name: "bing", url: "https://www.bing.com/search?q=%s", queryFormat: "%s 小说 免费阅读", result: ".b_algo", title: "h2 a" },
          { name: "ddg-html", url: "https://html.duckduckgo.com/html/?q=%s", result: ".result", title: ".result__a", linkParam: "uddg", disabled: true },
        ],
      };
    },
    async NovelBookSourceEnginesSave(_rulesJSON: string): Promise<number> {
      throw new Error("浏览器 mock 模式不支持保存引擎规则（无规则目录），请在应用内使用");
    },
  };
}
