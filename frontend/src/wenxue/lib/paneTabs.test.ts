// paneTabs — 子代理会话 pane（2026-11-26 文学同步批，镜像办公 06b2b3fd / 73d1a906）：
// 每 ref 单实例去重激活；持久化净化认 viewId=subagent 并保留 meta.subagentRef
// （丢它=页签在而窗渲染不出）。办公版 paneTabs.test.ts 的其余用例依赖 CAD /
// 进度计划等办公页签，不随拷；本文件只钉文学新增的这一面。
import { beforeEach, describe, expect, it } from "vitest";
import { sanitizePaneTabsSnapshot, usePaneTabsStore } from "./paneTabs";

const reset = () => usePaneTabsStore.setState({ tabs: [], active: null, order: [], sessionKey: null, reloadTicks: {} });

describe("usePaneTabsStore.openSubagentPane（文学）", () => {
  beforeEach(reset);

  it("每 ref 单实例：重复打开只激活不追加，标题保留首次", () => {
    const s = usePaneTabsStore.getState();
    s.openSubagentPane("sa_1", "核对第三章时间线");
    s.openSubagentPane("sa_1", "标题变了");
    s.openSubagentPane("sa_2", "梳理角色卡");
    const st = usePaneTabsStore.getState();
    expect(st.tabs.map((t) => t.id)).toEqual(["sub:sa_1", "sub:sa_2"]);
    expect(st.active).toBe("sub:sa_2");
    expect(st.tabs[0]).toEqual({
      id: "sub:sa_1", kind: "view", viewId: "subagent", title: "核对第三章时间线", meta: { subagentRef: "sa_1" },
    });
    // 回到已开的 ref：只激活
    st.openSubagentPane("sa_1", "ignored");
    expect(usePaneTabsStore.getState().active).toBe("sub:sa_1");
    expect(usePaneTabsStore.getState().tabs).toHaveLength(2);
  });

  it("空 ref 不开页签", () => {
    usePaneTabsStore.getState().openSubagentPane("", "空");
    expect(usePaneTabsStore.getState().tabs).toEqual([]);
  });
});

describe("sanitizePaneTabsSnapshot 子代理页签（文学）", () => {
  it("保留 viewId=subagent 与 meta.subagentRef", () => {
    const out = sanitizePaneTabsSnapshot({
      v: 1,
      active: "sub:sa_1",
      order: ["sub:sa_1"],
      tabs: [{ id: "sub:sa_1", kind: "view", viewId: "subagent", title: "核对第三章时间线", meta: { subagentRef: "sa_1" } }],
    });
    expect(out.tabs).toEqual([
      { id: "sub:sa_1", kind: "view", viewId: "subagent", title: "核对第三章时间线", meta: { subagentRef: "sa_1" } },
    ]);
  });

  it("办公页签 id 在文学树仍被净化丢弃（右栏一级视图清单不变）", () => {
    const out = sanitizePaneTabsSnapshot({
      v: 1,
      active: "view:cad",
      order: ["view:cad", "view:gallery"],
      tabs: [
        { id: "view:cad", kind: "view", viewId: "cad", title: "CAD 图纸" },
        { id: "view:gallery", kind: "view", viewId: "gallery", title: "插图" },
      ],
    });
    expect(out.tabs.map((t) => t.id)).toEqual(["view:gallery"]);
  });
});
