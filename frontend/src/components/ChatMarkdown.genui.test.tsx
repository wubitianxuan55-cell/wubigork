import { describe, expect, it } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import ChatMarkdown from "./ChatMarkdown";
import { MarkdownContent } from "./MarkdownContent";
import { GenuiScopeProvider } from "../genui/scope";
import { buildMarkdownGenuiOverrides } from "./chat/genuiAdapter";

const fence =
  '```genui\n{"title":"小看板","items":[{"type":"stat","label":"营收","value":"¥128k"}]}\n```';

describe("聊天 markdown 缝渲染 GenUI", () => {
  it("ChatMarkdown（plain 终态）渲染 genui 围栏为组件", () => {
    render(
      <GenuiScopeProvider scope={{ scope: "chat", sessionKey: "t1" }}>
        <ChatMarkdown text={`结论如下\n${fence}\n以上。`} genuiKey="m1" />
      </GenuiScopeProvider>,
    );
    expect(screen.getByText("小看板")).toBeTruthy();
    expect(screen.getByText("营收")).toBeTruthy();
    expect(screen.getByText("¥128k")).toBeTruthy();
  });

  it("MarkdownContent（人格/流式路径）经覆盖件渲染 genui", () => {
    render(
      <GenuiScopeProvider scope={{ scope: "chat", sessionKey: "t1" }}>
        <MarkdownContent
          source={`分析如下\n${fence}`}
          className="md-content"
          components={buildMarkdownGenuiOverrides({ scope: "chat", sessionKey: "t1" }, "m1")}
        />
      </GenuiScopeProvider>,
    );
    expect(screen.getByText("小看板")).toBeTruthy();
    expect(screen.getByText("¥128k")).toBeTruthy();
  });
});

describe("聊天线代码块语法高亮（懒加载缝）", () => {
  const goFence = "```go\npackage main\n\nfunc main() {}\n```";

  it("ChatMarkdown（plain 终态）暗面板 + 异步 hljs 令牌", async () => {
    const { container } = render(<ChatMarkdown text={goFence} />);
    expect(container.querySelector(".hl-scope-dark")).toBeTruthy();
    expect(container.textContent).toContain("package main");
    await waitFor(() => expect(container.querySelector(".hljs-keyword")).toBeTruthy());
  });

  it("MarkdownContent（companion/流式）经覆盖件走同一面板", async () => {
    const { container } = render(
      <MarkdownContent
        source={goFence}
        className="md-content"
        components={buildMarkdownGenuiOverrides({ scope: "chat", sessionKey: "t2" }, "m2")}
      />,
    );
    expect(container.querySelector(".hl-scope-dark")).toBeTruthy();
    await waitFor(() => expect(container.querySelector(".hljs-keyword")).toBeTruthy());
  });
});

describe("聊天线数学公式渲染（v4.239）", () => {
  it("ChatMarkdown：$..$ 行内公式渲染为 KaTeX", () => {
    const { container } = render(<ChatMarkdown text={"质能方程 $E=mc^2$ 很有名"} />);
    expect(container.querySelector(".katex")).toBeTruthy();
  });

  it("MarkdownContent（companion/流式）：同款 KaTeX 渲染", () => {
    const { container } = render(
      <MarkdownContent
        source={"面积 $A=\\pi r^2$。"}
        className="md-content"
        components={buildMarkdownGenuiOverrides({ scope: "chat", sessionKey: "t3" }, "m3")}
      />,
    );
    expect(container.querySelector(".katex")).toBeTruthy();
  });
});
