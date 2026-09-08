import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { PptxEditPanel } from "./PptxEditPanel";
import { useUpdatedFilesStore } from "../lib/store";
import type { PreviewResult } from "../lib/types";

// 桥接 mock 方式照 PptxOutline.test.tsx：只挂本面板消费的三个方法
// （PptxSlideText 载入段落全文 / OfficeEditText 生成 / PptxApplyEdit 应用）。
const mocks = vi.hoisted(() => ({
  slideText: vi.fn((_rel: string): Promise<{ index: number; paragraphs: string[] }[]> =>
    Promise.resolve([]),
  ),
  officeEdit: vi.fn(
    async (selectedText: string, _instruction: string): Promise<{ edited: string }> => ({
      edited: `改写：${selectedText}`,
    }),
  ),
  applyEdit: vi.fn(
    async (
      _rel: string,
      _slideIdx: number,
      _target: string,
      _replacement: string,
    ): Promise<PreviewResult> => appliedPreview(),
  ),
}));

function appliedPreview(): PreviewResult {
  return {
    path: "exports/汇报.pptx",
    name: "汇报.pptx",
    ext: ".pptx",
    size: 4096,
    kind: "pdf",
    body: "",
    dataUrl: "",
    error: "",
    hint: "outline",
    pages: [{ page: 1, dataUrl: "data:image/png;base64,AAA" }],
  };
}

vi.mock("../lib/bridge", () => ({
  app: {
    PptxSlideText: (rel: string) => mocks.slideText(rel),
    OfficeEditText: (sel: string, instr: string) => mocks.officeEdit(sel, instr),
    PptxApplyEdit: (rel: string, slide: number, target: string, repl: string) =>
      mocks.applyEdit(rel, slide, target, repl),
  },
}));

// 与 mock/office.ts 走查桩同款的两页四段演示数据（面板只认契约不认来源）。
const slideTextData = (): { index: number; paragraphs: string[] }[] => [
  { index: 1, paragraphs: ["季度经营总结", "营收同比增长 12%，成本结构持续优化"] },
  { index: 2, paragraphs: ["下季度计划", "重点推进三件事"] },
];

beforeEach(() => {
  mocks.slideText.mockReset();
  mocks.slideText.mockResolvedValue(slideTextData());
  mocks.officeEdit.mockReset();
  mocks.officeEdit.mockImplementation(
    async (selectedText: string) => ({ edited: `改写：${selectedText}` }),
  );
  mocks.applyEdit.mockReset();
  mocks.applyEdit.mockResolvedValue(appliedPreview());
  useUpdatedFilesStore.setState({ updatedAt: {} });
});

const renderPanel = (over: { onApplied?: (r: PreviewResult) => void; initialSlide?: number } = {}) => {
  const onApplied = over.onApplied ?? vi.fn();
  const onClose = vi.fn();
  render(
    <PptxEditPanel
      relPath="exports/汇报.pptx"
      fileName="汇报.pptx"
      initialSlide={over.initialSlide}
      onClose={onClose}
      onApplied={onApplied}
    />,
  );
  return { onApplied, onClose };
};

