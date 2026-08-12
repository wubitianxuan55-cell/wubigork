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

  it("没有结果条目时输出为空但卡片仍存在", () => {
    const { items } = rebuildHistoryItems([
      { role: "user", content: "hi" },
      { role: "tool", content: "", toolId: "c2", toolName: "write_file", toolArgs: '{"path":"a.md"}' },
    ]);
    const tool = items.find((it) => it.kind === "tool");
    expect(tool && tool.kind === "tool" ? tool.output : null).toBe("");
  });

  it("计算最后一个助手索引", () => {
    const { lastAssistantIdx } = rebuildHistoryItems(messages);
    const last = rebuildHistoryItems(messages).items[lastAssistantIdx];
    expect(last && last.kind === "assistant" ? last.text : null).toBe("完成");
  });
});
