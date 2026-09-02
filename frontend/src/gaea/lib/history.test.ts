import { describe, it, expect } from "vitest";
import { rebuildHistoryItems } from "./store";
import type { HistoryMessage } from "./types";

const messages: HistoryMessage[] = [
  { role: "user", content: "帮我改方案" },
  { role: "assistant", content: "好的" },
  { role: "tool", content: "", toolId: "call_1", toolName: "edit_file", toolArgs: '{"path":"方案.md","edits":[]}' },
  { role: "tool_result", content: "已更新 方案.md", toolId: "call_1", toolName: "edit_file" },
  { role: "assistant", content: "完成" },
];

describe("rebuildHistoryItems", () => {
  it("保留用户/助手正文顺序", () => {
    const { items } = rebuildHistoryItems(messages);
    const texts = items
      .filter((it) => it.kind === "user" || it.kind === "assistant")
      .map((it) => (it.kind === "user" ? it.text : it.text));
    expect(texts).toEqual(["帮我改方案", "好的", "完成"]);
  });

  it("把工具 dispatch 渲染为完成态卡片并合并结果", () => {
    const { items } = rebuildHistoryItems(messages);
    const tool = items.find((it) => it.kind === "tool" && it.id === "call_1");
    expect(tool).toBeDefined();
    if (tool && tool.kind === "tool") {
      expect(tool.name).toBe("edit_file");
      expect(tool.args).toContain("方案.md");
      expect(tool.status).toBe("done");
      expect(tool.output).toBe("已更新 方案.md");
    }
  });

  it("没有结果条目时输出为空、状态为 stopped（中断而非假完成）", () => {
    const { items } = rebuildHistoryItems([
      { role: "user", content: "hi" },
      { role: "tool", content: "", toolId: "c2", toolName: "write_file", toolArgs: '{"path":"a.md"}' },
    ]);
    const tool = items.find((it) => it.kind === "tool");
    expect(tool && tool.kind === "tool" ? tool.output : null).toBe("");
    expect(tool && tool.kind === "tool" ? tool.status : null).toBe("stopped");
  });

  it("计算最后一个助手索引", () => {
    const { lastAssistantIdx } = rebuildHistoryItems(messages);
    const last = rebuildHistoryItems(messages).items[lastAssistantIdx];
    expect(last && last.kind === "assistant" ? last.text : null).toBe("完成");
  });

  // v4.34 线B：assistant 历史条目透传 subagentRef（Go HistoryMessage.SubagentRef），
  // 恢复会话后「子代理」徽标数据就位。渲染层徽标行为已由 components/Message.test.tsx
  // 对同一 assistant Item 形状覆盖，此处只测数据层透传。
  it("assistant 带 subagentRef：透传到 item（徽标数据就位）", () => {
    const { items } = rebuildHistoryItems([
      { role: "user", content: "跑个子任务" },
      { role: "assistant", content: "子代理答复", subagentRef: "sa_20260902_01" },
    ]);
    const a = items.find((it) => it.kind === "assistant");
    expect(a && a.kind === "assistant" ? a.subagentRef : null).toBe("sa_20260902_01");
  });

  it("assistant 不带 subagentRef：字段 undefined（旧行为不变）", () => {
    const { items } = rebuildHistoryItems([{ role: "assistant", content: "主回答" }]);
    const a = items.find((it) => it.kind === "assistant");
    expect(a && a.kind === "assistant" ? a.subagentRef : null).toBeUndefined();
  });

  it("空串 subagentRef 归一为 undefined（避免徽标空渲染）", () => {
    const { items } = rebuildHistoryItems([{ role: "assistant", content: "回答", subagentRef: "" }]);
    const a = items.find((it) => it.kind === "assistant");
    expect(a && a.kind === "assistant" ? a.subagentRef : null).toBeUndefined();
  });
});
