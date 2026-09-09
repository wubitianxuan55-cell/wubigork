// Zustand store — 小型 UI store 合并：useUpdatedFilesStore（已编辑文件标记）、
// useComposerInsertStore（面板 → Composer 一键引用通道）。
// P4 结构刀：由原 lib/store.ts 按 store 域拆分而来，导出名保留。

import { create } from "zustand";

// 已编辑文件标记：docx/xlsx 预览内编辑成功后写入，交付卡片/产物面板据此
// 显示「已更新」徽标（对标"改完即交付"的即时反馈）。
interface UpdatedFilesState {
  updatedAt: Record<string, number>;
  markUpdated: (path: string) => void;
}

export const useUpdatedFilesStore = create<UpdatedFilesState>()((set) => ({
  updatedAt: {},
  markUpdated: (path: string) =>
    set((s) => ({ updatedAt: { ...s.updatedAt, [path]: Date.now() } })),
}));

// 面板 → Composer 的「一键引用」通道：右侧资料概览把 @路径 插入输入框。
interface ComposerInsertState {
  pendingAt: string | null;
  requestAt: (path: string) => void;
  consumeAt: () => string | null;
  pendingText: string | null;
  requestText: (text: string) => void;
  consumeText: () => string | null;
}

export const useComposerInsertStore = create<ComposerInsertState>()((set, get) => ({
  pendingAt: null,
  requestAt: (path: string) => set({ pendingAt: path }),
  consumeAt: () => {
    const p = get().pendingAt;
    set({ pendingAt: null });
    return p;
  },
  pendingText: null,
  requestText: (text: string) => set({ pendingText: text }),
  consumeText: () => {
    const t = get().pendingText;
    set({ pendingText: null });
    return t;
  },
}));
