import { describe, expect, it } from "vitest";
import { DEFAULT_WORKSPACE_TAB, WORKSPACE_TABS, WORKSPACE_TAB_IDS, WORKSPACE_GROUPS, isWorkspaceTabId, groupOfTab, defaultTabOfGroup } from "./workspaceTabs";

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
    expect(groupOfTab("subagents").id).toBe("running");
    expect(groupOfTab("stats").id).toBe("insight");
    expect(defaultTabOfGroup("files")).toBe("files");
    expect(defaultTabOfGroup("running")).toBe("tasks");
    expect(defaultTabOfGroup("outputs")).toBe("deliverables");
  });
});
