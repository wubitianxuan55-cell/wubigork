import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor, act } from "@testing-library/react";
import { FilePreviewModal } from "./FilePreviewModal";
import { usePreviewStore, useComposerInsertStore } from "../lib/store";
import type { PreviewResult, PptxOutlineView } from "../lib/types";

// 既有用例依赖浏览器开发 mock（README.md/big.pdf/scan.pdf/docs/方案.docx 的
// 预设行为），v4.31 B 新增的 kind=pdf（.pptx 逐页预览）用例需要注入 pages——
// 浏览器 mock 的 GaeaPreview 永不返回 kind="pdf"。因此这里 mock bridge：
// app 委托给真实 makeMockApp()，仅 Preview/PptxOutline 两个方法可被测试覆写
// （mocks.preview 为 null 时 Preview 原样委托，既有用例零行为变化；
// onEvent 走真实 mockSubscribe，scan.pdf 的 OCR 进度事件照常送达）。
const mocks = vi.hoisted(() => ({
  preview: null as PreviewResult | null,
  pptxOutline: vi.fn(async (_rel: string): Promise<PptxOutlineView> => ({ available: true, slides: [] })),
}));

vi.mock("../lib/bridge", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../lib/bridge")>();
  const { makeMockApp } = await import("../lib/mock");
  const base = makeMockApp();
  return {
    ...actual,
    app: new Proxy(base, {
      get(t, prop) {
        if (prop === "Preview") {
          return (rel: string) => (mocks.preview ? Promise.resolve(mocks.preview) : base.Preview(rel));
        }
        if (prop === "PptxOutline") return (rel: string) => mocks.pptxOutline(rel);
        const v = (t as unknown as Record<string, unknown>)[String(prop)];
        return typeof v === "function" ? (v as (...a: unknown[]) => unknown).bind(t) : v;
      },
    }),
  };
});

// kind=pdf 的 .pptx 预览负载（对齐 FilePreview.test.tsx 的 pptxPreview 构造）。
function pptxPreview(over: Partial<PreviewResult> = {}): PreviewResult {
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
    pages: [
      { page: 1, dataUrl: "data:image/png;base64,AAA1" },
      { page: 2, dataUrl: "data:image/png;base64,AAA2" },
    ],
    ...over,
  };
}

function outlineView(): PptxOutlineView {
  return {
    available: true,
    slides: [
      { index: 1, title: "封面", texts: ["标题行"], shapeCount: 3 },
      { index: 2, title: "数据", texts: ["营收 12%"], shapeCount: 4 },
    ],
  };
}

beforeEach(() => {
  mocks.preview = null;
  mocks.pptxOutline.mockReset();
  mocks.pptxOutline.mockResolvedValue({ available: true, slides: [] });
  useComposerInsertStore.setState({ pendingText: null, pendingAt: null });
  usePreviewStore.setState({ previewFile: null });
});

describe("FilePreviewModal", () => {
  it("打开预览时调用 Preview 并渲染 Markdown", async () => {
    usePreviewStore.setState({ previewFile: "README.md" });
    render(<FilePreviewModal />);
    expect(await screen.findByText(/Browser-dev workspace preview/)).toBeTruthy();
    expect(screen.getAllByText(/README\.md/).length).toBeGreaterThanOrEqual(1);
  });

  it("Esc 关闭弹层", async () => {
    usePreviewStore.setState({ previewFile: "README.md" });
    render(<FilePreviewModal />);
    await screen.findByText(/Browser-dev workspace preview/);
    fireEvent.keyDown(window, { key: "Escape" });
    expect(usePreviewStore.getState().previewFile).toBeNull();
  });

  it("无文件时不渲染", () => {
    usePreviewStore.setState({ previewFile: null });
    const { container } = render(<FilePreviewModal />);
    expect(container.firstChild).toBeNull();
  });

  it("PDF 预览页数截断时显示提示", async () => {
    usePreviewStore.setState({ previewFile: "big.pdf" });
    render(<FilePreviewModal />);
    expect(await screen.findByText(/仅展示前部内容/)).toBeTruthy();
    expect(screen.getAllByText(/1200/).length).toBeGreaterThanOrEqual(1);
  });

  it("扫描件 PDF 预览显示 OCR 逐页进度", async () => {
    usePreviewStore.setState({ previewFile: "scan.pdf" });
    render(<FilePreviewModal />);
    expect(await screen.findByText(/OCR 识别中/)).toBeTruthy();
    expect(await screen.findByText(/扫描页内容/)).toBeTruthy();
  });

  it("可转换格式显示导出 PDF 按钮，点击后打开 PDF 预览（mock）", async () => {
    usePreviewStore.setState({ previewFile: "docs/方案.docx" });
    render(<FilePreviewModal />);
    const btn = await screen.findByTitle(/LibreOffice 把当前文档转换为 PDF/);
    fireEvent.click(btn);
    // mock ConvertToPdf 返回 .gaea/exports/方案-mock.pdf → 自动切到该预览
    expect(await screen.findByText(/方案-mock\.pdf/)).toBeTruthy();
  });

  it("PDF 本身与图片不显示导出 PDF 按钮", async () => {
    usePreviewStore.setState({ previewFile: "big.pdf" });
    render(<FilePreviewModal />);
    await screen.findByText(/仅展示前部内容/);
    expect(screen.queryByTitle(/LibreOffice 把当前文档转换为 PDF/)).toBeNull();
  });
});

