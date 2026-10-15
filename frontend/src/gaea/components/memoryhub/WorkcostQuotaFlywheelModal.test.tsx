// WorkcostQuotaFlywheelModal.test.tsx — 企业定额自生成弹窗（11.6）
// 契约：打开即预览（候选卡=成员/聚合行/多数派标注+跳过原因）；点名后 Apply
// 只传选中词干；回执=生成码列表；onApplied 刷父层。
import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { WorkcostQuotaFlywheelModal } from "./WorkcostQuotaFlywheelModal";
import type { WorkcostQuotaFlywheelPreview, WorkcostQuotaFlywheelResult } from "../../lib/types";

const state = vi.hoisted(() => ({
  preview: null as WorkcostQuotaFlywheelPreview | null,
  previewError: "" as string,
  applyCalls: [] as string[][],
  applyResult: null as WorkcostQuotaFlywheelResult | null,
}));

vi.mock("../../lib/bridge", () => ({
  app: {
    CostQuotaFlywheelPreview: async (): Promise<WorkcostQuotaFlywheelPreview> => {
      if (state.previewError) throw new Error(state.previewError);
      if (!state.preview) throw new Error("no preview seeded");
      return state.preview;
    },
    CostQuotaFlywheelApply: async (codes: string[]): Promise<WorkcostQuotaFlywheelResult> => {
      state.applyCalls.push(codes);
      return state.applyResult ?? { saved: [], skipped: [] };
    },
  },
}));

import { LocaleProvider } from "../../lib/i18n";

const seedPreview = (): WorkcostQuotaFlywheelPreview => ({
  clusters: 2,
  candidates: [
    {
      origCode: "WP-100",
      newCode: "ENT-WP-100",
      title: "污染土壤开挖",
      unit: "m3",
      sources: 3,
      members: [
        { code: "WP-100", origCode: "WP-100", source: "项目导入：甲.xlsx", itemCount: 3 },
        { code: "WP-100-2", origCode: "WP-100", source: "项目导入：乙.xlsx", itemCount: 2 },
      ],
      rows: [
        { resourceCode: "RG0001", kind: "人工", title: "普工", unit: "工日", quantity: 0.2, coverage: 2, min: 0.1, max: 0.3, majority: true },
        { resourceCode: "CL0001", kind: "材料", title: "P.O 42.5 水泥", unit: "kg", quantity: 2, coverage: 1, min: 2, max: 2, majority: false },
      ],
    },
  ],
  skipped: [{ key: "B-1", title: "同源变体", reason: "2 条变体全部来自同一来源（项目导入：甲.xlsx），不是跨项目分歧" }],
});

const renderModal = (onApplied = vi.fn()) => {
  const onClosed = vi.fn();
  render(
    <LocaleProvider>
      <WorkcostQuotaFlywheelModal open onClose={onClosed} onApplied={onApplied} />
    </LocaleProvider>,
  );
  return { onApplied, onClosed };
};

beforeEach(() => {
  state.preview = null;
  state.previewError = "";
  state.applyCalls = [];
  state.applyResult = null;
});

describe("WorkcostQuotaFlywheelModal 企业定额自生成", () => {
  it("打开即预览：候选卡显成员/聚合行/多数派标注，跳过簇带原因；默认全选", async () => {
    state.preview = seedPreview();
    renderModal();
    expect(await screen.findByText("污染土壤开挖")).toBeDefined();
    expect(screen.getByTestId("quota-flywheel-preview")).toBeDefined();
    // 聚合行：中位数 + 覆盖度 + 多数派/少数派。
    const text = document.body.textContent ?? "";
    expect(text).toContain("中位 0.2");
    expect(text).toContain("2/2 变体");
    expect(text).toContain("多数派");
    expect(text).toContain("少数派");
    // 跳过原因如实显形。
    expect(text).toContain("不是跨项目分歧");
    // 默认全选：生成按钮计数 1。
    expect(screen.getByText("生成所选（1）")).toBeDefined();
  });

  it("取消勾选后 Apply 只传选中词干；回执显生成码并触发 onApplied", async () => {
    state.preview = seedPreview();
    state.applyResult = { saved: ["ENT-WP-100"], skipped: [] };
    const { onApplied } = renderModal();
    await screen.findByText("污染土壤开挖");
    fireEvent.click(screen.getByTestId("flywheel-pick-WP-100"));
    expect(screen.getByText("生成所选（0）")).toBeDefined();
    const applyBtn = screen.getByText("生成所选（0）").closest("button") as HTMLButtonElement;
    expect(applyBtn.disabled).toBe(true);
    // 重新勾选再生成。
    fireEvent.click(screen.getByTestId("flywheel-pick-WP-100"));
    fireEvent.click(screen.getByText("生成所选（1）"));
    await waitFor(() => expect(screen.getByTestId("quota-flywheel-result")).toBeDefined());
    expect(state.applyCalls).toEqual([["WP-100"]]);
    expect(document.body.textContent).toContain("ENT-WP-100");
    expect(onApplied).toHaveBeenCalledTimes(1);
  });

  it("零候选：警示横幅+生成钮禁用", async () => {
    state.preview = { clusters: 3, candidates: [], skipped: [{ key: "C-1", title: "含量全等", reason: "各变体含量完全一致（没有分歧可聚合；如重复出现可走编码统一收编）" }] };
    renderModal();
    expect(await screen.findByText(/候选 0 簇/)).toBeDefined();
    const applyBtn = screen.getByText("生成所选（0）").closest("button") as HTMLButtonElement;
    expect(applyBtn.disabled).toBe(true);
  });

  it("预览失败：错误透出（诚实显形）", async () => {
    state.previewError = "数据库未打开";
    renderModal();
    expect(await screen.findByText("数据库未打开")).toBeDefined();
  });
});
