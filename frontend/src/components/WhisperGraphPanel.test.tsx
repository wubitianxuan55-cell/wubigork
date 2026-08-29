import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { WhisperGraphPanel } from "./WhisperGraphPanel";
import { ToastProvider } from "../gaea/components/Toast";

const { subgraphSpy, proactiveSpy } = vi.hoisted(() => ({
  subgraphSpy: vi.fn(),
  proactiveSpy: vi.fn(),
}));

vi.mock("../gaea/lib/bridge", () => ({
  app: {
    WhisperGraphSubgraph: (...args: unknown[]) => subgraphSpy(...args),
    WhisperProactiveNow: (...args: unknown[]) => proactiveSpy(...args),
  },
}));

const wrap = (node: React.ReactNode) => <ToastProvider>{node}</ToastProvider>;

// 基础样例：中心「阿黎」+ 一跳邻接「爬山」（边类型「喜欢」）。
const GRAPH_1 = {
  nodes: [
    { id: "n0", name: "阿黎", type: "person", weight: 0.9 },
    { id: "n1", name: "爬山", type: "hobby", weight: 0.6 },
  ],
  edges: [{ from: "n0", to: "n1", type: "喜欢", weight: 0.8 }],
};

// hops=2 样例：中心「阿黎」→ 一跳「爬山」→ 二跳「山顶茶馆」。
const GRAPH_2 = {
  nodes: [
    { id: "n0", name: "阿黎", type: "person", weight: 0.9 },
    { id: "n1", name: "爬山", type: "hobby", weight: 0.6 },
    { id: "n2", name: "山顶茶馆", type: "place", weight: 0.4 },
  ],
  edges: [
    { from: "n0", to: "n1", type: "喜欢", weight: 0.8 },
    { from: "n1", to: "n2", type: "常去", weight: 0.5 },
  ],
};

describe("WhisperGraphPanel 轻语关系图谱", () => {
  beforeEach(() => {
    subgraphSpy.mockReset();
    proactiveSpy.mockReset();
  });

  it("未查询前显示提示文案", () => {
    render(wrap(<WhisperGraphPanel open personalityId="pid-1" onClose={() => {}} />));
    expect(screen.getByText(/输入实体并点击「查询」/)).toBeTruthy();
  });

  it("查询 h1：渲染中心节点 + 邻接节点 + 边标签，调用 bridge(personalityId, entity, hops)", async () => {
    subgraphSpy.mockResolvedValue(GRAPH_1);
    render(wrap(<WhisperGraphPanel open personalityId="pid-1" onClose={() => {}} />));

    fireEvent.change(screen.getByLabelText("实体名"), { target: { value: "阿黎" } });
    fireEvent.click(screen.getByRole("button", { name: /查询/ }));

    expect(await screen.findByText("爬山")).toBeTruthy();
    expect(screen.getByText("阿黎")).toBeTruthy();
    // 边标签（关系类型）
    expect(screen.getByText("喜欢")).toBeTruthy();
    expect(subgraphSpy).toHaveBeenCalledWith("pid-1", "阿黎", 1);
  });

  it("hops=2：外圈二跳节点渲染，bridge 收到 hops=2", async () => {
    subgraphSpy.mockResolvedValue(GRAPH_2);
    render(wrap(<WhisperGraphPanel open personalityId="pid-1" onClose={() => {}} />));

    fireEvent.change(screen.getByLabelText("实体名"), { target: { value: "阿黎" } });
    fireEvent.change(screen.getByLabelText("跳数"), { target: { value: "2" } });
    fireEvent.click(screen.getByRole("button", { name: /查询/ }));

    expect(await screen.findByText("山顶茶馆")).toBeTruthy();
    expect(screen.getByText("爬山")).toBeTruthy();
    expect(screen.getByText("常去")).toBeTruthy();
    expect(subgraphSpy).toHaveBeenCalledWith("pid-1", "阿黎", 2);
  });

  it("空态：nodes 为空时展示「图谱暂无关系」", async () => {
    subgraphSpy.mockResolvedValue({ nodes: [], edges: [] });
    render(wrap(<WhisperGraphPanel open personalityId="pid-1" onClose={() => {}} />));

    fireEvent.change(screen.getByLabelText("实体名"), { target: { value: "阿黎" } });
    fireEvent.click(screen.getByRole("button", { name: /查询/ }));

    expect(await screen.findByText(/图谱暂无关系/)).toBeTruthy();
  });

  it("主动关心 shouldSend=true：展示消息类型徽标 + promptHint + 评估说明", async () => {
    proactiveSpy.mockResolvedValue({
      shouldSend: true,
      messageType: "check_in",
      promptHint: "昨晚你说今天要赶项目，想问问进展顺利吗？",
    });
    render(wrap(<WhisperGraphPanel open personalityId="pid-1" onClose={() => {}} />));

    fireEvent.click(screen.getByRole("button", { name: /轻语先开口/ }));

    expect(await screen.findByText("关怀问候")).toBeTruthy();
    expect(screen.getByText("昨晚你说今天要赶项目，想问问进展顺利吗？")).toBeTruthy();
    expect(screen.getByText(/这是评估结果，实际推送由定时器或用户确认后触发/)).toBeTruthy();
    expect(proactiveSpy).toHaveBeenCalledWith("pid-1");
  });

  it("主动关心 shouldSend=false：toast「现在没有合适的时机，轻语选择安静陪伴」", async () => {
    proactiveSpy.mockResolvedValue({ shouldSend: false });
    render(wrap(<WhisperGraphPanel open personalityId="pid-1" onClose={() => {}} />));

    fireEvent.click(screen.getByRole("button", { name: /轻语先开口/ }));

    expect(await screen.findByText(/现在没有合适的时机，轻语选择安静陪伴/)).toBeTruthy();
    expect(proactiveSpy).toHaveBeenCalledWith("pid-1");
  });

  it("点击节点以该节点为中心重新查询（交互加分项）", async () => {
    subgraphSpy.mockResolvedValue(GRAPH_1);
    render(wrap(<WhisperGraphPanel open personalityId="pid-1" onClose={() => {}} />));

    fireEvent.change(screen.getByLabelText("实体名"), { target: { value: "阿黎" } });
    fireEvent.click(screen.getByRole("button", { name: /查询/ }));
    await screen.findByText("爬山");

    fireEvent.click(screen.getByText("爬山"));
    expect(subgraphSpy).toHaveBeenLastCalledWith("pid-1", "爬山", 1);
  });
});
