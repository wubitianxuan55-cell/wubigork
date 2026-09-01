// VersionTimeline.test.tsx — 版本时间线组件单测（加载/空态/列表/预览/恢复回调）。
// 组件为纯展示层：records / onPreview / onRestore 全经 props 注入，无需 mock bridge。
import { describe, expect, it, vi } from "vitest";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { VersionTimeline } from "./VersionTimeline";
import type { JournalChangeRecord } from "../lib/types";

// 固定本地时间构造（避免时区漂移）：r1=14:05、r2=15:20、r3=12:01。
const t = (h: number, m: number) => new Date(2026, 8, 1, h, m, 0).getTime();

const rec = (over: Partial<JournalChangeRecord> & { id: string }): JournalChangeRecord => ({
  sessionId: "s1",
  space: "work",
  turn: 1,
  tool: "edit_file",
  target: "docs/周报.docx",
  beforeSummary: "旧内容",
  afterSummary: "新内容",
  at: t(14, 5),
  status: "pending_verify",
  baselinePath: "C:/ws/.gaea/work/rollback/xxx.snap",
  ...over,
});

describe("VersionTimeline 版本时间线", () => {
  it("records=null：加载态（转圈 + 文案），不渲染列表", () => {
    render(<VersionTimeline path="docs/周报.docx" records={null} onPreview={() => {}} onRestore={() => {}} />);
    expect(screen.getByTestId("version-timeline-loading")).toBeTruthy();
    expect(screen.getByText("正在加载版本记录…")).toBeTruthy();
    expect(screen.queryByTestId("version-timeline-row")).toBeNull();
  });

  it("records=[]：空态（无基线快照提示），恢复说明常驻", () => {
    render(<VersionTimeline path="docs/周报.docx" records={[]} onPreview={() => {}} onRestore={() => {}} />);
    expect(screen.getByTestId("version-timeline-empty")).toBeTruthy();
    expect(screen.getByText(/暂无可回滚的版本快照/)).toBeTruthy();
    // 顶部恢复语义说明（预览即护栏，不做二次确认弹窗）
    expect(screen.getByText("恢复会把该文件写回所选版本，当前内容成为新版本")).toBeTruthy();
  });

  it("列表渲染：按传入顺序（调用方已 at 倒序）逐行展示时间/工具/轮次/状态", () => {
    const rows = [
      rec({ id: "r2", at: t(15, 20), tool: "xlsx_apply", turn: 3, status: "verified" }),
      rec({ id: "r1", at: t(14, 5), tool: "edit_file", turn: 2, status: "pending_verify" }),
      rec({ id: "r3", at: t(12, 1), tool: "write_file", turn: 0, status: "failed" }),
    ];
    render(<VersionTimeline path="docs/周报.docx" records={rows} onPreview={() => {}} onRestore={() => {}} />);
    const items = screen.getAllByTestId("version-timeline-row");
    expect(items).toHaveLength(3);
    // 最新在前：第一行 15:20 · xlsx_apply · 第 3 轮 · 复核通过
    expect(items[0].textContent).toContain("15:20");
    expect(screen.getByText("xlsx_apply")).toBeTruthy();
    expect(screen.getByText("第 3 轮")).toBeTruthy();
    expect(screen.getByText("复核通过")).toBeTruthy();
    expect(items[1].textContent).toContain("14:05");
    expect(screen.getByText("第 2 轮")).toBeTruthy();
    expect(screen.getByText("待复核")).toBeTruthy();
    // 轮次 0 显示「轮外」；failed 状态徽标
    expect(items[2].textContent).toContain("12:01");
    expect(screen.getByText("轮外")).toBeTruthy();
    expect(screen.getByText("复核未通过")).toBeTruthy();
    // 快照计数徽标
    expect(screen.getByText("3")).toBeTruthy();
  });

  it("预览回调：点「预览」注入基线快照路径", () => {
    const onPreview = vi.fn();
    render(
      <VersionTimeline
        path="docs/周报.docx"
        records={[rec({ id: "r1", baselinePath: "C:/ws/.gaea/work/rollback/a.snap" })]}
        onPreview={onPreview}
        onRestore={() => {}}
      />,
    );
    fireEvent.click(screen.getByTitle("预览该版本快照"));
    expect(onPreview).toHaveBeenCalledTimes(1);
    expect(onPreview).toHaveBeenCalledWith("C:/ws/.gaea/work/rollback/a.snap");
  });

  it("恢复回调：点「恢复」注入该条记录；恢复进行中禁用全部恢复按钮，完成后恢复可用", async () => {
    let resolveRestore: () => void = () => {};
    const onRestore = vi.fn(
      (_r: JournalChangeRecord) => new Promise<void>((res) => { resolveRestore = res; }),
    );
    render(
      <VersionTimeline
        path="docs/周报.docx"
        records={[rec({ id: "r1" }), rec({ id: "r2", at: t(15, 20) })]}
        onPreview={() => {}}
        onRestore={onRestore}
      />,
    );
    fireEvent.click(screen.getByTitle("恢复到 14:05 版本：将回滚到该时间版本"));
    expect(onRestore).toHaveBeenCalledTimes(1);
    expect(onRestore).toHaveBeenCalledWith(expect.objectContaining({ id: "r1" }));
    // 恢复进行中：所有恢复按钮禁用（避免并发写盘），本行转圈
    const restoreBtns = screen.getAllByTitle(/^恢复到 \d{2}:\d{2} 版本/);
    expect(restoreBtns).toHaveLength(2);
    restoreBtns.forEach((b) => expect((b as HTMLButtonElement).disabled).toBe(true));
    // 完成 → 按钮恢复可用
    await act(async () => { resolveRestore(); });
    await waitFor(() =>
      restoreBtns.forEach((b) => expect((b as HTMLButtonElement).disabled).toBe(false)),
    );
  });
});
