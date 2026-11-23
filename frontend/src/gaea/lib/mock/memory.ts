// mock/memory.ts — 记忆/知识/资料域（T6-10.1 拆分自 lib/mock.ts，方法体零改动）。
import type { AppBindings } from "../bridge";
import type {
  ProfileFactView,
  SkillSuggestion,
} from "../types";
import { emitMock as emit } from "./shared";
import type { MakeMockState } from "./state";

// 轻语主动式频控配置槽（批 44 NOT_MOCKED 续刀；默认值同 Go defaultProactivePushCfg：
// enabled=true/limitPerHour=3/intervalMin=30/时窗 -1=未启用）。
const whisperProactiveCfg = {
  enabled: true,
  limitPerHour: 3,
  intervalMin: 30,
  quietStartHour: -1,
  quietEndHour: -1,
};

type MemoryMethods = Pick<
  AppBindings,
  | "Memory" | "MemoryArchivedList" | "MemoryCleanupArchived" | "MemoryUnarchive"
  | "MemoryUnarchiveBatch" | "MemorySetRetentionDays"
  | "Remember" | "Forget" | "SaveDoc" | "UpdateFact" | "ChangeFactType"
  | "SetMemoryEnabled" | "MemorySuggestions" | "MorningPreload" | "SetMorningPreload"
  | "AcceptSkillSuggestion" | "AcceptMergeSuggestion"
  | "AIFeedbackList" | "AIFeedbackDelete"
  | "FactBase" | "FactBaseClear" | "FactBasePromote"
  | "MemoryHubOverview" | "ProfileList" | "ProfileSave" | "ProfileDelete"
  | "ProfileConflicts" | "ProfileResolveConflict"
  // 数据库存档（v4.479：本设备聊天记录与上传文件管理）
  | "DatabaseOverview" | "UploadsList" | "DeleteUpload"
