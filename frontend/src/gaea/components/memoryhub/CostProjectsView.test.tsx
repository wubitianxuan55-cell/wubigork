import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { CostProjectsView } from "./CostProjectsView";
import { ToastProvider } from "../Toast";
import type { CostEstimateItem, CostEstimateVersion, CostProject, CostProjectSummary, CostStageCompareRow, CostStageDeviation, CostStageValue } from "../../lib/types";

// ── 内存态 mock（对齐 lib/mock/cost.ts 的测算项目实现口径）──
const state = vi.hoisted(() => {
  let seq = 1;
  return {
    projects: [] as CostProject[],
    items: [] as CostEstimateItem[],
    versions: [] as CostEstimateVersion[],
    // GA6-09：成本库单价搜索下拉（EntryPicker）的分页检索桩
    searchPage: vi.fn(),
    seq,
    reset() {
      this.projects = [];
      this.items = [];
      this.versions = [];
      seq = 1;
    },
    next() {
      return seq++;
    },
  };
});

vi.mock("../../lib/bridge", () => ({
  app: {
    CostProjectList: async (): Promise<CostProjectSummary[]> =>
      state.projects.map((p) => {
        const items = state.items.filter((i) => i.projectId === p.id);
        const versions = state.versions.filter((v) => v.projectId === p.id);
        return {
          ...p,
          itemCount: items.length,
          total: items.reduce((s, i) => s + (i.quantity || 0) * (i.price || 0), 0),
          versionCount: versions.length,
        };
      }),
    CostProjectGet: async (id: string): Promise<CostProject | null> => state.projects.find((p) => p.id === id) ?? null,
    CostProjectSave: async (p: CostProject): Promise<string> => {
      if (!p.id) {
        const np = { ...p, id: `p${state.next()}` };
        state.projects.push(np);
        return np.id;
      }
      state.projects = state.projects.map((x) => (x.id === p.id ? { ...x, ...p } : x));
      return p.id;
    },
    CostProjectDelete: async (id: string) => {
      state.projects = state.projects.filter((p) => p.id !== id);
    },
    CostEstimateItems: async (projectId: string): Promise<CostEstimateItem[]> => state.items.filter((i) => i.projectId === projectId),
    CostEstimateItemSave: async (i: CostEstimateItem): Promise<number> => {
      if (!i.id) {
        const n = { ...i, id: state.next() };
        state.items.push(n);
        return n.id;
      }
      state.items = state.items.map((x) => (x.id === i.id ? { ...x, ...i } : x));
      return i.id;
    },
    CostEstimateItemDelete: async (id: number) => {
      state.items = state.items.filter((i) => i.id !== id);
    },
    CostEstimateVersions: async (projectId: string): Promise<CostEstimateVersion[]> =>
      state.versions.filter((v) => v.projectId === projectId).sort((a, b) => b.version - a.version),
    CostEstimateVersionSave: async (projectId: string, note: string): Promise<CostEstimateVersion> => {
      const items = state.items.filter((i) => i.projectId === projectId);
      const v: CostEstimateVersion = {
        id: state.next(),
        projectId,
        version: state.versions.filter((x) => x.projectId === projectId).length + 1,
        total: items.reduce((s, i) => s + (i.quantity || 0) * (i.price || 0), 0),
        snapshot: JSON.stringify(items),
        note,
        createdAt: new Date().toISOString(),
      };
      state.versions.push(v);
      return v;
    },
    CostEstimateSediment: async (projectId: string, ids: number[]): Promise<number> => {
      const idset = new Set(ids);
      return state.items.filter((i) => i.projectId === projectId && idset.has(i.id ?? -1) && (i.price || 0) > 0).length;
    },
    CostSearch: async () => [],
    // GA6-09：EntryPicker 走的 CostSearchPage（返回 {items,total,error?}）
    CostSearchPage: (...args: unknown[]) => state.searchPage(...args),
    // v4.2 五算对比（FiveCalcPanel 挂载即调）：空实现不干扰既有用例。
    CostStages: async (): Promise<CostStageValue[]> => [],
    CostStageSave: async () => {},
    CostStageCompare: async (): Promise<CostStageCompareRow[]> => [],
    CostStageDeviations: async (): Promise<CostStageDeviation[]> => [],
  },
}));

