// CadPanel.portal.test.tsx — §7.3「检查器随工具」portal 行为锁：
// 右栏宿主在挂时，文本清单/符号统计两段渲染进宿主（sheet 本体不再含检查器，
// 根节点 data-inspector=external）；宿主撤走回落内联（回归合并前形态）。
// 状态与回调零上移的等价性由 CadPanel.test.tsx（无宿主=内联）继续锁定。
import { expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { useState } from "react";
import { CadPanel } from "./CadPanel";
import { cadRecentResetForTest } from "./cadRecent";
import { cadCacheResetForTest } from "./cadToolCache";

vi.mock("../../lib/bridge", () => ({
  app: {
    PickFiles: async () => [{ path: "C:/d/plan.dxf", name: "plan.dxf", size: 1000 }],
    CadOpen: async () =>
      JSON.stringify({
        kind: "dxf",
        version: "AC1021",
        entityCount: 10,
        skipped: 0,
        textCount: 2,
        layers: ["图框"],
        warnings: [],
        file: { name: "plan.dxf", size: 1000 },
        sheets: [{ name: "JS-01", box: [0, 0, 1, 1], passRate: 0.9, texts: 2, png: "" }],
      }),
    CadRaster: async () =>
      JSON.stringify({ pngPath: "C:/cache/s.png", dataUrl: "data:image/png;base64,iVBORw0KGgo=" }),
    CadTexts: async () =>
      JSON.stringify({ total: 1, truncated: false, texts: [{ text: "JS-01", layer: "图框", x: 1, y: 2, h: 9, sheet: 1 }] }),
    CadCount: async () =>
      JSON.stringify({ items: [], layers: [], warnings: [] }),
    RecognizeImage: async () => "答",
  },
}));

import { LocaleProvider } from "../../lib/i18n";

/** 宿主在挂：检查器两段住在 rail 宿主里，sheet 本体让位成两栏画布区 */
it("检查器随工具：宿主在挂时文本/统计渲染进右栏宿主，sheet 本体不含检查器", async () => {
  cadRecentResetForTest();
  cadCacheResetForTest();
  function Harness() {
    const [host, setHost] = useState<HTMLDivElement | null>(null);
    return (
      <>
        <div ref={setHost} data-testid="rail-host" />
        <LocaleProvider>
          <CadPanel variant="sheet" inspectorHost={host} />
        </LocaleProvider>
      </>
    );
  }
  render(<Harness />);
  // sheet 变体空态没有 cad-import 钮（有文件后才出现）；locale 随挂载环境可能落
  // 英文——按稳定 testid 走空态 CTA，不按文案
  fireEvent.click(await screen.findByTestId("cad-import-empty"));
  // 打开成功后检查器节点应出现在宿主里，且不在 sheet 本体里
  await waitFor(() => expect(screen.getByTestId("cad-inspector")).toBeTruthy());
  const host = screen.getByTestId("rail-host");
  expect(host.querySelector('[data-testid="cad-inspector"]')).toBeTruthy();
  const panel = screen.getByTestId("cad-panel");
  expect(panel.getAttribute("data-inspector")).toBe("external");
  expect(panel.querySelector('[data-testid="cad-inspector"]')).toBeNull();
});

/** 宿主撤走（右栏收起）：检查器回落 sheet 内联——与合并前 DOM 形态一致 */
it("宿主撤走回落内联：检查器回到 sheet 本体，data-inspector=inline", async () => {
  cadRecentResetForTest();
  cadCacheResetForTest();
  function Harness() {
    const [host, setHost] = useState<HTMLDivElement | null>(null);
    const [railOpen, setRailOpen] = useState(true);
    return (
      <>
        {railOpen && <div ref={setHost} data-testid="rail-host" />}
        <button data-testid="toggle-rail" onClick={() => setRailOpen(v => !v)}>收放右栏</button>
        <LocaleProvider>
          <CadPanel variant="sheet" inspectorHost={host} />
        </LocaleProvider>
      </>
    );
  }
  render(<Harness />);
  // sheet 变体空态没有 cad-import 钮（有文件后才出现）；locale 随挂载环境可能落
  // 英文——按稳定 testid 走空态 CTA，不按文案
  fireEvent.click(await screen.findByTestId("cad-import-empty"));
  await waitFor(() => expect(screen.getByTestId("cad-inspector")).toBeTruthy());
  expect(screen.getByTestId("rail-host").querySelector('[data-testid="cad-inspector"]')).toBeTruthy();
  // 收起右栏 → 宿主卸载 → 检查器回落 sheet 本体
  fireEvent.click(screen.getByTestId("toggle-rail"));
  await waitFor(() => expect(screen.getByTestId("cad-panel").getAttribute("data-inspector")).toBe("inline"));
  expect(screen.getByTestId("cad-panel").querySelector('[data-testid="cad-inspector"]')).toBeTruthy();
  expect(screen.queryByTestId("rail-host")).toBeNull();
});
