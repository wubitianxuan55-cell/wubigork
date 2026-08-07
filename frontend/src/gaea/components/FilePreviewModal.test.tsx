import { describe, expect, it } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
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
});
