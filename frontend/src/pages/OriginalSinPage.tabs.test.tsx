import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ContextTimeline, Trajectory } from "../gaea/lib/types";

// 页面 import 链最重（antd+办公组件族+react-window），全量并行负载下按
// ContextView 先例放宽本文件时限，只放宽时限不改断言。
vi.setConfig({ testTimeout: 30_000 });

// OriginalSinPage.tabs.test.tsx — v4.412 原罪主区页签接线：故事/轨迹/上下文
// 三页签复用办公 ChatTabs/TrajectoryView/ContextView，数据源走
// SinTrajectory/SinContextView/SinContextNodeDetail（估算口径）。
// 页面壳（故事架/右栏/Composer）与本测试无关：sin hooks 与重组件全部 mock 掉，
// 只验「页签切换 → 自定义数据源被按当前故事 id 调用 → 看板渲染」。

const sinTrajectoryMock = vi.fn();
const sinContextViewMock = vi.fn();
const sinNodeDetailMock = vi.fn();

vi.mock("../gaea/lib/bridge", () => ({
  app: {
    SinTrajectory: (...a: unknown[]) => sinTrajectoryMock(...a),
    SinContextView: (...a: unknown[]) => sinContextViewMock(...a),
    SinContextNodeDetail: (...a: unknown[]) => sinNodeDetailMock(...a),
  },
  openExternal: vi.fn(),
  onEvent: vi.fn(() => () => {}),
}));

vi.mock("./sin/useSinStory", () => ({
  useSinStory: () => ({
    stories: [{ id: "sin_t1", title: "雨夜电车", createdAt: "", updatedAt: "", preview: "" }],
    activeId: "sin_t1",
    activeStory: { id: "sin_t1", title: "雨夜电车", createdAt: "", updatedAt: "", preview: "" },
    messages: [],
    initializing: false,
    sending: false,
    notice: "",
    clearNotice: () => {},
    showNotice: () => {},
    selectStory: async () => {},
    createStory: async () => {},
    renameStory: async () => {},
    deleteStory: async () => {},
    clearStory: async () => {},
    send: async () => {},
    cancel: () => undefined,
    setIllustration: () => {},
    reloadMessages: async () => {},
  }),
}));
vi.mock("./sin/useSinCast", () => ({
  useSinCast: () => ({ cast: [], castIds: [], library: [], saving: false, libraryLoading: false, libraryError: "", saveCast: async () => [], reloadLibrary: () => {} }),
}));
vi.mock("./sin/useSinNotes", () => ({
  useSinNotes: () => ({ doc: null, error: "", loading: false, save: async () => ({}), reload: () => {} }),
}));
vi.mock("../hooks/useFeatureModel", () => ({
  useFeatureModel: () => ({ engine: "", model: "", enabled: false }),
}));
vi.mock("../gaea/components/Composer", () => ({
  Composer: () => <div data-testid="sin-composer-stub" />,
}));

const TRAJECTORY: Trajectory = {
  ok: true,
  turns: [{
    turn: 1, startedAt: 1750000000, records: [
      { seq: 10, kind: "user", ts: 1750000000, user: { text: "写一个开场" } },
      { seq: 19, kind: "assistant", ts: 1750000001, assistant: { text: "夜色落下来。" } },
    ],
  }],
};
const TIMELINE: ContextTimeline = {
  ok: true, window: 0,
  current: { system: 100, tools: 0, user: 200, inject: 50, assistant: 300, tool: 0 },
  stats: { turns: 1, steps: 1, injects: 0, compacts: 0, prunes: 0, toolCalls: 0, images: 0 },
  requests: [{
    seq: 1, ts: 1750000001, turn: 1, step: 1,
    category: { system: 100, tools: 0, user: 200, inject: 50, assistant: 300, tool: 0 },
    estimated: true, delta: { items: 3, tokens: 650, first: true },
  }],
  events: [], nodes: [], archive: [], files: [],
};

describe("OriginalSinPage v4.412 主区页签（复用办公轨迹/上下文看板）", () => {
  beforeEach(() => {
    localStorage.setItem("gaea-lang", "zh");
    sinTrajectoryMock.mockReset().mockResolvedValue(TRAJECTORY);
    sinContextViewMock.mockReset().mockResolvedValue(TIMELINE);
    sinNodeDetailMock.mockReset().mockResolvedValue({ seq: 10, kind: "user_message", text: "写一个开场", lines: 1 });
  });

  it("缺省故事页签：不拉看板数据", async () => {
    const OriginalSinPage = (await import("./OriginalSinPage")).default;
    render(<OriginalSinPage />);
    expect(await screen.findByTestId("sin-composer-stub")).toBeTruthy();
    expect(sinTrajectoryMock).not.toHaveBeenCalled();
    expect(sinContextViewMock).not.toHaveBeenCalled();
  });

  it("切「轨迹」：SinTrajectory 按当前故事 id 拉取并渲染轮次", async () => {
    const OriginalSinPage = (await import("./OriginalSinPage")).default;
    render(<OriginalSinPage />);
    fireEvent.click(screen.getByText("轨迹").closest("button") as HTMLElement);
    await waitFor(() => expect(sinTrajectoryMock).toHaveBeenCalledWith("sin_t1"));
    expect(await screen.findByText("写一个开场")).toBeTruthy();
  });

  it("切「上下文」：SinContextView 拉取；窗口未知显示「—」；Agent 网络不订阅", async () => {
    const OriginalSinPage = (await import("./OriginalSinPage")).default;
    render(<OriginalSinPage />);
    fireEvent.click(screen.getByText("上下文").closest("button") as HTMLElement);
    await waitFor(() => expect(sinContextViewMock).toHaveBeenCalledWith("sin_t1"));
    expect(await screen.findByText("当前上下文")).toBeTruthy();
    expect(screen.getByTitle("上下文窗口未知（无用量上报）")).toBeTruthy();
    expect(screen.getByText("娱乐空间")).toBeTruthy(); // spaceOverride（sin 无路径可判）
  });

  it("页签条复用办公样式：故事/轨迹/上下文三枚，主签显示「故事」", async () => {
    const OriginalSinPage = (await import("./OriginalSinPage")).default;
    render(<OriginalSinPage />);
    const labels = Array.from(document.querySelectorAll("button"))
      .map((b) => b.textContent)
      .filter((t) => ["故事", "轨迹", "上下文"].includes(t ?? ""));
    expect(labels).toEqual(["故事", "轨迹", "上下文"]);
    await waitFor(() => expect(screen.getByTestId("sin-composer-stub")).toBeTruthy());
  });
});
