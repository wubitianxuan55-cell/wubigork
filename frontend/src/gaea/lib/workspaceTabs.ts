// 右侧面板 Tab 体系声明（办公工作台装配层的轻量 manifest）。
//
// Why: 评审 docs/gaea3-review/03-office-frontend.md 缺陷 4 —— 右侧面板的
// Tab 入口此前散在四处（rightTab 联合类型、App.tsx Tab 按钮条、渲染分支、
// CommandPalette 面板项），新增/调整一个 Tab 要同步改多处且图标映射曾两处
// 不一致（资料/产物在按钮条与命令面板中图标互换）。本文件收敛为单一数据源：
// App.tsx 的 Tab 按钮条、CommandPalette 面板项、激活态判定全部从这里派生。
//
// How to apply: 新增右侧 Tab 只需在此追加一项（id/label/icon/keywords），
// 再在 App.tsx 的 workspacePanelRender 分支补对应面板渲染；按钮条与命令
// 面板零改动。label 与现状一致使用中文（未接 i18n，与既有行为对齐）。

import type { Icon } from "../icons";
import { BarChart3, ClipboardList, Diff, FileText, FolderTree, Paperclip } from "../icons";

/** 右侧面板 Tab 的稳定 id —— 由清单推导，App.tsx 的 rightTab 状态直接消费。 */
export const WORKSPACE_TAB_IDS = ["files", "materials", "deliverables", "changes", "stats", "tasks"] as const;
export type WorkspaceTabId = (typeof WORKSPACE_TAB_IDS)[number];

export interface WorkspaceTabDef {
  id: WorkspaceTabId;
  /** 按钮条与命令面板共用的显示名。 */
  label: string;
  /** 图标组件引用（渲染时由消费方决定尺寸）。 */
  icon: Icon;
  /** CommandPalette 模糊搜索关键词（中英文）。 */
  keywords: string[];
}

/** 单一数据源：右侧面板全部 Tab 的声明。 */
export const WORKSPACE_TABS: WorkspaceTabDef[] = [
  { id: "files", label: "文件", icon: FolderTree, keywords: ["files", "文件", "工作区", "树"] },
  { id: "materials", label: "资料", icon: Paperclip, keywords: ["materials", "资料", "素材", "钉住"] },
  { id: "deliverables", label: "产物", icon: FileText, keywords: ["deliverables", "产物", "交付", "成果"] },
  { id: "changes", label: "变更", icon: Diff, keywords: ["changes", "变更", "修改", "diff"] },
  { id: "stats", label: "统计", icon: BarChart3, keywords: ["stats", "统计", "token", "成本", "用量"] },
  { id: "tasks", label: "任务", icon: ClipboardList, keywords: ["tasks", "任务", "进度", "调度"] },
];

/** 默认激活 Tab（启动即文件面板）。 */
export const DEFAULT_WORKSPACE_TAB: WorkspaceTabId = "files";

/** 字符串 → WorkspaceTabId 类型守卫（用于外来状态/存储值收敛）。 */
export function isWorkspaceTabId(value: string): value is WorkspaceTabId {
  return (WORKSPACE_TAB_IDS as readonly string[]).includes(value);
}
