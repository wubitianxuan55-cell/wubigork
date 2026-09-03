// 注册表测试（v4.23 A0 框架先行）：完整性 + 每项可经 render 挂载。
// 面板组件本体在各自测试文件已有覆盖，这里只验证「经注册表渲染」接线成立。
import { describe, expect, it, vi } from "vitest";
import { Fragment } from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import { LocaleProvider } from "./i18n";
import { SIDEBAR_REGISTRY, getWorkspaceRegistration, type WorkspacePanelContext } from "./sidebarRegistry";
import { WORKSPACE_TABS, WORKSPACE_TAB_IDS } from "./workspaceTabs";
import { resetPaneTabsForTest, usePaneTabsStore } from "./paneTabs";
import { ToastProvider } from "../components/Toast";

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
    // 面板组件接了 useT（i18n 批），裸渲染会抛——包 Provider 并钉 zh
    localStorage.setItem("gaea-lang", "zh");
    const { container } = render(
      <LocaleProvider><>{SIDEBAR_REGISTRY.map((entry) => <Fragment key={entry.id}>{entry.render(ctx)}</Fragment>)}</></LocaleProvider>,
    );
    expect(container.children.length).toBe(SIDEBAR_REGISTRY.length);
  });
});

describe("sidebarRegistry 产物行打开接线（better-sidebar pane 文件 tab）", () => {
  it("点击产物行 → 新增/激活 pane 文件 tab，不回落大预览回调", async () => {
    localStorage.setItem("gaea-lang", "zh");
    resetPaneTabsForTest();
    const onOpenFile = vi.fn();
    const deliverCtx: WorkspacePanelContext = {
      ...ctx,
      onOpenFile,
      cwd: "/mock",
      sessionDeliverables: [{ path: "docs/竞品调研报告.md", sourceId: "a1", turn: 1, versions: 2 }],
    };
    render(
      <LocaleProvider>
        <ToastProvider>
          {getWorkspaceRegistration("deliverables").render(deliverCtx)}
        </ToastProvider>
      </LocaleProvider>,
    );

    // 会话产物列表与权威登记均渲染同名行（mock 登记表含同路径），任一行为
    // 同一 open 注入 → pane 文件 tab；取第一个即可。
    const row = screen.getAllByRole("button", { name: /竞品调研报告\.md/ })[0];
    fireEvent.click(row);

    const { tabs, active } = usePaneTabsStore.getState();
    expect(tabs.some((t) => t.kind === "file" && t.path === "docs/竞品调研报告.md")).toBe(true);
    expect(active).toBe("file:docs/竞品调研报告.md");
    expect(onOpenFile).not.toHaveBeenCalled();
  });
});
