import { describe, expect, it, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { TaskCenter } from "./TaskCenter";
import { ToastProvider } from "./Toast";
import type { TaskOutputView, TaskView } from "../lib/types";

const tasks = vi.hoisted(() => ({
  list: [] as TaskView[],
  output: {} as Record<string, TaskOutputView>,
}));

vi.mock("../lib/bridge", () => ({
  app: {
    TaskList: async (): Promise<TaskView[]> => [...tasks.list],
    TaskCancel: async () => {},
    TaskRetry: async () => {},
    TaskOutput: async (id: string): Promise<TaskOutputView> => tasks.output[id] ?? { tail: "", truncated: false },
  },
  onTaskEvent: (cb: (t: TaskView) => void) => {
    taskEventCb = cb;
    return () => {
      taskEventCb = null;
    };
  },
}));

// C9：事件回调句柄（onTaskEvent 注册时捕获），测试用它模拟 gaea-task 推送。
let taskEventCb: ((t: TaskView) => void) | null = null;

const wrap = (node: React.ReactNode) => <ToastProvider>{node}</ToastProvider>;

function makeTask(over: Partial<TaskView>): TaskView {
  return {
    id: "tsk_1",
    kind: "price_fetch",
    label: "抓取四川造价信息网",
    status: "running",
    progress: 50,
    message: "正在抓取…",
    error: "",
    retryCount: 0,
    maxRetries: 2,
    payload: "{}",
    result: "",
    createdAt: Date.now(),
    startedAt: Date.now(),
    finishedAt: 0,
    ...over,
  };
}

afterEach(() => {
  tasks.list = [];
  tasks.output = {};
});

describe("TaskCenter 任务中心（C1 实时输出 + 结束态细分）", () => {
  it("渲染任务列表与状态徽标（含 stopping 停止中）", async () => {
    tasks.list = [
      makeTask({ id: "t1", label: "抓取A", status: "running" }),
      makeTask({ id: "t2", label: "抓取B", status: "stopping", progress: 80, message: "正在停止…" }),
      makeTask({ id: "t3", label: "索引", status: "succeeded", progress: 100, kind: "file_index" }),
    ];
    render(wrap(<TaskCenter />));
    expect(await screen.findByText("抓取A")).toBeTruthy();
    expect(screen.getByText("进行中")).toBeTruthy();
    expect(screen.getByText("停止中")).toBeTruthy();
    expect(screen.getByText("已完成")).toBeTruthy();
    expect(screen.getByText("2 个进行中")).toBeTruthy(); // running+stopping 计入活跃
  });

  it("点击任务行 → 输出 dock 回放实时输出，可关闭", async () => {
    tasks.list = [makeTask({ id: "t1", label: "抓取四川造价信息网", status: "running" })];
    tasks.output = {
      t1: { tail: "[10:00:00] 开始 抓取四川造价信息网\n[10:00:01] 正在抓取…", truncated: false },
    };
    render(wrap(<TaskCenter />));
    await screen.findByText("抓取四川造价信息网");

    fireEvent.click(screen.getByText("抓取四川造价信息网"));
    expect(await screen.findByText(/\[10:00:01\] 正在抓取/)).toBeTruthy();
    expect(screen.getByText(/输出 · 抓取四川造价信息网/)).toBeTruthy();

    fireEvent.click(screen.getByTitle("关闭输出"));
    await waitFor(() => expect(screen.queryByText(/输出 · 抓取四川造价信息网/)).toBeNull());
  });

  it("C9：事件携带 outputTail 时输出 dock 即推更新（不等轮询）", async () => {
    tasks.list = [makeTask({ id: "t1", label: "抓取A", status: "running" })];
    tasks.output = { t1: { tail: "[10:00:00] 开始", truncated: false } };
    render(wrap(<TaskCenter />));
    await screen.findByText("抓取A");

    fireEvent.click(screen.getByText("抓取A"));
    expect(await screen.findByText(/\[10:00:00\] 开始/)).toBeTruthy();

    // 模拟后端输出变更事件：tail 整尾回放，dock 立即更新
    expect(taskEventCb).toBeTruthy();
    taskEventCb!({
      ...makeTask({ id: "t1", status: "running", progress: 60 }),
      outputTail: "[10:00:00] 开始\n[10:00:01] 已索引 64/128 个文件",
    } as TaskView);
    expect(await screen.findByText(/已索引 64\/128 个文件/)).toBeTruthy();

    // 事件不带 outputTail（纯状态变更）→ 不覆盖 dock 内容
    taskEventCb!({ ...makeTask({ id: "t1", status: "stopping", progress: 60 }) });
    expect(screen.getByText(/已索引 64\/128 个文件/)).toBeTruthy();
  });

  it("截断输出显示标注", async () => {
    tasks.list = [makeTask({ id: "t1", label: "索引", status: "succeeded", progress: 100 })];
    tasks.output = { t1: { tail: "很多行…", truncated: true } };
    render(wrap(<TaskCenter />));
    await screen.findByText("索引");

    fireEvent.click(screen.getByText("索引"));
    expect(await screen.findByText(/输出过长已截断/)).toBeTruthy();
  });
});