const wrap = (node: React.ReactNode) => <ToastProvider>{node}</ToastProvider>;

beforeEach(() => {
  state.reset();
  state.searchPage.mockReset();
  state.searchPage.mockResolvedValue({ items: [], total: 0, error: "" });
});

describe("CostProjectsView 测算项目", () => {
  it("空态引导 → 新建项目 → 列表出现并可选中", async () => {
    render(wrap(<CostProjectsView />));
    expect(await screen.findByText(/还没有测算项目/)).toBeTruthy();

    fireEvent.click(screen.getByText("新建项目"));
    fireEvent.change(await screen.findByPlaceholderText("如：XX 市政道路土方测算"), { target: { value: "市政道路土方测算" } });
    fireEvent.click([...screen.getAllByRole("button", { name: /^保\s*存$/ })].pop()!);

    expect(await screen.findByText("市政道路土方测算")).toBeTruthy();
    // 选中后右侧显示项目详情头
    expect(screen.getByText("编辑信息")).toBeTruthy();
  });

  it("新增明细行并保存：金额=数量×单价，合计正确", async () => {
    render(wrap(<CostProjectsView />));
    fireEvent.click(await screen.findByText("新建项目"));
    fireEvent.change(await screen.findByPlaceholderText("如：XX 市政道路土方测算"), { target: { value: "土方测算" } });
    fireEvent.click([...screen.getAllByRole("button", { name: /^保\s*存$/ })].pop()!);
    await screen.findByText("编辑信息");

    fireEvent.click(screen.getByText("加行"));
    const titleInput = (await screen.findAllByPlaceholderText("名称（必填）"))[0];
    fireEvent.change(titleInput, { target: { value: "机械挖土方" } });
    const qty = screen.getByPlaceholderText("数量");
    const price = screen.getByPlaceholderText("单价");
    fireEvent.change(qty, { target: { value: "10" } });
    fireEvent.change(price, { target: { value: "12.5" } });

    // 失焦触发保存 → 行获得 id → 可勾选沉淀
    fireEvent.blur(price);
    await waitFor(() => expect(screen.getByText("¥125")).toBeTruthy());
    const checkbox = (await screen.findAllByRole("checkbox"))[0];
    expect((checkbox as HTMLInputElement).disabled).toBe(false);
    fireEvent.click(checkbox);
    expect(screen.getByText("沉淀选中(1)")).toBeTruthy();
  });

  it("保存版本后版本列表出现，可查看快照", async () => {
    render(wrap(<CostProjectsView />));
    fireEvent.click(await screen.findByText("新建项目"));
    fireEvent.change(await screen.findByPlaceholderText("如：XX 市政道路土方测算"), { target: { value: "土方测算" } });
    fireEvent.click([...screen.getAllByRole("button", { name: /^保\s*存$/ })].pop()!);
    await screen.findByText("编辑信息");

    fireEvent.click(screen.getByText("加行"));
    fireEvent.change((await screen.findAllByPlaceholderText("名称（必填）"))[0], { target: { value: "机械挖土方" } });
    fireEvent.change(screen.getByPlaceholderText("单价"), { target: { value: "12.5" } });
    fireEvent.blur(screen.getByPlaceholderText("单价"));

    fireEvent.click(screen.getByText("保存版本"));
    fireEvent.change(await screen.findByPlaceholderText("版本备注（可选），如：土方工程 V1 初稿"), { target: { value: "V1 初稿" } });
    fireEvent.click([...screen.getAllByRole("button", { name: /^保\s*存$/ })].pop()!);

    expect(await screen.findByText("v1")).toBeTruthy();
    fireEvent.click(screen.getByText("V1 初稿"));
    expect(await screen.findByText("机械挖土方")).toBeTruthy();
  });

  it("沉淀选中行返回条数并提示", async () => {
    render(wrap(<CostProjectsView />));
    fireEvent.click(await screen.findByText("新建项目"));
    fireEvent.change(await screen.findByPlaceholderText("如：XX 市政道路土方测算"), { target: { value: "土方测算" } });
    fireEvent.click([...screen.getAllByRole("button", { name: /^保\s*存$/ })].pop()!);
    await screen.findByText("编辑信息");

    fireEvent.click(screen.getByText("加行"));
    fireEvent.change((await screen.findAllByPlaceholderText("名称（必填）"))[0], { target: { value: "机械挖土方" } });
    fireEvent.change(screen.getByPlaceholderText("单价"), { target: { value: "12.5" } });
    fireEvent.blur(screen.getByPlaceholderText("单价"));
    await waitFor(() => expect((screen.getAllByRole("checkbox")[0] as HTMLInputElement).disabled).toBe(false));
    fireEvent.click(screen.getAllByRole("checkbox")[0]);
    await waitFor(() => expect(screen.getByText("沉淀选中(1)")).toBeTruthy());

    fireEvent.click(screen.getByText("沉淀选中(1)"));
    expect(await screen.findByText("已沉淀 1 条明细到成本库")).toBeTruthy();
  });
});

