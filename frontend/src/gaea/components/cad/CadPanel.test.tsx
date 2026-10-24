// CadPanel.test.tsx — CAD 图纸卡组件端到端金样（vi.mock bridge 全量替换）：
// 导入→自动选质量最好图框→分幅图→切框→问图纸（加宽喂图红线）→发到对话；
// 最近图纸（内存态跨卸载）与文本清单全文复制。
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { CadPanel } from "./CadPanel";
import { cadRecentResetForTest } from "./cadRecent";

const state = vi.hoisted(() => ({
  pickFilters: [] as string[],
  openPaths: [] as string[],
  rasterCalls: [] as Array<[number, number]>,
  recognizeCalls: [] as Array<[string, string]>,
  askSent: [] as string[],
}));

const openFixture = {
  kind: "dxf",
  version: "AC1021",
  entityCount: 7104,
  skipped: 0,
  textCount: 542,
  layers: ["图框", "E-系统文字"],
  warnings: ["测试警告：建议导出 T3"],
  file: { name: "plan.dxf", size: 7683275 },
  sheets: [
    { name: "整图", box: [0, 0, 87631, 116825], passRate: 0.03, texts: 135, png: "" },
    { name: "JS-01-一层平面图", box: [0, 0, 59000, 42000], passRate: 0.9, texts: 39, png: "" },
  ],
};

vi.mock("../../lib/bridge", () => ({
  app: {
    PickFiles: async (filters?: string) => {
      state.pickFilters.push(filters ?? "");
      return [{ path: "C:/d/plan.dxf", name: "plan.dxf", size: 1000 }];
    },
    CadOpen: async (path: string) => {
      state.openPaths.push(path);
      return JSON.stringify(openFixture);
    },
    CadRaster: async (_p: string, sheet: number, width: number) => {
      state.rasterCalls.push([sheet, width]);
      return JSON.stringify({
        pngPath: `C:/cache/sheet-${sheet}-w${width}.png`,
        dataUrl: "data:image/png;base64,iVBORw0KGgo=",
      });
    },
    CadTexts: async () =>
      JSON.stringify({
        total: 2,
        truncated: false,
        texts: [
          { text: "JS-01", layer: "图框", x: 1, y: 2, h: 125, sheet: 2 },
          { text: "一层平面图", layer: "图框", x: 1, y: 3, h: 125, sheet: 2 },
        ],
      }),
    CadCount: async () =>
      JSON.stringify({
        items: [
          { block: "$equip$0001", name: "带保护接点暗装插座", confidence: 1, count: 31, perSheet: { "2": 27, "1": 4 }, sample: [0, 0] },
          { block: "SYM2", name: "", confidence: 0, count: 6, perSheet: { "2": 6 }, sample: [0, 0] },
        ],
        layers: [{ layer: "E-干线", length: 668050, lines: 120 }],
        warnings: [],
      }),
    RecognizeImage: async (imagePath: string, prompt: string) => {
      state.recognizeCalls.push([imagePath, prompt]);
      return "图号为 JS-01。";
    },
  },
}));

import { LocaleProvider } from "../../lib/i18n";

const renderPanel = (onAsk?: (text: string) => void) =>
  render(
    <LocaleProvider>
      <CadPanel onAsk={onAsk} />
    </LocaleProvider>,
  );

beforeEach(() => {
  state.pickFilters = [];
  state.openPaths = [];
  state.rasterCalls = [];
  state.recognizeCalls = [];
  state.askSent = [];
  cadRecentResetForTest();
  localStorage.setItem("gaea-lang", "zh");
});

describe("CadPanel 导入与图框质量闸", () => {
  it("导入→自动选中 passRate 最高的图框并渲染分幅图；警告原样透出", async () => {
    renderPanel();
    fireEvent.click(await screen.findByTestId("cad-import"));
    // passRate 0.9 的「JS-01-一层平面图」应被自动选中（quality gate）
    await waitFor(() => expect(state.rasterCalls).toContainEqual([2, 0]));
    expect(state.pickFilters).toEqual(["dwg,dxf,pdf"]);
    expect(state.openPaths).toEqual(["C:/d/plan.dxf"]);
    expect(screen.getByTestId("cad-sheet-2").textContent).toContain("JS-01-一层平面图");
    const img = screen.getByTestId("cad-raster-img") as HTMLImageElement;
    expect(img.src.startsWith("data:image/png;base64,")).toBe(true);
    expect(screen.getByText(/测试警告：建议导出 T3/)).toBeTruthy();
  });

  it("切图框重新取图", async () => {
    renderPanel();
    fireEvent.click(await screen.findByTestId("cad-import"));
    await screen.findByTestId("cad-raster-img");
    fireEvent.click(screen.getByTestId("cad-sheet-1"));
    await waitFor(() => expect(state.rasterCalls).toContainEqual([1, 0]));
  });
});

