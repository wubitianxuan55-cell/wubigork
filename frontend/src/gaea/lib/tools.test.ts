import { createElement } from "react";
import { beforeAll, describe, expect, it } from "vitest";
import { render } from "@testing-library/react";
import {
  boundedOutput,
  diffStatFor,
  summarize,
  subjectOf,
  TOOL_OUTPUT_MAX_PREVIEW_LINES,
} from "./tools";
import { LocaleProvider } from "./i18n";

// summarize 走模块级 t()（无 React 上下文，默认 en）。挂一次 zh 的
// LocaleProvider（渲染期同步刷新 i18n 镜像）让断言钉住中文文案。
beforeAll(() => {
  localStorage.setItem("gaea-lang", "zh");
  render(createElement(LocaleProvider, null, createElement("div")));
});

describe("diffStatFor 编辑类工具 diffstat（v4.27 Codex 式 +N−M 芯片）", () => {
  it("edit_file：按 old/new 行差异输出 +N/−N", () => {
    expect(
      diffStatFor("edit_file", JSON.stringify({ path: "a.md", old_string: "a\nb", new_string: "a\nb\nc" })),
    ).toEqual({ add: 1, del: 0 });
    expect(
      diffStatFor("edit_file", JSON.stringify({ path: "a.md", old_string: "a\nb\nc", new_string: "a\nc" })),
    ).toEqual({ add: 0, del: 1 });
  });

  it("multi_edit：逐组累加行级增减", () => {
    expect(
      diffStatFor("multi_edit", JSON.stringify({
        path: "a.md",
        edits: [
          { old_string: "x", new_string: "x\ny" },
          { old_string: "a\nb", new_string: "a" },
        ],
      })),
    ).toEqual({ add: 1, del: 1 });
  });

  it("非编辑工具 / 缺参返回 null", () => {
    expect(diffStatFor("read_file", JSON.stringify({ path: "a.md" }))).toBeNull();
    expect(diffStatFor("write_file", JSON.stringify({ path: "a.md", content: "x" }))).toBeNull();
    expect(diffStatFor("edit_file", JSON.stringify({ path: "a.md" }))).toBeNull();
  });
});

describe("boundedOutput 大工具输出有界预览（P2-2）", () => {
  it("空输出原样返回且不折叠", () => {
    const r = boundedOutput("");
    expect(r.full).toBe("");
    expect(r.collapsed).toBe(false);
  });

  it("undefined 视为空", () => {
    const r = boundedOutput(undefined);
    expect(r.full).toBe("");
    expect(r.collapsed).toBe(false);
  });

  it("行数未超阈值不折叠", () => {
    const out = Array.from({ length: 10 }, (_, i) => `line${i}`).join("\n");
    const r = boundedOutput(out, 60);
    expect(r.collapsed).toBe(false);
    expect(r.preview).toBe(out);
    expect(r.totalLines).toBe(10);
  });

  it("超长输出折叠为头部 + 折叠提示行", () => {
    const out = Array.from({ length: 100 }, (_, i) => `line${i}`).join("\n");
    const r = boundedOutput(out, 60);
    expect(r.collapsed).toBe(true);
    expect(r.totalLines).toBe(100);
    expect(r.hiddenLines).toBe(40);
    expect(r.preview).toContain("line0");
    expect(r.preview).toContain("line59");
    expect(r.preview).not.toContain("line60");
    expect(r.preview).toContain("… 已折叠 40 行");
    expect(r.full).toBe(out);
  });

  it("默认阈值 60 行生效", () => {
    expect(TOOL_OUTPUT_MAX_PREVIEW_LINES).toBe(60);
    const out = Array.from({ length: 61 }, (_, i) => `l${i}`).join("\n");
    expect(boundedOutput(out).collapsed).toBe(true);
  });
});

// ── 办公工具行首主体与结果摘要（工具可读化批）──────────────────────────

// 后端 WrapText 的 ToolEnvelope 包装
const envelope = (message: string) =>
  JSON.stringify({ ok: true, success: true, code: 0, message });