// ── GA6-09：引用成本库下拉（EntryPicker）消费 CostSearchPage.Error ──
describe("测算项目 · 成本库引用读取失败可见化（GA6-09）", () => {
  /** 建一个含一行的项目，返回「引用成本库单价」搜索输入。 */
  async function openRowWithPicker() {
    render(wrap(<CostProjectsView />));
    fireEvent.click(await screen.findByText("新建项目"));
    fireEvent.change(await screen.findByPlaceholderText("如：XX 市政道路土方测算"), { target: { value: "土方测算" } });
    fireEvent.click([...screen.getAllByRole("button", { name: /^保\s*存$/ })].pop()!);
    await screen.findByText("编辑信息");
    fireEvent.click(screen.getByText("加行"));
    return await screen.findByPlaceholderText("引用成本库单价（搜索）或留空手动估价");
  }

  it("error 空：下拉无提示条（行为零变化）", async () => {
    const ref = await openRowWithPicker();
    state.searchPage.mockResolvedValue({ items: [], total: 0, error: "" });
    fireEvent.change(ref, { target: { value: "钢" } });
    ref.focus();
    await waitFor(() => expect(state.searchPage).toHaveBeenCalled());
    expect(screen.queryByTestId("cost-picker-read-error")).toBeNull();
    expect(screen.getByText("输入关键词搜索成本库单价…")).toBeTruthy();
  });

  it("error 非空：下拉内提示条含「读取失败」+ 重试，不显示「输入关键词…」空态", async () => {
    const ref = await openRowWithPicker();
    state.searchPage.mockResolvedValue({ items: [], total: 0, error: "读取失败（磁盘 IO），结果可能不完整" });
    fireEvent.change(ref, { target: { value: "钢" } });
    ref.focus();

    const bar = await screen.findByTestId("cost-picker-read-error");
    expect(bar.getAttribute("role")).toBe("alert");
    expect(bar.textContent).toContain("读取失败");
    // 不把「读坏了」当「没搜到」
    expect(screen.getByText(/成本库读取失败，未取到条目/)).toBeTruthy();
    expect(screen.queryByText("输入关键词搜索成本库单价…")).toBeNull();

    // 可重试：点「重试」重新检索（改坏锚点：去掉 error 消费 → 本断言链全 FAIL）
    const calls = state.searchPage.mock.calls.length;
    fireEvent.click(screen.getByTestId("cost-picker-read-error-retry"));
    await waitFor(() => expect(state.searchPage.mock.calls.length).toBeGreaterThan(calls));
  });
});
