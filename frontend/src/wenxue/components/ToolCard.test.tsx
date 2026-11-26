import { afterEach, describe, expect, it } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import { LocaleProvider } from "../lib/i18n";
import { ToolCard } from "./ToolCard";
import type { ToolCardProps } from "../../gaea/components/ToolCard";

function tool(name: string): ToolCardProps["item"] {
  return {
    kind: "tool",
    id: `t-${name}`,
    name,
    args: "{}",
    readOnly: true,
    status: "done",
    output: "ok",
  };
}

describe("Wenxue ToolCard shared renderer", () => {
  afterEach(cleanup);

  it("renders an allowed tool through the office single source", () => {
    render(<LocaleProvider><ToolCard item={tool("read_file")} /></LocaleProvider>);
    expect(screen.getByText("read_file")).toBeTruthy();
  });

  it("keeps the Wenxue denied-tool filter at the shared boundary", () => {
    const view = render(<LocaleProvider><ToolCard item={tool("cost_project")} /></LocaleProvider>);
    expect(view.container.textContent).toBe("");
  });
});
