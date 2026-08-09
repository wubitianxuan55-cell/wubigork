import { describe, expect, it } from "vitest";
import { findFileMentions, isLocalFilePath } from "./fileLinks";

describe("isLocalFilePath", () => {
  it("识别本地文件 href，排除 URL", () => {
    expect(isLocalFilePath("C:\\AI\\a.docx")).toBe(true);
    expect(isLocalFilePath("reports/汇总.md")).toBe(true);
    expect(isLocalFilePath("../docs/a.xlsx")).toBe(true);
    expect(isLocalFilePath("https://example.com/a.docx")).toBe(false);
    expect(isLocalFilePath("mailto:a@b.com")).toBe(false);
    expect(isLocalFilePath("#anchor")).toBe(false);
  });
});

describe("findFileMentions", () => {
  it("识别绝对/相对/裸文件名三类引用", () => {
    const text =
      "见 exports/方案.docx 与 C:\\AI\\bangong\\表.xlsx，输出文件：成本测算.xlsx，已生成：报告.pdf";
    const got = findFileMentions(text).map((m) => m.path);
    expect(got).toEqual([
      "exports/方案.docx",
      "C:\\AI\\bangong\\表.xlsx",
      "成本测算.xlsx",
      "报告.pdf",
    ]);
  });

  it("URL 与域名式路径不识别", () => {
    expect(findFileMentions("https://example.com/a.pdf www.x.com/b.docx")).toEqual([]);
  });

  it("文件名带中文标点只取路径本身", () => {
    const got = findFileMentions("保存到：报告.docx。（详见）");
    expect(got).toHaveLength(1);
    expect(got[0].label).toBe("报告.docx");
    expect(got[0].end).toBe(got[0].start + "报告.docx".length);
  });

  it("关键词与文件名之间要求冒号或空格，不误伤普通词语", () => {
    expect(findFileMentions("文件夹：临时.docx")).toHaveLength(0);
    expect(findFileMentions("profile: a.csv")).toHaveLength(1);
  });
});
