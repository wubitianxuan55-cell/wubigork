// CostEntryModal.test.tsx — 成本条目弹窗「组价依据」回看（v4.158 AI 组价复核闭环）。
// 覆盖：①有记录出只读折叠区（标题计数 + 时间倒序 + LLM/规则徽标），展开单条
// 显示价格带一行（P25/P50/P75）+ 人材机组成小表 + 证据链小表；②无记录不渲染
// （零噪音）；③加载失败静默收起（不弹错）；④编辑表单本身不受影响。
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { CostEntryModal } from "./CostEntryModal";
import type { CostComposeRecord, CostComposeView, CostSummary } from "../../lib/types";

const { costGetSpy, recordsSpy } = vi.hoisted(() => ({
  costGetSpy: vi.fn(),
  recordsSpy: vi.fn(),
}));

vi.mock("../../lib/bridge", () => ({
  app: {
    CostCategories: vi.fn().mockResolvedValue([]),
    CostGet: (...args: unknown[]) => costGetSpy(...args),
    CostComposeRecords: (...args: unknown[]) => recordsSpy(...args),
    CostSave: vi.fn().mockResolvedValue(undefined),
  },
}));

// 编辑对象（CostSummary 契约最小集）。
const ENTRY: CostSummary = {
  name: "c30",
  title: "C30 商品混凝土",
  category: "混凝土",
  categoryPath: "综合单价/混凝土",
  unit: "m³",
  price: 480,
  spec: "",
  source: "AI 组价",
  tags: [],
  status: "现行",
};

// 确认时留痕的完整视图快照（时间用无时区后缀的本地时间串，断言与时区无关）。
const SNAPSHOT: CostComposeView = {
  description: "C30 混凝土浇筑",
  unit: "m³",
  band: {
    samples: 6,
    min: 980,
    max: 2300,
    mean: 1418.33,
    median: 1200,
    p25: 1000,
    p75: 1400,
    spreadPct: 33.3,
    outliers: 1,
    confidence: "高",
    sources: [],
  },
  recommendedPrice: 1200,
  reason: "综合 6 个相似样本，中位数推荐。",
  components: [
    { kind: "人工", title: "混凝土浇筑人工", unit: "工日", quantity: 0.6, price: 250, amount: 150 },
    { kind: "材料", title: "C30 商品混凝土", unit: "m³", quantity: 1.02, price: 480, amount: 489.6 },
  ],
  llmUsed: true,
  evidence: [
    {
      name: "c30-b", title: "C30 商品混凝土 泵送", category: "混凝土", unit: "m³", spec: "C30 泵送",
      price: 1050, source: "四川造价信息网", region: "成都", priceDate: "2026-07", priceType: "到场价",
    },
  ],
};

// 两条记录（乱序传入，折叠区应按 createdAt 倒序展示）：新=LLM 含快照，旧=规则无快照。
const RECORDS: CostComposeRecord[] = [
  { id: 2, entryName: "c30", createdAt: "2026-09-01T10:30", llmUsed: true, snapshot: SNAPSHOT },
  { id: 1, entryName: "c30", createdAt: "2026-08-01T08:00", llmUsed: false, snapshot: null },
];

const renderModal = () =>
  render(<CostEntryModal open editing={ENTRY} onClose={() => {}} onSaved={() => {}} />);

beforeEach(() => {
  costGetSpy.mockReset().mockResolvedValue(null);
  recordsSpy.mockReset();
});

