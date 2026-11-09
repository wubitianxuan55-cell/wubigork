import { describe, expect, it } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import type { ReactElement } from "react";
import { DeliverablesPanel } from "./DeliverablesPanel";
import { ToastProvider } from "./Toast";
import { LocaleProvider } from "../lib/i18n";

// P2 条件视觉首渲·面板级集成（2026-11-10 众神殿裁决 action-5 后半）：
// 无基线交付物的「首渲覆盖」行（visualPagesReviewed/Total/SkippedReason 透明
// 口径，备忘 :42-48 形状继承）。走内置开发 mock：mock VerifyRecord 的 ev_1002
// 携带 first-review 字段（8/12 页 + 跳过原因）；ev_1003 的 diff 行
// （视觉复核）与 ev_1001 的旧形态互斥样本不动。
const renderT = (ui: ReactElement) => {
  localStorage.setItem("gaea-lang", "zh");
  return render(<ToastProvider><LocaleProvider>{ui}</LocaleProvider></ToastProvider>);
};

const openEvidence = async () => {
  renderT(
    <DeliverablesPanel items={[{ path: "docs/成本测算.xlsx", sourceId: "a1" }]} onOpenFile={() => {}} />,
  );
  fireEvent.click(screen.getByText("证据链"));
  await screen.findByText("3 条变更证据卡");
};

describe("DeliverablesPanel 首渲覆盖行（P2 条件视觉）", () => {
  it("first-review verdict：覆盖率行渲染（抽样原因透明）+ 查看产物入口", async () => {
    await openEvidence();
    // 卡序=JournalList 序（ev_1003/ev_1002/ev_1001）→ [1]=ev_1002（首渲样本）
    fireEvent.click(screen.getAllByTitle("双通道复核（结构/引用完整性 + 视觉健全性）")[1]);
    await screen.findByText(/首渲覆盖：8\/12 页 · 超 8 页抽样/);
    // diff 行（视觉复核）与首渲覆盖行互斥形态：本卡不出现 diff 行
    expect(screen.queryByText(/视觉复核：像素差异率/)).toBeNull();
    expect(screen.getByTitle("查看复核产物（before/after PDF + 逐页 PNG）")).toBeTruthy();
  });

  it("有基线 diff 卡：覆盖率行不出现（diff 走视觉复核行，双行不并存）", async () => {
    await openEvidence();
    fireEvent.click(screen.getAllByTitle("双通道复核（结构/引用完整性 + 视觉健全性）")[0]);
    await screen.findByText("视觉复核：像素差异率 1.3% · 3 页");
    expect(screen.queryByText(/首渲覆盖/)).toBeNull();
  });
});
