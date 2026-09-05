import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor, act } from "@testing-library/react";
import { FilePreview } from "./FilePreview";
import { ToastProvider } from "./Toast";
import { useComposerInsertStore } from "../lib/store";
import type { PreviewResult, PptxOutlineView } from "../lib/types";

const mocks = vi.hoisted(() => ({
  preview: null as PreviewResult | null,
  writeFile: vi.fn(async (_rel: string, _content: string) => {}),
  previewCall: vi.fn(async (_rel: string): Promise<PreviewResult> => mocks.preview ?? emptyPreview()),
  pptxOutline: vi.fn(async (_rel: string): Promise<PptxOutlineView> => ({ available: true, slides: [] })),
}));

function emptyPreview(): PreviewResult {
  return { path: "", name: "", ext: "", size: 0, kind: "text", body: "", dataUrl: "", error: "" };
}

vi.mock("../lib/bridge", () => ({
  app: {
    Preview: (rel: string) => mocks.previewCall(rel),
    WriteFile: (rel: string, content: string) => mocks.writeFile(rel, content),
    PptxOutline: (rel: string) => mocks.pptxOutline(rel),
    RevealWorkspacePath: async () => {},
    OpenWorkspacePath: async () => {},
  },
  onEvent: () => () => {},
}));

const wrap = (node: React.ReactNode) => <ToastProvider>{node}</ToastProvider>;

function markdownPreview(over: Partial<PreviewResult> = {}): PreviewResult {
  return {
    path: "notes/a.md",
    name: "a.md",
    ext: ".md",
    size: 12,
    kind: "markdown",
    body: "旧内容",
    dataUrl: "",
    error: "",
    ...over,
  };
}

beforeEach(() => {
  mocks.preview = markdownPreview();
  mocks.writeFile.mockClear();
  mocks.previewCall.mockClear();
  mocks.previewCall.mockImplementation(async (_rel: string) => mocks.preview ?? emptyPreview());
  mocks.pptxOutline.mockReset();
  mocks.pptxOutline.mockResolvedValue({ available: true, slides: [] });
  useComposerInsertStore.setState({ pendingText: null, pendingAt: null });
});

describe("FilePreview 工作区内联编辑（C5）", () => {
  it("文本/markdown 预览显示「编辑」按钮，图片预览不显示", async () => {
    const { rerender } = render(wrap(<FilePreview relPath="notes/a.md" onClose={() => {}} />));
    expect(await screen.findByText("旧内容")).toBeTruthy();
    expect(screen.getByText("编辑")).toBeTruthy();

    mocks.preview = { ...markdownPreview(), kind: "image", dataUrl: "data:image/png;base64,x", body: "" };
    rerender(wrap(<FilePreview relPath="pic.png" onClose={() => {}} />));
    await waitFor(() => expect(screen.queryByText("编辑")).toBeNull());
  });

  it("截断的预览不显示「编辑」（内容不完整，写回会丢数据）", async () => {
    mocks.preview = { ...markdownPreview(), truncated: true, size: 5 * 1024 * 1024 };
    render(wrap(<FilePreview relPath="big.md" onClose={() => {}} />));
    await screen.findByText("旧内容");
    expect(screen.queryByText("编辑")).toBeNull();
  });

  it("进入编辑 → 修改显示脏标记 → 保存调用 WriteFile 并刷新预览", async () => {
    render(wrap(<FilePreview relPath="notes/a.md" onClose={() => {}} />));
    await screen.findByText("旧内容");

    fireEvent.click(screen.getByText("编辑"));
    const textarea = await screen.findByLabelText("文本编辑");
    fireEvent.change(textarea, { target: { value: "新内容\n第二行" } });
    expect(document.querySelector(".bg-amber-400")).toBeTruthy(); // 脏标记

    // 保存时后端会重读文件 → mock 预置保存后的内容
    mocks.preview = { ...markdownPreview(), body: "新内容\n第二行" };
    fireEvent.click(screen.getByText("保存"));
    await waitFor(() => expect(mocks.writeFile).toHaveBeenCalledWith("notes/a.md", "新内容\n第二行"));
    // 保存后脏标记消失（仍停留在编辑器），预览已重读
    await waitFor(() => expect(document.querySelector(".bg-amber-400")).toBeNull());
    expect(screen.getByLabelText("文本编辑")).toBeTruthy();
  });

  it("脏状态下取消编辑弹内联确认，确认后退出编辑", async () => {
    render(wrap(<FilePreview relPath="notes/a.md" onClose={() => {}} />));
    await screen.findByText("旧内容");

    fireEvent.click(screen.getByText("编辑"));
    const textarea = await screen.findByLabelText("文本编辑");
    fireEvent.change(textarea, { target: { value: "改了一半" } });

    fireEvent.click(screen.getByText("取消"));
    expect(await screen.findByText(/有未保存的修改/)).toBeTruthy();
    fireEvent.click(screen.getByText("放弃修改"));
    await waitFor(() => expect(screen.queryByLabelText("文本编辑")).toBeNull());
  });

  it("Ctrl+S 触发保存", async () => {
    render(wrap(<FilePreview relPath="notes/a.md" onClose={() => {}} />));
    await screen.findByText("旧内容");

    fireEvent.click(screen.getByText("编辑"));
    const textarea = await screen.findByLabelText("文本编辑");
    fireEvent.change(textarea, { target: { value: "Ctrl+S 保存内容" } });

    fireEvent.keyDown(window, { key: "s", ctrlKey: true });
    await waitFor(() => expect(mocks.writeFile).toHaveBeenCalledWith("notes/a.md", "Ctrl+S 保存内容"));
  });
});

