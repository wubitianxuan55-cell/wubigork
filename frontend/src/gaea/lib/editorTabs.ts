// 右栏编辑器 tab 状态机（v4.25 A3「文件工作台」）。
//
// Why: A3 规划要求文件树点开 → 右栏内多文件编辑器 tab（双入口保留：主区
// 预览 pane 不删）。tab 状态不能内联在 WorkspacePanel 的 useState 里——
// 主代理要在 App 侧实现「模型主动打开」（sidebar_open 工具事件 → 程序化
// 在右栏开文件 tab），必须能从组件外部命令式驱动。故仿 lib/store.ts 的
// zustand 模块级 store 先例：hook（EditorTabs/WorkspacePanel 消费）+
// getState() 命令式 API（App 事件侧调用）双出口。
//
// How to apply: 组件内 `useEditorTabsStore((s) => s.tabs)` 订阅；命令式
// `openEditorTab(rel) / closeEditorTab(rel) / activateEditorTab(rel)`（内部
// 即 getState() 转发）。状态机规则：
//   - open：已开则激活；新开追加并激活；超上限 12 时 LRU 驱逐「最久未
//     激活」的 tab（平局取打开顺序更早者）；
//   - close：关闭激活 tab 时激活相邻（先右邻、末位取左邻）；关非激活不动；
//   - activate：仅已开的路径可激活，激活即触碰 LRU。
// 持久化 key `gaea.rightPanel.editorTabs.v1`：坏值/半坏值逐字段兜底回空，
// 写失败（隐私模式/配额）静默——学 workspaceTabs.ts 的宽容 sanitize 纪律。

import { create } from "zustand";

/** tab 数量上限：超限时 LRU 驱逐最久未激活的 tab。 */
export const EDITOR_TABS_MAX = 12;

/** localStorage 持久化 key（坏值兜底回空）。 */
export const EDITOR_TABS_STORAGE_KEY = "gaea.rightPanel.editorTabs.v1";

/** 持久化快照形状（v1）。 */
export interface EditorTabsSnapshot {
  v: 1;
  /** 打开的文件相对路径（顺序 = 打开顺序，视觉稳定；LRU 用 lastActiveAt 另记）。 */
  tabs: string[];
  /** 当前激活 tab（不在 tabs 内视为坏指针，收敛回首项）。 */
  active: string | null;
  /** 各 tab 最近激活时间戳（LRU 驱逐依据；重启后恢复，避免重载后误驱逐）。 */
  lastActiveAt: Record<string, number>;
}

export interface EditorTabsState extends EditorTabsSnapshot {
  /** 打开文件：已开则激活；新开追加并激活；超上限 LRU 驱逐。空路径忽略。 */
  open: (path: string) => void;
  /** 关闭文件 tab：未开 no-op；关闭激活 tab 时激活相邻（先右后左）。 */
  close: (path: string) => void;
  /** 激活已开的 tab（触碰 LRU）；未开 no-op。 */
  activate: (path: string) => void;
}

// LRU 时钟：模块级单调递增计数器。不用 Date.now()——快速连续操作时间戳
// 会撞车、LRU 平局无法判定；计数器天然全序且测试可复现。初值取持久化
// lastActiveAt 的最大值，保证重载后新计数不会小于旧值。
let lruClock = 0;

const EMPTY_SNAPSHOT: EditorTabsSnapshot = { v: 1, tabs: [], active: null, lastActiveAt: {} };

/** 快照净化：坏值/半坏值逐字段兜底（非字符串路径丢弃、去重、封顶、
 *  失效激活指针修正到首项、LRU 表只留已开路径的有限数字）。 */
export function sanitizeEditorTabsSnapshot(value: unknown): EditorTabsSnapshot {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    return { ...EMPTY_SNAPSHOT, tabs: [], lastActiveAt: {} };
  }
  const rec = value as Record<string, unknown>;
  const tabs: string[] = [];
  const seen = new Set<string>();
  if (Array.isArray(rec.tabs)) {
    for (const item of rec.tabs) {
      if (typeof item === "string" && item !== "" && !seen.has(item)) {
        seen.add(item);
        tabs.push(item);
        if (tabs.length >= EDITOR_TABS_MAX) break;
      }
    }
  }
  const lastActiveAt: Record<string, number> = {};
  if (rec.lastActiveAt !== null && typeof rec.lastActiveAt === "object" && !Array.isArray(rec.lastActiveAt)) {
    for (const [k, v] of Object.entries(rec.lastActiveAt as Record<string, unknown>)) {
      if (seen.has(k) && typeof v === "number" && Number.isFinite(v)) lastActiveAt[k] = v;
    }
  }
  const active =
    typeof rec.active === "string" && seen.has(rec.active)
      ? rec.active
      : tabs[0] ?? null;
  return { v: 1, tabs, active, lastActiveAt };
}

