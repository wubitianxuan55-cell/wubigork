// 工作台面板注册表（v4.23 A0「框架先行」，蒸馏 dsh-better-sidebar registerTab 服务化理念）。
//
// Why: 右侧面板此前「元数据（lib/workspaceTabs 清单）」与「渲染接线（App.tsx
// 渲染分支）」分居两处，面板框架与 tab 内容未解耦——新增面板要同步改清单与
// App 分支，未来扩展 tab（A2 浏览器观察窗 / A3 编辑器 tab）没有平等挂载点。
// 本文件把 7 个内置面板以统一形状登记（元数据复用清单 + render 接线），App
// 右栏与命令面板一律经注册表派生：内置 tab 与未来扩展能力对等（gaea 无插件
// 系统，先做代码级注册点，为板块 Manifest 化留缝）。
//
// How to apply: 新增面板 → lib/workspaceTabs.ts 对应组追加清单项（含
// defaultEnabled），再在本文件 RENDERERS 补一条 render 接线；按钮条 / 设置
// 卡片 / 命令面板零改动。面板组件本体不感知注册表（学 better-sidebar：框架
// 与 tab 内容解耦）。用 createElement 而非 JSX：注册表是 .ts 数据模块。

import { createElement } from "react";
import type { ReactNode } from "react";
import { WorkspacePanel } from "../components/WorkspacePanel";
import { MaterialsPanel } from "../components/MaterialsPanel";
import { CostLibraryPanel } from "../components/CostLibraryPanel";
import { DeliverablesPanel, type SessionDeliverable } from "../components/DeliverablesPanel";
import { ChangesPanel } from "../components/ChangesPanel";
import { TaskCenter } from "../components/TaskCenter";
import { SubagentsPanel } from "../components/SubagentsPanel";
import type { SessionChange } from "./changes";
import type { Icon } from "../icons";
import { WORKSPACE_GROUPS, type WorkspaceGroupId, type WorkspaceTabId } from "./workspaceTabs";

/** 注册表渲染上下文：App 右栏注入的面板公共依赖（面板 props 与旧渲染分支逐一对应）。 */
export interface WorkspacePanelContext {
  /** 工作区根目录（文件面板 / 变更面板）。 */
  cwd?: string;
  /** 当前预览中的文件（文件面板选中态）。 */
  selectedFile?: string;
  /** 文件树刷新键（编辑回写后 +1）。 */
  refreshKey: number;
  /** 当前会话路径（分工面板按会话拉取子代理）。 */
  currentSessionPath?: string;
  /** 会话产物清单（产物面板）。 */
  sessionDeliverables: SessionDeliverable[];
  /** 会话文件变更清单（变更面板）。 */
  sessionChanges: SessionChange[];
  /** 打开文件 → 主区预览（Codex 式）。 */
  onOpenFile: (rel: string) => void;
  /** 收起右栏（文件面板头部 X / 产物定位回正文）。 */
  onClosePanel: () => void;
  /** 刷新文件树。 */
  onRefreshPanel: () => void;
  /** 产物定位来源轮次（收起右栏并滚动到对应消息）。 */
  onLocateSource: (turn: number) => void;
}

export interface WorkspaceTabRegistration {
  readonly id: WorkspaceTabId;
  readonly label: string;
  readonly icon: Icon;
  /** 所属主 Tab（分组）。 */
  readonly group: WorkspaceGroupId;
  /** 组顺序（第一级 Tab 次序）。 */
  readonly groupOrder: number;
  /** 组内顺序（第二级小 Tab 次序）。 */
  readonly order: number;
  /** 声明式设置卡的默认启用态。 */
  readonly defaultEnabled: boolean;
  /** CommandPalette 模糊搜索关键词。 */
  readonly keywords: string[];
  /** 面板渲染器：右栏激活时调用（每次只挂载激活面板，与旧行为一致）。 */
  readonly render: (ctx: WorkspacePanelContext) => ReactNode;
}

// 渲染接线（与 App.tsx 旧渲染分支逐一对应，行为不变；面板组件本体零改动）。
const RENDERERS: Record<WorkspaceTabId, (ctx: WorkspacePanelContext) => ReactNode> = {
  files: (ctx) =>
    createElement(WorkspacePanel, {
      cwd: ctx.cwd,
      selectedFile: ctx.selectedFile,
      refreshKey: ctx.refreshKey,
      onSelectFile: ctx.onOpenFile,
      onRefresh: ctx.onRefreshPanel,
      onClose: ctx.onClosePanel,
    }),
  materials: (ctx) => createElement(MaterialsPanel, { onOpenFile: ctx.onOpenFile }),
  cost: () => createElement(CostLibraryPanel),
  deliverables: (ctx) =>
    createElement(DeliverablesPanel, {
      items: ctx.sessionDeliverables,
      onOpenFile: ctx.onOpenFile,
      onLocateSource: ctx.onLocateSource,
    }),
  changes: (ctx) =>
    createElement(ChangesPanel, {
      changes: ctx.sessionChanges,
      cwd: ctx.cwd,
      onOpenFile: ctx.onOpenFile,
    }),
  tasks: () => createElement(TaskCenter),
  subagents: (ctx) => createElement(SubagentsPanel, { sessionPath: ctx.currentSessionPath }),
};

/** 单一数据源：7 个内置面板的注册表（按组序 → 组内序扁平，顺序即展示序）。
 *  stats 已迁主区「概览」tab（v4.23 A4），不再注册。 */
export const SIDEBAR_REGISTRY: readonly WorkspaceTabRegistration[] = WORKSPACE_GROUPS.flatMap((g, groupOrder) =>
  g.tabs.map((tab, order): WorkspaceTabRegistration => ({
    id: tab.id,
    label: tab.label,
    icon: tab.icon,
    group: g.id,
    groupOrder,
    order,
    defaultEnabled: tab.defaultEnabled !== false,
    keywords: tab.keywords,
    render: RENDERERS[tab.id],
  })),
);

const REGISTRY_BY_ID = new Map<WorkspaceTabId, WorkspaceTabRegistration>(
  SIDEBAR_REGISTRY.map((entry) => [entry.id, entry]),
);

/** 按 id 取注册项（清单与注册表同源派生，必然命中；兜底首项防外来 id）。 */
export function getWorkspaceRegistration(id: WorkspaceTabId): WorkspaceTabRegistration {
  return REGISTRY_BY_ID.get(id) ?? SIDEBAR_REGISTRY[0];
}
