// 右侧面板 Tab 体系声明（办公工作台装配层的轻量 manifest）。
//
// Why: 评审 docs/gaea3-review/03-office-frontend.md 缺陷 4 —— 右侧面板的
// Tab 入口此前散在四处（rightTab 联合类型、App.tsx Tab 按钮条、渲染分支、
// CommandPalette 面板项），新增/调整一个 Tab 要同步改多处且图标映射曾两处
// 不一致（资料/产物在按钮条与命令面板中图标互换）。本文件收敛为单一数据源：
// App.tsx 的 Tab 按钮条、CommandPalette 面板项、激活态判定全部从这里派生。
//
// v3.0.8 曾把子面板按「文件域 / 成果域 / 运行域」归入主 Tab + 第二级小 Tab
// 两级体系；v4.27 用户拍板扁平化：删除「资料」「成本库」两个面板，取消
// 第二级标签，剩余面板全部平铺为一级 Tab（文件 / 产物 / 变更 / 任务 / 分工）；
// v4.28 A2 追加「浏览器」观察窗 tab（截图步进流 + 操作时间线）；
// v4.53 用户拍板化繁为简：产物与变更、任务与分工**直接合并为一个面板**
// （MergedPanel 上下分区同屏全可见，不是二级标签），6→4。
//
// How to apply: 新增面板 → 在本文件 WORKSPACE_TABS 追加一项（id/label/icon/
// keywords/defaultEnabled），再在 lib/sidebarRegistry.ts 的 RENDERERS 补渲染接线；
// 按钮条 / 设置卡片 / 命令面板零改动。label 与现状一致使用中文（未接 i18n，
// 与既有行为对齐）。

import type { Icon } from "../icons";
import { Blocks, ClipboardList, FileText, FolderTree, GitBranch, Globe } from "../icons";
import { readWorkbenchValue, writeWorkbenchValue } from "./workbenchStorage";
import { loadOptionalLayoutSize, saveLayoutSize } from "./layoutPreferences";

/** 子面板（具体功能页）的稳定 id —— App.tsx 的 rightTab 状态直接消费。
 *  v4.23 起不含 stats（曾迁主区「概览」，v4.72 随 tab 删除）；v4.27 起不含 materials/cost
 *  （资料/成本库标签删除）；v4.28 增 browser（浏览器观察窗）；
 *  v4.53 用户拍板化繁为简：产物与变更、任务与分工**直接合并为一个面板**
 *  （MergedPanel 上下分区同屏全可见，不是二级标签），6→4。
 *  旧存储值经 normalizeWorkspaceTabId 别名收敛（changes→deliverables、
 *  subagents→tasks），非法值回默认「文件」。 */
// v4.86 2b：追加 git（Git 面板最小集，D3 单仓库无 push）——v4.53 合并后首个新一级 Tab。
export const WORKSPACE_TAB_IDS = ["files", "deliverables", "tasks", "browser", "git", "ui"] as const;
export type WorkspaceTabId = (typeof WORKSPACE_TAB_IDS)[number];

export interface WorkspaceTabDef {
  id: WorkspaceTabId;
  /** 按钮条与命令面板共用的显示名。 */
  label: string;
  /** 图标组件引用（渲染时由消费方决定尺寸）。 */
  icon: Icon;
  /** CommandPalette 模糊搜索关键词（中英文）。 */
  keywords: string[];
  /** v4.23 声明式设置：默认是否启用（缺省视为 true；用户开关覆盖之）。 */
  defaultEnabled?: boolean;
}

/** 单一数据源：右侧面板全部一级 Tab 的声明（v4.53 合并后 4 Tab）。
 *  产物面板 = 会话产物 + 文件变更上下分区；任务面板 = 任务中心 + 分工。
 *  defaultEnabled 全 true：v4.23 行为基线（面板全可见），用户经设置卡片停用
 *  后覆盖。 */
