import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { MemoryView } from "../lib/types";
import { LocaleProvider } from "../lib/i18n";
import { ToastProvider } from "./Toast";

const mocks = vi.hoisted(() => ({
  morningPreload: vi.fn(),
  setMorningPreload: vi.fn(),
  projectBrief: vi.fn(),
  setProjectBrief: vi.fn(),
}));

vi.mock("../lib/bridge", () => ({
  app: new Proxy({}, {
    get(_t, prop) {
      if (prop === "MorningPreload") return mocks.morningPreload;
      if (prop === "SetMorningPreload") return mocks.setMorningPreload;
      if (prop === "MemoryBrief") return mocks.projectBrief;
      if (prop === "SetMemoryBrief") return mocks.setProjectBrief;
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

describe("MemoryPanel 记忆主区视图（v4.73）", () => {
  beforeEach(() => {
    mocks.morningPreload.mockReset();
    mocks.setMorningPreload.mockReset();
    mocks.morningPreload.mockResolvedValue(true);
    mocks.setMorningPreload.mockResolvedValue(undefined);
    mocks.projectBrief.mockResolvedValue(true);
    mocks.setProjectBrief.mockResolvedValue(undefined);
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
});
