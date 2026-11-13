// cadOpen.test.ts — 办公板块「打开 CAD」路由（文件树/交付卡/Git 行点 dwg/dxf
// → 右栏 CAD 图纸面板）契约锁：扩展名判定、可选卡自动启用、视图 tab 打开、
// 请求总线投递与 nonce 递增。
import { beforeEach, describe, expect, it } from "vitest";
import { CAD_FILE_EXTS, isCadFilePath, openCadPaneFile, useCadOpenBus } from "./cadOpen";
import { resetPaneTabsForTest, usePaneTabsStore } from "./paneTabs";
import { usePaneEnabledStore } from "./paneEnabled";

describe("cadOpen 文件路由（2026-11-14 CAD 回归右栏一级 Tab）", () => {
  beforeEach(() => {
    resetPaneTabsForTest();
    useCadOpenBus.setState({ request: null });
  });

  it("CAD_FILE_EXTS 契约：右栏面板只承接矢量图纸，pdf 不在内（走文档预览链）", () => {
    expect(CAD_FILE_EXTS).toEqual(["dwg", "dxf"]);
  });

  it("isCadFilePath：dwg/dxf 大小写不敏感命中；pdf/其它不归本面板", () => {
    expect(isCadFilePath("a/车间布置图.DWG")).toBe(true);
    expect(isCadFilePath("b\\电气-自建房_t8.dxf")).toBe(true);
    expect(isCadFilePath("docs/图纸.pdf")).toBe(false);
    expect(isCadFilePath("docs/报告.docx")).toBe(false);
    expect(isCadFilePath("无扩展名")).toBe(false);
  });

  it("openCadPaneFile：停用的可选卡自动启用 + 开 cad 视图 tab + 请求总线投递", () => {
    usePaneEnabledStore.getState().setEnabled("cad", false);
    expect(usePaneEnabledStore.getState().enabled.cad).toBe(false);

    openCadPaneFile("prod/结构平面图.dwg");

    expect(usePaneEnabledStore.getState().enabled.cad).toBe(true);
    const { tabs, active } = usePaneTabsStore.getState();
    const tab = tabs.find((t) => t.kind === "view" && t.viewId === "cad");
    expect(tab).toBeTruthy();
    expect(tab!.title).toBe("CAD 图纸");
    expect(active).toBe(tab!.id);
    const req = useCadOpenBus.getState().request;
    expect(req?.path).toBe("prod/结构平面图.dwg");
    expect(req?.name).toBe("结构平面图.dwg");
    expect(typeof req?.nonce).toBe("number");
  });

  it("已启用时直接开 tab 投递（不写覆盖集）；重复打开 nonce 递增驱动重放", () => {
    usePaneEnabledStore.getState().setEnabled("cad", true);
    const overridesBefore = usePaneEnabledStore.getState().overrides.cad;
    openCadPaneFile("a.dwg");
    const first = useCadOpenBus.getState().request!.nonce;
    openCadPaneFile("b.dwg");
    const second = useCadOpenBus.getState().request!;
    expect(usePaneEnabledStore.getState().overrides.cad).toBe(overridesBefore);
    expect(second.path).toBe("b.dwg");
    expect(second.nonce).toBeGreaterThan(first);
  });

  it("非 CAD 路径不投递不开 tab", () => {
    openCadPaneFile("docs/图纸.pdf");
    expect(useCadOpenBus.getState().request).toBeNull();
    expect(usePaneTabsStore.getState().tabs.some((t) => t.viewId === "cad")).toBe(false);
  });
});
