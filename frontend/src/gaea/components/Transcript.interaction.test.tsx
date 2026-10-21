// v4.510 办公对话流交互三修回归：
// ①「回到底部」按钮必须是滚动容器 .transcript 的兄弟节点（结构契约）——
//    此前 absolute 挂在滚动容器内部，绝对定位随内容滚走（无头 Edge 实证：
//    scrollTop=900 时按钮已滚出视口 666px），长会话上翻时按钮恰在视口外
//    永远不可见，只有首屏内短暂可见；
// ②回退菜单真实事件序（mousedown → click）下回退动作必须真正执行——
//    Transcript 的外点守卫在 document 上监听 mousedown，closest(".rewind")
//    必须命中回退行（此前该类不存在于任何元素：点菜单项的 mousedown 先
//    触发守卫关菜单，click 落在已卸载的按钮上，回退从未真正执行；
//    fireEvent.click 只派发单事件，对此类缺陷恒假绿——必须 mouseDown+click
//    双事件复现真实浏览器序）；
// ③回退行自带定位锚（relative）：菜单 bottom-full 相对本行而非一路锚到
//    滚动容器顶部（此前菜单渲染在内容流顶端之外基本不可见）。
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render } from "@testing-library/react";
import { LocaleProvider } from "../lib/i18n";
import { Transcript } from "./Transcript";
import { initialState, useStore } from "../lib/store";
import type { Item } from "../lib/store";

const wrap = (node: React.ReactNode) => {
  localStorage.setItem("gaea-lang", "zh");
  return <LocaleProvider>{node}</LocaleProvider>;
};

const items: Item[] = [
  { kind: "user", id: "u1", text: "第一问" },
  { kind: "assistant", id: "a1", text: "第一答", reasoning: "", streaming: false },
];

describe("回到底部按钮（v4.510 兄弟挂载根修）", () => {
  beforeEach(() => {
    useStore.setState({ ...initialState, _dispatch: useStore.getState()._dispatch });
  });

  it("上翻后按钮渲染且在滚动容器之外（不随内容滚动）", () => {
    useStore.setState({ items, running: false, turnStartAt: Date.now() });
    const view = render(wrap(<Transcript onPrompt={() => {}} running={false} />));
    const scroller = view.container.querySelector(".transcript");
    expect(scroller).not.toBeNull();
    // 初始贴底：按钮不渲染
    expect(view.queryByRole("button", { name: "回到底部" })).toBeNull();

    // jsdom 无布局：直接伪造成「长会话上翻」的滚动几何（2000 内容/600 视口/滚到 800）
    const el = scroller as HTMLElement;
    Object.defineProperty(el, "scrollHeight", { value: 2000, configurable: true });
    Object.defineProperty(el, "clientHeight", { value: 600, configurable: true });
    Object.defineProperty(el, "scrollTop", { value: 800, configurable: true });
    fireEvent.scroll(el);

    const fab = view.getByRole("button", { name: "回到底部" });
    // 结构契约：按钮绝不在滚动容器内部（内部 absolute 会随内容滚走）
    expect(el.contains(fab)).toBe(false);
    // 按钮与滚动容器同父（锚定外层包装，上翻时常驻视口底部）
    expect(fab.parentElement).toBe(el.parentElement);
  });
});

describe("回退菜单（v4.510 锚定+外点守卫双修）", () => {
  beforeEach(() => {
    useStore.setState({ ...initialState, _dispatch: useStore.getState()._dispatch });
  });

  it("真实事件序 mouseDown→click 点菜单项：回退回调真正执行", () => {
    const onRewind = vi.fn();
    useStore.setState({ items, running: false, turnStartAt: Date.now() });
    const view = render(wrap(<Transcript onPrompt={() => {}} running={false} onRewind={onRewind} />));

    // 打开菜单（click 不触发守卫，模拟点击回退钮）
    fireEvent.click(view.getByRole("button", { name: "回退" }));

    // 守卫选择器契约：回退行挂 .rewind 类（外点守卫 closest(".rewind") 的命中目标）
    const triggerRow = view.getByRole("button", { name: "回退" }).parentElement;
    expect(triggerRow?.className).toContain("rewind");
    // 定位锚契约：回退行自带 relative（菜单 bottom-full 相对本行）
    expect(triggerRow?.className).toContain("relative");

    // 真实浏览器序：mousedown 先于 click——守卫不误关菜单，onClick 真正执行
    const scopeBtn = view.getByRole("button", { name: "文件 + 对话" });
    fireEvent.mouseDown(scopeBtn);
    fireEvent.click(scopeBtn);
    expect(onRewind).toHaveBeenCalledWith(0, "both");
  });

  it("点菜单外任意处：菜单关闭（守卫仍然生效）", () => {
    const onRewind = vi.fn();
    useStore.setState({ items, running: false, turnStartAt: Date.now() });
    const view = render(wrap(<Transcript onPrompt={() => {}} running={false} onRewind={onRewind} />));

    fireEvent.click(view.getByRole("button", { name: "回退" }));
    expect(view.getByRole("button", { name: "文件 + 对话" })).toBeTruthy();

    // 菜单外 mousedown（正文区域）：关闭
    fireEvent.mouseDown(view.getByText("第一答"));
    expect(view.queryByRole("button", { name: "文件 + 对话" })).toBeNull();
    expect(onRewind).not.toHaveBeenCalled();
  });
});

