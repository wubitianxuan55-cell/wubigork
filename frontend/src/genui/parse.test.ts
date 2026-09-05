import { describe, expect, it } from "vitest";
import {
  collectPartialCandidates,
  parseGenuiFenceBody,
  parsePartialGenuiSpec,
  splitGenuiFences,
  stripTrailingCommas,
} from "./parse";

describe("parse.splitGenuiFences", () => {
  it("切分 genui 围栏并保留其他代码块在文本段", () => {
    const text = [
      "开头",
      "```genui",
      '{"items":[{"type":"text","content":"hi"}]}',
      "```",
      "中间",
      "```js",
      "console.log(1)",
      "```",
      "结尾",
    ].join("\n");
    const { segments } = splitGenuiFences(text);
    expect(segments).toHaveLength(3);
    expect(segments[0]).toMatchObject({ kind: "text" });
    expect(segments[1]).toMatchObject({ kind: "fence", lang: "genui", closed: true });
    expect(segments[2].kind).toBe("text");
    expect(segments[2].kind === "text" && segments[2].text).toContain("```js");
  });

  it("未闭合围栏标记 closed=false；dsh-ui 兼容", () => {
    const { segments } = splitGenuiFences('```dsh-ui\n{"items":[]}');
    expect(segments[segments.length - 1]).toMatchObject({
      kind: "fence",
      lang: "dsh-ui",
      closed: false,
    });
  });
});

describe("parse.stripTrailingCommas", () => {
  it("只清字符串外的尾随逗号", () => {
    const input = '{"items":[{"type":"text","content":"a,},"},{"type":"stat","label":"x","value":"1",}]}';
    const out = stripTrailingCommas(input);
    expect(out).toBe('{"items":[{"type":"text","content":"a,},"},{"type":"stat","label":"x","value":"1"}]}');
  });
});

describe("parse.parseGenuiFenceBody", () => {
  it("完整围栏体修复渲染", () => {
    const spec = parseGenuiFenceBody(
      '{"title":"看板","items":[{"type":"stat","label":"a","value":"1"}]}',
    );
    expect(spec?.items[0]).toMatchObject({ type: "stat" });
  });

  it("坏 JSON / 超长围栏返回 null", () => {
    expect(parseGenuiFenceBody("{oops")).toBeNull();
    expect(parseGenuiFenceBody(`{"items":[{"type":"text","content":"${"x".repeat(70000)}"}]}`)).toBeNull();
  });
});

describe("parse.collectPartialCandidates", () => {
  it("有界候选收集，跳过字符串与转义", () => {
    const body = '{"items":[{"type":"text","content":"}"}]}';
    const { candidates } = collectPartialCandidates(body);
    expect(candidates.length).toBeGreaterThan(0);
    expect(candidates.every((c) => c.end <= body.length)).toBe(true);
  });
});

describe("parse.parsePartialGenuiSpec", () => {
  it("只渲染已写完的组件前缀", () => {
    const partial = parsePartialGenuiSpec(
      '{"items":[{"type":"text","content":"第一段"},{"type":"badge","label":"第二段"',
    );
    expect(partial?.items).toHaveLength(1);
    const first = partial?.items[0];
    if (first?.type !== "text") throw new Error("text expected");
    expect(first.content).toBe("第一段");
  });

  it("完整围栏直接返回全量", () => {
    const spec = parsePartialGenuiSpec(
      '{"items":[{"type":"text","content":"a"},{"type":"text","content":"b"}]}',
    );
    expect(spec?.items).toHaveLength(2);
  });
});
