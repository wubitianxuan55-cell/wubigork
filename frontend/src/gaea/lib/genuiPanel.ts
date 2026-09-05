// 办公 UI 面板 store：模型以 panel:true 围栏投递的常驻会话级工作台。
// 内容随会话隔离（内存）；历史重放按消息顺序重发即可收敛终态。
import { create } from "zustand";
import type { GenuiSpec } from "../../genui";
import { fingerprint } from "../../genui/fingerprint";

export interface GenuiPanelSession {
  content?: GenuiSpec;
  /** 已处理的发布键（内容指纹），防流式/重放/resync 重复发布。 */
  seen: Set<string>;
  appendCount: number;
}

interface GenuiPanelState {
  sessions: Record<string, GenuiPanelSession>;
  publish: (sessionKey: string, sourceKey: string, spec: GenuiSpec) => void;
  clear: (sessionKey: string) => void;
}

/** 会话路径 → 稳定 key（与 App currentSessionKey 同口径）。 */
export function sanitizeSessionKey(path?: string): string {
  return path ? path.replace(/[\\/:*?"<>|]/g, "_") : "unsaved";
}

function mergeAppend(existing: GenuiSpec | undefined, add: GenuiSpec): GenuiSpec {
  if (existing === undefined) return add;
  const existingTabs = existing.items.find((i) => i.type === "tabs");
  const addTabs = add.items.find((i) => i.type === "tabs");
  if (existingTabs?.type === "tabs" && addTabs?.type === "tabs") {
    const merged = [...existingTabs.tabs];
    for (const tab of addTabs.tabs) {
      const hit = merged.find((m) => m.label === tab.label);
      if (hit) hit.items = [...hit.items, ...tab.items];
      else merged.push(tab);
    }
    const items = existing.items.map((i) => (i.type === "tabs" ? { ...i, tabs: merged } : i));
    return { ...existing, items: clipItems(items) };
  }
  return { ...existing, items: clipItems([...existing.items, ...add.items]) };
}

function clipItems(items: GenuiSpec["items"]): GenuiSpec["items"] {
  const budget = 200;
  return items.slice(0, budget);
}

/**
 * 发布去重键：内容指纹（审计 2026-09 #6 的口径选择——忽略 sourceKey）。
 * 会话中途 resync 会更换消息 id（a<seq> → a<日志序>），append 规格若按
 * sourceKey 去重会以新 id 重复发布、面板 tab 重复追加。相同内容的重复发布
 * 对面板永远是无意义操作（REPLACE 内容不变、append 只会叠加重复行），按
 * 内容指纹去重无信息损失；备选「resync 重建时清空 seen」需触碰 store.ts
 * 跨层接线，不采用。
 */
function specFingerprint(spec: GenuiSpec): string {
  return fingerprint(JSON.stringify(spec));
}

export const useGenuiPanelStore = create<GenuiPanelState>()((set) => ({
  sessions: {},
  publish: (sessionKey, _sourceKey, spec) => {
    set((s) => {
      const prev = s.sessions[sessionKey] ?? { seen: new Set(), appendCount: 0 };
      const dedupeKey = specFingerprint(spec);
      if (prev.seen.has(dedupeKey)) return s;
      const seen = new Set(prev.seen);
      seen.add(dedupeKey);
      if (seen.size > 400) {
        const oldest = seen.values().next().value;
        if (oldest !== undefined) seen.delete(oldest);
      }
      const content =
        spec.append === true ? mergeAppend(prev.content, spec) : { ...spec, append: undefined, panel: undefined };
      return {
        sessions: {
          ...s.sessions,
          [sessionKey]: {
            content,
            seen,
            appendCount: spec.append === true ? prev.appendCount + 1 : prev.appendCount,
          },
        },
      };
    });
  },
  clear: (sessionKey) =>
    set((s) => {
      if (!s.sessions[sessionKey]) return s;
      const next = { ...s.sessions };
      delete next[sessionKey];
      return { sessions: next };
    }),
}));

export function clearGenuiPanel(sessionPath?: string): void {
  useGenuiPanelStore.getState().clear(sanitizeSessionKey(sessionPath));
}
