// 知识库绑定面 nil 切片 JSON null 崩前端回归钉（对齐 v4.467.1 造价同款契约）：
// 旧构建 Go 产物 tags/columns/rows 序列化为 null，KnowledgePanel/EntryRow/EditForm
// 裸读 .length/.map/.join 进 ErrorBoundary。修复后 Go 构造即非 nil，此处用 null
// 桩钉前端兜底，保证对旧构建 Go 产物同样免疫。
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
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

// 模拟旧构建 Go 产物：tags 序列化为 null（TS 类型不认，运行时真实形态）。
const nullTagsEntry = {
  name: "demo-001", title: "测试条目 A", category: "工程案例", phase: "实施", tags: null, status: "现行", updatedAt: "2025-01-01T00:00:00.000Z",
} as unknown as KnowledgeSummary;

describe("KnowledgePanel nil 切片 JSON null 兜底", () => {
  afterEach(() => {
    delete (window as unknown as G).go;
  });

  it("page 变体：tags=null 条目正常渲染，不进 ErrorBoundary", async () => {
    const f: Record<string, unknown> = { GaeaKnowledgeList: vi.fn().mockResolvedValue([nullTagsEntry]) };
    (window as unknown as G).go = { app: { CoreB: f } };
    render(wrap(<KnowledgePanel onClose={() => {}} variant="page" />));

    expect(await screen.findByText("测试条目 A")).toBeTruthy();
    expect(screen.queryByText(/加载失败/)).toBeNull();
  });

  it("modal 变体：tags=null 条目正常渲染（v4.469 真机崩溃点）", async () => {
    const f: Record<string, unknown> = { GaeaKnowledgeList: vi.fn().mockResolvedValue([nullTagsEntry]) };
    (window as unknown as G).go = { app: { CoreB: f } };
    render(wrap(<KnowledgePanel onClose={() => {}} />));

    expect(await screen.findByText("测试条目 A")).toBeTruthy();
    expect(screen.queryByText(/加载失败/)).toBeNull();
  });

  it("列表整体为 null（空库降级）渲染空态，不崩", async () => {
    const f: Record<string, unknown> = { GaeaKnowledgeList: vi.fn().mockResolvedValue(null) };
    (window as unknown as G).go = { app: { CoreB: f } };
    render(wrap(<KnowledgePanel onClose={() => {}} variant="page" />));

    expect(await screen.findByText("Knowledge base is empty")).toBeTruthy();
    expect(screen.queryByText(/加载失败/)).toBeNull();
  });
});
