// 季度报告会话恢复回放（mock/chat.ts ResumeSession → /mock/sessions/a.jsonl）
// 的工具卡契约：五个办公工具 dispatch/result 配对、信封 JSON 可解析、关键
// 格式串与 lib/tools.ts summarize/subjectOf 的解析口径逐字对齐（信封 message、
// 全角括号「（31409 字节，类型: …）」、knowledge_search「### 」行首小节、
// diagram_gen 裸 JSON、multi_edit diffstat）。防止 mock 样例与前端解析器漂移。
import { describe, expect, it } from "vitest";
import { makeMockApp } from "./mock";
import { rebuildHistoryItems } from "./store";
import type { Item } from "./store";
import { diffStatFor, subjectOf, summarize } from "./tools";
import type { HistoryMessage } from "./types";

type ToolItem = Extract<Item, { kind: "tool" }>;

const replay = (): Promise<HistoryMessage[]> => makeMockApp().ResumeSession("/mock/sessions/a.jsonl");

// 按恢复消息序列还原工具卡（与 store.rebuildHistoryItems 消费口径一致）
async function toolItems(): Promise<ToolItem[]> {
  const { items } = rebuildHistoryItems(await replay());
  return items.filter((it): it is ToolItem => it.kind === "tool");
}

async function toolByName(name: string): Promise<ToolItem> {
  const t = (await toolItems()).find((it) => it.name === name);
  expect(t, `工具卡 ${name} 应存在于回放中`).toBeDefined();
  return t as ToolItem;
}

describe("mock 恢复回放 · 季度报告会话（a.jsonl）", () => {
  it("六个工具 dispatch/result 一一配对：全部完成态且输出就位（顺序即叙事）", async () => {
    const items = await toolItems();
    expect(items.map((t) => t.name)).toEqual([
      "ls", "format_convert", "knowledge_search", "chart_gen", "diagram_gen", "multi_edit",
    ]);
    for (const t of items) {
      expect(t.status).toBe("done");
      expect(t.output).not.toBe("");
    }
    // 首条是用户请求（叙事起点），收尾有 assistant 正文
    const msgs = await replay();
    expect(msgs[0].role).toBe("user");
    expect(msgs[0].content).toContain("季度报告");
    expect(msgs[msgs.length - 1].role).toBe("assistant");
  });

  it("format_convert：信封 message 逐字对齐，args path=素材.docx，summarize 取路径+字符数", async () => {
    const t = await toolByName("format_convert");
    const env = JSON.parse(t.output ?? "") as { ok?: boolean; message?: string };
    expect(env.ok).toBe(true);
    expect(env.message).toBe("已转换并保存为 out/季度报告.md（3210 字符）");
    expect(JSON.parse(t.args)).toMatchObject({ path: "素材.docx" });
    expect(summarize("format_convert", t.args, t.output)).toContain("out/季度报告.md");
    expect(subjectOf("format_convert", t.args)).toBe("素材.docx");
  });

  it("knowledge_search：信封 message 含两个「### 」行首小节，summarize 计 2 条命中", async () => {
    const t = await toolByName("knowledge_search");
    const env = JSON.parse(t.output ?? "") as { ok?: boolean; message?: string };
    expect(env.ok).toBe(true);
    const sections = (env.message ?? "").match(/^### .+/gm) ?? [];
    expect(sections).toHaveLength(2);
    expect(summarize("knowledge_search", t.args, t.output)).toContain("2");
    expect(subjectOf("knowledge_search", t.args)).toContain("图表规范");
  });

  it("chart_gen：信封可解析且格式串逐字对齐（全角括号/「类型: grouped_bar」/「系列: 2 · 类别: 3」）", async () => {
    const t = await toolByName("chart_gen");
    const env = JSON.parse(t.output ?? "") as { ok?: boolean; success?: boolean; code?: number; message?: string };
    expect(env.ok).toBe(true);
    expect(env.success).toBe(true);
    expect(env.code).toBe(0);
    expect(env.message).toContain("✅ 图表已生成: charts/cost.png（31409 字节，类型: grouped_bar）");
    expect(env.message).toContain("\n标题: 分项单价对比");
    expect(env.message).toContain("\n系列: 2 · 类别: 3");
    // args 形状：title/labels/chart_type/series（2 系列 × 3 类别）
    const args = JSON.parse(t.args) as { title?: string; labels?: string[]; chart_type?: string; series?: unknown[] };
    expect(args.title).toBe("分项单价对比");
    expect(args.chart_type).toBe("grouped_bar");
    expect(args.labels).toHaveLength(3);
    expect(args.series).toHaveLength(2);
    // 前端解析口径：subject 取 title，summarize 含类型与类别数
    expect(subjectOf("chart_gen", t.args)).toBe("分项单价对比");
    const s = summarize("chart_gen", t.args, t.output);
    expect(s).toContain("grouped_bar");
    expect(s).toContain("3");
  });

  it("diagram_gen：裸 JSON（非信封），summarize 返回输出路径", async () => {
    const t = await toolByName("diagram_gen");
    expect(JSON.parse(t.output ?? "")).toEqual({ ok: true, output: "diagrams/架构.png", size_bytes: 40960 });
    expect(summarize("diagram_gen", t.args, t.output)).toBe("diagrams/架构.png");
    expect(subjectOf("diagram_gen", t.args)).toBe("季度报告处理链路");
  });

  it("multi_edit：edits 2-3 组 old/new，diffstat 芯片数据 +N−M 就位", async () => {
    const t = await toolByName("multi_edit");
    const edits = (JSON.parse(t.args) as { path?: string; edits?: { old_string?: string; new_string?: string }[] }).edits ?? [];
    expect(edits.length).toBeGreaterThanOrEqual(2);
    expect(edits.length).toBeLessThanOrEqual(3);
    for (const e of edits) {
      expect(typeof e.old_string).toBe("string");
      expect(typeof e.new_string).toBe("string");
    }
    const stat = diffStatFor("multi_edit", t.args);
    expect(stat).not.toBeNull();
    expect(stat?.add).toBeGreaterThan(0);
    expect(stat?.del).toBeGreaterThan(0);
    expect(summarize("multi_edit", t.args, t.output)).toContain("3");
  });

  it("既有行为不回归：其余会话保持原序列，interrupted 会话（d.jsonl）注入恢复摘要", async () => {
    const app = makeMockApp();
    const d = await app.ResumeSession("/mock/sessions/d.jsonl");
    expect(d[0].content).toContain("上次会话中断");
    expect(d.some((m) => m.role === "tool" && m.toolName === "edit_file")).toBe(true);
    const b = await makeMockApp().ResumeSession("/mock/sessions/b.jsonl");
    expect(b[0].content).toBe("(mock) resumed /mock/sessions/b.jsonl");
    expect(b.some((m) => m.role === "tool" && m.toolName === "edit_file")).toBe(true);
  });
});
