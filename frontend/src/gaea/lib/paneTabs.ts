// 右栏 pane tab 状态机（对标 dsh-better-sidebar 的 pane 语义）。
//
// Why: 办公右栏此前是「固定模块 tab（文件/产物/任务/浏览器）+ 面板内再开
// 编辑器 tab」的两级结构。用户拍板对齐 better-sidebar 的工作台语义：
//   - 面板刚打开且没有任何 tab 时 = 欢迎卡片（点卡片才开对应视图）；
//   - 点「文件」卡片 → 开一个视图 tab（资源管理器本身只是其中一个 tab，
//     不是常驻 tab）；
//   - 在资源管理器里点文件 → **再新增一个文件 tab** 去浏览/编辑该文件；
//   - 关闭全部 tab → 回到欢迎卡片态。
// 本文件只维护 pane 级 tab 状态（视图 tab + 文件 tab 混在同一 tab 条），
// 渲染由 WorkspacePane 消费；编辑器文件内容仍复用 FilePreview（换壳不换芯）。
//
// How to apply: openView(id) 开视图（单实例：已开则激活）；openFile(path)
// 开文件 tab（按路径去重：已开则激活）；close/activate 规则与既有
// EditorTabs 一致（关激活 tab 先右邻、末位取左邻；激活触碰最近顺序）。

import { create } from "zustand";
import { WORKSPACE_TABS } from "./workspaceTabs";

/** 一个 pane tab：视图（如资源管理器/产物/任务/浏览器）或文件。 */
export interface PaneTab {
  /** 稳定 id：视图 = `view:<viewId>`，文件 = `file:<path>`。 */
  id: string;
  /** 视图 tab / 文件 tab。 */
  kind: "view" | "file";
  /** kind=view 时的视图 id（files | deliverables | tasks | browser）。 */
  viewId?: string;
  /** kind=file 时的文件相对路径。 */
  path?: string;
  /** tab 条显示标题（视图名 / 文件名）。 */
  title: string;
  /** 视图级附加状态（目前：浏览器视图当前 URL）。 */
  meta?: { url?: string };
}

export interface PaneTabsState {
  /** 打开顺序 = tab 条顺序；空数组 = 欢迎卡片态。 */
  tabs: PaneTab[];
  /** 当前激活 tab id；不在 tabs 内为坏指针（收敛到末位）。 */
  active: string | null;
  /** 最近激活序（关闭时选邻、未来 LRU 用；单调递增）。 */
  order: string[];
  /** 当前会话 key（null = 未绑定，欢迎态不落盘）。 */
  sessionKey: string | null;
  /** 打开视图 tab：同 viewId 已开则激活，否则追加激活。 */
  openView: (viewId: string, title: string) => void;
  /** 打开文件 tab：同路径已开则激活，否则追加激活。 */
  openFile: (path: string, title: string) => void;
  /** 关闭 tab：激活相邻（先右后左）；关唯一 tab 回空（欢迎态）。 */
  close: (id: string) => void;
  /** 激活已开 tab。 */
  activate: (id: string) => void;
  /** 绑定会话（key 变化时读档；null = 欢迎态不落盘）。 */
  setSessionKey: (key: string | null) => void;
  /** 更新某 tab 的视图附加状态（浏览器 URL 等；不存在 no-op）。 */
  updateTabUrl: (id: string, url: string) => void;
}

function touch(prev: string[], id: string): string[] {
  return [...prev.filter((x) => x !== id), id];
}

