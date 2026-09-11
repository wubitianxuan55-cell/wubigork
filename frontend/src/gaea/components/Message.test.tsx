// v4.26 子代理答复徽标：带 subagentRef 的 assistant 消息加「子代理」小徽标；
// 字段缺省（undefined）行为与现状完全一致（契约：store 稍后为 assistant item
// 追加可选字段 subagentRef?: string，渲染层已提前承接）。
import { afterEach, describe, expect, it, vi } from "vitest";
import { ToastProvider } from "./Toast";
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

describe("AssistantMessage 重新生成按钮（v4.232）", () => {
  it("canRegenerate：渲染按钮，点击按轮号回调", () => {
    const onRegenerateTurn = vi.fn();
    const view = render(wrap(
      <AssistantMessage item={assistant({ text: "可重发的回答" })} turnNo={2} canRegenerate onRegenerateTurn={onRegenerateTurn} />,
    ));
    const btn = view.getByRole("button", { name: /regenerate|重新生成/i });
    fireEvent.click(btn);
    expect(onRegenerateTurn).toHaveBeenCalledWith(2);
  });

  it("无资格（canRegenerate 缺省）：不渲染按钮", () => {
    const view = render(wrap(
      <AssistantMessage item={assistant({ text: "历史回答" })} turnNo={0} onRegenerateTurn={vi.fn()} />,
    ));
    expect(view.queryByRole("button", { name: /regenerate|重新生成/i })).toBeNull();
  });

  it("流式中不渲染（与复制/沉淀同闸）", () => {
    const view = render(wrap(
      <AssistantMessage item={assistant({ text: "进行中…", streaming: true })} turnNo={0} canRegenerate onRegenerateTurn={vi.fn()} />,
    ));
    expect(view.queryByRole("button", { name: /regenerate|重新生成/i })).toBeNull();
  });
});

describe("AssistantMessage 回答反馈（v4.238 能力层）", () => {
  type G = { go?: { app?: Record<string, Record<string, unknown>> } };
  function injectFeedback(ok: boolean) {
    const spy = vi.fn(
      ok
        ? async () => undefined
        : async () => { throw new Error("库不可用"); },
    );
    (window as unknown as { go?: unknown }).go = {
      app: {
        OfficeB: { GaeaMemoryFeedback: spy, GaeaLogFrontendError: vi.fn(async () => {}) },
      },
    };
    return spy;
  }
  afterEach(() => {
    delete (window as unknown as { go?: unknown }).go;
  });

  it("点 👍：按（消息id, up, 正文, work）落库，按钮进已反馈态", async () => {
    const spy = injectFeedback(true);
    const view = render(wrap(
      <AssistantMessage item={assistant({ id: "a9", text: "值得点赞的回答" })} onFeedback />,
    ));
    fireEvent.click(view.getByRole("button", { name: "有用" }));
    await vi.waitFor(() => expect(spy).toHaveBeenCalledTimes(1));
    expect(spy).toHaveBeenCalledWith("a9", "up", "值得点赞的回答", "work");
    // 已反馈：双钮禁用（事件追加式不可撤回）
    expect((view.getByRole("button", { name: "有用" }) as HTMLButtonElement).disabled).toBe(true);
    expect((view.getByRole("button", { name: "需改进" }) as HTMLButtonElement).disabled).toBe(true);
  });

  it("落库失败：回退可重试 + toast 可见", async () => {
    const spy = injectFeedback(false);
    const view = render(wrap(
      <ToastProvider>
        <AssistantMessage item={assistant({ id: "a10", text: "回答" })} onFeedback />
      </ToastProvider>
    ));
    fireEvent.click(view.getByRole("button", { name: "需改进" }));
    await vi.waitFor(() => expect(view.getByText(/反馈失败/)).toBeTruthy());
    // 回退：按钮恢复可点
    expect((view.getByRole("button", { name: "需改进" }) as HTMLButtonElement).disabled).toBe(false);
    expect(spy).toHaveBeenCalledTimes(1);
  });

  it("未传 onFeedback：不渲染反馈钮（能力面熄灯口径）", () => {
    const view = render(wrap(<AssistantMessage item={assistant({ text: "回答" })} />));
    expect(view.queryByRole("button", { name: "有用" })).toBeNull();
  });
});
