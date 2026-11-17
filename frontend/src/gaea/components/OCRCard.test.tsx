// OCRCard — OCR 专属大卡（2026-11-18）渲染态测试：
// 引擎拉起提示 / PDF 页粒度进度条 / 缓存命中 / 完成字数 / 失败原因 / 引擎 chip。
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
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

describe("OCRCard OCR 专属大卡", () => {
  it("PDF 识别中：页粒度进度条 + 引擎 chip + 状态词", () => {
    const { container } = renderT(<OCRCard item={base} />);
    expect(screen.getByTestId("ocr-card")).toBeTruthy();
    expect(screen.getByText("扫描件.pdf")).toBeTruthy();
    expect(screen.getByText("uocr2")).toBeTruthy();
    expect(screen.getByText("识别中")).toBeTruthy();
    expect(screen.getByText("3/20 页")).toBeTruthy();
    const bar = container.querySelector(".ocr-card-bar > i") as HTMLElement;
    expect(bar.style.width).toBe("15%");
  });

  it("引擎拉起态：单文件不画进度条，给冷启动提示", () => {
    const view = renderT(<OCRCard item={{ ...base, stage: "engine-start", total: 1, done: 0 }} />);
    expect(screen.getByText("引擎拉起中")).toBeTruthy();
    expect(screen.getByText(/冷启动/)).toBeTruthy();
    expect(view.container.querySelector(".ocr-card-bar")).toBeNull();
  });

  it("缓存命中与完成态：字数可见，进度条退场", () => {
    const view = renderT(<OCRCard item={{ ...base, stage: "cached", total: 1, chars: 842 }} />);
    expect(screen.getByText("缓存命中")).toBeTruthy();
    expect(screen.getByText("842 字")).toBeTruthy();
    expect(view.container.querySelector(".ocr-card-bar")).toBeNull();
    const done = renderT(<OCRCard item={{ ...base, stage: "done", total: 1, chars: 1200 }} />);
    expect(done.getByText("完成")).toBeTruthy();
    expect(done.getByText("1200 字")).toBeTruthy();
  });

  it("失败态：原因可见，卡片标红", () => {
    const view = renderT(<OCRCard item={{ ...base, stage: "failed", err: "无可用 OCR 引擎" }} />);
    expect(screen.getByText("失败")).toBeTruthy();
    expect(screen.getByText("无可用 OCR 引擎")).toBeTruthy();
    expect((view.container.querySelector('[data-testid="ocr-card"]') as HTMLElement).getAttribute("data-stage")).toBe("failed");
  });
});
