import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { MemoryView } from "../lib/types";
import { LocaleProvider } from "../lib/i18n";
import { ToastProvider } from "./Toast";

vi.setConfig({ testTimeout: 15_000 }); // 全量并行下 import/setup 变慢,既有 2.7s 用例会超默认 5s(ContextView 40s 同配方)

const mocks = vi.hoisted(() => ({
  morningPreload: vi.fn(),
  setMorningPreload: vi.fn(),
  projectBrief: vi.fn(),
  setProjectBrief: vi.fn(),
  // v4.377 建议制
  setDreamMode: vi.fn(),
  dismissSuggestion: vi.fn(),
  dreamPurgePreview: vi.fn(),
  dreamPurge: vi.fn(),
}));

vi.mock("../lib/bridge", () => ({
  app: new Proxy({}, {
    get(_t, prop) {
      if (prop === "MorningPreload") return mocks.morningPreload;
      if (prop === "SetMorningPreload") return mocks.setMorningPreload;
      if (prop === "MemoryBrief") return mocks.projectBrief;
      if (prop === "SetMemoryBrief") return mocks.setProjectBrief;
      if (prop === "SetDreamMode") return mocks.setDreamMode;
      if (prop === "DismissMemorySuggestion") return mocks.dismissSuggestion;
      if (prop === "DreamPurgePreview") return mocks.dreamPurgePreview;
      if (prop === "DreamPurge") return mocks.dreamPurge;
      return vi.fn();
    },
  }),
  openExternal: vi.fn(),
}));

async function renderPanel() {
  const noop = async () => {};
  const { MemoryPanel } = await import("./MemoryPanel");
  return render(
    <LocaleProvider>
      <ToastProvider>
        <MemoryPanel
          view={null as unknown as MemoryView}
          onRemember={noop}
          onForget={noop}
          onSaveDoc={noop}
          onSaveFact={noop}
          onChangeType={noop}
          onAcceptMemorySuggestion={noop}
          onAcceptSkillSuggestion={noop}
          onAcceptMergeSuggestion={noop}
          onRefreshSuggestions={async () => null}
        />
      </ToastProvider>
    </LocaleProvider>,
  );
}

// v4.377 建议制测试：注入建议返回值（onRefreshSuggestions）。
async function renderPanelWithSuggestions(refresh: () => Promise<unknown>) {
  const noop = async () => {};
  const { MemoryPanel } = await import("./MemoryPanel");
  return render(
    <LocaleProvider>
      <ToastProvider>
        <MemoryPanel
          view={null as unknown as MemoryView}
          onRemember={noop}
          onForget={noop}
          onSaveDoc={noop}
          onSaveFact={noop}
          onChangeType={noop}
          onAcceptMemorySuggestion={noop}
          onAcceptSkillSuggestion={noop}
          onAcceptMergeSuggestion={noop}
          onRefreshSuggestions={refresh as never}
        />
      </ToastProvider>
    </LocaleProvider>,
  );
}

