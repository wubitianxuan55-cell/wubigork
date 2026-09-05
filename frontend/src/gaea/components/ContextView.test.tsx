import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactElement } from "react";
import { LocaleProvider } from "../lib/i18n";
import type { ContextTimeline } from "../lib/types";

// 本文件组件重（SVG 径向图/大卡片墙），全量 vitest 并行负载下 5s 默认上限
// 会超时（v4.95 收口实测：单独复跑全绿、全量三连超时）——文件级放宽到 20s，
// 只放宽时限不改断言。
vi.setConfig({ testTimeout: 20_000 });

// ContextView 走 useT：钉住 zh 让既有中文文案断言继续成立
const renderT = (ui: ReactElement) => {
  localStorage.setItem("gaea-lang", "zh");
  return render(<LocaleProvider>{ui}</LocaleProvider>);
};

const contextViewMock = vi.fn();
const contextNodeDetailMock = vi.fn();
const subscribeAgentNetworkMock = vi.fn((cb: (n: unknown) => void) => {
  cb({
    ok: true,
    window: 1_000_000,
    root: { id: "root", name: "主 agent", kind: "root", status: "completed", toolCalls: 0, errors: 0, tokens: 0, children: [] },
  });
  return () => {};
});
const reloadAgentNetworkMock = vi.fn();

vi.mock("../lib/bridge", () => ({
  app: { ContextView: contextViewMock, ContextNodeDetail: contextNodeDetailMock },
  openExternal: vi.fn(),
  onEvent: vi.fn(() => () => {}),
}));
vi.mock("../lib/agentNetworkStore", () => ({
  subscribeAgentNetwork: (cb: (n: unknown) => void) => subscribeAgentNetworkMock(cb),
  reloadAgentNetwork: () => reloadAgentNetworkMock(),
}));

const TIMELINE: ContextTimeline = {
  ok: true,
  window: 1_000_000,
  current: { system: 2100, tools: 10400, user: 20, inject: 21900, assistant: 93400, tool: 114000 },
  stats: { turns: 2, steps: 60, injects: 6, compacts: 1, prunes: 0, toolCalls: 279, images: 0, cacheHitPercent: 99.57, costEstimate: 3.83 },
  requests: [
    {
      seq: 8, ts: 1750000000, turn: 1, step: 1,
      category: { system: 2100, tools: 10400, user: 20, inject: 21000, assistant: 1200, tool: 8000 },
      briefUser: "grep internal/gaea/config",
      briefResp: "read internal/gaea/config/config.go",
      promptTokens: 350600, outputTokens: 93, cacheHitTokens: 350200, cacheMissTokens: 400,
    },
  ],
  events: [
    { kind: "inject", seq: 2, delta: 10500, source: "指令注入 · .gaea\\AGENTS.md", turn: 1, step: 4, ts: 1750000000 },
    { kind: "compact", seq: 30, delta: -535500, source: "ratio", turn: 2, step: 20, ts: 1750000100 },
  ],
  nodes: [
    { seq: 1, cat: "system", tokens: 2100, text: "你是 gaea，土壤修复工程办公专用 AI 助手……" },
    { seq: 4, cat: "tool", tokens: 8000, text: "package main 这是一段足够长的工具输出内容，用于验证上下文浏览器节点的展开与收起交互，全文预览在这里展示。" },
  ],
  archive: [
    { seq: 30, cat: "user", tokens: 12000, text: "旧的一轮用户输入内容", gone: 30 },
  ],
  files: [
    { seq: 5, ts: 1750000005, turn: 1, step: 1, tool: "read_file", action: "read", path: "internal/gaea/config/config.go" },
    { seq: 9, ts: 1750000009, turn: 1, step: 2, tool: "write_file", action: "write", path: "docs/结论.md" },
  ],
};

