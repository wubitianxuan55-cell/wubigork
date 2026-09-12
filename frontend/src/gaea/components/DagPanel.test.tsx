import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactElement } from "react";
import { DagPanel } from "./DagPanel";
import { LocaleProvider } from "../lib/i18n";
import { usePreviewStore } from "../lib/store";
import type { DagRunView, DagTemplateView } from "../lib/types";

// DagPanel 走查测试：桥接 mock 的工厂必须给齐 ../lib/bridge 的全部 named exports
// （仓库已知坑：工厂缺导出不报错，但 lib/store/controller 对 app/onEvent/onReady
// 的真引用会炸）；预览通道用真 usePreviewStore（与 DeliverablesPanel 同口径，
// 断言产物 chip 点击后 previewFile 进入全局预览队列）。
const bridge = vi.hoisted(() => ({
  dagList: vi.fn(),
  dagGet: vi.fn(),
  dagRun: vi.fn(),
  dagNodeRun: vi.fn(),
  dagNodeSteer: vi.fn(),
  dagNodeAccept: vi.fn(),
  dagAcceptAll: vi.fn(),
  dagCancel: vi.fn(),
  dagTemplateList: vi.fn(),
  dagTemplateSave: vi.fn(),
  dagTemplateNew: vi.fn(),
  dagTemplateDelete: vi.fn(),
}));

vi.mock("../lib/bridge", () => ({
  app: {
    DagList: bridge.dagList,
    DagGet: bridge.dagGet,
    DagRun: bridge.dagRun,
    DagNodeRun: bridge.dagNodeRun,
    DagNodeSteer: bridge.dagNodeSteer,
    DagNodeAccept: bridge.dagNodeAccept,
    DagAcceptAll: bridge.dagAcceptAll,
    DagCancel: bridge.dagCancel,
    DagTemplateList: bridge.dagTemplateList,
    DagTemplateSave: bridge.dagTemplateSave,
    DagTemplateNew: bridge.dagTemplateNew,
    DagTemplateDelete: bridge.dagTemplateDelete,
  },
  onEvent: vi.fn(() => () => {}),
  onSubagentText: vi.fn(() => () => {}),
  onUpdaterProgress: vi.fn(() => () => {}),
  onTaskEvent: vi.fn(() => () => {}),
  onReady: vi.fn(() => () => {}),
  BridgeError: class BridgeError extends Error {
    code: string;
    constructor(code: string, message: string) {
      super(message);
      this.code = code;
    }
  },
  workApp: {},
  playApp: {},
  sharedApp: {},
  openExternal: vi.fn(),
  waitMockReady: vi.fn(async () => null),
  initBridge: vi.fn(async () => undefined),
}));

const renderT = (ui: ReactElement) => {
  localStorage.setItem("gaea-lang", "zh");
  return render(<LocaleProvider>{ui}</LocaleProvider>);
};

const runId = "dag_monthly_report";

// 混合态示例链（对应 mock 的月度报告流水线口径）：done / failed / pending / accepted
// 四种节点状态徽标 + failed 派生态（起跑/续跑按钮显隐）。
const RUN: DagRunView = {
  id: runId,
  goal: "读三份月度报表，出一份带图表的月度经营报告",
  createdAt: "2026-09-11T09:00:00+08:00",
  updatedAt: "2026-09-11T09:30:00+08:00",
  derived: "failed",
  nodes: [
    {
      id: "n1",
      title: "读三份月度 xlsx 报表",
      prompt: "读取三份报表并提取核心表。",
      status: "done",
      runCount: 1,
      outputs: ["docs/月度汇总.xlsx"],
    },
    {
      id: "n2",
      title: "透视汇总生成 summary.xlsx",
      prompt: "按月聚合营收/成本/利润。",
      status: "failed",
      runCount: 1,
      error: "第 3 行公式解析失败（mock）",
    },
    {
      id: "n3",
      title: "嵌图表出报告 docx",
      prompt: "生成柱状/饼图并撰写报告。",
      status: "pending",
      runCount: 0,
    },
    {
      id: "n4",
      title: "导出 PDF 归档",
      prompt: "无头转换为 PDF 归档。",
      status: "accepted",
      runCount: 1,
      acceptedAt: "2026-09-11T09:28:00+08:00",
    },
  ],
};

