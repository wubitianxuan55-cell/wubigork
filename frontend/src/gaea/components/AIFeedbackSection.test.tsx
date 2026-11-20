import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { LocaleProvider } from "../lib/i18n";
import { ToastProvider } from "./Toast";
import { AIFeedbackSection } from "./AIFeedbackSection";

vi.setConfig({ testTimeout: 15_000 });

// AI 反馈分区（2026-11-20）：空态/列表勾选导出/逐条删除三态。
// bridge Proxy mock 同 MemoryPanel.test 配方。

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  del: vi.fn(),
  saveAs: vi.fn(),
}));

vi.mock("../lib/bridge", () => ({
  app: new Proxy({}, {
    get(_t, prop) {
      if (prop === "AIFeedbackList") return mocks.list;
      if (prop === "AIFeedbackDelete") return mocks.del;
      if (prop === "SaveFileAs") return mocks.saveAs;
      return vi.fn();
    },
  }),
  openExternal: vi.fn(),
}));

// 壳内路径钉死：saveExportBlob 据此走 SaveFileAs（jsdom 默认非壳会落
// <a download>，断言面就抓不到另存为调用了）。
vi.mock("../lib/pickFile", () => ({ inShellEnv: () => true }));

const item = (id: string, title: string, over: Record<string, string> = {}) => ({
  id, title, detail: `详情 ${id}`, createdAt: "2026-11-20T10:00:00Z",
  board: "office", category: "tool-friction", severity: "low", ...over,
});

function renderSection() {
  return render(
    <LocaleProvider>
      <ToastProvider>
        <AIFeedbackSection active />
      </ToastProvider>
    </LocaleProvider>,
  );
}

describe("AIFeedbackSection", () => {
  beforeEach(() => {
    // jsdom navigator=en-US 会漂到英文词典（BootSplash.test 同款钉中文）
    localStorage.setItem("gaea-lang", "zh");
    mocks.list.mockReset().mockResolvedValue([]);
    mocks.del.mockReset().mockResolvedValue(undefined);
    mocks.saveAs.mockReset().mockResolvedValue("C:\\out\\ai-feedback.md");
  });

  it("空库显示空态文案", async () => {
    renderSection();
    await waitFor(() => expect(mocks.list).toHaveBeenCalled());
    expect(await screen.findByText(/ai_feedback 工具提交/)).toBeTruthy();
  });

  it("列表渲染徽标+勾选后导出 md（分类分组）", async () => {
    mocks.list.mockResolvedValue([
      item("fb-2", "read_file 采样危险", { severity: "high" }),
      item("fb-1", "缺页数能力", { category: "missing-capability", board: "wenxue" }),
    ]);
    renderSection();
    expect(await screen.findByText("read_file 采样危险")).toBeTruthy();
    expect(screen.getByText("office")).toBeTruthy();
    expect(screen.getByText("wenxue")).toBeTruthy();
    // 全选 → 导出 → SaveFileAs 收到 md 内容（b64）
    fireEvent.click(screen.getByLabelText("全选"));
    fireEvent.click(screen.getByText(/导出选中 \(2\)/));
    await waitFor(() => expect(mocks.saveAs).toHaveBeenCalled());
    const [name] = mocks.saveAs.mock.calls[0];
    expect(name).toMatch(/^ai-feedback-\d{4}-\d{2}-\d{2}\.md$/);
  });

  it("勾选前导出给提示不弹保存", async () => {
    mocks.list.mockResolvedValue([item("fb-1", "一条")]);
    renderSection();
    await screen.findByText("一条");
    // 未勾选时按钮禁用
    const btn = screen.getByText(/导出选中 \(0\)/).closest("button");
    expect(btn?.hasAttribute("disabled")).toBe(true);
  });

  it("逐条删除后列表收敛", async () => {
    mocks.list.mockResolvedValue([item("fb-1", "留一条"), item("fb-2", "删这条")]);
    renderSection();
    await screen.findByText("删这条");
    fireEvent.click(screen.getAllByText("删除")[1]);
    await waitFor(() => expect(mocks.del).toHaveBeenCalledWith("fb-2"));
    await waitFor(() => screen.getByText("留一条"));
    expect(screen.queryByText("删这条")).toBeNull();
  });
});