export const usePaneTabsStore = create<PaneTabsState>()((set, get) => ({
  tabs: [],
  active: null,
  order: [],
  sessionKey: null,

  openView: (viewId, title) => {
    if (!viewId) return;
    const { tabs, order } = get();
    const id = `view:${viewId}`;
    const existing = tabs.find((t) => t.id === id);
    if (existing) {
      set({ active: id, order: touch(order, id) });
      return;
    }
    const tab: PaneTab = { id, kind: "view", viewId, title };
    set({
      tabs: [...tabs, tab],
      active: id,
      order: touch(order, id),
    });
  },

  openFile: (path, title) => {
    if (!path) return;
    const { tabs, order } = get();
    const id = `file:${path}`;
    const existing = tabs.find((t) => t.id === id);
    if (existing) {
      set({ active: id, order: touch(order, id) });
      return;
    }
    const tab: PaneTab = { id, kind: "file", path, title };
    set({
      tabs: [...tabs, tab],
      active: id,
      order: touch(order, id),
    });
  },

  close: (id) => {
    const { tabs, active, order } = get();
    const idx = tabs.findIndex((t) => t.id === id);
    if (idx < 0) return;
    const next = tabs.filter((t) => t.id !== id);
    const nextOrder = order.filter((x) => x !== id);
    let nextActive: string | null = null;
    if (active === id && next.length > 0) {
      // 先右邻；原为末位则取左邻（与 EditorTabs 关闭语义一致）
      nextActive = next[Math.min(idx, next.length - 1)]?.id ?? null;
    } else if (active !== id) {
      nextActive = active;
    }
    set({ tabs: next, active: nextActive, order: nextOrder });
  },

  activate: (id) => {
    const { tabs, active, order } = get();
    if (!tabs.some((t) => t.id === id)) return;
    if (active !== id) {
      set({ active: id, order: touch(order, id) });
    }
  },

  setSessionKey: (key) => {
    const current = get().sessionKey;
    const norm = key ? key.replace(/[\\/:*?"<>|]/g, "_") : null;
    if (norm === current) return;
    if (norm === null) {
      set({ tabs: [], active: null, order: [], sessionKey: null });
      return;
    }
    const saved = loadPaneTabsSnapshot(norm);
    set({
      tabs: saved.tabs,
      active: saved.active,
      order: saved.order,
      sessionKey: norm,
    });
  },

  updateTabUrl: (id, url) => {
    const { tabs } = get();
    const next = tabs.map((t) => (t.id === id ? { ...t, meta: { ...t.meta, url } } : t));
    if (next.some((t, i) => t !== tabs[i])) {
      set({ tabs: next });
    }
  },
}));

// ── 按会话持久化 ────────────────────────────────────────────────
const PANE_STORAGE_PREFIX = "gaea.paneTabs.v1:";
const VIEW_IDS = new Set<string>(WORKSPACE_TABS.map((t) => t.id));

export interface PaneTabsSnapshot {
  v: 1;
  tabs: PaneTab[];
  active: string | null;
  order: string[];
}

/** 净化一条持久化快照：非法视图 id / 坏形状逐条丢弃，不整体报废。 */
export function sanitizePaneTabsSnapshot(value: unknown): PaneTabsSnapshot {
  const empty: PaneTabsSnapshot = { v: 1, tabs: [], active: null, order: [] };
  if (value === null || typeof value !== "object" || Array.isArray(value)) return empty;
  const rec = value as Record<string, unknown>;
  if (!Array.isArray(rec.tabs)) return empty;
  const tabs: PaneTab[] = [];
  const seen = new Set<string>();
  for (const item of rec.tabs) {
    if (item === null || typeof item !== "object") continue;
    const raw = item as Record<string, unknown>;
    if (typeof raw.id !== "string" || typeof raw.title !== "string" || seen.has(raw.id)) continue;
    if (raw.kind === "view" && typeof raw.viewId === "string" && VIEW_IDS.has(raw.viewId)) {
      const meta =
        raw.meta !== null && typeof raw.meta === "object"
          ? { url: typeof (raw.meta as Record<string, unknown>).url === "string" ? (raw.meta as Record<string, string>).url : undefined }
          : undefined;
      tabs.push({ id: raw.id, kind: "view", viewId: raw.viewId, title: raw.title, ...(meta?.url ? { meta } : {}) });
      seen.add(raw.id);
    } else if (raw.kind === "file" && typeof raw.path === "string" && raw.path !== "") {
      tabs.push({ id: raw.id, kind: "file", path: raw.path, title: raw.title });
      seen.add(raw.id);
    }
  }
  const order = Array.isArray(rec.order)
    ? rec.order.filter((x): x is string => typeof x === "string" && seen.has(x))
    : tabs.map((t) => t.id);
  const active = typeof rec.active === "string" && seen.has(rec.active) ? rec.active : tabs[tabs.length - 1]?.id ?? null;
  return { v: 1, tabs, active, order };
}

function loadPaneTabsSnapshot(key: string): PaneTabsSnapshot {
  try {
    const raw = localStorage.getItem(`${PANE_STORAGE_PREFIX}${key}`);
    if (!raw) return { v: 1, tabs: [], active: null, order: [] };
    return sanitizePaneTabsSnapshot(JSON.parse(raw) as unknown);
  } catch {
    return { v: 1, tabs: [], active: null, order: [] };
  }
}

usePaneTabsStore.subscribe((s) => {
  if (!s.sessionKey) return;
  try {
    localStorage.setItem(
      `${PANE_STORAGE_PREFIX}${s.sessionKey}`,
      JSON.stringify({ v: 1, tabs: s.tabs, active: s.active, order: s.order } satisfies PaneTabsSnapshot),
    );
  } catch {
    /* 配额/隐私模式静默 */
  }
});

/** 测试辅助：清空状态（vitest 用例间隔离）。 */
export function resetPaneTabsForTest(): void {
  try {
    const keys: string[] = [];
    for (let i = 0; i < localStorage.length; i++) {
      const k = localStorage.key(i);
      if (k && k.startsWith(PANE_STORAGE_PREFIX)) keys.push(k);
    }
    for (const k of keys) localStorage.removeItem(k);
  } catch {
    /* ignore */
  }
  usePaneTabsStore.setState({ tabs: [], active: null, order: [], sessionKey: null });
}
