import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, act } from "@testing-library/react";
import type { ReactElement } from "react";
import { TasksWorkbench } from "./TasksWorkbench";
import { LocaleProvider } from "../lib/i18n";
import type { AgentNetwork, SubagentRunView } from "../lib/types";
import type { AgentNetworkMeta } from "../lib/agentNetworkStore";
import type { SubagentRunsMeta } from "../lib/subagentRunsStore";

// TasksWorkbench 迁共享 store（v4.66）后的接线测试：组件零自管定时器，双源
// （GaeaAgentNetwork + GaeaSubagentRuns）数据/状态全部来自 store 广播。这里
// mock 两个 store 与 bridge，断言「订阅/重订阅/reload 接线 + 渲染语义」；
// store 内部的单定时器、单在途、hidden 门控由各自的 store 测试钉住
// （subagentRunsStore.test.ts / agentNetworkStore.test.ts），此处不重复。

const renderT = (ui: ReactElement) => {
  localStorage.setItem("gaea-lang", "zh");
  return render(<LocaleProvider>{ui}</LocaleProvider>);
};

// ── mock：双 store + bridge（useLiveReload 的 onEvent）+ TaskCenter（自管理，不在本测试范围） ──
const storeMocks = vi.hoisted(() => ({
  subscribeSubagentRuns: vi.fn(),
  reloadSubagentRuns: vi.fn(),
  subscribeAgentNetwork: vi.fn(),
  reloadAgentNetwork: vi.fn(),
}));
const bridgeMocks = vi.hoisted(() => ({
  onEvent: vi.fn(() => () => {}),
}));

vi.mock("../lib/bridge", () => ({ onEvent: bridgeMocks.onEvent }));
vi.mock("../lib/agentNetworkStore", () => ({
  subscribeAgentNetwork: storeMocks.subscribeAgentNetwork,
  reloadAgentNetwork: storeMocks.reloadAgentNetwork,
}));
vi.mock("../lib/subagentRunsStore", () => ({
  subscribeSubagentRuns: storeMocks.subscribeSubagentRuns,
  reloadSubagentRuns: storeMocks.reloadSubagentRuns,
}));
vi.mock("./TaskCenter", () => ({ TaskCenter: () => null }));

// ── 受控的 fake store：测试里手工驱动 cb 广播（对齐真实 store 的 cb(runs, meta) 契约） ──
type RunsCb = (runs: SubagentRunView[], meta: SubagentRunsMeta) => void;
type NetCb = (net: AgentNetwork | null, meta: AgentNetworkMeta) => void;
let runsCb: RunsCb | null = null;
let netCb: NetCb | null = null;
let unsubRuns = vi.fn();
let unsubNet = vi.fn();

function emitRuns(runs: SubagentRunView[], meta: SubagentRunsMeta): void {
  act(() => runsCb?.(runs, meta));
}
function emitNet(net: AgentNetwork | null, meta: AgentNetworkMeta): void {
  act(() => netCb?.(net, meta));
}

// ── fixtures（字段形状对齐 SubagentsPanel.test.tsx） ──
const netA: AgentNetwork = {
  ok: true,
  window: 0,
  root: {
    id: "root",
    name: "主 agent",
    kind: "root",
    status: "running",
    toolCalls: 3,
    errors: 0,
    tokens: 420000,
    firstTs: 1750000000,
    lastTs: 1750000200,
    children: [
      {
        id: "sa_1_a1a1a1a1",
        name: "task",
        kind: "subagent",
        status: "completed",
        task: "收集 2026 年办公 Agent 竞品更新信息",
        model: "deepseek-v4-flash",
        toolCalls: 3,
        errors: 0,
        tokens: 180000,
        firstTs: 1750000100,
        lastTs: 1750000150,
      },
    ],
  },
};