describe("CostEntryModal 组价依据回看（v4.158 复核闭环）", () => {
  it("有记录：出「组价依据（2 次）」折叠区，记录按时间倒序 + LLM/规则徽标 + 推荐价", async () => {
    recordsSpy.mockResolvedValue(RECORDS);
    renderModal();
    expect(await screen.findByText("组价依据（2 次）")).toBeTruthy();
    expect(recordsSpy).toHaveBeenCalledWith("c30");
    // 展开折叠区。
    fireEvent.click(screen.getByText("组价依据（2 次）"));
    // 时间倒序：新记录（LLM · ¥1,200/m³）在上，旧记录（规则 · 无快照价）在下。
    const llm = screen.getByText("LLM");
    const rule = screen.getByText("规则");
    expect(llm.compareDocumentPosition(rule) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(screen.getByText("¥1,200/m³")).toBeTruthy();
    expect(screen.getByText("2026-09-01 10:30")).toBeTruthy();
    expect(screen.getByText("2026-08-01 08:00")).toBeTruthy();
  });

  it("展开单条记录：价格带一行（P25/P50/P75）+ 人材机组成小表 + 证据链小表", async () => {
    recordsSpy.mockResolvedValue(RECORDS);
    renderModal();
    await screen.findByText("组价依据（2 次）");
    fireEvent.click(screen.getByText("组价依据（2 次）"));
    // 展开第一条（时间最新、含快照）记录。
    fireEvent.click(screen.getAllByText("展开")[0]);
    // 价格带一行。
    const bandLine = screen.getByText(/价格带/).closest("div");
    expect(bandLine?.textContent).toContain("P25 ¥1,000");
    expect(bandLine?.textContent).toContain("P50 ¥1,200");
    expect(bandLine?.textContent).toContain("P75 ¥1,400");
    // 人材机组成小表（只读；列头带（元），单元格为纯数字）。
    expect(screen.getByText("人材机组成（2 行）")).toBeTruthy();
    expect(screen.getByText("混凝土浇筑人工")).toBeTruthy();
    expect(screen.getByText("489.6")).toBeTruthy();
    // 证据链小表：与 ComposeModal 同一套溯源列（共享 ComposeEvidenceTable）。
    expect(screen.getByText("证据链（1 条）")).toBeTruthy();
    for (const h of ["标题", "来源", "地区", "期数", "口径"]) {
      expect(screen.getByRole("columnheader", { name: h })).toBeTruthy();
    }
    // 「单价(元)」列头两处都有：人材机组成小表 + 证据链小表。
    expect(screen.getAllByRole("columnheader", { name: "单价(元)" })).toHaveLength(2);
    expect(screen.getByText("四川造价信息网")).toBeTruthy();
    // 收起第一条，展开第二条（快照缺失）→ 只给占位说明，不渲染小表。
    fireEvent.click(screen.getByText("收起"));
    const expands = screen.getAllByText("展开");
    fireEvent.click(expands[expands.length - 1]);
    expect(screen.getByText("该次确认未留存快照，仅记录时间与拆解方式")).toBeTruthy();
    // 快照缺失记录不渲染小表（表单的「人材机组成（二级明细）」编辑项除外）。
    expect(screen.queryByText("人材机组成（2 行）")).toBeNull();
    expect(screen.queryByText("证据链（1 条）")).toBeNull();
  });

  it("无记录：折叠区不渲染（零噪音）", async () => {
    recordsSpy.mockResolvedValue([]);
    renderModal();
    await waitFor(() => expect(recordsSpy).toHaveBeenCalled());
    expect(screen.queryByText(/组价依据/)).toBeNull();
  });

  it("加载失败：静默收起（不渲染折叠区、不弹错误）", async () => {
    recordsSpy.mockRejectedValue(new Error("记录服务不可用"));
    renderModal();
    await waitFor(() => expect(recordsSpy).toHaveBeenCalled());
    expect(screen.queryByText(/组价依据/)).toBeNull();
    expect(screen.queryByRole("alert")).toBeNull();
  });

  it("编辑表单不受影响：标题/单价字段照常渲染，折叠区在表单之外", async () => {
    recordsSpy.mockResolvedValue(RECORDS);
    renderModal();
    // 表单字段先于记录加载完成即存在（编辑功能零改动）。
    expect(screen.getByText("标题")).toBeTruthy();
    expect(await screen.findByText("组价依据（2 次）")).toBeTruthy();
  });

  it("定额/清单编码（v4.178 匹配键刀）：编辑回显 + 保存随条目提交", async () => {
    const { app } = await import("../../lib/bridge");
    const saveSpy = app.CostSave as ReturnType<typeof vi.fn>;
    costGetSpy.mockResolvedValue({
      ...ENTRY,
      code: "A1-12",
      body: "",
      createdAt: "2026-09-01T00:00:00Z",
      updatedAt: "2026-09-01T00:00:00Z",
    });
    recordsSpy.mockResolvedValue([]);
    const { container } = renderModal();
    // 编辑回显：CostGet 返回的编码进入表单。
    const codeInput = (await screen.findByPlaceholderText("如：A1-12 / 040101001")) as HTMLInputElement;
    await waitFor(() => expect(codeInput.value).toBe("A1-12"));
    // 保存：编码随条目提交（antd 受控表单在 jsdom 下 DOM change 不同步
    // rc-field-form store，此处锁「编辑值进表单→随 CostSave 提交」链路）。
    fireEvent.click(screen.getByRole("button", { name: /保\s*存/ }));
    await waitFor(() => expect(saveSpy).toHaveBeenCalled());
    const payload = saveSpy.mock.calls[0][0] as { code?: string };
    expect(payload.code).toBe("A1-12");
    expect(container).toBeTruthy();
  });
});
