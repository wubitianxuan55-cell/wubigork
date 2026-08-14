import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { CostLibraryView, CostRow, ListView } from "./CostLibraryView";

const CAT_TREE = [
  {
    id: 2, parentId: 0, name: "材料", sort: 0, count: 0,
    children: [
      {
        id: 21, parentId: 2, name: "土建材料", sort: 0, count: 0,
        children: [
          { id: 211, parentId: 21, name: "钢材", sort: 0, count: 1 },
        ],
      },
    ],
  },
  {
    id: 3, parentId: 0, name: "机械", sort: 0, count: 0,
    children: [{ id: 31, parentId: 3, name: "桩基机械", sort: 0, count: 1 }],
  },
];

const ENTRIES = [
  {
    name: "steel-h", title: "H 型钢", category: "钢材", categoryPath: "材料/土建材料/钢材",
    unit: "吨", price: 5200, spec: "400×200", source: "市场询价", tags: [], status: "现行", updatedAt: "2026-08-11T00:00:00Z",
  },
  {
    name: "hp300", title: "HP300 高频液压振动锤", category: "桩基机械", categoryPath: "机械/桩基机械",
    unit: "台班", price: 3200, spec: "300kW", source: "市场询价", tags: [], status: "现行", updatedAt: "2026-08-11T00:00:00Z",
  },
];

const { searchSpy } = vi.hoisted(() => ({ searchSpy: vi.fn() }));

// 比价弹层 mock 数据（CostCompareRow 契约样例）。
const COMPARE_ROWS = [
  { source: "重庆造价信息网", period: "2026-08", price: 4000, diffPct: 25, fetchedAt: "2026-08-10T00:00:00Z", kind: "fetch" },
  { source: "市场询价", period: "2026-06", price: 3300, diffPct: 3.1, fetchedAt: "2026-06-15T00:00:00Z", kind: "history" },
];

vi.mock("../lib/bridge", () => ({
  app: {
    CostSearch: (...args: unknown[]) => searchSpy(...args),
    CostCategories: async () => CAT_TREE,
    CostGet: async () => null,
    CostSave: async () => {},
    CostDelete: async () => {},
    CostCategorySave: async () => 1,
    CostCategoryDelete: async () => {},
    PickFiles: async () => [],
    CostImportPreview: async () => ({ rows: [] }),
    CostImportAIParse: async () => ({ rows: [] }),
    CostImportApply: async () => 0,
    CostCompare: async () => COMPARE_ROWS,
    PriceHistory: async () => [],
  },
}));

describe("CostLibraryView 多级分类 + 列表/表格", () => {
  beforeEach(() => {
    searchSpy.mockReset();
    searchSpy.mockResolvedValue(ENTRIES);
  });

  it("渲染多级分类树与条目，点击分类按路径过滤", async () => {
    render(<CostLibraryView />);

    expect(await screen.findByText("H 型钢")).toBeTruthy();
    expect(screen.getByText("HP300 高频液压振动锤")).toBeTruthy();

    // 分类树渲染（含三级「钢材」）。
    expect(screen.getByText("材料")).toBeTruthy();
    expect(screen.getByText("机械")).toBeTruthy();
    expect(screen.getByText("土建材料")).toBeTruthy();
    expect(screen.getByText("钢材")).toBeTruthy();

    // 展开「材料」→ 选中「钢材」，搜索按完整路径过滤。
    fireEvent.click(screen.getByTitle("材料"));
    fireEvent.click(screen.getByText("钢材"));
    await waitFor(() => expect(searchSpy).toHaveBeenCalledWith("", "材料/土建材料/钢材", "all"));
  });

  it("条目行比价按钮打开供应商比价弹层", async () => {
    render(<CostLibraryView />);
    await screen.findByText("H 型钢");

    // 列表条目行含「比价」入口（标题处 + 操作区），点击打开弹层。
    fireEvent.click(screen.getAllByTitle("比价")[0]);
    expect(await screen.findByText(/供应商比价：H 型钢/)).toBeTruthy();
    expect(screen.getByText("重庆造价信息网")).toBeTruthy();
    expect(screen.getByText("¥4,000")).toBeTruthy();
  });

  it("切换表格视图显示排序表头与条目行", async () => {
    render(<CostLibraryView />);
    await screen.findByText("H 型钢");

    fireEvent.click(screen.getByTitle("表格视图"));

    expect(screen.getByText("单价（元）")).toBeTruthy();
    expect(screen.getByText("规格")).toBeTruthy();
    expect(screen.getByText("H 型钢")).toBeTruthy();
    expect(screen.getByText("材料/土建材料/钢材")).toBeTruthy();

    // 点单价表头排序（升序后 HP300 在前）。
    fireEvent.click(screen.getByText(/单价（元）/));
    const priceCells = screen.getAllByText(/^¥/);
    expect(priceCells[0].textContent).toBe("¥3,200");
  });
  it("CostRow memo：相同 props 不重渲染，selected 变化才重渲染", () => {
    const priceSpy = vi.fn((p: number) => "¥" + p);
    const callbacks = {
      onToggleSelect: vi.fn(),
      onEdit: vi.fn(),
      onDelete: vi.fn(),
      onHistory: vi.fn(),
      onCompare: vi.fn(),
    };
    const props = {
      row: ENTRIES[0],
      selected: false,
      compact: false,
      priceText: priceSpy,
      ...callbacks,
    };
    const { rerender } = render(<CostRow {...props} />);
    const base = priceSpy.mock.calls.length;
    expect(base).toBeGreaterThan(0);

    // 相同 props 重渲染 → memo 跳过，行体不再执行（priceText 不再被调用）
    rerender(<CostRow {...props} />);
    expect(priceSpy.mock.calls.length).toBe(base);

    // selected 变化 → 该行重渲染
    rerender(<CostRow {...props} selected={true} />);
    expect(priceSpy.mock.calls.length).toBeGreaterThan(base);
  });

  it("ListView memo：相同 props 重渲染时列表项不重渲染", () => {
    const priceSpy = vi.fn((p: number) => "¥" + p);
    const props = {
      rows: ENTRIES,
      selected: new Set<string>(),
      toggleSelect: vi.fn(),
      priceText: priceSpy,
      onEdit: vi.fn(),
      onDelete: vi.fn(),
      onHistory: vi.fn(),
      onCompare: vi.fn(),
      compact: false,
    };
    const { rerender } = render(<ListView {...props} />);
    const base = priceSpy.mock.calls.length;
    expect(base).toBeGreaterThan(0);

    // 相同 props → ListView memo 跳过，行不重渲染
    rerender(<ListView {...props} />);
    expect(priceSpy.mock.calls.length).toBe(base);

    // 选中集变化 → 对应行重渲染
    rerender(<ListView {...props} selected={new Set([ENTRIES[0].name])} />);
    expect(priceSpy.mock.calls.length).toBeGreaterThan(base);
  });
});
