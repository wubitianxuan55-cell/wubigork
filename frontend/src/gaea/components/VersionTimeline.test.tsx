// VersionTimeline.test.tsx — 版本时间线组件单测（加载/空态/列表/预览/恢复回调 +
// A1「与当前对比」：内联 diff 区 / unsupported / contentMissing / 无差异 / 收起 /
// 竞态防护 / 长 diff 折叠）。
// 组件为纯展示层：records / onPreview / onRestore 全经 props 注入；对比取数
// mock lib/versionCompare 的 compareVersionWithCurrent（保留其余真实导出，
// clampDiffRows 走真函数）。i18n：组件经 useT 读 vcompare.* 字典，render 统一
// 包 LocaleProvider + localStorage.setItem("gaea-lang","zh") 锁 zh 断言。
import { describe, expect, it, vi, beforeEach } from "vitest";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { VersionTimeline } from "./VersionTimeline";
import { LocaleProvider } from "../lib/i18n";
import { compareVersionWithCurrent, type VersionCompareResult } from "../lib/versionCompare";
import type { JournalChangeRecord } from "../lib/types";

vi.mock("../lib/versionCompare", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../lib/versionCompare")>()),
  compareVersionWithCurrent: vi.fn(),
}));

// 固定本地时间构造（避免时区漂移）：r1=14:05、r2=15:20、r3=12:01。
const t = (h: number, m: number) => new Date(2026, 8, 1, h, m, 0).getTime();

const rec = (over: Partial<JournalChangeRecord> & { id: string }): JournalChangeRecord => ({
  sessionId: "s1",
  space: "work",
  turn: 1,
  tool: "edit_file",
  target: "docs/周报.docx",
  beforeSummary: "旧内容",
  afterSummary: "新内容",
  at: t(14, 5),
  status: "pending_verify",
  baselinePath: "C:/ws/.gaea/work/rollback/xxx.snap",
  ...over,
});

// zh 模式渲染：组件 useT 依赖 LocaleProvider（缺它直接抛错）。
const renderT = (ui: React.ReactNode) => {
  localStorage.setItem("gaea-lang", "zh");
  return render(<LocaleProvider>{ui}</LocaleProvider>);
};

const mockedCompare = vi.mocked(compareVersionWithCurrent);

// 便捷文本对比结果构造。
const txt = (over: Partial<Extract<VersionCompareResult, { kind: "text" }>> = {}): VersionCompareResult => ({
  kind: "text",
  rows: [
    { type: "del", text: "old line" },
    { type: "ctx", text: "same line" },
    { type: "add", text: "new line" },
  ],
  add: 1,
  del: 1,
  contentMissing: false,
  ...over,
});

beforeEach(() => {
  mockedCompare.mockReset();
});

