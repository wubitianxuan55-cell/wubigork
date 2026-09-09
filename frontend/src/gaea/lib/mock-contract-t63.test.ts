// T6-3「对话·流可靠」契约层 mock 冒烟测试：
// ChatTopicsList/ChatMessagesList 成功返回 T[] 数组（Go 侧 ([]T, error)，
// Wails 绑定后失败为 rejected promise；历史曾误标 [T[], unknown] 元组，
// 已修正为数组）；ChatAppendMessages 为新增绑定（语音消息持久化）。
// 锁定 AppBindings 契约 + mock 实现形状。
import { describe, expect, it } from "vitest";
import { app } from "./bridge";

describe("T6-3 对话 chat 契约同步", () => {
  it("ChatAppendMessages 存在于 AppBindings（契约：可调用、不 undefined）", async () => {
    expect(typeof app.ChatAppendMessages).toBe("function");
    // mock 实现为 no-op（浏览器开发环境不落库），成功 resolve undefined
    await expect(app.ChatAppendMessages("t-mock", [])).resolves.toBeUndefined();
  });

  it("ChatTopicsList mock 返回 T[] 数组（成功：数组形态，失败即 rejected）", async () => {
    const topics = await app.ChatTopicsList();
    expect(Array.isArray(topics)).toBe(true);
  });

  it("ChatMessagesList mock 返回 T[] 数组（成功：数组形态，失败即 rejected）", async () => {
    const msgs = await app.ChatMessagesList("t-mock");
    expect(Array.isArray(msgs)).toBe(true);
  });

  it("ChatMessagesList mock 接收话题 ID 参数", async () => {
    // 契约：调用点传 topicID（原 () => Topic[] 的无参形态已废弃）
    const msgs = await app.ChatMessagesList("topic-42");
    expect(Array.isArray(msgs)).toBe(true);
  });
});
