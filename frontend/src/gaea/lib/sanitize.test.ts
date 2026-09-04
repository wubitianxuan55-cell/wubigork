// sanitize 单测（3b：DOMPurify 消毒层）。
// jsdom 下 DOMPurify 可直接运行。
import { describe, expect, it } from "vitest";
import { sanitizeHtml } from "./sanitize";
import { htmlFileLinks, fileChipHtml } from "./fileLinks";

describe("sanitizeHtml 流式尾部消毒", () => {
  it("保留文件 chip 的 span/button 与 data-file-preview 属性", () => {
    const chip = fileChipHtml("docs/调研结论.md");
    const out = sanitizeHtml(`<p>${chip}</p>`);
    expect(out).toContain("data-file-preview");
    expect(out).toContain("调研结论.md");
  });

  it("htmlFileLinks 的转义输出经消毒后不含活事件处理器", () => {
    const out = sanitizeHtml(htmlFileLinks('<img src=x onerror="alert(1)">见附件'));
    // htmlFileLinks 已整体转义：<img 变成 &lt;img（纯文本，惰性）
    expect(out).not.toContain("<img");
    expect(out).toContain("见附件");
  });

  it("手写事件属性被剥离（兜底未来 renderPending 回归）", () => {
    const out = sanitizeHtml('<span data-x="1" onclick="alert(1)">文本</span>');
    expect(out).toContain("文本");
    expect(out).not.toContain("onclick");
    expect(out).toContain("data-x");
  });
});
