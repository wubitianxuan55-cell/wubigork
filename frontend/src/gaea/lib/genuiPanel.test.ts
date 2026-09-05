import { beforeEach, describe, expect, it } from "vitest";
import { useGenuiPanelStore } from "./genuiPanel";
import type { GenuiSpec } from "../../genui/spec";

const statSpec = (label: string): GenuiSpec => ({
  items: [{ type: "stat", label, value: "1" }],
});

beforeEach(() => {
  useGenuiPanelStore.setState({ sessions: {} });
});

describe("genuiPanel store", () => {
  it("REPLACE 发布与来源去重（同一消息+内容指纹只生效一次）", () => {
    const s = useGenuiPanelStore.getState();
    s.publish("s1", "m1#0", statSpec("A"));
    s.publish("s1", "m1#0", statSpec("A"));
    expect(useGenuiPanelStore.getState().sessions["s1"]?.content?.items[0]).toMatchObject({
      label: "A",
    });
    // 不同消息同内容 → 仍是 replace 生效（B 覆盖 A）
    s.publish("s1", "m2#0", statSpec("B"));
    expect(useGenuiPanelStore.getState().sessions["s1"]?.content?.items[0]).toMatchObject({
      label: "B",
    });
  });

  it("append 合并 items（截断到 200）", () => {
    const s = useGenuiPanelStore.getState();
    s.publish("s1", "m1#0", statSpec("A"));
    s.publish("s1", "m2#0", { append: true, items: [statSpec("B").items[0]] });
    const content = useGenuiPanelStore.getState().sessions["s1"]?.content;
    expect(content?.items).toHaveLength(2);
  });

  it("按会话隔离，clear 只清当前会话", () => {
    const s = useGenuiPanelStore.getState();
    s.publish("s1", "m1#0", statSpec("A"));
    s.publish("s2", "m1#0", statSpec("C"));
    s.clear("s1");
    expect(useGenuiPanelStore.getState().sessions["s1"]).toBeUndefined();
    expect(useGenuiPanelStore.getState().sessions["s2"]?.content?.items[0]).toMatchObject({
      label: "C",
    });
  });

  // 审计 2026-09 #6：会话中途 resync 更换消息 id（a<seq> → a<日志序>），
  // append 规格以新 sourceKey 重发同一内容时不得重复追加。
  it("append 规格 resync 换 sourceKey 后同内容不重复追加", () => {
    const s = useGenuiPanelStore.getState();
    s.publish("s1", "a1#0", { append: true, items: [statSpec("A").items[0]] });
    // resync：同一围栏内容以新消息 id 重新发布
    s.publish("s1", "a99#0", { append: true, items: [statSpec("A").items[0]] });
    const session = useGenuiPanelStore.getState().sessions["s1"];
    expect(session?.appendCount).toBe(1);
    expect(session?.content?.items).toHaveLength(1);
    expect(session?.seen.size).toBe(1);
  });

  // 内容不同的新消息照常追加（内容指纹去重不吞合法更新）。
  it("不同内容的新 append 仍正常叠加", () => {
    const s = useGenuiPanelStore.getState();
    s.publish("s1", "a1#0", { append: true, items: [statSpec("A").items[0]] });
    s.publish("s1", "a2#0", { append: true, items: [statSpec("B").items[0]] });
    const session = useGenuiPanelStore.getState().sessions["s1"];
    expect(session?.appendCount).toBe(2);
    expect(session?.content?.items).toHaveLength(2);
  });
});
