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

  it("不合规格的围栏不再静默蒸发：降级为可展开原文（2026-11-11 根修）", () => {
    // guard.ts 契约「结构不可用降级代码块」的落地断言：解析失败渲染折叠
    // details，内容可见，而非什么都没有。
    const broken = '```genui\n{"title":"废规格","items":[{"label":"无名无型"}]}\n```';
    render(
      <GenuiScopeProvider scope={{ scope: "chat", sessionKey: "t4" }}>
        <ChatMarkdown text={`前后文\n${broken}\n后文`} genuiKey="m4" />
      </GenuiScopeProvider>,
    );
    const details = document.querySelector("details.gui-fence-fallback");
    expect(details).toBeTruthy();
    expect(details?.textContent).toContain("废规格");
    expect(screen.getByText("前后文")).toBeTruthy();
  });

  it("真实方言形状（单组件根 + label/content 条目）在聊天缝渲染为组件", () => {
    // 聊天史实测高频输出形状（2026-11-11 前整块蒸发，修复后按上游归一口径出卡）。
    const dialect =
      '```genui\n{"type":"list","title":"四个维度","items":[{"label":"无常","content":"梦醒即逝。"}]}\n```';
    render(
      <GenuiScopeProvider scope={{ scope: "chat", sessionKey: "t5" }}>
        <ChatMarkdown text={`拆解如下\n${dialect}`} genuiKey="m5" />
      </GenuiScopeProvider>,
    );
    expect(screen.getByText("四个维度")).toBeTruthy();
    expect(screen.getByText("无常")).toBeTruthy();
    expect(screen.getByText("梦醒即逝。")).toBeTruthy();
    expect(document.querySelector(".gui-list-record")).toBeTruthy();
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
