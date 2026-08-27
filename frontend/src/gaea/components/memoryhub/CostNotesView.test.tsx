import { describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { CostNotesView } from "./CostNotesView";
import { ToastProvider } from "../Toast";
import type { CostReviewNote } from "../../lib/types";

const notes = vi.hoisted(() => ({ list: [] as CostReviewNote[], seq: 1 }));

vi.mock("../../lib/bridge", () => ({
  app: {
    CostNoteList: async (query: string, status: string): Promise<CostReviewNote[]> => {
      const q = (query ?? "").toLowerCase();
      return notes.list
        .filter((n) => (status && status !== "all" ? n.status === status : true))
        .filter((n) => !q || [n.title, n.conclusion].some((s) => (s ?? "").toLowerCase().includes(q)))
        .sort((a, b) => (b.updatedAt ?? "").localeCompare(a.updatedAt ?? ""));
    },
    CostNoteSave: async (n: CostReviewNote): Promise<number> => {
      if (!n.id) {
        const np = { ...n, id: notes.seq++ };
        notes.list.push(np);
        return np.id;
      }
      notes.list = notes.list.map((x) => (x.id === n.id ? { ...x, ...n } : x));
      return n.id;
    },
    CostNoteDelete: async (id: number) => {
      notes.list = notes.list.filter((n) => n.id !== id);
    },
  },
}));

const wrap = (node: React.ReactNode) => <ToastProvider>{node}</ToastProvider>;

describe("CostNotesView 复盘笔记", () => {
  it("空态引导 → 新建笔记 → 列表出现（结论/可信度/状态徽标）", async () => {
    render(wrap(<CostNotesView />));
    expect(await screen.findByText("还没有复盘笔记")).toBeTruthy();

    fireEvent.click(screen.getByText("写下第一条"));
    fireEvent.change(await screen.findByPlaceholderText("如：市政道路土方综合单价复盘"), { target: { value: "土方单价复盘" } });
    fireEvent.change(await screen.findByPlaceholderText("这次测算的核心结论…"), { target: { value: "机械挖土方综合单价约 12.5 元/m³" } });
    fireEvent.click(screen.getByRole("button", { name: /^保\s*存$/ }));

    expect(await screen.findByText("土方单价复盘")).toBeTruthy();
    // 弹窗关闭动画期间可能残留一份表单文本，用 getAllByText 容错
    expect(screen.getAllByText("机械挖土方综合单价约 12.5 元/m³").length).toBeGreaterThan(0);
    expect(screen.getByText("可信度 中")).toBeTruthy();
    // 「草稿」在卡片徽标与弹窗残留的 select 选项中都可能出现
    expect(screen.getAllByText("草稿").length).toBeGreaterThan(0);
  });

  it("状态过滤生效：只显示已确认笔记", async () => {
    notes.list = [
      { id: 1, title: "已确认笔记", conclusion: "c1", boundary: "", risk: "", evidence: "", confidence: "高", status: "已确认" },
      { id: 2, title: "草稿笔记", conclusion: "c2", boundary: "", risk: "", evidence: "", confidence: "中", status: "草稿" },
    ];
    render(wrap(<CostNotesView />));
    expect(await screen.findByText("已确认笔记")).toBeTruthy();
    expect(screen.getByText("草稿笔记")).toBeTruthy();

    fireEvent.change(screen.getByTitle("状态过滤"), { target: { value: "已确认" } });
    expect(await screen.findByText("已确认笔记")).toBeTruthy();
    expect(screen.queryByText("草稿笔记")).toBeNull();
  });
});
