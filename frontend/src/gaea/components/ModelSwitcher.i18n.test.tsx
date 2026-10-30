// v4.57 i18n 收尾冒烟 + v4.126 刀1/刀2 本地模型使用：ModelSwitcher 三语字典接线、
// 按引擎分组（本地置前）、运行徽标、预估按目标模型传参、云端直切。
// v4.530 用户拍板「切换模型不要弹窗」：冷启动不再弹确认框——直接切换 + 非阻塞
// message.info 告知预计等待。jsdom 下用 spy 捕获 message.info 配置断言字典文案
// （zh 逐字 + en 抽查），并断言 onPick 立即生效（不等用户确认）。
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, cleanup } from "@testing-library/react";
import { message } from "antd";
import { ModelSwitcher } from "./ModelSwitcher";
import { LocaleProvider } from "../lib/i18n";

const mocks = vi.hoisted(() => ({
  Models: vi.fn(),
  ModelSwitchEstimate: vi.fn(),
}));

vi.mock("../lib/bridge", () => ({
  app: mocks,
  onEvent: vi.fn(() => () => {}),
  onReady: vi.fn(() => () => {}),
}));

// P4-H7：en 字典按需异步加载——en 用例须先 await loadLocale('en') 再渲染，
// 否则 DICTS.en 未就绪、译文回退 zh（断言英文必失败）。
import { loadLocale } from "../lib/i18n";
const renderT = async (ui: React.ReactNode, lang: "zh" | "en" = "zh") => {
  if (lang === "en") await loadLocale("en");
  localStorage.setItem("gaea-lang", lang);
  return render(<LocaleProvider>{ui}</LocaleProvider>);
};

// message.info 捕获：beforeEach 统一 spy，用例直接读 noticeCalls。
type NoticeCall = { content: string };
let noticeCalls: NoticeCall[] = [];

