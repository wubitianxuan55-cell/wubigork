// 右侧面板 Tab 体系声明（办公工作台装配层的轻量 manifest）。
//
// Why: 评审 docs/gaea3-review/03-office-frontend.md 缺陷 4 —— 右侧面板的
// Tab 入口此前散在四处（rightTab 联合类型、App.tsx Tab 按钮条、渲染分支、
// CommandPalette 面板项），新增/调整一个 Tab 要同步改多处且图标映射曾两处
// 不一致（资料/产物在按钮条与命令面板中图标互换）。本文件收敛为单一数据源：
// App.tsx 的 Tab 按钮条、CommandPalette 面板项、激活态判定全部从这里派生。
//
// v3.0.8 收敛（用户决策「只保留 4 个标签」）：7 个子面板按「文件域 / 成果域 /
// 运行域 / 分析域」归入 4 个主 Tab（WORKSPACE_GROUPS），主 Tab 条只显示 4 个
// 组名，组内子面板由 WorkspaceTabs 第二级小 Tab 切换。7 个功能全保留、层级
// 清晰，避免右侧面板堆成杂物抽屉。
//
// How to apply: 新增子面板 → 在对应组的 tabs 追加一项（id/label/icon/keywords），
// 再在 App.tsx 的 workspacePanelRender 分支补对应面板渲染；按钮条与命令
// 面板零改动。label 与现状一致使用中文（未接 i18n，与既有行为对齐）。

import type { Icon } from "../icons";
import { BarChart3, ClipboardList, Coins, Diff, FileText, FolderTree, Paperclip, Users } from "../icons";

/** 子面板（具体功能页）的稳定 id —— App.tsx 的 rightTab 状态直接消费。 */
export const WORKSPACE_TAB_IDS = ["files", "materials", "cost", "deliverables", "changes", "stats", "tasks", "subagents"] as const;
export type WorkspaceTabId = (typeof WORKSPACE_TAB_IDS)[number];

/** 主 Tab（分组）的稳定 id。 */
export const WORKSPACE_GROUP_IDS = ["files", "outputs", "running", "insight"] as const;
export type WorkspaceGroupId = (typeof WORKSPACE_GROUP_IDS)[number];

export interface WorkspaceTabDef {
  id: WorkspaceTabId;
  /** 按钮条与命令面板共用的显示名。 */
  label: string;
  /** 图标组件引用（渲染时由消费方决定尺寸）。 */
  icon: Icon;
  /** CommandPalette 模糊搜索关键词（中英文）。 */
  keywords: string[];
}

export interface WorkspaceGroupDef {
  id: WorkspaceGroupId;
  label: string;
  icon: Icon;
  keywords: string[];
  /** 组内子面板（点击组 Tab 默认落到第一个）。 */
  tabs: WorkspaceTabDef[];
}

/** 单一数据源：右侧面板全部主 Tab（分组）的声明。 */
export const WORKSPACE_GROUPS: WorkspaceGroupDef[] = [
  {
    id: "files",
    label: "文件",
    icon: FolderTree,
    keywords: ["files", "文件", "工作区", "树", "资料"],
    tabs: [
      { id: "files", label: "文件", icon: FolderTree, keywords: ["files", "文件", "工作区", "树"] },
      { id: "materials", label: "资料", icon: Paperclip, keywords: ["materials", "资料", "素材", "钉住"] },
      { id: "cost", label: "成本库", icon: Coins, keywords: ["cost", "成本库", "造价", "单价", "价格"] },
    ],
  },
  {
    id: "outputs",
    label: "成果",
    icon: FileText,
    keywords: ["outputs", "成果", "产物", "交付", "变更"],
    tabs: [
      { id: "deliverables", label: "产物", icon: FileText, keywords: ["deliverables", "产物", "交付", "成果"] },
      { id: "changes", label: "变更", icon: Diff, keywords: ["changes", "变更", "修改", "diff"] },
    ],
  },
  {
    id: "running",
    label: "运行",
    icon: ClipboardList,
    keywords: ["running", "运行", "任务", "分工", "子代理", "进度"],
    tabs: [
      { id: "tasks", label: "任务", icon: ClipboardList, keywords: ["tasks", "任务", "进度", "调度"] },
      { id: "subagents", label: "分工", icon: Users, keywords: ["subagent", "子代理", "分工", "团队", "并发"] },
    ],
  },
  {
    id: "insight",
    label: "分析",
    icon: BarChart3,
    keywords: ["insight", "分析", "统计", "token", "成本", "用量"],
    tabs: [
      { id: "stats", label: "统计", icon: BarChart3, keywords: ["stats", "统计", "token", "成本", "用量"] },
    ],
  },
];

