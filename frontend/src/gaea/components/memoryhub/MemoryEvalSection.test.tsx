import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryEvalSection } from "./MemoryEvalSection";
import type { MemoryEvalReport } from "../../lib/types/memory";

// 注入体检面板测试（市场调研候选2）：mock bridge，验证通过/违规两态渲染与
// 运行按钮链路（确定性纯读，无模型调用）。

const { evalRunMock } = vi.hoisted(() => ({ evalRunMock: vi.fn() }));

vi.mock("../../lib/bridge", () => ({
  app: { MemoryEvalRun: evalRunMock },
}));

vi.mock("antd", async () => {
  const actual = await vi.importActual<typeof import("antd")>("antd");
  return { ...actual };
});

const PASS: MemoryEvalReport = {
  memoryEnabled: true,
  morningPreload: true,
  projectBrief: true,
  spaceModeOn: true,
  preloadPresent: true,
  briefPresent: true,
  preloadRunes: 96,
  briefRunes: 148,
  preloadBudget: 600,
  briefBudget: 600,
  entryCount: 2,
  refCount: 2,
  pinnedTotal: 1,
  pinnedInBrief: 1,
  violations: [],
  passed: true,
};

describe("MemoryEvalSection 记忆注入体检", () => {
  beforeEach(() => {
    evalRunMock.mockReset();
  });

  it("运行体检：通过态渲染两块计数与固化覆盖", async () => {
    evalRunMock.mockResolvedValue(PASS);
    render(<MemoryEvalSection open onClose={() => {}} />);
    // Drawer 挂 portal，按钮以文案定位
    fireEvent.click(screen.getByText("运行体检"));
    await waitFor(() => expect(evalRunMock).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(screen.getByText(/通过：五条结构不变量全部成立/)).toBeTruthy());
    expect(screen.getByText(/2 条 · 96\/600 runes/)).toBeTruthy();
    expect(screen.getByText(/2 引用 · 148\/600 runes/)).toBeTruthy();
    expect(screen.getByText(/1\/1/)).toBeTruthy();
  });

  it("违规态：逐条透出违规且通过横幅不出现", async () => {
    evalRunMock.mockResolvedValue({
      ...PASS,
      passed: false,
      briefPresent: false,
      violations: ["本体块引用 [MEM:ghost] 不在 work 视图（悬空注入）"],
    });
    render(<MemoryEvalSection open onClose={() => {}} />);
    fireEvent.click(screen.getByText("运行体检"));
    await waitFor(() =>
      expect(screen.getByText(/未通过：存在违规项/)).toBeTruthy(),
    );
    expect(screen.getByText(/悬空注入/)).toBeTruthy();
    expect(screen.queryByText(/五条结构不变量全部成立/)).toBeNull();
  });

  it("失败回退：绑定报错不崩面板（错误走 message，不渲染报告）", async () => {
    evalRunMock.mockRejectedValue(new Error("办公记忆库不可用"));
    render(<MemoryEvalSection open onClose={() => {}} />);
    fireEvent.click(screen.getByText("运行体检"));
    await waitFor(() => expect(evalRunMock).toHaveBeenCalledTimes(1));
    // 报告区仍为未运行提示
    expect(screen.getByText(/尚未运行/)).toBeTruthy();
  });
});
