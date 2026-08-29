// workbenchStorage.test.ts — S2.2 工作台 localStorage 空间分键
import { afterEach, describe, expect, it } from "vitest";
import { workbenchKey, readWorkbenchValue, writeWorkbenchValue } from "./workbenchStorage";

afterEach(() => {
  try {
    localStorage.removeItem("gaea.work.chatTab");
    localStorage.removeItem("gaea.chatTab");
  } catch { /* ignore */ }
});

describe("workbenchKey 空间分键", () => {
  it("gaea.<x> → gaea.work.<x>，前缀与会话 key 同样映射", () => {
    expect(workbenchKey("gaea.chatTab")).toBe("gaea.work.chatTab");
    expect(workbenchKey("gaea.workspace.rightTab")).toBe("gaea.work.workspace.rightTab");
    expect(workbenchKey("gaea.rightPanel.v1:s1")).toBe("gaea.work.rightPanel.v1:s1");
    expect(workbenchKey("other")).toBe("other"); // 非 gaea. 前缀不动
  });
});

describe("readWorkbenchValue / writeWorkbenchValue（旧 key 兼容迁移）", () => {
  it("写入只走空间分键；读取优先分键", () => {
    writeWorkbenchValue("gaea.chatTab", "context");
    expect(localStorage.getItem("gaea.work.chatTab")).toBe("context");
    expect(localStorage.getItem("gaea.chatTab")).toBeNull(); // 不写旧 key
    expect(readWorkbenchValue("gaea.chatTab")).toBe("context");
  });

  it("旧 key 值只读回退（迁移）：分键无值时读到旧值", () => {
    localStorage.setItem("gaea.chatTab", "trajectory");
    expect(readWorkbenchValue("gaea.chatTab")).toBe("trajectory");
    // 分键值优先
    localStorage.setItem("gaea.work.chatTab", "chat");
    expect(readWorkbenchValue("gaea.chatTab")).toBe("chat");
  });
});
