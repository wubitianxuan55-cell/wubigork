// v4.25 A3 编辑器 tab 状态机测试：开/关/激活/上限 LRU/持久化与坏存储兜底/
// 命令式 API（App sidebar_open 事件侧入口）。
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import {
  EDITOR_TABS_MAX,
  EDITOR_TABS_STORAGE_KEY,
  activateEditorTab,
  closeEditorTab,
  loadPersistedEditorTabs,
  openEditorTab,
  resetEditorTabsForTest,
  sanitizeEditorTabsSnapshot,
  useEditorTabsStore,
} from "./editorTabs";

function tabs(): string[] {
  return useEditorTabsStore.getState().tabs;
}
function active(): string | null {
  return useEditorTabsStore.getState().active;
}

describe("editorTabs 状态机（v4.25 A3）", () => {
  beforeEach(() => resetEditorTabsForTest());
  afterEach(() => localStorage.clear());

  it("open：新文件追加并激活", () => {
    useEditorTabsStore.getState().open("a.md");
    useEditorTabsStore.getState().open("docs/b.md");
    expect(tabs()).toEqual(["a.md", "docs/b.md"]);
    expect(active()).toBe("docs/b.md");
  });

  it("open：已开文件激活去重，不改变 tab 顺序", () => {
    useEditorTabsStore.getState().open("a.md");
    useEditorTabsStore.getState().open("b.md");
    useEditorTabsStore.getState().open("a.md");
    expect(tabs()).toEqual(["a.md", "b.md"]);
    expect(active()).toBe("a.md");
  });

  it("open：空路径忽略", () => {
    useEditorTabsStore.getState().open("");
    expect(tabs()).toEqual([]);
    expect(active()).toBeNull();
  });

  it("close：关非激活 tab，激活态不变", () => {
    const s = useEditorTabsStore.getState();
    s.open("a.md");
    s.open("b.md");
    s.close("a.md");
    expect(tabs()).toEqual(["b.md"]);
    expect(active()).toBe("b.md");
  });

  it("close：关激活 tab 激活右邻；原为末位取左邻；唯一 tab 关后清空", () => {
    const s = useEditorTabsStore.getState();
    s.open("a.md");
    s.open("b.md");
    s.open("c.md");
    s.close("b.md");
    expect(active()).toBe("c.md"); // 右邻
    s.close("c.md");
    expect(active()).toBe("a.md"); // 末位 → 左邻
    s.close("a.md");
    expect(tabs()).toEqual([]);
    expect(active()).toBeNull();
  });

  it("close：未开的路径 no-op", () => {
    const s = useEditorTabsStore.getState();
    s.open("a.md");
    s.close("ghost.md");
    expect(tabs()).toEqual(["a.md"]);
    expect(active()).toBe("a.md");
  });

  it("activate：切换激活 tab；未开路径 no-op", () => {
    const s = useEditorTabsStore.getState();
    s.open("a.md");
    s.open("b.md");
    s.activate("a.md");
    expect(active()).toBe("a.md");
    s.activate("ghost.md");
    expect(active()).toBe("a.md");
  });

  it(`上限 ${EDITOR_TABS_MAX}：超限 LRU 驱逐最久未激活（被触碰的旧 tab 存活）`, () => {
    const s = useEditorTabsStore.getState();
    for (let i = 1; i <= EDITOR_TABS_MAX; i++) s.open(`t${i}.md`);
    expect(tabs()).toHaveLength(EDITOR_TABS_MAX);
    // activate(t1) 触碰最旧的 tab；再开新 tab → 驱逐对象应是 t2（次旧且未触碰）
    s.activate("t1.md");
    s.open("t13.md");
    expect(tabs()).toHaveLength(EDITOR_TABS_MAX);
    expect(tabs()).toContain("t1.md");
    expect(tabs()).not.toContain("t2.md");
    expect(active()).toBe("t13.md");
  });

  it("上限 12：已开文件重复 open（激活）不触发驱逐", () => {
    const s = useEditorTabsStore.getState();
    for (let i = 1; i <= EDITOR_TABS_MAX; i++) s.open(`t${i}.md`);
    s.open("t1.md");
    expect(tabs()).toHaveLength(EDITOR_TABS_MAX);
    expect(tabs()).toContain("t1.md");
  });
});

