import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, act, fireEvent } from "@testing-library/react";
import { BrowserPanel, extractBrowserActions, type BrowserObserveView } from "./BrowserPanel";
import type { Trajectory, TrajectoryRecord, TrajectoryToolRec } from "../lib/types";

// BrowserPanel 测试（v4.28 A2）：空态可用性门控、截图/时间线渲染、
// 2.5s 轮询的可见性门控（vi.useFakeTimers）、自动弹出开关持久化。

const AVAILABLE_VIEW: BrowserObserveView = {
  available: true,
  url: "http://fake.local/page",
  title: "观察页",
  image: "data:image/jpeg;base64,ZmFrZWpwZWc=",
  width: 1280,
  height: 600,
  updatedAt: Date.now() - 30_000,
};

function toolRec(seq: number, ts: number, tool: Partial<TrajectoryToolRec> & { name: string }): TrajectoryRecord {
  return { seq, kind: "tool", ts, tool: { id: `t${seq}`, status: "ok", ...tool } };
}

function userRec(seq: number, ts: number): TrajectoryRecord {
  return { seq, kind: "user", ts, user: { text: "帮忙打开 example.com" } };
}

const TRAJECTORY: Trajectory = {
  ok: true,
  turns: [
    {
      turn: 1,
      records: [
        userRec(1, 1000),
        toolRec(2, 2000, { name: "readfile", args: '{"path":"a.md"}' }),
        toolRec(3, 3000, { name: "browser_navigate", args: '{"url":"http://fake.local/page"}' }),
        toolRec(4, 4000, { name: "browser_click", status: "error", err: "未找到元素：#x", args: '{"selector":"#x"}' }),
        toolRec(5, 5000, { name: "browser_type", args: '{"selector":"#q","text":"gaea"}' }),
      ],
    },
  ],
  betweenTurns: [toolRec(6, 6000, { name: "browser_snapshot" })],
};

describe("extractBrowserActions 轨迹过滤（browser_* 记录 → 时间线行）", () => {
  it("只收 browser_* 记录，倒序（新→旧），非 browser 工具被滤掉", () => {
    const rows = extractBrowserActions(TRAJECTORY);
    expect(rows.map((r) => r.name)).toEqual(["browser_snapshot", "browser_type", "browser_click", "browser_navigate"]);
    expect(rows.some((r) => r.name === "readfile")).toBe(false);
    expect(rows.find((r) => r.name === "browser_click")?.err).toBe("未找到元素：#x");
  });

  it("参数摘要：JSON 取字段串联；超长截断", () => {
    const rows = extractBrowserActions(TRAJECTORY);
    expect(rows.find((r) => r.name === "browser_navigate")?.args).toBe("url=http://fake.local/page");
    expect(rows.find((r) => r.name === "browser_type")?.args).toBe("selector=#q · text=gaea");
    const long = extractBrowserActions({
      ok: true,
      turns: [{ turn: 1, records: [toolRec(1, 1, { name: "browser_read", args: JSON.stringify({ url: "x".repeat(200) }) })] }],
    });
    expect(long[0].args.length).toBeLessThanOrEqual(81);
    expect(long[0].args.endsWith("…")).toBe(true);
  });

  it("上限 20 条（只看最近动作），空轨迹返回空", () => {
    const many: Trajectory = {
      ok: true,
      turns: [
        {
          turn: 1,
          records: Array.from({ length: 23 }, (_, i) => toolRec(i + 1, i + 1, { name: "browser_scroll" })),
        },
      ],
    };
    const rows = extractBrowserActions(many);
    expect(rows).toHaveLength(20);
    expect(rows[0].ts).toBe(23); // 最新在前
    expect(extractBrowserActions(null)).toEqual([]);
  });
});