describe("VersionTimeline 版本时间线", () => {
  it("records=null：加载态（转圈 + 文案），不渲染列表", () => {
    renderT(<VersionTimeline path="docs/周报.docx" records={null} onPreview={() => {}} onRestore={() => {}} />);
    expect(screen.getByTestId("version-timeline-loading")).toBeTruthy();
    expect(screen.getByText("正在加载版本记录…")).toBeTruthy();
    expect(screen.queryByTestId("version-timeline-row")).toBeNull();
  });

  it("records=[]：空态（无基线快照提示），恢复说明常驻", () => {
    renderT(<VersionTimeline path="docs/周报.docx" records={[]} onPreview={() => {}} onRestore={() => {}} />);
    expect(screen.getByTestId("version-timeline-empty")).toBeTruthy();
    expect(screen.getByText(/暂无可回滚的版本快照/)).toBeTruthy();
    // 顶部恢复语义说明（预览即护栏，不做二次确认弹窗）
    expect(screen.getByText("恢复会把该文件写回所选版本，当前内容成为新版本")).toBeTruthy();
  });

  it("列表渲染：按传入顺序（调用方已 at 倒序）逐行展示时间/工具/轮次/状态", () => {
    const rows = [
      rec({ id: "r2", at: t(15, 20), tool: "xlsx_apply", turn: 3, status: "verified" }),
      rec({ id: "r1", at: t(14, 5), tool: "edit_file", turn: 2, status: "pending_verify" }),
      rec({ id: "r3", at: t(12, 1), tool: "write_file", turn: 0, status: "failed" }),
    ];
    renderT(<VersionTimeline path="docs/周报.docx" records={rows} onPreview={() => {}} onRestore={() => {}} />);
    const items = screen.getAllByTestId("version-timeline-row");
    expect(items).toHaveLength(3);
    // 最新在前：第一行 15:20 · xlsx_apply · 第 3 轮 · 复核通过
    expect(items[0].textContent).toContain("15:20");
    expect(screen.getByText("xlsx_apply")).toBeTruthy();
    expect(screen.getByText("第 3 轮")).toBeTruthy();
    expect(screen.getByText("复核通过")).toBeTruthy();
    expect(items[1].textContent).toContain("14:05");
    expect(screen.getByText("第 2 轮")).toBeTruthy();
    expect(screen.getByText("待复核")).toBeTruthy();
    // 轮次 0 显示「轮外」；failed 状态徽标
    expect(items[2].textContent).toContain("12:01");
    expect(screen.getByText("轮外")).toBeTruthy();
    expect(screen.getByText("复核未通过")).toBeTruthy();
    // 快照计数徽标
    expect(screen.getByText("3")).toBeTruthy();
  });

  it("预览回调：点「预览」注入基线快照路径", () => {
    const onPreview = vi.fn();
    renderT(
      <VersionTimeline
        path="docs/周报.docx"
        records={[rec({ id: "r1", baselinePath: "C:/ws/.gaea/work/rollback/a.snap" })]}
        onPreview={onPreview}
        onRestore={() => {}}
      />,
    );
    fireEvent.click(screen.getByTitle("预览该版本快照"));
    expect(onPreview).toHaveBeenCalledTimes(1);
    expect(onPreview).toHaveBeenCalledWith("C:/ws/.gaea/work/rollback/a.snap");
  });

  it("恢复回调：点「恢复」注入该条记录；恢复进行中禁用全部恢复按钮，完成后恢复可用", async () => {
    let resolveRestore: () => void = () => {};
    const onRestore = vi.fn(
      (_r: JournalChangeRecord) => new Promise<void>((res) => { resolveRestore = res; }),
    );
    renderT(
      <VersionTimeline
        path="docs/周报.docx"
        records={[rec({ id: "r1" }), rec({ id: "r2", at: t(15, 20) })]}
        onPreview={() => {}}
        onRestore={onRestore}
      />,
    );
    fireEvent.click(screen.getByTitle("恢复到 14:05 版本：将回滚到该时间版本"));
    expect(onRestore).toHaveBeenCalledTimes(1);
    expect(onRestore).toHaveBeenCalledWith(expect.objectContaining({ id: "r1" }));
    // 恢复进行中：所有恢复按钮禁用（避免并发写盘），本行转圈
    const restoreBtns = screen.getAllByTitle(/^恢复到 \d{2}:\d{2} 版本/);
    expect(restoreBtns).toHaveLength(2);
    restoreBtns.forEach((b) => expect((b as HTMLButtonElement).disabled).toBe(true));
    // 完成 → 按钮恢复可用
    await act(async () => { resolveRestore(); });
    await waitFor(() =>
      restoreBtns.forEach((b) => expect((b as HTMLButtonElement).disabled).toBe(false)),
    );
  });
});

