import { afterEach, describe, expect, it } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { GenuiPanel } from "./GenuiPanel";
import { useGenuiPanelStore } from "../lib/genuiPanel";
import type { GenuiSpec } from "../../genui/spec";

const spec: GenuiSpec = {
  title: "订单看板",
  items: [
    { type: "stat", label: "营收", value: "¥128k" },
    { type: "button", label: "刷新", action: "refresh" },
  ],
};

afterEach(() => {
  useGenuiPanelStore.setState({ sessions: {} });
});

describe("GenuiPanel 会话 UI 面板", () => {
  it("空态提示；有内容时渲染 GenUI 块", () => {
    const { unmount } = render(<GenuiPanel sessionPath="s1" />);
    expect(screen.getByText(/内容会显示在这里/)).toBeTruthy();
    unmount();

    useGenuiPanelStore.getState().publish("s1", "m1#0", spec);
    render(<GenuiPanel sessionPath="s1" />);
    expect(screen.getByText("订单看板")).toBeTruthy();
    expect(screen.getByText("营收")).toBeTruthy();
    // 无宿主 action（测试环境未注册）→ 按钮诚实禁用
    expect((screen.getByRole("button", { name: "刷新" }) as HTMLButtonElement).disabled).toBe(true);
  });

  it("清空需两击确认", () => {
    useGenuiPanelStore.getState().publish("s2", "m1#0", spec);
    render(<GenuiPanel sessionPath="s2" />);
    const clear = screen.getByRole("button", { name: "清空" });
    fireEvent.click(clear);
    expect(screen.getByRole("button", { name: "确认清空？" })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "确认清空？" }));
    expect(screen.queryByText("订单看板")).toBeNull();
    expect(useGenuiPanelStore.getState().sessions["s2"]).toBeUndefined();
  });
});
