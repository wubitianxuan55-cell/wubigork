// 能力面板「工具」tab 装配面活取（与办公 ToolsSection.surface.test 同源镜像，
// 数据面换文学口径）：活取面=功能分组渲染（wenxue_* 归「图文专属」）+ 藏面灰显组；
// 引擎未就绪（reject/空面）=liveTools/静态清单兜底。
import { describe, expect, it, vi, afterEach } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { ToolsTabContent } from "./ToolsSection";
import type { ToolSurfaceViewB } from "../../lib/bridge/wenxue";

const surfaceFn = vi.fn<() => Promise<ToolSurfaceViewB>>();

vi.mock("../../lib/bridge", () => ({
  app: {
    WenxueToolSurface: () => surfaceFn(),
  },
}));

function surface(tools: ToolSurfaceViewB["tools"]): ToolSurfaceViewB {
  return { board: "wenxue", mode: "", tools };
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("ToolsTabContent 装配面活取（文学口径）", () => {
  it("按功能分组：wenxue_* 归图文专属，通用工具归语义小节，藏面灰显组收尾", async () => {
    surfaceFn.mockResolvedValue(surface([
      { name: "wenxue_chapter_list", description: "章节清单——章号/标题/字数", source: "extra-app", group: "", hidden: false },
      { name: "bash", description: "执行shell命令", source: "builtin-domain", group: "", hidden: false },
      { name: "image_gen", description: "生图", source: "extra-app", group: "media", hidden: false },
      { name: "mcp_deferred", description: "延迟加载的 MCP 工具", source: "mcp:demo", hidden: true },
    ]));
    render(<ToolsTabContent toolCounts={{ bash: 3 }} />);
    await waitFor(() => expect(screen.getByLabelText("可见 3 / 共 4")).toBeTruthy());
    // 功能分组（与办公 2026-11-21 拍板同源：按功能，不按装配来源）
    expect(screen.getByText("图文专属")).toBeTruthy();
    expect(screen.getByText("文件与命令")).toBeTruthy();
    expect(screen.getByText("图面与生图")).toBeTruthy();
    expect(screen.getByText("本档隐藏（1）")).toBeTruthy();
    // 模型可见口径描述进卡（活取描述优先于静态表）
    expect(screen.getByText("章节清单——章号/标题/字数")).toBeTruthy();
    expect(screen.queryByText("内置域工具")).toBeNull();
  });

  it("引擎未就绪（reject）：图文静态清单兜底", async () => {
    surfaceFn.mockRejectedValue(new Error("engine not ready"));
    render(<ToolsTabContent toolCounts={{}} />);
    // 折叠组内卡片不进断言面（GSAP collapse）——组头即兜底标志。
    await waitFor(() => expect(screen.getByText("图文专属")).toBeTruthy());
    expect(screen.getByText("文件与命令")).toBeTruthy();
    expect(screen.queryByText("本档隐藏（")).toBeNull();
    expect(screen.queryByText("其他")).toBeNull();
  });

  it("引擎未就绪但有实时名单：名单里不在任何分组的工具归「其他」", async () => {
    surfaceFn.mockRejectedValue(new Error("engine not ready"));
    render(<ToolsTabContent toolCounts={{}} liveTools={["bash", "custom_tool_x"]} />);
    await waitFor(() => expect(screen.getByText("文件与命令")).toBeTruthy());
    expect(screen.getByText("其他")).toBeTruthy();
    expect(screen.getByText("custom_tool_x")).toBeTruthy();
  });

  it("空面（未初始化）：同样兜底", async () => {
    surfaceFn.mockResolvedValue(surface([]));
    render(<ToolsTabContent toolCounts={{}} />);
    await waitFor(() => expect(screen.getByText("图文专属")).toBeTruthy());
    // 文学无运行模式：不出现模式档徽标
    expect(document.querySelector(".tools-mode-badge")).toBeNull();
  });

  it("搜索无结果给出清除动作，不留下死胡同", async () => {
    surfaceFn.mockResolvedValue(surface([
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
