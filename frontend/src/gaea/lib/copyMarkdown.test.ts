import { describe, expect, it } from "vitest";
import { escapeMarkdownCell, snapshotTable, tableToMarkdown } from "./copyMarkdown";

describe("table copy Markdown source", () => {
  it("serializes a DOM table as GFM with alignment and escaped pipes", () => {
    const host = document.createElement("div");
    host.innerHTML = [
      "<table>",
      "<thead><tr><th align=\"left\">名称</th><th align=\"right\">命令</th></tr></thead>",
      "<tbody><tr><td>a|b</td><td>x y</td></tr></tbody>",
      "</table>",
    ].join("");
    const table = host.querySelector("table")!;
    const snapshot = snapshotTable(table);
    expect(snapshot).not.toBeNull();
    const markdown = tableToMarkdown(snapshot!);
    expect(markdown).toBe("| 名称 | 命令 |\n|:--|--:|\n| a\\|b | x y |");
    expect(markdown).not.toContain("\t");
  });

  it("escapes backslashes and pipes deterministically", () => {
    expect(escapeMarkdownCell("a\\b|c")).toBe("a\\\\b\\|c");
  });
});
