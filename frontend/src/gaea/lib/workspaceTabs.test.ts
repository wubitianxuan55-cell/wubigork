import { afterEach, describe, expect, it } from "vitest";
import {
  DEFAULT_WORKSPACE_TAB, WORKSPACE_TABS, WORKSPACE_TAB_IDS, WORKSPACE_GROUPS, isWorkspaceTabId, groupOfTab, defaultTabOfGroup,
  loadPersistedRightTab, savePersistedRightTab,
  WORKSPACE_MIN_WIDTH, WORKSPACE_MAX_WIDTH, WORKSPACE_DEFAULT_WIDTH,
  clampWorkspaceWidth, loadWorkspaceWidth, saveWorkspaceWidth,
  loadEnabledTabs, saveEnabledTabs, sanitizeEnabledTabs, resolveEnabledTabs, firstEnabledTab,
  loadPersistedRightPanelState, savePersistedRightPanelState,
  type WorkspaceEnabledMap,
} from "./workspaceTabs";

// v4.23 新增持久化键的统一清理（含 layoutPreferences 全局 blob）
function cleanupStorage(): void {
  const keys = [
    "gaea.work.workspace.rightTab", "gaea.workspace.rightTab",
    "gaea.work.rightPanel.v1:s1", "gaea.work.rightPanel.v1:s2",
    "gaea.rightPanel.v1:s1", "gaea.rightPanel.v1:s2",
    "gaea.work.rightPanel.v1:tabsEnabled", "gaea.rightPanel.v1:tabsEnabled",
    "gaea.work.layoutPreferences.v1", "gaea.layoutPreferences.v1",
    "gaea.work.workspacePanel.width", "gaea.workspacePanel.width",
  ];
  for (const key of keys) {
    try { localStorage.removeItem(key); } catch { /* ignore */ }
  }
}

describe("workspaceTabs 清单完整性（v3.0.8 分组收敛）", () => {
  it("扁平清单与 id 常量一一对应且无重复", () => {
    expect(WORKSPACE_TABS).toHaveLength(WORKSPACE_TAB_IDS.length);
    const ids = WORKSPACE_TABS.map((t) => t.id);
    expect(new Set(ids).size).toBe(ids.length);
    expect(ids.sort()).toEqual([...WORKSPACE_TAB_IDS].sort());
  });

  it("主 Tab（分组）为 3 个且覆盖全部子面板", () => {
    expect(WORKSPACE_GROUPS).toHaveLength(3);
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
    expect(groupOfTab("changes").id).toBe("outputs");
    expect(defaultTabOfGroup("files")).toBe("files");
    expect(defaultTabOfGroup("running")).toBe("tasks");
    expect(defaultTabOfGroup("outputs")).toBe("deliverables");
  });
});