// v4.31 B（收 v4.28 欠账）：弹窗路径的 kind=pdf（.pptx → soffice PDF）不再只
// 显示「演示文稿」标签，而是逐页缩略 + PptxOutline 大纲卡 + 页锚点滚动 +
// 「针对第 N 页修改」composer 插入（与 FilePreview v4.28 B2 同能力）。
describe("FilePreviewModal pdf 逐页预览 + 大纲卡（v4.31 B）", () => {
  it("渲染逐页缩略 + 大纲卡；点页条目滚动到页锚点；点修改插入 composer", async () => {
    const scrollSpy = vi.fn();
    (HTMLElement.prototype as unknown as { scrollIntoView?: unknown }).scrollIntoView = scrollSpy;
    mocks.preview = pptxPreview();
    mocks.pptxOutline.mockResolvedValue(outlineView());
    usePreviewStore.setState({ previewFile: "exports/汇报.pptx" });
    try {
      render(<FilePreviewModal />);
      // 逐页缩略：两页 <img>（alt=第 N 页）+ 页脚标签 + 头部「演示文稿」徽标
      expect(await screen.findByAltText("第 1 页")).toBeTruthy();
      expect(screen.getByAltText("第 2 页")).toBeTruthy();
      expect(screen.getByText("第 2 页")).toBeTruthy();
      expect(screen.getByText("演示文稿")).toBeTruthy();
      // 大纲卡拉取并渲染
      expect(await screen.findByText("封面")).toBeTruthy();
      expect(screen.getByText("数据")).toBeTruthy();
      expect(mocks.pptxOutline).toHaveBeenCalledWith("exports/汇报.pptx");

      // 点大纲页条目 → 滚动到对应页锚点
      fireEvent.click(screen.getByTestId("pptx-page-item-2"));
      expect(scrollSpy).toHaveBeenCalled();

      // 「针对第 N 页修改」→ composer 插入模板（不自动发送）
      fireEvent.click(screen.getByTestId("pptx-modify-btn-2"));
      await waitFor(() =>
        expect(useComposerInsertStore.getState().pendingText).toBe("请修改 汇报.pptx 的第 2 页："),
      );
    } finally {
      delete (HTMLElement.prototype as unknown as { scrollIntoView?: unknown }).scrollIntoView;
    }
  });

  it("无逐页缩略（pdftoppm 缺失）→ 回退整本 PDF dataUrl 内嵌查看", async () => {
    mocks.preview = pptxPreview({ pages: undefined, dataUrl: "data:application/pdf;base64,AAAA" });
    mocks.pptxOutline.mockResolvedValue(outlineView());
    usePreviewStore.setState({ previewFile: "exports/汇报.pptx" });
    render(<FilePreviewModal />);
    expect(await screen.findByTitle("PDF 预览")).toBeTruthy();
    // 大纲卡仍在（页锚点滚动不可用，但「针对第 N 页修改」仍可插入指令）
    expect(await screen.findByText("封面")).toBeTruthy();
  });

  it("无逐页缩略且无整本 dataUrl → 诚实提示，不回退「外部打开」外显", async () => {
    mocks.preview = pptxPreview({ pages: undefined, dataUrl: "" });
    mocks.pptxOutline.mockResolvedValue(outlineView());
    usePreviewStore.setState({ previewFile: "exports/汇报.pptx" });
    render(<FilePreviewModal />);
    expect(await screen.findByText(/无可渲染的页面内容/)).toBeTruthy();
    expect(screen.getByText(/summarize_file/)).toBeTruthy();
  });
});

// v4.32 C（收 v4.31 欠账「弹窗 pdf 不虚拟化」）：逐页路径改 IntersectionObserver
// 单向懒加载——初始只挂首窗页，进视口（rootMargin 800px）才挂 <img>，已挂载
// 页不卸载；无 IO 环境（jsdom 默认）全量渲染 = 既有行为；大纲跳转目标页强制
// 渲染真身（锚点立即可见，避免 scrollIntoView 落在估计高度占位盒上跳偏）。
// 判定纯函数（初始窗口 / buffer 扩张 / 强制集合合并）见 lib/pageLazy.test.ts。
class FakeIntersectionObserver {
  static instances: FakeIntersectionObserver[] = [];
  callback: IntersectionObserverCallback;
  options: IntersectionObserverInit | undefined;
  observed = new Map<number, Element>();
  constructor(callback: IntersectionObserverCallback, options?: IntersectionObserverInit) {
    this.callback = callback;
    this.options = options;
    FakeIntersectionObserver.instances.push(this);
  }
  observe(el: Element) {
    const page = Number(el.getAttribute("data-pptx-page"));
    if (page > 0) this.observed.set(page, el);
  }
  unobserve(el: Element) {
    this.observed.delete(Number(el.getAttribute("data-pptx-page")));
  }
  disconnect() {
    this.observed.clear();
  }
  /** 模拟这些页进入视口（含 rootMargin 范围），手动驱动 IO 回调。 */
  enterView(pages: number[]) {
    const entries = pages
      .map((p) => this.observed.get(p))
      .filter((el): el is Element => !!el)
      .map((el) => ({ isIntersecting: true, target: el }));
    if (entries.length > 0) {
      this.callback(entries as unknown as IntersectionObserverEntry[], this as unknown as IntersectionObserver);
    }
  }
}

