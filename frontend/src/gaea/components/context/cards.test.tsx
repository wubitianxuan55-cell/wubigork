// cards.test.tsx — dsh-context 头部仪表卡（cards.tsx）定向 jsdom 测试。
// 被测组件为纯展示（无 bridge/i18n 依赖），无需 mock 外部模块。
import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { ContextRequestRecord, ContextStats, ContextTiming } from "../../lib/types";
import { fmtDuration, SessionInfoCard, StatsCard, summarizeTokens, SummaryBar, TimingCard, TokenCard } from "./cards";
import type { ReactElement } from "react";
import { LocaleProvider } from "../../lib/i18n";

const renderZ = (ui: ReactElement) => {
  localStorage.setItem("gaea-lang", "zh");
  return render(<LocaleProvider>{ui}</LocaleProvider>);
};

const REQ = (seq: number, hit: number, miss: number, out: number): ContextRequestRecord => ({
  seq,
  ts: 1750000000 + seq,
  turn: 1,
  step: seq,
  category: { system: 0, tools: 0, user: 0, inject: 0, assistant: 0, tool: 0 },
  cacheHitTokens: hit,
  cacheMissTokens: miss,
  outputTokens: out,
});

// ─── fmtDuration / summarizeTokens 纯函数 ───────────────────────

describe("fmtDuration 时长格式化", () => {
  it("秒级与分钟级两种形态", () => {
    expect(fmtDuration(0)).toBe("0 秒");
    expect(fmtDuration(30000)).toBe("30 秒");
    expect(fmtDuration(65000)).toBe("1 分 5 秒");
    expect(fmtDuration(60000)).toBe("1 分 0 秒");
    expect(fmtDuration(3661000)).toBe("61 分 1 秒");
  });
});

describe("summarizeTokens 汇总", () => {
  it("累计 hit/miss/out，空请求归零", () => {
    expect(summarizeTokens([REQ(1, 300000, 5000, 1000), REQ(2, 100000, 5000, 2000)])).toEqual({
      hit: 400000, miss: 10000, out: 3000,
    });
    expect(summarizeTokens([])).toEqual({ hit: 0, miss: 0, out: 0 });
  });
});

// ─── StatsCard 上下文统计 ────────────────────────────────────────

describe("StatsCard", () => {
  const stats: ContextStats = {
    turns: 2, steps: 60, injects: 6, compacts: 1, prunes: 0, toolCalls: 279, images: 3, costEstimate: 3.83,
  };

  it("渲染 8 格 label 与数值（费用两位小数）", () => {
    renderZ(<StatsCard stats={stats} />);
    for (const label of ["轮次", "步数", "工具调用", "图片", "预估费用", "注入", "压缩", "剪枝"]) {
      expect(screen.getByText(label)).toBeTruthy();
    }
    expect(screen.getByText("2")).toBeTruthy();
    expect(screen.getByText("60")).toBeTruthy();
    expect(screen.getByText("279")).toBeTruthy();
    expect(screen.getByText("¥3.83")).toBeTruthy();
  });

  it("costEstimate 缺失时显示「—」不伪造", () => {
    renderZ(<StatsCard stats={{ ...stats, costEstimate: undefined }} />);
    expect(screen.getByText("—")).toBeTruthy();
  });
});

// ─── TokenCard ──────────────────────────────────────────────────

describe("TokenCard", () => {
  const requests = [REQ(1, 300000, 5000, 1000), REQ(2, 100000, 5000, 2000)];

  it("环形中心显示缓存命中率，分解行显示三分类值与占比", () => {
    const { container } = renderZ(<TokenCard requests={requests} />);
    // 命中率 = hit/(hit+miss) = 400000/410000 → 97.56%（v4.69 环心两位小数）
    expect(screen.getByText("97.56%")).toBeTruthy();
    expect(screen.getByText("缓存输入")).toBeTruthy();
    expect(screen.getByText("400k")).toBeTruthy();
    expect(screen.getByText("10k")).toBeTruthy();
    expect(screen.getByText("3k")).toBeTruthy();
    // 占比 = 各分类 / 总量(413000)：97% / 2% / 1%
    expect(screen.getByText("97%")).toBeTruthy();
    expect(screen.getByText("2%")).toBeTruthy();
    expect(screen.getByText("1%")).toBeTruthy();
    // 环形：track + 3 段；绿段（缓存输入）dash 长度占总长 = 400000/413000
    const dashed = Array.from(container.querySelectorAll<SVGCircleElement>("circle[stroke-dasharray]"));
    expect(dashed.length).toBe(3);
    const lens = dashed.map((c) => Number(c.getAttribute("stroke-dasharray")!.split(" ")[0]));
    const green = container.querySelector('circle[stroke="#22c55e"]')!;
    expect(green).toBeTruthy();
    const greenLen = Number(green.getAttribute("stroke-dasharray")!.split(" ")[0]);
    expect(greenLen / lens.reduce((a, b) => a + b, 0)).toBeCloseTo(400000 / 413000, 3);
  });

  it("空请求：中心与数值空态「—」，占比 0%", () => {
    const { container } = renderZ(<TokenCard requests={[]} />);
    expect(screen.getAllByText("—").length).toBe(4); // 中心 + 3 行数值
    expect(screen.getAllByText("0%").length).toBe(3);
    // 只有 track，无数据段
    expect(container.querySelectorAll("circle").length).toBe(1);
  });
});