// ── v4.512 流式滚动手势化 ────────────────────────────────────────────
// stick（跟随）的清除权唯一归滚轮上滚手势；scroll 事件只在贴底时恢复跟随，
// 不再在不贴底时清除——折叠卡收起时浏览器滚动锚定/夹紧产生的程序性 scroll
// 事件曾把跟随误杀（v2.13.5「冻住假象」根因，当时以运行中强拉底部权宜，
// 代价=流式生成时用户无法上翻阅读）。三钉：①运行中上翻=暂停（内容更新
// 不再拽回）；②布局扰动（无手势的 scroll 事件）杀不死跟随（v2.13.5
// 回归钉）；③「回到底部」点击恢复跟随。
describe("流式滚动跟随手势化（v4.512）", () => {
  beforeEach(() => {
    useStore.setState({ ...initialState, _dispatch: useStore.getState()._dispatch });
  });

  // jsdom 无布局：伪造成长会话滚动几何（2000 内容/600 视口），scrollTop
  // 为可写属性（Transcript 的落底=赋值 scrollHeight，可观测）。
  const mockGeometry = (el: HTMLElement, top: number) => {
    let cur = top;
    Object.defineProperty(el, "scrollTop", { configurable: true, get: () => cur, set: (v: number) => { cur = v; } });
    Object.defineProperty(el, "scrollHeight", { configurable: true, value: 2000 });
    Object.defineProperty(el, "clientHeight", { configurable: true, value: 600 });
    return { set: (v: number) => { cur = v; }, get: () => cur };
  };
  const nextFrame = () => new Promise<void>((r) => requestAnimationFrame(() => r()));

  it("运行中滚轮上翻=暂停跟随：后续内容更新不拽回（v2.13.5 强拉退役）", async () => {
    useStore.setState({ items, running: true, turnStartAt: Date.now() });
    const view = render(wrap(<Transcript onPrompt={() => {}} running={true} />));
    const el = view.container.querySelector(".transcript") as HTMLElement;
    const geo = mockGeometry(el, 2000);
    await nextFrame(); // 挂载自动落底（top=2000）

    // 上滚手势 + 离开底部（读历史）
    fireEvent.wheel(el, { deltaY: -120 });
    geo.set(800);
    fireEvent.scroll(el);

    // 流式新 chunk（内容变化触发滚动跟随路径）
    useStore.setState({
      items: [...items, { kind: "assistant", id: "a2", text: "流式第二段", reasoning: "", streaming: true }],
    });
    await nextFrame();

    // 暂停生效：不拽回 2000，停在阅读位置
    expect(geo.get()).toBe(800);
  });

  it("布局扰动（无手势 scroll 事件）杀不死跟随：v2.13.5 冻住假象不复发", async () => {
    useStore.setState({ items, running: true, turnStartAt: Date.now() });
    const view = render(wrap(<Transcript onPrompt={() => {}} running={true} />));
    const el = view.container.querySelector(".transcript") as HTMLElement;
    const geo = mockGeometry(el, 2000);
    await nextFrame();

    // 程序性扰动：滚动锚定/夹紧把视口挪到中部（scroll 事件但无滚轮手势）
    geo.set(800);
    fireEvent.scroll(el);

    // 内容更新：跟随仍活着，自动补拉回底部
    useStore.setState({
      items: [...items, { kind: "assistant", id: "a2", text: "流式第二段", reasoning: "", streaming: true }],
    });
    await nextFrame();
    expect(geo.get()).toBe(2000);
  });

  it("暂停后「回到底部」点击恢复跟随", async () => {
    useStore.setState({ items, running: true, turnStartAt: Date.now() });
    const view = render(wrap(<Transcript onPrompt={() => {}} running={true} />));
    const el = view.container.querySelector(".transcript") as HTMLElement;
    const geo = mockGeometry(el, 2000);
    await nextFrame();

    fireEvent.wheel(el, { deltaY: -120 });
    geo.set(800);
    fireEvent.scroll(el);
    const fab = view.getByRole("button", { name: "回到底部" });
    fireEvent.click(fab);
    await nextFrame();

    expect(geo.get()).toBe(2000);
  });
});