describe("workspaceTabs 会话隔离（蒸馏 dsh-better-sidebar 布局持久化）", () => {
  afterEach(() => {
    try {
      localStorage.removeItem("gaea.work.workspace.rightTab");
      localStorage.removeItem("gaea.work.rightPanel.v1:s1");
      localStorage.removeItem("gaea.work.rightPanel.v1:s2");
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
    savePersistedRightTab("cost");
    expect(loadPersistedRightTab(undefined)).toBe("cost");
    expect(loadPersistedRightTab("other")).toBe(DEFAULT_WORKSPACE_TAB);
  });

  it("S2.2 旧 key 只读迁移：旧全局/会话值仍被读取，新写入走空间分键", () => {
    try { localStorage.setItem("gaea.workspace.rightTab", "deliverables"); } catch { /* ignore */ }
    expect(loadPersistedRightTab()).toBe("deliverables");
    try { localStorage.setItem("gaea.rightPanel.v1:s1", "cost"); } catch { /* ignore */ }
    expect(loadPersistedRightTab("s1")).toBe("cost");
    savePersistedRightTab("subagents", "s1");
    expect(localStorage.getItem("gaea.work.rightPanel.v1:s1")).toBe("subagents");
    expect(loadPersistedRightTab("s1")).toBe("subagents"); // 分键优先于旧值
  });
});

// ── v4.23 工作台外壳（蒸馏 dsh-better-sidebar：全局宽度键 / 启用集 / 记录 v2）──

describe("workspaceTabs 宽度契约（全局键：最后一次拖拽胜出）", () => {
  afterEach(cleanupStorage);

  it("钳制到 280–720 并取整", () => {
    expect(WORKSPACE_MIN_WIDTH).toBe(280);
    expect(WORKSPACE_MAX_WIDTH).toBe(720);
    expect(clampWorkspaceWidth(100)).toBe(280);
    expect(clampWorkspaceWidth(279.6)).toBe(280);
    expect(clampWorkspaceWidth(500.4)).toBe(500);
    expect(clampWorkspaceWidth(9999)).toBe(720);
  });

  it("默认宽度 340 对齐 styles.css --workspace-width 基线", () => {
    expect(WORKSPACE_DEFAULT_WIDTH).toBe(340);
    expect(loadWorkspaceWidth()).toBe(340);
  });

  it("保存后读取走全局键（跨会话跟随）", () => {
    saveWorkspaceWidth(560);
    expect(loadWorkspaceWidth()).toBe(560);
    // 换一个「会话」读取：宽度是布局偏好，全局生效
    expect(loadWorkspaceWidth(null)).toBe(560);
    expect(loadWorkspaceWidth(300)).toBe(560); // 全局键胜出会话快照
  });

  it("全局键缺省时用会话快照兜底（首装 seed）", () => {
    expect(loadWorkspaceWidth(600)).toBe(600);
    expect(loadWorkspaceWidth(9999)).toBe(720); // 快照同样被钳制
    expect(loadWorkspaceWidth(-5)).toBe(340); // 非法快照回默认
    expect(loadWorkspaceWidth(null)).toBe(340);
  });
});

describe("workspaceTabs 声明式设置启用集（全局键，学 booleanMapOf 净化）", () => {
  afterEach(cleanupStorage);

  it("无记录时为空覆盖（全部按清单 defaultEnabled 启用）", () => {
    expect(loadEnabledTabs()).toEqual({});
    const resolved = resolveEnabledTabs({});
    for (const id of WORKSPACE_TAB_IDS) expect(resolved[id]).toBe(true);
  });

  it("保存后读取往返（工作台空间分键）", () => {
    const map: WorkspaceEnabledMap = { changes: false, files: true };
    saveEnabledTabs(map);
    expect(localStorage.getItem("gaea.work.rightPanel.v1:tabsEnabled")).toBe(JSON.stringify(map));
    expect(loadEnabledTabs()).toEqual(map);
  });

  it("sanitize 丢弃非法 id 与非布尔值（学 sanitizeState：坏值逐项丢弃不崩）", () => {
    expect(sanitizeEnabledTabs({ changes: false, nope: true, files: "yes", cost: 1 })).toEqual({ changes: false });
    expect(sanitizeEnabledTabs(null)).toEqual({});
    expect(sanitizeEnabledTabs(["files"])).toEqual({});
    expect(sanitizeEnabledTabs("x")).toEqual({});
    try { localStorage.setItem("gaea.work.rightPanel.v1:tabsEnabled", "{bad json"); } catch { /* ignore */ }
    expect(loadEnabledTabs()).toEqual({});
  });

  it("resolveEnabledTabs 合并默认与覆盖；全禁时兜底回默认集", () => {
    expect(resolveEnabledTabs({ changes: false }).changes).toBe(false);
    expect(resolveEnabledTabs({ changes: false }).files).toBe(true);
    const allOff: WorkspaceEnabledMap = {};
    for (const id of WORKSPACE_TAB_IDS) allOff[id] = false;
    const rescued = resolveEnabledTabs(allOff);
    for (const id of WORKSPACE_TAB_IDS) expect(rescued[id]).toBe(true);
  });

  it("firstEnabledTab 按组序/组内序回退到第一个启用面板", () => {
    const allOn = resolveEnabledTabs({});
    expect(firstEnabledTab(allOn)).toBe("files");
    expect(firstEnabledTab({ ...allOn, files: false })).toBe("materials");
    expect(firstEnabledTab({ ...allOn, files: false, materials: false, cost: false })).toBe("deliverables");
    expect(firstEnabledTab(resolveEnabledTabs({}))).toBe("files");
  });
});

describe("workspaceTabs 会话记录 v2（激活 tab + 启用集快照 + 宽度快照，兼容 v1 裸 id）", () => {
  afterEach(cleanupStorage);

  it("v2 记录往返：tab/enabled/width 均恢复，宽度落盘前钳制", () => {
    savePersistedRightPanelState({ v: 1, tab: "cost", enabled: { cost: false }, width: 99999 }, "s1");
    const loaded = loadPersistedRightPanelState("s1");
    expect(loaded).toEqual({ v: 1, tab: "cost", enabled: { cost: false }, width: 720 });
    expect(loadPersistedRightTab("s1")).toBe("cost");
  });

  it("v4.22 会话记录的 tab:\"stats\"（已迁主区概览）宽容收敛回默认「文件」", () => {
    try { localStorage.setItem("gaea.rightPanel.v1:s1", JSON.stringify({ v: 1, tab: "stats", enabled: null, width: 400 })); } catch { /* ignore */ }
    const loaded = loadPersistedRightPanelState("s1");
    expect(loaded.tab).toBe(DEFAULT_WORKSPACE_TAB);
    expect(loaded.width).toBe(400); // 合法字段照常恢复
  });

  it("v1 裸 id 旧值仍可读（向后兼容），enabled/width 视为未记录", () => {
    try { localStorage.setItem("gaea.work.rightPanel.v1:s1", "deliverables"); } catch { /* ignore */ }
    expect(loadPersistedRightPanelState("s1")).toEqual({
      v: 1, tab: "deliverables", enabled: null, width: null,
    });
  });

  it("损坏/非法值宽容净化：坏 JSON、坏 tab、坏字段逐项兜底", () => {
    try { localStorage.setItem("gaea.work.rightPanel.v1:s1", "not json {{{"); } catch { /* ignore */ }
    expect(loadPersistedRightPanelState("s1")).toEqual({ v: 1, tab: DEFAULT_WORKSPACE_TAB, enabled: null, width: null });
    try { localStorage.setItem("gaea.work.rightPanel.v1:s1", JSON.stringify({ tab: "nope", width: "x", enabled: 3 })); } catch { /* ignore */ }
    expect(loadPersistedRightPanelState("s1")).toEqual({ v: 1, tab: DEFAULT_WORKSPACE_TAB, enabled: {}, width: null });
    try { localStorage.setItem("gaea.work.rightPanel.v1:s1", JSON.stringify({ tab: "files", enabled: ["files"], width: -3 })); } catch { /* ignore */ }
    const loaded = loadPersistedRightPanelState("s1");
    expect(loaded.tab).toBe("files");
    expect(loaded.enabled).toEqual({}); // 数组 → 空覆盖（逐项丢弃）
    expect(loaded.width).toBeNull(); // 非法宽度 → 未记录
  });

  it("无记录 / 无会话 key 时回退默认（与 v1 行为一致）", () => {
    expect(loadPersistedRightPanelState()).toEqual({ v: 1, tab: DEFAULT_WORKSPACE_TAB, enabled: null, width: null });
    expect(loadPersistedRightTab()).toBe(DEFAULT_WORKSPACE_TAB);
  });
});
