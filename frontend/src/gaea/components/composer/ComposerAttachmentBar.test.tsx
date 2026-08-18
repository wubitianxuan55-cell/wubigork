import { describe, expect, it, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ComposerAttachmentBar } from "./ComposerAttachmentBar";
import { usePreviewStore } from "../../lib/store";
import type { Attachment } from "../../hooks/useComposerAttachments";

const noop = () => {};

describe("ComposerAttachmentBar 附件预览条", () => {
  beforeEach(() => {
    usePreviewStore.setState({ previewFile: null, previewList: [], previewIndex: -1 });
  });

  it("无附件时返回 null", () => {
    const { container } = render(
      <ComposerAttachmentBar attachments={[]} running={false} recognizingPath={null} ocrPath={null} onRecognize={noop} onOCRText={noop} onRemove={noop} />,
    );
    expect(container.firstChild).toBeNull();
  });

  it("图片附件渲染缩略图与大小", () => {
    const atts: Attachment[] = [
      { path: ".gaea/uploads/a.png", previewUrl: "data:image/png;base64,AAAA", type: "image", size: 4096 },
    ];
    render(
      <ComposerAttachmentBar attachments={atts} running={false} recognizingPath={null} ocrPath={null} onRecognize={noop} onOCRText={noop} onRemove={noop} />,
    );
    expect(screen.getByAltText("")).toBeTruthy();
    // P2-4 上下文占用透明化：显示 4.0 KB
    expect(screen.getByText("4.0 KB")).toBeTruthy();
  });

  it("非图片附件渲染 chip（文件名 + badge），点击打开预览", () => {
    const atts: Attachment[] = [
      { path: ".gaea/uploads/报告.docx", previewUrl: "", type: "file", size: 2048 },
    ];
    render(
      <ComposerAttachmentBar attachments={atts} running={false} recognizingPath={null} ocrPath={null} onRecognize={noop} onOCRText={noop} onRemove={noop} />,
    );
    const btn = screen.getByRole("button", { name: /预览 报告\.docx/ });
    expect(btn.textContent).toContain("报告.docx");
    expect(btn.textContent).toContain("docx");
    expect(btn.textContent).toContain("2.0 KB");
    fireEvent.click(btn);
    expect(usePreviewStore.getState().previewFile).toBe(".gaea/uploads/报告.docx");
  });

  it("移除按钮触发 onRemove", () => {
    const removed: string[] = [];
    const atts: Attachment[] = [
      { path: ".gaea/uploads/a.docx", previewUrl: "", type: "file" },
    ];
    render(
      <ComposerAttachmentBar attachments={atts} running={false} recognizingPath={null} ocrPath={null} onRecognize={noop} onOCRText={noop} onRemove={(p) => removed.push(p)} />,
    );
    fireEvent.click(screen.getByTitle("移除"));
    expect(removed).toEqual([".gaea/uploads/a.docx"]);
  });

  it("无 size 的附件不显示大小", () => {
    const atts: Attachment[] = [
      { path: ".gaea/uploads/b.docx", previewUrl: "", type: "file" },
    ];
    render(
      <ComposerAttachmentBar attachments={atts} running={false} recognizingPath={null} ocrPath={null} onRecognize={noop} onOCRText={noop} onRemove={noop} />,
    );
    expect(screen.queryByText(/KB|MB|B$/)).toBeNull();
  });
});
