// processStatus.ts — 过程卡状态四态推导（纯函数，供 Transcript/测试共用）。
// 状态不只靠颜色：色 + 图标 + 文字三重传达（design-system/gaea/pages/gaea.md 验收）。
// label 是 i18n 键（非文案）：翻译在渲染点（ProcessCard）做，保持本模块纯数据。
import type { DictKey } from "../locales/en";
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
  { cls: string; icon: "alert" | "ban" | "check" | null; labelKey: DictKey }
> = {
  running: { cls: "text-accent bg-accent/10 border-accent/25", icon: null, labelKey: "process.statusRunning" },
  error: { cls: "text-err bg-err/10 border-err/25", icon: "alert", labelKey: "process.statusError" },
  stopped: { cls: "text-warning bg-warning/10 border-warning/25", icon: "ban", labelKey: "process.statusStopped" },
  done: { cls: "text-ok bg-ok/10 border-ok/25", icon: "check", labelKey: "process.statusDone" },
};
