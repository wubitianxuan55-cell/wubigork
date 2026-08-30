import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { WhisperGraphPanel } from "./WhisperGraphPanel";
import { ToastProvider } from "../gaea/components/Toast";

const { subgraphSpy, proactiveSpy } = vi.hoisted(() => ({
  subgraphSpy: vi.fn(),
  proactiveSpy: vi.fn(),
}));

// 事件订阅捕获：WhisperGraphPanel 通过 subscribeForSpace 订阅
// gaea-whisper-proactive（v4.3c 定时推送）。mock 掉 events 模块，
// 捕获注册的 handler，测试里手动触发模拟后端推送。
const proactiveHandlers: Array<(data: unknown) => void> = [];
vi.mock("../events", () => ({
  BACKEND_EVENTS: { WHISPER_PROACTIVE: "gaea-whisper-proactive" },
  subscribeForSpace: (_event: string, handler: (data: unknown) => void) => {
    proactiveHandlers.push(handler);
    return () => {};
  },
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

// v4.9 图谱情绪维度：边带 emotionLabel（正面）。
const GRAPH_EMOTION = {
  nodes: [
    { id: "n0", name: "阿黎", type: "person", weight: 0.9 },
    { id: "n1", name: "爬山", type: "hobby", weight: 0.6 },
  ],
  edges: [{ from: "n0", to: "n1", type: "喜欢", weight: 0.8, emotionLabel: "正面" }],
};

// v4.9 因果关联边（event_chain → 「因果」，虚线琥珀色）。
const GRAPH_CAUSAL = {
  nodes: [
    { id: "n0", name: "工作", type: "", weight: 0.9 },
    { id: "n1", name: "睡眠", type: "", weight: 0.5 },
  ],
  edges: [{ from: "n0", to: "n1", type: "因果", weight: 0.5 }],
};

describe("WhisperGraphPanel 轻语关系图谱", () => {
  beforeEach(() => {
    subgraphSpy.mockReset();
    proactiveSpy.mockReset();
    proactiveHandlers.length = 0;
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

  it("v4.9 图谱边按情绪着色：图例渲染 + 正面边使用绿色 stroke", async () => {
    subgraphSpy.mockResolvedValue(GRAPH_EMOTION);
    render(wrap(<WhisperGraphPanel open personalityId="pid-1" onClose={() => {}} />));

    fireEvent.change(screen.getByLabelText("实体名"), { target: { value: "阿黎" } });
    fireEvent.click(screen.getByRole("button", { name: /查询/ }));

    expect(await screen.findByText("爬山")).toBeTruthy();
    // 情绪图例
    expect(screen.getByText("正面")).toBeTruthy();
    expect(screen.getByText("负面")).toBeTruthy();
    expect(screen.getByText("中性")).toBeTruthy();
    // 正面边 → emerald 描边
    const line = document.querySelector("line");
    expect(line?.getAttribute("class")).toContain("stroke-emerald-400/70");
  });

  it("v4.9 因果关联边使用虚线琥珀色描边，图例含「因果」", async () => {
    subgraphSpy.mockResolvedValue(GRAPH_CAUSAL);
    render(wrap(<WhisperGraphPanel open personalityId="pid-1" onClose={() => {}} />));

    fireEvent.change(screen.getByLabelText("实体名"), { target: { value: "工作" } });
    fireEvent.click(screen.getByRole("button", { name: /查询/ }));

    expect(await screen.findByText("睡眠")).toBeTruthy();
    const line = document.querySelector("line");
    expect(line?.getAttribute("class")).toContain("stroke-amber-400/80");
    expect(line?.getAttribute("stroke-dasharray")).toBe("4 2");
    // 图例「因果」+ 边标签「因果」都可能出现，用 getAllByText
    expect(screen.getAllByText("因果").length).toBeGreaterThanOrEqual(1);
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
    expect(screen.getByText(/手动评估结果；定时推送到达时也会在这里显示/)).toBeTruthy();
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

  it("定时推送事件（v4.3c）：收到 gaea-whisper-proactive 显示气泡 + birthday 徽标", async () => {
    render(wrap(<WhisperGraphPanel open personalityId="pid-1" onClose={() => {}} />));

    // 打开时注册了订阅 handler
    expect(proactiveHandlers.length).toBeGreaterThan(0);
    const handler = proactiveHandlers[proactiveHandlers.length - 1];

    handler({
      personalityID: "pid-1",
      messageType: "birthday",
      promptHint: "今天是你的生日，祝你生日快乐！",
      space: "play",
    });

    expect(await screen.findByText("生日祝福")).toBeTruthy();
    expect(screen.getByText("今天是你的生日，祝你生日快乐！")).toBeTruthy();
  });

  it("定时推送事件：其他人格（personalityID 不匹配）不显示", () => {
    render(wrap(<WhisperGraphPanel open personalityId="pid-1" onClose={() => {}} />));
    const handler = proactiveHandlers[proactiveHandlers.length - 1];

    handler({ personalityID: "pid-other", messageType: "check_in", promptHint: "别的轻语", space: "play" });

    expect(screen.queryByText("别的轻语")).toBeNull();
    expect(screen.queryByText("关怀问候")).toBeNull();
  });
});
