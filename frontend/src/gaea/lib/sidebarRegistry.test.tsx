// 注册表测试（v4.23 A0 框架先行）：完整性 + 每项可经 render 挂载。
// 面板组件本体在各自测试文件已有覆盖，这里只验证「经注册表渲染」接线成立。
import { describe, expect, it } from "vitest";
import { Fragment } from "react";
import { render } from "@testing-library/react";
import { SIDEBAR_REGISTRY, getWorkspaceRegistration, type WorkspacePanelContext } from "./sidebarRegistry";
import { WORKSPACE_TABS, WORKSPACE_TAB_IDS } from "./workspaceTabs";

const ctx: WorkspacePanelContext = {
  refreshKey: 0,
  sessionDeliverables: [],
  sessionChanges: [],
  onOpenFile: () => {},
  onClosePanel: () => {},
  onRefreshPanel: () => {},
  onLocateSource: () => {},
};

describe("sidebarRegistry 注册表完整性（蒸馏 dsh-better-sidebar registerTab 形状）", () => {
  it("全部内置面板注册且与清单 id 一一对应（v4.28 起含浏览器观察窗）", () => {
    expect(SIDEBAR_REGISTRY).toHaveLength(WORKSPACE_TAB_IDS.length);
    const ids = SIDEBAR_REGISTRY.map((r) => r.id);
    expect(new Set(ids).size).toBe(ids.length);
    expect([...ids].sort()).toEqual([...WORKSPACE_TAB_IDS].sort());
    // v4.27 已删除的资料/成本库不在注册表
    expect(SIDEBAR_REGISTRY.some((r) => (r.id as string) === "materials" || (r.id as string) === "cost")).toBe(false);
    // v4.28 A2：浏览器观察窗注册项（渲染接线 + 元数据同源）
    const browser = SIDEBAR_REGISTRY.find((r) => r.id === "browser");
    expect(browser).toBeTruthy();
    expect(browser!.label).toBe("浏览器");
    expect(browser!.defaultEnabled).toBe(true);
    expect(typeof browser!.render).toBe("function");
  });

  it("元数据复用清单（单一数据源）：label/icon/keywords/defaultEnabled 同源", () => {
    for (const entry of SIDEBAR_REGISTRY) {
      expect(entry.label.trim().length).toBeGreaterThan(0);
      expect(entry.icon).toBeTypeOf("function");
      expect(entry.keywords.length).toBeGreaterThan(0);
      expect(entry.defaultEnabled).toBe(true); // v4.23 行为基线：面板全默认启用
      const def = WORKSPACE_TABS.find((t) => t.id === entry.id);
      expect(def).toBeTruthy();
    }
  });

  it("顺序与清单声明顺序一致（顺序即展示序）", () => {
    expect(SIDEBAR_REGISTRY.map((r) => r.id)).toEqual(WORKSPACE_TABS.map((t) => t.id));
    expect(SIDEBAR_REGISTRY.map((r) => r.order)).toEqual([...SIDEBAR_REGISTRY.keys()]);
  });

  it("getWorkspaceRegistration 按 id 命中", () => {
    for (const entry of SIDEBAR_REGISTRY) {
      expect(getWorkspaceRegistration(entry.id).id).toBe(entry.id);
    }
  });
});

describe("sidebarRegistry 渲染接线（右栏经注册表渲染，面板组件本体零改动）", () => {
  it("每个注册项的 render 均可挂载产出内容", () => {
    const { container } = render(
      <>{SIDEBAR_REGISTRY.map((entry) => <Fragment key={entry.id}>{entry.render(ctx)}</Fragment>)}</>,
    );
    expect(container.children.length).toBe(SIDEBAR_REGISTRY.length);
  });
});
