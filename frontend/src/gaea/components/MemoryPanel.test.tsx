import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { MemoryView } from "../lib/types";
import { LocaleProvider } from "../lib/i18n";
import { ToastProvider } from "./Toast";
import { MemoryPanel } from "./MemoryPanel" // 静态 import：动态加载在并发全量下会迟到，render 落进下一用例（Found multiple elements 既坑，setup.ts 注释同款）

vi.setConfig({ testTimeout: 15_000 }); // 全量并行下 import/setup 变慢,既有 2.7s 用例会超默认 5s(ContextView 40s 同配方)

const mocks = vi.hoisted(() => ({
  setMemoryEnabled: vi.fn(),
}));

// 2026-11-25 办公工具梳理批：晨报预载（MorningPreload）与项目本体（MemoryBrief）
// 两个开关连同绑定一并撤下——那两块注入与记忆索引逐条重复，已停注。bridge mock
// 只留记忆总开关；退役的四个绑定名若再被访问会落到兜底 vi.fn()，由下方回归钉
// 的「面板不出现这两个开关」断言兜住。
vi.mock("../lib/bridge", () => ({
  app: new Proxy({}, {
    get(_t, prop) {
      if (prop === "SetMemoryEnabled") return mocks.setMemoryEnabled;
      return vi.fn();
    },
  }),
  openExternal: vi.fn(),
}));

async function renderPanel() {
  const noop = async () => {};
  // MemoryPanel 已文件顶部静态导入（迟到 render 既坑根修）
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
  // MemoryPanel 已文件顶部静态导入（迟到 render 既坑根修）
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
          onAcceptSkillSuggestion={noop}
          onAcceptMergeSuggestion={noop}
          onRefreshSuggestions={refresh as never}
        />
      </ToastProvider>
    </LocaleProvider>,
  );
}

describe("MemoryPanel 记忆主区视图（v4.73）", () => {
  // v4.377：组件回调走 .then 链，mock 必须返回 Promise（裸 vi.fn() 返回
  // undefined，.then 崩成 Uncaught Exception 拖红整个文件）。
  it("总览三枚统计小卡 + 记忆开关：点击切换并持久化", { timeout: 20000 }, async () => {
    mocks.setMemoryEnabled.mockReset();
    mocks.setMemoryEnabled.mockResolvedValue(undefined);
    await renderPanel();
    expect(screen.getByTestId("memory-view")).toBeTruthy();
    // v4.74：总览三枚统计小卡（事实/文档/建议）
    expect(screen.getByTestId("memory-kpi-facts")).toBeTruthy();
    expect(screen.getByTestId("memory-kpi-docs")).toBeTruthy();
    expect(screen.getByTestId("memory-kpi-suggestions")).toBeTruthy();
    fireEvent.click(screen.getByTitle(/点击关闭记忆注入/));
    await waitFor(() => expect(mocks.setMemoryEnabled).toHaveBeenCalledWith(false));
  });

  // 2026-11-25 回归钉：晨报预载 / 项目本体两个开关不再出现在面板上（两块注入已停注，
  // 开关与绑定同批撤下——留着就是一个拨了没反应的假开关）。
  it("晨报预载/项目本体两个开关已撤下", { timeout: 20000 }, async () => {
    await renderPanel();
    expect(await screen.findByText(/^记忆 (开|关)$/)).toBeTruthy();
    expect(screen.queryByText(/晨报预载/)).toBeNull();
    expect(screen.queryByText(/项目本体/)).toBeNull();
  });

  // v4.478 建议可见性：挂载首拉后统计卡显示真实计数（无需打开 tab、无需手动扫描）。
  // 2026-11-21 改判：auto-dream 删除后建议=技能候选+合并候选（memories 队列消亡）。
  it("建议首拉：挂载即自动拉取，统计卡显示真实计数", async () => {
    const refresh = vi.fn(async () => ({
      skills: [{ id: "s1", name: "workflow-x", description: "技能候选", scope: "project", body: "", reason: "", evidence: [] }],
      merges: [],
      generatedAt: new Date().toISOString(),
      available: true,
      source: "test",
    }));
    await renderPanelWithSuggestions(refresh);
    await waitFor(() => {
      expect(screen.getByTestId("memory-kpi-suggestions").textContent).toContain("1");
    });
  });

});
