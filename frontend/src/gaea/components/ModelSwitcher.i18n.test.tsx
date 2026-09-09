// v4.57 i18n 收尾冒烟 + v4.126 刀1/刀2 本地模型使用：ModelSwitcher 三语字典接线、
// 按引擎分组（本地置前）、运行徽标、预估按目标模型传参、云端直切。
// antd 静态 Modal.confirm 在 jsdom 下不渲染 → 用 spy 捕获 confirm 配置断言
// 字典文案（zh 逐字 + en 抽查），并驱动 onOk/onCancel 钉回调路径。
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, cleanup } from "@testing-library/react";
import { Modal } from "antd";
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

// Modal.confirm 配置捕获：beforeEach 统一 spy，用例直接读 confirmCalls；
// afterEach restore，避免用例间 mock 串扰（antd 静态弹层在 jsdom 不渲染）。
type ConfirmConfig = { title: string; content: string; okText: string; cancelText: string; onOk?: () => void; onCancel?: () => void };
let confirmCalls: ConfirmConfig[] = [];

describe("ModelSwitcher i18n 冒烟", () => {
  beforeEach(() => {
    mocks.Models.mockReset();
    mocks.ModelSwitchEstimate.mockReset();
    confirmCalls = [];
    vi.spyOn(Modal, "confirm").mockImplementation(((config: ConfirmConfig) => {
      confirmCalls.push(config);
      return { destroy: () => {}, update: () => {} };
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

  it("herdsman 冷启动：预估按目标模型传参，确认弹层文案走字典（zh），确认后回调 onPick", async () => {
    mocks.Models.mockResolvedValue([{ ref: "herdsman/qwen3-32b", model: "qwen3-32b", local: true, current: true }]);
    mocks.ModelSwitchEstimate.mockResolvedValue({ status: "cold", waitSeconds: 20, note: "" });
    const onPick = vi.fn();
    renderT(<ModelSwitcher label="" onPick={onPick} />);
    fireEvent.click(screen.getByTitle("切换模型"));
    fireEvent.click(await screen.findByText("qwen3-32b"));
    await vi.waitFor(() => expect(confirmCalls).toHaveLength(1));
    // 刀2：预估带目标模型（不再只按引擎默认模型口径）。
    expect(mocks.ModelSwitchEstimate).toHaveBeenCalledWith("herdsman", "qwen3-32b");
    const config = confirmCalls[0];
    expect(config.title).toBe("切换本地模型");
    expect(config.content).toBe("herdsman/qwen3-32b 未在运行（需冷启动）。\n预计等待 20 秒，确定继续切换吗？");
    expect(config.okText).toBe("继续切换");
    expect(config.cancelText).toBe("取消");
    config.onOk?.();
    await vi.waitFor(() => expect(onPick).toHaveBeenCalledWith("herdsman/qwen3-32b"));
  });

  it("en 抽查：确认弹层标题/按钮渲染英文键值；取消则不切换", async () => {
    mocks.Models.mockResolvedValue([{ ref: "herdsman/qwen3-32b", model: "qwen3-32b", local: true, current: false }]);
    mocks.ModelSwitchEstimate.mockResolvedValue({ status: "cold", waitSeconds: 0, note: "" });
    const onPick = vi.fn();
    await renderT(<ModelSwitcher label="" onPick={onPick} />, "en");
    fireEvent.click(screen.getByTitle("Switch model"));
    fireEvent.click(await screen.findByText("qwen3-32b"));
    await vi.waitFor(() => expect(confirmCalls).toHaveLength(1));
    const config = confirmCalls[0];
    expect(config.title).toBe("Switch local model");
    expect(config.okText).toBe("Switch anyway");
    config.onCancel?.();
    await vi.waitFor(() => expect(onPick).not.toHaveBeenCalled());
  });

  it("云端模型直接切换（无确认弹层、不发预估）", async () => {
    mocks.Models.mockResolvedValue([{ ref: "xai/grok-5", model: "grok-5", current: false }]);
    const onPick = vi.fn();
    renderT(<ModelSwitcher label="" onPick={onPick} />);
    fireEvent.click(screen.getByTitle("切换模型"));
    fireEvent.click(await screen.findByText("grok-5"));
    await vi.waitFor(() => expect(onPick).toHaveBeenCalledWith("xai/grok-5"));
    expect(mocks.ModelSwitchEstimate).not.toHaveBeenCalled();
    expect(confirmCalls).toHaveLength(0);
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
    // 点选未加载的本地模型：走预估确认，展示名/模型名文本可点。
    mocks.ModelSwitchEstimate.mockResolvedValue({ status: "hot", waitSeconds: 1, note: "模型已加载" });
    fireEvent.click(screen.getByText("q3:14b"));
    await vi.waitFor(() => expect(onPick).toHaveBeenCalledWith("ollama/q3:14b"));
    expect(mocks.ModelSwitchEstimate).toHaveBeenCalledWith("ollama", "q3:14b");
  });

  it("展示名 label 优先渲染，title 保留模型 id", async () => {
    mocks.Models.mockResolvedValue([
      { ref: "modelhub/ollama-manifest:tiny", provider: "modelhub", model: "ollama-manifest:tiny", label: "Tinyrick Q6", local: true, status: "stopped", current: false },
    ]);
    renderT(<ModelSwitcher label="" onPick={() => {}} />);
    fireEvent.click(screen.getByTitle("切换模型"));
    expect(await screen.findByText("Tinyrick Q6")).toBeTruthy();
    expect(screen.getByTitle("Tinyrick Q6 (ollama-manifest:tiny)")).toBeTruthy();
    expect(screen.queryByText("ollama-manifest:tiny")).toBeNull();
  });
});