/** 读持久化快照：坏 JSON/异常回空（调用方拿到的永远是合法形状）。 */
export function loadPersistedEditorTabs(): EditorTabsSnapshot {
  try {
    const raw = localStorage.getItem(EDITOR_TABS_STORAGE_KEY);
    if (!raw) return { ...EMPTY_SNAPSHOT, tabs: [], lastActiveAt: {} };
    return sanitizeEditorTabsSnapshot(JSON.parse(raw) as unknown);
  } catch {
    return { ...EMPTY_SNAPSHOT, tabs: [], lastActiveAt: {} };
  }
}

/** 写持久化快照：配额/隐私模式静默失败，不影响主流程。 */
function persistEditorTabs(s: EditorTabsSnapshot): void {
  try {
    localStorage.setItem(
      EDITOR_TABS_STORAGE_KEY,
      JSON.stringify({ v: 1, tabs: s.tabs, active: s.active, lastActiveAt: s.lastActiveAt } satisfies EditorTabsSnapshot),
    );
  } catch {
    // 静默失败，不影响主流程
  }
}

// 初始状态：模块加载时读档一次；LRU 时钟对齐历史最大时间戳。
const initial = loadPersistedEditorTabs();
lruClock = Math.max(0, ...Object.values(initial.lastActiveAt));

export const useEditorTabsStore = create<EditorTabsState>()((set, get) => ({
  ...initial,
  open: (path: string) => {
    if (!path) return;
    const { tabs, lastActiveAt } = get();
    const ts = ++lruClock;
    // 已开 → 激活 + LRU 触碰（不改变 tab 视觉顺序）
    if (tabs.includes(path)) {
      set({ active: path, lastActiveAt: { ...lastActiveAt, [path]: ts } });
      return;
    }
    // 新开 → 追加 + 激活；超上限驱逐最久未激活（lastActiveAt 平局取打开序更早者）
    let nextTabs = [...tabs, path];
    const nextLru = { ...lastActiveAt, [path]: ts };
    if (nextTabs.length > EDITOR_TABS_MAX) {
      let victim = nextTabs[0];
      for (const p of nextTabs) {
        if ((nextLru[p] ?? 0) < (nextLru[victim] ?? 0)) victim = p;
      }
      nextTabs = nextTabs.filter((p) => p !== victim);
      delete nextLru[victim];
    }
    set({ tabs: nextTabs, active: path, lastActiveAt: nextLru });
  },
  close: (path: string) => {
    const { tabs, active, lastActiveAt } = get();
    const idx = tabs.indexOf(path);
    if (idx < 0) return;
    const nextTabs = tabs.filter((p) => p !== path);
    const nextLru = { ...lastActiveAt };
    delete nextLru[path];
    // 关闭激活 tab → 激活相邻（先右邻；原为末位则取左邻）；关非激活 → 保持当前
    const nextActive =
      active === path ? nextTabs[Math.min(idx, nextTabs.length - 1)] ?? null : active;
    set({ tabs: nextTabs, active: nextActive, lastActiveAt: nextLru });
  },
  activate: (path: string) => {
    const { tabs, lastActiveAt } = get();
    if (!tabs.includes(path)) return;
    set({ active: path, lastActiveAt: { ...lastActiveAt, [path]: ++lruClock } });
  },
}));

// 状态任何变化 → 持久化（含测试 reset 写回空快照，无害）。
useEditorTabsStore.subscribe((s) => persistEditorTabs(s));

// ── 命令式 API（App 事件侧消费：sidebar_open 工具事件 → 程序化开 tab）──

/** 命令式打开编辑器 tab（App「模型主动打开」入口；内部 getState().open）。 */
export function openEditorTab(path: string): void {
  useEditorTabsStore.getState().open(path);
}

/** 命令式关闭编辑器 tab。 */
export function closeEditorTab(path: string): void {
  useEditorTabsStore.getState().close(path);
}

/** 命令式激活编辑器 tab。 */
export function activateEditorTab(path: string): void {
  useEditorTabsStore.getState().activate(path);
}

/** 测试辅助：清空存储 + 复位状态机（vitest 用例间隔离）。 */
export function resetEditorTabsForTest(): void {
  try {
    localStorage.removeItem(EDITOR_TABS_STORAGE_KEY);
  } catch {
    /* ignore */
  }
  lruClock = 0;
  useEditorTabsStore.setState({ v: 1, tabs: [], active: null, lastActiveAt: {} });
}
