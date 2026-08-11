import { describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { CostImportModal } from "./CostImportModal";
import { ToastProvider } from "../Toast";

const wrap = (node: React.ReactNode) => <ToastProvider>{node}</ToastProvider>;

describe("CostImportModal 成本导入", () => {
  it("解析预览候选条目，取消勾选后确认导入只提交剩余行", async () => {
    const onClose = vi.fn();
    const onImported = vi.fn();
    render(
      wrap(
        <CostImportModal
          open
          path="C:/tmp/报价单.xlsx"
          fileName="报价单.xlsx"
          onClose={onClose}
          onImported={onImported}
        />,
      ),
    );

    // mock 预览返回 2 条候选（名称在可编辑 input 的 value 中）。
    expect(await screen.findByDisplayValue("HP300 高频液压振动锤")).toBeTruthy();
    expect(screen.getByDisplayValue("P.O 42.5 水泥")).toBeTruthy();
    expect(screen.getByText(/已选 2 \/ 2 条/)).toBeTruthy();

    // 取消第一条的勾选 → 已选 1 条；确认导入。
    const boxes = screen.getAllByRole("checkbox");
    fireEvent.click(boxes[0]);
    expect(screen.getByText(/已选 1 \/ 2 条/)).toBeTruthy();
    fireEvent.click(screen.getByText("确认导入 1 条"));

    await waitFor(() => expect(screen.getByText("已导入 0 条成本条目")).toBeTruthy());
    expect(onImported).toHaveBeenCalled();
    expect(onClose).toHaveBeenCalled();
  });

  it("AI 智能解析按钮切换为 AI 结果", async () => {
    render(
      wrap(
        <CostImportModal
          open
          path="C:/tmp/报价单.xlsx"
          fileName="报价单.xlsx"
          onClose={() => {}}
          onImported={() => {}}
        />,
      ),
    );
    await screen.findByDisplayValue("HP300 高频液压振动锤");

    fireEvent.click(screen.getByText("AI 智能解析"));
    await waitFor(() => expect(screen.getByText(/AI 智能解析完成，请核对后确认导入。/)).toBeTruthy());
  });
});
