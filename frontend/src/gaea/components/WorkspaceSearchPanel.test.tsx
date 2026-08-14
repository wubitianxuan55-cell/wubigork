import { describe, expect, it, beforeEach, vi } from "vitest";
import { render, screen, fireEvent, waitFor, act } from "@testing-library/react";
import { WorkspaceSearchPanel } from "./WorkspaceSearchPanel";
import { ToastProvider } from "./Toast";
import { mockTaskListeners } from "../lib/mock";
import type { TaskView } from "../lib/types";

const wrap = (node: React.ReactNode) => <ToastProvider>{node}</ToastProvider>;

// vi.mock 工厂内通过 hoisted 变量配置 FileIndexRebuild 的覆盖实现
// （模拟「任务异步执行中」的场景，真实 mock 总是立即成功）。
const h = vi.hoisted(() => {
  let rebuildImpl: (() => Promise<TaskView>) | undefined;
  return {
    setRebuildImpl: (fn: (() => Promise<TaskView>) | undefined) => {
      rebuildImpl = fn;
    },
    rebuildImpl: () => rebuildImpl,
  };
});

vi.mock("../lib/bridge", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../lib/bridge")>();
  return {
    ...actual,
    app: new Proxy(actual.app, {
      get(target, prop, receiver) {
        if (prop === "FileIndexRebuild") {
          const override = h.rebuildImpl();
          if (override) return override;
        }
        return Reflect.get(target, prop, receiver);
      },
    }),
  };
});

describe("WorkspaceSearchPanel 工作区搜索", () => {
  beforeEach(() => {
    // 默认走真实 mock（提交即成功）；仅事件驱动用例设置覆盖。
    h.setRebuildImpl(undefined);
  });

  it("语义模式：开启后展示本地语义命中并可重建索引", async () => {
    render(wrap(<WorkspaceSearchPanel onOpenFile={() => {}} />));

    fireEvent.click(screen.getByTitle("语义检索（本地 bge-m3，需先重建索引）"));
    fireEvent.change(screen.getByPlaceholderText("搜索资料正文，如：成本 / 预算 / 方案…"), {
      target: { value: "打桩" },
    });

    expect(await screen.findByText(/语义命中（本地 bge-m3）/)).toBeTruthy();
    expect(screen.getByText("桩基施工方案.md")).toBeTruthy();
    expect(screen.getByText("82%")).toBeTruthy();
  });

  it("重建索引：提交即成功（succeeded TaskView）直接显示结果", async () => {
    render(wrap(<WorkspaceSearchPanel onOpenFile={() => {}} />));

    fireEvent.click(screen.getByTitle("重建工作区文件语义索引"));
    // 真实 mock 的 FileIndexRebuild 立即返回 succeeded 任务，结果在 task.result 里。
    await waitFor(() => expect(screen.getByText(/已索引 3 个文件/)).toBeTruthy());
  });

  it("重建索引：异步任务经 gaea-task 事件推送完成提示", async () => {
    const queued: TaskView = {
      id: "tsk_search_evt",
      kind: "file_index",
      label: "工作区语义索引",
      status: "queued",
      progress: 0,
      message: "排队中",
      error: "",
      retryCount: 0,
      maxRetries: 2,
      payload: "{}",
      result: "",
      createdAt: Date.now(),
      startedAt: 0,
      finishedAt: 0,
    };
    // 模拟后端先入队：提交返回 queued 任务视图。
    h.setRebuildImpl(async () => queued);

    render(wrap(<WorkspaceSearchPanel onOpenFile={() => {}} />));
    fireEvent.click(screen.getByTitle("重建工作区文件语义索引"));

    // 提交后先展示进行中提示
    await waitFor(() => expect(screen.getByText("正在索引工作区…")).toBeTruthy());
    // 等 rebuildIndex 完成 await 并订阅 gaea-task 事件（刷新微任务队列）
    await act(async () => {});

    // 模拟后端经 gaea-task 事件推送该任务终态（succeeded，result 含 total/skipped）
    const done: TaskView = {
      ...queued,
      status: "succeeded",
      progress: 100,
      message: "完成",
      result: JSON.stringify({ total: 5, skipped: 1 }),
      startedAt: Date.now(),
      finishedAt: Date.now(),
    };
    act(() => {
      mockTaskListeners.forEach((l) => l(done));
    });

    await waitFor(() => expect(screen.getByText(/已索引 5 个文件（跳过 1）/)).toBeTruthy());
  });
});
