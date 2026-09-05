// VersionTimeline.test.tsx — 版本时间线组件单测（加载/空态/列表/预览/恢复回调 +
// A1「与当前对比」：内联 diff 区 / unsupported / contentMissing / 无差异 / 收起 /
// 竞态防护 / 长 diff 折叠）。
// v4.87 统一 diff 查看器收口：text/docx/xlsx 对比体经 ChangesDiff 渲染——行在
// vcompare-diff 包裹层的 changes-diff-hunk 内，docx 段号 / xlsx ref 走 marker 列，
// 连续 ctx 行会被 ChangesDiff 折叠（长 diff 计数按折叠口径断言）。
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
    // 结果就绪：diffstat 芯片 + 统一 diff 查看器（del 红行 / add 绿行 / ctx 弱化行）
    expect(await screen.findByTestId("vcompare-diff")).toBeTruthy();
    expect(screen.getByTestId("vcompare-stat").textContent).toBe("+1−1");
    const diff = screen.getByTestId("vcompare-diff");
    expect(diff.textContent).toContain("old line");
    expect(diff.textContent).toContain("new line");
    expect(diff.textContent).toContain("same line");
    // 行渲染归 ChangesDiff：vcompare-diff 只是无障碍包裹，行在其 hunk 内
    const row = diff.querySelector("[data-testid='changes-diff-hunk'] > div > div") as HTMLElement;
    expect(row.textContent).toContain("old line");
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
    // clampDiffRows 先裁到 200 行（del/add 在第 200 行之后 → 不可见）；
    // ChangesDiff 再把连续 ctx 折叠（首尾各留 3 行）：3 ctx + 折叠开关 + 3 ctx = 7
    const hunkBody = () =>
      diff.querySelector("[data-testid='changes-diff-hunk'] > div") as HTMLElement;
    expect(hunkBody().children).toHaveLength(7);
    expect(hunkBody().querySelectorAll("[data-testid='diff-fold-toggle']")).toHaveLength(1);
    expect(hunkBody().textContent).toContain("L199");
    expect(diff.textContent).not.toContain("L200");
    expect(diff.textContent).not.toContain("del-0");
    expect(diff.textContent).not.toContain("add-24");
    fireEvent.click(screen.getByTestId("vcompare-expand"));
    // 展开全部：250 行全量进入（ctx 段仍折叠 = 3+1+3）+ 25 del + 25 add = 57
    expect(hunkBody().children).toHaveLength(57);
    expect(hunkBody().textContent).toContain("del-24");
    expect(screen.getByTestId("vcompare-expand").textContent).toBe("收起");
    // 收起 → 回到前 200 行
    fireEvent.click(screen.getByTestId("vcompare-expand"));
    expect(hunkBody().children).toHaveLength(7);
    expect(diff.textContent).not.toContain("del-0");
  });

  // ── A2 结构化对比渲染 ──────────────────────────────────────────

  it("docx 段级 diff：经 ChangesDiff 渲染，段落序号进 marker 列，diffstat 芯片照常", async () => {
    mockedCompare.mockResolvedValue({
      kind: "docx",
      rows: [
        { type: "ctx", index: 1, text: "标题段" },
        { type: "del", index: 2, text: "旧段" },
        { type: "ctx", index: 2, text: "同段" },
        { type: "add", index: 3, text: "新段" },
      ],
      add: 1,
      del: 1,
      contentMissing: false,
    });
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
    expect(screen.getByTestId("vcompare-stat").textContent).toBe("+1−1");
    const diff = screen.getByTestId("vcompare-diff");
    expect(diff.textContent).toContain("旧段");
    expect(diff.textContent).toContain("新段");
    // 段落序号列（del 取基线序号 2 / add 取当前序号 3）→ DiffRow.marker：
    // 行文本 = 符号列 + marker + 正文（ctx 行符号列为空格）
    const rows = Array.from(
      diff.querySelectorAll("[data-testid='changes-diff-hunk'] > div > div"),
    ) as HTMLElement[];
    expect(rows).toHaveLength(4);
    expect(rows[0].textContent).toBe(" 1标题段");
    expect(rows[1].textContent).toBe("-2旧段");
    expect(rows[2].textContent).toBe(" 2同段");
    expect(rows[3].textContent).toBe("+3新段");
    // del/add 不相邻（中间隔 ctx）→ 不配对，保持红/绿
    expect(rows[1].getAttribute("data-pair")).toBeNull();
    // unsupported 分支不再出现
    expect(screen.queryByTestId("vcompare-unsupported")).toBeNull();
  });

  it("xlsx 结构化对比：sheet hunk label（名 · 状态）+ change 单元格配对行 + 整表删单行", async () => {
    mockedCompare.mockResolvedValue({
      kind: "xlsx",
      sheets: [
        {
          name: "销量",
          state: "changed",
          cells: [
            { kind: "change", ref: "B2", old: "10", new: "20" },
            { kind: "add", ref: "C3", old: "", new: "新增值" },
          ],
          add: 1,
          del: 0,
          change: 1,
          total: 2,
          truncated: false,
        },
        { name: "旧表", state: "del", cells: [], add: 0, del: 0, change: 0, total: 0, truncated: false },
      ],
      add: 3,
      del: 2,
      change: 1,
      contentMissing: false,
    });
    renderT(
      <VersionTimeline
        path="out/报表.xlsx"
        records={[rec({ id: "r1", target: "out/报表.xlsx" })]}
        onPreview={() => {}}
        onRestore={() => {}}
      />,
    );
    fireEvent.click(screen.getByTitle("与当前对比"));
    await screen.findByTestId("vcompare-sheet-0");
    // diffstat 芯片消费 xlsx 汇总
    expect(screen.getByTestId("vcompare-stat").textContent).toBe("+3−2");
    // sheet 0：hunk label = 名字 · 变更 N 处（sheetChanged 拼进 label）
    const sheet0 = screen.getByTestId("vcompare-sheet-0");
    expect(sheet0.textContent).toContain("销量 · 变更 2 处");
    // 行：ref 进 marker 列；change 单元格 = 相邻 del+add → 改蓝配对（data-pair）
    const rows0 = Array.from(
      sheet0.querySelectorAll("[data-testid='changes-diff-hunk'] > div > div"),
    ) as HTMLElement[];
    expect(rows0).toHaveLength(3);
    expect(rows0[0].textContent).toBe("-B210");
    expect(rows0[0].getAttribute("data-pair")).toBe("old");
    expect(rows0[1].textContent).toBe("+B220");
    expect(rows0[1].getAttribute("data-pair")).toBe("new");
    // 余量（纯新增单元格）不配对，保持绿行
    expect(rows0[2].textContent).toBe("+C3新增值");
    expect(rows0[2].getAttribute("data-pair")).toBeNull();
    // sheet 1：整表删除 → 单行（text=sheet 名，无 ref 列）
    const sheet1 = screen.getByTestId("vcompare-sheet-1");
    expect(sheet1.textContent).toContain("旧表 · 已删除工作表");
    const rows1 = Array.from(
      sheet1.querySelectorAll("[data-testid='changes-diff-hunk'] > div > div"),
    ) as HTMLElement[];
    expect(rows1).toHaveLength(1);
    expect(rows1[0].textContent).toBe("-旧表");
    // unsupported 分支不再出现
    expect(screen.queryByTestId("vcompare-unsupported")).toBeNull();
  });

  it("xlsx formula：new 值后追加 fx 后缀（old 侧不带）", async () => {
    mockedCompare.mockResolvedValue({
      kind: "xlsx",
      sheets: [
        {
          name: "Q3",
          state: "changed",
          cells: [{ kind: "change", ref: "B2", old: "10", new: "20", formula: "SUM(A1:A2)" }],
          add: 0,
          del: 0,
          change: 1,
          total: 1,
          truncated: false,
        },
      ],
      add: 1,
      del: 1,
      change: 1,
      contentMissing: false,
    });
    renderT(
      <VersionTimeline
        path="out/Q3.xlsx"
        records={[rec({ id: "r1", target: "out/Q3.xlsx" })]}
        onPreview={() => {}}
        onRestore={() => {}}
      />,
    );
    fireEvent.click(screen.getByTitle("与当前对比"));
    await screen.findByTestId("vcompare-sheet-0");
    const sheet0 = screen.getByTestId("vcompare-sheet-0");
    expect(sheet0.textContent).toContain("Q3 · 变更 1 处");
    const rows = Array.from(
      sheet0.querySelectorAll("[data-testid='changes-diff-hunk'] > div > div"),
    ) as HTMLElement[];
    expect(rows).toHaveLength(2);
    expect(rows[0].textContent).toBe("-B210");
    expect(rows[1].textContent).toBe("+B220  fx: =SUM(A1:A2)");
  });

  it("xlsx 截断：truncated=true 时展示「仅展示前 N 条」提示", async () => {
    mockedCompare.mockResolvedValue({
      kind: "xlsx",
      sheets: [
        {
          name: "大表",
          state: "changed",
          cells: [{ kind: "change", ref: "A1", old: "0", new: "1" }],
          add: 0,
          del: 0,
          change: 300,
          total: 300,
          truncated: true,
        },
      ],
      add: 300,
      del: 300,
      change: 300,
      contentMissing: false,
    });
    renderT(
      <VersionTimeline
        path="out/大表.xlsx"
        records={[rec({ id: "r1", target: "out/大表.xlsx" })]}
        onPreview={() => {}}
        onRestore={() => {}}
      />,
    );
    fireEvent.click(screen.getByTitle("与当前对比"));
    await screen.findByTestId("vcompare-sheet-0");
    // hunk label 三段拼接：sheet 名 · 变更 N 处 · 截断提示（数据层已截，如实标注）
    const sheet0 = screen.getByTestId("vcompare-sheet-0");
    expect(sheet0.textContent).toContain("大表 · 变更 300 处 · 变更过多，仅展示前 1 条");
    // 仅展示截断后的 1 条变更（ref 进 marker 列，值进正文）
    expect(sheet0.textContent).toContain("A1");
    expect(sheet0.textContent).toContain("1");
  });
});
