import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import MorningBriefCard from "./MorningBriefCard";
import { LocaleProvider } from "../lib/i18n";

const { briefMock } = vi.hoisted(() => ({ briefMock: vi.fn() }));

vi.mock("../lib/bridge", () => ({
  app: { MemoryMorningBrief: briefMock },
}));

function wrap(ui: React.ReactNode) {
  return <LocaleProvider>{ui}</LocaleProvider>;
}

// 组装一条合法晨报 JSON 串（对齐 Go 侧 GaeaMemoryMorningBrief 契约）。
function briefJSON(overrides: Record<string, unknown> = {}) {
  return JSON.stringify({
    items: [
      { name: "fact-a", description: "第一条工作事实摘要", kind: "semantic", category: "project", updatedAt: Date.now() },
      { name: "fact-b", description: "第二条用户事实摘要", kind: "semantic", category: "user", updatedAt: Date.now() },
    ],
    rules: ["常驻规则：回复先说结论"],
    dreamed24h: 3,
    generatedAt: Date.now(),
    ...overrides,
  });
}

describe("MorningBriefCard 今日晨报（做梦 2.0 主动预取）", () => {
  beforeEach(() => {
    briefMock.mockReset();
  });

  it("正常渲染：标题 + items 列表 + rules 区 + 近 24h 沉淀计数", async () => {
    briefMock.mockResolvedValue(briefJSON());
    render(wrap(<MorningBriefCard />));
    // 标题（i18n home.morningBrief.title；测试环境默认英文，三语皆可）
    expect(await screen.findByText(/今日晨报|Morning Brief|今日晨報/)).toBeTruthy();
    // items 名称 + 描述
    expect(screen.getByText("fact-a")).toBeTruthy();
    expect(screen.getByText("第一条工作事实摘要")).toBeTruthy();
    expect(screen.getByText("fact-b")).toBeTruthy();
    // rules 区（有才显示）：标题（三语）+ 规则内容
    expect(screen.getAllByText(/常驻规则|Standing Rules|常駐規則/).length).toBeGreaterThan(0);
    expect(screen.getByText("常驻规则：回复先说结论")).toBeTruthy();
    // 沉淀计数
    expect(screen.getByText(/近 24h 沉淀 3 条记忆|3 memories settled in 24h|近 24h 沉澱 3 條記憶/)).toBeTruthy();
  });

  it("空 items 静默隐藏：不渲染任何内容", async () => {
    briefMock.mockResolvedValue(briefJSON({ items: [] }));
    render(wrap(<MorningBriefCard />));
    await screen.findByText("今日晨报").catch(() => undefined); // 等待首次渲染落地
    // items 为空 → 组件返回 null，标题与计数都不应出现
    expect(screen.queryByText("今日晨报")).toBeNull();
    expect(screen.queryByText(/近 24h 沉淀/)).toBeNull();
  });

  it("解析失败静默隐藏：非法 JSON 不渲染、不弹 toast", async () => {
    briefMock.mockResolvedValue("not-json{{{");
    render(wrap(<MorningBriefCard />));
    // 静默降级：无标题、无列表、无错误文案
    await new Promise((r) => setTimeout(r, 20));
    expect(screen.queryByText("今日晨报")).toBeNull();
    expect(screen.queryByText(/fact-a/)).toBeNull();
  });

  it("绑定拒绝（rejected promise）同样静默隐藏", async () => {
    briefMock.mockRejectedValue(new Error("bind failed"));
    render(wrap(<MorningBriefCard />));
    await new Promise((r) => setTimeout(r, 20));
    expect(screen.queryByText("今日晨报")).toBeNull();
  });
});
