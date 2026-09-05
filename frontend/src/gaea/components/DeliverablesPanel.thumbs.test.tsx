import { describe, expect, it } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import type { ReactElement } from "react";
import { DeliverablesPanel } from "./DeliverablesPanel";
import { ToastProvider } from "./Toast";
import { LocaleProvider } from "../lib/i18n";

// 老账收口·面板级集成：证据卡「视觉复核」行的「查看缩略图」入口接线。
// 与 DeliverablesPanel.test.tsx 同款姿势：不 mock 桥接，走内置开发 mock
// （mock VerifyRecord 的 ev_1003 携带 channelBRatio/channelBArtifacts；
// ev_1001 保持旧形态无通道 B 字段；mock ListDir 对 ev_1003 产物目录返回
// before/after 各 3 页 PNG（v4.96 走查样例，布局对齐 gaea_verify.go）→
// 展开后缩略卡真实渲染）。降级分支由 VerifyArtifactsThumbs.test.tsx 的
// bridge mock 层全覆盖。
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

describe("DeliverablesPanel 视觉复核行·查看缩略图入口（老账）", () => {
  it("通道 B 字段齐备：视觉复核行出现「查看缩略图」入口（与「查看产物」并列）", async () => {
    await openEvidence();
    fireEvent.click(screen.getAllByTitle("双通道复核（结构/引用完整性 + 视觉健全性）")[0]);
    await screen.findByText("视觉复核：像素差异率 1.3% · 3 页");
    // 既有「查看产物」保留；新增「查看缩略图」入口（懒加载，展开才拉）
    expect(screen.getByTitle("查看复核产物（before/after PDF + 逐页 PNG）")).toBeTruthy();
    expect(screen.getByTestId("verify-thumbs-toggle")).toBeTruthy();
    expect(screen.queryByTestId("verify-thumbs-grid")).toBeNull(); // 未展开
  });

  it("展开缩略图：mock 产物目录有 before/after 各 3 页 → 缩略卡成对真实渲染", async () => {
    await openEvidence();
    fireEvent.click(screen.getAllByTitle("双通道复核（结构/引用完整性 + 视觉健全性）")[0]);
    await screen.findByText("视觉复核：像素差异率 1.3% · 3 页");
    fireEvent.click(screen.getByTestId("verify-thumbs-toggle"));
    await screen.findByTestId("verify-thumbs-grid");
    const cells = screen.getAllByTestId("verify-thumb-cell");
    expect(cells.length).toBe(6); // before 3 页 + after 3 页
    expect(screen.queryByTestId("verify-thumbs-fault")).toBeNull();
  });

  it("旧 verdict 无通道 B 字段：不渲染「视觉复核」行，也无缩略图入口（向后兼容）", async () => {
    await openEvidence();
    fireEvent.click(screen.getAllByTitle("双通道复核（结构/引用完整性 + 视觉健全性）")[2]);
    await screen.findByText("复核警告");
    expect(screen.queryByText(/视觉复核/)).toBeNull();
    expect(screen.queryByTestId("verify-thumbs-toggle")).toBeNull();
  });
});
