import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, act } from "@testing-library/react";
import { ComputerStatusBar } from "./ComputerStatusBar";

// W6：状态条监听 computer-status——using 显示「正在操控 <窗口>」，idle 消失。
const handlers = new Map<string, (p: unknown) => void>();
const fakeRuntime = {
  EventsOn: vi.fn((channel: string, handler: (p: unknown) => void) => {
    handlers.set(channel, handler);
    return () => handlers.delete(channel);
  }),
};

describe("ComputerStatusBar", () => {
  beforeEach(() => {
    handlers.clear();
    vi.clearAllMocks();
    (window as unknown as { runtime?: unknown }).runtime = fakeRuntime;
  });

  it("using 状态显示正在操控，idle 后消失", () => {
    render(<ComputerStatusBar />);
    expect(screen.queryByTestId("computer-status-bar")).toBeNull();
    act(() => {
      handlers.get("computer-status")?.({
        state: "using",
        app: "process:C:\\windows\\notepad.exe",
        window: "记事本",
        turn_id: "",
      });
    });
    const bar = screen.getByTestId("computer-status-bar");
    expect(bar.textContent).toContain("正在操控 记事本");
    expect(bar.textContent).toContain("notepad");
    act(() => {
      handlers.get("computer-status")?.({ state: "idle" });
    });
    expect(screen.queryByTestId("computer-status-bar")).toBeNull();
  });

  it("无 runtime（预览壳等）不挂监听不渲染", () => {
    (window as unknown as { runtime?: unknown }).runtime = undefined;
    render(<ComputerStatusBar />);
    expect(screen.queryByTestId("computer-status-bar")).toBeNull();
  });
});
