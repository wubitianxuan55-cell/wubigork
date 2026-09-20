import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { CostLibraryView, CostRow, ListView } from "./CostLibraryView";
import { message } from "antd";

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

const { searchSpy, statusSpy, backfillSpy } = vi.hoisted(() => ({
  searchSpy: vi.fn(),
  statusSpy: vi.fn(),
  backfillSpy: vi.fn(),
}));

// 比价弹层 mock 数据（CostCompareRow 契约样例）。
const COMPARE_ROWS = [
  { source: "重庆造价信息网", period: "2026-08", price: 4000, diffPct: 25, fetchedAt: "2026-08-10T00:00:00Z", kind: "fetch" },
  { source: "市场询价", period: "2026-06", price: 3300, diffPct: 3.1, fetchedAt: "2026-06-15T00:00:00Z", kind: "history" },
];

vi.mock("../lib/bridge", () => ({
  app: {
    CostSearch: (...args: unknown[]) => searchSpy(...args),
    SemanticIndexStatus: (...args: unknown[]) => statusSpy(...args),
    SemanticIndexBackfill: (...args: unknown[]) => backfillSpy(...args),
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
  statusSpy.mockResolvedValue({ total: 2, indexed: 1, modelOk: true, modelNote: "" });
  backfillSpy.mockResolvedValue({ total: 2, updated: 1, indexed: 2, missing: 0 });
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

  it("语义索引 chip：部分覆盖可点击补齐，全覆盖禁用", async () => {
    // 首次状态=部分覆盖；补齐后 loadIndex 刷新为全覆盖。
    statusSpy.mockResolvedValueOnce({ total: 2, indexed: 1, modelOk: true, modelNote: "" });
    statusSpy.mockResolvedValue({ total: 2, indexed: 2, modelOk: true, modelNote: "" });
    render(<CostLibraryView />);
    const chip = await screen.findByTestId("semantic-index-chip");
    await waitFor(() => expect(chip.textContent).toContain("1/2"));
    expect((chip as HTMLButtonElement).disabled).toBe(false);
    const callsBefore = backfillSpy.mock.calls.length; // spy 计数跨用例累计，只断言增量
    fireEvent.click(chip);
    await waitFor(() => expect(backfillSpy.mock.calls.length).toBeGreaterThan(callsBefore));
    // 补齐后刷新状态 → 全覆盖 → 禁用。
    await waitFor(() => expect(chip.textContent).toContain("2/2"));
    expect((chip as HTMLButtonElement).disabled).toBe(true);
  });

  it("语义索引 chip：模型未配置显示未启用且不可点", async () => {
    statusSpy.mockResolvedValue({ total: 2, indexed: 0, modelOk: false, modelNote: "本地语义模型未配置" });
    const callsBefore = backfillSpy.mock.calls.length;
    render(<CostLibraryView />);
    const chip = await screen.findByTestId("semantic-index-chip");
    expect(chip.textContent).toContain("未启用");
    expect(chip.tagName).toBe("SPAN");
    expect(backfillSpy.mock.calls.length).toBe(callsBefore);
  });
});

describe("批量改状态失败可见化（v4.361）", () => {
  it("逐条失败时计数警告提示而非假成功", async () => {
    const warnSpy = vi.spyOn(message, "warning");
    const infoSpy = vi.spyOn(message, "info");
    render(<CostLibraryView />);
    await waitFor(() => expect(screen.getByText("H 型钢")).toBeTruthy());

    // 勾选第一条（CostGet mock 返回 null=读取失败→该条计入失败）
    fireEvent.click(screen.getAllByRole("checkbox")[0]);
    expect(screen.getByText(/已选 1/)).toBeTruthy();

    // 触发批量改状态（工具条上的「改状态…」select——页内有多个 select，按
    // option 文本精确定位）
    const select = Array.from(document.querySelectorAll("select")).find(
      (el) => el.textContent?.includes("改状态"),
    ) as HTMLSelectElement | undefined;
    expect(select).toBeTruthy();
    fireEvent.change(select!, { target: { value: "草稿" } });

    await waitFor(() => expect(warnSpy).toHaveBeenCalled(), { timeout: 3000 });
    expect(String(warnSpy.mock.calls[0][0])).toContain("1 条失败");
    expect(infoSpy).not.toHaveBeenCalled();
    warnSpy.mockRestore();
    infoSpy.mockRestore();
  });
});
