import { describe, expect, it, beforeEach, vi } from "vitest";
import { render, screen, fireEvent, waitFor, act } from "@testing-library/react";
import { WorkspaceSearchPanel } from "./WorkspaceSearchPanel";
import { ToastProvider } from "./Toast";
import { mockTaskListeners } from "../lib/mock";
import { noteSpaceActivated } from "../lib/useSpaceScope";
import type { SpaceActiveView, TaskView } from "../lib/types";

const wrap = (node: React.ReactNode) => <ToastProvider>{node}</ToastProvider>;

// vi.mock 工厂内通过 hoisted 变量配置 FileIndexRebuild 的覆盖实现
// （模拟「任务异步执行中」的场景，真实 mock 总是立即成功）；
// S1.2-C 另记录 UnifiedSearch 调用参数（断言 scope 传递）并支持覆盖
// GaeaSpaceActive（真实 mock 带 80ms delay，测试用即时返回保持确定性）。
const h = vi.hoisted(() => {
  let rebuildImpl: (() => Promise<TaskView>) | undefined;
  let spaceImpl: (() => Promise<SpaceActiveView>) | undefined;
  let unifiedCalls: unknown[][] = [];
  return {
    setRebuildImpl: (fn: (() => Promise<TaskView>) | undefined) => {
      rebuildImpl = fn;
    },
    rebuildImpl: () => rebuildImpl,
    setSpaceImpl: (fn: (() => Promise<SpaceActiveView>) | undefined) => {
      spaceImpl = fn;
    },
    spaceImpl: () => spaceImpl,
    recordUnified: (args: unknown[]) => {
      unifiedCalls.push(args);
    },
    unifiedCalls: () => unifiedCalls,
    resetUnifiedCalls: () => {
      unifiedCalls = [];
    },
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
        if (prop === "GaeaSpaceActive") {
          const override = h.spaceImpl();
          if (override) return override;
        }
        if (prop === "UnifiedSearch") {
          const orig = Reflect.get(target, prop, receiver) as (...a: unknown[]) => unknown;
          return (...args: unknown[]) => {
            h.recordUnified(args);
            return orig(...args);
          };
        }
        return Reflect.get(target, prop, receiver);
      },
    }),
  };
});

const WORK_VIEW: SpaceActiveView = {
  space: "work",
  modeOn: true,
  exportsDir: ".gaea/exports",
  workDir: ".gaea/work",
};

describe("WorkspaceSearchPanel 工作区搜索", () => {
  beforeEach(() => {
    // 默认走真实 mock（提交即成功）；仅事件驱动用例设置覆盖。
    h.setRebuildImpl(undefined);
    // GaeaSpaceActive 即时返回 work（双空间默认空间），避免真实 mock 的 80ms delay。
    h.setSpaceImpl(async () => WORK_VIEW);
    h.resetUnifiedCalls();
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

  it("跨库模式：UnifiedSearch 收到面板 scope（默认=当前生效空间 work）", async () => {
    render(wrap(<WorkspaceSearchPanel onOpenFile={() => {}} />));
    fireEvent.click(screen.getByTitle(/跨库统一检索/));
    fireEvent.change(screen.getByPlaceholderText("搜索资料正文，如：成本 / 预算 / 方案…"), {
      target: { value: "振动锤" },
    });

    // 300ms 防抖后发起跨库检索：签名 (query, scope, topN)，默认 scope=work。
    await waitFor(() => expect(h.unifiedCalls().length).toBeGreaterThan(0));
    const [q, scope, topN] = h.unifiedCalls()[0];
    expect(q).toBe("振动锤");
    expect(scope).toBe("work");
    expect(topN).toBe(20);
  });

  it("跨库模式：切「全部」后 UnifiedSearch 收到 scope=\"\"（旧行为，显式选择才跨空间）", async () => {
    // useSpaceScope 模块缓存在同文件用例间共享：先归一回 work，保证确定性。
    act(() => {
      noteSpaceActivated(WORK_VIEW);
    });
    render(wrap(<WorkspaceSearchPanel onOpenFile={() => {}} />));
    fireEvent.click(screen.getByTitle(/跨库统一检索/));
    fireEvent.change(screen.getByPlaceholderText("搜索资料正文，如：成本 / 预算 / 方案…"), {
      target: { value: "振动锤" },
    });
    await waitFor(() => expect(h.unifiedCalls().length).toBeGreaterThan(0));
    expect(h.unifiedCalls()[0][1]).toBe("work");

    // 切「全部」（scope=""）→ run 重建 → 防抖后自动重搜一次。
    fireEvent.click(screen.getByTitle(/跨书房与庭院检索/));
    await waitFor(() => expect(h.unifiedCalls().length).toBeGreaterThanOrEqual(2));
    const last = h.unifiedCalls()[h.unifiedCalls().length - 1];
    expect(last[0]).toBe("振动锤");
    expect(last[1]).toBe("");
    expect(last[2]).toBe(20);
  });

  it("跨库模式：SpaceChip 激活 play 后（noteSpaceActivated 广播）默认 scope=play", async () => {
    // 放在最后一个用例：noteSpaceActivated 会改写模块缓存（cached=play）。
    render(wrap(<WorkspaceSearchPanel onOpenFile={() => {}} />));
    // 模拟 SpaceChip 激活庭院空间：广播给已挂载的检索面（scope 未被用户改过）。
    act(() => {
      noteSpaceActivated({
        space: "play",
        modeOn: true,
        exportsDir: ".gaea/play/exports",
        workDir: ".gaea/play/work",
      });
    });
    fireEvent.click(screen.getByTitle(/跨库统一检索/));
    fireEvent.change(screen.getByPlaceholderText("搜索资料正文，如：成本 / 预算 / 方案…"), {
      target: { value: "游戏" },
    });

    await waitFor(() => expect(h.unifiedCalls().length).toBeGreaterThan(0));
    const [, scope] = h.unifiedCalls()[0];
    expect(scope).toBe("play");
  });
});
