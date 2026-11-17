import { afterEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, render, screen } from "@testing-library/react";
import { HintStrip } from "./HintStrip";
import type { HintItem } from "../../lib/hintLayout";

const nav: HintItem = { id: "nav", keys: ["↑", "↓"], label: "导航" };
const select: HintItem = { id: "select", keys: ["↵"], label: "选择" };
const close: HintItem = { id: "close", keys: ["esc"], label: "关闭" };
const variants = [[nav, select, close], [nav, close], [close]];
const originalResizeObserver = window.ResizeObserver;

describe("HintStrip", () => {
  afterEach(() => {
    cleanup();
    window.ResizeObserver = originalResizeObserver;
  });

  it("switches to a whole shorter variant and disconnects the observer", () => {
    let observerCallback: ResizeObserverCallback = () => {};
    const observe = vi.fn();
    const disconnect = vi.fn();
    class ObserverStub {
      constructor(callback: ResizeObserverCallback) {
        observerCallback = callback;
      }
      observe = observe;
      disconnect = disconnect;
      unobserve = vi.fn();
    }
    window.ResizeObserver = ObserverStub as unknown as typeof ResizeObserver;

    const view = render(<HintStrip variants={variants} />);
    expect(observe).toHaveBeenCalledTimes(1);
    expect(screen.getByText("选择")).toBeTruthy();

    act(() => {
      observerCallback([{ contentRect: { width: 12 } } as ResizeObserverEntry], {} as ResizeObserver);
    });
    expect(screen.getByText("关闭")).toBeTruthy();
    expect(screen.queryByText("选择")).toBeNull();
    expect(screen.queryByText("导航")).toBeNull();

    view.unmount();
    expect(disconnect).toHaveBeenCalledTimes(1);
  });
});
