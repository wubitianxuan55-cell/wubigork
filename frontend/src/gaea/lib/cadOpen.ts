// cadOpen — 办公板块「打开 CAD 文件」的统一路由（2026-11-14 用户拍板：办公
// 板块可以打开 CAD；CAD 识图回归右栏一级 Tab）。
//
// Why: dwg/dxf 是二进制矢量图纸，pane 文件 tab 的文本/图片预览读出来只会是
// 乱码或报错；用户在文件树/交付卡/Git 变更行点开 CAD 文件时应落右栏
// 「CAD 图纸」面板装载解析，而不是开一个坏预览 tab。
//
// How: isCadFilePath 判扩展名；openCadPaneFile = 启用 cad 可选卡（显式打开
// =用户意图，与右栏＋下拉「点击即启用」同语义）+ 请求总线投递 + 开 pane 视
// 图 tab。CadPanel 订阅请求总线按 nonce 消费（面板未挂载时请求先落总线，挂
// 载后首个 effect 消费；卸载重挂不重放旧请求）。

import { create } from "zustand";
import { WORKSPACE_TABS } from "./workspaceTabs";
import { usePaneEnabledStore } from "./paneEnabled";
import { usePaneTabsStore } from "./paneTabs";

/** 右栏 CAD 面板承接的图纸扩展名（pdf 走既有文档预览链，不在此列）。 */
export const CAD_FILE_EXTS = ["dwg", "dxf"] as const;

function extOf(name: string): string {
  const i = name.lastIndexOf(".");
  return i >= 0 ? name.slice(i + 1).toLowerCase() : "";
}

/** 该路径是否为右栏 CAD 面板承接的图纸文件。 */
export function isCadFilePath(rel: string): boolean {
  const name = rel.split(/[\\/]/).pop() || rel;
  return (CAD_FILE_EXTS as readonly string[]).includes(extOf(name));
}

export interface CadOpenRequest {
  path: string;
  name: string;
  nonce: number;
}

interface CadOpenBusState {
  request: CadOpenRequest | null;
  requestOpen: (path: string) => void;
}

let cadOpenNonce = 0;

/** CAD 打开请求总线：CadPanel 订阅消费（面板未挂载时请求暂存，挂载后消费）。 */
export const useCadOpenBus = create<CadOpenBusState>()((set) => ({
  request: null,
  requestOpen: (path) => {
    const name = path.split(/[\\/]/).pop() || path;
    cadOpenNonce += 1;
    set({ request: { path, name, nonce: cadOpenNonce } });
  },
}));

function cadTabLabel(): string {
  return WORKSPACE_TABS.find((t) => t.id === "cad")?.label ?? "CAD 图纸";
}

/** CAD 文件打开的统一落点：启用可选卡 + 请求总线 + 开 pane 视图 tab。
 *  调用方（openPaneFile / openPaneFileTab）先 isCadFilePath 判定再进本函数；
 *  面板未开时本函数只落 store——pane 视图 tab 在 WorkspacePane 渲染，右栏
 *  开合由调用方处理（工作台语境右栏已开；打开链兜底见 usePreviewPanel）。 */
export function openCadPaneFile(rel: string): void {
  if (!isCadFilePath(rel)) return;
  if (!usePaneEnabledStore.getState().enabled.cad) {
    usePaneEnabledStore.getState().setEnabled("cad", true);
  }
  useCadOpenBus.getState().requestOpen(rel);
  usePaneTabsStore.getState().openView("cad", cadTabLabel());
}
