// subagentRender.ts — 子代理 transcript → Codex 式渲染项的纯映射层。
//
// Why: SubagentThread 的消息流与主对话完全同款（v4.63）——assistant 正文/
// 思考走 AssistantMessage，tool 走 ToolCard。但 transcript 的消息形态（扁平
// role 流，tool 结果只带 toolCallId）与主对话 items（ToolCard 需要聚合后的
// name/args/output）不同，这里做纯函数映射：assistant.toolCalls 与后续 tool
// 结果按 toolCallId 配对成完整工具项；未配对的孤儿 tool 行降级为独立卡
// （name+输出，诚实展示）。零 React 依赖，可独立单测。

import type { SubagentTranscriptMessage } from "./types";

export type SubagentRenderItem =
  | { type: "system" | "user"; text: string }
  | { type: "assistant"; id: string; text: string; reasoning?: string; live: boolean }
  | { type: "tool"; id: string; name: string; args: string; output?: string; pending: boolean };

export function buildRenderItems(messages: SubagentTranscriptMessage[], running: boolean): SubagentRenderItem[] {
  const items: SubagentRenderItem[] = [];
  const openByCallId = new Map<string, Extract<SubagentRenderItem, { type: "tool" }>>();
  messages.forEach((m, idx) => {
    if (m.role === "assistant") {
      for (const tc of m.toolCalls ?? []) {
        const item: Extract<SubagentRenderItem, { type: "tool" }> = {
          type: "tool",
          id: tc.id || `tc-${idx}-${items.length}`,
          name: tc.name,
          args: tc.arguments,
          pending: true,
        };
        items.push(item);
        if (tc.id) openByCallId.set(tc.id, item);
      }
      items.push({
        type: "assistant",
        id: `a-${idx}`,
        text: m.content ?? "",
        reasoning: m.reasoning || undefined,
        live: running && idx === messages.length - 1,
      });
      return;
    }
    if (m.role === "tool") {
      const paired = m.toolCallId ? openByCallId.get(m.toolCallId) : undefined;
      if (paired) {
        paired.output = m.content ?? "";
        paired.pending = false;
      } else {
        items.push({
          type: "tool",
          id: m.toolCallId || `tool-${idx}`,
          name: m.name || "tool",
          args: "",
          output: m.content ?? "",
          pending: false,
        });
      }
      return;
    }
    items.push({ type: m.role, text: m.content ?? "" });
  });
  return items;
}

// tool 配对状态：结果未到且运行中 → running；否则 done（transcript 无独立
// error 位，失败以输出内容/信封如实展示——与轨迹折叠同口径）。
export function toolStatus(
  item: Extract<SubagentRenderItem, { type: "tool" }>,
  running: boolean,
): "running" | "done" {
  return item.pending && running ? "running" : "done";
}
