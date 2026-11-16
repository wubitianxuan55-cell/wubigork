import { beforeEach, describe, expect, it } from "vitest";
import {
  getComposerDraft,
  makeComposerSessionKey,
  resetComposerDraftsForTests,
  saveComposerDraft,
} from "./composerDrafts";

describe("composerDrafts", () => {
  beforeEach(() => resetComposerDraftsForTests());

  it("按空间、会话路径和未保存会话 nonce 隔离", () => {
    expect(makeComposerSessionKey("work", "C:/sessions/a.jsonl", "C:/ws", 2))
      .not.toBe(makeComposerSessionKey("play", "C:/sessions/a.jsonl", "C:/ws", 2));
    expect(makeComposerSessionKey("work", undefined, "C:/ws", 1))
      .not.toBe(makeComposerSessionKey("work", undefined, "C:/ws", 2));
  });

  it("不同会话草稿互不串话，空草稿自动删除", () => {
    saveComposerDraft("work:a", {
      text: "办公草稿",
      attachments: [{ path: "C:/ws/a.docx", previewUrl: "", type: "file", size: 12 }],
      queue: [{ id: "q1", text: "稍后发送", status: "pending" }],
    });
    saveComposerDraft("work:b", { text: "另一会话", attachments: [], queue: [] });

    expect(getComposerDraft("work:a").text).toBe("办公草稿");
    expect(getComposerDraft("work:b").text).toBe("另一会话");
    expect(getComposerDraft("work:a").attachments[0].path).toBe("C:/ws/a.docx");

    saveComposerDraft("work:b", { text: "", attachments: [], queue: [] });
    expect(getComposerDraft("work:b").text).toBe("");
  });

  it("持久化不保存大图预览和待发送队列，恢复时保留文本与附件路径", () => {
    saveComposerDraft("play:a", {
      text: "下一章从这里继续",
      attachments: [{
        path: "C:/tuwen/book/参考.png",
        previewUrl: "data:image/png;base64,AAAA",
        type: "image",
        size: 3,
      }],
      queue: [{ id: "q9", text: "这条队列不跨重启", status: "pending" }],
    });

    resetComposerDraftsForTests(false);
    const restored = getComposerDraft("play:a");
    expect(restored.text).toBe("下一章从这里继续");
    expect(restored.attachments).toEqual([{
      path: "C:/tuwen/book/参考.png",
      previewUrl: "",
      type: "image",
      size: 3,
    }]);
    expect(restored.queue).toEqual([]);
  });
});
