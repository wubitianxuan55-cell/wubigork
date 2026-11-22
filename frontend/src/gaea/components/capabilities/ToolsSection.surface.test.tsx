// 能力面板「工具」tab 装配面活取（2026-11-21 面板缝）：
// 活取面=分组渲染（内置域/应用/MCP）+ 模式藏面灰显组 + 模式徽标；
// 引擎未就绪（reject/空面）=静态清单兜底（现状行为）。
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { ToolsTabContent } from "./ToolsSection";
import type { ToolSurfaceViewB } from "../../lib/bridge/model";

const gaeaSurface = vi.fn<() => Promise<ToolSurfaceViewB>>();

vi.mock("../../lib/bridge", () => ({
  app: {
    GaeaToolSurface: () => gaeaSurface(),
    WenxueToolSurface: vi.fn().mockRejectedValue(new Error("mock")),
  },
}));

function surface(mode: string, tools: ToolSurfaceViewB["tools"]): ToolSurfaceViewB {
  return { board: "gaea", mode, tools };
}

beforeEach(() => {
  localStorage.setItem("gaea-lang", "zh"); // jsdom navigator=en-US 漂英文词典（同款钉法）
});
afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("ToolsTabContent 装配面活取", () => {
  it("造价档：分组渲染 + 模式徽标 + 藏面灰显组", async () => {
    gaeaSurface.mockResolvedValue(surface("cost", [
      { name: "bash", description: "执行shell命令(合并stdout+stderr)", source: "builtin-domain", group: "", hidden: false },
      { name: "cost_project", description: "读取工料机造价库（workcost）", source: "extra-app", group: "cost", hidden: false },
      { name: "cost_search", description: "搜索成本库", source: "builtin-domain", group: "cost", hidden: false },
      { name: "longdoc_build", description: "按清单组装整册", source: "extra-app", group: "longdoc", hidden: true },
    ]));
    render(<ToolsTabContent toolCounts={{ bash: 3 }} />);
    await waitFor(() => expect(screen.getByText("造价档")).toBeTruthy());
    // 功能分组（2026-11-21 拍板）：bash→文件与命令、cost 族→造价专用、藏面组
    expect(screen.getByText("文件与命令")).toBeTruthy();
    expect(screen.getByText("造价专用")).toBeTruthy();
    expect(screen.getByText("本档隐藏（1）")).toBeTruthy();
    expect(screen.getByText("cost_project")).toBeTruthy();
    expect(screen.getByText("longdoc_build")).toBeTruthy();
    // 模型可见口径描述进卡（活取描述优先于静态表）
    expect(screen.getByText("读取工料机造价库（workcost）")).toBeTruthy();
    // 头部计数=可见/总（拆成主数字与单位，读取可访问名锁定口径）
    expect(screen.getByLabelText("可见 3 / 共 4")).toBeTruthy();
  });

  it("引擎未就绪（reject）：静态清单兜底", async () => {
    gaeaSurface.mockRejectedValue(new Error("engine not ready"));
    render(<ToolsTabContent toolCounts={{}} />);
    // 折叠组内卡片不进断言面（GSAP collapse）——组头即兜底标志。
    await waitFor(() => expect(screen.getByText("文件与命令")).toBeTruthy());
    expect(screen.queryByText("本档隐藏（")).toBeNull();
  });

  it("空面（未初始化）：同样兜底", async () => {
    gaeaSurface.mockResolvedValue(surface("standard", []));
    render(<ToolsTabContent toolCounts={{}} />);
    await waitFor(() => expect(screen.getByText("文件与命令")).toBeTruthy());
    expect(screen.queryByText(/档$/)).toBeNull();
  });

  it("搜索无结果给出清除动作，不留下死胡同", async () => {
    gaeaSurface.mockResolvedValue(surface("standard", [
      { name: "bash", description: "执行 shell 命令", source: "builtin-domain", group: "", hidden: false },
    ]));
    render(<ToolsTabContent toolCounts={{}} />);
    await waitFor(() => expect(screen.getByText("文件与命令")).toBeTruthy());

    fireEvent.change(screen.getByLabelText("搜索工具"), { target: { value: "不存在的工具" } });
    expect(screen.getByText("没有找到工具")).toBeTruthy();
    expect(screen.getByText(/“不存在的工具” 的匹配结果/)).toBeTruthy();

    fireEvent.click(screen.getByText("清除搜索"));
    expect((screen.getByLabelText("搜索工具") as HTMLInputElement).value).toBe("");
    expect(screen.getByText("文件与命令")).toBeTruthy();
  });
});
