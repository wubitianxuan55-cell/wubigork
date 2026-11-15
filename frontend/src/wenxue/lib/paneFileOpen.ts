// paneFileOpen — 正文/卡片「点文件」的统一打开入口（T-…，对标既有
// setTaskCardActivityProvider 的模块级注入模式）。
//
// 语义：工作台语境（App 挂载时注册 openPaneFile）→ 开右栏 pane 文件 tab；
// 其它页面/单测未注册时回退 usePreviewStore（弹窗/内嵌预览），行为与旧版一致。

import { usePreviewStore } from "./store";

type PaneFileOpenHandler = (rel: string) => void;

let paneFileOpenHandler: PaneFileOpenHandler | null = null;

/** App 注册工作台打开器（开 pane 文件 tab）；卸载时置空回落预览。 */
export function setPaneFileOpenHandler(fn: PaneFileOpenHandler | null): void {
  paneFileOpenHandler = fn;
}

/** 打开一个文件引用：优先 pane 文件 tab，未注册则退回大预览/弹窗通道。 */
export function openPaneFileOrPreview(rel: string): void {
  if (!rel) return;
  if (paneFileOpenHandler) {
    paneFileOpenHandler(rel);
    return;
  }
  usePreviewStore.getState().openFilePreview(rel);
}
