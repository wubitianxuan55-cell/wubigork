import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { CostInquiryPanel } from "./CostInquiryPanel";
import { ToastProvider } from "../Toast";
import type { CostAdjustSuggestion, CostEntry, CostInquiryRecord } from "../../lib/types";

const { listSpy, expiringSpy, saveSpy, deleteSpy, adjustSpy, getSpy, costSaveSpy } = vi.hoisted(() => ({
  listSpy: vi.fn(),
  expiringSpy: vi.fn(),
  saveSpy: vi.fn(),
  deleteSpy: vi.fn(),
  adjustSpy: vi.fn(),
  getSpy: vi.fn(),
  costSaveSpy: vi.fn(),
}));

vi.mock("../../lib/bridge", () => ({
  app: {
    CostInquiryList: (...args: unknown[]) => listSpy(...args),
    CostInquiryExpiring: (...args: unknown[]) => expiringSpy(...args),
    CostInquirySave: (...args: unknown[]) => saveSpy(...args),
    CostInquiryDelete: (...args: unknown[]) => deleteSpy(...args),
    CostInquiryAdjust: (...args: unknown[]) => adjustSpy(...args),
    CostGet: (...args: unknown[]) => getSpy(...args),
    CostSave: (...args: unknown[]) => costSaveSpy(...args),
  },
}));

const wrap = (node: React.ReactNode) => <ToastProvider>{node}</ToastProvider>;

// CostInquiryRecord 契约样例：四源各一条。
const RECORDS: CostInquiryRecord[] = [
  { id: 1, title: "P.O 42.5 水泥", spec: "散装", unit: "吨", price: 380, source: "信息价", supplier: "华新水泥", region: "成都", priceDate: "2026-08", validUntil: "2026-12-31", note: "", status: "现行", createdAt: "", updatedAt: "" },
  { id: 2, title: "HRB400 螺纹钢", spec: "Φ16", unit: "吨", price: 4020, source: "OCR报价", supplier: "攀钢集团", region: "成都", priceDate: "2026-07", validUntil: "", note: "含税到站价", status: "现行", createdAt: "", updatedAt: "" },
  { id: 3, title: "HP300 高频液压振动锤", spec: "300kW", unit: "台班", price: 3450, source: "供应商比价", supplier: "中联重科", region: "重庆", priceDate: "2026-08", validUntil: "2026-12-31", note: "", status: "现行", createdAt: "", updatedAt: "" },
  { id: 4, title: "商品混凝土 C30", spec: "", unit: "m³", price: 430, source: "手动询价", supplier: "", region: "成都", priceDate: "2026-08", validUntil: "2026-09-15", note: "", status: "现行", createdAt: "", updatedAt: "" },
];

// 到期预警数据点（与列表独立，避免文本双匹配）。
const EXPIRING: CostInquiryRecord[] = [
  { id: 20, title: "SBS 改性沥青卷材", spec: "4mm", unit: "m²", price: 32.5, source: "信息价", supplier: "科顺防水", region: "成都", priceDate: "2026-07", validUntil: "2026-08-30", note: "", status: "现行", createdAt: "", updatedAt: "" },
  { id: 21, title: "铝合金窗", spec: "断桥 60 系列", unit: "m²", price: 520, source: "手动询价", supplier: "坚朗", region: "成都", priceDate: "2026-08", validUntil: "2026-09-10", note: "", status: "现行", createdAt: "", updatedAt: "" },
];

// 调差建议：+30%（红，≥10）/ +5%（琥珀，>2）/ -2%（绿，降）。
const ADJUSTS: CostAdjustSuggestion[] = [
  { entryName: "cement", entryTitle: "P.O 42.5 水泥", entryPrice: 350, latestPrice: 455, latestDate: "2026-08-10", latestSource: "信息价", diff: 105, diffPct: 30, unit: "吨" },
  { entryName: "rebar", entryTitle: "HRB400 螺纹钢", entryPrice: 4000, latestPrice: 4200, latestDate: "2026-08-10", latestSource: "供应商比价", diff: 200, diffPct: 5, unit: "吨" },
  { entryName: "concrete", entryTitle: "商品混凝土 C30", entryPrice: 440, latestPrice: 431.2, latestDate: "2026-08-10", latestSource: "手动询价", diff: -8.8, diffPct: -2, unit: "m³" },
];