/** 扁平子面板清单（兼容旧导出：命令面板遍历、测试断言）。 */
export const WORKSPACE_TABS: WorkspaceTabDef[] = WORKSPACE_GROUPS.flatMap((g) => g.tabs);

/** 默认激活子面板（启动即文件面板）。 */
export const DEFAULT_WORKSPACE_TAB: WorkspaceTabId = "files";

/** 字符串 → WorkspaceTabId 类型守卫（用于外来状态/存储值收敛）。 */
export function isWorkspaceTabId(value: string): value is WorkspaceTabId {
  return (WORKSPACE_TAB_IDS as readonly string[]).includes(value);
}

/** 子面板 → 所属主 Tab（组）。 */
export function groupOfTab(tabId: WorkspaceTabId): WorkspaceGroupDef {
  return WORKSPACE_GROUPS.find((g) => g.tabs.some((t) => t.id === tabId)) ?? WORKSPACE_GROUPS[0];
}

/** 主 Tab 的默认子面板（点击组 Tab 时落到第一个）。 */
export function defaultTabOfGroup(groupId: WorkspaceGroupId): WorkspaceTabId {
  const g = WORKSPACE_GROUPS.find((x) => x.id === groupId);
  return g?.tabs[0]?.id ?? DEFAULT_WORKSPACE_TAB;
}

// ── 会话隔离（蒸馏 dsh-better-sidebar 布局持久化）────────────────────
// 记住用户上次选中的右侧面板子 Tab：重开办公板块/重启应用后恢复，而不是
// 每次回到「文件」。v3.0.8 起支持**按会话**持久化（C3 升级）：有 sessionKey
// 时写入 `gaea.rightPanel.v1:<sessionKey>`（切会话/新建/恢复各自恢复面板
// 关注点）；无 sessionKey 时回退全局 key `gaea.workspace.rightTab`（旧版
// 行为，向后兼容）。损坏/非法值经 isWorkspaceTabId 收敛回默认，写失败静默
// （与 layoutPreferences/recentFiles 同款容错）。
const RIGHT_TAB_KEY = "gaea.workspace.rightTab";
const RIGHT_TAB_SESSION_PREFIX = "gaea.rightPanel.v1:";

/** 会话级 key：直接拼接（sessionKey 已由 App 侧清洗非法字符）。 */
function rightTabKey(sessionKey?: string): string {
  return sessionKey ? `${RIGHT_TAB_SESSION_PREFIX}${sessionKey}` : RIGHT_TAB_KEY;
}

/** 读取上次选中的子面板；无记录/非法值回退默认「文件」。
 *  sessionKey 提供时按会话读取（C3），否则读全局 key（旧版兼容）。 */
export function loadPersistedRightTab(sessionKey?: string): WorkspaceTabId {
  try {
    const raw = localStorage.getItem(rightTabKey(sessionKey));
    if (raw && isWorkspaceTabId(raw)) return raw;
  } catch {
    /* 隐私模式/配额等：回退默认 */
  }
  return DEFAULT_WORKSPACE_TAB;
}

/** 记录当前子面板选择（切换时由 App 调用）。sessionKey 语义同上。 */
export function savePersistedRightTab(tab: WorkspaceTabId, sessionKey?: string): void {
  try {
    localStorage.setItem(rightTabKey(sessionKey), tab);
  } catch {
    /* 静默失败，不影响主流程 */
  }
}
