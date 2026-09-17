// TaskInboxPanel（阶段七 7.3-1 任务收件箱）：open 拉取 + 变更后重拉、四档
// tab 计数、状态机动作出现性（CanTransition 前端镜像）、跳转回调、空态两分。
// 纪律：vi.mock("../lib/bridge") 注入四方法桩（SkillDistillSection.test 先例）；
// 文案全部走字典（zh 钉死）；零轮询断言（List 只在 open/变更后被调用）。
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { LocaleProvider } from "../lib/i18n";
import type { TaskInboxView } from "../lib/types";
import { TaskInboxPanel } from "./TaskInboxPanel";

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  save: vi.fn(),
  setStatus: vi.fn(),
  del: vi.fn(),
}));

vi.mock("../lib/bridge", () => ({
  app: {
    GaeaTaskInboxList: (...a: unknown[]) => mocks.list(...a),
    GaeaTaskInboxSave: (...a: unknown[]) => mocks.save(...a),
    GaeaTaskInboxSetStatus: (...a: unknown[]) => mocks.setStatus(...a),
    GaeaTaskInboxDelete: (...a: unknown[]) => mocks.del(...a),
  },
}));

function makeTask(over: Partial<TaskInboxView>): TaskInboxView {
  return {
    id: "ti-000000000001",
    title: "写周报",
    space: "work",
    status: "pending",
    source: "ctrlk",
    createdAt: Date.now() - 3600_000,
    updatedAt: Date.now() - 1800_000,
    ...over,
  };
}

const SAMPLE: TaskInboxView[] = [
  makeTask({ id: "ti-aaa000000001", title: "打开绘梦看图", source: "ctrlk", action: "navigate", target: "imagegen", session: "C:/s/a.jsonl" }),
  makeTask({ id: "ti-aaa000000002", title: "存个任务：汇报PPT", source: "voice" }),
  makeTask({ id: "ti-bbb000000003", title: "整理台账", source: "inbox", status: "doing" }),
  makeTask({ id: "ti-ccc000000004", title: "旧任务", source: "palette", status: "done" }),
  makeTask({ id: "ti-ddd000000005", title: "放弃的任务", source: "weixin", status: "abandoned" }),
];

function renderPanel(p?: { open?: boolean; space?: "work" | "play"; onNavigate?: (b: string) => void; onResumeSession?: (path: string) => void }) {
  return render(
    <LocaleProvider>
      <TaskInboxPanel
        open={p?.open ?? true}
        onClose={vi.fn()}
        space={p?.space ?? "work"}
        onNavigate={p?.onNavigate}
        onResumeSession={p?.onResumeSession}
      />
    </LocaleProvider>,
  );
}

