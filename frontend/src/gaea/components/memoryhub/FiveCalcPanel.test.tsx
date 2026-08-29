import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { FiveCalcPanel } from "./FiveCalcPanel";
import { ToastProvider } from "../Toast";
import type { CostStageCompareRow, CostStageDeviation, CostStageValue } from "../../lib/types";

// ── bridge mock（对齐 CostCompareModal.test 的 spy 模式）──
const { stagesSpy, compareSpy, devsSpy, saveSpy } = vi.hoisted(() => ({
  stagesSpy: vi.fn(),
  compareSpy: vi.fn(),
  devsSpy: vi.fn(),
  saveSpy: vi.fn(),
}));

vi.mock("../../lib/bridge", () => ({
  app: {
    CostStages: (...args: unknown[]) => stagesSpy(...args),
    CostStageCompare: (...args: unknown[]) => compareSpy(...args),
    CostStageDeviations: (...args: unknown[]) => devsSpy(...args),
    CostStageSave: (...args: unknown[]) => saveSpy(...args),
  },
}));

const wrap = (node: React.ReactNode) => <ToastProvider>{node}</ToastProvider>;
const PROJECT_ID = "p1";

const STAGES_ONE: CostStageValue[] = [
  { id: 1, projectId: PROJECT_ID, stage: "估算", amount: 1000000, date: "2026-01-15", note: "", createdAt: "", updatedAt: "" },
];

const STAGES_TWO: CostStageValue[] = [
  ...STAGES_ONE,
  { id: 2, projectId: PROJECT_ID, stage: "概算", amount: 1180000, date: "2026-02-20", note: "", createdAt: "", updatedAt: "" },
];

// 固定 5 行对比（无缺失阶段）：概算环比 +18% 深红、预算环比 +8% 琥珀、
// 结算环比 -4% 默认绿、决算环比 +12% 琥珀；累计差逐行 ≥15% 深红。
const ROWS: CostStageCompareRow[] = [
  { stage: "估算", amount: 1000000, hasValue: true, prevStage: "", hasPrev: false, chainDiff: 0, chainDiffPct: 0, baseDiff: 0, baseDiffPct: 0 },
  { stage: "概算", amount: 1180000, hasValue: true, prevStage: "估算", hasPrev: true, chainDiff: 180000, chainDiffPct: 18, baseDiff: 180000, baseDiffPct: 18 },
  { stage: "预算", amount: 1260000, hasValue: true, prevStage: "概算", hasPrev: true, chainDiff: 80000, chainDiffPct: 8, baseDiff: 260000, baseDiffPct: 26 },
  { stage: "结算", amount: 1220000, hasValue: true, prevStage: "预算", hasPrev: true, chainDiff: -40000, chainDiffPct: -4, baseDiff: 220000, baseDiffPct: 22 },
  { stage: "决算", amount: 1366400, hasValue: true, prevStage: "结算", hasPrev: true, chainDiff: 146400, chainDiffPct: 12, baseDiff: 366400, baseDiffPct: 36.6 },
];

// 缺失阶段：预算 hasValue=false（金额/环比/累计差整行「—」）。
const ROWS_MISSING: CostStageCompareRow[] = [
  ROWS[0],
  ROWS[1],
  { stage: "预算", amount: 0, hasValue: false, prevStage: "", hasPrev: false, chainDiff: 0, chainDiffPct: 0, baseDiff: 0, baseDiffPct: 0 },
  ROWS[3],
  ROWS[4],
];

// 偏差三档：关注 / 正常 / 异常（suggestion 为后端规则文案，原样展示）。
const DEVS: CostStageDeviation[] = [
  { fromStage: "估算", toStage: "概算", fromAmount: 1000000, toAmount: 1180000, diff: 180000, diffPct: 18.2, direction: "上升", level: "关注", suggestion: "概算较估算 +18.2%,建议核查工程量或单价差异" },
  { fromStage: "概算", toStage: "预算", fromAmount: 1180000, toAmount: 1142000, diff: -38000, diffPct: -3.2, direction: "下降", level: "正常", suggestion: "预算较概算 -3.2%,处于正常波动范围" },
  { fromStage: "预算", toStage: "结算", fromAmount: 1142000, toAmount: 884000, diff: -258000, diffPct: -22.6, direction: "下降", level: "异常", suggestion: "结算较预算 -22.6%,异常偏离,建议核查变更签证与调价依据" },
];

beforeEach(() => {
  stagesSpy.mockReset();
  compareSpy.mockReset();
  devsSpy.mockReset();
  saveSpy.mockReset();
  stagesSpy.mockResolvedValue([]);
  compareSpy.mockResolvedValue([]);
  devsSpy.mockResolvedValue([]);
  saveSpy.mockResolvedValue(undefined);
});