export const WORKSPACE_TABS: WorkspaceTabDef[] = [
  { id: "files", label: "文件", icon: FolderTree, keywords: ["files", "文件", "工作区", "树"], defaultEnabled: true },
  { id: "deliverables", label: "产物", icon: FileText, keywords: ["deliverables", "产物", "交付", "成果", "变更", "diff", "修改"], defaultEnabled: true },
  { id: "tasks", label: "任务", icon: ClipboardList, keywords: ["tasks", "任务", "进度", "调度", "分工", "子代理", "团队", "并发"], defaultEnabled: true },
  // v4.28 A2 浏览器观察窗：受控 Edge 的截图步进流 + 操作时间线（被动观察，
  // 不拉起浏览器；「新 browser_* 工具自动弹出」由 App 经 browserPrefs 接线）。
  { id: "browser", label: "浏览器", icon: Globe, keywords: ["browser", "浏览器", "观察", "网页"], defaultEnabled: true },
  // 2b Git 面板最小集（分支/状态/暂存/提交/历史；仓库锚定工作区 cwd）。
  { id: "git", label: "Git", icon: GitBranch, keywords: ["git", "版本", "提交", "分支", "暂存", "diff", "commit", "history"], defaultEnabled: true },
  { id: "ui", label: "UI", icon: Blocks, keywords: ["genui", "ui", "面板", "交互", "dashboard", "看板"], defaultEnabled: true },
];

// v4.53 合并后的旧 id 别名：历史持久化值（裸 tab id / 记录 / 启用集）读入时
// 一律先过这张表收敛到合并后的宿主 Tab（学 sanitize 精神：旧值可读、不报废）。
export const LEGACY_TAB_ALIASES: Readonly<Record<string, WorkspaceTabId>> = {
  changes: "deliverables",
  subagents: "tasks",
};

/** 字符串 → WorkspaceTabId：先查旧 id 别名（合并前持久化值），再查现行 id。 */
export function normalizeWorkspaceTabId(value: string): WorkspaceTabId | null {
  const aliased = LEGACY_TAB_ALIASES[value];
  if (aliased) return aliased;
  return isWorkspaceTabId(value) ? value : null;
}

/** 默认激活面板（启动即文件面板）。 */
export const DEFAULT_WORKSPACE_TAB: WorkspaceTabId = "files";

/** 字符串 → WorkspaceTabId 类型守卫（用于外来状态/存储值收敛）。 */
export function isWorkspaceTabId(value: string): value is WorkspaceTabId {
  return (WORKSPACE_TAB_IDS as readonly string[]).includes(value);
}

// ── 会话隔离（蒸馏 dsh-better-sidebar 布局持久化）────────────────────
// 记住用户上次选中的右侧面板 Tab：重开办公板块/重启应用后恢复，而不是
// 每次回到「文件」。v3.0.8 起支持**按会话**持久化（C3 升级）：有 sessionKey
// 时写入 `gaea.work.rightPanel.v1:<sessionKey>`（切会话/新建/恢复各自恢复
// 面板关注点）；无 sessionKey 时回退全局 key `gaea.work.workspace.rightTab`。
// S2.2 空间分键：旧 key（`gaea.rightPanel.v1:` / `gaea.workspace.rightTab`）
// 只读回退迁移。损坏/非法值经 isWorkspaceTabId 收敛回默认，写失败静默。
const RIGHT_TAB_KEY = "gaea.workspace.rightTab";
const RIGHT_TAB_SESSION_PREFIX = "gaea.rightPanel.v1:";

/** 旧 key（读回退用）：会话级直接拼接（sessionKey 已由 App 侧清洗非法字符）。 */
function legacyRightTabKey(sessionKey?: string): string {
  return sessionKey ? `${RIGHT_TAB_SESSION_PREFIX}${sessionKey}` : RIGHT_TAB_KEY;
}

/** 读取上次选中的面板；无记录/非法值回退默认「文件」。
 *  sessionKey 提供时按会话读取（C3），否则读全局 key（旧版兼容）。
 *  v4.23 起底层走 loadPersistedRightPanelState（v2 JSON 记录 / v1 裸 id 双形状）。 */
export function loadPersistedRightTab(sessionKey?: string): WorkspaceTabId {
  return loadPersistedRightPanelState(sessionKey).tab;
}

/** 记录当前面板选择（切换时由 App 调用）。sessionKey 语义同上。 */
export function savePersistedRightTab(tab: WorkspaceTabId, sessionKey?: string): void {
  writeWorkbenchValue(legacyRightTabKey(sessionKey), tab);
}

// ── v4.23 工作台外壳（对标 dsh-better-sidebar）────────────────────────────
// 借了三个交互形状（学其键设计，不抄代码）：
//   1. 全局宽度键：宽度是布局偏好而非会话内容，最后一次拖拽胜出、跨会话即时
//      跟随（对应其 `dsh-sidebar:v1:width`；这里落 lib/layoutPreferences 的
//      workspacePanelWidth 尺寸——全局 blob，非会话分键）。
//   2. 声明式设置的启用集：每 tab 一个布尔开关（对应其 prefs.tabsEnabled 的
//      booleanMapOf——只留合法 id 的布尔值，缺省 = 清单 defaultEnabled）。
//      启用集同为布局偏好 → 全局键，跨会话跟随；会话记录仅随存快照。
//   3. 宽容 sanitize：持久化值损坏/非法时收敛回默认而非崩溃（对应其
//      sanitizeState：旧形状可读、坏字段逐项兜底、失效激活指针修正）。

