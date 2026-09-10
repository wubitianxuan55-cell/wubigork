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

// MemoryLifecycleItem / MemoryLifecycleView 是三态生命周期视图
// （GaeaMemoryLifecycle：固化/衰减/归档，5.3）。
export type MemoryLifecycleItem = WireShape<AppModels.MemoryLifecycleItem>;

export type MemoryLifecycleView = WireShape<AppModels.MemoryLifecycleView>;

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

/** MemoryMergeSuggestion 蒸馏合并候选（做梦 2.0：确定性重复记忆，批准后归档较旧条）。 */
export type MemoryMergeSuggestion = NonNullable<MemorySuggestionsView["merges"]>[number];
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
