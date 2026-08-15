// T7-4（v2.37.0）MarkdownContent / ChatMarkdown 的 React.memo 稳定性。
import { describe, expect, it, vi } from "vitest";
import { render } from "@testing-library/react";
import { MarkdownContent } from "./MarkdownContent";
import ChatMarkdown from "./ChatMarkdown";

vi.mock("react-markdown", () => ({
  default: vi.fn(() => null),
}));
import ReactMarkdown from "react-markdown";

describe("MarkdownContent memo（T7-4）", () => {
  it("是 React.memo 组件", () => {
    expect((MarkdownContent as unknown as { $$typeof?: unknown }).$$typeof).toBe(Symbol.for("react.memo"));
  });

  it("source 未变时 rerender 不重渲染 ReactMarkdown；变化才重渲染", () => {
    const markdownMock = vi.mocked(ReactMarkdown);
    markdownMock.mockClear();
    const { rerender } = render(<MarkdownContent source="**加粗**" />);
    expect(markdownMock).toHaveBeenCalledTimes(1);

    rerender(<MarkdownContent source="**加粗**" />);
    expect(markdownMock).toHaveBeenCalledTimes(1); // memo 生效，跳过

    rerender(<MarkdownContent source="新内容" />);
    expect(markdownMock).toHaveBeenCalledTimes(2);
  });
});

describe("ChatMarkdown memo（T7-4）", () => {
  it("已是 React.memo 组件（存量验证）", () => {
    expect((ChatMarkdown as unknown as { $$typeof?: unknown }).$$typeof).toBe(Symbol.for("react.memo"));
  });

  it("text 未变时 rerender 不重渲染 ReactMarkdown", () => {
    const markdownMock = vi.mocked(ReactMarkdown);
    markdownMock.mockClear();
    const { rerender } = render(<ChatMarkdown text="你好" />);
    expect(markdownMock).toHaveBeenCalledTimes(1);

    rerender(<ChatMarkdown text="你好" />);
    expect(markdownMock).toHaveBeenCalledTimes(1);

    rerender(<ChatMarkdown text="变了" />);
    expect(markdownMock).toHaveBeenCalledTimes(2);
  });
});