/** 右栏宽度下限（px）。 */
export const WORKSPACE_MIN_WIDTH = 280;
/** 右栏宽度上限（px）。v4.27 起放宽（原 720）：Codex 式右侧面板可拖到很宽，
 *  实际可用上限由 App 侧按视口与对话区最小宽度动态钳制（见 startWorkspaceResize）。 */
export const WORKSPACE_MAX_WIDTH = 1600;
/** 右栏默认宽度：对齐 styles.css `.layout` 的 --workspace-width: 340px 基线。 */
export const WORKSPACE_DEFAULT_WIDTH = 340;

/** Tab 条自适应图标化的宽度阈值（v4.29 化繁为简）：容器窄于此值时按钮条只显示
 *  图标（label 以 CSS 隐藏，aria-label/title 保留全名），宽栏恢复文字——
 *  6 tab 集合不变，只调呈现密度（对标 Notion 视图 tab 的 Icon only / Text only）。 */
export const WORKSPACE_TAB_COMPACT_WIDTH = 420;

/** 宽度钳制（280–1600 取整；拖拽/读档统一走这里）。 */
export function clampWorkspaceWidth(width: number): number {
  return Math.min(WORKSPACE_MAX_WIDTH, Math.max(WORKSPACE_MIN_WIDTH, Math.round(width)));
}

/** 读全局宽度（最后一次拖拽胜出）。全局无记录时回退会话快照，再回退默认。 */
export function loadWorkspaceWidth(sessionWidth?: number | null): number {
  const global = loadOptionalLayoutSize("workspacePanelWidth", clampWorkspaceWidth);
  if (global !== null) return global;
  if (typeof sessionWidth === "number" && Number.isFinite(sessionWidth) && sessionWidth > 0) {
    return clampWorkspaceWidth(sessionWidth);
  }
  return WORKSPACE_DEFAULT_WIDTH;
}

/** 写全局宽度（拖拽松手时调用；钳制后落 layoutPreferences 全局 blob）。 */
export function saveWorkspaceWidth(width: number): void {
  saveLayoutSize("workspacePanelWidth", width, clampWorkspaceWidth);
}

/** 声明式设置的启用覆盖集：id → 布尔；缺省 id = 清单 defaultEnabled。 */
export type WorkspaceEnabledMap = Partial<Record<WorkspaceTabId, boolean>>;

/** 启用集全局键（无会话后缀——学 better-sidebar 全局宽度键的命名位置）。 */
const RIGHT_ENABLED_KEY = "gaea.rightPanel.v1:tabsEnabled";

/** 启用集净化：只保留合法 id 的布尔值（旧 id 先过别名表；坏值逐项丢弃不崩）。 */
export function sanitizeEnabledTabs(value: unknown): WorkspaceEnabledMap {
  if (value === null || typeof value !== "object" || Array.isArray(value)) return {};
  const out: WorkspaceEnabledMap = {};
  for (const [id, flag] of Object.entries(value as Record<string, unknown>)) {
    if (typeof flag === "boolean") {
      const tabId = normalizeWorkspaceTabId(id);
      if (tabId) out[tabId] = flag;
    }
  }
  return out;
}

/** 读启用覆盖集（全局键；损坏/非法值收敛回空覆盖 = 全部按默认启用）。 */
export function loadEnabledTabs(): WorkspaceEnabledMap {
  try {
    const raw = readWorkbenchValue(RIGHT_ENABLED_KEY);
    if (raw) return sanitizeEnabledTabs(JSON.parse(raw) as unknown);
  } catch {
    /* 隐私模式/配额/坏 JSON：回退空覆盖 */
  }
  return {};
}

/** 写启用覆盖集（设置开关切换时调用）。 */
export function saveEnabledTabs(map: WorkspaceEnabledMap): void {
  writeWorkbenchValue(RIGHT_ENABLED_KEY, JSON.stringify(map));
}

/** 覆盖集 + 清单默认 → 全量启用表（渲染与派生的权威入参）。
 *  兜底：全部被禁（手改存储/坏数据）时回默认集，右栏永远至少一个可用面板。 */
