// 面板启用集 store（v4.524「可选使用」红线的响应式单源）。
//
// Why: lib/workspaceTabs.ts 的 loadEnabledTabs/saveEnabledTabs/resolveEnabledTabs
// 是纯 localStorage 读写（v4.86 声明，当时零消费）；CAD 图纸卡（defaultEnabled:
// false）成为首个可选面板后，欢迎网格 / ＋下拉 / 命令面板三处入口必须按同一份
// 启用集收敛，且跨组件即时一致（管理开关在 WorkspacePane，命令面板在 App）——
// 各自 useState 读档会漂移，故按 paneTabs 先例收 zustand store。
//
// 语义：overrides 持久化全局键（学 better-sidebar prefs.tabsEnabled 的
// booleanMapOf——坏值净化、缺省 = 清单 defaultEnabled）；enabled =
// resolveEnabledTabs(overrides)（含全禁兜底：右栏永远至少一个可用面板）。
// 已打开的 tab 不因停用而强关——启用集只管「入口可见性」，关 tab 是用户操作。

import { create } from "zustand";
import {
  loadEnabledTabs,
  resolveEnabledTabs,
  saveEnabledTabs,
  type WorkspaceEnabledMap,
  type WorkspaceTabId,
} from "./workspaceTabs";

interface PaneEnabledState {
  /** 用户覆盖集（持久化权威）。 */
  overrides: WorkspaceEnabledMap;
  /** 覆盖集 + 清单默认 → 全量启用表（渲染与派生的权威入参）。 */
  enabled: Record<WorkspaceTabId, boolean>;
  /** 切换开关：写持久化 + 同步重算派生表。 */
  setEnabled: (id: WorkspaceTabId, on: boolean) => void;
}

export const usePaneEnabledStore = create<PaneEnabledState>()((set) => ({
  overrides: loadEnabledTabs(),
  enabled: resolveEnabledTabs(loadEnabledTabs()),
  setEnabled: (id, on) =>
    set((s) => {
      const overrides = { ...s.overrides, [id]: on };
      saveEnabledTabs(overrides);
      return { overrides, enabled: resolveEnabledTabs(overrides) };
    }),
}));

// 注：原 resetPaneEnabledForTest 死导出已删（2026-10-27 knip 抓零使用；测试需要
// 隔离时再按用例现场补，不再预留无消费的导出面）。
