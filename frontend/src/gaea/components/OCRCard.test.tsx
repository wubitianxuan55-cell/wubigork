// OCRCard — 文件识别作业卡：进度可访问性、文件类型、终态与失败恢复提示。
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { LocaleProvider } from "../lib/i18n";
import type { OCRItem } from "./OCRCard";
import { OCRCard } from "./OCRCard";

const renderT = (ui: React.ReactElement) => {
  localStorage.setItem("gaea-lang", "zh");
  return render(<LocaleProvider>{ui}</LocaleProvider>);
};

const base: OCRItem = {
  kind: "ocr",
  id: "ocr:C:/x/扫描件.pdf",
  path: "C:/x/扫描件.pdf",
  name: "扫描件.pdf",
  engine: "uocr2",
  stage: "running",
  done: 3,
  total: 20,
  chars: 0,
  err: "",
};

describe("OCRCard 文件识别作业卡", () => {
  afterEach(cleanup);

  it("PDF 识别中：文件身份、页级进度与可访问进度值完整", () => {
    const { container } = renderT(<OCRCard item={base} />);
    const card = screen.getByTestId("ocr-card");
    expect(card.getAttribute("data-file-kind")).toBe("pdf");
    expect(card.getAttribute("aria-busy")).toBe("true");
    expect(screen.getByText("文字识别")).toBeTruthy();
    expect(screen.getByRole("heading", { level: 3, name: "扫描件.pdf" })).toBeTruthy();
    expect(screen.getByLabelText("识别引擎：uocr2")).toBeTruthy();
    expect(screen.getByRole("status").textContent).toContain("识别中");
    expect(screen.getByText("正在识别第 3 页，共 20 页")).toBeTruthy();
    expect(screen.getByText("15%")).toBeTruthy();
    expect(screen.getByText("3/20 页")).toBeTruthy();
    const progress = screen.getByRole("progressbar", { name: "文字识别进度：3/20 页" });
    expect(progress.getAttribute("aria-valuemin")).toBe("0");
    expect(progress.getAttribute("aria-valuemax")).toBe("20");
    expect(progress.getAttribute("aria-valuenow")).toBe("3");
    const fill = container.querySelector(".ocr-card-bar > i") as HTMLElement;
    expect(fill.style.transform).toBe("scaleX(0.15)");
  });

  it("引擎拉起态：说明冷启动成本，不伪造页进度", () => {
    renderT(<OCRCard item={{ ...base, stage: "engine-start", done: 0 }} />);
    expect(screen.getAllByText("引擎拉起中").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText(/冷启动可能持续数分钟/)).toBeTruthy();
    expect(screen.queryByRole("progressbar")).toBeNull();
  });

  it("单图识别中：给出正在提取文字的明确反馈", () => {
    const { container } = renderT(<OCRCard item={{ ...base, name: "签证单.png", total: 1, done: 0 }} />);
    expect((container.querySelector('[data-testid="ocr-card"]') as HTMLElement).getAttribute("data-file-kind")).toBe("image");
    expect(screen.getByText("正在提取图像中的文字")).toBeTruthy();
    expect(screen.queryByRole("progressbar")).toBeNull();
  });

  it("单页 PDF 仍按页面语义反馈，不误标为图片", () => {
    const { container } = renderT(<OCRCard item={{ ...base, name: "一页合同.pdf", total: 1, done: 0 }} />);
    expect((container.querySelector('[data-testid="ocr-card"]') as HTMLElement).getAttribute("data-file-kind")).toBe("pdf");
    expect(screen.getByText("正在提取页面中的文字")).toBeTruthy();
    expect(screen.queryByText("正在提取图像中的文字")).toBeNull();
  });

  it("缓存命中：解释无需重跑，并显示可复用字数", () => {
    renderT(<OCRCard item={{ ...base, stage: "cached", total: 1, done: 1, chars: 842 }} />);
    expect(screen.getByText("缓存命中")).toBeTruthy();
    expect(screen.getByText("已复用缓存，无需重新识别")).toBeTruthy();
    expect(screen.getByText(/842 字/)).toBeTruthy();
    expect(screen.getByText(/1\/1 页/)).toBeTruthy();
    expect(screen.queryByRole("progressbar")).toBeNull();
  });

  it("识别完成：显示结果可用性、格式化字数与 PDF 页数", () => {
    renderT(<OCRCard item={{ ...base, stage: "done", done: 20, chars: 1200 }} />);
    expect(screen.getByText("识别完成")).toBeTruthy();
    expect(screen.getByText("文字已提取，可用于后续整理")).toBeTruthy();
    expect(screen.getByText(/1,200 字/)).toBeTruthy();
    expect(screen.getByText(/20\/20 页/)).toBeTruthy();
  });

  it("失败态：原因与恢复路径同屏，并使用 alert 播报", () => {
    renderT(<OCRCard item={{ ...base, stage: "failed", total: 1, done: 0, err: "无可用 OCR 引擎" }} />);
    const alert = screen.getByRole("alert");
    expect(alert.textContent).toContain("无可用 OCR 引擎");
    expect(alert.textContent).toContain("重新导入文件即可重试");
    expect(screen.getByRole("status").textContent).toContain("失败");
    expect(screen.getByTestId("ocr-card").getAttribute("data-stage")).toBe("failed");
  });
});
