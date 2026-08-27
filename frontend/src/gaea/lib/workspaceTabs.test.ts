import { afterEach, describe, expect, it } from "vitest";
import { DEFAULT_WORKSPACE_TAB, WORKSPACE_TABS, WORKSPACE_TAB_IDS, WORKSPACE_GROUPS, isWorkspaceTabId, groupOfTab, defaultTabOfGroup, loadPersistedRightTab, savePersistedRightTab } from "./workspaceTabs";

describe("workspaceTabs 清单完整性（v3.0.8 分组收敛）", () => {
  it("扁平清单与 id 常量一一对应且无重复", () => {
    expect(WORKSPACE_TABS).toHaveLength(WORKSPACE_TAB_IDS.length);
    const ids = WORKSPACE_TABS.map((t) => t.id);
    expect(new Set(ids).size).toBe(ids.length);
    expect(ids.sort()).toEqual([...WORKSPACE_TAB_IDS].sort());
  });

  it("主 Tab（分组）为 4 个且覆盖全部子面板", () => {
    expect(WORKSPACE_GROUPS).toHaveLength(4);
    const flat = WORKSPACE_GROUPS.flatMap((g) => g.tabs.map((t) => t.id));
    expect(flat.sort()).toEqual([...WORKSPACE_TAB_IDS].sort());
  });

  it("每组都有非空 label/icon/keywords 且含默认子面板", () => {
    for (const g of WORKSPACE_GROUPS) {
      expect(g.label.trim().length).toBeGreaterThan(0);
      expect(g.icon).toBeTypeOf("function");
      expect(g.keywords.length).toBeGreaterThan(0);
      expect(g.tabs.length).toBeGreaterThan(0);
    }
  });

  it("每项都有非空 label/icon/keywords", () => {
    for (const tab of WORKSPACE_TABS) {
      expect(tab.label.trim().length).toBeGreaterThan(0);
      expect(tab.icon).toBeTypeOf("function");
      expect(tab.keywords.length).toBeGreaterThan(0);
    }
  });

  it("keywords 同时含中英文便于命令面板检索", () => {
    for (const tab of WORKSPACE_TABS) {
      const hasCn = tab.keywords.some((k) => /[\u4e00-\u9fa5]/.test(k));
      const hasEn = tab.keywords.some((k) => /^[a-z]+$/i.test(k));
      expect(hasCn, `${tab.id} 缺中文关键词`).toBe(true);
      expect(hasEn, `${tab.id} 缺英文关键词`).toBe(true);
    }
  });

  it("默认激活 Tab 是 files 且存在于清单", () => {
    expect(DEFAULT_WORKSPACE_TAB).toBe("files");
    expect(WORKSPACE_TABS.some((t) => t.id === DEFAULT_WORKSPACE_TAB)).toBe(true);
  });

  it("isWorkspaceTabId 守卫正确", () => {
    for (const id of WORKSPACE_TAB_IDS) {
      expect(isWorkspaceTabId(id)).toBe(true);
    }
    expect(isWorkspaceTabId("unknown")).toBe(false);
    expect(isWorkspaceTabId("")).toBe(false);
  });

  it("groupOfTab / defaultTabOfGroup 映射正确", () => {
    expect(groupOfTab("materials").id).toBe("files");
    expect(groupOfTab("cost").id).toBe("files"); // 成本库 = 文件组子面板（v3.1.0 接线）
    expect(groupOfTab("subagents").id).toBe("running");
    expect(groupOfTab("stats").id).toBe("insight");
    expect(defaultTabOfGroup("files")).toBe("files");
    expect(defaultTabOfGroup("running")).toBe("tasks");
    expect(defaultTabOfGroup("outputs")).toBe("deliverables");
  });
});

describe("workspaceTabs 会话隔离（蒸馏 dsh-better-sidebar 布局持久化）", () => {
  afterEach(() => {
    try {
      localStorage.removeItem("gaea.workspace.rightTab");
      localStorage.removeItem("gaea.rightPanel.v1:s1");
      localStorage.removeItem("gaea.rightPanel.v1:s2");
    } catch { /* ignore */ }
  });

  it("无记录时回退默认「文件」", () => {
    expect(loadPersistedRightTab()).toBe(DEFAULT_WORKSPACE_TAB);
  });

  it("保存后读取恢复上次选择", () => {
    savePersistedRightTab("deliverables");
    expect(loadPersistedRightTab()).toBe("deliverables");
    savePersistedRightTab("subagents");
    expect(loadPersistedRightTab()).toBe("subagents");
  });

  it("存储值损坏/非法时收敛回默认，不抛错", () => {
    try { localStorage.setItem("gaea.workspace.rightTab", "nope"); } catch { /* ignore */ }
    expect(loadPersistedRightTab()).toBe(DEFAULT_WORKSPACE_TAB);
    try { localStorage.setItem("gaea.workspace.rightTab", "123"); } catch { /* ignore */ }
    expect(loadPersistedRightTab()).toBe(DEFAULT_WORKSPACE_TAB);
  });

  it("C3 按会话读写互不影响（各会话记忆自己的面板）", () => {
    savePersistedRightTab("deliverables", "s1");
    savePersistedRightTab("subagents", "s2");
    expect(loadPersistedRightTab("s1")).toBe("deliverables");
    expect(loadPersistedRightTab("s2")).toBe("subagents");
    expect(loadPersistedRightTab()).toBe(DEFAULT_WORKSPACE_TAB); // 全局未写 → 默认
  });

  it("C3 会话 key 的非法值收敛回默认", () => {
    try { localStorage.setItem("gaea.rightPanel.v1:s1", "unknown"); } catch { /* ignore */ }
    expect(loadPersistedRightTab("s1")).toBe(DEFAULT_WORKSPACE_TAB);
  });

  it("C3 无会话 key 时写全局 key（向后兼容旧行为）", () => {
    savePersistedRightTab("stats");
    expect(loadPersistedRightTab(undefined)).toBe("stats");
    expect(loadPersistedRightTab("other")).toBe(DEFAULT_WORKSPACE_TAB);
  });
});