| "WhisperMemories" | "WhisperEpisodes" | "WhisperEpisodeReplay" | "WhisperAnchors" | "WhisperAnchorReplay" | "WhisperMemoryRetell" | "WhisperCausalExplain" | "WhisperExportArchive" | "MemoryGraph"
  // 轻语主动式频控配置对（v4.3c play 空间；批 44 NOT_MOCKED 续刀）。
  | "WhisperProactiveConfig" | "WhisperProactiveSetConfig"
  // 轻语图谱子图/立即评估（批 46 NOT_MOCKED 收尾刀）。
  | "WhisperGraphSubgraph" | "WhisperProactiveNow"
  | "MemoryDuplicates" | "MemoryMerge" | "SemanticGraph" | "MemoryPin" | "MemoryFeedback" | "MemoryLifecycle" | "MemoryEvalRun" | "MemoryBrief" | "SetMemoryBrief"
  | "MemoryRecall"
  | "WenxueMemoryBrowse"
  | "MemoryMorningBrief"
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
            pinned: true,
          },
        ],
        enabled: true,
        dreamMode: _s.dreamMode,
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
    async MorningPreload() {
      return _s.morningPreload;
    },
    async SetMorningPreload(enabled: boolean) {
      _s.setMorningPreload(enabled);
    },
    async MemorySuggestions() {
      return { memories: [], skills: [], merges: [], generatedAt: new Date().toISOString(), available: false, source: "mock" };
    },
    async AcceptMergeSuggestion(_keep: string, _archive: string) {
      return "mock-merge";
    },
    async AcceptSkillSuggestion(_candidate: SkillSuggestion) {
      return "mock-skill-path";
    },
    // AI 反馈库（2026-11-20）：browser dev mock 空库（真实库在 Go 侧 jsonl）。
    async AIFeedbackList() {
      return [];
    },
    async AIFeedbackDelete(_id: string) {},
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
      return { profileCount: 4, officeCount: 5, costCount: 3, whisperCount: 5, wenxueCount: 6, pinnedCount: 4, sessionCount: 0, uploadCount: 0, latestUpdated: "刚刚" };
    },
    // ── 数据库存档 mock（v4.479）：浏览器开发态无设备存档，空态/no-op。──
    async DatabaseOverview() {
      return { projects: [], totalSessions: 0, totalArchived: 0, totalUploads: 0, totalBytes: 0 };
    },
    async UploadsList(_root: string) {
      return [];
    },
    async DeleteUpload(_path: string) {
      // mock: no-op——浏览器开发态无文件可删。
    },
    async ProfileList() {
      return [];
    },
    // ── 以下族形参对齐 Go 门面签名（X1-11 契约测试）：浏览器 dev 回退为固定
    // 空态/no-op，实参只收不用（_ 前缀）。行为零变化。──
    async ProfileSave(_f: ProfileFactView) {
      // mock: no-op——浏览器开发环境无人物档案库（真实实现写入记忆中枢 profile 表）。
    },
    async ProfileDelete(_name: string) {
      // mock: no-op——同上，无档案可删。
    },
    async ProfileConflicts() {
      return [];
    },
    async ProfileResolveConflict(_name: string, _prefer: string) {
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
    async WhisperMemoryRetell(_kind: "episode" | "anchor", _id: string, _personalityId: string) {
      return "（mock）那天的雨声、你说话的样子，我都还记得。";
    },
    async WhisperCausalExplain(_entity: string, _personalityId: string) {
      return "（mock）看起来是最近的加班让睡眠变差了。";
    },

    // ── 轻语主动式频控配置对（v4.3c play 空间；批 44 NOT_MOCKED 续刀）──────
    // 模块级配置槽 + Go 同款校验（limitPerHour ≥ 1；intervalMin 10–120；时窗
    // -1 或 0–23），部分字段更新缺省保持原值——面板读写往返可走查。
    async WhisperProactiveConfig() {
      return { ...whisperProactiveCfg };
    },
    async WhisperProactiveSetConfig(cfgJSON: string) {
      let raw: {
        enabled?: boolean; limitPerHour?: number; intervalMin?: number;
        quietStartHour?: number; quietEndHour?: number;
      };
      try {
        raw = JSON.parse(cfgJSON) as typeof raw;
      } catch (e) {
        throw new Error(`解析主动关心配置失败: ${e instanceof Error ? e.message : String(e)}`);
      }
      if (raw.limitPerHour !== undefined) {
        if (raw.limitPerHour < 1) throw new Error("limitPerHour 必须 ≥ 1（关闭请用 enabled=false）");
        whisperProactiveCfg.limitPerHour = raw.limitPerHour;
      }
      if (raw.intervalMin !== undefined) {
        if (raw.intervalMin < 10 || raw.intervalMin > 120) throw new Error("intervalMin 必须在 10–120 分钟");
        whisperProactiveCfg.intervalMin = raw.intervalMin;
      }
      for (const v of [raw.quietStartHour, raw.quietEndHour]) {
        if (v !== undefined && (v < -1 || v > 23)) throw new Error("时窗小时必须为 -1（未启用）或 0–23");
      }
      if (raw.quietStartHour !== undefined) whisperProactiveCfg.quietStartHour = raw.quietStartHour;
      if (raw.quietEndHour !== undefined) whisperProactiveCfg.quietEndHour = raw.quietEndHour;
      if (raw.enabled !== undefined) whisperProactiveCfg.enabled = raw.enabled;
    },
    async MemoryGraph() {
      // 浏览器开发态样例：让记忆中枢的星图/记忆流在无后端时可评审。
      const nodes = [
        { id: "o91", type: "office", name: "振动锤选型规范", desc: "按地质与桩径选型，先查历史台班价。", val: 1.4 },
        { id: "o92", type: "office", name: "钢结构防腐做法", desc: "除锈等级与涂层配套对照表。", val: 1.1 },
        { id: "o93", type: "office", name: "清单计价规则", desc: "综合单价组成与人材机拆分口径。", val: 1.2 },
        { id: "o94", type: "office", name: "混凝土养护要点", desc: "不同气温下的养护时间与覆盖要求。", val: 0.9 },
        { id: "o95", type: "office", name: "安全交底模板", desc: "班前交底六项固定动作。", val: 0.8 },
        { id: "o96", type: "office", name: "图纸会审流程", desc: "会审前置条件与问题闭环。", val: 0.7 },
        { id: "p:1", type: "profile", name: "偏好先看结论", desc: "长回答先给结论，再展开依据。", val: 1.3 },
        { id: "p:2", type: "profile", name: "造价口径偏好", desc: "习惯用本地信息价与历史组价对照。", val: 1.1 },
        { id: "p:3", type: "profile", name: "短句沟通", desc: "不喜欢过度铺陈，句子尽量短。", val: 0.9 },
        { id: "p:4", type: "profile", name: "深夜工作", desc: "夜间更常处理造价与文档。", val: 0.7 },
        { id: "o:1", type: "office", name: "广联达导入注意", desc: "导入前统一清单编码与单位。", val: 1.2 },
        { id: "o:2", type: "office", name: "周报模板", desc: "本周完成 / 风险 / 下周计划三段式。", val: 0.9 },
        { id: "o:3", type: "office", name: "项目 A 组价习惯", desc: "措施费单列，不并入综合单价。", val: 1 },
        { id: "o:4", type: "office", name: "询价流程", desc: "至少三家比价，记录询价时间。", val: 0.8 },
        { id: "o:5", type: "office", name: "交付前自检", desc: "格式 / 数字 / 引用三遍检查。", val: 0.7 },
        { id: "w:1", type: "whisper", name: "深夜聊项目压力", desc: "那晚说进度压得喘不过气，后来一起拆成了三步。", val: 1.3 },
        { id: "w:2", type: "whisper", name: "喜欢的说话方式", desc: "直接、不绕弯，但别冷冰冰。", val: 1.1 },
        { id: "w:3", type: "whisper", name: "一次被安慰的记忆", desc: "方案被推翻后，先被接住情绪再谈修改。", val: 1 },
        { id: "w:4", type: "whisper", name: "生日提醒", desc: "提前一周提醒，不要说教。", val: 0.8 },
        { id: "w:5", type: "whisper", name: "周末计划", desc: "想去看展，顺路吃一家小面。", val: 0.7 },
        { id: "m:1", type: "material", name: "施工组织设计.docx", desc: "项目 A 实施版，含总进度与资源计划。", val: 1.1 },
        { id: "m:2", type: "material", name: "综合单价分析.xlsx", desc: "清单第 3 册，含人材机拆解。", val: 1 },
        { id: "m:3", type: "material", name: "现场照片", desc: "桩基施工与围挡现状。", val: 0.7 },
        { id: "m:4", type: "material", name: "合同条款", desc: "付款节点与变更签证约定。", val: 0.9 },
        { id: "c:1", type: "cost", name: "振动锤台班价", desc: "本地市场 2026-Q3 询价中位值。", val: 1.2 },
        { id: "c:2", type: "cost", name: "钢筋信息价", desc: "HRB400E 综合到货价。", val: 1 },
        { id: "c:3", type: "cost", name: "模板周转", desc: "按 5 次周转摊销。", val: 0.8 },
      ];
      const links = [
        { source: "k:1", target: "c:1", type: "reference" },
        { source: "k:1", target: "o:4", type: "same-tag" },
        { source: "k:2", target: "m:1", type: "reference" },
        { source: "k:3", target: "m:2", type: "reference" },
        { source: "k:3", target: "c:2", type: "reference" },
        { source: "k:3", target: "c:3", type: "reference" },
        { source: "k:4", target: "m:1", type: "same-category" },
        { source: "k:5", target: "o:5", type: "same-tag" },
        { source: "k:6", target: "m:1", type: "same-category" },
        { source: "p:1", target: "o:2", type: "same-tag" },
        { source: "p:1", target: "w:2", type: "same-tag" },
        { source: "p:2", target: "o:3", type: "same-tag" },
        { source: "p:2", target: "c:2", type: "reference" },
        { source: "p:3", target: "w:2", type: "same-tag" },
        { source: "p:4", target: "w:1", type: "same-tag" },
        { source: "o:1", target: "m:2", type: "reference" },
        { source: "o:3", target: "m:2", type: "reference" },
        { source: "o:4", target: "c:1", type: "reference" },
        { source: "o:5", target: "m:4", type: "same-category" },
        { source: "w:1", target: "p:4", type: "same-tag" },
        { source: "w:3", target: "p:1", type: "same-tag" },
        { source: "w:4", target: "p:1", type: "same-tag" },
        { source: "m:1", target: "c:3", type: "same-category" },
        { source: "m:3", target: "m:1", type: "same-tag" },
      ];
      return { nodes, links };
    },
    async WhisperGraphSubgraph(_personalityID: string, _entity: string, _hops: number) {
      // 只读空态（批 46 NOT_MOCKED 收尾刀）：Go 口径「entity 不存在返回空子图
      // （非 nil，空 slice）」。批 50 已在 Go 侧补 json 标签对齐契约小写键
      // （此前无标签序列化大写键，真机面板读契约字段恒空图——观察池疑点收口）。
      return { nodes: [], edges: [] };
    },
    async WhisperProactiveNow(_personalityID: string) {
      // 立即评估（批 46 NOT_MOCKED 收尾刀）：mock 无轻语会话状态机，恒门控
      // 未过（Go 合法返回 {shouldSend:false}，非错误）。
      return { shouldSend: false };
    },
    async MemoryBrief() {
      return true;
    },
    async SetMemoryBrief(_enabled: boolean) {
      // mock：开关写回即成功（面板 toast 由调用方提示）。
    },
    async MemoryFeedback(_messageID: string, _rating: string, _excerpt: string, _space: string) {
      // mock：反馈即成功（真机落 memory_events 事件，事件图谱可查）。
      return undefined;
    },
    async MemoryPin(_name: string, _pinned: boolean) {
      // mock：固化切换成功即回（面板 reload 后 pinned 徽标随 mock 数据展示）。
    },
    async MemoryLifecycle() {
      return {
        pinned: [
          { name: "cost-rule", title: "造价规则", description: "组价先查历史价", kind: "semantic", type: "feedback", space: "work", score: 1, daysIdle: 0, lastUsedAt: "", stabilityDays: 67.5, citeCount: 2 },
        ],
        decaying: [
          { name: "old-note", title: "旧偏好", description: "很久没被引用的项目笔记", kind: "episodic", type: "project", space: "work", score: 0.12, daysIdle: 118, lastUsedAt: "", stabilityDays: 30, citeCount: 0, dormant: true },
        ],
        activeCount: 5,
        archivedCount: 2,
        retentionDays: 90,
        staleAfterDays: 60,
      };
    },
    async MemoryEvalRun() {
      // mock：全合规小样例（结构不变量体检，确定性无模型调用）。
      return {
        memoryEnabled: true,
        morningPreload: true,
        projectBrief: true,
        spaceModeOn: true,
        preloadPresent: true,
        briefPresent: true,
        preloadRunes: 96,
        briefRunes: 148,
        preloadBudget: 600,
        briefBudget: 600,
        entryCount: 2,
        refCount: 2,
        pinnedTotal: 1,
        pinnedInBrief: 1,
        violations: [],
        passed: true,
      };
    },
    async SemanticGraph() {
      // mock：entity/event/source 三向边的小样例（事件日志投影形态，节点只带
      // 视图五字段 id/name/type/desc/val，状态/口径文案进 desc）。
      const nodes = [
        { id: "mem:mock/cost-rule", type: "entity", name: "造价规则", desc: "active · feedback · work · 组价先查历史价", val: 1 },
        { id: "mem:mock/price-band", type: "entity", name: "价格带", desc: "active · project · work · P25/中位/P75", val: 1 },
        { id: "mem:mock/style-note", type: "entity", name: "文风偏好", desc: "archived · project · play", val: 1 },
        { id: "ev:1", type: "event", name: "save · 造价规则", desc: "组价先查历史价", val: 0.5 },
        { id: "ev:2", type: "event", name: "cite · 价格带", desc: "", val: 0.5 },
        { id: "ev:3", type: "event", name: "archive · 文风偏好", desc: "", val: 0.5 },
        { id: "src:mock/local", type: "source", name: "本机写入", desc: "", val: 0.8 },
        { id: "src:mock/session-x", type: "source", name: "session-x", desc: "", val: 0.8 },
      ];
      const links = [
        { source: "src:mock/local", target: "ev:1", type: "produces" },
        { source: "ev:1", target: "mem:mock/cost-rule", type: "affects" },
        { source: "src:mock/session-x", target: "ev:2", type: "produces" },
        { source: "ev:2", target: "mem:mock/price-band", type: "affects" },
        { source: "mem:mock/cost-rule", target: "mem:mock/price-band", type: "references" },
        { source: "src:mock/local", target: "ev:3", type: "produces" },
        { source: "ev:3", target: "mem:mock/style-note", type: "affects" },
      ];
      return { nodes, links, eventCount: 3, entityCount: 3, danglingRefs: ["ghost-key"], projectedSeq: 3 };
    },
    // ── 知识库导入（mock）──
    async MemoryRecall(query: string, _limit: number) {
      return { query, total: 0, hits: [], degraded: { semantic: "dev mock：无 embedding" } };
    },
    async MemoryDuplicates(_min: number) {
      return [
        {
          keep: "pile", keepTitle: "桩基施工要点",
          dup: "pile-v2", dupTitle: "桩基施工 要点（修订）",
          score: 0.87,
        },
      ];
    },
    async MemoryMerge(targetName: string, _sourceNames: string[]) {
      return targetName;
    },
    async MemoryMorningBrief() {
      // mock: 返回「今日晨报」JSON 串（对齐 Go 侧 GaeaMemoryMorningBrief 契约，
      // 前端 JSON.parse 后渲染；items 为空时组件静默隐藏）。
      const now = Date.now();
      return JSON.stringify({
        items: [
          { name: "示例工作记忆", description: "办公会话沉淀的示例事实：本地优先、零 LLM 晨报预取", kind: "semantic", category: "project", updatedAt: now - 3600_000, lastUsedAt: now },
        ],
        rules: ["晨报只读 work 空间记忆，play 不渲染"],
        dreamed24h: 0,
        generatedAt: now,
      });
    },
    async WhisperExportArchive(_dir: string): Promise<number> {
      // mock: no-op——无真实 whisper 库可打包，返回导出条目数 0。
      return 0;
    },
    // WenxueMemoryBrowse mock：图文脑独立库，mock 环境无图文引擎 → 诚实空态。
    async WenxueMemoryBrowse() {
      return { available: false, storeDir: "", count: 0, facts: [] };
    },
  };
}

