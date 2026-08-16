// processStatus.ts — 过程卡状态四态推导（纯函数，供 Transcript/测试共用）。
// 状态不只靠颜色：色 + 图标 + 文字三重传达（design-system/gaea/pages/gaea.md 验收）。
import type { Item } from "./store";

export type ProcessStatus = "running" | "done" | "error" | "stopped" | "idle";

export function deriveProcessStatus(items: Item[], running: boolean): ProcessStatus {
  if (running) return "running";
  let hasTool = false;
  let hasError = false;
  let hasStopped = false;
  for (const it of items) {
    if (it.kind !== "tool" || it.parentId) continue;
    hasTool = true;
    if (it.status === "error") hasError = true;
    if (it.status === "stopped") hasStopped = true;
  }
  if (hasError) return "error";
  if (hasStopped) return "stopped";
  if (hasTool) return "done";
  return "idle";
}

export const PROCESS_STATUS_META: Record<
  Exclude<ProcessStatus, "idle">,
  { cls: string; icon: "alert" | "ban" | "check" | null; label: string }
> = {
  running: { cls: "text-accent bg-accent/10 border-accent/25", icon: null, label: "处理中" },
  error: { cls: "text-err bg-err/10 border-err/25", icon: "alert", label: "有错误" },
  stopped: { cls: "text-warning bg-warning/10 border-warning/25", icon: "ban", label: "已中断" },
  done: { cls: "text-ok bg-ok/10 border-ok/25", icon: "check", label: "完成" },
};
