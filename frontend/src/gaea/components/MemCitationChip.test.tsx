import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { Markdown } from "./Markdown";
import { LocaleProvider } from "../lib/i18n";

const { memoryMock } = vi.hoisted(() => ({ memoryMock: vi.fn() }));

vi.mock("../lib/bridge", () => ({
  app: { Memory: memoryMock },
  openExternal: vi.fn(),
}));

function wrap(ui: React.ReactNode) {
  return <LocaleProvider>{ui}</LocaleProvider>;
}

describe("Markdown 记忆引用徽标（C2 引用可追溯）", () => {
  beforeEach(() => {
    memoryMock.mockReset();
  });

  it("[MEM:name] 渲染为引用徽标，点击弹层展示记忆详情与来源", async () => {
    memoryMock.mockResolvedValue({
      facts: [
        {
          name: "cost-rule",
          title: "成本测算规则",
          description: "先对科目再汇总",
          type: "project",
          body: "科目 → 数量×单价 → 汇总",
          lastUsedAt: "2026-08-29T08:00:00Z",
          sourceSession: "session-20260828-demo.jsonl",
          sourceMessage: "turn 3",
        },
      ],
    });
    render(wrap(<Markdown text={"按 [MEM:cost-rule] 的规则汇总，金额用公式。"} />));
    const chip = screen.getByRole("button", { name: /cost-rule/ });
    fireEvent.click(chip);
    await waitFor(() => expect(screen.getByText("成本测算规则")).toBeTruthy());
    expect(screen.getByText(/先对科目再汇总/)).toBeTruthy();
    expect(screen.getByText(/session-20260828-demo\.jsonl/)).toBeTruthy();
    // 再次点击关闭弹层
    fireEvent.click(chip);
    await waitFor(() => expect(screen.queryByText("成本测算规则")).toBeNull());
  });

  it("引用键在记忆库不存在时展示未找到提示（静默降级）", async () => {
    memoryMock.mockResolvedValue({ facts: [] });
    render(wrap(<Markdown text={"参考 [MEM:ghost-memory] 完成。"} />));
    fireEvent.click(screen.getByRole("button", { name: /ghost-memory/ }));
    await waitFor(() => expect(screen.getByText(/未在记忆库中找到该引用|Citation not found/)).toBeTruthy());
  });

  it("行内代码中的引用键不渲染为徽标", () => {
    render(wrap(<Markdown text={"格式说明：`[MEM:name]` 是引用键写法。"} />));
    expect(screen.queryByRole("button", { name: /name/ })).toBeNull();
  });
});