export function resolveEnabledTabs(overrides: WorkspaceEnabledMap): Record<WorkspaceTabId, boolean> {
  const out = {} as Record<WorkspaceTabId, boolean>;
  for (const tab of WORKSPACE_TABS) out[tab.id] = overrides[tab.id] ?? tab.defaultEnabled !== false;
  if (!WORKSPACE_TABS.some((tab) => out[tab.id])) {
    // 兜底按清单默认（而非覆盖集——覆盖集正是全禁的来源）
    for (const tab of WORKSPACE_TABS) out[tab.id] = tab.defaultEnabled !== false;
  }
  return out;
}

/** 第一个启用的面板（激活 tab 被停用时的收敛目标，学 sanitizeState 失效指针修正）。 */
export function firstEnabledTab(enabled: Record<WorkspaceTabId, boolean>): WorkspaceTabId {
  for (const tab of WORKSPACE_TABS) {
    if (enabled[tab.id]) return tab.id;
  }
  return DEFAULT_WORKSPACE_TAB;
}

// ── 会话记录 v2：激活 tab + 启用集快照 + 宽度快照（向后兼容 v1 裸 id）──────
// 旧值 = 裸 tab id 字符串（v4.22 及之前），必须继续可读；新值 = JSON 记录。
// 宽度/启用集的**权威**在全局键（布局偏好跨会话跟随），会话记录只随存快照：
// 全局键缺省时（首次运行/换机器）用会话快照兜底，语义对齐 better-sidebar 的
// 「session state 自带 width，全局宽度键在读档时胜出」。
export interface WorkspacePanelPersistedState {
  /** 记录形状版本。 */
  v: 1;
  /** 激活面板。 */
  tab: WorkspaceTabId;
  /** 启用集快照（null = 未记录；权威值在 RIGHT_ENABLED_KEY 全局键）。 */
  enabled: WorkspaceEnabledMap | null;
  /** 宽度快照（null = 未记录；权威值在 layoutPreferences 全局键）。 */
  width: number | null;
}

const EMPTY_PERSISTED_STATE: WorkspacePanelPersistedState = {
  v: 1,
  tab: DEFAULT_WORKSPACE_TAB,
  enabled: null,
  width: null,
};

/** 逐字段净化一条持久化记录（学 sanitizeState：坏字段回默认，不整体报废）。 */
function sanitizePersistedState(parsed: unknown): WorkspacePanelPersistedState {
  if (parsed === null || typeof parsed !== "object") {
    // 旧形状：裸 tab id 字符串（changes/subagents 别名收敛到合并宿主）
    const bare = typeof parsed === "string" ? normalizeWorkspaceTabId(parsed) : null;
    return bare ? { ...EMPTY_PERSISTED_STATE, tab: bare } : { ...EMPTY_PERSISTED_STATE };
  }
  const rec = parsed as Record<string, unknown>;
  return {
    v: 1,
    // 旧裸 id / 旧记录的 changes/subagents 经别名表收敛到合并宿主 Tab
    tab: (typeof rec.tab === "string" ? normalizeWorkspaceTabId(rec.tab) : null) ?? DEFAULT_WORKSPACE_TAB,
    enabled: rec.enabled === null || rec.enabled === undefined ? null : sanitizeEnabledTabs(rec.enabled),
    width:
      typeof rec.width === "number" && Number.isFinite(rec.width) && rec.width > 0
        ? clampWorkspaceWidth(rec.width)
        : null,
  };
}

/** 读取该会话的右栏布局记录（v2 JSON 记录 / v1 裸 id 旧值 / 损坏值 → 兜底默认）。 */
export function loadPersistedRightPanelState(sessionKey?: string): WorkspacePanelPersistedState {
  try {
    const raw = readWorkbenchValue(legacyRightTabKey(sessionKey));
    if (raw === null) return { ...EMPTY_PERSISTED_STATE };
    try {
      return sanitizePersistedState(JSON.parse(raw) as unknown);
    } catch {
      // 非 JSON：按旧形状（裸 tab id 字符串）收敛
      return sanitizePersistedState(raw);
    }
  } catch {
    return { ...EMPTY_PERSISTED_STATE };
  }
}

/** 写入该会话的右栏布局记录（激活 tab / 启用集 / 宽度变化时由 App 调用）。 */
export function savePersistedRightPanelState(state: WorkspacePanelPersistedState, sessionKey?: string): void {
  writeWorkbenchValue(legacyRightTabKey(sessionKey), JSON.stringify(state));
}