describe("VersionTimeline 与当前对比", () => {
  it("点「与当前对比」：以（基线快照, 当前文件）取数，行下展开内联 diff 区（loading → 行级红绿 diff + diffstat 芯片）", async () => {
    let resolveCmp: (r: VersionCompareResult) => void = () => {};
    mockedCompare.mockImplementation(
      () => new Promise<VersionCompareResult>((res) => { resolveCmp = res; }),
    );
    renderT(
      <VersionTimeline
        path="docs/周报.docx"
        records={[rec({ id: "r1", baselinePath: "snap/a", target: "docs/周报.docx" })]}
        onPreview={() => {}}
        onRestore={() => {}}
      />,
    );
    fireEvent.click(screen.getByTitle("与当前对比"));
    // 语义固定：该基线快照 vs 当前工作区文件
    expect(mockedCompare).toHaveBeenCalledWith("snap/a", "docs/周报.docx");
    // 取数进行中：spinner 占位，无 diff 内容
    expect(screen.getByTestId("vcompare-panel")).toBeTruthy();
    expect(screen.getByTestId("vcompare-loading")).toBeTruthy();
    expect(screen.queryByTestId("vcompare-diff")).toBeNull();
    await act(async () => { resolveCmp(txt()); });
    // 结果就绪：diffstat 芯片 + 行级 diff（del 红行 / add 绿行 / ctx 弱化行）
    expect(await screen.findByTestId("vcompare-diff")).toBeTruthy();
    expect(screen.getByTestId("vcompare-stat").textContent).toBe("+1−1");
    const diff = screen.getByTestId("vcompare-diff");
    expect(diff.textContent).toContain("old line");
    expect(diff.textContent).toContain("new line");
    expect(diff.textContent).toContain("same line");
    const row = diff.firstElementChild as HTMLElement;
    expect(row.style.background).toContain("--del-bg");
    // 标题走字典（zh）：版本对比 · {label}
    expect(screen.getByText("版本对比 · 14:05")).toBeTruthy();
  });

  it("unsupported：非文本格式显示弱化降级提示，不渲染 diff", async () => {
    mockedCompare.mockResolvedValue({ kind: "unsupported", ext: ".docx" });
    renderT(
      <VersionTimeline
        path="docs/周报.docx"
        records={[rec({ id: "r1" })]}
        onPreview={() => {}}
        onRestore={() => {}}
      />,
    );
    fireEvent.click(screen.getByTitle("与当前对比"));
    expect(await screen.findByTestId("vcompare-unsupported").then((el) => el.textContent)).toBe(
      "该格式暂不支持文本对比，可分别预览两个版本",
    );
    expect(screen.queryByTestId("vcompare-diff")).toBeNull();
  });

  it("contentMissing：顶部提示内容不可用，diff 照常渲染", async () => {
    mockedCompare.mockResolvedValue(txt({ contentMissing: true }));
    renderT(
      <VersionTimeline
        path="docs/周报.docx"
        records={[rec({ id: "r1" })]}
        onPreview={() => {}}
        onRestore={() => {}}
      />,
    );
    fireEvent.click(screen.getByTitle("与当前对比"));
    await screen.findByTestId("vcompare-diff");
    expect(screen.getByTestId("vcompare-content-missing").textContent).toBe(
      "基线或当前内容不可用，结果可能不完整",
    );
  });

  it("无差异（add+del=0）：显示「两个版本内容一致」，不渲染 diff 区", async () => {
    mockedCompare.mockResolvedValue(
      txt({ rows: [{ type: "ctx", text: "same" }], add: 0, del: 0 }),
    );
    renderT(
      <VersionTimeline
        path="docs/周报.docx"
        records={[rec({ id: "r1" })]}
        onPreview={() => {}}
        onRestore={() => {}}
      />,
    );
    fireEvent.click(screen.getByTitle("与当前对比"));
    expect(await screen.findByTestId("vcompare-empty").then((el) => el.textContent)).toBe(
      "两个版本内容一致",
    );
    expect(screen.queryByTestId("vcompare-diff")).toBeNull();
    expect(screen.queryByTestId("vcompare-stat")).toBeNull();
  });

  it("再点同一行（或点「收起对比」）：折叠并取消挂载对比内容，不重复取数", async () => {
    mockedCompare.mockResolvedValue(txt());
    renderT(
      <VersionTimeline
        path="docs/周报.docx"
        records={[rec({ id: "r1" })]}
        onPreview={() => {}}
        onRestore={() => {}}
      />,
    );
    fireEvent.click(screen.getByTitle("与当前对比"));
    await screen.findByTestId("vcompare-diff");
    // 行按钮切换为「收起对比」，点击后取消挂载
    fireEvent.click(screen.getByTitle("收起对比"));
    expect(screen.queryByTestId("vcompare-panel")).toBeNull();
    // 面板内「收起对比」按钮同样收起（重新展开 → 只取数两次）
    fireEvent.click(screen.getByTitle("与当前对比"));
    await screen.findByTestId("vcompare-diff");
    fireEvent.click(screen.getByTestId("vcompare-hide"));
    expect(screen.queryByTestId("vcompare-panel")).toBeNull();
    expect(mockedCompare).toHaveBeenCalledTimes(2);
  });

  it("竞态防护：连点两行，旧请求的慢返回不覆盖新行结果", async () => {
    let resolveSlow: (r: VersionCompareResult) => void = () => {};
    mockedCompare.mockImplementation((baseline: string) =>
      baseline === "snap/slow"
        ? new Promise<VersionCompareResult>((res) => { resolveSlow = res; })
        : Promise.resolve(txt({ rows: [{ type: "add", text: "fast-B" }], add: 1, del: 0 })),
    );
    renderT(
      <VersionTimeline
        path="docs/周报.docx"
        records={[
          rec({ id: "r-slow", baselinePath: "snap/slow", at: t(15, 20) }),
          rec({ id: "r-fast", baselinePath: "snap/fast", at: t(14, 5) }),
        ]}
        onPreview={() => {}}
        onRestore={() => {}}
      />,
    );
    // 第一行展开（慢请求在途）→ 切到第二行（快请求）
    fireEvent.click(screen.getAllByTitle("与当前对比")[0]);
    fireEvent.click(screen.getByTitle("与当前对比"));
    expect(await screen.findByTestId("vcompare-diff").then((el) => el.textContent)).toContain("fast-B");
    // 慢请求此刻才返回：必须被淘汰，面板仍是第二行的结果
    await act(async () => {
      resolveSlow(txt({ rows: [{ type: "del", text: "SLOW-A" }], add: 0, del: 1 }));
    });
    await waitFor(() => {
      const panel = screen.getByTestId("vcompare-panel").textContent;
      expect(panel).toContain("fast-B");
      expect(panel).not.toContain("SLOW-A");
    });
  });

  it("长 diff 折叠：超 200 行只渲染前 200 行 + 「展开全部」，展开后可收起", async () => {
    const rows = [
      ...Array.from({ length: 200 }, (_, i) => ({ type: "ctx" as const, text: `L${i}` })),
      ...Array.from({ length: 25 }, (_, i) => ({ type: "del" as const, text: `del-${i}` })),
      ...Array.from({ length: 25 }, (_, i) => ({ type: "add" as const, text: `add-${i}` })),
    ];
    mockedCompare.mockResolvedValue(txt({ rows, add: 25, del: 25 }));
    renderT(
      <VersionTimeline
        path="docs/周报.docx"
        records={[rec({ id: "r1" })]}
        onPreview={() => {}}
        onRestore={() => {}}
      />,
    );
    fireEvent.click(screen.getByTitle("与当前对比"));
    await screen.findByTestId("vcompare-diff");
    const diff = screen.getByTestId("vcompare-diff");
    // 200 行 + 截断开关 = 201 个子节点；第 200 行起的 del/add 不可见
    expect(diff.children).toHaveLength(201);
    expect(diff.textContent).toContain("L199");
    expect(diff.textContent).not.toContain("L200");
    expect(diff.textContent).not.toContain("del-0");
    expect(diff.textContent).not.toContain("add-24");
    fireEvent.click(screen.getByTestId("vcompare-expand"));
    // 展开：250 行 + 「收起」开关
    expect(diff.children).toHaveLength(251);
    expect(diff.textContent).toContain("del-24");
    expect(screen.getByTestId("vcompare-expand").textContent).toBe("收起");
    // 收起 → 回到前 200 行
    fireEvent.click(screen.getByTestId("vcompare-expand"));
    expect(diff.children).toHaveLength(201);
    expect(diff.textContent).not.toContain("del-0");
  });
});
