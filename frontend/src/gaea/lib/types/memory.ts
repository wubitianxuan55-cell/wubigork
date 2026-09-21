import type { WireShape } from "./shared";
import type { app as AppModels } from "../../../../wailsjs/go/models";

// Memory panel payloads (desktop/app.go MemoryView).
export type MemoryDoc = WireShape<AppModels.MemoryDoc>;

export type MemoryFact = WireShape<AppModels.MemoryFact>;

export type MemoryScope = WireShape<AppModels.MemoryScope>;

export interface MemoryView {
  docs: MemoryDoc[];
  facts: MemoryFact[];
  scopes: MemoryScope[];
  storeDir: string;
  available: boolean;
  enabled?: boolean; // 记忆开关（当前生效值）
  dreamMode?: string; // 自动做梦模式（off/suggest/auto，v4.377 建议制默认 suggest）
  archives?: MemoryArchive[];
}
export interface MemoryArchive {
  name: string;
  title?: string;
  description: string;
  type: string;
  body: string;
  path?: string;
  archivedAt?: string;
}

// MemoryArchivedView 是办公记忆归档列表中的一条（GaeaMemoryArchivedList）：
// 归档超过 90 天的事实为硬删除候选（GaeaMemoryCleanupArchived 清理）。
export type MemoryArchivedView = WireShape<AppModels.MemoryArchivedView>;


export type MemoryLifecycleView = WireShape<AppModels.MemoryLifecycleView>;

// MemoryEvalReport 是记忆注入体检报告（GaeaMemoryEvalRun，市场调研候选2）。
// 手写接口（wailsjs models 为生成物不随源入库，MemoryArchivedPage 同口径）。
export interface MemoryEvalReport {
  memoryEnabled: boolean;
  morningPreload: boolean;
  projectBrief: boolean;
  spaceModeOn: boolean;
  preloadPresent: boolean;
  briefPresent: boolean;
  preloadRunes: number;
  briefRunes: number;
  preloadBudget: number;
  briefBudget: number;
  entryCount: number;
  refCount: number;
  pinnedTotal: number;
  pinnedInBrief: number;
  missingPinned?: string[];
  violations?: string[];
  passed: boolean;
  note?: string;
}

// MemoryArchivedPage 是归档列表分页结果（GaeaMemoryArchivedList）。
export interface MemoryArchivedPage {
  items: MemoryArchivedView[];
  total: number;
  limit: number;
  offset: number;
  /** 归档保留期（天），后端 gaea_memory_lifecycle.go RetentionDays 下发；mock 未带时 UI 回退 90 */
  retentionDays?: number;
}

export interface MemorySuggestion {
  id: string;
  name: string;
  title?: string;
  description: string;
  type: string;
  body: string;
  reason: string;
  evidence: string[];
}

export interface SkillSuggestion {
  id: string;
  name: string;
  description: string;
  scope: string;
  body: string;
  reason: string;
  evidence: string[];
}

export type MemorySuggestionsView = WireShape<AppModels.MemorySuggestionsView>;

// ── 记忆中枢（Memory Hub）类型 ──────────────────────────────────────

// ProfileFactView 主脑全局画像事实（跨板块共享的用户画像）。
export type ProfileFactView = WireShape<AppModels.ProfileFactView>;
// MemoryHubOverview 记忆中枢聚合总览。
export type MemoryHubOverview = WireShape<AppModels.MemoryHubOverview>;

// ── 记忆图谱 ────────────────────────────────────────────────────────
export type GraphNode = WireShape<AppModels.GraphNode>;
export type GraphLink = WireShape<AppModels.GraphLink>;
export type MemoryGraphView = WireShape<AppModels.MemoryGraphView>;

// SemanticGraphView 是记忆语义图谱（事件日志投影，GaeaMemorySemanticGraph）：
// 与 MemoryGraphView 同渲染面（GraphView source="semantic"），节点 type 取
// entity/event/source，另带日志/悬空统计。
export type SemanticGraphView = WireShape<AppModels.SemanticGraphView>;
// 办公记忆疑似重复对（keep 为建议保留项）。
export type MemoryDuplicateView = WireShape<AppModels.MemoryDuplicateView>;

// ── 流程蒸馏（阶段七 7.2-2：journal 历史挖掘 → 技能结晶建议）──────────

// SkillDistillCandidateView 是一条跨会话重复流程候选（GaeaSkillDistillCandidates）：
// pattern 为 PatternLine 步骤行（如 "edit_file .md"，视图层直显）；repeat 为出现
// 该模式的不同会话数；evidence 为人类可读证据行。手写接口（MemoryEvalReport
// 先例：不依赖 wailsjs 生成物，防包环）。
export interface SkillDistillCandidateView {
  id: string;
  pattern: string[];
  repeat: number;
  sessions: string[];
  evidence: string[];
  lastAt: number;
}

// SkillDistillView 是流程蒸馏视图：候选列表（至多 3 条）+ journal 可用性
// （available=false 表示历史执行数据不可读/无沉淀）+ 生成时间。
export interface SkillDistillView {
  candidates: SkillDistillCandidateView[];
  available: boolean;
  generatedAt: string;
}

// SkillStatView 是技能调用计数视图（7.2-2 判据②）：calls 降序；成功率=工具级
// （read_skill 交付正文 / run_skill 管线无错=ok）。
export interface SkillStatView {
  name: string;
  calls: number;
  ok: number;
  lastAt: number;
}