const CEMENT_ENTRY = {
  name: "cement",
  title: "P.O 42.5 水泥",
  category: "材料",
  categoryPath: "材料/水泥",
  unit: "吨",
  price: 350,
  spec: "散装",
  source: "成本库",
  region: "成都",
  priceDate: "2026-06",
  validUntil: "",
  tags: [],
  status: "现行",
  body: "",
  updatedAt: "2026-08-01T00:00:00Z",
} satisfies CostEntry;

describe("CostInquiryPanel 询价飞轮", () => {
  beforeEach(() => {
    listSpy.mockReset();
    expiringSpy.mockReset();
    saveSpy.mockReset();
    deleteSpy.mockReset();
    adjustSpy.mockReset();
    getSpy.mockReset();
    costSaveSpy.mockReset();
    listSpy.mockResolvedValue(RECORDS);
    expiringSpy.mockResolvedValue([]);
    adjustSpy.mockResolvedValue([]);
  });

  it("渲染到期预警横幅：列出 30 天内到期的数据点（品名/有效期/价格）", async () => {
    expiringSpy.mockResolvedValue(EXPIRING);
    render(wrap(<CostInquiryPanel />));

    expect(await screen.findByText("⚠ 2 条询价到期预警")).toBeTruthy();
    expect(screen.getByText("SBS 改性沥青卷材")).toBeTruthy();
    expect(screen.getByText("铝合金窗")).toBeTruthy();
    expect(screen.getByText("至 2026-09-10")).toBeTruthy();
    expect(screen.getByText("¥32.5")).toBeTruthy();
    expect(expiringSpy).toHaveBeenCalledWith(30);
  });

  it("渲染数据点列表与四源徽标（不同颜色）", async () => {
    render(wrap(<CostInquiryPanel />));

    expect(await screen.findByText("P.O 42.5 水泥")).toBeTruthy();
    expect(screen.getByText("HRB400 螺纹钢")).toBeTruthy();
    expect(screen.getByText("HP300 高频液压振动锤")).toBeTruthy();
    expect(screen.getByText("商品混凝土 C30")).toBeTruthy();

    // 四源徽标配色区分。
    expect(screen.getByText("信息价").className).toContain("text-sky-400");
    expect(screen.getByText("OCR报价").className).toContain("text-purple-400");
    expect(screen.getByText("供应商比价").className).toContain("text-emerald-400");
    expect(screen.getByText("手动询价").className).toContain("text-amber-400");

    // 行内信息：供应商 / 规格 / 单价。
    expect(screen.getByText("供应商：华新水泥")).toBeTruthy();
    expect(screen.getByText("Φ16")).toBeTruthy();
    expect(screen.getByText("¥4,020")).toBeTruthy();

    expect(listSpy).toHaveBeenCalledWith("", 100);
  });

  it("搜索防抖 250ms：输入后以新词调用 CostInquiryList", async () => {
    render(wrap(<CostInquiryPanel />));
    await screen.findByText("P.O 42.5 水泥");
    expect(listSpy).toHaveBeenCalledWith("", 100);

    vi.useFakeTimers();
    try {
      fireEvent.change(screen.getByPlaceholderText(/搜索品名/), { target: { value: "螺纹钢" } });
      // 250ms 内不触发新查询。
      expect(listSpy).not.toHaveBeenCalledWith("螺纹钢", 100);
      act(() => {
        vi.advanceTimersByTime(300);
      });
      expect(listSpy).toHaveBeenCalledWith("螺纹钢", 100);
    } finally {
      vi.useRealTimers();
    }
  });

  it("新增询价：填表保存调用 CostInquirySave（spy 断言参数）", async () => {
    saveSpy.mockResolvedValue(6);
    render(wrap(<CostInquiryPanel />));
    await screen.findByText("P.O 42.5 水泥");

    fireEvent.click(screen.getByTitle("新增询价"));
    const dialog = await screen.findByRole("dialog");

    fireEvent.change(within(dialog).getByLabelText("品名"), { target: { value: "铝合金窗" } });
    fireEvent.change(within(dialog).getByLabelText("规格"), { target: { value: "断桥 60 系列" } });
    fireEvent.change(within(dialog).getByLabelText("单位"), { target: { value: "m²" } });
    fireEvent.change(within(dialog).getByLabelText("单价"), { target: { value: "520" } });
    fireEvent.change(within(dialog).getByLabelText("来源"), { target: { value: "OCR报价" } });
    fireEvent.change(within(dialog).getByLabelText("供应商"), { target: { value: "坚朗" } });
    fireEvent.change(within(dialog).getByLabelText("地区"), { target: { value: "成都" } });
    fireEvent.change(within(dialog).getByLabelText("期数"), { target: { value: "2026-08" } });
    fireEvent.change(within(dialog).getByLabelText("有效期至"), { target: { value: "2026-12-31" } });
    fireEvent.change(within(dialog).getByLabelText("备注"), { target: { value: "含安装" } });

    fireEvent.click(within(dialog).getByRole("button", { name: /保\s*存/ }));

    await waitFor(() =>
      expect(saveSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          id: 0,
          title: "铝合金窗",
          spec: "断桥 60 系列",
          unit: "m²",
          price: 520,
          source: "OCR报价",
          supplier: "坚朗",
          region: "成都",
          priceDate: "2026-08",
          validUntil: "2026-12-31",
          note: "含安装",
        }),
      ),
    );
    // 保存成功后弹窗关闭并出现成功提示。
    expect(await screen.findByText("询价已保存")).toBeTruthy();
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  });

  it("删除询价：确认弹窗 → 调用 CostInquiryDelete → 提示", async () => {
    deleteSpy.mockResolvedValue(undefined);
    render(wrap(<CostInquiryPanel />));
    await screen.findByText("P.O 42.5 水泥");

    fireEvent.click(screen.getAllByTitle("删除")[0]);
    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).getAllByText("删除询价数据").length).toBeGreaterThan(0);

    fireEvent.click(within(dialog).getByRole("button", { name: /删\s*除/ }));

    await waitFor(() => expect(deleteSpy).toHaveBeenCalledWith(1));
    expect(await screen.findByText("询价已删除")).toBeTruthy();
  });

  it("调差建议：差幅着色 + 更新成本库（CostGet → CostSave 仅改 price）", async () => {
    adjustSpy.mockResolvedValue(ADJUSTS);
    getSpy.mockResolvedValue(CEMENT_ENTRY);
    costSaveSpy.mockResolvedValue(undefined);
    render(wrap(<CostInquiryPanel />));

    expect(await screen.findByText("调差建议")).toBeTruthy();

    // 现价 → 最新价 + 差幅着色：+30% 红、+5% 琥珀、-2% 绿。
    expect(screen.getByText("¥350 → ¥455")).toBeTruthy();
    expect(screen.getByText("+30%").className).toContain("text-red-400");
    expect(screen.getByText("+5%").className).toContain("text-amber-400");
    expect(screen.getByText("-2%").className).toContain("text-emerald-400");

    // 更新成本库：先 CostGet 取原条目，再 CostSave 保留 unit/分类等、仅改 price。
    fireEvent.click(screen.getAllByText("更新成本库")[0]);
    await waitFor(() => expect(getSpy).toHaveBeenCalledWith("cement"));
    await waitFor(() =>
      expect(costSaveSpy).toHaveBeenCalledWith(
        expect.objectContaining({ name: "cement", title: "P.O 42.5 水泥", price: 455, unit: "吨", categoryPath: "材料/水泥" }),
      ),
    );
    expect(await screen.findByText(/已更新成本库/)).toBeTruthy();
    // 建议区刷新。
    await waitFor(() => expect(adjustSpy).toHaveBeenCalledTimes(2));
  });

  it("无数据时展示空态提示", async () => {
    listSpy.mockResolvedValue([]);
    render(wrap(<CostInquiryPanel />));

    expect(await screen.findByText(/暂无询价数据——导入报价单（成本库导入）或手动录入/)).toBeTruthy();
  });
});