describe("MemoryPanel 记忆主区视图（v4.73）", () => {
  beforeEach(() => {
    mocks.morningPreload.mockReset();
    mocks.setMorningPreload.mockReset();
    mocks.morningPreload.mockResolvedValue(true);
    mocks.setMorningPreload.mockResolvedValue(undefined);
    mocks.projectBrief.mockResolvedValue(true);
    mocks.setProjectBrief.mockResolvedValue(undefined);
    // v4.377：组件回调走 .then 链，mock 必须返回 Promise（裸 vi.fn() 返回
    // undefined，.then 崩成 Uncaught Exception 拖红整个文件）。
    mocks.setDreamMode.mockReset();
    mocks.setDreamMode.mockResolvedValue(undefined);
    mocks.dismissSuggestion.mockReset();
    mocks.dismissSuggestion.mockResolvedValue(undefined);
    mocks.dreamPurgePreview.mockReset();
    mocks.dreamPurgePreview.mockResolvedValue([]);
    mocks.dreamPurge.mockReset();
    mocks.dreamPurge.mockResolvedValue(0);
  });

  it("项目本体开关：读取初始状态 + 点击切换并持久化（6.2 可关闭）", async () => {
    await renderPanel();
    expect(await screen.findByText(/项目本体 开/)).toBeTruthy();
    fireEvent.click(screen.getByTitle(/点击关闭项目本体/));
    await waitFor(() => expect(mocks.setProjectBrief).toHaveBeenCalledWith(false));
    expect(await screen.findByText(/项目本体 关/)).toBeTruthy();
  });

  it("晨报预载开关：读取初始状态 + 点击切换并持久化", async () => {
    await renderPanel();
    expect(screen.getByTestId("memory-view")).toBeTruthy();
    // v4.74：总览三枚统计小卡（事实/文档/建议）
    expect(screen.getByTestId("memory-kpi-facts")).toBeTruthy();
    expect(screen.getByTestId("memory-kpi-docs")).toBeTruthy();
    expect(screen.getByTestId("memory-kpi-suggestions")).toBeTruthy();
    expect(await screen.findByText("晨报预载 开")).toBeTruthy();
    fireEvent.click(screen.getByText("晨报预载 开"));
    expect(await screen.findByText("晨报预载 关")).toBeTruthy();
    expect(mocks.setMorningPreload).toHaveBeenCalledWith(false);
    expect(mocks.morningPreload).toHaveBeenCalledTimes(1);
  });

  // v4.377 建议制：三态循环 建议 → 直写 → 关 → 建议。
  it("自动做梦模式开关：三态循环切换并持久化", async () => {
    await renderPanel();
    expect(await screen.findByText(/自动做梦 建议/)).toBeTruthy();
    fireEvent.click(screen.getByTestId("memory-dream-mode"));
    expect(await screen.findByText(/自动做梦 直写/)).toBeTruthy();
    expect(mocks.setDreamMode).toHaveBeenCalledWith("auto");
    fireEvent.click(screen.getByTestId("memory-dream-mode"));
    expect(await screen.findByText(/自动做梦 关/)).toBeTruthy();
    expect(mocks.setDreamMode).toHaveBeenCalledWith("off");
    fireEvent.click(screen.getByTestId("memory-dream-mode"));
    expect(await screen.findByText(/自动做梦 建议/)).toBeTruthy();
    expect(mocks.setDreamMode).toHaveBeenCalledWith("suggest");
  });

  // v4.377 建议制：待确认建议可忽略（出队不写库），卡片即时消失。
  it("记忆建议忽略：队列建议带忽略按钮，点击后调绑定并从列表移除", async () => {
    await renderPanelWithSuggestions(async () => ({
      memories: [
        {
          id: "d1758000000000000000-1",
          name: "user-unit",
          description: "自动做梦提炼的事实",
          type: "user",
          body: "用户单位为 XX 公司",
          reason: "待确认",
          evidence: [],
        },
      ],
      skills: [],
      merges: [],
      generatedAt: new Date().toISOString(),
      available: true,
      source: "test",
    }));
    // 打开建议 tab 并扫描
    fireEvent.click(screen.getByRole("button", { name: /^(建议|Suggestions)(?!s*d)/ }));
    const scanBtn = await screen.findByRole("button", { name: /扫描|scan/i });
    fireEvent.click(scanBtn);
    expect(await screen.findByText("user-unit")).toBeTruthy();
    const ignoreBtn = screen.getByRole("button", { name: /^(忽略|Ignore)$/ });
    fireEvent.click(ignoreBtn);
    await waitFor(() => expect(mocks.dismissSuggestion).toHaveBeenCalledWith("d1758000000000000000-1"));
    await waitFor(() => expect(screen.queryByText("user-unit")).toBeNull());
  });

  // v4.377 清污：两步式——首次拉预览，再点执行删除。
  it("清污按钮：预览 → 确认态 → 执行 DreamPurge", async () => {
    mocks.dreamPurgePreview.mockResolvedValue(["bad-1", "bad-2"]);
    mocks.dreamPurge.mockResolvedValue(2);
    await renderPanelWithSuggestions(async () => ({
      memories: [], skills: [], merges: [],
      generatedAt: new Date().toISOString(), available: true, source: "test",
    }));
    fireEvent.click(screen.getByRole("button", { name: /^(建议|Suggestions)(?!s*d)/ }));
    const scanBtn = await screen.findByRole("button", { name: /扫描|scan/i });
    fireEvent.click(scanBtn);
    const purgeBtn = await screen.findByTestId("memory-purge-auto-dream");
    fireEvent.click(purgeBtn);
    await waitFor(() => expect(mocks.dreamPurgePreview).toHaveBeenCalledTimes(1));
    expect(await screen.findByText(/确认删除 2 条|Delete 2 auto-written/)).toBeTruthy();
    fireEvent.click(screen.getByTestId("memory-purge-auto-dream"));
    await waitFor(() => expect(mocks.dreamPurge).toHaveBeenCalledWith(["bad-1", "bad-2"]));
  });
});
