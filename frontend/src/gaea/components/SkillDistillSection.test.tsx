// SkillDistillSection（阶段七 7.2-2 流程蒸馏分区）：候选卡=模式步骤序 +
// 「重复 N 次」徽章 + 最近日期 + 证据折叠 + 两动作；空态两分（不可用/暂无）；
// loading 态；动作进行中按钮互斥 loading。文案全部走字典（zh 钉死）。
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import type { SkillDistillView } from "../lib/types";
import { LocaleProvider } from "../lib/i18n";
import { SkillDistillSection } from "./SkillDistillSection";

// 纪律：组件测试一律 vi.mock("../lib/bridge") 提供三桩（分区本身 props 驱动
// 不直调 bridge，桩防未来引入 + 保持与面板测试同构）。
const mocks = vi.hoisted(() => ({
  candidates: vi.fn(),
  draft: vi.fn(),
  decide: vi.fn(),
}));

vi.mock("../lib/bridge", () => ({
  app: {
    SkillDistillCandidates: (...a: unknown[]) => mocks.candidates(...a),
    SkillDistillDraft: (...a: unknown[]) => mocks.draft(...a),
    SkillDistillDecide: (...a: unknown[]) => mocks.decide(...a),
  },
  openExternal: vi.fn(),
}));

const VIEW: SkillDistillView = {
  available: true,
  generatedAt: "2026-09-15T10:00:00Z",
  candidates: [
    {
      id: "jd-abcd1234",
      pattern: ["read_file .md", "edit_file .xlsx"],
      repeat: 3,
      sessions: ["s1", "s2", "s3"],
      evidence: ["会话 s1（09-01）：read_file 周报.md → 编辑后落盘", "会话 s2（09-08）：read_file 周报.md → 编辑后落盘"],
      lastAt: 1757894400000,
    },
    {
      id: "jd-efgh5678",
      pattern: ["write_file .csv"],
      repeat: 4,
      sessions: [],
      evidence: [],
      lastAt: 0,
    },
  ],
};

function renderSection(p?: {
  view?: SkillDistillView | null;
  loading?: boolean;
  onDraft?: (id: string) => Promise<void> | void;
  onIgnore?: (id: string) => Promise<void> | void;
}) {
  return render(
    <LocaleProvider>
      <SkillDistillSection
        view={p?.view === undefined ? VIEW : p.view}
        loading={p?.loading}
        onDraft={p?.onDraft ?? vi.fn()}
        onIgnore={p?.onIgnore ?? vi.fn()}
      />
    </LocaleProvider>,
  );
}

beforeEach(() => {
  // 钉死 zh：避免 detectLocale 走 navigator（en）触发异步字典切换撕断言。
  localStorage.setItem("gaea-lang", "zh");
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("SkillDistillSection 流程蒸馏分区", () => {
  it("渲染候选卡：模式步骤序（code 工具名）+ 重复徽章 + 最近日期 + 证据折叠入口 + 两动作", () => {
    renderSection();
    expect(screen.getByTestId("distill-section")).toBeTruthy();
    expect(screen.getByText("流程蒸馏")).toBeTruthy();

    const card = within(screen.getByTestId("distill-card-jd-abcd1234"));
    expect(card.getByText("重复 3 次")).toBeTruthy();
    expect(card.getByText(/最近\s*\d/)).toBeTruthy();
    expect(card.getByText("read_file")).toBeTruthy();
    expect(card.getByText(".md")).toBeTruthy();
    expect(card.getByText("edit_file")).toBeTruthy();
    expect(card.getByText(".xlsx")).toBeTruthy();
    expect(card.getByText("执行证据")).toBeTruthy();
    expect(card.getByRole("button", { name: "结晶为技能" })).toBeTruthy();
    expect(card.getByRole("button", { name: "不再提示" })).toBeTruthy();

    // 第二张卡：evidence/sessions 皆空 → 不渲染证据折叠入口
    const card2 = within(screen.getByTestId("distill-card-jd-efgh5678"));
    expect(card2.getByText("重复 4 次")).toBeTruthy();
    expect(card2.queryByText("执行证据")).toBeNull();
  });

  it("证据折叠：展开后可见 sessions/evidence 证据行", async () => {
    renderSection();
    fireEvent.click(screen.getByText("执行证据"));
    expect(await screen.findByText(/会话 s1（09-01）/)).toBeTruthy();
    expect(screen.getByText(/会话 s2（09-08）/)).toBeTruthy();
  });

  it("点「结晶为技能」：onDraft(id) 逐参传递，进行中该卡按钮 loading 且互斥，完成后恢复", async () => {
    let resolveDraft!: () => void;
    const onDraft = vi.fn(
      () => new Promise<void>((r) => { resolveDraft = r; }),
    );
    renderSection({ onDraft });

    const card = within(screen.getByTestId("distill-card-jd-abcd1234"));
    const other = within(screen.getByTestId("distill-card-jd-efgh5678"));
    fireEvent.click(card.getByRole("button", { name: "结晶为技能" }));

    expect(onDraft).toHaveBeenCalledWith("jd-abcd1234");
    // loading 图标的 aria-label 会并进可访问名，按文本断言而非 role 名
    await waitFor(() => expect(card.getByText("正在蒸馏…")).toBeTruthy());
    expect(card.getByText("正在蒸馏…").closest("button")!.disabled).toBe(true);
    // 互斥：同卡「不再提示」禁用；另一卡不受影响
    expect(card.getByText("不再提示").closest("button")!.disabled).toBe(true);
    expect((other.getByRole("button", { name: "结晶为技能" }) as HTMLButtonElement).disabled).toBe(false);

    resolveDraft();
    await waitFor(() => expect(card.getByText("结晶为技能")).toBeTruthy());
    expect(card.getByText("结晶为技能").closest("button")!.disabled).toBe(false);
  });

  it("点「不再提示」：onIgnore(id) 逐参传递", async () => {
    const onIgnore = vi.fn(async () => {});
    renderSection({ onIgnore });
    fireEvent.click(within(screen.getByTestId("distill-card-jd-abcd1234")).getByRole("button", { name: "不再提示" }));
    await waitFor(() => expect(onIgnore).toHaveBeenCalledWith("jd-abcd1234"));
  });

  it("空态两分：view=null / available=false → 不可用文案；candidates 空 → 暂无重复流程", () => {
    const { unmount } = renderSection({ view: null });
    expect(screen.getByText("历史执行数据不可用")).toBeTruthy();
    unmount();

    renderSection({ view: { ...VIEW, available: false, candidates: [] } });
    expect(screen.getByText("历史执行数据不可用")).toBeTruthy();
    unmount();

    renderSection({ view: { ...VIEW, available: true, candidates: [] } });
    expect(screen.getByText(/暂无重复流程/)).toBeTruthy();
  });

  it("loading 态：无候选时出挖掘中文案；有候选时卡片照常渲染", () => {
    const { unmount } = renderSection({ view: null, loading: true });
    expect(screen.getByTestId("distill-loading").textContent).toContain("正在挖掘历史执行记录");
    unmount();

    renderSection({ loading: true });
    expect(screen.getByTestId("distill-card-jd-abcd1234")).toBeTruthy();
  });
});
