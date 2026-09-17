import { describe, expect, it, beforeEach, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { FileVersionStrip } from "./FileVersionStrip";
import { LocaleProvider } from "../lib/i18n";

const { gaeaJournalList, rollbackRecord, openFilePreview } = vi.hoisted(() => ({
  gaeaJournalList: vi.fn(),
  rollbackRecord: vi.fn(),
  openFilePreview: vi.fn(),
}));

vi.mock("../lib/bridge", () => ({
  app: { GaeaJournalList: gaeaJournalList, RollbackRecord: rollbackRecord },
}));
vi.mock("../lib/store", () => ({
  usePreviewStore: (sel: (s: { openFilePreview: unknown }) => unknown) =>
    sel({ openFilePreview }),
}));
vi.mock("./Toast", () => ({
  useToast: () => ({ show: vi.fn() }),
}));

const RECORDS = [
  {
    id: "rec-2",
    sessionId: "s",
    space: "work",
    turn: 2,
    tool: "pptx_apply",
    target: "docs/报价.pptx",
    beforeSummary: 'p3 "旧文案"',
    afterSummary: 'p3 → "新文案"',
    baselinePath: "/abs/.gaea/work/rollback/x.before",
    at: 1726000002000,
    status: "pending_verify",
  },
  {
    id: "rec-1",
    sessionId: "s",
    space: "work",
    turn: 1,
    tool: "edit_file",
    target: "docs/报价.pptx",
    beforeSummary: "before",
    afterSummary: "after",
    baselinePath: "/abs/.gaea/work/rollback/y.before",
    at: 1726000001000,
    status: "verified",
  },
];

describe("FileVersionStrip 文件版本时间线一级入口（6.1 可审计默认化）", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    gaeaJournalList.mockResolvedValue(RECORDS);
  });

  it("默认可见：渲染版本数/最近改动/状态，展开为该文件的逐版本时间线", async () => {
    render(<LocaleProvider><FileVersionStrip relPath="docs/报价.pptx" /></LocaleProvider>);

    // 条本体默认可见（无版本也可见，见下例）
    const strip = await screen.findByText(/版本/);
    expect(strip).toBeTruthy();
    expect(await screen.findByText(/最近 AI 改动/)).toBeTruthy();
    expect(screen.getByText(/可恢复/)).toBeTruthy();

    // 展开 → VersionTimeline（最新在前：p3 页码归因来自证据卡摘要）
    fireEvent.click(screen.getByTitle(/本文件的 AI 改动版本时间线/));
    await waitFor(() => expect(screen.getAllByText(/pptx_apply/).length).toBeGreaterThan(0));
    // 恢复按钮来自 VersionTimeline 行操作（恢复到 HH:MM 版本，2 版本=2 行）
    const restoreBtns = await screen.findAllByTitle(/恢复到 .* 版本/);
    expect(restoreBtns.length).toBe(2);
  });

  it("恢复：调用 RollbackRecord 并触发 onRestored（父级刷预览）", async () => {
    rollbackRecord.mockResolvedValue(undefined);
    const onRestored = vi.fn();
    render(<LocaleProvider><FileVersionStrip relPath="docs/报价.pptx" onRestored={onRestored} /></LocaleProvider>);

    fireEvent.click(screen.getByTitle(/本文件的 AI 改动版本时间线/));
    const restoreBtns = await screen.findAllByTitle(/恢复到 .* 版本/);
    fireEvent.click(restoreBtns[0]);
    await waitFor(() => expect(rollbackRecord).toHaveBeenCalledWith("rec-2"));
    await waitFor(() => expect(onRestored).toHaveBeenCalled());
  });

  it("无版本：条仍在（说明文案），展开为空态——回滚入口默认在", async () => {
    gaeaJournalList.mockResolvedValue([]);
    render(<LocaleProvider><FileVersionStrip relPath="docs/新文件.docx" /></LocaleProvider>);

    expect(await screen.findByText(/AI 改动此文件会自动成为版本点/)).toBeTruthy();
    fireEvent.click(screen.getByTitle(/本文件的 AI 改动版本时间线/));
    await waitFor(() => expect(screen.getByText(/暂无可回滚的版本快照/)).toBeTruthy());
  });

  it("证据链不可用：静默降级为 0 版本条，不抛错", async () => {
    gaeaJournalList.mockRejectedValue(new Error("no journal"));
    render(<LocaleProvider><FileVersionStrip relPath="a.md" /></LocaleProvider>);
    expect(await screen.findByText(/AI 改动此文件会自动成为版本点/)).toBeTruthy();
  });
});
