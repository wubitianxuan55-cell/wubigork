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
});
