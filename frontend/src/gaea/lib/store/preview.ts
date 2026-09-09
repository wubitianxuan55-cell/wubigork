// Zustand store — 文件预览队列（usePreviewStore）：全局 UI 状态，独立于 controller。
// P4 结构刀：由原 lib/store.ts 按 store 域拆分而来（preview 域），导出名保留。

import { create } from "zustand";

// 文件预览：全局 UI 状态，独立于 controller（对话内点击文件路径打开）。
// P1-1 多文件预览队列（调研 2026-08-16）：previewFile 保持兼容（= 当前索引
// 对应文件），新增 previewList/index 支持 ←/→ 切换与位置指示。
interface PreviewState {
  previewFile: string | null;
  previewList: string[];
  previewIndex: number;
  openFilePreview: (rel: string) => void;
  closeFilePreview: () => void;
  /** 在多文件队列中前后切换（dir=1 下一个 / dir=-1 上一个）；越界无操作。 */
  navPreview: (dir: 1 | -1) => void;
  /** 跳到队列指定索引（C7：预览队列 chip 点击）；越界无操作。 */
  navTo: (index: number) => void;
  /** 关闭队列中指定索引的文件（C7：预览队列可点条 × 关闭 / 中键关闭）。
   *  移除后当前索引收敛：删当前项跳到相邻项、删唯一项清空队列关闭预览。 */
  closePreviewAt: (index: number) => void;
}

const PREVIEW_MAX_QUEUE = 50;

export const usePreviewStore = create<PreviewState>()((set, get) => ({
  previewFile: null,
  previewList: [],
  previewIndex: -1,
  openFilePreview: (rel: string) => {
    if (!rel) return;
    const { previewList } = get();
    // 已在队列：移动为当前（不重复入列）
    const existIdx = previewList.indexOf(rel);
    if (existIdx >= 0) {
      set({ previewIndex: existIdx, previewFile: rel });
      return;
    }
    // 新文件：追加到队列末尾（上限裁剪），并设为当前
    const next = [...previewList, rel].slice(-PREVIEW_MAX_QUEUE);
    set({ previewList: next, previewIndex: next.length - 1, previewFile: rel });
  },
  closeFilePreview: () => set({ previewFile: null, previewIndex: -1, previewList: [] }),
  navPreview: (dir: 1 | -1) => {
    const { previewList, previewIndex } = get();
    if (previewList.length === 0 || previewIndex < 0) return;
    const next = Math.max(0, Math.min(previewList.length - 1, previewIndex + dir));
    if (next === previewIndex) return;
    set({ previewIndex: next, previewFile: previewList[next] });
  },
  navTo: (index: number) => {
    const { previewList, previewIndex } = get();
    if (index < 0 || index >= previewList.length || index === previewIndex) return;
    set({ previewIndex: index, previewFile: previewList[index] });
  },
  closePreviewAt: (index: number) => {
    const { previewList, previewIndex } = get();
    if (index < 0 || index >= previewList.length) return;
    const nextList = previewList.filter((_, i) => i !== index);
    if (nextList.length === 0) {
      set({ previewFile: null, previewIndex: -1, previewList: [] });
      return;
    }
    // 删除当前项：跳到相邻项（偏向前一个，头项则下一个）；删除其他项：保持当前。
    let nextIndex = previewIndex;
    if (index === previewIndex) {
      nextIndex = Math.min(index, nextList.length - 1);
    } else if (index < previewIndex) {
      nextIndex = previewIndex - 1;
    }
    set({ previewList: nextList, previewIndex: nextIndex, previewFile: nextList[nextIndex] });
  },
}));
