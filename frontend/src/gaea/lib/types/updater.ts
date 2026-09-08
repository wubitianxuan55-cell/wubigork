// Auto-updater payloads (desktop/updater.go). UpdateInfo drives the update banner;
// UpdateProgress streams on the "updater:progress" event during ApplyUpdate.
export interface UpdateInfo {
  available: boolean;
  current: string;
  latest: string;
  version?: string; // S2.3 types 兼容：后端 stub GaeaCheckUpdate 的 {version, notes} 契约
  notes: string;
  canSelfUpdate: boolean; // win/linux true; macOS false (no cert → manual download)
  downloadUrl: string; // human-facing releases page (macOS path / fallback link)
  assetSize: number; // running platform's artifact size, for the progress bar
  err?: string; // set when the check itself failed (both endpoints down)
}

export interface UpdateProgress {
  phase: "downloading" | "verifying" | "applying" | "done" | "error";
  received: number;
  total: number;
  err?: string;
}

// TCCA 缓存报告（V3.0 — 匹配 internal/context/metrics.go CacheReport）
export interface TCCAReport {
  l1Size: number;
  l2Size: number;
  l3Version: number;
  l4Messages: number;
  savedByCompact: number;
  savedByFork: number;
  forkCount: number;
  savedUsd: number;
  savedLatencyMs: number;
  compactionCount: number;
  // V5.30: 全会话缓存命中统计 (来自 agent + context metrics)
  cacheHitTokens: number;
  cacheMissTokens: number;
  breakCount: number;
}
/** TraceStep 记录一次推理过程中的关键步骤，用于实时推理可视化面板 */
export interface TraceStep {
  id: string;
  turnTimestamp: number; // 所属回合开始时间戳
  seq: number;           // 步骤序号（递增）
  type: "phase" | "tool" | "thinking" | "decision" | "compaction";
  timestamp: number;     // 事件发生时间戳（ms）
  label: string;         // 简短描述（如 "规划阶段"、"read_file"、"压缩上下文"）
  detail?: string;       // 详细内容（展开时显示）
  status?: "pending" | "running" | "done" | "error";  // 当前状态
}
