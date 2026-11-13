// 工作台面板注册表（v4.23 A0「框架先行」，蒸馏 dsh-better-sidebar registerTab 服务化理念）。
//
// Why: 右侧面板此前「元数据（lib/workspaceTabs 清单）」与「渲染接线（App.tsx
// 渲染分支）」分居两处，面板框架与 tab 内容未解耦——新增面板要同步改清单与
// App 分支，未来扩展 tab（A2 浏览器观察窗 / A3 编辑器 tab）没有平等挂载点。
// 本文件把全部内置面板以统一形状登记（元数据复用清单 + render 接线），App
// 右栏与命令面板一律经注册表派生：内置 tab 与未来扩展能力对等（gaea 无插件
// 系统，先做代码级注册点，为板块 Manifest 化留缝）。
//
// How to apply: 新增面板 → lib/workspaceTabs.ts 对应组追加清单项（含
// defaultEnabled），再在本文件 RENDERERS 补一条 render 接线；按钮条 / 设置
// 卡片 / 命令面板零改动。面板组件本体不感知注册表（学 better-sidebar：框架
// 与 tab 内容解耦）。用 createElement 而非 JSX：注册表是 .ts 数据模块。

import { Suspense, createElement, lazy } from "react";
import type { ReactNode } from "react";
import { DeliverablesPanel, type SessionDeliverable } from "../components/DeliverablesPanel";
import { ChangesPanel } from "../components/ChangesPanel";
import { GitPanel } from "../components/GitPanel";
import { BrowserPanel } from "../components/BrowserPanel";
import { MergedPanel } from "../components/MergedPanel";
import { ExplorerView } from "../components/ExplorerView";
import { usePaneTabsStore } from "./paneTabs";
import { isCadFilePath, openCadPaneFile } from "./cadOpen";
import { GenuiPanel } from "../components/GenuiPanel";

// 2026-11-14 CAD/进度计划回归右栏一级 Tab（退役 ToolSheet 承载）：保持既有
// React.lazy 代码切分——注册表经 lazy 引用面板组件，激活对应视图 tab 才拉 chunk。
const LazyCadPanel = lazy(() => import("../components/cad/CadPanel").then((m) => ({ default: m.CadPanel })));
const LazyScheduleTool = lazy(() => import("../components/schedule/ScheduleTool").then((m) => ({ default: m.ScheduleTool })));

import type { SessionChange } from "./changes";
import type { Icon } from "../icons";
import { WORKSPACE_TABS, type WorkspaceTabId } from "./workspaceTabs";

/** 注册表渲染上下文：App 右栏注入的面板公共依赖（面板 props 与旧渲染分支逐一对应）。 */
export interface WorkspacePanelContext {
  /** 工作区根目录（文件面板 / 变更面板）。 */
  cwd?: string;
  /** 当前预览中的文件（文件面板选中态）。 */
  selectedFile?: string;
  /** 文件树刷新键（编辑回写后 +1）。 */
  refreshKey: number;
  /** 当前会话路径（需按会话取数的面板共享）。 */
  currentSessionPath?: string;
  /** 会话产物清单（产物面板）。 */
  sessionDeliverables: SessionDeliverable[];
  /** v4.30 产物自动置前：本会话新出现（尚未查看）的产物路径（产物 tab 角标
   *  与面板「新」徽标同源；激活产物 tab 后 App 清零）。 */
  freshDeliverablePaths?: string[];
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
  /** 产物行「树中定位」（v4.25 A3 reveal）：亮文件 tab 并高亮该行。 */
  onRevealInTree?: (rel: string) => void;
  /** 树中定位请求体（nonce 变化触发一次：文件树展开父链 + 滚动 + 闪烁）。 */
  revealRequest?: { rel: string; nonce: number } | null;
  /** 打开第一个文件时自动加宽右栏到舒适阅读宽度（v4.27 Codex 式）。 */
  onAutoWidenPanel?: () => void;
  /** 工具面板回话通道（CAD「问图纸/交 AI 算量」、进度计划「问这个计划」）：
   *  关右栏收焦点回对话并发送（App 接 send；v4.524 曾以此接 CadPanel，工具化
   *  期间由 ToolSheet 分支消费，2026-11-14 回归右栏后回到本上下文）。 */
  onAsk?: (text: string) => void;
}

export interface WorkspaceTabRegistration {
  readonly id: WorkspaceTabId;
  readonly label: string;
  readonly icon: Icon;
  /** 面板顺序（一级 Tab 次序，v4.27 扁平化后即展示序）。 */
  readonly order: number;
  /** 声明式设置卡的默认启用态。 */
  readonly defaultEnabled: boolean;
  /** CommandPalette 模糊搜索关键词。 */
  readonly keywords: string[];
  /** 面板渲染器：右栏激活时调用（每次只挂载激活面板，与旧行为一致）。 */
  readonly render: (ctx: WorkspacePanelContext) => ReactNode;
}

