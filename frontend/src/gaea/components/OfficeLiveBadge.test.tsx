// OfficeLiveBadge.test.tsx — 实时编辑胶囊（波3）：新鲜度门控、两种变体文案、
// 点击经 paneFileOpen 统一入口打开被编辑文档、过期后消失并就地清 store。
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, cleanup, act } from "@testing-library/react";
import { OfficeLiveBadge } from "./OfficeLiveBadge";
import { resetOfficeLiveStoreForTest, useOfficeLiveStore, OFFICE_LIVE_TTL_MS } from "../lib/officeLiveStore";
import { setPaneFileOpenHandler } from "../lib/paneFileOpen";

const T0 = 1_700_000_000_000;

function seedLive(patch?: { doc?: string; step?: { done: number; total: number }; ageMs?: number }) {
  useOfficeLiveStore.setState({
    activeDoc: patch?.doc ?? "docs/报告.docx",
    activeApp: "word",
    step: patch?.step ?? null,
    updatedAt: Date.now() - (patch?.ageMs ?? 0),
  });
}

beforeEach(() => {
  vi.useFakeTimers();
  vi.setSystemTime(T0);
  resetOfficeLiveStoreForTest();
});

afterEach(() => {
  cleanup();
  setPaneFileOpenHandler(null);
  vi.useRealTimers();
});

describe("OfficeLiveBadge", () => {
  it("无活动文档：不渲染", () => {
    render(<OfficeLiveBadge />);
    expect(screen.queryByTestId("office-live-badge")).toBeNull();
  });

  it("活动文档在 TTL 内：渲染文件名 + 「实时编辑中」", () => {
    seedLive();
    render(<OfficeLiveBadge />);
    const badge = screen.getByTestId("office-live-badge");
    expect(badge.textContent).toContain("报告.docx");
    expect(badge.textContent).toContain("实时编辑中");
  });

  it("有 step：显示「编辑中 · 第 done/total 步」", () => {
    seedLive({ step: { done: 2, total: 5 } });
    render(<OfficeLiveBadge />);
    expect(screen.getByTestId("office-live-badge").textContent).toContain("编辑中 · 第 2/5 步");
  });

  it("relPath 与 activeDoc 同文件：渲染「此文件正在被 AI 实时编辑」变体（归一化忽略分隔符/大小写）", () => {
    seedLive({ doc: "docs/报告.docx" });
    render(<OfficeLiveBadge relPath="DOCS/报告.docx" />);
    expect(screen.getByTestId("office-live-badge").textContent).toContain("此文件正在被 AI 实时编辑");
    // 不同文件 → 他文件变体（显示被编辑文档名）
    cleanup();
    resetOfficeLiveStoreForTest();
    seedLive({ doc: "docs/其他.docx" });
    render(<OfficeLiveBadge relPath="docs/报告.docx" />);
    expect(screen.getByTestId("office-live-badge").textContent).toContain("其他.docx");
  });

  it("点击：经 openPaneFileOrPreview 打开被编辑文档", () => {
    const opened: string[] = [];
    setPaneFileOpenHandler((rel) => opened.push(rel));
    seedLive({ doc: "exports/成本测算.xlsx" });
    render(<OfficeLiveBadge />);
    fireEvent.click(screen.getByTestId("office-live-badge"));
    expect(opened).toEqual(["exports/成本测算.xlsx"]);
  });

  it("超过 TTL：不渲染，且惰性清空 store（下次挂载也不再复活）", () => {
    seedLive({ ageMs: OFFICE_LIVE_TTL_MS + 1000 });
    render(<OfficeLiveBadge />);
    expect(screen.queryByTestId("office-live-badge")).toBeNull();
    expect(useOfficeLiveStore.getState().activeDoc).toBeNull();
  });

  it("TTL 内活动但随后过期（时钟推进）：胶囊消失", () => {
    seedLive();
    render(<OfficeLiveBadge />);
    expect(screen.getByTestId("office-live-badge")).toBeTruthy();
    vi.setSystemTime(T0 + OFFICE_LIVE_TTL_MS + 2000);
    act(() => {
      vi.advanceTimersByTime(2000); // 触发 useNow 秒钟重渲染（React 状态更新须在 act 内冲刷）
    });
    expect(screen.queryByTestId("office-live-badge")).toBeNull();
    expect(useOfficeLiveStore.getState().activeDoc).toBeNull();
  });
});
