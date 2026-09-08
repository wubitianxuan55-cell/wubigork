import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import type { ReactElement } from "react";
import { ScheduleFileCard } from "./ScheduleFileCard";
import { ToastProvider } from "./Toast";
import { LocaleProvider } from "../lib/i18n";
import { useStore } from "../lib/store";
import { FRONTEND_EVENTS } from "../../events";
import { parseSchedSummary } from "../../schedule/gschedSummary";
import type { SchedProject } from "../../schedule/types";

const mocks = vi.hoisted(() => ({
  submit: vi.fn(async (_text: string) => {}),
  steer: vi.fn(async (_text: string) => {}),
  scheduleSave: vi.fn(async (_json: string) => ({ path: "进度计划/当前计划.gsched.json", savedAt: "2026-01-05 08:00", duration: 5, critical: 2 })),
  // v4.139 #15 多工程：切指针通道（原「复制为当前计划」的 ScheduleSave 语义废弃）
  projectOpen: vi.fn(async (rel: string) => ({ current: rel })),
}));

vi.mock("../lib/bridge", () => ({
  app: {
    Submit: (text: string) => mocks.submit(text),
    Steer: (text: string) => mocks.steer(text),
    ScheduleSave: (json: string) => mocks.scheduleSave(json),
    ScheduleProjectOpen: (rel: string) => mocks.projectOpen(rel),
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

const wrap = (ui: ReactElement) => (
  <LocaleProvider>
    <ToastProvider>{ui}</ToastProvider>
  </LocaleProvider>
);

const CURRENT = "进度计划/当前计划.gsched.json";
const OTHER = "归档/旧计划.gsched.json";

beforeEach(() => {
  localStorage.setItem("gaea-lang", "zh"); // 钉住 zh，中文文案断言稳定
  mocks.submit.mockClear();
  mocks.steer.mockClear();
  mocks.scheduleSave.mockClear();
  mocks.projectOpen.mockClear();
  useStore.setState({ items: [], running: false, approval: undefined, pendingUser: undefined, seq: 0 });
});

afterEach(() => {
  useStore.setState({ items: [], running: false, approval: undefined, pendingUser: undefined, seq: 0 });
});

describe("ScheduleFileCard 办公侧计划摘要卡", () => {
  it("渲染摘要 chips（总工期/工作/关键/搭接）与开竣工区间", () => {
    const s = parseSchedSummary(JSON.stringify(project()))!;
    render(wrap(<ScheduleFileCard relPath={CURRENT} summary={s} raw="{}" />));
    expect(screen.getByText("办公楼施工")).toBeTruthy();
    expect(screen.getByText("5 天")).toBeTruthy();
    expect(screen.getAllByText("2").length).toBe(2); // 工作 2 项 · 关键 2 项
    expect(screen.getByText(/开工 2026-01-05/)).toBeTruthy();
    expect(screen.getByText(/竣工 2026-01-12/)).toBeTruthy();
  });

  // knife-2 G-2（v4.169.0）：多基线槽位计数进摘要卡（v4.137 #11 多基线 max3）
  it("保存了基线槽位：渲染「基线 N」chip（文本=标签+计数）", () => {
    const p = project();
    p.baselines = [
      { name: "基线一", savedAt: "2026-01-05 08:00", duration: 4, rows: { A: { name: "A", es: 0, ef: 3, dur: 3, critical: true } } },
      { name: "基线二", savedAt: "2026-01-06 08:00", duration: 4, rows: { A: { name: "A", es: 0, ef: 3, dur: 3, critical: true } } },
    ];
    const s = parseSchedSummary(JSON.stringify(p))!;
    render(wrap(<ScheduleFileCard relPath={CURRENT} summary={s} raw="{}" />));
    expect(s.baselineCount).toBe(2);
    const label = screen.getByText("基线");
    expect(label).toBeTruthy();
    expect(label.parentElement?.textContent).toBe("基线2");
  });

  it("未保存基线槽位：不渲染「基线」chip（baselineCount=0）", () => {
    const s = parseSchedSummary(JSON.stringify(project()))!;
    expect(s.baselineCount).toBe(0);
    render(wrap(<ScheduleFileCard relPath={CURRENT} summary={s} raw="{}" />));
    expect(screen.queryByText("基线")).toBeNull();
  });

  it("当前计划文件：跳转按钮派发 navigate 事件到 schedule 板块", () => {
    const s = parseSchedSummary(JSON.stringify(project()))!;
    const seen: unknown[] = [];
    const listener = (e: Event) => seen.push((e as CustomEvent).detail);
    window.addEventListener(FRONTEND_EVENTS.NAVIGATE, listener);
    try {
      render(wrap(<ScheduleFileCard relPath={CURRENT} summary={s} raw="{}" />));
      fireEvent.click(screen.getByText("在进度计划中打开"));
      expect(seen).toContainEqual({ page: "schedule" });
    } finally {
      window.removeEventListener(FRONTEND_EVENTS.NAVIGATE, listener);
    }
  });

  it("空闲点「AI 进度周报」：落用户气泡并 app.Submit 周报指令", async () => {
    const s = parseSchedSummary(JSON.stringify(project()))!;
    render(wrap(<ScheduleFileCard relPath={CURRENT} summary={s} raw="{}" />));
    fireEvent.click(screen.getByText("AI 进度周报"));
    await waitFor(() => expect(mocks.submit).toHaveBeenCalledTimes(1));
    const items = useStore.getState().items;
    expect(items.length).toBe(1);
    expect(items[0]).toMatchObject({ kind: "user" });
    expect(String((items[0] as { text?: string }).text)).toContain("schedule_get");
    expect(mocks.steer).not.toHaveBeenCalled();
  });

  it("运行中点「AI 进度周报」：走 Steer 插话，不落气泡不 Submit", async () => {
    useStore.setState({ running: true });
    const s = parseSchedSummary(JSON.stringify(project()))!;
    render(wrap(<ScheduleFileCard relPath={CURRENT} summary={s} raw="{}" />));
    fireEvent.click(screen.getByText("AI 进度周报"));
    await waitFor(() => expect(mocks.steer).toHaveBeenCalledTimes(1));
    expect(mocks.submit).not.toHaveBeenCalled();
    expect(useStore.getState().items.length).toBe(0);
  });

  it("非当前计划文件：「打开」仍可用（v4.139 任意计划可开），周报不显示", () => {
    const s = parseSchedSummary(JSON.stringify(project()))!;
    render(wrap(<ScheduleFileCard relPath={OTHER} summary={s} raw="{}" />));
    expect(screen.getByText("在进度计划中打开")).toBeTruthy();
    expect(screen.queryByText("AI 进度周报")).toBeNull();
    expect(screen.getByText(/非当前计划文件/)).toBeTruthy();
  });

  it("循环依赖计划：展示告警而非工期 chips", () => {
    const p = project();
    p.links.push({ from: "B", to: "A", type: "FS", lag: 0 });
    const s = parseSchedSummary(JSON.stringify(p))!;
    render(wrap(<ScheduleFileCard relPath={CURRENT} summary={s} raw="{}" />));
    expect(screen.getByText(/循环依赖/)).toBeTruthy();
    expect(screen.queryByText("5 天")).toBeNull();
  });

  // v4.139 #15 语义升级（原用例「非当前计划可一键设为当前计划：写 ScheduleSave」）：
  // 「设为当前计划」从复制内容改为切指针——点「打开」调 ScheduleProjectOpen +
  // 跳板块；不再复制覆盖（ScheduleSave 联动路径废弃，多工程下复制会制造同源漂移）。
  it("非当前计划点「打开」=切指针：ScheduleProjectOpen+navigate，不再走 ScheduleSave 复制", async () => {
    const s = parseSchedSummary(JSON.stringify(project()))!;
    const seen: unknown[] = [];
    const listener = (e: Event) => seen.push((e as CustomEvent).detail);
    window.addEventListener(FRONTEND_EVENTS.NAVIGATE, listener);
    try {
      render(wrap(<ScheduleFileCard relPath={OTHER} summary={s} raw="{}" />));
      fireEvent.click(screen.getByText("在进度计划中打开"));
      await waitFor(() => expect(mocks.projectOpen).toHaveBeenCalledTimes(1));
      expect(mocks.projectOpen).toHaveBeenCalledWith(OTHER);
      expect(seen).toContainEqual({ page: "schedule" });
      await waitFor(() =>
        expect(screen.getAllByText("已切换为当前计划").length).toBeGreaterThanOrEqual(1),
      );
      expect(mocks.scheduleSave).not.toHaveBeenCalled();
    } finally {
      window.removeEventListener(FRONTEND_EVENTS.NAVIGATE, listener);
    }
  });

  it("当前计划点「打开」：直接跳板块，不重复切指针", () => {
    const s = parseSchedSummary(JSON.stringify(project()))!;
    const seen: unknown[] = [];
    const listener = (e: Event) => seen.push((e as CustomEvent).detail);
    window.addEventListener(FRONTEND_EVENTS.NAVIGATE, listener);
    try {
      render(wrap(<ScheduleFileCard relPath={CURRENT} summary={s} raw="{}" />));
      fireEvent.click(screen.getByText("在进度计划中打开"));
      expect(seen).toContainEqual({ page: "schedule" });
      expect(mocks.projectOpen).not.toHaveBeenCalled();
    } finally {
      window.removeEventListener(FRONTEND_EVENTS.NAVIGATE, listener);
    }
  });

  // v4.139 #15：循环依赖计划也开放「打开」——切指针不落盘（原复制语义因 Go Save
  // fail-closed 拒收而隐藏入口），修正搭接在板块内完成后回看即可。
  it("循环依赖计划也可「打开」：切指针不依赖 CPM 通过", async () => {
    const bad = project();
    bad.links.push({ from: "B", to: "A", type: "FS", lag: 0 });
    const badSummary = parseSchedSummary(JSON.stringify(bad))!;
    render(wrap(<ScheduleFileCard relPath={OTHER} summary={badSummary} raw="{}" />));
    fireEvent.click(screen.getByText("在进度计划中打开"));
    await waitFor(() => expect(mocks.projectOpen).toHaveBeenCalledTimes(1));
  });

  it("原始 JSON 开关：展开显示原文，收起消失", () => {
    const s = parseSchedSummary(JSON.stringify(project()))!;
    render(wrap(<ScheduleFileCard relPath={CURRENT} summary={s} raw='{"name":"办公楼施工"}' />));
    expect(screen.queryByText('{"name":"办公楼施工"}')).toBeNull();
    fireEvent.click(screen.getByText("原始 JSON"));
    expect(screen.getByText('{"name":"办公楼施工"}')).toBeTruthy();
    fireEvent.click(screen.getByText("原始 JSON"));
    expect(screen.queryByText('{"name":"办公楼施工"}')).toBeNull();
  });
});