// 渲染接线（面板组件本体零改动；v4.53 起产物/变更经 MergedPanel 上下分区
// 直接合并为一个面板；任务与 Agent 编队已迁主区「编队」）。
const RENDERERS: Record<WorkspaceTabId, (ctx: WorkspacePanelContext) => ReactNode> = {
  // 行内「打开文件」统一走 pane 文件 tab（better-sidebar：多个文件 tab 并存，
  // 同路径去重激活）。资源管理器与产物视图共用同一入口。
  files: (ctx) =>
    createElement(ExplorerView, {
      cwd: ctx.cwd,
      refreshKey: ctx.refreshKey,
      revealRequest: ctx.revealRequest,
      openFileTab: openPaneFileTab,
      openMainPreview: ctx.onOpenFile,
    }),
  deliverables: (ctx) =>
    createElement(MergedPanel, {
      primary: createElement(DeliverablesPanel, {
        items: ctx.sessionDeliverables,
        sessionPath: ctx.currentSessionPath,
        onOpenFile: openPaneFileTab,
        onLocateSource: ctx.onLocateSource,
        onRevealInTree: ctx.onRevealInTree,
        freshPaths: ctx.freshDeliverablePaths,
      }),
      secondary: createElement(ChangesPanel, {
        changes: ctx.sessionChanges,
        cwd: ctx.cwd,
        onOpenFile: openPaneFileTab,
      }),
    }),
  // v4.28 A2 浏览器观察窗：数据自取（GaeaBrowserObserve + Trajectory 过滤
  // browser_* 工具记录），不依赖 ctx —— 被动观察面与工作区/会话路径解耦。
  browser: () => createElement(BrowserPanel),
  // 2b Git 面板：数据自取（GaeaGit*，仓库=工作区 cwd）；onOpenFile 复用 pane 文件 tab。
  git: (ctx) => createElement(GitPanel, {
    onOpenFile: openPaneFileTab,
    changes: ctx.sessionChanges,
    cwd: ctx.cwd,
  }),
  // GenUI 会话面板：数据自取（genuiPanel store，按会话隔离）。
  ui: (ctx) => createElement(GenuiPanel, { sessionPath: ctx.currentSessionPath }),
  // CAD 图纸面板（pane 变体即本源形态；打开请求经 lib/cadOpen 总线投递，
  // 文件树/交付卡/Git 行点 dwg/dxf 直达本面板）。
  cad: (ctx) =>
    createElement(
      Suspense,
      { fallback: createElement("div", { className: "p-4 text-[12px] text-fg-dim" }, "正在加载图纸工具…") },
      createElement(LazyCadPanel, { onAsk: ctx.onAsk }),
    ),
  // 进度计划工具面：视图/检查器复用 schedule/ 既有件，与 SchedulePage 共享
  // useScheduleStore（agent schedule_* 写入即时可见）。
  schedule: (ctx) =>
    createElement(
      Suspense,
      { fallback: createElement("div", { className: "p-4 text-[12px] text-fg-dim" }, "正在加载进度工具…") },
      createElement(LazyScheduleTool, { onAsk: ctx.onAsk }),
    ),
};

/** 打开 pane 文件 tab（行内打开的统一入口）：路径去重、追加到工作台 tab 条。
 *  dwg/dxf 例外：统一落 CAD 图纸面板（lib/cadOpen），不开坏预览 tab。 */
function openPaneFileTab(rel: string): void {
  if (!rel) return;
  if (isCadFilePath(rel)) {
    openCadPaneFile(rel);
    return;
  }
  const name = rel.split(/[\\/]/).pop() || rel;
  usePaneTabsStore.getState().openFile(rel, name);
}

/** 单一数据源：内置面板注册表（v4.28 起含浏览器观察窗，顺序即展示序）。
 *  stats 曾迁主区「概览」tab（v4.23 A4），v4.72 随 tab 删除（统计入上下文页）；
 *  materials/cost 标签已删除（v4.27）。 */
export const SIDEBAR_REGISTRY: readonly WorkspaceTabRegistration[] = WORKSPACE_TABS.map(
  (tab, order): WorkspaceTabRegistration => ({
    id: tab.id,
    label: tab.label,
    icon: tab.icon,
    order,
    defaultEnabled: tab.defaultEnabled !== false,
    keywords: tab.keywords,
    render: RENDERERS[tab.id],
  }),
);

const REGISTRY_BY_ID = new Map<WorkspaceTabId, WorkspaceTabRegistration>(
  SIDEBAR_REGISTRY.map((entry) => [entry.id, entry]),
);

/** 按 id 取注册项（清单与注册表同源派生，必然命中；兜底首项防外来 id）。 */
export function getWorkspaceRegistration(id: WorkspaceTabId): WorkspaceTabRegistration {
  return REGISTRY_BY_ID.get(id) ?? SIDEBAR_REGISTRY[0];
}


