import { describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { KnowledgeImportModal } from "./KnowledgeImportModal";
import { ToastProvider } from "../Toast";

const wrap = (node: React.ReactNode) => <ToastProvider>{node}</ToastProvider>;

describe("KnowledgeImportModal 知识导入", () => {
  it("解析预览候选条目，取消勾选后确认导入只提交剩余行", async () => {
    const onClose = vi.fn();
    const onImported = vi.fn();
    render(
      wrap(
        <KnowledgeImportModal
          open
          path="C:/tmp/规范条文表.md"
          fileName="规范条文表.md"
          onClose={onClose}
          onImported={onImported}
        />,
      ),
    );

    expect(await screen.findByDisplayValue("GB 36600 风险管控")).toBeTruthy();
    expect(screen.getByDisplayValue("桩基施工要点")).toBeTruthy();
    expect(screen.getByText(/已选 2 \/ 2 条/)).toBeTruthy();

    const boxes = screen.getAllByRole("checkbox");
    fireEvent.click(boxes[0]);
    expect(screen.getByText(/已选 1 \/ 2 条/)).toBeTruthy();
    fireEvent.click(screen.getByText("确认导入 1 条"));

    await waitFor(() => expect(screen.getByText("已导入 0 条知识条目")).toBeTruthy());
    expect(onImported).toHaveBeenCalled();
    expect(onClose).toHaveBeenCalled();
  });

  it("AI 智能解析按钮切换为 AI 结果", async () => {
    render(
      wrap(
        <KnowledgeImportModal
          open
          path="C:/tmp/规范条文表.md"
          fileName="规范条文表.md"
          onClose={() => {}}
          onImported={() => {}}
        />,
      ),
    );
    await screen.findByDisplayValue("GB 36600 风险管控");

    fireEvent.click(screen.getByText("AI 智能解析"));
    await waitFor(() => expect(screen.getByText(/AI 智能解析完成，请核对后确认导入。/)).toBeTruthy());
  });
});
