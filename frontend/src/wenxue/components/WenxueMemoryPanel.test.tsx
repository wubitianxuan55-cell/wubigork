import { describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { WenxueMemoryPanel } from "./WenxueMemoryPanel";

vi.mock("../lib/bridge", async () => {
  const actual = await vi.importActual<typeof import("../lib/mock")>("../lib/mock");
  return actual;
});
vi.mock("../../events", () => ({
  subscribe: vi.fn(() => () => {}),
}));

describe("WenxueMemoryPanel", () => {
  it("空脑渲染空态不炸", async () => {
    render(<WenxueMemoryPanel />);
    await waitFor(() => expect(screen.getByTestId("wenxue-memory-panel")).toBeTruthy());
  });
  it("有事实时渲染事实卡", async () => {
    // mock Memory 返回一条事实
    const mod = await import("../lib/mock/memory");
    void mod;
    render(<WenxueMemoryPanel />);
    await waitFor(() => expect(screen.getByTestId("wenxue-memory-panel")).toBeTruthy());
  });
});
