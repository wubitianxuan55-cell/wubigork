import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ToolSources } from "./ToolSources";

describe("ToolSources", () => {
  it("去重后展开来源卡，链接使用新窗口且不泄露 opener", () => {
    render(
      <ToolSources sources={[
        { title: "Go 1.24", url: "https://go.dev/doc/go1.24", snippet: "release notes", source: "官方" },
        { title: "重复", url: "https://go.dev/doc/go1.24", source: "bing" },
      ]} />,
    );
    expect(screen.getByRole("button", { name: /来源 1/ })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: /来源 1/ }));
    const link = screen.getByRole("link", { name: /Go 1\.24/ });
    expect(link.getAttribute("href")).toBe("https://go.dev/doc/go1.24");
    expect(link.getAttribute("target")).toBe("_blank");
    expect(link.getAttribute("rel")).toBe("noreferrer");
  });

  it("没有来源时不占位", () => {
    const { container } = render(<ToolSources sources={[]} />);
    expect(container.firstChild).toBeNull();
  });
});
