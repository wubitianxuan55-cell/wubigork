import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import type { ProjectGroup } from "../lib/types";
import { WenxueHome, type WenxueHomeQuickAction } from "./WenxueHome";

const rootEntries = [
  { name: "章节", isDir: true, size: 0 },
  { name: "定稿", isDir: true, size: 0 },
  { name: "角色", isDir: true, size: 0 },
  { name: "设定", isDir: true, size: 0 },
  { name: "时间线", isDir: true, size: 0 },
];

vi.mock("../lib/bridge", () => ({
  app: {
    ListDir: vi.fn(async (rel: string) => {
      if (rel === "") return rootEntries;
      if (rel === "章节") {
        return [
          { name: "001-潮声.md", isDir: false, size: 80 },
          { name: "002-钟楼.md", isDir: false, size: 70 },
          { name: "003-雨夜证词.md", isDir: false, size: 90 },
        ];
      }
      if (rel === "定稿") return [{ name: "001-潮声.md", isDir: false, size: 80 }];
      if (rel === "角色") {
        return [
          { name: "林晚.md", isDir: false, size: 30 },
          { name: "陆则.md", isDir: false, size: 30 },
        ];
      }
      if (rel === "设定") {
        return [
          { name: "世界观.md", isDir: false, size: 30 },
          { name: "规则.md", isDir: false, size: 30 },
        ];
      }
      if (rel === "时间线") return [{ name: "时间线.md", isDir: false, size: 30 }];
      return [];
    }),
  },
}));

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

const quickActions: WenxueHomeQuickAction[] = [
  {
    key: "chapter-continue",
    label: "续写推进",
    Icon: ({ size }) => <span data-testid="qa-icon">{size}</span>,
    prompt: "续写下一章并回写时间线",
  },
  {
    key: "outline-push",
    label: "大纲推演",
    Icon: ({ size }) => <span data-testid="qa-icon">{size}</span>,
    prompt: "推演未来十章",
  },
];

function renderHome(overrides: Partial<Parameters<typeof WenxueHome>[0]> = {}) {
  const props: Parameters<typeof WenxueHome>[0] = {
    projects: [],
    currentProject: undefined,
    projectsLoaded: true,
    recentSessions: [],
    currentSessionPath: undefined,
    quickActions,
    onOpenProject: vi.fn(async () => "C:/AI/tuwen/雾港来信"),
    onCreateNovel: vi.fn(),
    onOpenHistory: vi.fn(),
    onResumeSession: vi.fn(),
    onNewSession: vi.fn(),
    onInsertPrompt: vi.fn(),
    onSwitchTab: vi.fn(),
    onOpenLibrary: vi.fn(),
    ...overrides,
  };
  return { props, ...render(<WenxueHome {...props} />) };
}

describe("WenxueHome", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("没有作品时展示书斋建档首页，主 CTA 创建第一部小说", () => {
    const { props } = renderHome();
    expect(screen.getByTestId("wenxue-home-library")).toBeTruthy();
    expect(screen.getByText("不需要先准备完整的世界。")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: /创建第一部小说/ }));
    expect(props.onCreateNovel).toHaveBeenCalled();
  });

  it("已有作品但未进入时以书架呈现，点击作品进入对应工作空间", async () => {
    const first = project({ current: false });
    const second = project({
      path: "C:/AI/tuwen/长夜灯塔",
      name: "长夜灯塔",
      current: false,
      sessions: [],
      modTime: Date.now() - 3 * 86_400_000,
    });
    const { props } = renderHome({ projects: [first, second] });
    expect(screen.getByText("作品书架")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: /长夜灯塔/ }));
    await waitFor(() =>
      expect(props.onOpenProject).toHaveBeenCalledWith("C:/AI/tuwen/长夜灯塔"));
  });

  it("进入作品后展示作品概况、最近章节和工作区入口", async () => {
    const current = project();
    const { props } = renderHome({ projects: [current], currentProject: current });
    expect(screen.getByTestId("wenxue-home-project")).toBeTruthy();
    await waitFor(() => {
      expect(screen.getByText("第 3 章 · 雨夜证词")).toBeTruthy();
      const metrics = screen.getByLabelText("作品概况");
      expect(within(metrics).getByText("3")).toBeTruthy();
      expect(within(metrics).getByText("1")).toBeTruthy();
      expect(within(metrics).getAllByText("2")).toHaveLength(2);
    });
    fireEvent.click(screen.getByRole("button", { name: /继续正文/ }));
    expect(props.onInsertPrompt).toHaveBeenCalledWith("续写下一章并回写时间线");
    fireEvent.click(screen.getByRole("button", { name: /角色人物志/ }));
    expect(props.onSwitchTab).toHaveBeenCalledWith("characters");
  });

  it("作品首页可回到最近会话与作品书架", () => {
    const current = project();
    const { props } = renderHome({ projects: [current], currentProject: current });
    fireEvent.click(screen.getByRole("button", { name: /回到最近会话/ }));
    expect(props.onResumeSession).toHaveBeenCalledWith(current.sessions[0].path, current.path);
    fireEvent.click(screen.getByRole("button", { name: /作品书架/ }));
    expect(props.onOpenLibrary).toHaveBeenCalled();
  });
});