describe("ContextView 上下文页卡片墙（v4.71 卡片化）", () => {
  beforeEach(() => {
    contextViewMock.mockReset();
    contextViewMock.mockResolvedValue(TIMELINE);
    contextNodeDetailMock.mockReset();
    contextNodeDetailMock.mockResolvedValue({
      seq: 4,
      kind: "tool_result",
      ts: 1750000004,
      tool: "read_file",
      args: '{"path":"internal/gaea/config/config.go"}',
      output: "```go\npackage main\nfunc Load() error {\n\treturn loadFromDisk()\n}\n```\n",
      lines: 5,
    });
    subscribeAgentNetworkMock.mockClear();
    reloadAgentNetworkMock.mockClear();
  });

  it("8 个统计项各自成小卡；Token 环形 / 耗时卡 / 会话信息为三张大卡", async () => {
    const { ContextView } = await import("./ContextView");
    const { container } = renderT(<ContextView running={false} />);
    expect(await screen.findByText("工具调用")).toBeTruthy();
    // v4.71：统计不再合成 1 张大卡——每个统计项是独立 tile（8 个）
    expect(container.querySelectorAll('[data-testid^="stat-tile-"]')).toHaveLength(8);
    expect(container.querySelector('[data-testid="stat-tile-turns"]')?.classList.contains("ctx-tile")).toBe(true);
    expect(screen.getByText("279")).toBeTruthy();
    expect(screen.getByText("99.89%")).toBeTruthy(); // TokenCard 环心缓存命中（requests 汇总 350200/350600，v4.69 两位小数）
    expect(screen.getByText("缓存输入")).toBeTruthy();
    expect(screen.getByText("未缓存输入")).toBeTruthy();
    expect(screen.getByText("耗时统计")).toBeTruthy();
    expect(screen.getByText("模型等待")).toBeTruthy();
    expect(screen.getByText("会话信息")).toBeTruthy();
  });

  it("当前上下文宽卡：水位、缓存/费用与六分类图例", async () => {
    const { ContextView } = await import("./ContextView");
    renderT(<ContextView running={false} />);
    expect(await screen.findByText("当前上下文")).toBeTruthy();
    // v4.68：水位只在当前上下文宽卡显示一处
    expect(screen.getAllByText(/241\.8k \/ 1\.0M · 24%/).length).toBe(1);
    // 缓存/费用信息已分别由 TokenCard / StatsCard 承载（dsh 同构）
  });

  it("接近上限时水位警示（≥90%）", async () => {
    contextViewMock.mockResolvedValue({
      ...TIMELINE,
      window: 100_000,
      current: { system: 20000, tools: 20000, user: 5000, inject: 20000, assistant: 20000, tool: 10000 },
    });
    const { ContextView } = await import("./ContextView");
    renderT(<ContextView running={false} />);
    expect(await screen.findByText(/上下文接近上限，建议压缩或新建会话/)).toBeTruthy();
    expect(screen.getByText(/已接近上下文上限/)).toBeTruthy();
  });

  it("空数据：空态引导 + 头部仍展示 0 水位", async () => {
    contextViewMock.mockResolvedValue({
      ok: true,
      window: 0,
      current: { system: 0, tools: 0, user: 0, inject: 0, assistant: 0, tool: 0 },
      stats: { turns: 0, steps: 0, injects: 0, compacts: 0, prunes: 0, toolCalls: 0, images: 0 },
      requests: [], events: [], nodes: [], archive: [], files: [],
    });
    const { ContextView } = await import("./ContextView");
    renderT(<ContextView running={false} />);
    expect(await screen.findByText("暂无上下文数据")).toBeTruthy();
    expect(screen.getByText(/\/ 0 tokens/)).toBeTruthy();
  });

  it("渲染趋势图与事件流；默认选中最新请求、详情内联趋势卡", async () => {
    const { ContextView } = await import("./ContextView");
    renderT(<ContextView running={false} />);
    expect(await screen.findByText("上下文趋势")).toBeTruthy();
    // v4.69：默认选中最新请求 → 详情（实际 prompt 等）无需点击即内联展示在趋势卡内
    expect(await screen.findByText(/实际 prompt 350\.6k/)).toBeTruthy();
    expect(await screen.findByText(/指令注入 · \.gaea/)).toBeTruthy();
    expect((await screen.findAllByText("压缩")).length).toBeGreaterThan(0);
    expect(await screen.findByText(/-535\.5k/)).toBeTruthy();
  });

  it("点击柱后展示步骤详情", async () => {
    const { ContextView } = await import("./ContextView");
    const { container } = renderT(<ContextView running={false} />);
    await screen.findByText("上下文趋势");
    const rect = container.querySelector("svg rect");
    if (!rect) throw new Error("trend bar rect not found");
    fireEvent.click(rect);
    expect(await screen.findByText(/grep internal\/gaea\/config/)).toBeTruthy();
    expect(screen.getByText(/实际 prompt 350.6k/)).toBeTruthy();
    expect(screen.getByText(/占窗口 4%/)).toBeTruthy();
  });

  it("v4.79 对比上一步：请求详情逐类 signed delta 徽标 + 跨压缩近似标注", async () => {
    const withDelta: ContextTimeline = {
      ...TIMELINE,
      requests: [
        {
          ...TIMELINE.requests[0],
          delta: {
            items: 3,
            tokens: 1500,
            approx: true,
            byCat: [
              { cat: "user", items: 1, tokens: 800 },
              { cat: "assistant", items: 1, tokens: 500 },
              { cat: "tool", items: 1, tokens: 200 },
            ],
          },
        },
      ],
    };
    contextViewMock.mockResolvedValue(withDelta);
    const { ContextView } = await import("./ContextView");
    renderT(<ContextView running={false} />);
    // v4.69 默认选中最新请求 → delta 条随详情内联展示
    const strip = await screen.findByTestId("ctx-delta-strip");
    expect(strip).toBeTruthy();
    expect(strip.textContent).toContain("较上一步");
    // 合计 +1.5k（绿）+ 跨压缩近似标注
    expect(strip.textContent).toContain("+1.5k");
    expect(strip.textContent).toContain("跨压缩，近似");
    // 逐类徽标：短名 + signed 项数/tokens（作用域限 delta 条，防构成卡图例同名干扰）
    expect(strip.textContent).toContain("用户");
    expect(strip.textContent).toContain("+1·+800");
    expect(strip.textContent).toContain("助手");
    expect(strip.textContent).toContain("工具结果");
  });

  it("v4.79 对比上一步：首个请求显示基线说明，无变化显示相同", async () => {
    const firstOnly: ContextTimeline = {
      ...TIMELINE,
      requests: [
        { ...TIMELINE.requests[0], delta: { items: 5, tokens: 2100, first: true, byCat: [{ cat: "system", items: 1, tokens: 2100 }] } },
      ],
    };
    contextViewMock.mockResolvedValue(firstOnly);
    const { ContextView } = await import("./ContextView");
    renderT(<ContextView running={false} />);
    expect(await screen.findByTestId("ctx-delta-strip")).toBeTruthy();
    expect(screen.getByText("首个请求（对比基线=空）")).toBeTruthy();

    const noChange: ContextTimeline = {
      ...TIMELINE,
      requests: [{ ...TIMELINE.requests[0], delta: { items: 0, tokens: 0 } }],
    };
    contextViewMock.mockResolvedValue(noChange);
    renderT(<ContextView running={false} />);
    expect(await screen.findByText("与上一步相同")).toBeTruthy();
  });

  it("事件筛选：dsh 多选语义（全亮 → 单类 → 再点恢复全选）", async () => {
    const { ContextView } = await import("./ContextView");
    renderT(<ContextView running={false} />);
    await screen.findByText("上下文事件");
    // 初始五类全亮：注入与压缩事件都可见
    expect(await screen.findByText(/指令注入 · \.gaea/)).toBeTruthy();
    expect(screen.getByText(/-535\.5k/)).toBeTruthy();
    // 全亮态点「压缩」→ 只留压缩（注入事件被筛掉）
    fireEvent.click(screen.getByRole("button", { name: "压缩" }));
    expect(screen.queryByText(/指令注入 · \.gaea/)).toBeNull();
    expect(screen.getByText(/-535\.5k/)).toBeTruthy();
    // 单选态再点「压缩」→ 恢复全选（注入事件回来）
    fireEvent.click(screen.getByRole("button", { name: "压缩" }));
    expect(await screen.findByText(/指令注入 · \.gaea/)).toBeTruthy();
  });

  it("构成卡：图例 hover 联动——该分类分段保持、其余分段淡出", async () => {
    const { ContextView } = await import("./ContextView");
    const { container } = renderT(<ContextView running={false} />);
    await screen.findByText("当前上下文");
    const chip = container.querySelector('[data-testid="comp-chip-user"]');
    const segUser = container.querySelector('[data-testid="comp-seg-user"]');
    const segTool = container.querySelector('[data-testid="comp-seg-tool"]');
    if (!chip || !segUser || !segTool) throw new Error("comp segments/chips missing");
    fireEvent.mouseEnter(chip);
    expect((segUser as HTMLElement).style.opacity).toBe("1");
    expect((segTool as HTMLElement).style.opacity).toBe("0.28");
    fireEvent.mouseLeave(chip);
    expect((segTool as HTMLElement).style.opacity).toBe("1");
  });

  it("悬停柱显示该步六分类构成详情条", async () => {
    const { ContextView } = await import("./ContextView");
    const { container } = renderT(<ContextView running={false} />);
    await screen.findByText("上下文趋势");
    const rect = container.querySelector("svg rect");
    if (!rect) throw new Error("trend bar rect not found");
    fireEvent.mouseEnter(rect);
    const detail = await screen.findByTestId("trend-hover-detail");
    expect(detail.textContent).toContain("第1轮·第1步");
    expect(detail.textContent).toContain("合计 42.7k");
    expect(detail.textContent).toContain("系统提示词 2.1k");
  });

  it("上下文浏览器：分类折叠组（项数）+ 展开节点 + 归档组", async () => {
    const { ContextView } = await import("./ContextView");
    renderT(<ContextView running={false} />);
    expect(await screen.findByText("上下文浏览器")).toBeTruthy();
    // 分类折叠行（button）与总览图例（span）同文 → 用 role 收紧定位
    fireEvent.click(screen.getByRole("button", { name: /系统提示词/ }));
    expect(await screen.findByText(/你是 gaea/)).toBeTruthy();
    // 工具结果分类：长文本展开/收起
    fireEvent.click(screen.getByRole("button", { name: /工具结果/ }));
    fireEvent.click(screen.getByText("展开"));
    expect(screen.getByText("收起")).toBeTruthy();
    expect(screen.getByText(/全文预览在这里展示/)).toBeTruthy();
    fireEvent.click(screen.getByText("收起"));
    expect(screen.queryByText("收起")).toBeNull();
    // 归档折叠组
    fireEvent.click(screen.getByText(/归档 1/));
    expect(screen.getByText(/旧的一轮用户输入内容/)).toBeTruthy();
    expect(screen.getAllByText("已压缩").length).toBeGreaterThan(0);
  });

  it("v4.80 深读：工具行来源 chip/error 点 + 完整调用懒加载（Raw/渲染切换）", async () => {
    const withTools: ContextTimeline = {
      ...TIMELINE,
      nodes: [
        { seq: 1, cat: "system", tokens: 2100, text: "你是 gaea……" },
        { seq: 4, cat: "tool", tokens: 800, tool: "read_file", text: "package main 输出预览…" },
        { seq: 5, cat: "tool", tokens: 120, tool: "bash", err: true, text: "[error] denied by policy" },
      ],
      archive: [],
    };
    contextViewMock.mockResolvedValue(withTools);
    const { ContextView } = await import("./ContextView");
    renderT(<ContextView running={false} />);
    expect(await screen.findByText("上下文浏览器")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: /工具结果/ }));
    // 来源 chip（工具名）与 error 语义点
    expect(await screen.findByText("read_file")).toBeTruthy();
    expect(screen.getByText("bash")).toBeTruthy();
    expect(screen.getByText("✗ error")).toBeTruthy();
    // 完整调用懒加载：点开 seq=4 行 → 详情面板（mock 返回样例）
    const detailBtns = screen.getAllByTestId("ctx-node-detail-btn");
    fireEvent.click(detailBtns[0]);
    const panel = await screen.findByTestId("ctx-node-detail");
    expect(panel.textContent).toContain("OK");
    expect(panel.textContent).toContain("5 行");
    expect(panel.textContent).toContain("loadFromDisk");
    // Raw → 渲染切换（Markdown 渲染出代码块）
    fireEvent.click(screen.getByText("渲染"));
    expect(panel.querySelector("pre, code")).toBeTruthy();
  });

  it("v4.80 分类内排序：大小序把大节点排前", async () => {
    const sortedNodes: ContextTimeline = {
      ...TIMELINE,
      nodes: [
        { seq: 1, cat: "tool", tokens: 100, tool: "small", text: "小" },
        { seq: 2, cat: "tool", tokens: 9000, tool: "big", text: "大" },
      ],
      archive: [],
    };
    contextViewMock.mockResolvedValue(sortedNodes);
    const { ContextView } = await import("./ContextView");
    renderT(<ContextView running={false} />);
    expect(await screen.findByText("上下文浏览器")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: /工具结果/ }));
    const list = screen.getByRole("button", { name: /工具结果/ }).parentElement!;
    expect(list.textContent?.indexOf("small")).toBeLessThan(list.textContent?.indexOf("big") ?? -1); // 默认时间序
    fireEvent.click(screen.getByText("大小序"));
    expect(list.textContent?.indexOf("big")).toBeLessThan(list.textContent?.indexOf("small") ?? -1);
  });

  it("v4.81 文件活动行级增量：±徽标 + 操作日志展开 + 详情跳转", async () => {
    const withFileDelta: ContextTimeline = {
      ...TIMELINE,
      files: [
        { seq: 21, ts: 1750000021, turn: 1, step: 1, tool: "write_file", action: "write", path: "docs/结论.md", added: 12 },
        { seq: 22, ts: 1750000022, turn: 1, step: 2, tool: "edit_file", action: "write", path: "docs/结论.md", added: 3, removed: 5 },
        { seq: 23, ts: 1750000023, turn: 1, step: 3, tool: "grep", action: "read", path: "internal", hits: 7 },
      ],
    };
    contextViewMock.mockResolvedValue(withFileDelta);
    // 详情 mock 按 seq 回带（校验跳转落到对应操作）
    contextNodeDetailMock.mockImplementation(async (seq: number) => ({
      seq,
      kind: "tool_result" as const,
      ts: 1750000000 + seq,
      tool: "grep",
      args: '{"path":"internal"}',
      output: "hit1\nhit2",
      lines: 2,
    }));
    const { ContextView } = await import("./ContextView");
    renderT(<ContextView running={false} />);
    expect(await screen.findByText("上下文浏览器")).toBeTruthy();
    // 文件行聚合 ±徽标（两次写合并：+15 = 12+3，−5）
    expect(screen.getByText(/docs\/结论\.md/)).toBeTruthy();
    expect(screen.getByText("+15")).toBeTruthy();
    expect(screen.getByText("−5")).toBeTruthy();
    // 展开操作日志：逐次操作行（含该次 ±12/−0）+ 详情懒加载（seq=23 grep → mock 详情）
    fireEvent.click(screen.getAllByTestId("file-ops-toggle")[0]);
    const opBtns = await screen.findAllByTestId("file-op-detail-btn");
    expect(opBtns.length).toBe(2);
    expect(screen.getByText("+12")).toBeTruthy();
    fireEvent.click(opBtns[0]);
    const panel = await screen.findByTestId("ctx-node-detail");
    expect(panel.textContent).toContain("grep");
    expect(contextNodeDetailMock).toHaveBeenCalledWith(21); // 跳转落到对应操作 seq
  });

  it("文件活动：按文件聚合 + 读写徽标 + 点击预览", async () => {
    const { ContextView } = await import("./ContextView");
    const { usePreviewStore } = await import("../lib/store");
    renderT(<ContextView running={false} />);
    expect(await screen.findByText("文件活动")).toBeTruthy();
    expect(screen.getByText("2 个文件")).toBeTruthy();
    expect(screen.getByText("internal/gaea/config/config.go")).toBeTruthy();
    fireEvent.click(screen.getByText("internal/gaea/config/config.go"));
    expect(usePreviewStore.getState().previewFile).toBe("internal/gaea/config/config.go");
  });

  it("Agent 网络：径向图渲染 + 订阅共享 store", async () => {
    const { ContextView } = await import("./ContextView");
    renderT(<ContextView running={false} />);
    expect(await screen.findByText("Agent 网络")).toBeTruthy();
    expect(screen.getByText(/1 个 Agent/)).toBeTruthy();
    expect(subscribeAgentNetworkMock).toHaveBeenCalled();
  });

  it("增量模式：点击「增量」展示净增减图例", async () => {
    const { ContextView } = await import("./ContextView");
    renderT(<ContextView running={false} />);
    await screen.findByText("上下文趋势");
    fireEvent.click(screen.getByText("增量"));
    expect(await screen.findByText(/净增/)).toBeTruthy();
    expect(screen.getByText(/净减/)).toBeTruthy();
  });

  it("底部会话汇总条与估算口径页脚", async () => {
    const { ContextView } = await import("./ContextView");
    renderT(<ContextView running={false} />);
    expect(await screen.findByText(/1 次请求/)).toBeTruthy();
    expect(screen.getByText(/累计费用/)).toBeTruthy();
    expect(screen.getByText(/估算口径/)).toBeTruthy();
  });

  it("加载失败显示错误", async () => {
    contextViewMock.mockRejectedValue(new Error("boom"));
    const { ContextView } = await import("./ContextView");
    renderT(<ContextView running={false} />);
    expect(await screen.findByText(/上下文视图加载失败/)).toBeTruthy();
  });
});