const runCompleted: SubagentRunView = {
  ref: "sa_1_a1a1a1a1",
  status: "completed",
  task: "收集 2026 年办公 Agent 竞品更新信息",
  model: "deepseek-v4-flash",
  toolCalls: 3,
  createdAt: "2026-08-17T10:00:00+08:00",
  updatedAt: "2026-08-17T10:30:00+08:00",
};

// 本地模型工具（mt_ 单轮调用）：走 ①b 扁平区块，不在树里。
const runModelTool: SubagentRunView = {
  ref: "mt_9_deadbeef",
  status: "running",
  kind: "model_tool",
  tool: "vision",
  task: "总结界面截图",
  toolCalls: 1,
  createdAt: "2026-08-17T11:00:00+08:00",
  updatedAt: "2026-08-17T11:01:00+08:00",
};

const runNew: SubagentRunView = {
  ref: "sa_2_b2b2b2b2",
  status: "running",
  task: "新增并行任务：整理竞品报价单",
  toolCalls: 0,
  createdAt: "2026-08-17T12:00:00+08:00",
  updatedAt: "2026-08-17T12:00:30+08:00",
};

const runsA: SubagentRunView[] = [runCompleted, runModelTool];

const _metaLoading: SubagentRunsMeta = { status: "loading", total: 0, running: 0 };
const metaReady = (running: number): SubagentRunsMeta => ({ status: "ready", available: true, total: 2, running });
const metaError = (running: number): SubagentRunsMeta => ({ status: "error", total: 2, running });
const _netLoadingM: AgentNetworkMeta = { status: "loading" };
const netReadyM: AgentNetworkMeta = { status: "ready" };
const netErrorM: AgentNetworkMeta = { status: "error" };

beforeEach(() => {
  vi.clearAllMocks();
  runsCb = null;
  netCb = null;
  unsubRuns = vi.fn();
  unsubNet = vi.fn();
  storeMocks.subscribeSubagentRuns.mockImplementation((_path: string, cb: RunsCb) => {
    runsCb = cb;
    return unsubRuns;
  });
  storeMocks.subscribeAgentNetwork.mockImplementation((cb: NetCb) => {
    netCb = cb;
    return unsubNet;
  });
  window.localStorage.removeItem("gaea.subagentAutoOpen");
});

