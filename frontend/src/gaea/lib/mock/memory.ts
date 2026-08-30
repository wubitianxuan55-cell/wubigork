// mock/memory.ts — 记忆/知识/资料域（T6-10.1 拆分自 lib/mock.ts，方法体零改动）。
import type { AppBindings } from "../bridge";
import type {
  KnowledgeEntry,
  KnowledgeSaveRequest,
  KnowledgeSummary,
  MemorySuggestion,
  SkillSuggestion,
} from "../types";
import { emit, pinnedMock } from "./shared";
import type { MakeMockState } from "./state";

type MemoryMethods = Pick<
  AppBindings,
  | "Memory" | "MemoryArchivedList" | "MemoryCleanupArchived" | "MemoryUnarchive"
  | "MemoryUnarchiveBatch" | "MemorySetRetentionDays"
  | "Remember" | "Forget" | "SaveDoc" | "UpdateFact" | "ChangeFactType"
  | "SetMemoryEnabled" | "MemorySuggestions"
  | "AcceptMemorySuggestion" | "AcceptSkillSuggestion"
  | "FactBase" | "FactBaseClear" | "FactBasePromote"
  | "MemoryHubOverview" | "ProfileList" | "ProfileSave" | "ProfileDelete"
  | "ProfileConflicts" | "ProfileResolveConflict"
| "WhisperMemories" | "WhisperEpisodes" | "WhisperEpisodeReplay" | "WhisperAnchors" | "WhisperAnchorReplay" | "WhisperMemoryRetell" | "WhisperCausalExplain" | "WhisperExportArchive" | "MemoryGraph"
  | "KnowledgeList" | "KnowledgeSearch" | "KnowledgeGet" | "KnowledgeSave" | "KnowledgeDelete"
  | "KnowledgeImportPreview" | "KnowledgeImportAIParse" | "KnowledgeImportApply"
  | "KnowledgeHistory" | "KnowledgeFindSimilar" | "KnowledgeExport" | "KnowledgeReview" | "KnowledgeMerge"
  | "MemoryDuplicates" | "MemoryMerge"
>;