describe("CadPanel 问图纸（加宽喂图红线）与发到对话", () => {
  it("问图纸：按 6000 宽重渲后喂识图，答案展示并可发对话流", async () => {
    const onAsk = vi.fn((text: string) => state.askSent.push(text));
    renderPanel(onAsk);
    fireEvent.click(await screen.findByTestId("cad-import"));
    await screen.findByTestId("cad-raster-img");
    fireEvent.change(screen.getByTestId("cad-ask-input"), { target: { value: "图号是什么" } });
    fireEvent.click(screen.getByTestId("cad-ask-btn"));
    await screen.findByTestId("cad-answer");
    // 红线：问图纸必须加宽重渲（6000）取 pngPath，不复用首屏 3000 图
    expect(state.rasterCalls).toContainEqual([2, 6000]);
    expect(state.recognizeCalls.length).toBe(1);
    const [pngPath, prompt] = state.recognizeCalls[0];
    expect(pngPath).toBe("C:/cache/sheet-2-w6000.png");
    expect(prompt).toContain("图号是什么");
    expect(prompt).toContain("不要猜测");
    // 发到对话：带图纸名与图框名的【图纸问答】前缀
    fireEvent.click(screen.getByTestId("cad-ask-send"));
    expect(state.askSent[0]).toContain("【图纸问答：plan.dxf｜JS-01-一层平面图】");
    expect(state.askSent[0]).toContain("答：图号为 JS-01。");
  });

  it("文本清单：拉取后按当前图框过滤展示", async () => {
    renderPanel();
    fireEvent.click(await screen.findByTestId("cad-import"));
    await screen.findByTestId("cad-raster-img");
    fireEvent.click(screen.getByRole("button", { name: /文本清单/ }));
    const box = await screen.findByTestId("cad-texts");
    expect(box.textContent).toContain("JS-01");
    expect(box.textContent).toContain("一层平面图");
  });

  it("符号统计（算量）：块计数/自动命名/未命名占位/分布/层线长", async () => {
    renderPanel();
    fireEvent.click(await screen.findByTestId("cad-import"));
    await screen.findByTestId("cad-raster-img");
    fireEvent.click(screen.getByTestId("cad-count-btn"));
    const box = await screen.findByTestId("cad-count");
    // 自动命名行 + 数量 + 图框分布（sheet 2=JS-01-一层平面图）
    expect(box.textContent).toContain("带保护接点暗装插座");
    expect(box.textContent).toContain("×31");
    expect(box.textContent).toContain("2:JS-01-一层×27");
    // 未命名行：占位提示 + 人工命名指引
    expect(box.textContent).toContain("（未命名——待确认）");
    expect(box.textContent).toContain("确认命名与挂价");
    // 层线长
    expect(box.textContent).toContain("E-干线");
    expect(box.textContent).toContain("668,050");
  });

  it("交 AI 算量：识别数据打包发对话流（未拉统计时自动补拉），消息含技能名/统计行/待命名标记", async () => {
    const onAsk = vi.fn();
    renderPanel(onAsk);
    fireEvent.click(await screen.findByTestId("cad-import"));
    await screen.findByTestId("cad-raster-img");
    // 未先点「符号统计」——交接按钮应自动补拉 CadCount
    fireEvent.click(screen.getByTestId("cad-handoff-pre"));
    await waitFor(() => expect(onAsk).toHaveBeenCalledTimes(1));
    const msg = onAsk.mock.calls[0][0] as string;
    expect(msg).toContain("【CAD 图纸算量】plan.dxf（DXF，2 张图）");
    expect(msg).toContain("cad-takeoff");
    expect(msg).toContain("带保护接点暗装插座|$equip$0001|31|1.00|1×4 2×27");
    expect(msg).toContain("（待命名）|SYM2|6|-|2×6");
    expect(msg).toContain("E-干线|668050");
    expect(msg).toContain("图上文字");
    expect(msg).toContain("[图框] JS-01");
  });
});

describe("CadPanel 最近图纸（内存态）与文本全文复制", () => {
  beforeEach(() => {
    state.pickFilters = [];
    state.openPaths = [];
    state.rasterCalls = [];
    state.recognizeCalls = [];
    state.askSent = [];
    cadRecentResetForTest();
    localStorage.setItem("gaea-lang", "zh");
  });

  it("导入成功入最近列表；面板卸载重挂后仍可一键重开（模块态跨卸载）", async () => {
    const first = renderPanel();
    fireEvent.click(await screen.findByTestId("cad-import"));
    await screen.findByTestId("cad-raster-img");
    expect(screen.getByTestId("cad-recent")).toBeTruthy();
    first.unmount();
    renderPanel();
    const chip = screen.getByTestId("cad-recent-plan.dxf");
    fireEvent.click(chip);
    await waitFor(() => expect(state.openPaths).toEqual(["C:/d/plan.dxf", "C:/d/plan.dxf"]));
  });

  it("文本清单全文复制：TSV 带表头（图框/图层/内容），成功反馈「已复制」", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    renderPanel();
    fireEvent.click(await screen.findByTestId("cad-import"));
    await screen.findByTestId("cad-raster-img");
    fireEvent.click(screen.getByRole("button", { name: /文本清单/ }));
    await screen.findByTestId("cad-texts");
    fireEvent.click(screen.getByTestId("cad-texts-copy"));
    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    const payload = writeText.mock.calls[0][0] as string;
    const lines = payload.split("\n");
    expect(lines[0]).toBe("图框\t图层\t内容");
    expect(lines[1]).toBe("2\t图框\tJS-01");
    expect(lines[2]).toBe("2\t图框\t一层平面图");
    expect(screen.getByTestId("cad-texts-copy").textContent).toContain("已复制");
  });
});
