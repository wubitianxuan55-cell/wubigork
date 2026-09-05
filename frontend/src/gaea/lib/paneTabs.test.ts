import { beforeEach, describe, expect, it } from "vitest";
import { resetPaneTabsForTest, usePaneTabsStore } from "./paneTabs";

beforeEach(() => resetPaneTabsForTest());

describe("paneTabs 右栏 pane tab 状态机（对标 better-sidebar）", () => {
  it("初始为空 = 欢迎卡片态", () => {
    const s = usePaneTabsStore.getState();
    expect(s.tabs).toEqual([]);
    expect(s.active).toBeNull();
  });

  it("点「文件」卡片 → 生成一个视图 tab（不是常驻）", () => {
    usePaneTabsStore.getState().openView("files", "文件");
    const s = usePaneTabsStore.getState();
    expect(s.tabs).toEqual([{ id: "view:files", kind: "view", viewId: "files", title: "文件" }]);
    expect(s.active).toBe("view:files");
  });

  it("再次开同一视图 → 去重聚焦，不新增 tab", () => {
    const api = usePaneTabsStore.getState();
    api.openView("files", "文件");
    api.openView("tasks", "任务");
    api.openView("files", "文件");
    const s = usePaneTabsStore.getState();
    expect(s.tabs).toHaveLength(2);
    expect(s.active).toBe("view:files");
  });

  it("资源管理器内点文件 → 新增一个文件 tab，两个 tab 并存", () => {
    const api = usePaneTabsStore.getState();
    api.openView("files", "文件");
    api.openFile("README.md", "README.md");
    const s = usePaneTabsStore.getState();
    expect(s.tabs.map((t) => t.id)).toEqual(["view:files", "file:README.md"]);
    expect(s.active).toBe("file:README.md");
  });

  it("打开已开文件 → 去重聚焦", () => {
    const api = usePaneTabsStore.getState();
    api.openFile("a.md", "a.md");
    api.openFile("b.md", "b.md");
    api.openFile("a.md", "a.md");
    const s = usePaneTabsStore.getState();
    expect(s.tabs.map((t) => t.id)).toEqual(["file:a.md", "file:b.md"]);
    expect(s.active).toBe("file:a.md");
  });

  it("关闭激活 tab：先右邻；末位取左邻", () => {
    const api = usePaneTabsStore.getState();
    api.openFile("a.md", "a.md");
    api.openFile("b.md", "b.md");
    api.openFile("c.md", "c.md");
    api.activate("file:b.md");
    api.close("file:b.md");
    expect(usePaneTabsStore.getState().active).toBe("file:c.md");

    api.activate("file:c.md");
    api.close("file:c.md");
    expect(usePaneTabsStore.getState().active).toBe("file:a.md");
  });

  it("关闭全部 tab → 回到欢迎卡片态", () => {
    const api = usePaneTabsStore.getState();
    api.openView("files", "文件");
    api.openFile("README.md", "README.md");
    api.close("file:README.md");
    api.close("view:files");
    const s = usePaneTabsStore.getState();
    expect(s.tabs).toEqual([]);
    expect(s.active).toBeNull();
  });

  it("按会话持久化：切走再切回恢复该会话的 tabs", () => {
    const api = usePaneTabsStore.getState();
    api.setSessionKey("sess-1");
    api.openView("files", "文件");
    api.openFile("README.md", "README.md");
    api.setSessionKey(null);
    expect(usePaneTabsStore.getState().tabs).toEqual([]);
    api.setSessionKey("sess-1");
    const s = usePaneTabsStore.getState();
    expect(s.tabs.map((t) => t.id)).toEqual(["view:files", "file:README.md"]);
    expect(s.active).toBe("file:README.md");
  });
});

// ── U4 写后预览实时跟随：reloadTicks 刷新总线（pane 文件 tab 与主区大预览共用）──
describe("paneTabs reloadTicks 写后预览刷新总线（U4）", () => {
  it("requestReload 递增对应路径序号；未刷过的路径取 0", () => {
    expect(usePaneTabsStore.getState().reloadTicks["docs/a.docx"] ?? 0).toBe(0);
    usePaneTabsStore.getState().requestReload("docs/a.docx");
    usePaneTabsStore.getState().requestReload("docs/a.docx");
    // zustand getState 是快照：断言前重取最新 state
    expect(usePaneTabsStore.getState().reloadTicks["docs/a.docx"]).toBe(2);
  });

  it("键走归一口径：反斜杠/大小写不同的同一文件命中同一序号", () => {
    usePaneTabsStore.getState().requestReload("Docs\\报告.DOCX");
    expect(usePaneTabsStore.getState().reloadTicks["docs/报告.docx"]).toBe(1);
  });

  it("空路径不动作", () => {
    usePaneTabsStore.getState().requestReload("");
    expect(usePaneTabsStore.getState().reloadTicks).toEqual({});
  });

  it("会话切换清空刷新序号（瞬态不跨会话、不落盘）", () => {
    usePaneTabsStore.getState().setSessionKey("sess-1");
    usePaneTabsStore.getState().requestReload("a.docx");
    expect(Object.keys(usePaneTabsStore.getState().reloadTicks).length).toBe(1);
    usePaneTabsStore.getState().setSessionKey("sess-2");
    expect(usePaneTabsStore.getState().reloadTicks).toEqual({});
  });
});
