import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { PptxOutline } from "./PptxOutline";
import { ToastProvider } from "./Toast";
import { useComposerInsertStore } from "../lib/store";
import type { PptxOutlineView } from "../lib/types";

const mocks = vi.hoisted(() => ({
  outline: vi.fn((_rel: string): Promise<PptxOutlineView> => Promise.resolve({ available: true, slides: [] })),
}));

vi.mock("../lib/bridge", () => ({
  app: {
    PptxOutline: (rel: string) => mocks.outline(rel),
  },
}));

const outlineView = (): PptxOutlineView => ({
  available: true,
  slides: [
    { index: 1, title: "季度经营总结", texts: ["营收增长 12%", "成本结构优化"], shapeCount: 5 },
    { index: 2, title: "", texts: ["备注事项"], shapeCount: 3 },
  ],
});

const wrap = (node: React.ReactNode) => <ToastProvider>{node}</ToastProvider>;

beforeEach(() => {
  mocks.outline.mockReset();
  mocks.outline.mockResolvedValue(outlineView());
  useComposerInsertStore.setState({ pendingText: null, pendingAt: null });
});

describe("PptxOutline 大纲卡（v4.28 B2）", () => {
  it("渲染每页条目：页码/标题/正文摘要与页数徽标", async () => {
    render(wrap(<PptxOutline relPath="exports/汇报.pptx" fileName="汇报.pptx" />));
    expect(await screen.findByText("季度经营总结")).toBeTruthy();
    expect(screen.getByText("营收增长 12% ｜ 成本结构优化")).toBeTruthy();
    // 无标题页也有条目；页数徽标 = slides.length
    expect(screen.getByTestId("pptx-page-item-2")).toBeTruthy();
    expect(screen.getByText("2 页")).toBeTruthy();
    expect(mocks.outline).toHaveBeenCalledWith("exports/汇报.pptx");
  });

  it("点页条目回调 onPageSelect(N)", async () => {
    const onSelect = vi.fn();
    render(wrap(<PptxOutline relPath="a.pptx" fileName="a.pptx" onPageSelect={onSelect} />));
    await screen.findByText("季度经营总结");
    fireEvent.click(screen.getByTestId("pptx-page-item-2"));
    expect(onSelect).toHaveBeenCalledWith(2);
  });

  it("「针对第 N 页修改」把指令模板插入 composer（不自动发送）", async () => {
    render(wrap(<PptxOutline relPath="a.pptx" fileName="汇报.pptx" />));
    await screen.findByText("季度经营总结");
    fireEvent.click(screen.getByTestId("pptx-modify-btn-2"));
    await waitFor(() =>
      expect(useComposerInsertStore.getState().pendingText).toBe("请修改 汇报.pptx 的第 2 页："),
    );
  });

  it("大纲不可用（python 缺失）→ 诚实提示原因，无页条目", async () => {
    mocks.outline.mockResolvedValue({ available: false, error: "python-pptx 不可用", slides: [] });
    render(wrap(<PptxOutline relPath="a.pptx" fileName="a.pptx" />));
    expect(await screen.findByText(/大纲不可用/)).toBeTruthy();
    expect(screen.getByText(/python-pptx 不可用/)).toBeTruthy();
    expect(screen.queryByTestId("pptx-page-item-1")).toBeNull();
  });

  it("拉取异常（reject）→ 同样诚实提示", async () => {
    mocks.outline.mockRejectedValue(new Error("boom"));
    render(wrap(<PptxOutline relPath="a.pptx" fileName="a.pptx" />));
    expect(await screen.findByText(/大纲不可用/)).toBeTruthy();
  });

  it("绑定未接线（方法缺失返回 undefined）→ 按不可用降级，不停留加载态", async () => {
    mocks.outline.mockResolvedValue(undefined as unknown as PptxOutlineView);
    render(wrap(<PptxOutline relPath="a.pptx" fileName="a.pptx" />));
    expect(await screen.findByText(/大纲不可用/)).toBeTruthy();
    expect(screen.queryByText("读取大纲…")).toBeNull();
  });
});