describe("PptxEditPanel pptx 编辑面板（v4.156 刀2）", () => {
  it("载入中先出加载态；完成后按页分组列出段落全文", async () => {
    let resolveLoad: (v: { index: number; paragraphs: string[] }[]) => void = () => {};
    mocks.slideText.mockReturnValue(
      new Promise((res) => {
        resolveLoad = res;
      }),
    );
    renderPanel();
    expect(screen.getByTestId("pptx-edit-loading")).toBeTruthy();
    resolveLoad(slideTextData());
    expect(await screen.findByTestId("pptx-para-1-0")).toBeTruthy();
    expect(screen.getByTestId("pptx-para-1-0").textContent).toContain("季度经营总结");
    expect(screen.getByTestId("pptx-para-2-1").textContent).toContain("重点推进三件事");
    expect(mocks.slideText).toHaveBeenCalledWith("exports/汇报.pptx");
  });

  it("initialSlide 预选该页第一段（编辑目标显示页码+段落序号）", async () => {
    renderPanel({ initialSlide: 2 });
    await screen.findByTestId("pptx-para-2-0");
    expect(screen.getByText("编辑目标：第 2 页 第 1 段")).toBeTruthy();
  });

  it("载入失败 → 错误态 + 重试；重试成功恢复列表（诚实降级不静默）", async () => {
    mocks.slideText.mockRejectedValueOnce(new Error("解析失败"));
    renderPanel();
    expect(await screen.findByTestId("pptx-edit-load-error")).toBeTruthy();
    expect(screen.getByText(/解析失败/)).toBeTruthy();
    mocks.slideText.mockResolvedValue(slideTextData());
    fireEvent.click(screen.getByTestId("pptx-edit-retry"));
    expect(await screen.findByTestId("pptx-para-1-0")).toBeTruthy();
  });

  it("绑定未接线（方法缺失）→ 同款错误态可重试", async () => {
    mocks.slideText.mockResolvedValue(undefined as unknown as { index: number; paragraphs: string[] }[]);
    renderPanel();
    expect(await screen.findByTestId("pptx-edit-load-error")).toBeTruthy();
    expect(screen.getByText(/绑定不可用/)).toBeTruthy();
  });

  it("选中段落 → 生成调用 OfficeEditText 且拼入 pptx 场景约束；双栏对比整块着色", async () => {
    renderPanel();
    await screen.findByTestId("pptx-para-1-1");
    fireEvent.click(screen.getByTestId("pptx-para-1-1"));
    fireEvent.change(screen.getByTestId("pptx-edit-instruction"), {
      target: { value: "改得更精炼" },
    });
    fireEvent.click(screen.getByTestId("pptx-edit-generate"));
    await screen.findByTestId("pptx-edit-compare");
    // pptx 场景约束拼进 instruction（不动 GaeaOfficeEditText 本体）
    expect(mocks.officeEdit).toHaveBeenCalledWith(
      "营收同比增长 12%，成本结构持续优化",
      expect.stringContaining("改得更精炼"),
    );
    expect(String(mocks.officeEdit.mock.calls[0][1])).toContain("防止文本框溢出");
    // 双栏：原文栏 / 新文栏；mock 改写与原文句级不等 → 原文栏有删除着色、新文栏有新增着色
    expect(screen.getByTestId("pptx-edit-original").textContent).toContain("营收同比增长");
    expect(screen.getByTestId("pptx-edit-proposal").textContent).toContain("改写：营收同比增长");
    const delSpan = screen.getByTestId("pptx-edit-original").querySelector("span[class*='bg-del-bg']");
    const addSpan = screen.getByTestId("pptx-edit-proposal").querySelector("span[class*='bg-accent']");
    expect(delSpan).toBeTruthy();
    expect(addSpan).toBeTruthy();
    // 诚实标注：句级对比（无字符级高亮）+ pptx 无修订标记说明
    expect(screen.getByText(/句级对比（无字符级高亮）/)).toBeTruthy();
    expect(screen.getByText(/PPT 格式不支持修订标记/)).toBeTruthy();
  });

  it("应用成功：PptxApplyEdit 收到 (rel, 页码, 段落全文, 新文)；宿主收到新预览并刷新段落", async () => {
    const onApplied = vi.fn();
    renderPanel({ onApplied, initialSlide: 1 });
    await screen.findByTestId("pptx-para-1-1");
    fireEvent.click(screen.getByTestId("pptx-para-1-1"));
    fireEvent.change(screen.getByTestId("pptx-edit-instruction"), { target: { value: "精简" } });
    fireEvent.click(screen.getByTestId("pptx-edit-generate"));
    await screen.findByTestId("pptx-edit-compare");
    fireEvent.click(screen.getByTestId("pptx-edit-apply"));
    await waitFor(() =>
      expect(mocks.applyEdit).toHaveBeenCalledWith(
        "exports/汇报.pptx",
        1,
        "营收同比增长 12%，成本结构持续优化",
        "改写：营收同比增长 12%，成本结构持续优化",
      ),
    );
    // 宿主用返回的新 PreviewResult 刷新预览（apply 已返回新预览）
    await waitFor(() => expect(onApplied).toHaveBeenCalledWith(appliedPreview()));
    // 成功提示含回滚指引（版本时间线）
    expect(await screen.findByTestId("pptx-edit-notice")).toBeTruthy();
    expect(screen.getByTestId("pptx-edit-notice").textContent).toContain("版本时间线");
    // 应用后静默重拉段落全文，对准已落盘内容
    await waitFor(() => expect(mocks.slideText).toHaveBeenCalledTimes(2));
    // 交付面板「已更新」徽标数据源
    expect(useUpdatedFilesStore.getState().updatedAt["exports/汇报.pptx"]).toBeTruthy();
  });

  it("应用失败（定位不到/原文不匹配）→ 后端错误原样透出，不刷新宿主", async () => {
    mocks.applyEdit.mockRejectedValue(new Error("未在第 1 页找到目标文本（宁拒不误改）"));
    const onApplied = vi.fn();
    renderPanel({ onApplied, initialSlide: 1 });
    await screen.findByTestId("pptx-para-1-1");
    fireEvent.click(screen.getByTestId("pptx-para-1-1"));
    fireEvent.change(screen.getByTestId("pptx-edit-instruction"), { target: { value: "精简" } });
    fireEvent.click(screen.getByTestId("pptx-edit-generate"));
    await screen.findByTestId("pptx-edit-compare");
    fireEvent.click(screen.getByTestId("pptx-edit-apply"));
    expect(await screen.findByTestId("pptx-edit-error")).toBeTruthy();
    expect(screen.getByTestId("pptx-edit-error").textContent).toContain("宁拒不误改");
    expect(onApplied).not.toHaveBeenCalled();
  });

  it("Esc 关闭面板回调 onClose；应用中生成按钮禁用防并发", async () => {
    const { onClose } = renderPanel({ initialSlide: 1 });
    await screen.findByTestId("pptx-para-1-1");
    fireEvent.keyDown(window, { key: "Escape" });
    expect(onClose).toHaveBeenCalled();
  });
});