// ─── TimingCard ─────────────────────────────────────────────────

describe("TimingCard", () => {
  const timing: ContextTiming = {
    wallMs: 65000,
    ttftMs: 12000,
    genMs: 30000,
    calls: 5,
    toolsMs: 15000,
    toolCalls: 8,
    tools: [
      { name: "read_file", calls: 3, ms: 9000 },
      { name: "bash", calls: 5, ms: 6000 },
      { name: "grep", calls: 2, ms: 12000 },
      { name: "sed", calls: 1, ms: 1000 },
    ],
  };

  it("环形中心为活跃时长，四分解行含时长/次数/占比，其他开销为推导值", () => {
    renderZ(<TimingCard timing={timing} />);
    expect(screen.getByText("1 分 5 秒")).toBeTruthy(); // 中心 wallMs
    // 行级断言（各行的「12 秒」「5 次」可能与工具排行重复，用行作用域消歧）
    const row = (label: string) => within(screen.getByText(label).closest("div")!);
    expect(row("模型等待").getByText("12 秒")).toBeTruthy();
    expect(row("模型等待").getByText("5 次")).toBeTruthy();
    expect(row("模型等待").getByText("18%")).toBeTruthy(); // 12000/65000
    expect(row("模型生成").getByText("30 秒")).toBeTruthy();
    expect(row("模型生成").getByText("46%")).toBeTruthy(); // 30000/65000
    expect(row("工具执行").getByText("15 秒")).toBeTruthy();
    expect(row("工具执行").getByText("8 次")).toBeTruthy();
    expect(row("工具执行").getByText("23%")).toBeTruthy(); // 15000/65000
    // 其他开销 = 65000-12000-30000-15000 = 8 秒，无计数显示「—」
    expect(row("其他开销").getByText("8 秒")).toBeTruthy();
    expect(row("其他开销").getByText("12%")).toBeTruthy(); // 8000/65000
    expect(row("其他开销").getByText("—")).toBeTruthy();
  });

  it("tools 排行取时长 top3 迷你行", () => {
    renderZ(<TimingCard timing={timing} />);
    const tools = within(screen.getByTestId("timing-tools"));
    expect(tools.getByText("grep")).toBeTruthy(); // 12000ms 居首
    expect(tools.getByText("read_file")).toBeTruthy();
    expect(tools.getByText("bash")).toBeTruthy();
    expect(tools.queryByText("sed")).toBeNull(); // 第 4 名被截断
    expect(tools.getByText("2 次")).toBeTruthy();
  });

  it("timing undefined：骨架空态全「—」，不伪造数值", () => {
    const { container } = renderZ(<TimingCard />);
    expect(screen.getAllByText("—").length).toBeGreaterThanOrEqual(10); // 中心 + 4 行 ×(时长/次数/占比)
    expect(screen.queryByText("1 分 5 秒")).toBeNull();
    expect(screen.queryByText("30 秒")).toBeNull();
    expect(screen.queryByText(/次/)).toBeNull();
    // 仅剩 track 圆环（骨架），无数据段
    expect(container.querySelectorAll("circle").length).toBe(1);
  });
});

// ─── SessionInfoCard ────────────────────────────────────────────

describe("SessionInfoCard", () => {
  it("行式键值直显五个字段", () => {
    renderZ(
      <SessionInfoCard sessionName="重构会话" space="办公空间" model="glm-5" window={128000} requests={42} />,
    );
    expect(screen.getByText("会话信息")).toBeTruthy();
    expect(screen.getByText("会话名")).toBeTruthy();
    expect(screen.getByText("重构会话")).toBeTruthy();
    expect(screen.getByText("办公空间")).toBeTruthy();
    expect(screen.getByText("glm-5")).toBeTruthy();
    expect(screen.getByText("128000")).toBeTruthy();
    expect(screen.getByText("42")).toBeTruthy();
  });
});

// ─── SummaryBar ─────────────────────────────────────────────────

describe("SummaryBar", () => {
  const requests = [REQ(1, 100, 100, 100), REQ(2, 100, 100, 100), REQ(3, 100, 100, 100)];

  it("单行拼装：会话徽标 + 水位 + 请求数 + 累计费用", () => {
    renderZ(<SummaryBar sessionName="重构会话" used={241800} window={1000000} requests={requests} costEstimate={3.83} />);
    expect(screen.getByText("重构会话")).toBeTruthy();
    expect(screen.getByText("241.8k / 1.0M · 24%")).toBeTruthy();
    expect(screen.getByText("3 次请求")).toBeTruthy();
    expect(screen.getByText("累计费用 ¥3.83")).toBeTruthy();
  });

  it("六分类图例 + 空闲窗口灰点；costEstimate 缺失显示「累计 —」", () => {
    renderZ(<SummaryBar sessionName="s" used={0} window={0} requests={[]} />);
    for (const label of ["系统提示词", "工具定义", "用户消息", "注入内容", "助手消息", "工具结果", "空闲窗口"]) {
      expect(screen.getByText(label)).toBeTruthy();
    }
    expect(screen.getByText("0 / 0 · 0%")).toBeTruthy();
    expect(screen.getByText("0 次请求")).toBeTruthy();
    expect(screen.getByText("累计费用 —")).toBeTruthy();
  });
});
