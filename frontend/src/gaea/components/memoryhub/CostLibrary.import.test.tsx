import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { CostLibrary } from "./CostLibrary";
import { ToastProvider } from "../Toast";

// 真实导入结果（来自用户实际导入 Hephaestus.db 的 5 条条目）。
const REAL_ENTRIES = [
  { name: "外加剂单价-元-t", title: "外加剂单价 (元/t)", category: "材料", unit: "元/t", price: 3500, spec: "", source: "市场参考", tags: [], status: "现行", updatedAt: "2026-08-10T06:19:18Z" },
  { name: "桩机台班费-含制浆站-泵-空压机-吊车配套-元-台班", title: "桩机台班费（含制浆站/泵/空压机/吊车配套） (元/台班)", category: "其他", unit: "元/台班", price: 6500, spec: "", source: "三轴搅拌桩机租赁+折旧+动力，参考 5500~7500", tags: [], status: "现行", updatedAt: "2026-08-10T06:19:18Z" },
  { name: "水泥单价-p-o42-5-散装-元-t", title: "水泥单价 P.O42.5 散装 (元/t)", category: "材料", unit: "元/t", price: 350, spec: "", source: "重庆市场参考 330~350", tags: [], status: "现行", updatedAt: "2026-08-10T06:19:18Z" },
  { name: "水费-每幅-米-元", title: "水费（每幅·米） (元)", category: "材料", unit: "元", price: 1, spec: "", source: "制浆用水分摊", tags: [], status: "现行", updatedAt: "2026-08-10T06:19:18Z" },
  { name: "膨润土单价-元-t", title: "膨润土单价 (元/t)", category: "材料", unit: "元/t", price: 900, spec: "", source: "市场参考 800~1000", tags: [], status: "现行", updatedAt: "2026-08-10T06:19:18Z" },
];

const { searchSpy, pickSpy, previewSpy, applySpy } = vi.hoisted(() => ({
  searchSpy: vi.fn(),
  pickSpy: vi.fn(),
  previewSpy: vi.fn(),
  applySpy: vi.fn(),
}));

vi.mock("../../lib/bridge", () => ({
  app: {
    CostSearch: (...args: unknown[]) => searchSpy(...args),
    CostList: async () => REAL_ENTRIES,
    CostGet: async () => null,
    CostSave: async () => {},
    CostDelete: async () => {},
    PickFiles: (...args: unknown[]) => pickSpy(...args),
    CostImportPreview: (...args: unknown[]) => previewSpy(...args),
    CostImportAIParse: async () => ({ rows: [] }),
    CostImportApply: (...args: unknown[]) => applySpy(...args),
    CostCategories: async () => [],
    CostCategorySave: async () => 1,
    CostCategoryDelete: async () => {},
    PriceSources: async () => [],
    PriceSourceSave: async () => {},
    PriceSourceDelete: async () => {},
    PriceFetch: async () => ({}),
    PriceFetches: async () => [],
    PriceFetchApply: async () => 0,
    PriceFetchIgnore: async () => {},
    PriceHistory: async () => [],
  },
}));

const wrap = (node: React.ReactNode) => <ToastProvider>{node}</ToastProvider>;

describe("CostLibrary 真实导入数据渲染", () => {
  beforeEach(() => {
    searchSpy.mockReset();
    pickSpy.mockReset();
    previewSpy.mockReset();
    applySpy.mockReset();
    searchSpy.mockResolvedValue(REAL_ENTRIES);
    pickSpy.mockResolvedValue([{ path: "C:/AI/bangong/三轴搅拌桩成本测算表.xlsx", name: "三轴搅拌桩成本测算表.xlsx", size: 209142 }]);
    previewSpy.mockResolvedValue({
      path: "C:/AI/bangong/三轴搅拌桩成本测算表.xlsx",
      fileName: "三轴搅拌桩成本测算表.xlsx",
      columns: ["参数名", "数值", "说明", "单位"],
      unmapped: [],
      message: "未识别到横向表头，已按纵向参数表提取 5 条单价类条目。",
      aiUsed: false,
      rows: REAL_ENTRIES.map((e) => ({
        name: e.name, title: e.title, category: e.category, unit: e.unit,
        price: e.price, spec: e.spec, source: e.source, status: e.status,
        existingName: "", existingPrice: 0, matchNote: "新增",
        raw: `${e.title} | ${e.price} | ${e.source} | ${e.unit}`, skip: false, skipReason: "",
      })),
    });
    applySpy.mockResolvedValue(REAL_ENTRIES.length);
  });

  it("导入真实条目→列表渲染→弹窗关闭，全程无崩溃且不重复请求", async () => {
    render(wrap(<CostLibrary />));

    // 初始列表渲染真实条目。
    expect(await screen.findByText("水泥单价 P.O42.5 散装 (元/t)")).toBeTruthy();
    expect(screen.getByText("桩机台班费（含制浆站/泵/空压机/吊车配套） (元/台班)")).toBeTruthy();

    // 导入：选文件 → 预览 → 确认导入。
    fireEvent.click(screen.getByTitle("导入 xlsx/csv 报价单或测算表"));
    expect(await screen.findByText(/已按纵向参数表提取 5 条单价类条目/)).toBeTruthy();
    fireEvent.click(screen.getByText("确认导入 5 条"));

    // 成功提示出现、弹窗关闭。
    await waitFor(() => expect(screen.getByText("已导入 5 条成本条目")).toBeTruthy());
    await waitFor(() => expect(screen.queryByText(/导入成本：/)).toBeNull());

    // 列表刷新后仍渲染 5 条真实条目。
    expect(await screen.findByText("外加剂单价 (元/t)")).toBeTruthy();
    expect(screen.getByText("膨润土单价 (元/t)")).toBeTruthy();
    expect(screen.getByText("水费（每幅·米） (元)")).toBeTruthy();
    expect(screen.getByText("¥3,500")).toBeTruthy();
    expect(screen.getByText("¥6,500")).toBeTruthy();

    // 等待 toast 2 秒自动消失：不应触发任何重渲染循环或重复请求。
    await new Promise((r) => setTimeout(r, 2600));
    expect(searchSpy).toHaveBeenCalled();
    expect(applySpy).toHaveBeenCalledTimes(1);
    expect(screen.getByText("水泥单价 P.O42.5 散装 (元/t)")).toBeTruthy();
  });
});
