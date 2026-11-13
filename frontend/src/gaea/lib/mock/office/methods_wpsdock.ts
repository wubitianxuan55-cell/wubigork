// mock/office/methods_wpsdock.ts — P1-4 WPS 面板停靠三绑定的 dev/vitest mock：
// 恒空清单 + 受理文案往返（jsdom 无 win32，真停靠只存在于桌面端）。
import type { OfficeMethods } from "./types";

export const wpsDockMethods: Partial<
  Pick<OfficeMethods, "WpsDockList" | "WpsDockApply" | "WpsDockRestore">
> = {
  WpsDockList: async () => [],
  WpsDockApply: async (_hwnd: number, _x: number, _y: number, _w: number, _h: number, _borderless: boolean) =>
    "已停靠（dev mock）",
  WpsDockRestore: async (_hwnd: number) => "已还原（dev mock）",
};
