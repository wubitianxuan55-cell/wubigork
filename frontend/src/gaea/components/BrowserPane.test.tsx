import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { BrowserPane } from "./BrowserPane";

describe("BrowserPane 沙箱浏览器", () => {
  it("导航使用可访问图标按钮，并给出下一步空态", () => {
    render(<BrowserPane onUrlChange={vi.fn()} />);

    expect(screen.getByRole("button", { name: "后退" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "前进" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "刷新" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "前往" })).toBeTruthy();
    expect(screen.getByText("在沙箱中打开网页")).toBeTruthy();
    expect(screen.getByText(/不会接触本地文件/)).toBeTruthy();
  });

  it("非法协议就近显示可感知错误，不更新地址", () => {
    const onUrlChange = vi.fn();
    render(<BrowserPane onUrlChange={onUrlChange} />);

    fireEvent.change(screen.getByRole("textbox", { name: "浏览器地址" }), {
      target: { value: "javascript:alert(1)" },
    });
    fireEvent.keyDown(screen.getByRole("textbox", { name: "浏览器地址" }), {
      key: "Enter",
    });

    expect(screen.getByRole("alert").textContent).toContain("仅支持 http/https");
    expect(onUrlChange).not.toHaveBeenCalled();
  });
});
