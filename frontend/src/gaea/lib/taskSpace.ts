// taskSpace.ts — S2.1 事件订阅空间过滤（docs/gaea-space-shell-design.md §4.7）
//
// 任务中心/运行角标只收「工位」任务：后端 Task.Space 已带 spaceId
// （json:"spaceId,omitempty"）；缺省（旧任务/旧后端）按 work 兼容放行——
// 与阶段 1 S1.1 旧数据回填 work 同语义。play 任务事件不打扰工位 UI。
import type { TaskView } from "./types";

export function isWorkSpaceTask(t: TaskView): boolean {
  return !t.spaceId || t.spaceId === "work";
}
