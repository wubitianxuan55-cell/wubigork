// subagentRender.test.ts — 子代理 transcript → Codex 式渲染项映射测试。
import { describe, expect, it } from "vitest";
import { buildRenderItems, toolStatus, unwrapEnvelopeText } from "./subagentRender";
import type { SubagentTranscriptMessage } from "./types";

describe("buildRenderItems 映射", () => {
  it("assistant.toolCalls 与后续 tool 结果按 toolCallId 配对成完整工具项", () => {
    const msgs: SubagentTranscriptMessage[] = [
      { role: "system", content: "你是子代理" },
      { role: "user", content: "任务" },
      {
        role: "assistant",
        content: "开始检索。",
        toolCalls: [
          { id: "call_1", name: "web_search", arguments: "{\"query\":\"cr\"}" },
          { id: "call_2", name: "knowledge_search", arguments: "{\"query\":\"修复\"}" },
        ],
      },
      { role: "tool", name: "knowledge_search", toolCallId: "call_2", content: "知识结果" },
      { role: "tool", name: "web_search", toolCallId: "call_1", content: "网页结果" },
      { role: "assistant", content: "结论如下。" },
    ];
    const items = buildRenderItems(msgs, false);
    // 2 个工具项（配对后不再有孤儿 tool 行）+ 2 个 assistant + system + user
    const tools = items.filter((i) => i.type === "tool");
    expect(tools).toHaveLength(2);
    const t1 = tools.find((t) => t.id === "call_1");
    expect(t1).toMatchObject({ name: "web_search", args: "{\"query\":\"cr\"}", output: "网页结果", pending: false });
    const t2 = tools.find((t) => t.id === "call_2");
    expect(t2).toMatchObject({ name: "knowledge_search", output: "知识结果", pending: false });
    // assistant live 只标最后一条（且仅运行中）
    const assistants = items.filter((i) => i.type === "assistant");
    expect(assistants.map((a) => ("live" in a ? a.live : null))).toEqual([false, false]);
  });

  it("乱序到达的 tool 结果也能按 id 配上（转录为快照态，顺序由落盘决定）", () => {
    const msgs: SubagentTranscriptMessage[] = [
      { role: "assistant", content: "", toolCalls: [{ id: "x1", name: "web_fetch", arguments: "{}" }] },
      { role: "assistant", content: "继续", toolCalls: [{ id: "x2", name: "web_fetch", arguments: "{}" }] },
      { role: "tool", toolCallId: "x1", name: "web_fetch", content: "一" },
      { role: "tool", toolCallId: "x2", name: "web_fetch", content: "二" },
    ];
    const tools = buildRenderItems(msgs, false).filter((i) => i.type === "tool") as Array<{
      id: string; output?: string; pending: boolean;
    }>;
    expect(tools.map((t) => [t.id, t.output])).toEqual([["x1", "一"], ["x2", "二"]]);
    expect(tools.every((t) => !t.pending)).toBe(true);
  });

  it("孤儿 tool 行（无配对调用）降级为独立卡；live 标记仅运行中最后一条", () => {
    const msgs: SubagentTranscriptMessage[] = [
      { role: "tool", name: "vision", content: "图像识别结果" },
      { role: "assistant", content: "完成" },
    ];
    const items = buildRenderItems(msgs, true);
    expect(items[0]).toMatchObject({ type: "tool", name: "vision", output: "图像识别结果", pending: false });
    expect(items[1]).toMatchObject({ type: "assistant", live: true });
    expect(toolStatus(items[0] as never, true)).toBe("done");
  });

  it("运行中：结果未到的工具项状态为 running", () => {
    const msgs: SubagentTranscriptMessage[] = [
      { role: "assistant", toolCalls: [{ id: "p1", name: "web_search", arguments: "{}" }] },
    ];
    const items = buildRenderItems(msgs, true);
    expect(toolStatus(items[0] as never, true)).toBe("running");
    expect(toolStatus(items[0] as never, false)).toBe("done");
  });
});

// unwrapEnvelopeText 显示侧拆包（v4.64.1）：救回写端修复前落盘的信封串。
describe("unwrapEnvelopeText 显示侧拆包", () => {
  it("双层信封递归拆到纯文本（字面转义换行还原为真实换行）", () => {
    const inner = '{"message": "第一段\\n\\n第二段"}';
    const outer = JSON.stringify({ ok: true, data: { result: inner } });
    expect(unwrapEnvelopeText(outer)).toBe("第一段\n\n第二段");
  });

  it("自由文本 / 破损 JSON 原样返回", () => {
    expect(unwrapEnvelopeText("普通正文")).toBe("普通正文");
    expect(unwrapEnvelopeText("{not-json")).toBe("{not-json");
  });
});
