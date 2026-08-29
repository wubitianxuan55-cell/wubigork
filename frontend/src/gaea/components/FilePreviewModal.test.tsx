import { describe, expect, it } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { FilePreviewModal } from "./FilePreviewModal";
import { usePreviewStore } from "../lib/store";

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
