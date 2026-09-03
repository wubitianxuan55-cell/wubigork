// v4.26 子代理答复徽标：带 subagentRef 的 assistant 消息加「子代理」小徽标；
// 字段缺省（undefined）行为与现状完全一致（契约：store 稍后为 assistant item
// 追加可选字段 subagentRef?: string，渲染层已提前承接）。
import { describe, expect, it, vi } from "vitest";
import { render, fireEvent } from "@testing-library/react";
import { AssistantMessage } from "./Message";
import { LocaleProvider } from "../lib/i18n";
import type { Item } from "../lib/store";

const wrap = (node: React.ReactNode) => {
  // Message 走 useT；钉住 zh 让徽标断言用中文文案
  localStorage.setItem("gaea-lang", "zh");
  return <LocaleProvider>{node}</LocaleProvider>;
};

type AssistantItem = Extract<Item, { kind: "assistant" }>;

const assistant = (patch: Partial<AssistantItem> & { subagentRef?: string } = {}): AssistantItem =>
  ({ kind: "assistant", id: "a1", text: "子代理干完了", reasoning: "", streaming: false, ...patch });

describe("AssistantMessage subagentRef 徽标", () => {
  it("带 subagentRef：渲染「子代理」徽标，ref 全文进 title", () => {
    const view = render(wrap(<AssistantMessage item={assistant({ subagentRef: "sa_20260901_01" })} />));
    const badge = view.container.querySelector('[data-testid="subagent-badge"]');
    expect(badge).not.toBeNull();
    expect(badge?.textContent).toContain("子代理");
    expect(badge?.getAttribute("title")).toContain("sa_20260901_01");
  });

  it("字段缺省：不渲染徽标，正文照常（现状不变）", () => {
    const view = render(wrap(<AssistantMessage item={assistant()} />));
    expect(view.container.querySelector('[data-testid="subagent-badge"]')).toBeNull();
    expect(view.container.textContent).toContain("子代理干完了");
  });

  it("正文操作：复制按钮把正文写入剪贴板并显示「已复制」", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    const view = render(wrap(<AssistantMessage item={assistant({ text: "可复制的内容" })} />));

    fireEvent.click(view.getByRole("button", { name: /copy|复制/i }));

    expect(writeText).toHaveBeenCalledWith("可复制的内容");
    expect(await view.findByText(/copied|已复制/i)).toBeTruthy();
  });

  it("流式中不渲染复制/沉淀操作（正文未定型）", () => {
    const view = render(wrap(<AssistantMessage item={assistant({ text: "进行中…", streaming: true })} />));
    expect(view.queryByRole("button", { name: /copy/i })).toBeNull();
  });
});