export function buildMemory(_s: MakeMockState): MemoryMethods {
  return {
    async Memory() {
      return {
        available: true,
        storeDir: "~/.config/gaea/projects/-mock/memory",
        docs: [
          {
            path: "REASONIX.md",
            scope: "project",
            body: "# gaea project memory\n\nMock doc shown in the browser dev seam.\n\n## Notes\n\n- prefers concise replies",
          },
          {
            path: "~/.config/gaea/REASONIX.md",
            scope: "user",
            body: "# User memory\n\nAlways respond in 中文.",
          },
        ],
        facts: [
          {
            name: "prefers-tabs",
            description: "User prefers tabs",
            type: "user",
            body: "Indent with tabs.",
            lastUsedAt: new Date(Date.now() - 3 * 86400000).toISOString(),
            sourceSession: "session-mock-demo.jsonl",
            sourceMessage: "turn 2",
          },
        ],
        enabled: true,
        scopes: [
          { scope: "user", path: "~/.config/gaea/REASONIX.md" },
          { scope: "project", path: "REASONIX.md" },
          { scope: "local", path: "REASONIX.local.md" },
        ],
      };
    },
    async MemoryArchivedList(limit: number, offset: number) {
      // mock: 演示归档分页（含 retentionDays，对齐 GaeaMemoryArchivedList）。
      const items = [
        {
          name: "pile-v2",
          title: "桩基施工 要点（修订）",
          description: "振动锤选型需匹配地质条件（旧版重复归档）",
          type: "reference",
          kind: "semantic",
          archivedAt: new Date(Date.now() - 3 * 86400000).toISOString(),
        },
        {
          name: "cost-2025",
          title: "2025 年机械台班价（已过期）",
          description: "过期的价格快照，保留期内待清理",
          type: "project",
          kind: "episodic",
          archivedAt: new Date(Date.now() - 40 * 86400000).toISOString(),
        },
      ];
      const total = items.length;
      const page = items.slice(offset, offset + limit);
      return { items: page, total, limit, offset, retentionDays: 90 };
    },
    async MemoryCleanupArchived() {
      // mock: 无真实归档库可清理，返回删除条数 0（Go 侧为清理条目计数）。
      return 0;
    },
    async MemoryUnarchive(name: string) {
      // mock: no-op——浏览器开发环境无持久化归档库（真实实现恢复事实回活跃列表）。
      emit({ kind: "notice", level: "info", text: `unarchived → ${name}` });
    },
    async MemoryUnarchiveBatch(names: string[]) {
      // mock: 批量恢复返回成功数（全部成功）。
      emit({ kind: "notice", level: "info", text: `batch unarchived → ${names.join(", ")}` });
      return names.length;
    },
    async MemorySetRetentionDays(days: number) {
      // mock: no-op——浏览器开发环境无持久化配置（真实实现写 config 并生效）。
      emit({ kind: "notice", level: "info", text: `retention → ${days} 天` });
    },
    async Remember(scope: string, note: string) {
      emit({ kind: "notice", level: "info", text: `remembered → ${scope}` });
      return `${scope} REASONIX.md (mock): ${note}`;
    },
    async Forget(name: string) {
      emit({ kind: "notice", level: "info", text: `forgot → ${name}` });
    },
    async SaveDoc(path: string, _body: string) {
      emit({ kind: "notice", level: "info", text: `saved → ${path}` });
      return path;
    },
    async UpdateFact(name: string, _body: string) {
      emit({ kind: "notice", level: "info", text: `updated → ${name}` });
      return name;
    },
    async ChangeFactType(name: string, typ: string) {
      emit({ kind: "notice", level: "info", text: `type changed → ${name} (${typ})` });
      return name;
    },
    async SetMemoryEnabled(enabled: boolean) {
      emit({ kind: "notice", level: "info", text: `memory ${enabled ? "enabled" : "disabled"}` });
    },
    async MemorySuggestions() {
      return { memories: [], skills: [], generatedAt: new Date().toISOString(), available: false, source: "mock" };
    },
    async AcceptMemorySuggestion(_candidate: MemorySuggestion) {
      return "mock-memory-path";
    },
    async AcceptSkillSuggestion(_candidate: SkillSuggestion) {
      return "mock-skill-path";
    },
    async FactBase() {
      return { facts: [], markdown: "", count: 0, path: "" };
    },
    async FactBaseClear() {
      // browser dev mock: nothing to clear（无持久化事实库）
    },
    async FactBasePromote() {
      return 0; // browser dev mock: no persistent memory（无事实可提升，返回 0 条）
    },
    // ── 记忆中枢 Mock ────────────────────────────────────────
    async MemoryHubOverview() {
      return { knowledgeCount: 4, profileCount: 0, officeCount: 0, costCount: 2, whisperCount: 0, pinnedCount: pinnedMock.length, latestUpdated: "" };
    },
    async ProfileList() {
      return [];
    },
    async ProfileSave() {
      // mock: no-op——浏览器开发环境无人物档案库（真实实现写入记忆中枢 profile 表）。
    },
    async ProfileDelete() {
      // mock: no-op——同上，无档案可删。
    },
    async ProfileConflicts() {
      return [];
    },
    async ProfileResolveConflict() {
      // mock: no-op——无冲突档案可仲裁。
    },
    async WhisperMemories() {
      return [];
    },
    async WhisperEpisodes() {
      return [];
    },
    async WhisperEpisodeReplay(episodeId: string) {
      return {
        id: episodeId, summary: "", dominantEmotion: "", emotionalIntensity: 0,
        keywords: [], createdAt: "", sourceSessionId: "", startTurn: 0, endTurn: 0,
        dialogue: [], replayable: false,
      };
    },
    async WhisperAnchors() {
      return [];
    },
    async WhisperAnchorReplay(anchorId: string) {
      return {
        anchorId, anchorDate: "", anchorType: "", domain: "", summary: "",
        emotionalValence: 0, emotionalIntensity: 0, linkedFactSummaries: [],
        replayable: false,
      };
    },
    async WhisperMemoryRetell() {
      return "（mock）那天的雨声、你说话的样子，我都还记得。";
    },
    async WhisperCausalExplain() {
      return "（mock）看起来是最近的加班让睡眠变差了。";
    },
    async MemoryGraph() {
      return { nodes: [], links: [] };
    },
    async KnowledgeList(): Promise<KnowledgeSummary[]> {
      return [
        { name: "gb50300-2024", title: "建筑工程施工质量验收统一标准 GB 50300-2024", category: "规范标准", tags: ["施工", "质量", "验收"], status: "现行", updatedAt: "2025-01-15T00:00:00.000Z" },
        { name: "case-bio-remediation", title: "某焦化厂生物修复工程案例", category: "工程案例", tags: ["焦化厂", "生物修复", "PAHs"], status: "已归档", updatedAt: "2024-11-20T00:00:00.000Z" },
        { name: "soil-sampling-guide", title: "污染场地土壤采样技术要点", category: "经验总结", tags: ["采样", "布点", "质量控制"], status: "常用", updatedAt: "2025-02-10T00:00:00.000Z" },
        { name: "hdp-liner-spec", title: "HDPE 土工膜施工技术规范", category: "材料工艺", tags: ["HDPE", "土工膜", "防渗"], status: "现行", updatedAt: "2024-09-05T00:00:00.000Z" },
      ];
    },
    async KnowledgeSearch(query: string, category: string, phase: string, status: string): Promise<KnowledgeSummary[]> {
      let list = await this.KnowledgeList();
      if (category && category !== "all") list = list.filter((e) => e.category === category);
      if (status && status !== "all") list = list.filter((e) => e.status === status);
      if (query) {
        const q = query.trim().toLowerCase();
        list = list.filter((e) => [e.title, e.name, e.category, ...e.tags].join(" ").toLowerCase().includes(q));
      }
      return list;
    },
    async KnowledgeGet(name: string): Promise<KnowledgeEntry | null> {
      const entries: Record<string, KnowledgeEntry> = {
        "gb50300-2024": {
          name: "gb50300-2024", title: "建筑工程施工质量验收统一标准 GB 50300-2024", category: "规范标准", tags: ["施工", "质量", "验收"], status: "现行", updatedAt: "2025-01-15T00:00:00.000Z",
          body: "## 适用范围\n\n本标准适用于建筑工程施工质量的验收，包括地基与基础、主体结构、建筑装饰装修、建筑屋面、建筑给排水及供暖、通风与空调、建筑电气、智能建筑、建筑节能、电梯等分部工程。\n\n## 基本规定\n\n1. 施工现场质量管理应有相应的技术标准。\n2. 建筑工程施工质量应按下列要求进行验收。\n3. 建筑工程施工质量验收应划分为单位工程、分部工程、分项工程和检验批。",
          phase: "施工验收", discipline: "土木工程", source: "住房和城乡建设部", version: 2, author: "住建部标准定额司", reviewer: "", createdAt: "2024-06-01T00:00:00.000Z",
        },
        "case-bio-remediation": {
          name: "case-bio-remediation", title: "某焦化厂生物修复工程案例", category: "工程案例", tags: ["焦化厂", "生物修复", "PAHs"], status: "已归档", updatedAt: "2024-11-20T00:00:00.000Z",
          body: "## 项目概况\n\n某焦化厂退役地块，占地面积约 120 亩。主要污染物为多环芳烃（PAHs）、苯系物（BTEX）和氰化物。\n\n## 修复方案\n\n采用原位生物通风+化学氧化联合修复工艺。\n- 生物通风：注入空气和营养盐，促进土著微生物降解\n- 化学氧化：注射过硫酸钠氧化高浓度区域\n\n## 修复效果\n\n经过 18 个月的修复运行，目标污染物去除率达到 85% 以上，达到修复目标值。",
          phase: "施工", discipline: "环境工程", source: "内部案例库", version: 1, author: "张三", reviewer: "", createdAt: "2024-06-15T00:00:00.000Z",
        },
        "soil-sampling-guide": {
          name: "soil-sampling-guide", title: "污染场地土壤采样技术要点", category: "经验总结", tags: ["采样", "布点", "质量控制"], status: "常用", updatedAt: "2025-02-10T00:00:00.000Z",
          body: "## 采样前准备\n\n1. 收集场地历史资料，了解潜在污染物类型\n2. 制定采样方案，明确布点方法和数量\n3. 准备采样设备、样品容器和现场记录表\n\n## 布点方法\n\n- 系统布点法：适用于污染物分布均匀的场地\n- 分层布点法：适用于污染来源明确的场地\n- 判断布点法：适用于历史污染区域\n\n## 质量控制\n\n- 现场平行样：每 10 个样品至少 1 个\n- 运输空白样：每批次至少 1 个\n- 设备清洗样：每个采样点之间采集",
          phase: "调查", discipline: "环境工程", source: "项目经验总结", version: 3, author: "李四", reviewer: "", createdAt: "2024-08-01T00:00:00.000Z",
        },
        "hdp-liner-spec": {
          name: "hdp-liner-spec", title: "HDPE 土工膜施工技术规范", category: "材料工艺", tags: ["HDPE", "土工膜", "防渗"], status: "现行", updatedAt: "2024-09-05T00:00:00.000Z",
          body: "## 材料要求\n\nHDPE 土工膜厚度不应小于 1.5mm，密度不低于 0.94g/cm³。\n\n## 施工要点\n\n1. 基底应平整压实，无尖锐物\n2. 膜与膜之间采用热熔焊接\n3. 焊缝强度不低于母材强度\n4. 铺设时应预留 5%-8% 的伸缩余量\n\n## 质量检验\n\n- 目测检查：膜面有无破损、褶皱\n- 气密性试验：焊缝处进行气压测试\n- 厚度检测：每 500m² 测一点",
          phase: "施工", discipline: "岩土工程", source: "施工技术手册", version: 2, author: "王五", reviewer: "", createdAt: "2024-07-01T00:00:00.000Z",
        },
      };
      return entries[name] || null;
    },
    // ── 知识库 CRUD Mock ────────────────────────────────────
    async KnowledgeSave(_entry: KnowledgeSaveRequest) {
      // mock: no-op——浏览器开发环境无持久化知识库（真实实现写 SQLite）。
    },
    async KnowledgeDelete(_name: string) {
      // mock: no-op——同上，无库可删。
    },
    // ── 知识库导入（mock）──
    async KnowledgeImportPreview(path: string) {
      return {
        path,
        fileName: path.split(/[\\/]/).pop() ?? path,
        columns: ["标题", "分类", "正文"],
        unmapped: [],
        rows: [
          {
            name: "gb36600", title: "GB 36600 风险管控", category: "规范标准", phase: "", discipline: "",
            tags: [], status: "现行", source: path.split(/[\\/]/).pop() ?? path,
            body: "建设用地土壤污染风险管控标准要点…",
            existingName: "", matchNote: "新增", similarName: "", similarNote: "",
            raw: "", skip: false, skipReason: "",
          },
          {
            name: "", title: "桩基施工要点", category: "工程案例", phase: "施工", discipline: "岩土工程",
            tags: ["振动锤", "桩基"], status: "现行", source: path.split(/[\\/]/).pop() ?? path,
            body: "振动锤选型需匹配地质条件…",
            existingName: "pile", matchNote: "将覆盖更新", similarName: "", similarNote: "",
            raw: "", skip: false, skipReason: "",
          },
        ],
        message: "",
        aiUsed: false,
      };
    },
    async KnowledgeImportAIParse(path: string) {
      const pv = await this.KnowledgeImportPreview(path);
      pv.aiUsed = true;
      pv.message = "AI 智能解析完成，请核对后确认导入。";
      return pv;
    },
    async KnowledgeImportApply() {
      return 0;
    },
    async KnowledgeHistory(name: string) {
      return [
        {
          name, title: "桩基施工要点", version: 2, category: "工程案例", phase: "施工",
          discipline: "岩土工程", tags: ["振动锤", "桩基"], status: "现行",
          author: "", reviewer: "", source: "导入文件",
          body: "旧版正文：振动锤选型需匹配地质条件…",
          changedAt: new Date().toISOString(), note: "内容更新",
        },
      ];
    },
    async KnowledgeFindSimilar(title: string) {
      if (!title.trim()) return [];
      return [{ name: "pile", title: "桩基施工要点", score: 0.87 }];
    },
    async KnowledgeExport() {
      return 4;
    },
    async KnowledgeReview() {
      // mock: no-op——浏览器开发环境无审核流状态（真实实现更新条目 review 状态）。
    },
    async KnowledgeMerge(_target: string, _sources: string[]) {
      return _target;
    },
    async MemoryDuplicates() {
      return [
        {
          keep: "pile", keepTitle: "桩基施工要点",
          dup: "pile-v2", dupTitle: "桩基施工 要点（修订）",
          score: 0.87,
        },
      ];
    },
    async MemoryMerge(target: string) {
      return target;
    },
    async WhisperExportArchive(_dir: string): Promise<number> {
      // mock: no-op——无真实 whisper 库可打包，返回导出条目数 0。
      return 0;
    },
  };
}
