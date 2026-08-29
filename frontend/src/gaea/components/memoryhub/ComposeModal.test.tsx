import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { ComposeModal } from "./ComposeModal";
import { ToastProvider } from "../Toast";
import type { CostComponent, CostComposeEvidence, CostComposeView } from "../../lib/types";

const { composeSpy } = vi.hoisted(() => ({ composeSpy: vi.fn() }));

vi.mock("../../lib/bridge", () => ({
  app: {
    CostCompose: (...args: unknown[]) => composeSpy(...args),
  },
}));

const wrap = (node: React.ReactNode) => <ToastProvider>{node}</ToastProvider>;

// 渲染弹窗（open 恒 true），返回可断言的 onApply/onClose spy。
const renderModal = (over: { initialDesc?: string; initialUnit?: string } = {}) => {
  const onApply = vi.fn();
  const onClose = vi.fn();
  render(
    wrap(
      <ComposeModal
        open
        initialDesc={over.initialDesc ?? "C30 混凝土浇筑"}
        initialUnit={over.initialUnit ?? "m³"}
        onClose={onClose}
        onApply={onApply}
      />,
    ),
  );
  return { onApply, onClose };
};

const compose = () => {
  fireEvent.click(screen.getByText("开始组价"));
};

// 应用回调负载（与组件 onApply 契约对齐）。
interface ApplyResult {
  desc: string;
  unit: string;
  price: number;
  components: CostComponent[];
  evidence: CostComposeEvidence[];
}

// ── CostComposeView 契约样例（band 分位数 R-7 口径；sources 已按价格升序）──
// 离群口径：IQR = P75-P25 = 400；越界 <1000-600=400 或 >1400+600=2000 → 2300 为离群。
const BAND = {
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
  sources: [
    { name: "c30-a", title: "C30 商品混凝土 非泵送", category: "混凝土", unit: "m³", spec: "C30 非泵送", source: "重庆造价信息网", region: "重庆", priceDate: "2026-05", priceType: "出厂价", price: 980, updatedAt: "" },
    { name: "c30-b", title: "C30 商品混凝土 泵送", category: "混凝土", unit: "m³", spec: "C30 泵送", source: "四川造价信息网", region: "成都", priceDate: "2026-07", priceType: "到场价", price: 1050, updatedAt: "" },
    { name: "c30-c", title: "C30 自拌混凝土", category: "混凝土", unit: "m³", spec: "C30 现场搅拌", source: "成本库", region: "贵阳", priceDate: "2026-06", priceType: "出厂价", price: 1220, updatedAt: "" },
    { name: "c30-d", title: "C30 细石混凝土", category: "混凝土", unit: "m³", spec: "C30 细石", source: "历史项目", region: "昆明", priceDate: "2026-04", priceType: "安装综合价", price: 1380, updatedAt: "" },
    { name: "c30-e", title: "C30 抗渗混凝土", category: "混凝土", unit: "m³", spec: "C30 P6 抗渗", source: "市场询价", region: "成都", priceDate: "2026-08", priceType: "到场价", price: 1580, updatedAt: "" },
    { name: "c30-f", title: "C30 高标号混凝土", category: "混凝土", unit: "m³", spec: "C30 高强", source: "供应商报价", region: "绵阳", priceDate: "2026-03", priceType: "出厂价", price: 2300, updatedAt: "" },
  ],
};

const COMPONENTS = [
  { kind: "人工", title: "混凝土浇筑人工", unit: "工日", quantity: 0.6, price: 250, amount: 150 },
  { kind: "材料", title: "C30 商品混凝土", unit: "m³", quantity: 1.02, price: 480, amount: 489.6 },
];

