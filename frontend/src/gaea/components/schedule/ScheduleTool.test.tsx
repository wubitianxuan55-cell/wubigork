// ScheduleTool.test.tsx — 进度计划工具面板（办公 ToolSheet 内容件）用例。
//
// 覆盖（docs/gaea-ai-workbench-ui-2026-10.md §7.4 工具化边界）：
// ①紧凑面契约：工程名 + 视图四档 Segmented + 状态栏（共 N 项/总工期/关键数）；
// ②视图切换换渲染面（gantt↔pdm↔dcma），store 视图停在编制档（usage/monte/evm）
//   时回落横道图而非出编制视图；③「问计划」回对话通道（onAsk 带工程名前缀）；
// ④空态（无任务）出「载入示例工程」非裸视图；⑤「完整工作台」深链接派发 NAVIGATE
//   （板块路由保留不 404）；⑥保存态文案跟随 store sync。
// 桥接 mock 同 ScheduleFileCard.test 口径；store 用例以 setState 预置
// （zustand 运行时合并任意键），动作用替身（SchedulePage.test 同式）。
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ScheduleTool } from "./ScheduleTool";
import { useScheduleStore } from "../../../schedule/store";
import { FRONTEND_EVENTS } from "../../../events";
import type { SchedProject } from "../../../schedule/types";

const mocks = vi.hoisted(() => ({
  // initScheduleSync → loadScheduleFile：exists=false 走内存态上文件（doSave 再
  // 摸 ScheduleSave）；断言不依赖回读内容，种子以 setState 为准。
  scheduleLoad: vi.fn(async () => ({ exists: false, path: "进度计划/当前计划.gsched.json", project: null })),
  scheduleSave: vi.fn(async () => ({ path: "进度计划/当前计划.gsched.json", savedAt: "2026-01-05 08:00" })),
  scheduleProjects: vi.fn(async () => []),
  readFile: vi.fn(async () => ""),
}));

vi.mock("../../lib/bridge", () => ({
  app: {
    ScheduleLoad: () => mocks.scheduleLoad(),
    ScheduleSave: (_json: string) => mocks.scheduleSave(),
    ScheduleProjects: () => mocks.scheduleProjects(),
    ReadFile: (_rel: string) => mocks.readFile(),
  },
}));

function project(): SchedProject {
  return {
    name: "办公楼施工",
    startDate: "2026-01-05",
    tasks: [
      { id: "A", name: "基础", duration: 3, level: 1, progress: 0 },
      { id: "B", name: "主体", duration: 2, level: 1, progress: 0 },
    ],
    links: [{ from: "A", to: "B", type: "FS", lag: 0 }],
  };
}

const seed = (patch: Record<string, unknown>) =>
  useScheduleStore.setState(patch as unknown as Parameters<typeof useScheduleStore.setState>[0]);

beforeEach(() => {
  localStorage.removeItem("gaea.schedule.v1");
  seed({
    project: project(),
    view: "gantt",
    selectedId: null,
    hydrated: true,
    sync: "saved",
    savedAt: "2026-01-05 08:00",
    syncError: null,
    past: [],
    future: [],
  });
});

describe("ScheduleTool 进度计划工具面板", () => {
  it("紧凑面契约：工程名 + 视图四档 + 状态栏读数（默认横道图）", () => {
    render(<ScheduleTool />);
    expect(screen.getByText("办公楼施工")).toBeTruthy();
    expect(screen.getByTestId("sched-gantt")).toBeTruthy();
    // 视图四档（编制档 usage/monte/evm 不在工具面板）
    for (const label of ["横道图", "单代号", "双代号", "质量体检"]) {
      expect(screen.getByText(label)).toBeTruthy();
    }
    expect(screen.queryByText("资源使用")).toBeNull();
    const bar = screen.getByTestId("sched-tool-statusbar");
    expect(bar.textContent).toContain("2");
    expect(bar.textContent).toContain("5"); // 总工期 5 天
    expect(screen.queryByText("读取计划…")).toBeNull(); // hydrated=true 不出读取态
  });

  it("视图切换换渲染面：横道图→单代号→质量体检（容器 testid 随档换）", () => {
    render(<ScheduleTool />);
    fireEvent.click(screen.getByText("单代号"));
    expect(screen.getByTestId("sched-pdm")).toBeTruthy();
    expect(screen.queryByTestId("sched-gantt")).toBeNull();
    fireEvent.click(screen.getByText("质量体检"));
    expect(screen.getByTestId("dcma-score")).toBeTruthy();
    expect(screen.queryByTestId("sched-pdm")).toBeNull();
  });

  it("store 视图停在编制档（usage）时回落横道图：工具面板不出编制视图", () => {
    seed({ view: "usage" });
    render(<ScheduleTool />);
    expect(screen.getByTestId("sched-gantt")).toBeTruthy();
    // Segmented 以回落档选中展示（不出无选中的悬空态）
    expect(screen.getByText("横道图")).toBeTruthy();
  });

  it("「问计划」回对话通道：onAsk 带工程名前缀，发送后清空输入", () => {
    const onAsk = vi.fn();
    render(<ScheduleTool onAsk={onAsk} />);
    fireEvent.change(screen.getByTestId("sched-tool-ask-input"), { target: { value: "关键线路压得下来吗" } });
    fireEvent.click(screen.getByText("问计划"));
    expect(onAsk).toHaveBeenCalledTimes(1);
    expect(onAsk.mock.calls[0][0]).toContain("【进度计划：办公楼施工】");
    expect(onAsk.mock.calls[0][0]).toContain("关键线路压得下来吗");
    expect((screen.getByTestId("sched-tool-ask-input") as HTMLInputElement).value).toBe("");
  });

  it("空态：无任务出「载入示例工程」卡，点按走 store.loadSample", () => {
    const loadSample = vi.fn();
    seed({ project: { ...project(), tasks: [], links: [] }, loadSample: loadSample as unknown as () => void });
    render(<ScheduleTool />);
    expect(screen.getByTestId("sched-tool-empty")).toBeTruthy();
    expect(screen.queryByTestId("sched-gantt")).toBeNull();
    fireEvent.click(screen.getByTestId("sched-tool-empty-sample"));
    expect(loadSample).toHaveBeenCalledTimes(1);
  });

  it("「完整工作台」深链接：派发 NAVIGATE 到 schedule 板块（旧路由保留）", () => {
    const seen: unknown[] = [];
    const listener = (e: Event) => seen.push((e as CustomEvent).detail);
    window.addEventListener(FRONTEND_EVENTS.NAVIGATE, listener);
    try {
      render(<ScheduleTool />);
      fireEvent.click(screen.getByTestId("sched-tool-full"));
      expect(seen).toContainEqual({ page: "schedule" });
    } finally {
      window.removeEventListener(FRONTEND_EVENTS.NAVIGATE, listener);
    }
  });

  it("保存态文案跟随 store：dirty=改动待保存，error=失败原因", async () => {
    seed({ sync: "dirty" });
    const { unmount } = render(<ScheduleTool />);
    expect(screen.getByText("改动待保存…")).toBeTruthy();
    unmount();

    seed({ sync: "error", syncError: "盘符不可写" });
    render(<ScheduleTool />);
    await waitFor(() => expect(screen.getByText("盘符不可写")).toBeTruthy());
  });
});
