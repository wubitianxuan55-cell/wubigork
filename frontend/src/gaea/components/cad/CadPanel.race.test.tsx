import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { CadPanel } from "./CadPanel";
import { cadRecentResetForTest } from "./cadRecent";
import { cadCacheResetForTest } from "./cadToolCache";
import { LocaleProvider } from "../../lib/i18n";

type Deferred<T> = {
  promise: Promise<T>;
  resolve: (value: T) => void;
  reject: (error: unknown) => void;
};

function deferred<T>(): Deferred<T> {
  let resolve!: (value: T) => void;
  let reject!: (error: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

const state = vi.hoisted(() => ({
  pickResults: [] as Array<{ path: string; name: string; size: number }>,
  openPaths: [] as string[],
  rasterCalls: [] as Array<[string, number, number]>,
  countPaths: [] as string[],
  textPaths: [] as string[],
  recognizeCalls: [] as Array<[string, string]>,
  askSent: [] as string[],
  openImpl: null as null | ((path: string) => Promise<string>),
  rasterImpl: null as null | ((path: string, sheet: number, width: number) => Promise<string>),
  countImpl: null as null | ((path: string) => Promise<string>),
  textsImpl: null as null | ((path: string) => Promise<string>),
  recognizeImpl: null as null | ((path: string, prompt: string) => Promise<string>),
}));

const openFixture = (name: string) => ({
  kind: "dxf",
  version: "AC1021",
  entityCount: 10,
  skipped: 0,
  textCount: 1,
  layers: ["0"],
  warnings: [],
  file: { name, size: 100 },
  sheets: [
    { name: "sheet-1", box: [0, 0, 10, 10], passRate: 0.9, texts: 1, png: "" },
    { name: "sheet-2", box: [10, 10, 20, 20], passRate: 0.8, texts: 0, png: "" },
  ],
});

const countFixture = (name: string) => ({
  items: [
    {
      block: `${name}-BLOCK`,
      name,
      confidence: 1,
      count: 1,
      perSheet: { "1": 1 },
      sample: [0, 0],
    },
  ],
  layers: [],
  warnings: [],
});

const textsFixture = (name: string) => ({
  total: 1,
  truncated: false,
  texts: [{ text: `${name}-TEXT`, layer: "0", x: 0, y: 0, h: 1, sheet: 1 }],
});

vi.mock("../../lib/bridge", () => ({
  app: {
    PickFiles: async () => {
      const picked = state.pickResults.shift();
      return picked ? [picked] : null;
    },
    CadOpen: async (path: string) => {
      state.openPaths.push(path);
      if (!state.openImpl) throw new Error("missing openImpl");
      return state.openImpl(path);
    },
    CadRaster: async (path: string, sheet: number, width: number) => {
      state.rasterCalls.push([path, sheet, width]);
      if (!state.rasterImpl) throw new Error("missing rasterImpl");
      return state.rasterImpl(path, sheet, width);
    },
    CadCount: async (path: string) => {
      state.countPaths.push(path);
      if (!state.countImpl) throw new Error("missing countImpl");
      return state.countImpl(path);
    },
    CadTexts: async (path: string) => {
      state.textPaths.push(path);
      if (!state.textsImpl) throw new Error("missing textsImpl");
      return state.textsImpl(path);
    },
    RecognizeImage: async (path: string, prompt: string) => {
      state.recognizeCalls.push([path, prompt]);
      if (!state.recognizeImpl) throw new Error("missing recognizeImpl");
      return state.recognizeImpl(path, prompt);
    },
    LogFrontendError: async () => {},
  },
}));

function renderPanel(onAsk?: (text: string) => void) {
  return render(
    <LocaleProvider>
      <CadPanel onAsk={onAsk} />
    </LocaleProvider>,
  );
}

async function importPath(path: string, name: string) {
  state.pickResults.push({ path, name, size: 100 });
  fireEvent.click(screen.getByTestId("cad-import"));
  await waitFor(() => expect(state.openPaths).toContain(path));
}

beforeEach(() => {
  cadRecentResetForTest();
  cadCacheResetForTest();
  localStorage.setItem("gaea-lang", "zh");
  state.pickResults = [];
  state.openPaths = [];
  state.rasterCalls = [];
  state.countPaths = [];
  state.textPaths = [];
  state.recognizeCalls = [];
  state.askSent = [];
  state.openImpl = async (path) => JSON.stringify(openFixture(path.split("/").pop() ?? "drawing"));
  state.rasterImpl = async (path, sheet, width) =>
    JSON.stringify({
      pngPath: `${path}-sheet-${sheet}-w${width}.png`,
      dataUrl: `data:image/png;base64,${sheet}-${width}`,
    });
  state.countImpl = async (path) => JSON.stringify(countFixture(path));
  state.textsImpl = async (path) => JSON.stringify(textsFixture(path));
  state.recognizeImpl = async () => "answer";
});

describe("CadPanel 异步代际隔离", () => {
  it("打开新图纸立即清空上一张图纸的统计与问答状态", async () => {
    renderPanel();
    await importPath("C:/drawings/A.dxf", "A.dxf");
    await screen.findByTestId("cad-raster-img");
    fireEvent.click(screen.getByTestId("cad-count-btn"));
    await screen.findByTestId("cad-count");
    expect(screen.getByTestId("cad-count").textContent).toContain("C:/drawings/A.dxf");

    await importPath("C:/drawings/B.dxf", "B.dxf");
    await screen.findByTestId("cad-raster-img");
    expect(screen.queryByTestId("cad-count")).toBeNull();
    expect(screen.queryByText(/C:\/drawings\/A\.dxf/)).toBeNull();
  });

  it("A 的慢统计晚于 B 打开返回时不得覆盖 B", async () => {
    const slowA = deferred<string>();
    state.countImpl = (path) =>
      path.endsWith("A.dxf") ? slowA.promise : Promise.resolve(JSON.stringify(countFixture("B")));
    renderPanel();
    await importPath("C:/drawings/A.dxf", "A.dxf");
    await screen.findByTestId("cad-raster-img");
    fireEvent.click(screen.getByTestId("cad-count-btn"));
    await waitFor(() => expect(state.countPaths).toContain("C:/drawings/A.dxf"));

    await importPath("C:/drawings/B.dxf", "B.dxf");
    await screen.findByTestId("cad-raster-img");
    slowA.resolve(JSON.stringify(countFixture("A")));
    await waitFor(() => expect(screen.queryByText(/A-BLOCK/)).toBeNull());
    expect(screen.queryByTestId("cad-count")).toBeNull();
  });

  it("快速切图框时旧图框的慢渲染不得覆盖新图框", async () => {
    const slowSheet2 = deferred<string>();
    state.rasterImpl = (path, sheet, width) => {
      if (sheet === 2 && width === 0) return slowSheet2.promise;
      return Promise.resolve(
        JSON.stringify({
          pngPath: `${path}-sheet-${sheet}-w${width}.png`,
          dataUrl: `data:image/png;base64,${sheet}-${width}`,
        }),
      );
    };
    renderPanel();
    await importPath("C:/drawings/A.dxf", "A.dxf");
    await screen.findByTestId("cad-raster-img");
    fireEvent.click(screen.getByTestId("cad-sheet-2"));
    await waitFor(() => expect(state.rasterCalls).toContainEqual(["C:/drawings/A.dxf", 2, 0]));
    fireEvent.click(screen.getByTestId("cad-sheet-1"));
    const img = await screen.findByTestId("cad-raster-img");
    expect(img.getAttribute("src")).toContain("1-0");

    slowSheet2.resolve(
      JSON.stringify({
        pngPath: "C:/drawings/A.dxf-sheet-2-w0.png",
        dataUrl: "data:image/png;base64,2-0",
      }),
    );
    await waitFor(() => expect(screen.getByTestId("cad-raster-img").getAttribute("src")).toContain("1-0"));
  });

  it("问图纸未完成时禁止切图框，迟到答案只属于发起时图框", async () => {
    const slowAnswer = deferred<string>();
    state.recognizeImpl = () => slowAnswer.promise;
    renderPanel();
    await importPath("C:/drawings/A.dxf", "A.dxf");
    await screen.findByTestId("cad-raster-img");
    fireEvent.change(screen.getByTestId("cad-ask-input"), { target: { value: "图号？" } });
    fireEvent.click(screen.getByTestId("cad-ask-btn"));
    await waitFor(() => expect(state.recognizeCalls.length).toBe(1));
    expect((screen.getByTestId("cad-sheet-1") as HTMLButtonElement).disabled).toBe(true);
    const initialSheet1RasterCalls = state.rasterCalls.filter((c) => c[1] === 1 && c[2] === 0).length;
    fireEvent.click(screen.getByTestId("cad-sheet-1"));
    expect(state.rasterCalls.filter((c) => c[1] === 1 && c[2] === 0)).toHaveLength(initialSheet1RasterCalls);

    slowAnswer.resolve("A 图号");
    await screen.findByTestId("cad-answer");
  });

  it("交 AI 算量只使用当前图纸代际的 count/texts", async () => {
    const slowA = deferred<string>();
    state.countImpl = (path) =>
      path.endsWith("A.dxf") ? slowA.promise : Promise.resolve(JSON.stringify(countFixture("B")));
    const onAsk = vi.fn();
    renderPanel(onAsk);
    await importPath("C:/drawings/A.dxf", "A.dxf");
    await screen.findByTestId("cad-raster-img");
    fireEvent.click(screen.getByTestId("cad-count-btn"));
    await waitFor(() => expect(state.countPaths).toContain("C:/drawings/A.dxf"));

    await importPath("C:/drawings/B.dxf", "B.dxf");
    await screen.findByTestId("cad-raster-img");
    fireEvent.click(screen.getByTestId("cad-handoff-pre"));
    await waitFor(() => expect(onAsk).toHaveBeenCalledTimes(1));
    slowA.resolve(JSON.stringify(countFixture("A")));
    await waitFor(() => expect(screen.queryByText(/A-BLOCK/)).toBeNull());
    const msg = onAsk.mock.calls[0][0] as string;
    expect(msg).toContain("B.dxf");
    expect(msg).toContain("B-BLOCK");
    expect(msg).not.toContain("A-BLOCK");
  });

  it("符号统计警告原样展示（散线/天正对象不适用时不静默）", async () => {
    state.countImpl = async () =>
      JSON.stringify({
        ...countFixture("WARN"),
        items: [],
        warnings: ["未发现块引用：散线或天正对象不适用块统计"],
      });
    renderPanel();
    await importPath("C:/drawings/WARN.dxf", "WARN.dxf");
    await screen.findByTestId("cad-raster-img");
    fireEvent.click(screen.getByTestId("cad-count-btn"));
    await screen.findByText(/未发现块引用/);
  });
});