describe("subjectOf 办公工具行首主体", () => {
  it("chart_gen/diagram_gen 取标题", () => {
    expect(subjectOf("chart_gen", JSON.stringify({ title: "单价对比", labels: ["a"] }))).toBe("单价对比");
    expect(subjectOf("diagram_gen", JSON.stringify({ title: "架构图", nodes: [] }))).toBe("架构图");
  });

  it("检索类工具取查询词", () => {
    expect(subjectOf("knowledge_search", JSON.stringify({ query: "混凝土配比" }))).toBe("混凝土配比");
    expect(subjectOf("memory_search", JSON.stringify({ query: "项目偏好" }))).toBe("项目偏好");
  });

  it("knowledge_add 取标题，read_skill 取技能名", () => {
    expect(subjectOf("knowledge_add", JSON.stringify({ title: "规范条目", body: "x" }))).toBe("规范条目");
    expect(subjectOf("read_skill", JSON.stringify({ name: "weekly-report" }))).toBe("weekly-report");
  });

  it("move_file 拼接 source → destination", () => {
    expect(
      subjectOf("move_file", JSON.stringify({ source: "a.md", destination: "docs/a.md" })),
    ).toBe("a.md → docs/a.md");
    expect(subjectOf("move_file", JSON.stringify({ source: "a.md" }))).toBe("a.md");
  });

  it("format_convert 等路径类回退默认 path", () => {
    expect(subjectOf("format_convert", JSON.stringify({ path: "报告.docx" }))).toBe("报告.docx");
  });
});

describe("summarize 办公工具完成态摘要", () => {
  it("format_convert：保存态取路径与字符数", () => {
    const out = envelope("已转换并保存为 out/报告.md（3210 字符）");
    expect(summarize("format_convert", JSON.stringify({ path: "报告.docx" }), out)).toBe("out/报告.md · 3210 字符");
  });

  it("format_convert：返回正文态取文档名", () => {
    const out = envelope("# 文档转换: 报告.docx\n\n正文…");
    expect(summarize("format_convert", JSON.stringify({ path: "报告.docx" }), out)).toBe("报告.docx");
  });

  it("chart_gen：类型 + 数据点（单系列）", () => {
    const out = envelope("✅ 图表已生成: chart.png（20480 字节，类型: bar）\n标题: 单价对比\n数据点: 4");
    expect(summarize("chart_gen", JSON.stringify({ labels: ["a"] }), out)).toBe("bar · 4 个数据点");
  });

  it("chart_gen：类型 + 类别数（多系列）", () => {
    const out = envelope("✅ 图表已生成: chart.png（20480 字节，类型: grouped_bar）\n标题: 对比\n系列: 2 · 类别: 3");
    expect(summarize("chart_gen", JSON.stringify({ labels: ["a"] }), out)).toBe("grouped_bar · 3 个数据点");
  });

  it("diagram_gen：裸 JSON 取输出路径", () => {
    const out = JSON.stringify({ ok: true, output: "diagram.png", size_bytes: 40960 });
    expect(summarize("diagram_gen", JSON.stringify({ title: "x" }), out)).toBe("diagram.png");
    expect(summarize("diagram_gen", JSON.stringify({ title: "x" }), "not json")).toBe("");
  });

  it("knowledge_add：入库确认", () => {
    const out = envelope("✅ 已保存知识条目\n\n标题：规范");
    expect(summarize("knowledge_add", JSON.stringify({ title: "规范" }), out)).toBe("已存入知识库");
  });

  it("knowledge_search：按 ### 条目计数", () => {
    const out = envelope("### 条目A\n\n正文\n\n---\n\n### 条目B\n\n正文\n\n---\n\n");
    expect(summarize("knowledge_search", JSON.stringify({ query: "x" }), out)).toBe("2 条命中");
  });

  it("memory_search：编号行计数 + more 折叠补全", () => {
    const raw = "1. ▮▮ 记忆A\n   预览\n2. ▮▮ 记忆B\n   预览\n\n... and 3 more. Narrow your query for fewer results.\n";
    expect(summarize("memory_search", JSON.stringify({ query: "x" }), raw)).toBe("5 条命中");
    expect(summarize("memory_search", JSON.stringify({ query: "x" }), "无结果")).toBe("");
  });

  it("read_skill：内容行数", () => {
    expect(summarize("read_skill", JSON.stringify({ name: "s" }), "# 技能\n正文两行\n第三行")).toBe("3 行");
  });

  it("error 态一律无摘要", () => {
    expect(summarize("chart_gen", "{}", undefined, "boom")).toBe("");
  });
});