beforeEach(() => {
  // 钉死 zh：避免 detectLocale 走 navigator（en）触发异步字典切换撕断言
  localStorage.setItem("gaea-lang", "zh");
  mocks.list.mockResolvedValue(SAMPLE);
  mocks.save.mockResolvedValue(makeTask({ id: "ti-new000000009", source: "inbox" }));
  mocks.setStatus.mockResolvedValue(makeTask({ status: "doing" }));
  mocks.del.mockResolvedValue(undefined);
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("TaskInboxPanel 任务收件箱", () => {
  it("open 时拉取当前空间；四档 tab 带计数；默认档渲染对应行", async () => {
    renderPanel();
    await waitFor(() => expect(mocks.list).toHaveBeenCalledWith("work"));
    expect(screen.getByTestId("task-inbox-panel")).toBeTruthy();
    // tab 计数：待处理 2 / 进行中 1 / 已完成 1 / 已放弃 1
    expect(screen.getByTestId("task-inbox-tab-pending").textContent).toBe("待处理 2");
    expect(screen.getByTestId("task-inbox-tab-doing").textContent).toBe("进行中 1");
    expect(screen.getByTestId("task-inbox-tab-done").textContent).toBe("已完成 1");
    expect(screen.getByTestId("task-inbox-tab-abandoned").textContent).toBe("已放弃 1");
    // 默认 pending 档只出 2 行
    const rows = screen.getAllByTestId("task-inbox-row");
    expect(rows.length).toBe(2);
    expect(rows[0].textContent).toContain("打开绘梦看图");
  });

  it("切档过滤：doing 档出 1 行且只带该档动作", async () => {
    renderPanel();
    await waitFor(() => expect(screen.getAllByTestId("task-inbox-row").length).toBe(2));
    fireEvent.click(screen.getByTestId("task-inbox-tab-doing"));
    const rows = screen.getAllByTestId("task-inbox-row");
    expect(rows.length).toBe(1);
    expect(rows[0].textContent).toContain("整理台账");
  });

  it("空间隔离：space=play 时按 play 拉取（判据③：各查各空间）", async () => {
    renderPanel({ space: "play" });
    await waitFor(() => expect(mocks.list).toHaveBeenCalledWith("play"));
  });

  it("手动新建：Input + 添加（source='inbox'）→ Save 传 JSON 载荷 → 成功后重拉清单", async () => {
    renderPanel();
    await waitFor(() => expect(mocks.list).toHaveBeenCalledTimes(1));
    // antd Input 无前后缀时把 data-testid 直透到 <input> 本体
    const input = screen.getByTestId("task-inbox-input") as HTMLInputElement;
    fireEvent.change(input, { target: { value: "  预约牙医  " } });
    fireEvent.click(screen.getByTestId("task-inbox-add"));
    await waitFor(() => expect(mocks.save).toHaveBeenCalledTimes(1));
    // 载荷：title trim + space + source=inbox（手动新建兜底来源）
    expect(JSON.parse(mocks.save.mock.calls[0][0] as string)).toEqual({
      title: "预约牙医",
      space: "work",
      source: "inbox",
    });
    // 成功后重拉（变更后重拉判据）+ 草稿清空
    await waitFor(() => expect(mocks.list).toHaveBeenCalledTimes(2));
    await waitFor(() => expect((screen.getByTestId("task-inbox-input") as HTMLInputElement).value).toBe(""));
    // 空草稿时添加禁用
    expect((screen.getByTestId("task-inbox-add") as HTMLButtonElement).disabled).toBe(true);
  });

  it("状态机动作出现性：pending→开始/放弃（无完成）；doing→完成/放弃（无开始）；终态只读；全态可删除", async () => {
    renderPanel();
    await waitFor(() => expect(screen.getAllByTestId("task-inbox-row").length).toBe(2));
    // pending 行：开始 + 放弃 + 删除；完成按钮不存在（非法迁移不渲染）
    const p1 = within(screen.getAllByTestId("task-inbox-row")[0]);
    expect(p1.getByTestId("task-inbox-start-ti-aaa000000001")).toBeTruthy();
    expect(p1.getByTestId("task-inbox-abandon-ti-aaa000000001")).toBeTruthy();
    expect(p1.queryByTestId("task-inbox-finish-ti-aaa000000001")).toBeNull();
    expect(p1.getByTestId("task-inbox-delete-ti-aaa000000001")).toBeTruthy();

    // doing 行：完成 + 放弃；开始不出现
    fireEvent.click(screen.getByTestId("task-inbox-tab-doing"));
    const doing = within(screen.getAllByTestId("task-inbox-row")[0]);
    expect(doing.getByTestId("task-inbox-finish-ti-bbb000000003")).toBeTruthy();
    expect(doing.getByTestId("task-inbox-abandon-ti-bbb000000003")).toBeTruthy();
    expect(doing.queryByTestId("task-inbox-start-ti-bbb000000003")).toBeNull();

    // 终态（done/abandoned）：无任何状态迁移按钮，仅删除（终态只读）
    fireEvent.click(screen.getByTestId("task-inbox-tab-done"));
    const done = within(screen.getAllByTestId("task-inbox-row")[0]);
    expect(done.queryByTestId("task-inbox-start-ti-ccc000000004")).toBeNull();
    expect(done.queryByTestId("task-inbox-finish-ti-ccc000000004")).toBeNull();
    expect(done.queryByTestId("task-inbox-abandon-ti-ccc000000004")).toBeNull();
    expect(done.getByTestId("task-inbox-delete-ti-ccc000000004")).toBeTruthy();
  });

  it("状态动作与删除：SetStatus/Delete 逐参传递且完成后重拉", async () => {
    renderPanel();
    await waitFor(() => expect(screen.getAllByTestId("task-inbox-row").length).toBe(2));
    fireEvent.click(screen.getByTestId("task-inbox-start-ti-aaa000000001"));
    await waitFor(() => expect(mocks.setStatus).toHaveBeenCalledWith("ti-aaa000000001", "doing"));
    await waitFor(() => expect(mocks.list).toHaveBeenCalledTimes(2));

    fireEvent.click(screen.getByTestId("task-inbox-delete-ti-aaa000000002"));
    await waitFor(() => expect(mocks.del).toHaveBeenCalledWith("ti-aaa000000002"));
    await waitFor(() => expect(mocks.list).toHaveBeenCalledTimes(3));
  });

  it("跳转：navigate 目标行出〔去板块〕onNavigate(Target)；带 Session 行出〔回会话〕onNavigate('gaea')", async () => {
    const onNavigate = vi.fn();
    renderPanel({ onNavigate });
    await waitFor(() => expect(screen.getAllByTestId("task-inbox-row").length).toBe(2));
    // 首行（ctrlk 导航意图 + session）：两跳转齐备
    fireEvent.click(screen.getByTestId("task-inbox-go-ti-aaa000000001"));
    expect(onNavigate).toHaveBeenCalledWith("imagegen");
    fireEvent.click(screen.getByTestId("task-inbox-back-ti-aaa000000001"));
    expect(onNavigate).toHaveBeenCalledWith("gaea");
    // 第二行（voice 来源、无 action/target/session）：无跳转按钮
    const row2 = within(screen.getAllByTestId("task-inbox-row")[1]);
    expect(row2.queryByText("去板块")).toBeNull();
    expect(row2.queryByText("回会话")).toBeNull();
  });

  it("空态两分：无任务 → V3Empty 暂无任务 + 提示；加载失败 → 失败文案 + 重试按钮可再拉", async () => {
    mocks.list.mockResolvedValue([]);
    const { unmount } = renderPanel();
    await waitFor(() => expect(screen.getByText("暂无任务")).toBeTruthy());
    expect(screen.getByText(/都会落到这里/)).toBeTruthy();
    unmount();

    mocks.list.mockRejectedValue(new Error("bridge down"));
    renderPanel();
    await waitFor(() => expect(screen.getByText("任务加载失败")).toBeTruthy());
    fireEvent.click(screen.getByTestId("task-inbox-retry"));
    // 本用例内第 3 次拉取：空表 1 + 失败渲染 1 + 重试 1
    await waitFor(() => expect(mocks.list).toHaveBeenCalledTimes(3));
  });

  it("零常驻：close 状态不拉取（判据④：读写均由用户动作触发）", () => {
    renderPanel({ open: false });
    expect(mocks.list).not.toHaveBeenCalled();
    expect(screen.queryByTestId("task-inbox-panel")).toBeNull();
  });

// ── 7.3-1 收口：会话级回源（有回调走精确路径；无回调回退板块粒度）──
describe('TaskInboxPanel 会话级回源（v4.333）', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.list.mockResolvedValue([
      { id: "ti-sec0000000001", title: "回源样本", status: "pending", source: "palette", session: "C:/ws/x/sessions/s1.json", createdAt: 1, updatedAt: 2 },
    ]);
  });

  it('有 onResumeSession：回会话按钮带会话路径精确回调（不再板块粒度）', async () => {
    const onResumeSession = vi.fn();
    const onNavigate = vi.fn();
    const { unmount } = renderPanel({ onNavigate, onResumeSession });
    await waitFor(() => expect(screen.getByText("回源样本")).toBeTruthy());
    fireEvent.click(screen.getByTestId("task-inbox-back-ti-sec0000000001"));
    expect(onResumeSession).toHaveBeenCalledWith("C:/ws/x/sessions/s1.json");
    expect(onNavigate).not.toHaveBeenCalledWith("gaea");
    unmount();
  });

  it('无回调：回退 V1 板块粒度 onNavigate("gaea")（既有行为零变化）', async () => {
    const onNavigate = vi.fn();
    const { unmount } = renderPanel({ onNavigate });
    await waitFor(() => expect(screen.getByText("回源样本")).toBeTruthy());
    fireEvent.click(screen.getByTestId("task-inbox-back-ti-sec0000000001"));
    expect(onNavigate).toHaveBeenCalledWith("gaea");
    unmount();
  });
});
});