const RUNNING_RUN: DagRunView = {
  id: "dag_running",
  goal: "整理会议纪要并出行动项表（running 样例）",
  createdAt: "2026-09-11T10:00:00+08:00",
  updatedAt: "2026-09-11T10:01:00+08:00",
  derived: "running",
  nodes: [
    { id: "r1", title: "整理纪要", prompt: "p", status: "done", runCount: 1 },
    { id: "r2", title: "生成行动项表", prompt: "p", status: "running", runCount: 1 },
  ],
};

// 模板样例（6.3 余项：存模板一键重建）。
const TPL: DagTemplateView = {
  id: "dagtpl_monthly",
  name: "月度经营报告",
  goal: "读三份月度报表，出一份带图表的月度经营报告",
  createdAt: "2026-09-10T09:00:00+08:00",
  sourceRunId: "dag_src",
  nodes: [
    { id: "n1", title: "读三份月度 xlsx 报表", prompt: "p", status: "pending", runCount: 0 },
    { id: "n2", title: "透视汇总生成 summary.xlsx", prompt: "p", dependsOn: ["n1"], status: "pending", runCount: 0 },
  ],
};

beforeEach(() => {
  vi.clearAllMocks();
  bridge.dagList.mockResolvedValue([]);
  bridge.dagTemplateList.mockResolvedValue([]);
  bridge.dagRun.mockResolvedValue("受理（mock）");
  bridge.dagNodeRun.mockResolvedValue("受理（mock）");
  bridge.dagNodeSteer.mockResolvedValue("受理（mock）");
  bridge.dagNodeAccept.mockResolvedValue("受理（mock）");
  bridge.dagCancel.mockResolvedValue("受理（mock）");
  bridge.dagTemplateSave.mockResolvedValue("已存为模板（mock）");
  bridge.dagTemplateNew.mockResolvedValue("已重建（mock）");
  bridge.dagTemplateDelete.mockResolvedValue("已删除（mock）");
  usePreviewStore.setState({ previewFile: null, previewList: [], previewIndex: -1 });
});

afterEach(() => {
  vi.useRealTimers();
});

