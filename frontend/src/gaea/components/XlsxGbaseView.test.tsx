import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { XlsxPreview } from "./XlsxPreview";
import { ToastProvider } from "./Toast";

// B1 多维表视图层组件测试：bridge 只 mock ReadFile（挂载即调）；
// 其余 app 方法在渲染路径外，不给默认实现（用到即失败 = 诚实暴露）。
const mocks = vi.hoisted(() => ({
  readFile: vi.fn(async (_rel: string) => ({ path: _rel, markdown: "", size: 0 })),
}));

vi.mock("../lib/bridge", () => ({
  app: {
    ReadFile: (rel: string) => mocks.readFile(rel),
    RevealWorkspacePath: async () => {},
    OpenWorkspacePath: async () => {},
  },
}));

const wrap = (node: React.ReactNode) => <ToastProvider>{node}</ToastProvider>;

const XLSX_BODY = JSON.stringify({
  sheets: [
    {
      name: "预算",
      rows: [
        [{ ref: "A1", value: "状态", type: "string" }, { ref: "B1", value: "金额", type: "string" }],
        [{ ref: "A2", value: "完成", type: "string" }, { ref: "B2", value: "120", type: "number" }],
        [{ ref: "A3", value: "进行中", type: "string" }, { ref: "B3", value: "80", type: "number" }],
        [{ ref: "A4", value: "完成", type: "string" }, { ref: "B4", value: "60", type: "number" }],
      ],
      merged: [],
      colWidths: {},
    },
    { name: "明细", rows: [[{ ref: "A1", value: "日期", type: "string" }]], colWidths: {} },
  ],
});

const CONFIG = JSON.stringify({
  version: 1,
  views: [
    {
      id: "byStatus",
      name: "按状态",
      type: "grid",
      groupBy: "状态",
      sort: [{ column: "金额", dir: "desc" }],
      colorRules: [{ column: "状态", op: "eq", value: "完成", color: "#dcfce7" }], // hex-exempt 测试数据（配置内容）
    },
  ],
});

beforeEach(() => {
  mocks.readFile.mockReset();
  mocks.readFile.mockResolvedValue({ path: "x", markdown: "", size: 0 });
});

describe("XlsxPreview 多维表视图层（B1）", () => {
  it("sidecar 合法 → 视图 chips 出现；点视图进分组渲染，回「表格」恢复 SheetGrid", async () => {
    mocks.readFile.mockResolvedValue({ path: "mock.gbase.json", markdown: CONFIG, size: 10 });
    render(wrap(<XlsxPreview body={XLSX_BODY} fileName="mock.xlsx" relPath="mock.xlsx" />));
    await waitFor(() => expect(screen.getByTestId("gbase-view-byStatus")).toBeTruthy());
    expect(screen.getByTestId("gbase-view-grid")).toBeTruthy();

    fireEvent.click(screen.getByTestId("gbase-view-byStatus"));
    const grouped = await screen.findByTestId("gbase-grouped");
    // 分组块：完成(2)/进行中(1)；组内按金额降序 120 在 60 前
    const groups = within(grouped).getAllByTestId("gbase-group");
    expect(groups).toHaveLength(2);
    expect(groups[0]!.textContent).toContain("完成");
    expect(groups[0]!.textContent).toContain("2 条");
    expect(groups[1]!.textContent).toContain("进行中");
    expect(within(grouped).getAllByText("120").length).toBeGreaterThan(0);

    fireEvent.click(screen.getByTestId("gbase-view-grid"));
    await waitFor(() => expect(screen.queryByTestId("gbase-grouped")).toBeNull());
  });

  it("视图引用列不存在 → 降级横幅 + 保持表格视图", async () => {
    const bad = JSON.stringify({
      version: 1,
      views: [{ id: "v1", name: "按阶段", type: "grid", groupBy: "阶段" }],
    });
    mocks.readFile.mockResolvedValue({ path: "mock.gbase.json", markdown: bad, size: 10 });
    render(wrap(<XlsxPreview body={XLSX_BODY} fileName="mock.xlsx" relPath="mock.xlsx" />));
    await waitFor(() => expect(screen.getByTestId("gbase-view-v1")).toBeTruthy());
    fireEvent.click(screen.getByTestId("gbase-view-v1"));
    await waitFor(() =>
      expect(screen.getByText(/引用的列不存在：阶段/)).toBeTruthy(),
    );
    expect(screen.queryByTestId("gbase-grouped")).toBeNull(); // 回退表格
    expect(screen.getByText("回表格")).toBeTruthy();
  });

  it("sheet 绑定失配 → 横幅提示且不渲染分组", async () => {
    const bound = JSON.stringify({
      version: 1,
      views: [{ id: "s1", name: "仅明细", type: "grid", sheet: "明细", groupBy: "日期" }],
    });
    mocks.readFile.mockResolvedValue({ path: "mock.gbase.json", markdown: bound, size: 10 });
    render(wrap(<XlsxPreview body={XLSX_BODY} fileName="mock.xlsx" relPath="mock.xlsx" />));
    await waitFor(() => expect(screen.getByTestId("gbase-view-s1")).toBeTruthy());
    fireEvent.click(screen.getByTestId("gbase-view-s1"));
    await waitFor(() => expect(screen.getByText(/仅作用于 sheet「明细」/)).toBeTruthy());
    expect(screen.queryByTestId("gbase-grouped")).toBeNull();
  });

  it("坏 JSON 但内容含 views → 配置告警横幅；不含 views 的文本静默忽略", async () => {
    mocks.readFile.mockResolvedValue({
      path: "mock.gbase.json",
      markdown: '{ "version": 1, "views": [ { bad',
      size: 10,
    });
    const { unmount } = render(wrap(<XlsxPreview body={XLSX_BODY} fileName="mock.xlsx" relPath="mock.xlsx" />));
    await waitFor(() => expect(screen.getByText(/视图配置告警/)).toBeTruthy());
    unmount();

    // 另一个文件：sidecar 内容不含 "views"（普通文本）→ 静默无视图（不误报告警）
    mocks.readFile.mockResolvedValue({
      path: "plain.gbase.json",
      markdown: "// 普通文本文件内容，不是视图配置\n",
      size: 10,
    });
    render(wrap(<XlsxPreview body={XLSX_BODY} fileName="plain.xlsx" relPath="plain.xlsx" />));
    await screen.findByText("预算");
    expect(screen.queryByText(/视图配置告警/)).toBeNull();
    expect(screen.queryByTestId("gbase-view-grid")).toBeNull();
  });

  it("sidecar 不存在（ReadFile 拒绝）→ 无 chips 无横幅（既有行为不变）", async () => {
    mocks.readFile.mockRejectedValue(new Error("not found"));
    render(wrap(<XlsxPreview body={XLSX_BODY} fileName="mock.xlsx" relPath="mock.xlsx" />));
    await screen.findByText("预算");
    expect(screen.queryByTestId("gbase-view-grid")).toBeNull();
    expect(screen.queryByText(/视图配置告警/)).toBeNull();
  });
});