describe("ModelSwitcher i18n 冒烟", () => {
  beforeEach(() => {
    mocks.Models.mockReset();
    mocks.ModelSwitchEstimate.mockReset();
    noticeCalls = [];
    vi.spyOn(message, "info").mockImplementation(((content?: any) => {
      noticeCalls.push({ content: String(content) });
      return undefined as never;
    }) as never);
  });
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it("按钮 title 走既有 status.switchModel 键；空模型列表回退 status.noModels", async () => {
    mocks.Models.mockResolvedValue([]);
    renderT(<ModelSwitcher label="herdsman/qwen3-32b" onPick={() => {}} />);
    expect(screen.getByTitle("切换模型")).toBeTruthy();
    fireEvent.click(screen.getByTitle("切换模型"));
    expect(await screen.findByText("未配置任何模型")).toBeTruthy();
  });

  it("herdsman 冷启动：预估按目标模型传参，不弹窗直接切换 + 非阻塞告知等待（zh），onPick 立即生效", async () => {
    mocks.Models.mockResolvedValue([{ ref: "herdsman/qwen3-32b", model: "qwen3-32b", local: true, current: true }]);
    mocks.ModelSwitchEstimate.mockResolvedValue({ status: "cold", waitSeconds: 20, note: "" });
    const onPick = vi.fn();
    renderT(<ModelSwitcher label="" onPick={onPick} />);
    fireEvent.click(screen.getByTitle("切换模型"));
    fireEvent.click(await screen.findByText("qwen3-32b"));
    // 不再弹确认框：立即切换。
    await vi.waitFor(() => expect(onPick).toHaveBeenCalledWith("herdsman/qwen3-32b"));
    // 刀2：预估带目标模型（不再只按引擎默认模型口径）。
    expect(mocks.ModelSwitchEstimate).toHaveBeenCalledWith("herdsman", "qwen3-32b");
    // 非阻塞 toast 告知冷启动等待，文案走字典。
    await vi.waitFor(() => expect(noticeCalls).toHaveLength(1));
    expect(noticeCalls[0].content).toBe("已切换 herdsman/qwen3-32b（需冷启动），预计等待 20 秒。");
  });

  it("en 抽查：冷启动提示渲染英文键值，切换照常立即生效", async () => {
    mocks.Models.mockResolvedValue([{ ref: "herdsman/qwen3-32b", model: "qwen3-32b", local: true, current: false }]);
    mocks.ModelSwitchEstimate.mockResolvedValue({ status: "cold", waitSeconds: 0, note: "" });
    const onPick = vi.fn();
    await renderT(<ModelSwitcher label="" onPick={onPick} />, "en");
    fireEvent.click(screen.getByTitle("Switch model"));
    fireEvent.click(await screen.findByText("qwen3-32b"));
    await vi.waitFor(() => expect(onPick).toHaveBeenCalledWith("herdsman/qwen3-32b"));
    await vi.waitFor(() => expect(noticeCalls).toHaveLength(1));
    expect(noticeCalls[0].content).toBe("Switched to herdsman/qwen3-32b (cold start required); this will take a moment.");
  });

  it("云端模型直接切换（无提示、不发预估）", async () => {
    mocks.Models.mockResolvedValue([{ ref: "xai/grok-5", model: "grok-5", current: false }]);
    const onPick = vi.fn();
    renderT(<ModelSwitcher label="" onPick={onPick} />);
    fireEvent.click(screen.getByTitle("切换模型"));
    fireEvent.click(await screen.findByText("grok-5"));
    await vi.waitFor(() => expect(onPick).toHaveBeenCalledWith("xai/grok-5"));
    expect(mocks.ModelSwitchEstimate).not.toHaveBeenCalled();
    expect(noticeCalls).toHaveLength(0);
  });

  it("刀1 分组：本地引擎组置前且带「本地」标，组内条目按引擎聚拢；运行中徽标", async () => {
    mocks.Models.mockResolvedValue([
      { ref: "xai/grok-4.20", provider: "xai", model: "grok-4.20", current: false },
      { ref: "ollama/q3:14b", provider: "ollama", model: "q3:14b", current: false, local: true },
      { ref: "ollama/q3:8b", provider: "ollama", model: "q3:8b", current: true, local: true, status: "running" },
      { ref: "deepseek/(默认)", provider: "deepseek", model: "(默认)", current: false },
    ]);
    const onPick = vi.fn();
    renderT(<ModelSwitcher label="ollama/q3:8b" onPick={onPick} />);
    fireEvent.click(screen.getByTitle("切换模型"));
    // 组名 caption：provider id + 「本地」标。
    expect(await screen.findByText("ollama")).toBeTruthy();
    expect(screen.getByText("· 本地")).toBeTruthy();
    expect(screen.getByText("xai")).toBeTruthy();
    // 运行中徽标只标 running 条目。
    expect(screen.getByText("运行中")).toBeTruthy();
    // 序断言：ollama 组条目先于 xai 条目出现（本地置前）。
    const items = [...screen.getAllByRole("option")].map((el) => el.textContent);
    expect(items.findIndex((s) => s?.includes("q3:8b"))).toBeLessThan(items.findIndex((s) => s?.includes("grok-4.20")));
    // 点选未加载的本地模型：走预估（hot 即静默直切），展示名/模型名文本可点。
    mocks.ModelSwitchEstimate.mockResolvedValue({ status: "hot", waitSeconds: 1, note: "模型已加载" });
    fireEvent.click(screen.getByText("q3:14b"));
    await vi.waitFor(() => expect(onPick).toHaveBeenCalledWith("ollama/q3:14b"));
    expect(mocks.ModelSwitchEstimate).toHaveBeenCalledWith("ollama", "q3:14b");
  });

  it("展示名 label 优先渲染，title 保留模型 id", async () => {
    mocks.Models.mockResolvedValue([
      { ref: "strata/qwen3.8-flash:tiny", provider: "strata", model: "qwen3.8-flash:tiny", label: "Tinyrick Q6", local: true, status: "stopped", current: false },
    ]);
    renderT(<ModelSwitcher label="" onPick={() => {}} />);
    fireEvent.click(screen.getByTitle("切换模型"));
    expect(await screen.findByText("Tinyrick Q6")).toBeTruthy();
    expect(screen.getByTitle("Tinyrick Q6 (qwen3.8-flash:tiny)")).toBeTruthy();
    expect(screen.queryByText("qwen3.8-flash:tiny")).toBeNull();
  });
});