describe("editorTabs 持久化（坏值兜底回空）", () => {
  beforeEach(() => resetEditorTabsForTest());
  afterEach(() => localStorage.clear());

  it("open/close/activate 变化落盘 localStorage", () => {
    useEditorTabsStore.getState().open("a.md");
    useEditorTabsStore.getState().open("docs/b.md");
    const raw = localStorage.getItem(EDITOR_TABS_STORAGE_KEY);
    expect(raw).not.toBeNull();
    const snap = JSON.parse(raw as string);
    expect(snap.v).toBe(1);
    expect(snap.tabs).toEqual(["a.md", "docs/b.md"]);
    expect(snap.active).toBe("docs/b.md");
    // 关闭后落盘同步更新
    closeEditorTab("docs/b.md");
    expect(JSON.parse(localStorage.getItem(EDITOR_TABS_STORAGE_KEY) as string).active).toBe("a.md");
  });

  it("loadPersistedEditorTabs：坏 JSON 回空", () => {
    localStorage.setItem(EDITOR_TABS_STORAGE_KEY, "{not json");
    const snap = loadPersistedEditorTabs();
    expect(snap.tabs).toEqual([]);
    expect(snap.active).toBeNull();
  });

  it("loadPersistedEditorTabs：非对象/数组回空", () => {
    localStorage.setItem(EDITOR_TABS_STORAGE_KEY, JSON.stringify([1, 2]));
    expect(loadPersistedEditorTabs().tabs).toEqual([]);
    expect(loadPersistedEditorTabs().active).toBeNull();
  });

  it("sanitize：半坏值逐字段兜底（坏路径丢弃/去重/封顶/激活指针失效收敛回首项）", () => {
    const raw = {
      v: 1,
      tabs: [42, "a.md", "a.md", "", "b.md"],
      active: "ghost.md",
      lastActiveAt: { "a.md": "x", "b.md": 3, ghost: 9 },
    };
    const snap = sanitizeEditorTabsSnapshot(raw);
    expect(snap.tabs).toEqual(["a.md", "b.md"]);
    expect(snap.active).toBe("a.md"); // 激活指针不在 tabs 内 → 收敛首项
    expect(snap.lastActiveAt).toEqual({ "b.md": 3 }); // 坏值丢弃、未开路径丢弃
  });

  it("sanitize：非对象输入回空", () => {
    expect(sanitizeEditorTabsSnapshot(null).tabs).toEqual([]);
    expect(sanitizeEditorTabsSnapshot("junk").active).toBeNull();
    expect(sanitizeEditorTabsSnapshot(3).tabs).toEqual([]);
  });

  it("重启恢复：落盘后 loadPersistedEditorTabs 读回同一份状态", () => {
    useEditorTabsStore.getState().open("a.md");
    useEditorTabsStore.getState().open("docs/b.md");
    const restored = loadPersistedEditorTabs();
    expect(restored.tabs).toEqual(["a.md", "docs/b.md"]);
    expect(restored.active).toBe("docs/b.md");
    expect(typeof restored.lastActiveAt["a.md"]).toBe("number");
  });
});

describe("editorTabs 命令式 API（App sidebar_open 事件侧入口）", () => {
  beforeEach(() => resetEditorTabsForTest());
  afterEach(() => localStorage.clear());

  it("openEditorTab/closeEditorTab/activateEditorTab 驱动同一 store", () => {
    openEditorTab("a.md");
    openEditorTab("b.md");
    expect(active()).toBe("b.md");
    activateEditorTab("a.md");
    expect(active()).toBe("a.md");
    closeEditorTab("a.md");
    expect(tabs()).toEqual(["b.md"]);
    expect(active()).toBe("b.md");
  });
});