// v4.25 A3 右栏编辑器 tab 嵌入式渲染：embedded 只收窄头部（文件名由 tab 条
// 展示），预览/编辑能力原样保留；默认 false 行为完全不变（上方用例即回归）。
describe("FilePreview 嵌入式渲染（v4.25 A3 embedded）", () => {
  it("embedded：隐藏头部文件名，预览内容与编辑按钮保留", async () => {
    render(wrap(<FilePreview relPath="notes/a.md" onClose={() => {}} embedded />));
    expect(await screen.findByText("旧内容")).toBeTruthy();
    expect(screen.queryByText("a.md")).toBeNull(); // 头部文件名不重复展示
    expect(screen.getByText("编辑")).toBeTruthy(); // 编辑能力保留
    expect(screen.getByTitle("在外部程序中打开")).toBeTruthy(); // 操作区保留
  });

  it("v4.29 化繁为简：打开/定位按钮图标化（无文字，title/aria-label 保留），编辑文字保留", async () => {
    render(wrap(<FilePreview relPath="notes/a.md" onClose={() => {}} />));
    await screen.findByText("旧内容");
    const openBtn = screen.getByTitle("在外部程序中打开");
    expect(openBtn.textContent).toBe(""); // 图标化：不再占文字宽度
    expect(openBtn.getAttribute("aria-label")).toBe("在外部程序中打开");
    expect(screen.getByTitle("在文件管理器中定位").textContent).toBe("");
    expect(screen.getByText("编辑")).toBeTruthy(); // 状态语义动作保留文字
  });

  it("默认（非 embedded）：头部文件名照常展示（向后兼容）", async () => {
    render(wrap(<FilePreview relPath="notes/a.md" onClose={() => {}} />));
    await screen.findByText("旧内容");
    expect(screen.getByText("a.md")).toBeTruthy();
  });

  it("v4.30 预览两档：不传 onToggleMaximize 时不渲染按钮（向后兼容）", async () => {
    render(wrap(<FilePreview relPath="notes/a.md" onClose={() => {}} />));
    await screen.findByText("旧内容");
    expect(screen.queryByTitle("最大化预览（占满可用宽度）")).toBeNull();
    expect(screen.queryByTitle("还原半幅（占当前宽度）")).toBeNull();
  });

  it("v4.30 预览两档：半幅态显示「最大化」按钮，点击回调；最大化态切换为「还原」", async () => {
    const onToggle = vi.fn();
    const { rerender } = render(
      wrap(<FilePreview relPath="notes/a.md" onClose={() => {}} maximized={false} onToggleMaximize={onToggle} />),
    );
    await screen.findByText("旧内容");
    const maxBtn = screen.getByTitle("最大化预览（占满可用宽度）");
    expect(maxBtn.getAttribute("aria-label")).toBe("最大化预览");
    fireEvent.click(maxBtn);
    expect(onToggle).toHaveBeenCalledTimes(1);

    rerender(
      wrap(<FilePreview relPath="notes/a.md" onClose={() => {}} maximized onToggleMaximize={onToggle} />),
    );
    expect(screen.getByTitle("还原半幅（占当前宽度）")).toBeTruthy();
    expect(screen.queryByTitle("最大化预览（占满可用宽度）")).toBeNull();
  });
});