describe("ContextView 2.5d：趋势 brief 跳转浏览器 + 偏好持久化", () => {
  const PREF_KEY = "gaea.context.prefs";
  const ANCHORED: ContextTimeline = {
    ...TIMELINE,
    requests: [
      {
        ...TIMELINE.requests[0],
        briefUser: "帮我梳理 config 装载链路",
        briefUserSeq: 91,
        briefResp: "read_file {\"path\":\"config.go\"}",
        briefRespSeq: 4,
      },
    ],
    nodes: [
      { seq: 4, cat: "tool", tokens: 8000, text: "package main 工具输出内容样例" },
      { seq: 91, cat: "user", tokens: 20, text: "帮我梳理 config 装载链路" },
    ],
    archive: [],
  };

  beforeEach(() => {
    localStorage.removeItem(PREF_KEY);
    contextViewMock.mockReset();
    contextViewMock.mockResolvedValue(ANCHORED);
    contextNodeDetailMock.mockReset();
    contextNodeDetailMock.mockResolvedValue({ seq: 4, kind: "tool_result", ts: 1750000004, tool: "read_file", output: "body", lines: 1 });
    // jsdom 未实现 scrollIntoView（2.5d 跳转滚动）
    Element.prototype.scrollIntoView = vi.fn();
  });

  it("brief 行带锚点时渲染为跳转按钮；点击展开对应组并高亮目标节点", async () => {
    const { ContextView } = await import("./ContextView");
    renderT(<ContextView running={false} />);
    const userBtn = await screen.findByTestId("ctx-brief-user");
    fireEvent.click(userBtn);
    // user 组展开且目标节点带高亮 outline
    const node = await screen.findByTestId("ctx-browser-node-91");
    expect(node.className).toContain("outline");
    // 组行 aria-expanded=true
    const groupBtn = screen.getByRole("button", { name: /用户/ });
    expect(groupBtn.getAttribute("aria-expanded")).toBe("true");
    // 再点「回复」行：锚到 tool 节点
    fireEvent.click(screen.getByTestId("ctx-brief-resp"));
    const toolNode = await screen.findByTestId("ctx-browser-node-4");
    expect(toolNode.className).toContain("outline");
  });

  it("锚点无对应节点时诚实不跳（不渲染按钮或点击无副作用）", async () => {
    contextViewMock.mockResolvedValue({
      ...ANCHORED,
      requests: [{ ...ANCHORED.requests[0], briefUserSeq: 999 }],
    });
    const { ContextView } = await import("./ContextView");
    renderT(<ContextView running={false} />);
    const btn = await screen.findByTestId("ctx-brief-user");
    fireEvent.click(btn);
    // 目标组未展开：浏览器内没有任何带高亮的节点
    expect(document.querySelector("[data-node-seq='999']")).toBeNull();
  });

  it("无锚点的 brief 行保持纯文本（旧数据兼容）", async () => {
    contextViewMock.mockResolvedValue(TIMELINE);
    const { ContextView } = await import("./ContextView");
    renderT(<ContextView running={false} />);
    await screen.findByText("grep internal/gaea/config");
    expect(screen.queryByTestId("ctx-brief-user")).toBeNull();
  });

  it("趋势粒度/模式切换写入偏好存储", async () => {
    const { ContextView } = await import("./ContextView");
    renderT(<ContextView running={false} />);
    await screen.findByTestId("ctx-brief-user");
    fireEvent.click(screen.getByRole("button", { name: "轮次" }));
    fireEvent.click(screen.getByRole("button", { name: "增量" }));
    const prefs = JSON.parse(localStorage.getItem(PREF_KEY) || "{}");
    expect(prefs.trendGranularity).toBe("turn");
    expect(prefs.trendMode).toBe("delta");
  });

  it("浏览器分类内排序切换写入偏好存储", async () => {
    const { ContextView } = await import("./ContextView");
    renderT(<ContextView running={false} />);
    await screen.findByTestId("ctx-brief-user");
    fireEvent.click(screen.getByRole("button", { name: "大小序" }));
    const prefs = JSON.parse(localStorage.getItem(PREF_KEY) || "{}");
    expect(prefs.browserSort).toBe("size");
  });
});

