// v4.26.1 回归排查：完工交付卡片（DeliverableCards）在真实事件流下的可见性。
// 用户报告：v4.26 后「看不见完工交付卡片，无法直接点击查看文件」。
import { beforeEach, describe, expect, it } from "vitest";
import { render } from "@testing-library/react";
import { LocaleProvider } from "../lib/i18n";
import { Transcript } from "./Transcript";
import { initialState, useStore } from "../lib/store";
import type { Item } from "../lib/store";

const wrap = (node: React.ReactNode) => <LocaleProvider>{node}</LocaleProvider>;

const text = "方案已完成，交付文件：C:\\AI\\bangong\\黄甲\\开工筹备计划（修订）.docx";
const items: Item[] = [
  { kind: "user", id: "u1", text: "帮我写方案" },
  { kind: "phase", id: "p1", text: "思考中" },
  { kind: "tool", id: "t1", name: "write_file", args: '{"path":"方案报告.docx"}', readOnly: false, status: "done" },
  { kind: "assistant", id: "a1", text, reasoning: "", streaming: false },
];

describe("完工交付卡片回归", () => {
  beforeEach(() => {
    useStore.setState({ ...initialState, _dispatch: useStore.getState()._dispatch });
  });

  it("正常完成轮：卡片渲染", () => {
    useStore.setState({ items, running: false, turnStartAt: Date.now() });
    const view = render(wrap(<Transcript onPrompt={() => {}} running={false} />));
    expect(view.container.textContent).toContain("交付文件");
    expect(view.container.textContent).toContain("开工筹备计划（修订）.docx");
  });

  it("resync 整体替换后（快照含正文）：卡片仍渲染", () => {
    const snap: Item[] = [
      { kind: "user", id: "u1", text: "帮我写方案" },
      { kind: "assistant", id: "a1", text, reasoning: "", streaming: false },
    ];
    useStore.setState({ items: snap, running: false, turnStartAt: Date.now() });
    const view = render(wrap(<Transcript onPrompt={() => {}} running={false} />));
    expect(view.container.textContent).toContain("交付文件");
  });
});
