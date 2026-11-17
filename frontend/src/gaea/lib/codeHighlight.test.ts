import { describe, expect, it } from "vitest";
import {
  MAX_HIGHLIGHT_BYTES,
  MAX_HIGHLIGHT_LINE_BYTES,
  MAX_HIGHLIGHT_LINES,
  applyHlTheme,
  canonFenceLang,
  exceedsHighlightLimits,
  highlightCode,
} from "./codeHighlight";

describe("canonFenceLang 围栏语言归一", () => {
  it("白名单语言直通（大小写/空白容错）", () => {
    expect(canonFenceLang("go")).toBe("go");
    expect(canonFenceLang(" SQL ")).toBe("sql");
    expect(canonFenceLang("PowerShell")).toBe("powershell");
  });

  it("别名经共享 ALIASES 归一（与编辑器/工具卡同源）", () => {
    expect(canonFenceLang("ts")).toBe("typescript");
    expect(canonFenceLang("py")).toBe("python");
    expect(canonFenceLang("sh")).toBe("bash");
    expect(canonFenceLang("c++")).toBe("cpp");
  });

  it("通用域白名单（v4.231 扩容：ruby/php/kotlin/perl/lua）", () => {
    expect(canonFenceLang("ruby")).toBe("ruby");
    expect(canonFenceLang("php")).toBe("php");
    expect(canonFenceLang("kotlin")).toBe("kotlin");
    expect(canonFenceLang("perl")).toBe("perl");
    expect(canonFenceLang("lua")).toBe("lua");
  });

  it("明示纯文本与未知语言回退 undefined（不硬造着色）", () => {
    expect(canonFenceLang("text")).toBeUndefined();
    expect(canonFenceLang("plaintext")).toBeUndefined();
    expect(canonFenceLang("")).toBeUndefined();
    expect(canonFenceLang(undefined)).toBeUndefined();
    expect(canonFenceLang("brainfuck")).toBeUndefined();
  });

  it("toml 借 ini 语法着色（无官方语法，结构同源）", () => {
    expect(canonFenceLang("toml")).toBe("ini");
  });
});

describe("highlightCode 懒加载高亮", () => {
  it("enforces aggregate byte/line/single-line gates before loading hljs", async () => {
    expect([MAX_HIGHLIGHT_BYTES, MAX_HIGHLIGHT_LINES, MAX_HIGHLIGHT_LINE_BYTES])
      .toEqual([512 * 1024, 10000, 4 * 1024]);
    expect(exceedsHighlightLimits(2000, 1, 2000)).toBe(false);
    expect(exceedsHighlightLimits(512 * 1024 + 1, 1, 1)).toBe(true);
    expect(exceedsHighlightLimits(1, 10001, 1)).toBe(true);
    expect(exceedsHighlightLimits(1, 1, 4097)).toBe(true);
    expect(await highlightCode("x".repeat(5000), "go")).toBeNull();
  });

  it("go 代码产出 hljs 令牌 span", async () => {
    const html = await highlightCode('package main\n\nfunc main() {\n\tprintln("hi")\n}', "go");
    expect(html).toContain("hljs-keyword");
  });

  it("别名语言同路着色（py 注释）", async () => {
    const html = await highlightCode('x = "hi"  # 注释', "py");
    expect(html).toContain("hljs-comment");
    expect(html).toContain("hljs-string");
  });

  it("未注册语言返回 null（纯文本降级）", async () => {
    expect(await highlightCode("hello world", "brainfuck")).toBeNull();
    expect(await highlightCode("hello world", undefined)).toBeNull();
    expect(await highlightCode("hello world", "text")).toBeNull();
  });

  it("源码 HTML 字符被转义，脚本标签不成活", async () => {
    const html = await highlightCode('const s = "<script>alert(1)</script>";', "js");
    expect(html).not.toContain("<script>");
    expect(html).toContain("&lt;script&gt;");
  });
});

describe("applyHlTheme 明暗挂钩", () => {
  it("写 data-hl dataset 供 styles.css 切换调色板", () => {
    applyHlTheme(true);
    expect(document.documentElement.dataset.hl).toBe("dark");
    applyHlTheme(false);
    expect(document.documentElement.dataset.hl).toBe("light");
  });
});
