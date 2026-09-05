// GenUI 蒸馏 · 办公验收走查（?mock=1 视觉通道不可用时的 jsdom DOM 等价路径）：
// 回答内嵌 genui → 消息流渲染 → action 回调 → panel 投递右栏 → 历史重放幂等。
import { afterEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { GenuiScopeProvider } from "../../genui/scope";
import { GenuiActionProvider, type GenuiActionHandler } from "../../genui/GenuiActionContext";
import { MemoMarkdown } from "./MemoMarkdown";
import { GenuiPanel } from "./GenuiPanel";
import { useGenuiPanelStore } from "../lib/genuiPanel";
import { getGenuiActionHandler, setGenuiActionHandler } from "../lib/genuiHost";

vi.mock("../lib/bridge", () => ({
  app: new Proxy({}, { get: () => () => Promise.resolve({}) }),
  openExternal: vi.fn(),
  onEvent: () => () => {},
}));

afterEach(() => {
  vi.useRealTimers();
  useGenuiPanelStore.setState({ sessions: {} });
  setGenuiActionHandler(undefined);
});

describe("办公 GenUI 验收走查", () => {
  it("回答内嵌 genui：消息流渲染组件，点击按钮经 action 宿主回调", () => {
    vi.useFakeTimers();
    const onAction = vi.fn<GenuiActionHandler>();
    const fence =
      '```genui\n{"title":"订单","items":[{"type":"stat","label":"营收","value":"¥128k"},{"type":"button","label":"刷新","action":"refresh"}]}\n```';
    render(
      <GenuiScopeProvider scope={{ scope: "office", sessionKey: "s1" }}>
        <GenuiActionProvider onAction={onAction}>
          <MemoMarkdown text={`看板如下\n${fence}`} streaming={false} genuiKey="a1" />
        </GenuiActionProvider>
      </GenuiScopeProvider>,
    );
    expect(screen.getByText("订单")).toBeTruthy();
    expect(screen.getByText("¥128k")).toBeTruthy();
    const btn = screen.getByRole("button", { name: "刷新" }) as HTMLButtonElement;
    expect(btn.disabled).toBe(false);
    fireEvent.click(btn);
    act(() => {
      vi.advanceTimersByTime(320);
    });
    expect(onAction).toHaveBeenCalledTimes(1);
    expect(onAction).toHaveBeenCalledWith("refresh", {});
  });

  it("panel:true 投递右栏 UI 面板；右栏可展示；重放同一消息不重复变更", () => {
    const fence =
      '```genui\n{"panel":true,"title":"成本看板","items":[{"type":"stat","label":"造价","value":"¥8.2万"}]}\n```';
    const shell = (
      <GenuiScopeProvider scope={{ scope: "office", sessionKey: "s1" }}>
        <MemoMarkdown text={fence} streaming={false} genuiKey="a2" />
      </GenuiScopeProvider>
    );
    const first = render(shell);
    expect(first.getByText("已更新 UI 面板")).toBeTruthy();
    expect(useGenuiPanelStore.getState().sessions["s1"]?.content?.items[0]).toMatchObject({
      type: "stat",
      label: "造价",
    });
    // 右栏 UI Tab 展示同一内容
    const panel = render(<GenuiPanel sessionPath="s1" />);
    expect(panel.getByText("成本看板")).toBeTruthy();
    expect(panel.getByText("造价")).toBeTruthy();
    // 历史重放（恢复会话/React 重挂）幂等：去重集合不增长、内容不被清空
    render(shell);
    expect(useGenuiPanelStore.getState().sessions["s1"]?.seen.size).toBe(1);
    expect(useGenuiPanelStore.getState().sessions["s1"]?.content?.items[0]).toMatchObject({
      label: "造价",
    });
  });

  it("右栏面板按钮经全局 action 宿主可点（App 注册后）", () => {
    vi.useFakeTimers();
    const host = vi.fn<GenuiActionHandler>();
    setGenuiActionHandler(host);
    useGenuiPanelStore.getState().publish("s2", "m1#0", {
      items: [{ type: "button", label: "重算", action: "recalc" }],
    });
    render(<GenuiPanel sessionPath="s2" />);
    const btn = screen.getByRole("button", { name: "重算" }) as HTMLButtonElement;
    expect(btn.disabled).toBe(false);
    fireEvent.click(btn);
    act(() => {
      vi.advanceTimersByTime(320);
    });
    expect(host).toHaveBeenCalledWith("recalc", {});
    expect(getGenuiActionHandler()).toBe(host);
  });
});
