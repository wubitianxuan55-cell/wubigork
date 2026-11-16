import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ProjectGroup } from "../lib/types";
import { WenxueHome } from "./WenxueHome";

const project = (overrides: Partial<ProjectGroup> = {}): ProjectGroup => ({
  path: "C:/AI/tuwen/雾港来信",
  name: "雾港来信",
  current: true,
  sessions: [
    {
      path: "C:/AI/tuwen/雾港来信/.gaea/sessions/a.jsonl",
      preview: "雨夜追查的证物逻辑",
      turns: 4,
      modTime: Date.now() - 60 * 60_000,
      current: false,
    },
  ],
  archived: [],
  modTime: Date.now() - 60 * 60_000,
  ...overrides,
});

function renderHome(overrides: Partial<Parameters<typeof WenxueHome>[0]> = {}) {
  const props: Parameters<typeof WenxueHome>[0] = {
    projects: [],
    projectsLoaded: true,
    onOpenProject: vi.fn(async () => "C:/AI/tuwen/雾港来信"),
    onCreateNovel: vi.fn(),
    onOpenHistory: vi.fn(),
    privateMode: false,
    onTogglePrivacy: vi.fn(),
    ...overrides,
  };
  return { props, ...render(<WenxueHome {...props} />) };
}

describe("WenxueHome", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("空作品库只保留一个建档主操作", () => {
    const { props } = renderHome();
    expect(screen.getByTestId("wenxue-home-library")).toBeTruthy();
    expect(screen.getByRole("heading", { name: "开始第一部作品" })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: /新建小说/ }));
    expect(props.onCreateNovel).toHaveBeenCalledTimes(1);
  });

  it("作品库用封面卡片呈现作品并打开对应目录", async () => {
    const first = project();
    const second = project({
      path: "C:/AI/tuwen/长夜灯塔",
      name: "长夜灯塔",
      sessions: [],
      modTime: Date.now() - 3 * 86_400_000,
    });
    const { props } = renderHome({ projects: [second, first] });

    expect(screen.getByRole("heading", { name: "选择一部作品" })).toBeTruthy();
    expect(screen.getByText("2 部作品 · 1 个会话")).toBeTruthy();
    expect(document.querySelectorAll(".wx-home-work-card")).toHaveLength(2);
    expect(document.querySelectorAll(".wx-home-work-cover")).toHaveLength(2);
    fireEvent.click(screen.getByRole("button", { name: /长夜灯塔/ }));
    await waitFor(() =>
      expect(props.onOpenProject).toHaveBeenCalledWith("C:/AI/tuwen/长夜灯塔"));
  });

  it("隐私模式隐藏作品名，并保留切换入口", () => {
    const { props } = renderHome({ projects: [project()], privateMode: true });
    expect(document.body.textContent).not.toContain("雾港来信");
    expect(screen.getByRole("button", { name: /作品 01/ })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: /已隐身/ }));
    expect(props.onTogglePrivacy).toHaveBeenCalledTimes(1);
  });

  it("加载中显示稳定骨架，不提前闪现空作品引导", () => {
    renderHome({ projectsLoaded: false });
    expect(screen.getByRole("heading", { name: "正在打开作品库" })).toBeTruthy();
    expect(screen.getByRole("status")).toBeTruthy();
    expect(screen.queryByText("开始第一部作品")).toBeNull();
  });
});
