import { describe, expect, it, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor, act } from "@testing-library/react";
import { TaskCenter } from "./TaskCenter";
import { ToastProvider } from "./Toast";
import { LocaleProvider } from "../lib/i18n";
import type { TaskOutputView, TaskView } from "../lib/types";

const tasks = vi.hoisted(() => ({
  list: [] as TaskView[],
  output: {} as Record<string, TaskOutputView>,
}));

// v4.78：Cancel/Kill 间谍（两击确认断言用）
const spies = vi.hoisted(() => ({
  cancel: vi.fn<(id: string) => Promise<void>>().mockResolvedValue(undefined),
  kill: vi.fn<(id: string) => Promise<void>>().mockResolvedValue(undefined),
}));

vi.mock("../lib/bridge", () => ({
  workApp: {
    TaskList: async (): Promise<TaskView[]> => [...tasks.list],
    TaskCancel: (id: string) => spies.cancel(id),
    TaskKill: (id: string) => spies.kill(id),
    TaskRetry: async () => {},
    TaskOutput: async (id: string): Promise<TaskOutputView> => tasks.output[id] ?? { tail: "", truncated: false },
  },
  // v4.5.1a：镜像真实 onTaskEvent 的订阅层空间过滤（work 订阅丢弃 play 事件）
  onTaskEvent: (cb: (t: TaskView) => void, space?: string) => {
    taskEventCb = (t: TaskView) => {
      if (space && t.spaceId && t.spaceId !== space) return;
      cb(t);
    };
    return () => {
      taskEventCb = null;
    };
  },
}));

// C9：事件回调句柄（onTaskEvent 注册时捕获），测试用它模拟 gaea-task 推送。
let taskEventCb: ((t: TaskView) => void) | null = null;

// TaskCenter 走 useT：钉住 zh 让「进行中/输出 · …」等中文断言继续成立
const wrap = (node: React.ReactNode) => {
  localStorage.setItem("gaea-lang", "zh");
  return (
    <LocaleProvider>
      <ToastProvider>{node}</ToastProvider>
    </LocaleProvider>
  );
};

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
  spies.cancel.mockClear();
  spies.kill.mockClear();
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

  it("v4.77：运行中任务提供「强制终止」，排队任务保留「取消」", async () => {
    tasks.list = [
      makeTask({ id: "f1", label: "抓取进程", status: "running" }),
      makeTask({ id: "q1", label: "排队任务", status: "queued", progress: 0 }),
    ];
    render(wrap(<TaskCenter />));
    expect(await screen.findByText("强制终止")).toBeTruthy();
    expect(screen.getByText("取消")).toBeTruthy();
  });

  it("v4.78 两击确认：首击仅进入确认态不调用 TaskKill，再击才强制终止；queued 单击取消直通", async () => {
    tasks.list = [
      makeTask({ id: "k1", label: "运行任务", status: "running" }),
      makeTask({ id: "q1", label: "排队任务", status: "queued", progress: 0 }),
    ];
    render(wrap(<TaskCenter />));
    const killBtn = await screen.findByTestId("task-kill-btn");
    fireEvent.click(killBtn);
    expect(screen.getByText("再击确认终止")).toBeTruthy();
    expect(spies.kill).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("task-kill-btn"));
    expect(spies.kill).toHaveBeenCalledTimes(1);
    expect(spies.kill).toHaveBeenCalledWith("k1");
    // queued：单击取消（不走两击确认）
    fireEvent.click(screen.getByTestId("task-cancel-btn"));
    expect(spies.cancel).toHaveBeenCalledTimes(1);
    expect(spies.cancel).toHaveBeenCalledWith("q1");
    expect(spies.kill).toHaveBeenCalledTimes(1);
  });

  it("v4.78 两击确认：3s 无再击自动回退；状态离开 running 复位确认态", async () => {
    tasks.list = [makeTask({ id: "k1", label: "运行任务", status: "running" })];
    render(wrap(<TaskCenter />));
    const btn = await screen.findByTestId("task-kill-btn");
    vi.useFakeTimers();
    try {
      fireEvent.click(btn);
      expect(screen.getByText("再击确认终止")).toBeTruthy();
      act(() => {
        vi.advanceTimersByTime(3100);
      });
      expect(screen.queryByText("再击确认终止")).toBeNull();
      expect(screen.getByText("强制终止")).toBeTruthy();
      expect(spies.kill).not.toHaveBeenCalled();
    } finally {
      vi.useRealTimers();
    }
    // 状态离开 running（用户经他处取消转 stopping）→ 确认态复位、按钮回取消语义
    taskEventCb?.(makeTask({ id: "k1", status: "stopping", progress: 60 }));
    const stopBtn = await screen.findByTestId("task-cancel-btn");
    fireEvent.click(stopBtn);
    expect(spies.cancel).toHaveBeenCalledWith("k1");
    expect(spies.kill).not.toHaveBeenCalled();
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

  it("S2.1 事件过滤：play 任务事件不进入工位任务中心，列表初始拉取同样过滤", async () => {
    tasks.list = [
      makeTask({ id: "w1", label: "工位索引", status: "running", spaceId: "work" }),
      makeTask({ id: "p1", label: "乐园任务", status: "running", spaceId: "play" }),
    ];
    render(wrap(<TaskCenter />));
    expect(await screen.findByText("工位索引")).toBeTruthy();
    expect(screen.queryByText("乐园任务")).toBeNull();

    // 事件流：play 事件丢弃、work 事件进入
    taskEventCb?.(makeTask({ id: "p2", label: "乐园事件", status: "running", spaceId: "play" }));
    taskEventCb?.(makeTask({ id: "w2", label: "工位事件", status: "queued", spaceId: "work" }));
    expect(await screen.findByText("工位事件")).toBeTruthy();
    expect(screen.queryByText("乐园事件")).toBeNull();
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

  it("退出码透出：失败随错误行常显，其他终态次行展示；0 不被吞，纯函数任务诚实留空", async () => {
    tasks.list = [
      makeTask({ id: "f1", label: "失败进程", status: "failed", error: "进程退出非零", exitCode: 2 }),
      makeTask({ id: "s1", label: "成功进程", status: "succeeded", progress: 100, exitCode: 0 }),
      makeTask({ id: "c1", label: "取消进程", status: "cancelled", exitCode: -1 }),
      makeTask({ id: "p1", label: "纯函数任务", status: "failed", error: "解析失败" }),
    ];
    render(wrap(<TaskCenter />));
    expect(await screen.findByText("失败进程")).toBeTruthy();

    // 失败 + 退出码：错误行常显（最直观）
    expect(screen.getByText(/· exit 2/)).toBeTruthy();
    // succeeded + exit 0：真实的成功退出码不被 omitempty 语义吞掉
    expect(screen.getByText("exit 0")).toBeTruthy();
    // cancelled 带被杀进程的退出码：次行如实透出
    expect(screen.getByText("exit -1")).toBeTruthy();
    // 全部退出码渲染恰好 3 处：纯函数任务（无退出码语义）诚实留空
    expect(screen.getAllByText(/exit -?\d+/)).toHaveLength(3);
    // 纯函数失败任务只显示错误文本，无退出码
    expect(screen.getByText("解析失败")).toBeTruthy();
  });
});