describe("ContextView 2.5b 后半：工具结果图片缩略卡 + token 估算", () => {
  const DETAIL_SEQ = 4;

  beforeEach(() => {
    localStorage.removeItem("gaea.context.prefs");
    contextViewMock.mockReset();
    contextViewMock.mockResolvedValue(TIMELINE);
    contextNodeDetailMock.mockReset();
    contextNodeDetailMock.mockResolvedValue({
      seq: DETAIL_SEQ,
      kind: "tool_result",
      ts: 1750000004,
      tool: "vision",
      args: '{"image_path":"C:/demo/报表截图.png"}',
      output: "图中是一张季度报表截图",
      lines: 1,
      imageRefs: ["C:/demo/报表截图.png", "C:/demo/缺失图.png"],
      images: [
        {
          ref: "C:/demo/报表截图.png",
          path: "C:/demo/报表截图.png",
          exists: true,
          width: 1000,
          height: 1000,
          scaledW: 1000,
          scaledH: 1000,
          stdTokens: 1296,
          highTokens: 1296,
        },
        { ref: "C:/demo/缺失图.png", path: "C:/demo/缺失图.png", exists: false },
      ],
    });
    Element.prototype.scrollIntoView = vi.fn();
  });

  it("详情面板渲染图片缩略卡：尺寸→缩放尺寸 + 官方口径 token 成对显示", async () => {
    const { ContextView } = await import("./ContextView");
    renderT(<ContextView running={false} />);
    await screen.findByText("上下文浏览器");
    // 浏览器默认全收起：先展开「工具结果」组
    fireEvent.click(screen.getByRole("button", { name: /工具结果/ }));
    const detailBtns = await screen.findAllByTestId("ctx-node-detail-btn");
    fireEvent.click(detailBtns[0]);
    const images = await screen.findByTestId("ctx-node-images");
    expect(images).toBeTruthy();
    const cards = images.querySelectorAll("[data-testid='ctx-img-card']");
    expect(cards.length).toBe(2);
    // 官方例：1000×1000 → 1296 tok（标准档）
    expect(screen.getByText("≈1296 tok · 标准档")).toBeTruthy();
    expect(screen.getByText("1000×1000 → 1000×1000")).toBeTruthy();
    // 缺失文件灰态
    expect(screen.getByText("文件不存在")).toBeTruthy();
  });

  it("尺寸未知的图片（svg/ico）不显示 token 行，只显示未知标注", async () => {
    contextNodeDetailMock.mockResolvedValue({
      seq: DETAIL_SEQ,
      kind: "tool_result",
      tool: "vision",
      images: [{ ref: "C:/demo/logo.svg", path: "C:/demo/logo.svg", exists: true }],
    });
    const { ContextView } = await import("./ContextView");
    renderT(<ContextView running={false} />);
    await screen.findByText("上下文浏览器");
    fireEvent.click(screen.getByRole("button", { name: /工具结果/ }));
    const detailBtns = await screen.findAllByTestId("ctx-node-detail-btn");
    fireEvent.click(detailBtns[0]);
    expect(await screen.findByTestId("ctx-node-images")).toBeTruthy();
    expect(screen.getByText("尺寸未知")).toBeTruthy();
    expect(screen.queryByText(/tok · 标准档/)).toBeNull();
  });
});