describe("TasksWorkbench 任务视图（双源入共享 store）", () => {
  it("双源渲染：树拓扑 + 本地模型工具区块 + 运行计数徽标；接线为 store 订阅", () => {
    renderT(<TasksWorkbench sessionPath="s1.jsonl" />);
    // 订阅接线：runs 按路径建册、net 单例；轮询/hidden 门控由 store 承担
    expect(storeMocks.subscribeSubagentRuns).toHaveBeenCalledWith("s1.jsonl", expect.any(Function));
    expect(storeMocks.subscribeAgentNetwork).toHaveBeenCalledTimes(1);
    emitNet(netA, netReadyM);
    emitRuns(runsA, metaReady(1));
    expect(screen.getByText("任务管理")).toBeTruthy();
    expect(screen.getByText("主 agent")).toBeTruthy();
    expect(screen.getByText("收集 2026 年办公 Agent 竞品更新信息")).toBeTruthy();
    // 本地模型工具区块（同一 runs 数据源的 kind=model_tool 子集）
    expect(screen.getByText("本地模型工具")).toBeTruthy();
    expect(document.querySelector('[data-model-tool-row="s1.jsonl:mt_9_deadbeef"]')).toBeTruthy();
    // 计数徽标：meta.running=1
    // 分组头与面板头都可能显示运行计数 → 多命中用 getAllBy
    expect(screen.getAllByText("1 运行中").length).toBeGreaterThan(0);
  });

  it("v4.76：点击「子代理」分组头可整体折叠/展开整棵子代理树", () => {
    renderT(<TasksWorkbench sessionPath="s1.jsonl" />);
    emitNet(netA, netReadyM);
    emitRuns(runsA, metaReady(1));
    const toggle = screen.getByTestId("subagent-section-toggle");
    expect(toggle.getAttribute("aria-expanded")).toBe("true");
    expect(screen.getByText("主 agent")).toBeTruthy();
    fireEvent.click(toggle);
    expect(toggle.getAttribute("aria-expanded")).toBe("false");
    expect(screen.queryByText("主 agent")).toBeNull();
    expect(screen.queryByText("收集 2026 年办公 Agent 竞品更新信息")).toBeNull();
    fireEvent.click(toggle);
    expect(screen.getByText("主 agent")).toBeTruthy();
  });

  it("加载态：首拉在途显示「读取子代理分工…」+ 刷新钮旋转，ready 后消失", () => {
    renderT(<TasksWorkbench sessionPath="s1.jsonl" />);
    expect(screen.getByText("读取子代理分工…")).toBeTruthy();
    const refreshBtn = screen.getByRole("button", { name: "刷新" });
    expect(refreshBtn.querySelector(".animate-spin")).toBeTruthy();
    emitNet(netA, netReadyM);
    emitRuns(runsA, metaReady(1));
    expect(screen.queryByText("读取子代理分工…")).toBeNull();
    expect(refreshBtn.querySelector(".animate-spin")).toBeNull();
  });

  it("失败空态：错误文案 + 重试按钮（不用「暂无」冒充失败）；重试走 runs reload（runs 错误优先）", () => {
    renderT(<TasksWorkbench sessionPath="s1.jsonl" />);
    emitNet(null, netErrorM);
    emitRuns([], metaError(0));
    expect(screen.getByText("子代理列表加载失败，点击重试")).toBeTruthy();
    fireEvent.click(screen.getByTestId("tasks-workbench-retry"));
    expect(storeMocks.reloadSubagentRuns).toHaveBeenCalledWith("s1.jsonl");
    expect(storeMocks.reloadAgentNetwork).not.toHaveBeenCalled();
  });

  it("失败空态（仅树失败）：重试走 reloadAgentNetwork", () => {
    renderT(<TasksWorkbench sessionPath="s1.jsonl" />);
    emitNet(null, netErrorM);
    emitRuns([], { status: "ready", available: false, total: 0, running: 0 });
    fireEvent.click(screen.getByTestId("tasks-workbench-retry"));
    expect(storeMocks.reloadAgentNetwork).toHaveBeenCalledTimes(1);
    expect(storeMocks.reloadSubagentRuns).not.toHaveBeenCalled();
  });

  it("失败有内容：细条横幅 + 重试，既有内容保持；恢复 ready 后横幅消失", () => {
    renderT(<TasksWorkbench sessionPath="s1.jsonl" />);
    emitNet(netA, netReadyM);
    emitRuns(runsA, metaReady(1));
    expect(screen.queryByText("子代理列表加载失败，点击重试")).toBeNull();
    // 拉取失败：store 保留上一份快照（同快照 + error meta），横幅出现、内容不动
    emitRuns(runsA, metaError(1));
    expect(screen.getByText("子代理列表加载失败，点击重试")).toBeTruthy();
    expect(screen.getByText("主 agent")).toBeTruthy();
    fireEvent.click(screen.getByTestId("tasks-workbench-retry"));
    expect(storeMocks.reloadSubagentRuns).toHaveBeenCalledWith("s1.jsonl");
    emitRuns(runsA, metaReady(1));
    expect(screen.queryByText("子代理列表加载失败，点击重试")).toBeNull();
  });

  it("新子代理检测：首拉只记基线不触发；基线上新增 ref 触发一次；重复快照不重复触发", () => {
    const onStarted = vi.fn();
    renderT(<TasksWorkbench sessionPath="s1.jsonl" onSubagentStarted={onStarted} />);
    emitRuns([runCompleted], metaReady(0));
    expect(onStarted).not.toHaveBeenCalled();
    emitRuns([runCompleted, runNew], metaReady(1));
    expect(onStarted).toHaveBeenCalledTimes(1);
    emitRuns([runCompleted, runNew], metaReady(1));
    expect(onStarted).toHaveBeenCalledTimes(1);
  });

  it("偏好关闭（gaea.subagentAutoOpen=0）：新子代理只更新数据，不触发 onSubagentStarted", () => {
    window.localStorage.setItem("gaea.subagentAutoOpen", "0");
    const onStarted = vi.fn();
    renderT(<TasksWorkbench sessionPath="s1.jsonl" onSubagentStarted={onStarted} />);
    emitRuns([runCompleted], metaReady(0));
    emitRuns([runCompleted, runNew], metaReady(1));
    expect(onStarted).not.toHaveBeenCalled();
    window.localStorage.removeItem("gaea.subagentAutoOpen");
  });

  it("无 sessionPath：不订阅 store、不 reload，显示空状态", () => {
    renderT(<TasksWorkbench />);
    // 「暂无子代理」与提示语同 span（<br/> 分隔）→ 子串匹配
    expect(screen.getByText(/暂无子代理/)).toBeTruthy();
    expect(storeMocks.subscribeSubagentRuns).not.toHaveBeenCalled();
    expect(storeMocks.subscribeAgentNetwork).not.toHaveBeenCalled();
    expect(storeMocks.reloadAgentNetwork).not.toHaveBeenCalled();
    expect(storeMocks.reloadSubagentRuns).not.toHaveBeenCalled();
  });

  it("会话切换：runs 重订阅新路径（旧订阅注销），net 显式 reload 补即时性", () => {
    const { rerender } = renderT(<TasksWorkbench sessionPath="s1.jsonl" />);
    const firstRunsUnsub = unsubRuns;
    rerender(<LocaleProvider><TasksWorkbench sessionPath="s2.jsonl" /></LocaleProvider>);
    expect(storeMocks.subscribeSubagentRuns).toHaveBeenLastCalledWith("s2.jsonl", expect.any(Function));
    expect(firstRunsUnsub).toHaveBeenCalledTimes(1);
    // net 单例不随路径重建 → 会话切换显式 reload（SubagentsPanel 同款接线）
    expect(storeMocks.reloadAgentNetwork).toHaveBeenCalledTimes(1);
  });

  it("卸载：注销双 store 订阅（停表由 store 的最后订阅者语义保证）", () => {
    const { unmount } = renderT(<TasksWorkbench sessionPath="s1.jsonl" />);
    expect(unsubRuns).not.toHaveBeenCalled();
    expect(unsubNet).not.toHaveBeenCalled();
    unmount();
    expect(unsubRuns).toHaveBeenCalledTimes(1);
    expect(unsubNet).toHaveBeenCalledTimes(1);
  });

  it("节点点击 → onOpenSubagent 携带会话/ref/状态（openThread 语义保持）", () => {
    const onOpen = vi.fn();
    renderT(<TasksWorkbench sessionPath="s1.jsonl" onOpenSubagent={onOpen} />);
    emitNet(netA, netReadyM);
    emitRuns(runsA, metaReady(0));
    // 子代理节点：ref 直等命中 run → 状态 completed、模型跟随
    fireEvent.click(screen.getByText("收集 2026 年办公 Agent 竞品更新信息"));
    expect(onOpen).toHaveBeenCalledWith({
      sessionPath: "s1.jsonl",
      ref: "sa_1_a1a1a1a1",
      task: "收集 2026 年办公 Agent 竞品更新信息",
      model: "deepseek-v4-flash",
      status: "completed",
    });
    // 本地模型工具行：同一 onOpenSubagent 出口
    const row = document.querySelector('[data-model-tool-row="s1.jsonl:mt_9_deadbeef"]');
    expect(row).toBeTruthy();
    fireEvent.click(row as Element);
    expect(onOpen).toHaveBeenLastCalledWith({
      sessionPath: "s1.jsonl",
      ref: "mt_9_deadbeef",
      task: "总结界面截图",
      status: "running",
    });
  });
});
