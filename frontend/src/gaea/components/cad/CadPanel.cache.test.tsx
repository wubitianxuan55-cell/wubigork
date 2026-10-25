// CadPanel.cache.test.tsx — cadToolCache 解析缓存（设计 §7.3「重开不重跑解析」）。
// ①纯函数：LRU 淘汰（文件数/每文件 raster 数）、touch 续期、reset 隔离；
// ②组件集成：开图→关（卸载）→重开（最近图纸一键）→零 CadOpen/CadRaster 调用，
//   图面/图框/文本/统计直接回填；切图框重渲正常（缓存只存默认宽度渲图）。
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { CadPanel } from "./CadPanel";
import { cadRecentResetForTest } from "./cadRecent";
import {
  cadCacheGet,
  cadCachePutOpen,
  cadCachePutRaster,
  cadCacheResetForTest,
} from "./cadToolCache";
import type { CadOpenResult } from "../../lib/bridge/cad";

const state = vi.hoisted(() => ({
  openPaths: [] as string[],
  rasterCalls: [] as Array<[string, number, number]>,
}));

function openFixture(tag: string): CadOpenResult {
  return {
    kind: "dxf",
    version: "AC1021",
    entityCount: 100,
    skipped: 0,
    textCount: 2,
    layers: ["图框"],
    warnings: [],
    file: { name: `${tag}.dxf`, size: 1000 },
    sheets: [
      { name: "整图", box: [0, 0, 100, 100], passRate: 0.03, texts: 1, png: "" },
      { name: `${tag}-平面图`, box: [0, 0, 90, 90], passRate: 0.9, texts: 1, png: "" },
    ],
  };
}

vi.mock("../../lib/bridge", () => ({
  app: {
    PickFiles: async () => [{ path: "C:/d/plan.dxf", name: "plan.dxf", size: 1000 }],
    CadOpen: async (path: string) => {
      state.openPaths.push(path);
      return JSON.stringify(openFixture(path.split("/").pop() ?? "d"));
    },
    CadRaster: async (path: string, sheet: number, width: number) => {
      state.rasterCalls.push([path, sheet, width]);
      return JSON.stringify({
        pngPath: `${path}-s${sheet}-w${width}.png`,
        dataUrl: `data:image/png;base64,${sheet}-${width}`,
      });
    },
    CadTexts: async () =>
      JSON.stringify({
        total: 1,
        truncated: false,
        texts: [{ text: "JS-01", layer: "图框", x: 1, y: 2, h: 125, sheet: 2 }],
      }),
    CadCount: async () =>
      JSON.stringify({
        items: [{ block: "SYM1", name: "插座", confidence: 1, count: 5, perSheet: { "2": 5 }, sample: [0, 0] }],
        layers: [],
        warnings: [],
      }),
  },
}));

import { LocaleProvider } from "../../lib/i18n";

const renderPanel = () =>
  render(
    <LocaleProvider>
      <CadPanel />
    </LocaleProvider>,
  );

beforeEach(() => {
  state.openPaths = [];
  state.rasterCalls = [];
  cadRecentResetForTest();
  cadCacheResetForTest();
  localStorage.setItem("gaea-lang", "zh");
});