describe("FilePreviewModal pdf 逐页懒加载（v4.32 C）", () => {
  const sixPages = Array.from({ length: 6 }, (_, i) => ({
    page: i + 1,
    dataUrl: `data:image/png;base64,P${i + 1}`,
  }));

  it("无 IntersectionObserver（jsdom 默认）→ 降级全量渲染，全部页挂 img", async () => {
    delete (window as unknown as { IntersectionObserver?: unknown }).IntersectionObserver;
    FakeIntersectionObserver.instances = [];
    mocks.preview = pptxPreview({ pages: sixPages });
    usePreviewStore.setState({ previewFile: "exports/汇报.pptx" });
    render(<FilePreviewModal />);
    expect(await screen.findByAltText("第 1 页")).toBeTruthy();
    for (let i = 1; i <= 6; i++) {
      expect(screen.getByAltText(`第 ${i} 页`)).toBeTruthy();
    }
    expect(document.querySelector("[data-pptx-page='6'] img")).toBeTruthy();
    expect(FakeIntersectionObserver.instances).toHaveLength(0);
  });

  it("IO 懒加载：初始仅首窗页挂 img（rootMargin 800px），进视口后挂载且已挂载页不卸载", async () => {
    (window as unknown as { IntersectionObserver?: unknown }).IntersectionObserver = FakeIntersectionObserver;
    FakeIntersectionObserver.instances = [];
    mocks.preview = pptxPreview({ pages: sixPages });
    usePreviewStore.setState({ previewFile: "exports/汇报.pptx" });
    try {
      render(<FilePreviewModal />);
      expect(await screen.findByAltText("第 1 页")).toBeTruthy();
      // 初始窗口（前 4 页）有 img，窗口外（5/6）还是占位盒
      for (let i = 1; i <= 4; i++) expect(screen.getByAltText(`第 ${i} 页`)).toBeTruthy();
      expect(screen.queryByAltText("第 5 页")).toBeNull();
      expect(document.querySelector("[data-pptx-page='5'] img")).toBeNull();
      // rootMargin 800px：视口外提前预挂，抵消快速滚动白屏
      const io = FakeIntersectionObserver.instances[0];
      expect(io.options?.rootMargin).toBe("800px");
      // 第 5 页进入视口 → 挂 img（buffer 邻页第 6 页一并挂载），
      // 已挂载的第 1 页不卸载（单向懒加载）
      act(() => io.enterView([5]));
      expect(await screen.findByAltText("第 5 页")).toBeTruthy();
      expect(screen.getByAltText("第 6 页")).toBeTruthy();
      expect(screen.getByAltText("第 1 页")).toBeTruthy();
      expect(document.querySelectorAll("[data-pptx-page] img")).toHaveLength(6);
    } finally {
      delete (window as unknown as { IntersectionObserver?: unknown }).IntersectionObserver;
    }
  });

  it("IO 懒加载：大纲卡跳转目标页强制渲染真身（不经 IO 触发）", async () => {
    (window as unknown as { IntersectionObserver?: unknown }).IntersectionObserver = FakeIntersectionObserver;
    FakeIntersectionObserver.instances = [];
    const eightPages = Array.from({ length: 8 }, (_, i) => ({
      page: i + 1,
      dataUrl: `data:image/png;base64,P${i + 1}`,
    }));
    mocks.preview = pptxPreview({ pages: eightPages });
    mocks.pptxOutline.mockResolvedValue({
      available: true,
      slides: [
        { index: 1, title: "封面", texts: [], shapeCount: 1 },
        { index: 7, title: "附录", texts: [], shapeCount: 2 },
      ],
    });
    usePreviewStore.setState({ previewFile: "exports/汇报.pptx" });
    try {
      render(<FilePreviewModal />);
      expect(await screen.findByAltText("第 1 页")).toBeTruthy();
      // 初始窗口外：第 7 页还是占位盒，无 img
      expect(screen.queryByAltText("第 7 页")).toBeNull();
      // 等大纲卡加载出第 7 页条目 → 点「针对第 7 页」滚动条目
      expect(await screen.findByText("附录")).toBeTruthy();
      fireEvent.click(screen.getByTestId("pptx-page-item-7"));
      // 目标页真身立即可渲染（强制渲染集合，无需 IO 触发）
      expect(await screen.findByAltText("第 7 页")).toBeTruthy();
    } finally {
      delete (window as unknown as { IntersectionObserver?: unknown }).IntersectionObserver;
    }
  });
});
