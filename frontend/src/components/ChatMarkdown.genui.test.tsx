import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
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
