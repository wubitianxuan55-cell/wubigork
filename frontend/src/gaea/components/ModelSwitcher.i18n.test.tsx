// v4.57 i18n 收尾冒烟：ModelSwitcher 本地模型切换确认弹层的三语字典接线。
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

const renderT = (ui: React.ReactNode, lang: "zh" | "en" = "zh") => {
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

  it("herdsman 冷启动：确认弹层文案走字典（zh），确认后回调 onPick", async () => {
    mocks.Models.mockResolvedValue([{ ref: "herdsman/qwen3-32b", current: true }]);
    mocks.ModelSwitchEstimate.mockResolvedValue({ status: "cold", waitSeconds: 20, note: "" });
    const onPick = vi.fn();
    renderT(<ModelSwitcher label="" onPick={onPick} />);
    fireEvent.click(screen.getByTitle("切换模型"));
    fireEvent.click(await screen.findByText("herdsman/qwen3-32b"));
    await vi.waitFor(() => expect(confirmCalls).toHaveLength(1));
    const config = confirmCalls[0];
    expect(config.title).toBe("切换本地模型");
    expect(config.content).toBe("herdsman/qwen3-32b 未在运行（需冷启动）。\n预计等待 20 秒，确定继续切换吗？");
    expect(config.okText).toBe("继续切换");
    expect(config.cancelText).toBe("取消");
    config.onOk?.();
    await vi.waitFor(() => expect(onPick).toHaveBeenCalledWith("herdsman/qwen3-32b"));
  });

  it("en 抽查：确认弹层标题/按钮渲染英文键值；取消则不切换", async () => {
    mocks.Models.mockResolvedValue([{ ref: "herdsman/qwen3-32b", current: false }]);
    mocks.ModelSwitchEstimate.mockResolvedValue({ status: "cold", waitSeconds: 0, note: "" });
    const onPick = vi.fn();
    renderT(<ModelSwitcher label="" onPick={onPick} />, "en");
    fireEvent.click(screen.getByTitle("Switch model"));
    fireEvent.click(await screen.findByText("herdsman/qwen3-32b"));
    await vi.waitFor(() => expect(confirmCalls).toHaveLength(1));
    const config = confirmCalls[0];
    expect(config.title).toBe("Switch local model");
    expect(config.okText).toBe("Switch anyway");
    config.onCancel?.();
    await vi.waitFor(() => expect(onPick).not.toHaveBeenCalled());
  });

  it("非 herdsman 模型直接切换（无确认弹层）", async () => {
    mocks.Models.mockResolvedValue([{ ref: "xai/grok-5", current: false }]);
    const onPick = vi.fn();
    renderT(<ModelSwitcher label="" onPick={onPick} />);
    fireEvent.click(screen.getByTitle("切换模型"));
    fireEvent.click(await screen.findByText("xai/grok-5"));
    await vi.waitFor(() => expect(onPick).toHaveBeenCalledWith("xai/grok-5"));
    expect(confirmCalls).toHaveLength(0);
  });
});