describe("FiveCalcPanel 五算对比", () => {
  it("渲染五阶段输入行（固定顺序与名称），未录入阶段显示占位", async () => {
    stagesSpy.mockResolvedValue(STAGES_ONE);
    render(wrap(<FiveCalcPanel projectId={PROJECT_ID} />));

    // 三个加载器并行挂载，均以 projectId 调用。
    await screen.findByTestId("stage-input-估算");
    expect(stagesSpy).toHaveBeenCalledWith(PROJECT_ID);
    expect(compareSpy).toHaveBeenCalledWith(PROJECT_ID);
    expect(devsSpy).toHaveBeenCalledWith(PROJECT_ID);

    // 固定 5 行、顺序 估/概/预/结/决。
    const rowEls = [...document.querySelectorAll<HTMLElement>('[data-testid^="stage-input-"]')];
    expect(rowEls.map((r) => r.dataset.testid?.replace("stage-input-", ""))).toEqual([
      "估算",
      "概算",
      "预算",
      "结算",
      "决算",
    ]);
    for (const stage of ["估算", "概算", "预算", "结算", "决算"]) {
      const row = screen.getByTestId(`stage-input-${stage}`);
      expect(row.textContent).toContain(stage);
      expect(within(row).getByPlaceholderText("金额（元）")).toBeTruthy();
      expect(within(row).getByRole("button", { name: `保存${stage}` })).toBeTruthy();
    }

    // 仅估算已录入，其余 4 阶段「未录入」占位。
    expect(screen.getAllByText("未录入")).toHaveLength(4);
  });

  it("已存阶段输入框回填金额、徽标显示录入日期", async () => {
    stagesSpy.mockResolvedValue(STAGES_ONE);
    render(wrap(<FiveCalcPanel projectId={PROJECT_ID} />));

    const row = await screen.findByTestId("stage-input-估算");
    expect((within(row).getByPlaceholderText("金额（元）") as HTMLInputElement).value).toBe("1000000");
    expect(within(row).getByText("01-15")).toBeTruthy(); // date.slice(5)
    expect(within(row).getByRole("button", { name: "保存估算" })).toBeTruthy();
  });

  it("对比表渲染：金额/环比/累计差，差幅按档位着色（深红/琥珀/默认绿）", async () => {
    stagesSpy.mockResolvedValue(STAGES_TWO);
    compareSpy.mockResolvedValue(ROWS);
    render(wrap(<FiveCalcPanel projectId={PROJECT_ID} />));

    await screen.findByTestId("compare-row-估算");

    // 金额列。
    expect(screen.getByText("¥1,000,000")).toBeTruthy();
    expect(screen.getByText("¥1,180,000")).toBeTruthy();
    expect(screen.getByText("¥1,260,000")).toBeTruthy();
    expect(screen.getByText("¥1,220,000")).toBeTruthy();
    expect(screen.getByText("¥1,366,400")).toBeTruthy();

    // 环比差额（chainDiff）。
    expect(screen.getAllByText("+¥180,000")).toHaveLength(2); // 概算环比 + 概算累计差
    expect(screen.getByText("+¥80,000")).toBeTruthy();
    expect(screen.getByText("-¥40,000")).toBeTruthy();
    expect(screen.getByText("+¥146,400")).toBeTruthy();

    // 基准行（估算）：环比无上一有值阶段 → 「—」；累计差 0%。
    const baseRow = within(screen.getByTestId("compare-row-估算"));
    expect(baseRow.getByText("—")).toBeTruthy();
    expect(baseRow.getByText("0%").className).toContain("text-fg-dim");

    // 概算：环比 +18% 与累计差 +18% 均深红（|pct|>=15）。
    const rowGaisuan = within(screen.getByTestId("compare-row-概算"));
    const deep18 = rowGaisuan.getAllByText("+18%");
    expect(deep18).toHaveLength(2);
    for (const el of deep18) expect(el.className).toContain("text-red-500");

    // 预算：环比 +8% 琥珀（5<=|pct|<15），累计差 +26% 深红。
    const rowYusuan = within(screen.getByTestId("compare-row-预算"));
    expect(rowYusuan.getByText("+8%").className).toContain("text-amber-400");
    expect(rowYusuan.getByText("+26%").className).toContain("text-red-500");

    // 结算：环比 -4% 默认绿（下降），累计差 +22% 深红。
    const rowJiesuan = within(screen.getByTestId("compare-row-结算"));
    expect(rowJiesuan.getByText("-4%").className).toContain("text-emerald-400");
    expect(rowJiesuan.getByText("+22%").className).toContain("text-red-500");

    // 决算：环比 +12% 琥珀，累计差 +36.6% 深红。
    const rowJuesuan = within(screen.getByTestId("compare-row-决算"));
    expect(rowJuesuan.getByText("+12%").className).toContain("text-amber-400");
    expect(rowJuesuan.getByText("+36.6%").className).toContain("text-red-500");
  });

  it("缺失阶段（hasValue=false）整行显示「—」", async () => {
    stagesSpy.mockResolvedValue(STAGES_ONE);
    compareSpy.mockResolvedValue(ROWS_MISSING);
    render(wrap(<FiveCalcPanel projectId={PROJECT_ID} />));

    const missing = within(await screen.findByTestId("compare-row-预算"));
    expect(missing.getByText("预算")).toBeTruthy();
    expect(missing.getAllByText("—")).toHaveLength(3); // 金额/环比/累计差

    // 有值行不受影响：结算环比 -4% 正常渲染。
    expect(within(screen.getByTestId("compare-row-结算")).getByText("-4%")).toBeTruthy();
  });

  it("偏差卡片渲染：level 徽标（正常绿/关注琥珀/异常红）+ suggestion 文案", async () => {
    stagesSpy.mockResolvedValue(STAGES_ONE);
    devsSpy.mockResolvedValue(DEVS);
    render(wrap(<FiveCalcPanel projectId={PROJECT_ID} />));

    const card1 = within(await screen.findByTestId("deviation-估算-概算"));
    expect(card1.getByText("概算较估算 +18.2%")).toBeTruthy(); // 卡片标题
    expect(card1.getByText("↑ 上升")).toBeTruthy();
    const badge1 = card1.getByText("关注");
    expect(badge1.className).toContain("text-amber-400");
    expect(card1.getByText("概算较估算 +18.2%,建议核查工程量或单价差异")).toBeTruthy();

    const card2 = within(screen.getByTestId("deviation-概算-预算"));
    expect(card2.getByText("预算较概算 -3.2%")).toBeTruthy();
    expect(card2.getByText("↓ 下降")).toBeTruthy();
    expect(card2.getByText("正常").className).toContain("text-ok");
    expect(card2.getByText("预算较概算 -3.2%,处于正常波动范围")).toBeTruthy();

    const card3 = within(screen.getByTestId("deviation-预算-结算"));
    expect(card3.getByText("异常").className).toContain("text-red-500");
    expect(card3.getByText("结算较预算 -22.6%,异常偏离,建议核查变更签证与调价依据")).toBeTruthy();
  });

  it("保存流程：SaveStage 参数正确 + 成功 toast + 刷新对比 + onChanged 回调", async () => {
    stagesSpy.mockResolvedValue(STAGES_ONE);
    compareSpy.mockResolvedValue(ROWS);
    devsSpy.mockResolvedValue(DEVS);
    const onChanged = vi.fn();
    render(wrap(<FiveCalcPanel projectId={PROJECT_ID} onChanged={onChanged} />));

    const row = await screen.findByTestId("stage-input-概算");
    fireEvent.change(within(row).getByPlaceholderText("金额（元）"), { target: { value: "1200000" } });
    fireEvent.click(within(row).getByRole("button", { name: "保存概算" }));

    await waitFor(() => expect(saveSpy).toHaveBeenCalledTimes(1));
    expect(saveSpy).toHaveBeenCalledWith(
      expect.objectContaining({ projectId: PROJECT_ID, stage: "概算", amount: 1200000, date: "", note: "" }),
    );
    // 成功 toast + 保存后刷新三路数据 + onChanged。
    expect(await screen.findByText("概算阶段已保存")).toBeTruthy();
    await waitFor(() => expect(compareSpy).toHaveBeenCalledTimes(2)); // 初次挂载 + 保存后刷新
    expect(stagesSpy).toHaveBeenCalledTimes(2);
    expect(devsSpy).toHaveBeenCalledTimes(2);
    expect(onChanged).toHaveBeenCalledTimes(1);
  });

  it("空态：无任何阶段值时提示录入引导，输入行仍可用", async () => {
    render(wrap(<FiveCalcPanel projectId={PROJECT_ID} />));

    expect(await screen.findByText("录入五个阶段的金额后自动生成对比与偏差诊断")).toBeTruthy();
    expect(screen.queryByText("对比表")).toBeNull();
    expect(screen.queryByText("偏差诊断")).toBeNull();
    // 输入行仍渲染，可开始录入。
    expect(screen.getAllByTestId(/^stage-input-/)).toHaveLength(5);
  });
});
