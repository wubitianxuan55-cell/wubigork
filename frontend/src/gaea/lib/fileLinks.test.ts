import { describe, expect, it } from "vitest";
import { deliverableMentions, findFileMentions, isLocalFilePath } from "./fileLinks";

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

describe("全角括号文件名（v4.26.1 回归）", () => {
  const ABS = "C:\\AI\\bangong\\黄甲\\开工筹备计划（修订）.docx";

  it("绝对路径带全角括号完整识别（真实办公会话：开工筹备计划（修订）.docx）", () => {
    const got = findFileMentions(`交付文件：${ABS}`).map((m) => m.path);
    expect(got).toEqual([ABS]);
  });

  it("关键词引导的裸文件名带全角括号", () => {
    const got = findFileMentions("已生成：报告（终稿）.docx").map((m) => m.path);
    expect(got).toEqual(["报告（终稿）.docx"]);
  });

  it("相对路径括号段在分隔符后", () => {
    const got = findFileMentions("见 exports/报告（终稿）.docx 一份").map((m) => m.path);
    expect(got).toEqual(["exports/报告（终稿）.docx"]);
  });

  it("扩展名锚定在匹配末尾：括号补语不吞进路径", () => {
    const got = findFileMentions("输出文件：报告.docx（三份）").map((m) => m.path);
    expect(got).toEqual(["报告.docx"]);
  });

  it("deliverableMentions 命中带括号路径（卡片数据源）", () => {
    expect(deliverableMentions(`交付文件：${ABS}`)).toEqual([ABS]);
  });
});