describe("DagPanel 办公流水线区块（6.3 办公多文件 DAG）", () => {
  it("空态：无流水线时展示引导文案（不走失败态）", async () => {
    bridge.dagList.mockResolvedValue([]);
    renderT(<DagPanel />);
    await waitFor(() => expect(bridge.dagList).toHaveBeenCalledTimes(1));
    expect(await screen.findByText(/暂无流水线/)).toBeTruthy();
    expect(screen.queryByTestId("dag-retry")).toBeNull();
  });

  it("首拉失败：错误文案 + 重试入口（不用「暂无」冒充失败），重试成功后恢复空态", async () => {
    bridge.dagList.mockRejectedValueOnce(new Error("boom"));
    renderT(<DagPanel />);
    expect(await screen.findByText(/流水线列表加载失败/)).toBeTruthy();
    bridge.dagList.mockResolvedValue([]);
    fireEvent.click(screen.getByTestId("dag-retry"));
    expect(await screen.findByText(/暂无流水线/)).toBeTruthy();
  });

  it("示例链渲染：run 卡 + 展开 4 节点状态徽标/产物 chip/失败文案；按钮按状态显隐", async () => {
    bridge.dagList.mockResolvedValue([RUN]);
    renderT(<DagPanel />);
    expect(await screen.findByTestId(`dag-run-${runId}`)).toBeTruthy();
    // 展开节点列表
    fireEvent.click(screen.getByText(RUN.goal));
    // 4 个节点状态点齐
    for (const n of RUN.nodes) {
      expect(screen.getByTestId(`dag-node-${runId}-${n.id}`)).toBeTruthy();
      expect(screen.getByTestId(`dag-node-status-${runId}-${n.id}`)).toBeTruthy();
    }
    // done 节点产物 chip 可见
    expect(screen.getByText("docs/月度汇总.xlsx")).toBeTruthy();
    // failed 节点错误文案（红字小号）
    expect(screen.getByText("第 3 行公式解析失败（mock）")).toBeTruthy();
    // done(n1)：重跑 + 验收 + steer 输入框
    expect(screen.getByTestId(`dag-node-rerun-${runId}-n1`)).toBeTruthy();
    expect(screen.getByTestId(`dag-node-accept-${runId}-n1`)).toBeTruthy();
    expect(screen.getByTestId(`dag-steer-input-${runId}-n1`)).toBeTruthy();
    // failed(n2)：验收 + steer 有，重跑无
    expect(screen.getByTestId(`dag-node-accept-${runId}-n2`)).toBeTruthy();
    expect(screen.getByTestId(`dag-steer-input-${runId}-n2`)).toBeTruthy();
    expect(screen.queryByTestId(`dag-node-rerun-${runId}-n2`)).toBeNull();
    // pending(n3)：只有单跑
    expect(screen.getByTestId(`dag-node-run-${runId}-n3`)).toBeTruthy();
    expect(screen.queryByTestId(`dag-node-accept-${runId}-n3`)).toBeNull();
    // accepted(n4)：重跑有、验收无（已验收不重复出验收口）
    expect(screen.getByTestId(`dag-node-rerun-${runId}-n4`)).toBeTruthy();
    expect(screen.queryByTestId(`dag-node-accept-${runId}-n4`)).toBeNull();
    // derived=failed → run 级「起跑/续跑」，无「终止」
    expect(screen.getByTestId(`dag-run-start-${runId}`)).toBeTruthy();
    expect(screen.queryByTestId(`dag-run-cancel-${runId}`)).toBeNull();
  });

  it("验收：点击节点验收按钮调 GaeaDagNodeAccept 并立刻重拉 GaeaDagList", async () => {
    bridge.dagList.mockResolvedValue([RUN]);
    renderT(<DagPanel />);
    await screen.findByTestId(`dag-run-${runId}`);
    fireEvent.click(screen.getByText(RUN.goal));
    fireEvent.click(screen.getByTestId(`dag-node-accept-${runId}-n1`));
    await waitFor(() => expect(bridge.dagNodeAccept).toHaveBeenCalledWith(runId, "n1"));
    await waitFor(() => expect(bridge.dagList).toHaveBeenCalledTimes(2));
  });

  it("起跑：run 级按钮调 GaeaDagRun（draft/failed/ready 显隐已在上例覆盖）", async () => {
    bridge.dagList.mockResolvedValue([RUN]);
    renderT(<DagPanel />);
    await screen.findByTestId(`dag-run-${runId}`);
    fireEvent.click(screen.getByTestId(`dag-run-start-${runId}`));
    await waitFor(() => expect(bridge.dagRun).toHaveBeenCalledWith(runId));
  });

  it("steer：输入补充指令点发送，调 GaeaDagNodeSteer 且输入框清空", async () => {
    bridge.dagList.mockResolvedValue([RUN]);
    renderT(<DagPanel />);
    await screen.findByTestId(`dag-run-${runId}`);
    fireEvent.click(screen.getByText(RUN.goal));
    const input = screen.getByTestId(`dag-steer-input-${runId}-n1`) as HTMLInputElement;
    fireEvent.change(input, { target: { value: "把柱状图改成饼图" } });
    fireEvent.click(screen.getByTestId(`dag-steer-send-${runId}-n1`));
    await waitFor(() =>
      expect(bridge.dagNodeSteer).toHaveBeenCalledWith(runId, "n1", "把柱状图改成饼图"),
    );
    await waitFor(() => expect((screen.getByTestId(`dag-steer-input-${runId}-n1`) as HTMLInputElement).value).toBe(""));
  });

  it("产物 chip 点击走全局预览通道（usePreviewStore.openFilePreview）", async () => {
    bridge.dagList.mockResolvedValue([RUN]);
    renderT(<DagPanel />);
    await screen.findByTestId(`dag-run-${runId}`);
    fireEvent.click(screen.getByText(RUN.goal));
    fireEvent.click(screen.getByText("docs/月度汇总.xlsx"));
    expect(usePreviewStore.getState().previewFile).toBe("docs/月度汇总.xlsx");
  });

  it("running 轮询：derived=running 时每 2.5s 重拉自校正，操作通道同源（终止）", async () => {
    vi.useFakeTimers();
    bridge.dagList.mockResolvedValue([RUNNING_RUN]);
    renderT(<DagPanel />);
    await act(async () => {
      await Promise.resolve();
    });
    expect(bridge.dagList).toHaveBeenCalledTimes(1);
    // running → 「终止」可见，「起跑/续跑」不出现
    expect(screen.getByTestId("dag-run-cancel-dag_running")).toBeTruthy();
    expect(screen.queryByTestId("dag-run-start-dag_running")).toBeNull();
    await act(async () => {
      vi.advanceTimersByTime(2_500);
    });
    expect(bridge.dagList).toHaveBeenCalledTimes(2);
    await act(async () => {
      vi.advanceTimersByTime(2_500);
    });
    expect(bridge.dagList).toHaveBeenCalledTimes(3);
    // 终止：调 GaeaDagCancel + 立刻重拉
    fireEvent.click(screen.getByTestId("dag-run-cancel-dag_running"));
    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });
    expect(bridge.dagCancel).toHaveBeenCalledWith("dag_running");
    expect(bridge.dagList.mock.calls.length).toBeGreaterThanOrEqual(4);
  });

  it("无 running 时停轮询：draft 列表只在挂载与操作后拉取", async () => {
    vi.useFakeTimers();
    bridge.dagList.mockResolvedValue([RUN]);
    renderT(<DagPanel />);
    await act(async () => {
      await Promise.resolve();
    });
    expect(bridge.dagList).toHaveBeenCalledTimes(1);
    await act(async () => {
      vi.advanceTimersByTime(7_500);
    });
    expect(bridge.dagList).toHaveBeenCalledTimes(1);
  });

  // ── 模板库（6.3 余项：存模板一键重建）───────────────────────────────
  it("模板区：无模板不显形；有模板折叠条默认收起，展开后逐条渲染+新建/删", async () => {
    bridge.dagList.mockResolvedValue([]);
    bridge.dagTemplateList.mockResolvedValue([TPL]);
    renderT(<DagPanel />);
    expect(await screen.findByTestId("dag-tpl-section")).toBeTruthy();
    // 默认收起：行不可见
    expect(screen.queryByTestId(`dag-tpl-row-${TPL.id}`)).toBeNull();
    fireEvent.click(screen.getByTestId("dag-tpl-toggle"));
    expect(await screen.findByTestId(`dag-tpl-row-${TPL.id}`)).toBeTruthy();
    expect(screen.getByText("月度经营报告")).toBeTruthy();
    expect(screen.getByText("2 节点")).toBeTruthy();
  });

  it("模板新建：点「新建」调 DagTemplateNew，run 列表随之重拉", async () => {
    bridge.dagList.mockResolvedValue([]);
    bridge.dagTemplateList.mockResolvedValue([TPL]);
    renderT(<DagPanel />);
    await screen.findByTestId("dag-tpl-section");
    fireEvent.click(screen.getByTestId("dag-tpl-toggle"));
    fireEvent.click(await screen.findByTestId(`dag-tpl-new-${TPL.id}`));
    await waitFor(() => expect(bridge.dagTemplateNew).toHaveBeenCalledWith(TPL.id));
    await waitFor(() => expect(bridge.dagList.mock.calls.length).toBeGreaterThanOrEqual(2));
  });

  it("存模板：run 卡「存模板」出内联输入（默认 goal 截断），提交调 DagTemplateSave", async () => {
    bridge.dagList.mockResolvedValue([RUN]);
    renderT(<DagPanel />);
    await screen.findByTestId(`dag-run-${runId}`);
    fireEvent.click(screen.getByTestId(`dag-run-save-tpl-${runId}`));
    const input = screen.getByTestId(`dag-tpl-input-${runId}`) as HTMLInputElement;
    // goal 共 21 字未超 24 截断阈值 → 原样带出
    expect(input.value).toBe("读三份月度报表，出一份带图表的月度经营报告");
    fireEvent.click(screen.getByTestId(`dag-tpl-save-${runId}`));
    await waitFor(() =>
      expect(bridge.dagTemplateSave).toHaveBeenCalledWith(runId, "读三份月度报表，出一份带图表的月度经营报告"),
    );
    // 提交后内联框收起
    await waitFor(() => expect(screen.queryByTestId(`dag-tpl-input-${runId}`)).toBeNull());
  });

  it("模板拉取失败静默：模板区不显形，不挡主列表空态", async () => {
    bridge.dagList.mockResolvedValue([]);
    bridge.dagTemplateList.mockRejectedValue(new Error("boom"));
    renderT(<DagPanel />);
    // 主列表空态正常（模板失败静默）
    expect(await screen.findByText(/暂无流水线/)).toBeTruthy();
    expect(screen.queryByTestId("dag-tpl-section")).toBeNull();
  });

  it("模板删除：点「删」调 DagTemplateDelete", async () => {
    bridge.dagList.mockResolvedValue([]);
    bridge.dagTemplateList.mockResolvedValue([TPL]);
    renderT(<DagPanel />);
    expect(await screen.findByTestId("dag-tpl-section")).toBeTruthy();
    fireEvent.click(screen.getByTestId("dag-tpl-toggle"));
    fireEvent.click(screen.getByTestId(`dag-tpl-del-${TPL.id}`));
    await waitFor(() => expect(bridge.dagTemplateDelete).toHaveBeenCalledWith(TPL.id));
    await waitFor(() => expect(bridge.dagTemplateList.mock.calls.length).toBeGreaterThanOrEqual(2));
  });

  it("成品直出：run 头显成品计数徽标（title=文件清单，done+accepted 口径）", async () => {
    bridge.dagList.mockResolvedValue([RUN]);
    bridge.dagTemplateList.mockResolvedValue([]);
    renderT(<DagPanel />);
    // n4（accepted）在夹具里无 outputs → 只有 done n1 的 1 件计入
    const chip = await screen.findByTitle("成品：docs/月度汇总.xlsx");
    expect(chip.textContent).toBe("1 件成品");
    // running run 无 done+accepted 产物链时不显徽标（RUNNING_RUN 有 done 无 outputs）
  });

  it("一键验收两段式：首击武装（文案变确认），再击调 GaeaDagAcceptAll 并重拉；running 不显按钮", async () => {
    bridge.dagList.mockResolvedValue([RUN, RUNNING_RUN]);
    bridge.dagTemplateList.mockResolvedValue([]);
    bridge.dagAcceptAll.mockResolvedValue("已一键验收 1 个节点、1 件产物回流记忆。");
    renderT(<DagPanel />);

    // RUN（derived=failed，1 个 done）：按钮显形，首击只武装
    const btn = await screen.findByTestId(`dag-run-acceptall-${runId}`);
    expect(btn.textContent).toBe("一键验收 1");
    fireEvent.click(btn);
    expect(bridge.dagAcceptAll).not.toHaveBeenCalled();
    expect(await screen.findByTestId(`dag-run-acceptall-${runId}`).then((b) => b.textContent)).toBe("确认验收 1 节点");

    // 再击执行并重拉
    fireEvent.click(screen.getByTestId(`dag-run-acceptall-${runId}`));
    await waitFor(() => expect(bridge.dagAcceptAll).toHaveBeenCalledWith(runId));
    await waitFor(() => expect(bridge.dagList.mock.calls.length).toBeGreaterThanOrEqual(2));

    // RUNNING_RUN（derived=running）：不显一键验收按钮
    expect(screen.queryByTestId("dag-run-acceptall-dag_running")).toBeNull();
  });
});
