import type { WireShape } from "./shared";
import type { app as AppModels } from "../../../../wailsjs/go/models";

// ── 阶段 5 T5-1：通用任务调度器视图 ──
// 长任务（价格抓取/文件索引重建等）统一走持久化任务队列；gaea-task 事件
// 实时推送任务视图（状态/进度/消息），任务中心据此渲染并支持取消/重试。
// stopping（C1）：用户已请求取消，等待 handler 退出（结束态细分）。
export type TaskStatus = "queued" | "running" | "stopping" | "succeeded" | "failed" | "cancelled";

export interface TaskView {
  id: string;
  kind: string; // price_fetch / price_fetch_all / file_index / …
  label: string;
  status: TaskStatus;
  progress: number; // 0-100
  message: string;
  error: string;
  retryCount: number;
  maxRetries: number;
  payload: string; // 不透明 JSON
  result: string; // 不透明 JSON（完成时按 kind 解析）
  createdAt: number; // unix 毫秒
  startedAt: number;
  finishedAt: number;
  // S1 空间归属（后端 Task.Space `json:"spaceId,omitempty"`）：任务中心/角标
  // 按当前空间过滤事件（S2.1 docs/gaea-space-shell-design.md §4.7）。
  spaceId?: string;
  // 会话归属（v4.180 结构刀，后端 Task.SessionID `json:"session_id,omitempty"`，
  // SchemaV20 session_id 列）：提交任务的会话标识，空串/缺省=非会话入口
  // （cron/系统周期任务诚实留空）；任务中心「本会话」chip 按此过滤。
  session_id?: string;
  // C9 事件视图字段：gaea-task 事件在输出变更/终态时携带输出尾部整尾回放
  // （有界环形缓冲），输出 dock 事件即推（轮询兜底）；列表/查询响应中缺省。
  outputTail?: string;
  outputTruncated?: boolean;
  // 进程类任务的真实退出码（后端 Task.ExitCode `json:"exitCode,omitempty"`，
  // 事件视图字段不落库，Get/List 合入）：handler 经 Progress.ExitCode 上报；
  // 纯函数任务无退出码语义，诚实缺省。undefined ≠ 0：0 是真实的成功退出码。
  exitCode?: number;
}
// TaskOutputView 是任务实时输出的尾部回放视图（C1：GaeaTaskOutput）。
export type TaskOutputView = WireShape<AppModels.TaskOutputView>;

// ── 阶段 5 T5-3：本地模型调度纵深 ────────────────────────────
// ModelSwitchEstimate 是换模预估结果（GaeaModelSwitchEstimate）：切换本地模型前
// 提示等待时长，让用户决定是否继续（hot=已运行可直接切换；cold=已安装需冷启动；
// download=未安装需先下载；unknown=无法评估）。
export interface ModelSwitchEstimate {
  engine: string; // 引擎 ID（如 herdsman）
  model: string; // 目标模型名（预估基于引擎聚合时可为空）
  status: "hot" | "cold" | "download" | "unknown";
  waitSeconds: number; // 预计等待秒数（hot 为 0）
  note: string; // 人类可读说明（如「已运行，约 0 秒」「需冷启动约 12 秒」）
}
