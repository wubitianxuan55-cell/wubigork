import { describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { WenxueMemoryPanel } from "./WenxueMemoryPanel";

// 数据面 mock：与兄弟 Hub 测试同惯用法（显式 app 桩）。旧工厂（importActual
// "../lib/mock"）缺 app 导出，前两例实为「app=undefined→错误态」空跑——
// 本文件 2026-11-16 审查批1 重写后 Memory 桩真实生效。
vi.mock("../lib/bridge", () => ({
  app: {
    // available 必须为真：面板事实列表门闸 = view.available（缺省=「图文脑不可用」分支）。
    Memory: vi.fn(async () => ({ available: true, facts: [] })),
    WenxueMemoryArchivedList: vi.fn(async () => []),
  },
}));

describe("WenxueMemoryPanel", () => {
  it("空脑渲染空态不炸", async () => {
    render(<WenxueMemoryPanel />);
    await waitFor(() => expect(screen.getByTestId("wenxue-memory-panel")).toBeTruthy());
  });
  it("有事实时渲染事实卡", async () => {
    const { app } = await import("../lib/bridge");
    vi.mocked(app.Memory).mockResolvedValueOnce({
      available: true,
      facts: [{ type: "user", name: "写作偏好", title: "", body: "第三人称限知视角" }],
    } as never);
    render(<WenxueMemoryPanel />);
    await waitFor(() => expect(screen.getByText("写作偏好")).toBeTruthy());
    expect(screen.getByText(/第三人称限知视角/)).toBeTruthy();
  });
  it("切书 DOM 事件（wenxue:workspace-changed）触发重载", async () => {
    // 审查批1 行为锁：旧实现用 Wails subscribe 订 DOM CustomEvent（两套系统
    // 不通=订空气），切书后记忆面板永不重载。修复后必须走 window 监听。
    const { app } = await import("../lib/bridge");
    const memoryFn = vi.mocked(app.Memory);
    render(<WenxueMemoryPanel />);
    await waitFor(() => expect(memoryFn.mock.calls.length).toBeGreaterThan(0));
    const before = memoryFn.mock.calls.length;
    window.dispatchEvent(new CustomEvent("wenxue:workspace-changed"));
    await waitFor(() => expect(memoryFn.mock.calls.length).toBeGreaterThan(before));
  });
});
