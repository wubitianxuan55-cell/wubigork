// taskSpace.test.ts — S2.1 任务事件空间过滤（docs/gaea-space-shell-design.md §4.7）
import { describe, expect, it } from "vitest";
import { isWorkSpaceTask } from "./taskSpace";
import type { TaskView } from "./types";

function task(over: Partial<TaskView>): TaskView {
  return {
    id: "t1", kind: "file_index", label: "索引", status: "running",
    progress: 0, message: "", error: "", retryCount: 0, maxRetries: 0,
    payload: "{}", result: "", createdAt: 0, startedAt: 0, finishedAt: 0,
    ...over,
  };
}

describe("isWorkSpaceTask", () => {
  it("缺省（旧任务/旧后端）按 work 兼容放行", () => {
    expect(isWorkSpaceTask(task({}))).toBe(true);
  });

  it("work 任务放行，play 任务丢弃", () => {
    expect(isWorkSpaceTask(task({ spaceId: "work" }))).toBe(true);
    expect(isWorkSpaceTask(task({ spaceId: "play" }))).toBe(false);
  });
});
