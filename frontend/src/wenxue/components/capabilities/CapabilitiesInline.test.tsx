import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { CapabilitiesInline } from "./CapabilitiesInline";

vi.mock("../../hooks/useCapabilitiesData", () => ({
  useCapabilitiesData: () => ({
    view: { servers: [], skills: [] },
    busy: false,
    err: null,
    notice: null,
    reloading: false,
    adding: false,
    addingContext7: false,
    confirming: null,
    setConfirming: vi.fn(),
    setAdding: vi.fn(),
    reloadEngine: vi.fn().mockResolvedValue({ tools: 0, skills: 0 }),
    mutate: vi.fn().mockResolvedValue(true),
    addContext7: vi.fn(),
    summary: "0 个已连接 · 0 个异常 · 0 个技能",
    serverGroups: { failed: [], active: [] },
  }),
}));

vi.mock("../../lib/i18n", () => ({
  useT: () => (key: string) => ({
    "caps.title": "MCP 与技能",
    "caps.connectorsTab": "MCP 服务器",
    "caps.skillsTab": "Agent 技能",
    "caps.loading": "加载中…",
    "caps.reload": "热加载",
    "caps.reloading": "正在热加载…",
    "caps.reloadHint": "重新读取",
    "caps.addContext7Hint": "添加实时文档",
    "caps.addContext7Busy": "正在添加…",
    "caps.addServer": "添加服务器",
    "caps.noServers": "未配置 MCP 服务器",
  })[key] ?? key,
}));

vi.mock("../../lib/bridge", () => ({
  app: {
    SkillStats: vi.fn().mockResolvedValue([]),
    RetryMCPServer: vi.fn(),
    RemoveMCPServer: vi.fn(),
    SetMCPServerEnabled: vi.fn(),
    AddMCPServer: vi.fn(),
  },
}));

vi.mock("./ServersSection", () => ({
  ServerGroup: () => null,
  FailedServersNotice: () => null,
  AddServerForm: () => null,
}));

vi.mock("./ToolsSection", () => ({
  ToolsTabContent: () => <div data-testid="tools-content">工具目录</div>,
}));

vi.mock("./SkillsSection", () => ({
  SkillsSection: () => <div data-testid="skills-content">技能列表</div>,
}));

describe("CapabilitiesInline 三页签", () => {
  afterEach(() => cleanup());

  it("方向键切换页签、同步焦点，并在离开后保留已访问内容", () => {
    render(<CapabilitiesInline variant="drawer" />);

    const servers = screen.getByRole("tab", { name: /MCP 服务器/ });
    const tools = screen.getByRole("tab", { name: /工具/ });
    const skills = screen.getByRole("tab", { name: /Agent 技能/ });
    expect(screen.queryByTestId("tools-content")).toBeNull();

    fireEvent.keyDown(servers, { key: "ArrowRight" });
    expect(tools.getAttribute("aria-selected")).toBe("true");
    expect(document.activeElement).toBe(tools);
    expect(screen.getByTestId("tools-content")).toBeTruthy();

    fireEvent.keyDown(tools, { key: "End" });
    expect(skills.getAttribute("aria-selected")).toBe("true");
    expect(document.activeElement).toBe(skills);
    expect(screen.getByTestId("skills-content")).toBeTruthy();
    // 工具页已访问且保持挂载，返回时不丢搜索/展开状态。
    expect(screen.getByTestId("tools-content")).toBeTruthy();
  });
});
