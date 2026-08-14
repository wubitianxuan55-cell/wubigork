import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { CostCompareModal } from "./CostCompareModal";
import { ToastProvider } from "../Toast";

const { compareSpy } = vi.hoisted(() => ({ compareSpy: vi.fn() }));

vi.mock("../../lib/bridge", () => ({
  app: {
    CostCompare: (...args: unknown[]) => compareSpy(...args),
  },
}));

const wrap = (node: React.ReactNode) => <ToastProvider>{node}</ToastProvider>;

// CostCompareRow 契约样例（kind：current/history/fetch）。
const ROWS = [
  { source: "重庆造价信息网", period: "2026-08", price: 4000, diffPct: 25, fetchedAt: "2026-08-10T00:00:00Z", kind: "fetch" },
  { source: "四川造价信息网", period: "2026-07", price: 3456, diffPct: 8, fetchedAt: "2026-07-20T00:00:00Z", kind: "fetch" },
  { source: "成本库", period: "", price: 3300, diffPct: 3.1, fetchedAt: "", kind: "current" },
  { source: "市场询价", period: "2026-05", price: 3050, diffPct: -4.7, fetchedAt: "2026-05-01T00:00:00Z", kind: "history" },
];

describe("CostCompareModal 供应商比价", () => {
  beforeEach(() => {
    compareSpy.mockReset();
  });

  it("展示多源对比，跳幅按 |diffPct| 着色（≥20 红 / >5 琥珀 / 其余绿）", async () => {
    compareSpy.mockResolvedValue(ROWS);
    render(
      wrap(
        <CostCompareModal
          open
          name="cement"
          title="P.O 42.5 水泥"
          currentPrice={3200}
          onClose={() => {}}
        />,
      ),
    );

    // 各来源行渲染（来源 + 价格 + 期数 + kind 中文标签）。
    expect(await screen.findByText("重庆造价信息网")).toBeTruthy();
    expect(screen.getByText("四川造价信息网")).toBeTruthy();
    expect(screen.getByText("成本库")).toBeTruthy();
    expect(screen.getByText("市场询价")).toBeTruthy();
    expect(screen.getByText("¥4,000")).toBeTruthy();
    expect(screen.getByText("¥3,456")).toBeTruthy();
    expect(compareSpy).toHaveBeenCalledWith("cement");

    // kind 中文标注：现价 / 历史快照 / 价格源抓取。
    expect(screen.getByText("现价")).toBeTruthy();
    expect(screen.getByText("历史快照")).toBeTruthy();
    expect(screen.getByText("市场询价")).toBeTruthy();
    expect(screen.getAllByText("价格源抓取").length).toBeGreaterThanOrEqual(2);

    // 现价基准行展示。
    expect(screen.getByText("¥3,200")).toBeTruthy();

    // 跳幅着色：+25% 红、+8% 琥珀、+3.1% 与 -4.7% 绿。
    const red = screen.getByText("+25%");
    expect(red.className).toContain("text-red-400");
    const amber = screen.getByText("+8%");
    expect(amber.className).toContain("text-amber-400");
    const greenUp = screen.getByText("+3.1%");
    expect(greenUp.className).toContain("text-emerald-400");
    const greenDown = screen.getByText("-4.7%");
    expect(greenDown.className).toContain("text-emerald-400");
  });

  it("无其他来源时展示「暂无其他来源」", async () => {
    compareSpy.mockResolvedValue([]);
    render(
      wrap(
        <CostCompareModal
          open
          name="cement"
          title="P.O 42.5 水泥"
          onClose={() => {}}
        />,
      ),
    );
    expect(await screen.findByText("暂无其他来源")).toBeTruthy();
  });

  it("查询失败展示持久错误提示", async () => {
    compareSpy.mockRejectedValue(new Error("比价服务不可用"));
    render(
      wrap(
        <CostCompareModal
          open
          name="cement"
          title="P.O 42.5 水泥"
          onClose={() => {}}
        />,
      ),
    );
    // 错误持久展示在弹窗内（role=alert；toast 也会短暂出现同样文案，避免多匹配）。
    const alert = await screen.findByRole("alert");
    expect(alert.textContent).toContain("比价失败");
    expect(alert.textContent).toContain("比价服务不可用");
  });
});
