import { describe, expect, it } from "vitest";
import { extractMarkdownOutline } from "./markdownOutline";

describe("extractMarkdownOutline", () => {
  it("extracts headings while ignoring fenced code", () => {
    const outline = extractMarkdownOutline([
      "# 总览",
      "## [安装](https://example.com) `CLI`",
      "```md",
      "# 不是标题",
      "```",
      "### **细节**",
    ].join("\n"));
    expect(outline).toEqual([
      { id: "md-heading-1", level: 1, line: 1, text: "总览" },
      { id: "md-heading-2", level: 2, line: 2, text: "安装 CLI" },
      { id: "md-heading-3", level: 3, line: 6, text: "细节" },
    ]);
  });
});
