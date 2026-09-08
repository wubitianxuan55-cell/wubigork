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

// v4.156 pptx 真编辑刀2「编辑面」：页条目可展开该页文本框清单（texts 截断
// 预览），点文本框条目或「编辑此页」→ onEditSlide(slideIdx) 通知宿主
// FilePreview 打开 PptxEditPanel（编辑面板另行拉取段落全文，大纲预览绝不当
// 替换目标）。onEditSlide 未接线（如弹窗宿主）→ 编辑入口不渲染、文本条目
// 退化为纯文本，既有导航能力零删减。
describe("PptxOutline 两级导航（v4.156 刀2）", () => {
  it("默认收起；点箭头展开该页文本框清单，再点收起", async () => {
    render(wrap(<PptxOutline relPath="a.pptx" fileName="a.pptx" onEditSlide={vi.fn()} />));
    await screen.findByText("季度经营总结");
    expect(screen.queryByTestId("pptx-text-1-0")).toBeNull();
    fireEvent.click(screen.getByTestId("pptx-expand-1"));
    expect(screen.getByTestId("pptx-text-1-0")).toBeTruthy();
    expect(screen.getByTestId("pptx-text-1-1")).toBeTruthy();
    expect(screen.getByTestId("pptx-text-1-0").textContent).toContain("营收增长 12%");
    fireEvent.click(screen.getByTestId("pptx-expand-1"));
    expect(screen.queryByTestId("pptx-text-1-0")).toBeNull();
  });

  it("点文本框条目 → onEditSlide(该页页码)", async () => {
    const onEdit = vi.fn();
    render(wrap(<PptxOutline relPath="a.pptx" fileName="a.pptx" onEditSlide={onEdit} />));
    await screen.findByText("季度经营总结");
    fireEvent.click(screen.getByTestId("pptx-expand-2"));
    fireEvent.click(screen.getByTestId("pptx-text-2-0"));
    expect(onEdit).toHaveBeenCalledWith(2);
  });

  it("「编辑此页」入口 → onEditSlide(该页页码)", async () => {
    const onEdit = vi.fn();
    render(wrap(<PptxOutline relPath="a.pptx" fileName="a.pptx" onEditSlide={onEdit} />));
    await screen.findByText("季度经营总结");
    fireEvent.click(screen.getByTestId("pptx-edit-btn-1"));
    expect(onEdit).toHaveBeenCalledWith(1);
  });

  it("onEditSlide 未接线 → 不渲染编辑入口，展开后文本条目为纯文本（向后兼容）", async () => {
    render(wrap(<PptxOutline relPath="a.pptx" fileName="a.pptx" />));
    await screen.findByText("季度经营总结");
    expect(screen.queryByTestId("pptx-edit-btn-1")).toBeNull();
    fireEvent.click(screen.getByTestId("pptx-expand-1"));
    const item = screen.getByTestId("pptx-text-1-0");
    expect(item.tagName).toBe("DIV"); // 纯文本，非按钮
    // 既有能力保留：页锚点滚动与「针对第 N 页修改」不受影响
    fireEvent.click(screen.getByTestId("pptx-page-item-1"));
    fireEvent.click(screen.getByTestId("pptx-modify-btn-1"));
    await waitFor(() =>
      expect(useComposerInsertStore.getState().pendingText).toBe("请修改 a.pptx 的第 1 页："),
    );
  });

  it("无文本的页无展开箭头（避免死按钮），条目与摘要照常", async () => {
    mocks.outline.mockResolvedValue({
      available: true,
      slides: [{ index: 1, title: "空白页", texts: [], shapeCount: 2 }],
    });
    render(wrap(<PptxOutline relPath="a.pptx" fileName="a.pptx" onEditSlide={vi.fn()} />));
    expect(await screen.findByText("空白页")).toBeTruthy();
    expect(screen.queryByTestId("pptx-expand-1")).toBeNull();
    expect(screen.getByTestId("pptx-edit-btn-1")).toBeTruthy();
  });
});