describe("cadToolCache 纯函数（LRU/touch/reset）", () => {
  it("putOpen 建 entry；raster/texts/count 逐层落；get 触摸续期", () => {
    cadCachePutOpen("a", openFixture("a"), 2);
    cadCachePutRaster("a", 2, { pngPath: "p", dataUrl: "d" });
    const e = cadCacheGet("a");
    expect(e?.bestSheet).toBe(2);
    expect(e?.rasters).toHaveLength(1);
    expect(e?.texts).toBeNull();
    // 未登记 path 的 put 静默忽略（putOpen 先行）
    cadCachePutRaster("ghost", 1, { pngPath: "p", dataUrl: "d" });
    expect(cadCacheGet("ghost")).toBeNull();
  });

  it("文件数上限 8：第 9 个挤掉最旧；touch 使被访问者续期", () => {
    for (let i = 1; i <= 8; i++) cadCachePutOpen(`f${i}`, openFixture(`f${i}`), 1);
    cadCacheGet("f1"); // f1 续期 → 最旧变 f2
    cadCachePutOpen("f9", openFixture("f9"), 1);
    expect(cadCacheGet("f1")).not.toBeNull();
    expect(cadCacheGet("f2")).toBeNull();
    expect(cadCacheGet("f9")).not.toBeNull();
  });

  it("每文件 raster 上限 3：旧渲图被挤掉（同图框重渲覆盖不重复）", () => {
    cadCachePutOpen("a", openFixture("a"), 1);
    for (const s of [1, 2, 3]) cadCachePutRaster("a", s, { pngPath: `p${s}`, dataUrl: `d${s}` });
    cadCachePutRaster("a", 1, { pngPath: "p1-new", dataUrl: "d1-new" }); // 覆盖不增
    expect(cadCacheGet("a")?.rasters.map((r) => r.sheet)).toEqual([2, 3, 1]);
    cadCachePutRaster("a", 4, { pngPath: "p4", dataUrl: "d4" }); // 挤掉最旧 sheet 2
    expect(cadCacheGet("a")?.rasters.map((r) => r.sheet)).toEqual([3, 1, 4]);
  });

  it("reset 清空（vitest 用例间隔离）", () => {
    cadCachePutOpen("a", openFixture("a"), 1);
    cadCacheResetForTest();
    expect(cadCacheGet("a")).toBeNull();
  });
});

describe("CadPanel 重开不重跑解析（缓存集成）", () => {
  it("开图→卸载→重开（最近图纸一键）：零 CadOpen 调用，图面/图框/统计直接在位", async () => {
    // 第一次会话：导入→等渲图→加载统计
    const first = renderPanel();
    fireEvent.click(await screen.findByTestId("cad-import"));
    await screen.findByTestId("cad-raster-img");
    fireEvent.click(screen.getByTestId("cad-count-btn"));
    await screen.findByTestId("cad-count");
    await waitFor(() => expect(state.openPaths).toEqual(["C:/d/plan.dxf"]));
    first.unmount();

    // 第二次会话：重开面板，点最近图纸
    renderPanel();
    fireEvent.click(await screen.findByTestId("cad-recent-plan.dxf"));
    await screen.findByTestId("cad-raster-img");
    await screen.findByTestId("cad-count");
    // 关键断言：CadOpen 没有第二次调用；默认宽度渲图命中缓存也不再调 CadRaster
    expect(state.openPaths).toEqual(["C:/d/plan.dxf"]);
    expect(state.rasterCalls).toEqual([["C:/d/plan.dxf", 2, 0]]);
    expect(screen.getByTestId("cad-count").textContent).toContain("5");
    expect(screen.getByTestId("cad-sheets")?.textContent).toContain("plan.dxf-平面图");
  });

  it("重开后切其他图框：仍正常重渲（缓存只存默认宽度渲图，按图框键）", async () => {
    const first = renderPanel();
    fireEvent.click(await screen.findByTestId("cad-import"));
    await screen.findByTestId("cad-raster-img");
    first.unmount();

    renderPanel();
    fireEvent.click(await screen.findByTestId("cad-recent-plan.dxf"));
    await screen.findByTestId("cad-raster-img");
    expect(state.rasterCalls).toEqual([["C:/d/plan.dxf", 2, 0]]);
    // 切到图框 1（未缓存）→ 正常走 CadRaster
    fireEvent.click(screen.getByTestId("cad-sheet-1"));
    await waitFor(() => expect(state.rasterCalls).toContainEqual(["C:/d/plan.dxf", 1, 0]));
    await screen.findByTestId("cad-raster-img");
  });
});
