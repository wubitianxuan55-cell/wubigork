import { describe, expect, it } from "vitest";
import { resolveMarkdownImagePath } from "./markdownImage";

describe("resolveMarkdownImagePath", () => {
  it("resolves relative images against the Markdown file directory", () => {
    expect(resolveMarkdownImagePath("./images/a.png", "docs/report.md")).toBe("docs/images/a.png");
    expect(resolveMarkdownImagePath("../assets/b.jpg", "docs/report.md")).toBe("assets/b.jpg");
    expect(resolveMarkdownImagePath("images/%E6%8A%A5%E5%91%8A.webp", "docs/report.md"))
      .toBe("docs/images/报告.webp");
  });

  it("understands Windows drive and file URLs", () => {
    expect(resolveMarkdownImagePath("file:///C:/Work/a.png", "docs/report.md")).toBe("C:/Work/a.png");
    expect(resolveMarkdownImagePath("D:\\Work\\b.png", "docs/report.md")).toBe("D:/Work/b.png");
    expect(resolveMarkdownImagePath("file://server/share/c.png", "docs/report.md")).toBe("//server/share/c.png");
  });

  it("passes direct images through and rejects active schemes", () => {
    expect(resolveMarkdownImagePath("data:image/png;base64,abc")).toBeNull();
    expect(resolveMarkdownImagePath("https://example.com/a.png")).toBeNull();
    expect(resolveMarkdownImagePath("javascript:alert(1)")).toBeNull();
  });
});
