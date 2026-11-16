import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ComposerAttachmentBar } from "./ComposerAttachmentBar";
import type { AttachmentUploadTask } from "../../hooks/useComposerAttachments";

describe("Wenxue ComposerAttachmentBar uploads", () => {
  it("失败任务显示原因并提供重试与移除", () => {
    const task: AttachmentUploadTask = {
      id: "wx-u1",
      name: "参考.pdf",
      type: "file",
      status: "error",
      error: "文件上传失败：timeout",
      retryable: true,
    };
    const retry = vi.fn();
    const remove = vi.fn();
    render(
      <ComposerAttachmentBar
        attachments={[]}
        uploads={[task]}
        running={false}
        recognizingPath={null}
        ocrPath={null}
        onRecognize={() => {}}
        onOCRText={() => {}}
        onRemove={() => {}}
        onRetryUpload={retry}
        onRemoveUpload={remove}
      />,
    );
    expect(screen.getByText(/timeout/)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "重试上传 参考.pdf" }));
    fireEvent.click(screen.getByRole("button", { name: "移除失败上传 参考.pdf" }));
    expect(retry).toHaveBeenCalledWith("wx-u1");
    expect(remove).toHaveBeenCalledWith("wx-u1");
  });
});
