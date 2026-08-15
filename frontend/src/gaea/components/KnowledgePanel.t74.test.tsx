// T7-4（v2.37.0）KnowledgePanel 加载三态：失败=错误 + 重试按钮。
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { KnowledgePanel } from "./KnowledgePanel";
import { LocaleProvider } from "../lib/i18n";
import type { KnowledgeSummary } from "../lib/types";

// jsdom 环境 localStorage 缺失（存量环境问题）：i18n 的 readPref 依赖它。
beforeAll(() => {
  if (typeof window !== "undefined" && (window.localStorage?.getItem == null)) {
    Object.defineProperty(window, "localStorage", {
      writable: true,
      value: { getItem: () => null, setItem: () => {}, removeItem: () => {} },
    });
  }
});

const wrap = (node: React.ReactNode) => <LocaleProvider>{node}</LocaleProvider>;
type G = { go?: { app?: Record<string, unknown> } };

const entry: KnowledgeSummary = {
  name: "demo-001", title: "测试条目 A", category: "工程案例", tags: [], status: "现行", updatedAt: "2025-01-01T00:00:00.000Z",
};

describe("KnowledgePanel 加载三态（T7-4）", () => {
  afterEach(() => {
    delete (window as unknown as G).go;
  });

  it("列表加载失败：显示错误信息 + 重试按钮，不再无限 loading/假空列表", async () => {
    const f: Record<string, unknown> = { GaeaKnowledgeList: vi.fn().mockRejectedValue(new Error("list down")) };
    (window as unknown as G).go = { app: { CoreB: f } };
    render(wrap(<KnowledgePanel onClose={() => {}} variant="page" />));

    expect(await screen.findByText(/加载失败：list down/)).toBeTruthy();
    expect(screen.getByText("重试")).toBeTruthy();
    expect(screen.queryByText("Loading…")).toBeNull(); // 不再停留 loading
  });

  it("点击重试：恢复列表数据并清除错误态", async () => {
    const f: Record<string, unknown> = { GaeaKnowledgeList: vi.fn().mockRejectedValue(new Error("list down")) };
    (window as unknown as G).go = { app: { CoreB: f } };
    render(wrap(<KnowledgePanel onClose={() => {}} variant="page" />));
    await screen.findByText(/加载失败/);

    (f.GaeaKnowledgeList as ReturnType<typeof vi.fn>).mockResolvedValue([entry]);
    fireEvent.click(screen.getByText("重试"));

    expect(await screen.findByText("测试条目 A")).toBeTruthy();
    expect(screen.queryByText(/加载失败/)).toBeNull();
  });
});