describe("BrowserPanel 观察窗", () => {
  beforeEach(() => {
    try { localStorage.removeItem("gaea.browserAutoOpen"); } catch { /* ignore */ }
  });
  afterEach(() => {
    vi.useRealTimers();
    // 恢复被测试改写的 visibilityState（jsdom 默认 visible）
    delete (document as unknown as Record<string, unknown>).visibilityState;
  });

  it("可用性门控：浏览器未运行 → 空态，不出截图（观察是被动动作）", async () => {
    const observe = vi.fn().mockResolvedValue({ ...AVAILABLE_VIEW, available: false, url: "", title: "", image: "", width: 0, height: 0, updatedAt: 0 });
    render(<BrowserPanel observe={observe} fetchTrajectory={vi.fn().mockResolvedValue(TRAJECTORY)} />);
    const empty = await screen.findByTestId("browser-empty");
    expect(empty.textContent).toContain("受控浏览器未运行");
    expect(empty.textContent).toContain("自动拉起");
    expect(screen.queryByTestId("browser-shot")).toBeNull();
  });

  it("观察帧渲染：URL/标题行 + 截图 + 权限说明；时间线只列 browser_* 且倒序", async () => {
    render(<BrowserPanel observe={vi.fn().mockResolvedValue(AVAILABLE_VIEW)} fetchTrajectory={vi.fn().mockResolvedValue(TRAJECTORY)} />);
    await screen.findByTestId("browser-shot");
    expect((screen.getByTestId("browser-shot") as HTMLImageElement).src).toBe(AVAILABLE_VIEW.image);
    expect(screen.getByTestId("browser-url").textContent).toBe("http://fake.local/page");
    expect(screen.getByTestId("browser-title").textContent).toBe("观察页");
    expect(screen.getByTestId("browser-perm-note").textContent).toContain("只读观察");
    const rows = screen.getAllByTestId("browser-timeline-row");
    expect(rows).toHaveLength(4);
    expect(rows[0].textContent).toContain("browser_snapshot");
    expect(rows[3].textContent).toContain("browser_navigate");
    expect(rows.some((r) => r.textContent!.includes("readfile"))).toBe(false);
  });

  it("放大层：点击截图弹出、Esc/点击背景关闭", async () => {
    render(<BrowserPanel observe={vi.fn().mockResolvedValue(AVAILABLE_VIEW)} fetchTrajectory={vi.fn().mockResolvedValue(null)} />);
    fireEvent.click(await screen.findByTestId("browser-shot"));
    expect(screen.getByTestId("browser-zoom")).toBeTruthy();
    fireEvent.keyDown(window, { key: "Escape" });
    expect(screen.queryByTestId("browser-zoom")).toBeNull();
    fireEvent.click(screen.getByTestId("browser-shot"));
    fireEvent.click(screen.getByTestId("browser-zoom"));
    expect(screen.queryByTestId("browser-zoom")).toBeNull();
  });

  it("2.5s 自动轮询仅当页面可见且可用；隐藏后停拍、恢复后继续（手动刷新不受门控）", async () => {
    vi.useFakeTimers();
    const observe = vi.fn().mockResolvedValue({ ...AVAILABLE_VIEW });
    const { unmount } = render(<BrowserPanel observe={observe} fetchTrajectory={vi.fn().mockResolvedValue(null)} />);
    await act(async () => { /* flush 挂载首拍 */ });
    const initial = observe.mock.calls.length;
    expect(initial).toBeGreaterThanOrEqual(1);

    // 可见 + 可用：前进 2.5s → 恰好一拍
    await act(async () => { vi.advanceTimersByTime(2500); });
    expect(observe.mock.calls.length).toBe(initial + 1);

    // 页面隐藏 → 门控关：interval 还在但空转
    Object.defineProperty(document, "visibilityState", { value: "hidden", configurable: true });
    await act(async () => {
      document.dispatchEvent(new Event("visibilitychange"));
      await Promise.resolve();
    });
    const paused = observe.mock.calls.length;
    await act(async () => { vi.advanceTimersByTime(7500); });
    expect(observe.mock.calls.length).toBe(paused);

    // 恢复可见 → 补拍继续
    Object.defineProperty(document, "visibilityState", { value: "visible", configurable: true });
    await act(async () => {
      document.dispatchEvent(new Event("visibilitychange"));
      await Promise.resolve();
    });
    await act(async () => { vi.advanceTimersByTime(2500); });
    expect(observe.mock.calls.length).toBeGreaterThan(paused);
    unmount();

    // 未运行（available=false）→ 不自动轮询，但手动刷新仍会探测
    const offline = vi.fn().mockResolvedValue({ ...AVAILABLE_VIEW, available: false });
    render(<BrowserPanel observe={offline} fetchTrajectory={vi.fn().mockResolvedValue(null)} />);
    await act(async () => { /* flush 首拍 */ });
    const off = offline.mock.calls.length;
    await act(async () => { vi.advanceTimersByTime(7500); });
    expect(offline.mock.calls.length).toBe(off);
    await act(async () => { fireEvent.click(screen.getByLabelText("刷新观察帧")); });
    expect(offline.mock.calls.length).toBe(off + 1);
  });

  it("观察失败降级：reject → 按未运行兜底，不抛未处理错误", async () => {
    const observe = vi.fn().mockRejectedValue(new Error("boom"));
    render(<BrowserPanel observe={observe} fetchTrajectory={vi.fn().mockRejectedValue(new Error("boom"))} />);
    const empty = await screen.findByTestId("browser-empty");
    expect(empty).toBeTruthy();
  });

  it("自动弹出开关：点击持久化到 gaea.browserAutoOpen 并翻转 aria-pressed", async () => {
    render(<BrowserPanel observe={vi.fn().mockResolvedValue({ ...AVAILABLE_VIEW, available: false, image: "" })} fetchTrajectory={vi.fn().mockResolvedValue(null)} />);
    const toggle = await screen.findByTestId("browser-auto-open-toggle");
    expect(toggle.getAttribute("aria-pressed")).toBe("true");
    fireEvent.click(toggle);
    expect(localStorage.getItem("gaea.browserAutoOpen")).toBe("0");
    expect(toggle.getAttribute("aria-pressed")).toBe("false");
    fireEvent.click(toggle);
    expect(localStorage.getItem("gaea.browserAutoOpen")).toBe("1");
  });
});