const VIEW: CostComposeView = {
  description: "C30 混凝土浇筑",
  unit: "m³",
  band: BAND,
  recommendedPrice: 1200,
  reason: "综合 6 个相似样本，中位数 1,200 元/m³，置信度高，推荐取中位数。",
  components: COMPONENTS,
  componentsNote: "损耗系数已计入含量",
  llmUsed: true,
  evidence: [
    { name: "c30-b", title: "C30 商品混凝土 泵送", category: "混凝土", unit: "m³", spec: "C30 泵送", price: 1050, source: "四川造价信息网", region: "成都", priceDate: "2026-07", priceType: "到场价" },
    { name: "c30-e", title: "C30 抗渗混凝土", category: "混凝土", unit: "m³", spec: "C30 P6 抗渗", price: 1580, source: "市场询价", region: "成都", priceDate: "2026-08", priceType: "到场价" },
  ],
};

const EMPTY_VIEW: CostComposeView = {
  description: "预制桩静压施工",
  unit: "台班",
  band: null,
  recommendedPrice: 0,
  reason: "",
  llmUsed: false,
  evidence: [],
};

describe("ComposeModal AI 组价", () => {
  beforeEach(() => {
    composeSpy.mockReset();
  });

  it("band=null 时空态提示，composeSpy 收到描述与单位", async () => {
    composeSpy.mockResolvedValue(EMPTY_VIEW);
    renderModal({ initialDesc: "预制桩静压施工", initialUnit: "台班" });
    compose();
    expect(await screen.findByText("成本库暂无相似条目,请先录入或导入成本数据")).toBeTruthy();
    expect(composeSpy).toHaveBeenCalledWith("预制桩静压施工", "台班");
    // band=null 无可应用结果，底部无「应用」按钮。
    expect(screen.queryByText("应用")).toBeNull();
  });

  it("价格带卡片：样本数/置信度/P25/P50/P75/均值/离散度/离群数 + 推荐价与理由", async () => {
    composeSpy.mockResolvedValue(VIEW);
    renderModal();
    compose();
    await screen.findByText("价格带推荐");
    // 样本数 + 置信度徽标（高 → 绿色）。
    expect(screen.getByText("6 个样本")).toBeTruthy();
    const conf = screen.getByText("高");
    expect(conf).toBeTruthy();
    expect(conf.className).toContain("text-emerald-400");
    // 分位数/均值/离散度/离群数。
    expect(screen.getByText("¥1,000")).toBeTruthy(); // P25
    expect(screen.getByText("¥1,400")).toBeTruthy(); // P75
    expect(screen.getByText("¥1,418.33")).toBeTruthy(); // 均值
    expect(screen.getByText("33.3%")).toBeTruthy(); // 离散度
    expect(screen.getByText("1 个")).toBeTruthy(); // 离群数
    // P50 与推荐价同为 1,200（中位数推荐）。
    expect(screen.getAllByText("¥1,200").length).toBeGreaterThanOrEqual(2);
    // 推荐理由文案。
    expect(screen.getByText(/综合 6 个相似样本/)).toBeTruthy();
  });

  it("证据链表渲染溯源字段，离群样本标注「离群」", async () => {
    composeSpy.mockResolvedValue(VIEW);
    renderModal();
    compose();
    await screen.findByText("证据链（6 条）");
    // 表头（标题/规格/单价/单位/来源/地区/期数/口径）。
    expect(screen.getByText("标题")).toBeTruthy();
    expect(screen.getByText("规格")).toBeTruthy();
    expect(screen.getByText("来源")).toBeTruthy();
    expect(screen.getByText("地区")).toBeTruthy();
    expect(screen.getByText("期数")).toBeTruthy();
    expect(screen.getByText("口径")).toBeTruthy();
    // 样本行渲染（来源/规格/价格）。
    expect(screen.getByText("重庆造价信息网")).toBeTruthy();
    expect(screen.getByText("四川造价信息网")).toBeTruthy();
    expect(screen.getByText("C30 泵送")).toBeTruthy(); // 规格
    expect(screen.getByText("¥1,050")).toBeTruthy();
    expect(screen.getByText("贵阳")).toBeTruthy();
    // 离群样本（2300 > P75+1.5IQR）唯一标注「离群」且为红色。
    expect(screen.getByText("¥2,300")).toBeTruthy();
    const outlier = screen.getByText("离群");
    expect(outlier.className).toContain("text-red-400");
    expect(screen.getAllByText("离群")).toHaveLength(1);
  });

  it("人材机拆解渲染，改含量金额自动重算，可增删行", async () => {
    composeSpy.mockResolvedValue(VIEW);
    renderModal();
    compose();
    await screen.findByText("人材机拆解（2 行）");
    // 初始行渲染（含量 0.6 × 单价 250 = 150）。
    expect(screen.getByDisplayValue("混凝土浇筑人工")).toBeTruthy();
    expect(screen.getByDisplayValue("0.6")).toBeTruthy();
    expect(screen.getByText("¥150")).toBeTruthy();
    // 备注与 LLM 正常（无规则降级）。
    expect(screen.getByText(/损耗系数已计入含量/)).toBeTruthy();
    expect(screen.queryByText("规则降级")).toBeNull();
    // 改含量 → 金额自动重算（3 × 250 = 750）。
    fireEvent.change(screen.getAllByPlaceholderText("含量")[0], { target: { value: "3" } });
    expect(screen.getByText("¥750")).toBeTruthy();
    // 加行 → 3 行；删行 → 回到 2 行。
    fireEvent.click(screen.getByText("＋ 添加组成行"));
    expect(screen.getAllByPlaceholderText("含量")).toHaveLength(3);
    fireEvent.click(screen.getAllByTitle(/删除第/)[2]);
    expect(screen.getAllByPlaceholderText("含量")).toHaveLength(2);
  });

  it("「应用」回传 desc/unit/推荐价/组件/证据链并关闭", async () => {
    composeSpy.mockResolvedValue(VIEW);
    const { onApply, onClose } = renderModal();
    compose();
    await screen.findByText("价格带推荐");
    fireEvent.click(screen.getByText("应用"));
    expect(onApply).toHaveBeenCalledTimes(1);
    const r = onApply.mock.calls[0][0] as ApplyResult;
    expect(r.desc).toBe("C30 混凝土浇筑");
    expect(r.unit).toBe("m³");
    expect(r.price).toBe(1200);
    expect(r.components).toHaveLength(2);
    expect(r.components[0].title).toBe("混凝土浇筑人工");
    expect(r.evidence).toHaveLength(2);
    expect(r.evidence[0].source).toBe("四川造价信息网");
    expect(onClose).toHaveBeenCalled();
  });

  it("组价失败：toast warn + 弹窗内持久错误展示（可修改重试）", async () => {
    composeSpy.mockRejectedValue(new Error("组价服务不可用"));
    renderModal();
    compose();
    // 持久错误框（role=alert；warn toast 无 role，避免双匹配）。
    const alert = await screen.findByRole("alert");
    expect(alert.textContent).toContain("组价失败");
    expect(alert.textContent).toContain("组价服务不可用");
    // toast 也弹出同文案（半角冒号，与错误框全角冒号区分）。
    expect(screen.getByText("组价失败:组价服务不可用")).toBeTruthy();
    // 输入表单仍在，可修改后重试。
    expect(screen.getByPlaceholderText(/清单描述/)).toBeTruthy();
    expect(screen.getByText("开始组价")).toBeTruthy();
  });

  it("打开时预填 initialDesc / initialUnit", () => {
    composeSpy.mockResolvedValue(VIEW);
    renderModal({ initialDesc: "预制桩静压施工", initialUnit: "台班" });
    expect(screen.getByDisplayValue("预制桩静压施工")).toBeTruthy();
    expect(screen.getByDisplayValue("台班")).toBeTruthy();
    // 描述为空时按钮禁用（预填后可用）。
    const btn = screen.getByText("开始组价") as HTMLButtonElement;
    expect(btn.disabled).toBe(false);
  });
});
