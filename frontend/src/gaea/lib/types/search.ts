import type { WireShape } from "./shared";
import type { app as AppModels } from "../../../../wailsjs/go/models";
import type { FileSemanticHit, WorkspaceSearchHit } from "./files";

// 跨库统一语义检索命中（cost / knowledge / office，本地 bge-m3）。
export interface SemanticHitView {
  kind: "cost" | "knowledge" | "office" | "file";
  name: string;
  score: number;
  text: string;
}

// BrainHit 是三脑统一检索命中（brain.main 主脑 / brain.left 左脑 / brain.right 右脑）。
export interface BrainHit {
  brain: string;
  entity: string;
  text: string;
  score: number;
}
// SearchScope 是统一检索的空间范围（S1.2-C，docs/gaea-memory-isolation-design.md）：
// ""=全部（旧行为，仅用户显式选择「全部」时使用）；"work"/"play"=只搜对应空间。
// 双空间红线：默认只搜当前空间（GaeaSpaceActive 下发的 space）。
export type SearchScope = "" | "work" | "play";

// UnifiedSearchView 是一次「跨库统一检索」调用的完整结果（记忆统一层第一刀：
// hub 搜索由 4 绑定前端拼装收敛为 1 绑定后端聚合）：
// keyword = 工作区全文关键词命中（轻量 RAG），semantic = 跨库语义命中，
// brain = 三脑命中（brain.main/left/right），files = 文件语义命中。
export interface UnifiedSearchView {
  keyword: WorkspaceSearchHit[];
  semantic: SemanticHitView[];
  brain?: BrainHit[];
  files?: FileSemanticHit[];
}

// IntentResultView 是统一意图路由（v4.5 指令中枢）的执行结果（S4.6 命令面板接内核）：
// reply=回执文本（dry-run 时为「将发生什么」预览语）；cardPath=能力产物文件路径
// （如生图落盘，非空时可做文件卡片）；handled=是否命中；action/target=命中动作与
// 目标原文（前端指令卡片渲染用；未命中时后端返回零值）。
export interface IntentResultView {
  reply: string;
  cardPath?: string;
  handled: boolean;
  action?: string;
  target?: string;
}


// RetrievalEvalReport 是检索质量测评结果：内置查询集跑一遍统一检索，
// 统计平均 recall@10，并与达标门槛比较给出通过状态。
export type RetrievalEvalReport = WireShape<AppModels.RetrievalEvalReport>;