// v4.28 B2「pptx 最小交互」：GaeaPreview 返回 kind=pdf（soffice→PDF 逐页
// 缩略）时纵向铺页并叠 PptxOutline 大纲卡；点大纲页条目滚到页锚点；
// 「针对第 N 页修改」走 composer 插入通道；大纲不可用降级诚实提示。
describe("FilePreview pptx 逐页预览 + 大纲卡（v4.28 B2）", () => {
  const pptxPreview = (over: Partial<PreviewResult> = {}): PreviewResult => ({
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
  });
  const outlineView = (): PptxOutlineView => ({
    available: true,
    slides: [
      { index: 1, title: "封面", texts: ["标题行"], shapeCount: 3 },
      { index: 2, title: "数据", texts: ["营收 12%"], shapeCount: 4 },
    ],
  });

  it("渲染逐页缩略 + 大纲卡；点页条目滚动到页锚点；点修改插入 composer", async () => {
    const scrollSpy = vi.fn();
    (HTMLElement.prototype as unknown as { scrollIntoView?: unknown }).scrollIntoView = scrollSpy;
    mocks.preview = pptxPreview();
    mocks.pptxOutline.mockResolvedValue(outlineView());
    try {
      render(wrap(<FilePreview relPath="exports/汇报.pptx" onClose={() => {}} />));
      // 逐页缩略：两页 <img>（alt=第 N 页）+ 页脚标签
      expect(await screen.findByAltText("第 1 页")).toBeTruthy();
      expect(screen.getByAltText("第 2 页")).toBeTruthy();
      expect(screen.getByText("第 2 页")).toBeTruthy();
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

  it("大纲不可用（python 缺失）→ 诚实提示，逐页缩略保留", async () => {
    mocks.preview = pptxPreview();
    mocks.pptxOutline.mockResolvedValue({ available: false, error: "python-pptx 不可用", slides: [] });
    render(wrap(<FilePreview relPath="exports/汇报.pptx" onClose={() => {}} />));
    expect(await screen.findByAltText("第 1 页")).toBeTruthy();
    expect(await screen.findByText(/大纲不可用/)).toBeTruthy();
    expect(screen.getByText(/python-pptx 不可用/)).toBeTruthy();
    expect(screen.queryByTestId("pptx-page-item-1")).toBeNull();
  });

  it("无逐页缩略（pdftoppm 缺失）→ 回退整本 PDF dataUrl 内嵌查看", async () => {
    mocks.preview = pptxPreview({ pages: undefined, dataUrl: "data:application/pdf;base64,AAAA" });
    mocks.pptxOutline.mockResolvedValue(outlineView());
    render(wrap(<FilePreview relPath="exports/汇报.pptx" onClose={() => {}} />));
    expect(await screen.findByTitle("PDF 预览")).toBeTruthy();
    // 大纲卡仍在（页锚点滚动不可用，但「针对第 N 页修改」仍可插入指令）
    expect(await screen.findByText("封面")).toBeTruthy();
  });
});

// v4.33.0 C（对齐弹窗 v4.32 C）：主区 FilePreview 的 pdf 逐页路径改
// IntersectionObserver 单向懒加载——初始只挂首窗页（前 4 页），进视口
//（rootMargin 800px）才挂 <img>，已挂载页不卸载；大纲跳转目标页强制渲染真身
//（锚点立即可见，避免 scrollIntoView 落在估计高度占位盒上跳偏）。无 IO 环境
//（jsdom 默认）全量渲染 = 既有行为（上方用例即回归）。判定纯函数见
// lib/pageLazy.test.ts；FakeIntersectionObserver 模式仿 FilePreviewModal.test.tsx。
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

describe("FilePreview pdf 逐页懒加载（v4.33.0 C）", () => {
  const pptxLazyPreview = (pages: PreviewResult["pages"]): PreviewResult => ({
    path: "exports/汇报.pptx",
    name: "汇报.pptx",
    ext: ".pptx",
    size: 4096,
    kind: "pdf",
    body: "",
    dataUrl: "",
    error: "",
    hint: "outline",
    pages,
  });
  const sixPages = Array.from({ length: 6 }, (_, i) => ({
    page: i + 1,
    dataUrl: `data:image/png;base64,P${i + 1}`,
  }));

  it("无 IntersectionObserver（jsdom 默认）→ 降级全量渲染，全部页挂 img", async () => {
    delete (window as unknown as { IntersectionObserver?: unknown }).IntersectionObserver;
    FakeIntersectionObserver.instances = [];
    mocks.preview = pptxLazyPreview(sixPages);
    render(wrap(<FilePreview relPath="exports/汇报.pptx" onClose={() => {}} />));
    expect(await screen.findByAltText("第 1 页")).toBeTruthy();
    for (let i = 1; i <= 6; i++) {
      expect(screen.getByAltText(`第 ${i} 页`)).toBeTruthy();
    }
    expect(document.querySelector("[data-pptx-page='6'] img")).toBeTruthy();
    expect(FakeIntersectionObserver.instances).toHaveLength(0);
  });

  it("IO 懒加载：初始仅首窗页（前 4 页）挂 img，进视口后挂载且已挂载页不卸载", async () => {
    (window as unknown as { IntersectionObserver?: unknown }).IntersectionObserver = FakeIntersectionObserver;
    FakeIntersectionObserver.instances = [];
    mocks.preview = pptxLazyPreview(sixPages);
    try {
      render(wrap(<FilePreview relPath="exports/汇报.pptx" onClose={() => {}} />));
      expect(await screen.findByAltText("第 1 页")).toBeTruthy();
      // 初始窗口（前 4 页）有 img，窗口外（5/6）还是占位盒（figure 锚点保留）
      for (let i = 1; i <= 4; i++) expect(screen.getByAltText(`第 ${i} 页`)).toBeTruthy();
      expect(screen.queryByAltText("第 5 页")).toBeNull();
      expect(document.querySelector("[data-pptx-page='5'] img")).toBeNull();
      expect(document.querySelector("[data-pptx-page='5']")).toBeTruthy();
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
    mocks.preview = pptxLazyPreview(eightPages);
    mocks.pptxOutline.mockResolvedValue({
      available: true,
      slides: [
        { index: 1, title: "封面", texts: [], shapeCount: 1 },
        { index: 7, title: "附录", texts: [], shapeCount: 2 },
      ],
    });
    const scrollSpy = vi.fn();
    (HTMLElement.prototype as unknown as { scrollIntoView?: unknown }).scrollIntoView = scrollSpy;
    try {
      render(wrap(<FilePreview relPath="exports/汇报.pptx" onClose={() => {}} />));
      expect(await screen.findByAltText("第 1 页")).toBeTruthy();
      // 初始窗口外：第 7 页还是占位盒，无 img
      expect(screen.queryByAltText("第 7 页")).toBeNull();
      // 等大纲卡加载出第 7 页条目 → 点页条目跳转
      expect(await screen.findByText("附录")).toBeTruthy();
      fireEvent.click(screen.getByTestId("pptx-page-item-7"));
      // 目标页真身立即可渲染（强制渲染集合，无需 IO 触发），且触发了滚动
      expect(await screen.findByAltText("第 7 页")).toBeTruthy();
      expect(scrollSpy).toHaveBeenCalled();
      // 其余窗口外页（第 8 页）仍是占位盒
      expect(screen.queryByAltText("第 8 页")).toBeNull();
    } finally {
      delete (HTMLElement.prototype as unknown as { scrollIntoView?: unknown }).scrollIntoView;
      delete (window as unknown as { IntersectionObserver?: unknown }).IntersectionObserver;
    }
  });
});

describe("FilePreview HTML 沙箱预览（1c）", () => {
  function htmlPreview(over: Partial<PreviewResult> = {}): PreviewResult {
    return {
      path: "reports/r.html",
      name: "r.html",
      ext: ".html",
      size: 120,
      kind: "html",
      body: "<html><body><h1>季报</h1></body></html>",
      dataUrl: "",
      error: "",
      ...over,
    };
  }

  beforeEach(() => {
    mocks.preview = htmlPreview();
  });

  it("kind=html 渲染沙箱 iframe（无同源 + 原文 srcDoc）与沙箱标注条", async () => {
    render(wrap(<FilePreview relPath="reports/r.html" onClose={() => {}} />));
    const iframe = await screen.findByTestId("sandboxed-html");
    expect(iframe.getAttribute("sandbox")).toBe("allow-scripts");
    // 无 allow-same-origin：通过属性串断言（Chromium csp 属性经 attribute 落 DOM）
    expect(iframe.getAttribute("csp")).toContain("default-src 'none'");
    expect(iframe.getAttribute("srcdoc")).toContain("<h1>季报</h1>");
    expect(screen.getByTestId("sandbox-html-note")).toBeTruthy();
  });

  it("截断的 html 显示截断提示", async () => {
    mocks.preview = htmlPreview({ truncated: true });
    render(wrap(<FilePreview relPath="reports/r.html" onClose={() => {}} />));
    await screen.findByTestId("sandboxed-html");
    expect(screen.getByText(/预览已截断/)).toBeTruthy();
  });
});

describe("FilePreview markdown 导图视图（M1）", () => {
  beforeEach(() => {
    localStorage.removeItem("gaea.preview.mdView");
  });

  it("markdown 预览出现文档/导图切换；点导图渲染交互节点，可切回", async () => {
    mocks.preview = markdownPreview({ body: "# 标题\n- 甲\n- 乙\n" });
    render(wrap(<FilePreview relPath="notes/a.md" onClose={() => {}} />));
    await screen.findByText("标题");
    fireEvent.click(screen.getByTestId("md-view-mindmap"));
    expect(await screen.findByTestId("mind-map")).toBeTruthy();
    expect(screen.getByText("甲")).toBeTruthy();
    // 编辑能力保留红线：导图态「编辑」入口常驻（点击进文本编辑器）
    expect(screen.getByText("编辑")).toBeTruthy();
    fireEvent.click(screen.getByTestId("md-view-doc"));
    await waitFor(() => expect(screen.queryByTestId("mind-map")).toBeNull());
  });

  it("导图偏好持久化：切到导图后重挂载仍为导图态", async () => {
    mocks.preview = markdownPreview({ body: "# 标题\n- 甲\n" });
    const { unmount } = render(wrap(<FilePreview relPath="notes/a.md" onClose={() => {}} />));
    await screen.findByText("标题");
    fireEvent.click(screen.getByTestId("md-view-mindmap"));
    expect(await screen.findByTestId("mind-map")).toBeTruthy();
    unmount();
    render(wrap(<FilePreview relPath="notes/b.md" onClose={() => {}} />));
    await waitFor(() => expect(screen.queryByTestId("mind-map")).not.toBeNull());
    localStorage.removeItem("gaea.preview.mdView");
  });

  it("text 等非 markdown 文件不显示视图切换", async () => {
    mocks.preview = { ...markdownPreview(), kind: "text" };
    render(wrap(<FilePreview relPath="notes/a.txt" onClose={() => {}} />));
    await screen.findByText("旧内容");
    expect(screen.queryByTestId("md-view-mindmap")).toBeNull();
  });
});

// U4 写后预览实时跟随（docs/gaea-u4-render-evidence-inventory-2026-09.md §3）：
// reloadSignal 递增（App 从 office 写类工具成功回执派生，800ms 防抖后递增
// paneTabs.reloadTicks）→ 静默重读 app.Preview：不重挂、不进 loading、滚动位
// 保持；编辑态跳过（绝不拿旧草稿覆盖 agent 新写入的内容）；未接线调用方
// （不传 reloadSignal）行为完全不变。
describe("FilePreview U4 写后预览实时跟随（reloadSignal 静默重载）", () => {
  it("reloadSignal 递增 → 重读 app.Preview、内容更新并亮「已自动刷新」徽标", async () => {
    const { rerender } = render(
      wrap(<FilePreview relPath="notes/a.md" onClose={() => {}} reloadSignal={0} />),
    );
    await screen.findByText("旧内容");
    expect(mocks.previewCall).toHaveBeenCalledTimes(1);
    expect(screen.queryByTestId("office-auto-refreshed")).toBeNull();

    mocks.preview = markdownPreview({ body: "新内容", size: 15 });
    rerender(wrap(<FilePreview relPath="notes/a.md" onClose={() => {}} reloadSignal={1} />));
    expect(await screen.findByText("新内容")).toBeTruthy();
    expect(mocks.previewCall).toHaveBeenCalledTimes(2); // 静默重读恰好一次
    expect(screen.getByTestId("office-auto-refreshed")).toBeTruthy();
    // 刷新是内容更新而非重新打开：无 loading 占位（旧内容被新内容原位替换）
    expect(screen.queryByText("加载中…")).toBeNull();
  });

  it("同序号重渲染不重读（reloadSignal 未变 = 无新信号）", async () => {
    const { rerender } = render(
      wrap(<FilePreview relPath="notes/a.md" onClose={() => {}} reloadSignal={3} />),
    );
    await screen.findByText("旧内容");
    expect(mocks.previewCall).toHaveBeenCalledTimes(1);
    rerender(wrap(<FilePreview relPath="notes/a.md" onClose={() => {}} reloadSignal={3} />));
    await act(async () => { await Promise.resolve(); });
    expect(mocks.previewCall).toHaveBeenCalledTimes(1);
  });

  it("编辑态跳过刷新（不打断输入，退出后下一次信号仍跟随）", async () => {
    const { rerender } = render(
      wrap(<FilePreview relPath="notes/a.md" onClose={() => {}} reloadSignal={0} />),
    );
    await screen.findByText("旧内容");
    fireEvent.click(screen.getByText("编辑")); // 进入编辑态
    mocks.preview = markdownPreview({ body: "新内容" });
    rerender(wrap(<FilePreview relPath="notes/a.md" onClose={() => {}} reloadSignal={1} />));
    await act(async () => { await Promise.resolve(); });
    expect(mocks.previewCall).toHaveBeenCalledTimes(1); // 编辑中不重读
    expect(screen.queryByTestId("office-auto-refreshed")).toBeNull();
  });

  it("未接线（reloadSignal 缺省）行为不变：重渲染不触发额外重读", async () => {
    const { rerender } = render(wrap(<FilePreview relPath="notes/a.md" onClose={() => {}} />));
    await screen.findByText("旧内容");
    expect(mocks.previewCall).toHaveBeenCalledTimes(1);
    rerender(wrap(<FilePreview relPath="notes/a.md" onClose={() => {}} />));
    await act(async () => { await Promise.resolve(); });
    expect(mocks.previewCall).toHaveBeenCalledTimes(1);
  });
});
